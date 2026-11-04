//go:build !cli

package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Cria uma instância da aplicação
	app := NewApp()

	// Configuração do Wails v2
	err := wails.Run(&options.App{
		Title:             "UEPG-Net-Tools - Monitoramento Distribuído",
		Width:             1200,
		Height:            800,
		MinWidth:          640,
		MinHeight:         520,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 248, G: 250, B: 252, A: 255}, // Light Base #f8fafc
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
			Theme:                windows.Light,
		},
	})

	if err != nil {
		log.Fatalf("Erro ao iniciar aplicação Wails: %v", err)
	}
}
