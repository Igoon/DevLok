package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"devlok/internal/devlokerr"
	rt "devlok/internal/runtime"
)

const DefaultRegistryTTL = 6 * time.Hour

type registryCache struct {
	Schema    int             `json:"schema"`
	FetchedAt time.Time       `json:"fetched_at"`
	Envelope  json.RawMessage `json:"envelope"`
}

func RegistryCachePath(stateDir string) string {
	return filepath.Join(stateDir, "cache", "registry.json")
}

func RemoteRegistries(ctx context.Context, cfg *Config, cachePath string, ttl time.Duration) ([]rt.Registry, error) {
	if cfg == nil {
		return nil, nil
	}
	key, err := configKey(cfg)
	if err != nil {
		return nil, err
	}
	cached, hasCache, fresh := readRegistryCache(cachePath, key, ttl)
	if fresh {
		return validateRegistries(cached)
	}
	p, body, err := fetchPayload(ctx, cfg)
	if err != nil {
		if hasCache {
			return validateRegistries(cached)
		}
		return nil, err
	}
	_ = writeRegistryCache(cachePath, body)
	return validateRegistries(p.Runtimes)
}

func validateRegistries(regs []rt.Registry) ([]rt.Registry, error) {
	for _, reg := range regs {
		if err := rt.ValidateRegistry(reg); err != nil {
			return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Registry runtime remote tidak valid: "+reg.Name, "Jangan gunakan manifest yang tidak ditandatangani publisher resmi.")
		}
	}
	return regs, nil
}

func readRegistryCache(path string, key []byte, ttl time.Duration) ([]rt.Registry, bool, bool) {
	if path == "" {
		return nil, false, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 2<<20 {
		return nil, false, false
	}
	var cache registryCache
	if err := json.Unmarshal(data, &cache); err != nil || cache.Schema != 1 || len(cache.Envelope) == 0 {
		return nil, false, false
	}
	p, err := verifyPayload(cache.Envelope, key)
	if err != nil {
		return nil, false, false
	}
	fresh := ttl > 0 && time.Since(cache.FetchedAt) <= ttl
	return p.Runtimes, true, fresh
}

func writeRegistryCache(path string, body []byte) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(registryCache{
		Schema:    1,
		FetchedAt: time.Now().UTC(),
		Envelope:  json.RawMessage(body),
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}
