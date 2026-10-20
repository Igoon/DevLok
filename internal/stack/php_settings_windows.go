//go:build windows

package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writePHPExtensionSettings(b *strings.Builder, phpDir string, extensions []string) {
	extDir := filepath.Join(phpDir, "ext")
	fmt.Fprintf(b, "extension_dir = %q\n", filepath.ToSlash(extDir))
	for _, ext := range extensions {
		if _, err := os.Stat(filepath.Join(extDir, "php_"+ext+".dll")); err == nil {
			fmt.Fprintf(b, "extension=%s\n", ext)
		}
	}
}
