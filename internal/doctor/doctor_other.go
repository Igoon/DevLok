//go:build !windows

package doctor

import "devlok/internal/platform"

func platformChecks(plat platform.Platform) []Check {
	return []Check{{
		Name:    "platform",
		Status:  StatusOK,
		Message: "Pemeriksaan platform " + plat.Name() + " belum lengkap pada v0.1.0.",
	}}
}
