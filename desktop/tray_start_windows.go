//go:build windows

package main

import "github.com/energye/systray"

func (a *App) startTray() {
	go systray.Run(a.onTrayReady, a.onTrayExit)
}

func (a *App) stopTray() {
	systray.Quit()
}
