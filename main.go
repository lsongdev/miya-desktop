package main

import (
	"embed"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/lsongdev/miya-agents/logging"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
	wailsupdater "github.com/wailsapp/wails/v3/pkg/updater"
	githubupdater "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

//go:embed all:frontend/dist/*
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

var appVersion = "dev"

func main() {
	if err := logging.SetupFromDefaultConfig("miya-desktop"); err != nil {
		log.Printf("[WARN] logging setup failed: %v", err)
	}

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
	appService.setUpdater(wailsApp.Updater)
	if err := configureUpdater(wailsApp); err != nil {
		log.Printf("[updater] init failed: %v", err)
	}

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
	repairWindowsWebview := func(e *application.WindowEvent) {
		go func() {
			time.Sleep(50 * time.Millisecond)
			window.ZoomReset()
			width, height := window.Size()
			window.SetSize(width, height)
		}()
	}
	if runtime.GOOS == "windows" {
		window.RegisterHook(events.Common.WindowUnMinimise, repairWindowsWebview)
	}

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

func configureUpdater(app *application.App) error {
	gh, err := githubupdater.New(githubupdater.Config{
		Repository:    "lsongdev/miya-desktop",
		Prerelease:    true,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  miyaReleaseAssetMatcher,
	})
	if err != nil {
		return err
	}
	return app.Updater.Init(wailsupdater.Config{
		CurrentVersion: appVersion,
		Providers:      []wailsupdater.Provider{gh},
		Window: &wailsupdater.BuiltinWindow{
			Options: wailsupdater.WindowOptions{
				Title: "Miya Desktop Update",
			},
		},
	})
}

func miyaReleaseAssetMatcher(req wailsupdater.CheckRequest, assets []githubupdater.ReleaseAsset) int {
	platform := strings.ToLower(req.Platform)
	arch := strings.ToLower(req.Arch)
	best := -1
	bestScore := -1
	for i, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".asc") || strings.Contains(name, "sha256sums") {
			continue
		}
		if !strings.Contains(name, platform) {
			continue
		}
		score := 1
		if platform == "darwin" && strings.Contains(name, "universal") {
			score = 2
		}
		if arch != "" && strings.Contains(name, arch) {
			score = 3
		}
		if arch == "amd64" && (strings.Contains(name, "x86_64") || strings.Contains(name, "x64")) {
			score = 3
		}
		if arch == "arm64" && strings.Contains(name, "aarch64") {
			score = 3
		}
		if score > bestScore {
			best = i
			bestScore = score
		}
	}
	return best
}
