package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	backend := NewBackend()
	err := wails.Run(&options.App{
		Title:     "imv",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  backend.startup,
		OnShutdown: backend.shutdown,
		Bind: []interface{}{
			backend,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:  true,
			CSSDropProperty: "--wails-drop-target",
			CSSDropValue:    "drop",
		},
		BackgroundColour: &options.RGBA{R: 250, G: 247, B: 242, A: 255},
	})
	if err != nil {
		log.Fatal(err)
	}
}
