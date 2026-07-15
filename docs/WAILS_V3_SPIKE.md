# Wails v3 Migration Spike

Branch: `spike/wails-v3`

Date: 2026-07-15

## Goal

Evaluate whether Miya Desktop should migrate from Wails v2 to Wails v3, mainly to unlock a clean system tray implementation:

- tray icon
- close window without quitting
- restore main window from tray
- quit from tray menu
- keep current React frontend and Go backend bindings working

## Current Baseline

Miya Desktop currently uses Wails v2:

- CLI: `wails v2.12.0`
- Go module: `github.com/wailsapp/wails/v2`
- config schema: `https://wails.io/schemas/config.v2.json`
- app entry: `wails.Run(&options.App{...})`
- frontend bindings generated under `frontend/wailsjs`

Wails v2 has `HideWindowOnClose`, but the public `options.App` API does not expose a clean tray menu/icon setup path. Enabling hide-on-close without a tray restore path would leave Windows users with a hidden background process and no obvious way back to the main window.

## Wails v3 Findings

The current available v3 tags include:

- `v3.0.0-alpha.102`
- `v3.0.0-alpha.101`
- earlier `v3.0.0-alpha.*` tags

The old `v3-alpha` ref is no longer available through the GitHub contents API.

Wails v3 examples under `v3.0.0-alpha.102` include explicit system tray examples:

- `v3/examples/systray-basic`
- `v3/examples/systray-menu`
- `v3/examples/systray-clock`
- `v3/examples/systray-custom`

The v3 API shape is materially different from v2:

```go
app := application.New(application.Options{
    Name:        "Miya Desktop",
    Description: "AI agent desktop client",
})

window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:  "Miya Desktop",
    Width:  960,
    Height: 600,
})

tray := app.SystemTray.New()
menu := app.NewMenu()
menu.Add("Show Miya Desktop").OnClick(func(ctx *application.Context) {
    window.Show()
})
menu.Add("Quit").OnClick(func(ctx *application.Context) {
    app.Quit()
})
tray.SetMenu(menu)
```

The examples also show a window close hook using `events.Common.WindowClosing`, where the close can be cancelled and the window hidden instead. That is the behavior Miya Desktop needs.

## Current Status

The spike now compiles and builds with Wails v3:

```sh
go test ./...
go run /usr/local/share/go/pkg/mod/github.com/wailsapp/wails/v3@v3.0.0-alpha.102/cmd/wails3 build
```

The v3 build currently uses a minimal project `Taskfile.yaml`:

- builds the React frontend with `npm run build`
- builds the Go app to `bin/miya-desktop`
- keeps the existing v2 `wails.json` untouched while this remains a spike

The frontend bindings are generated into `frontend/bindings` using bundled runtime imports:

```sh
go run /usr/local/share/go/pkg/mod/github.com/wailsapp/wails/v3@v3.0.0-alpha.102/cmd/wails3 generate bindings -ts -b .
```

Bundled runtime mode avoids depending on the npm package `@wailsio/runtime`; Vite keeps `/wails/runtime.js` external because Wails serves it at runtime.

## Migration Work Items

Completed in this spike:

1. Added `github.com/wailsapp/wails/v3@v3.0.0-alpha.102`.
2. Replaced the v2 app entry with `application.New`.
3. Served embedded `frontend/dist` via `application.BundledAssetFileServer`.
4. Registered the existing `App` service through v3 `application.NewService`.
5. Moved app lifecycle onto `ServiceStartup` and `ServiceShutdown`.
6. Decoupled `internal/agent` from Wails v2 runtime event APIs by injecting an event emitter.
7. Added v3 system tray setup:
   - app icon
   - show/focus main window
   - quit menu item
8. Added close-to-tray behavior:
   - intercept window closing
   - hide window
   - cancel close
9. Generated v3 frontend bindings.
10. Updated frontend imports from v2 `wailsjs` to v3 `frontend/bindings`.
11. Verified:
   - `npm run build`
   - `go test ./...`
   - `wails3 build`

Remaining work before considering merge:

1. Run the app interactively and verify service startup, channel auto-start, agent connection, event streaming, and tray restore/quit behavior.
2. Replace the minimal spike `Taskfile.yaml` with the full v3 build asset layout if we want native app bundles, installers, and cross-platform packaging parity.
3. Validate Windows build and Windows tray behavior.
4. Decide whether to remove the old v2 `wails.json` and `frontend/wailsjs` outputs after the branch is no longer a spike.

## Risk Assessment

Migration complexity is medium-high, mostly because v3 is still alpha and changes several integration surfaces at once:

- app/window construction
- lifecycle hooks
- asset serving
- frontend binding generation
- build config
- tray/menu API

The tray API looks significantly better in v3 than v2. The risk is not conceptual feasibility; the risk is v3 alpha churn and build/tooling compatibility.

## Recommendation

Keep this branch as a migration spike.

Do not merge into `master` until a full dev/build loop passes on macOS and Windows. If v3 downloads and builds cleanly, the migration is worth continuing because it gives Miya Desktop a first-class tray model instead of patching around Wails v2 internals.
