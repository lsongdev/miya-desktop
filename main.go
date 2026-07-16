package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

var appVersion = "dev"

func main() {
	var wailsApp *application.App
	appService := NewApp(func(name string, data ...any) {
		if wailsApp != nil {
			wailsApp.Event.Emit(name, data...)
		}
	})

	wailsApp = application.New(application.Options{
		Name:        "Miya Desktop",
		Description: "AI agent desktop client",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Miya Desktop",
		Width:     960,
		Height:    600,
		MinWidth:  800,
		MinHeight: 450,
		URL:       "/",
	})

	quitting := false
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if quitting {
			return
		}
		window.Hide()
		e.Cancel()
	})

	tray := wailsApp.SystemTray.New()
	tray.SetIcon(appIcon)
	tray.SetTooltip("Miya Desktop")
	tray.OnClick(func() {
		window.Show()
		window.Focus()
	})

	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("Show Miya Desktop").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		quitting = true
		wailsApp.Quit()
	})
	tray.SetMenu(trayMenu)

	err := wailsApp.Run()
	if err != nil {
		log.Fatal(err)
	}
}
