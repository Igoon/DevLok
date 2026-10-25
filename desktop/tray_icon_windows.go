//go:build windows

package main

import "github.com/energye/systray"

func setTrayIcon(running bool) {
	if running {
		systray.SetIcon(runningIcon)
		return
	}
	systray.SetIcon(stoppedIcon)
}
