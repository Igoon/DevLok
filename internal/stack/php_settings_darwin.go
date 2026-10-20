//go:build darwin

package stack

import "strings"

func writePHPExtensionSettings(b *strings.Builder, phpDir string, extensions []string) {
	_ = b
	_ = phpDir
	_ = extensions
}
