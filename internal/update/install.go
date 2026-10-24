package update

import "path/filepath"

func RollbackBackupDir(appDir string) string {
	return filepath.Join(appDir, "backup", "previous")
}
