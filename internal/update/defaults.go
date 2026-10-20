package update

import (
	"context"

	"devlok/internal/devlokerr"
)

const (
	DefaultManifestURL = "https://github.com/Igoon/DevLok/releases/latest/download/updates.json"
	DefaultPublicKey   = "kq28S/Gisl3yN8SOaWrjyv8vs6Fdktoyn1y35h4WSkU="
)

func DefaultConfig() *Config {
	return &Config{
		Schema:      1,
		ManifestURL: DefaultManifestURL,
		PublicKey:   DefaultPublicKey,
	}
}

func AppUpdateCheck(ctx context.Context, cfg *Config) (*AppUpdate, error) {
	if cfg != nil {
		return RemoteCheck(ctx, cfg)
	}
	signed, err := RemoteCheck(ctx, DefaultConfig())
	if err == nil {
		return signed, nil
	}
	if e, ok := devlokerr.As(err); ok && (e.Code == devlokerr.CodeChecksumMismatch || e.Code == devlokerr.CodeInvalidArgument) {
		return nil, err
	}
	return GitHubCheck(ctx)
}
