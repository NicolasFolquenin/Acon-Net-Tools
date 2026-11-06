//go:build !cli

package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Cria uma instância da aplicação
	app := NewApp()

	execName := "desktop-client-2026"
	if p, err := os.Executable(); err == nil {
		execName = strings.ToLower(filepath.Base(p))
	}
	lockId := "uepg-net-tools-" + execName

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
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 248, G: 250, B: 252, A: 255}, // Light Base #f8fafc
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			initSystemTray(ctx, app)
		},
		OnShutdown: func(ctx context.Context) {
			cleanupTray()
			app.shutdown(ctx)
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			showTrayBalloon("UEPG Net Tools", "A sonda continua monitorando a rede em segundo plano. Clique no ícone da bandeja para reabrir.")
			return false // HideWindowOnClose: true esconde a janela sem encerrar o processo
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: lockId,
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if app.ctx != nil {
					runtime.WindowShow(app.ctx)
					runtime.WindowUnminimise(app.ctx)
				}
			},
		},
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
