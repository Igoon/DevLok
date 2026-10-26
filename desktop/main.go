//go:build windows || darwin

package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"devlok/internal/instance"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	browserMode := flag.Bool("browser", false, "Jalankan GUI di browser (tanpa WebView)")
	port := flag.Int("port", 0, "Port server mode browser (0 = otomatis)")
	flag.Parse()

	lock, err := instance.Acquire("gui")
	if err != nil && !errors.Is(err, instance.ErrAlreadyRunning) {
		log.Fatal(err)
	}
	if errors.Is(err, instance.ErrAlreadyRunning) {
		return
	}
	defer lock.Close()

	app := NewApp()
	if *browserMode {
		if err := runBrowser(app, *port); err != nil {
			log.Fatal(err)
		}
		return
	}
	err = wails.Run(&options.App{
		Title:            "DevLok",
		Width:            1120,
		Height:           740,
		MinWidth:         900,
		MinHeight:        600,
		DisableResize:    false,
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 22, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			app.startTray()
		},
		OnBeforeClose: app.beforeClose,
		OnShutdown: func(ctx context.Context) {
			app.stopTray()
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
