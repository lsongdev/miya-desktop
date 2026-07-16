package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed all:frontend/dist/*
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
	notificationService := notifications.New()

	wailsApp = application.New(application.Options{
		Name:        "Miya Desktop",
		Description: "AI agent desktop client",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(appService),
			application.NewService(notificationService),
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

	showWindow := func() {
		window.Show()
		window.Focus()
		wailsApp.Event.Emit("app:window-shown")
	}
	hideWindow := func() {
		window.Hide()
		wailsApp.Event.Emit("app:window-hidden")
	}

	quitting := false
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if quitting {
			return
		}
		hideWindow()
		e.Cancel()
	})

	notificationService.OnNotificationResponse(func(result notifications.NotificationResult) {
		if result.Error != nil {
			log.Printf("[notifications] response error: %v", result.Error)
			return
		}
		showWindow()
		wailsApp.Event.Emit("notification:action", result.Response)
	})

	tray := wailsApp.SystemTray.New()
	tray.SetIcon(appIcon)
	tray.SetTooltip("Miya Desktop")
	tray.OnClick(func() {
		showWindow()
	})

	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("Show Miya Desktop").OnClick(func(ctx *application.Context) {
		showWindow()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		quitting = true
		wailsApp.Quit()
	})
	tray.SetMenu(trayMenu)

	err := wailsApp.Run()
	if err != nil {
		log.Fatal(fmt.Errorf("run app: %w", err))
	}
}
