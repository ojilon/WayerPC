// Command wayerpc is the WayerPC desktop app: phone ↔ PC file transfer.
//
// Terminal use (the installer puts bin/ on PATH):
//
//	wayerpc --version    print version and exit
package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"wayerpc/internal/version"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion || (len(os.Args) > 1 && os.Args[1] == "--version") {
		fmt.Println(version.Current().String())
		return
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "WayerPC",
		Width:  1180,
		Height: 720,
		MinWidth:  960,
		MinHeight: 600,
		Frameless: true, // custom titlebar in frontend/src (frameless design)
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 15, G: 20, B: 25, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []any{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "wayerpc:", err)
		os.Exit(1)
	}
}
