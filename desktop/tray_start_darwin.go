//go:build darwin

package main

// Tray macOS dinonaktifkan: systray membutuhkan main thread Cocoa yang sudah
// dipakai Wails. Mengaktifkannya menyebabkan SIGABRT saat startup.
func (a *App) startTray() {}

func (a *App) stopTray() {}
