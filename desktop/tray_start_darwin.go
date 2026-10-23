//go:build darwin

package main

/*
#include <dispatch/dispatch.h>

extern void nativeStart(void);

static void devlokTrayTrampoline(void *ctx) {
	(void)ctx;
	nativeStart();
}

static void devlokScheduleTrayStart(void) {
	dispatch_async_f(dispatch_get_main_queue(), NULL, devlokTrayTrampoline);
}
*/
import "C"

import "github.com/energye/systray"

// Tray macOS: systray harus membuat NSStatusItem di main thread Cocoa, sedangkan
// Wails menjalankan OnStartup di goroutine lain dan sudah memakai main thread
// untuk run loop-nya. Callback didaftarkan lewat Register lalu nativeStart
// dijadwalkan pada main queue.
func (a *App) startTray() {
	systray.Register(a.onTrayReady, a.onTrayExit)
	C.devlokScheduleTrayStart()
}

func (a *App) stopTray() {
	systray.Quit()
}
