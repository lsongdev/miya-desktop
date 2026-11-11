package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	nativeupdater "github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
	"github.com/lsongdev/miya-agents/logging"
)

//go:embed build/appicon.png
var appIcon []byte

var appVersion = "dev"

func main() {
	if err := runDesktop(); err != nil {
		log.Fatal(err)
	}
}

func runDesktop() error {
	state := newNativeUI()
	service := NewApp(state.emit)
	state.backend = service
	mygo.Use(nativeupdater.New(updater.Options{Icon: appIcon}))
	mygo.App.SetName("Miya Desktop")
	if version := mygo.App.Version(); isReleaseVersion(version) {
		appVersion = version
	}
	if isReleaseVersion(appVersion) {
		mygo.App.SetVersion(appVersion)
	}
	mygo.Updater.SetPrereleases(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		if state.audio != nil {
			state.audio.Close()
		}
		service.shutdown()
	}()
	mygo.App.WhenReady(func() {
		if err := configureExecutablePath(); err != nil {
			log.Printf("[desktop] executable path: %v", err)
		}
		if err := logging.SetupFromDefaultConfig("miya-desktop"); err != nil {
			log.Printf("[desktop] logging: %v", err)
		}
		service.startup(ctx)
		openDesktopWindow(state)
		state.load()
	})
	if err := mygo.App.Run(); err != nil {
		return fmt.Errorf("run app: %w", err)
	}
	return nil
}

func openDesktopWindow(state *nativeUI) {
	state.logo, _ = ui.DecodeBitmap(appIcon)
	state.md.WrapCode = true
	state.md.OnLink = state.openAttachment
	state.md.LoadImage = state.markdownImage
	window := mygo.NewWindow(mygo.WindowOptions{
		Title: "Miya Desktop", Content: ui.View(state.view),
		Width: 1100, Height: 760, MinWidth: 860, MinHeight: 600,
		StateKey: "main-native",
		// No title bar: the window buttons sit over the translucent sidebar,
		// level with its sidebar toggle.
		TitleBarStyle:        mygo.TitleBarHidden,
		TitleBarHeight:       40,
		TrafficLightPosition: &mygo.Point{X: 20, Y: 20},
		Vibrancy:             mygo.VibrancySidebar,
		BackgroundColor:      "light-dark(#e8e9ee, #0d0d0f)",
	})
	state.win, state.dispatch = window, window.Update
	state.notify = func(note DesktopNotification) {
		go func() {
			if err := state.backend.SendDesktopNotification(note); err != nil {
				log.Printf("[notifications] %v", err)
			}
		}()
	}
	showWindow := func() { window.Show(); window.Focus() }
	window.OnFocus(func() { state.active = true })
	window.OnBlur(func() { state.active = false })
	window.OnHide(func() { state.active = false })
	mygo.App.OnActivate(func(bool) { showWindow() })
	mygo.App.OnNotificationClick(func(id string) { state.page = "Chat"; showWindow(); window.Invalidate() })
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Show Miya Desktop", Click: func(*mygo.MenuItem, *mygo.Window) { showWindow() }},
		mygo.Separator(),
		{Label: "Quit", Role: mygo.RoleQuit},
	})
	tray, err := mygo.NewTray(mygo.TrayOptions{ID: "miya-desktop", Icon: appIcon, ToolTip: "Miya Desktop", Menu: menu})
	if err != nil {
		log.Printf("[desktop] tray: %v", err)
		return
	}
	tray.OnClick(showWindow)
	quitting := false
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
	window.OnClose(func(e *mygo.CloseEvent) {
		if !quitting {
			e.PreventDefault()
			window.Hide()
		}
	})
}
