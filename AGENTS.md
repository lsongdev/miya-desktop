# Repository Guidelines

Miya Desktop is an ACP-native agent workspace. Its interface is written entirely
in Go with MyGo's native UI toolkit (`github.com/egoist/mygo/ui`). Do not add
React, HTML, JavaScript bindings or a webview to implement application screens.
The update window uses `plugins/updater/native` too.

## Architecture

Use `docs/ARCHITECTURE.md` as the architecture reference.

- ACP is the boundary shared by embedded and external agents and Channels.
- The backend owns conversations, messages, tools and streaming reduction.
- Native views consume normalized snapshots directly through Go services.
- Native presentation state belongs to the window thread. Capture immutable
  parameters in background jobs and commit results through `Window.Update`.
- Each `ui.Element` is valid only during its build pass. Store `ui.Handle`,
  `ui.ListState` and model data rather than Elements.
- Give dynamic controls stable keys. Use `ui.NewTester` to check real controls.
- Use a multi-line `ui.TextArea` composer; thoughts, tools, plans and usage are
  first-class timeline blocks.

## Files

- `main.go`: native application lifecycle, window and tray.
- `native_state.go`: jobs, sessions, snapshot delivery and notifications.
- `native_view.go`, `native_chat.go`: shell and conversation interface.
- `native_settings.go`, `native_forms.go`: settings, forms and registries.
- `app.go`, `registry.go`: application services.
- `internal/`: configuration, ACP adapter, manager, conversation store and channels.
- `mygo.json`: native package configuration.
- `scripts/configure`: Go-only release metadata preparation.

## Development

```sh
make dev
make test
go test -race ./...
go tool mygo vet
make build
make build-macos
make build-windows
```

Go 1.27.3+ is required. No Node.js, npm, frontend build or webview runtime is
required. Format Go files with gofmt. Prefer focused state, protocol, persistence
and native interaction tests.

Preserve existing worktree changes. Use rg for search and apply_patch for edits.
Keep runtime data and credentials out of Git, under `~/.miya/`. Native
preferences live in desktop.json beside the shared config.json. Configuration
should follow miya-agents first, with deliberate desktop/channel extensions.
Keep transport and protocol details outside native rendering code.

The sibling miya-agents and miya-channels repositories are independent versioned
Go modules. Change them only when integration requires it; preserve standalone
use and verify each changed repository separately.
