//go:build !windows

package update

import "devlok/internal/devlokerr"

func LaunchInstaller(string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Update aplikasi tidak tersedia di platform ini.", "Unduh versi terbaru dari halaman rilis dan pasang manual.")
}

func LaunchInstallerDetached(string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Update aplikasi tidak tersedia di platform ini.", "Unduh versi terbaru dari halaman rilis dan pasang manual.")
}

func ScheduleRollback(string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Rollback otomatis tidak tersedia di platform ini.", "Pasang ulang aplikasi dari halaman rilis.")
}
