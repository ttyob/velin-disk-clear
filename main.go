package main

import (
	"embed"
	"flag"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()
	devAPI := flag.Bool("dev-api", false, "serve the browser preview API")
	devAddr := flag.String("dev-api-addr", "0.0.0.0:8787", "browser preview API address")
	flag.Parse()
	if *devAPI {
		if err := serveDevAPI(app, *devAddr); err != nil {
			println("Dev API error:", err.Error())
		}
		return
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Velin Clear",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
