//go:build windows || darwin

package main

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"devlok/internal/api"
	"devlok/internal/platform"
)

// runBrowser menjalankan GUI dalam mode browser: menyajikan frontend lewat
// HTTP lokal dan membuka browser default. Tidak memakai WebView maupun tray.
func runBrowser(app *App, port int) error {
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return err
	}
	srv, err := api.New(api.Options{Target: app, Assets: sub})
	if err != nil {
		return err
	}
	addr := "127.0.0.1:0"
	if port > 0 {
		addr = fmt.Sprintf("127.0.0.1:%d", port)
	}
	ln, err := srv.Listen(addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String() + "/"
	app.emit = srv.Emit
	app.startHeadless()
	go func() {
		_ = srv.Serve(ln)
	}()
	fmt.Fprintln(os.Stdout, "DevLok mode browser:", url)
	time.Sleep(300 * time.Millisecond)
	if err := platform.Current().OpenBrowser(url); err != nil {
		fmt.Fprintln(os.Stderr, "Buka manual di browser:", url)
	}
	select {}
}
