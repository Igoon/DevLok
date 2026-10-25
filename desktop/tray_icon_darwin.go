//go:build darwin

package main

import (
	_ "embed"

	"github.com/energye/systray"
)

//go:embed assets/tray-running.png
var trayRunningIcon []byte

//go:embed assets/tray-stopped.png
var trayStoppedIcon []byte

func setTrayIcon(running bool) {
	if running {
		systray.SetTemplateIcon(trayRunningIcon, trayRunningIcon)
		return
	}
	systray.SetTemplateIcon(trayStoppedIcon, trayStoppedIcon)
}
