//go:build windows

package stack

import (
	"context"

	rt "devlok/internal/runtime"
)

func installPlatformRuntime(ctx context.Context, mgr *rt.Manager, name, constraint, wantSHA, stateDir string) (rt.Installed, error) {
	return installRuntime(ctx, mgr, name, constraint, wantSHA)
}
