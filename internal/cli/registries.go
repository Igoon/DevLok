package cli

import (
	"context"
	"time"

	"devlok/internal/platform"
	rt "devlok/internal/runtime"
	"devlok/internal/update"
)

func loadRemoteRegistries(stateDir string) []rt.Registry {
	if stateDir == "" {
		var err error
		stateDir, err = platform.Current().StateDir()
		if err != nil {
			return nil
		}
	}
	cfg, err := update.LoadConfig(stateDir)
	if err != nil {
		return nil
	}
	if cfg == nil {
		cfg = update.DefaultConfig()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	remote, err := update.RemoteRegistries(ctx, cfg, update.RegistryCachePath(stateDir), update.DefaultRegistryTTL)
	if err != nil || len(remote) == 0 {
		return nil
	}
	embedded, err := rt.LoadEmbedded()
	if err != nil {
		return remote
	}
	return mergeRegistryLists(embedded, remote)
}

func mergeRegistryLists(base, override []rt.Registry) []rt.Registry {
	byName := map[string]rt.Registry{}
	var order []string
	for _, reg := range base {
		if _, ok := byName[reg.Name]; !ok {
			order = append(order, reg.Name)
		}
		byName[reg.Name] = reg
	}
	for _, reg := range override {
		if _, ok := byName[reg.Name]; !ok {
			order = append(order, reg.Name)
		}
		byName[reg.Name] = reg
	}
	out := make([]rt.Registry, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out
}
