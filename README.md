<p align="center">
  <img src="build/appicon.png" alt="Miya" width="96" height="96">
</p>

<h1 align="center">Miya Desktop</h1>

A native Go workspace for AI agents, built with [MyGo](https://github.com/egoist/mygo).
The entire interface uses `github.com/egoist/mygo/ui`; it has no React, HTML,
JavaScript, Vite, npm dependencies or webview.

## Features

- Built-in Miya profiles and external stdio ACP agents.
- Multi-agent session list, search, create, resume, close and delete.
- Streaming Markdown conversations, selectable text and native copy actions.
- Multi-line composer with Enter to send, Shift+Enter for a newline, and Stop.
- Timeline blocks for thoughts, tools and their input/output, plans, errors,
  context usage, mode and completion reasons.
- Native image previews, audio playback, attachment opening and Save dialogs.
- Provider, profile, external agent, MCP and channel editors.
- Model discovery, MCP marketplace and Skills registry.
- WeChat QR login, embedded gateway controls and automatic channel startup.
- First-run provider/model setup, light/dark appearance and background notifications.
- Tray, native menus, remembered window size, and a native signed-update window.

The agent runtime and channel gateway are embedded in the application.
ACP remains the integration boundary for external agents.

## Development

Requirements: **Go 1.27.3+**. Node.js and a webview runtime are not needed.

```sh
cd miya-desktop
make dev
```

The MyGo CLI is pinned as a Go tool. It watches Go source and restarts the
native application after changes.

```sh
make test                   # backend and native UI tests, plus go vet
go test -race ./...          # race checks
go tool mygo vet             # native view lifetime checks
make build                  # package for the current platform
make build-macos            # universal macOS app and disk image
make build-macos-arm64      # Apple Silicon app and disk image
make build-windows          # Windows amd64 executable; installer requires NSIS
```

Packages go to `bin/<os>-<arch>/`, for example
`bin/darwin-arm64/Miya Desktop.app`. MyGo generates icons, bundles,
manifests and installers from `mygo.json` and `build/appicon.png`.
macOS requires 12.0+. Windows does not require WebView2. On Windows the CLI
can obtain NSIS automatically; cross-compiling an installer needs `makensis`.

The version lives in `mygo.json`. To update it and build:

```sh
make build APP_VERSION=26.10.29
```

This updates the version in `mygo.json`. The Go configuration helper keeps
release packaging independent of JavaScript. Direct `go tool mygo build`
uses the checked-in metadata.

## Configuration and persistence

Configuration and session data keep their existing paths:

```text
~/.miya/config.json
~/.miya/sessions/
~/.miya/workspace/
~/.miya/logs/
```

Important configuration sections:

- `profiles`: embedded Miya agent profiles.
- `providers`: provider credentials, API base URLs, protocol and model mappings.
- `agents`: external ACP endpoints.
- `mcpServers`: stdio, Streamable HTTP or SSE tool servers.
- `channels` and `channelsEnabled`: channel instances and gateway auto-start.

Native preferences are stored in `~/.miya/desktop.json`. Existing configuration
and sessions need no migration. Browser-only appearance, selected-agent and
notification preferences start with native defaults.

Example external ACP endpoint:

```json
{
  "id": "opencode",
  "name": "OpenCode",
  "enabled": true,
  "type": "stdio",
  "command": "opencode",
  "args": ["acp"]
}
```

Each built-in profile also appears in the agent selector. Argument fields
accept a JSON string array or one argument per line, preserving paths and
arguments that contain spaces.

Provider renames update dependent profiles. Editors preserve existing
provider protocol/model mappings and advanced profile options. Gateway
configuration changes restart a running gateway.

## Architecture

```text
MyGo native view -> Go application services -> ACP client manager
                                          -> embedded runtime / external agent

ACP updates -> adapter -> conversation store -> native state -> Window.Update
Channel gateway -> shared ACP client factory -> embedded runtime / external agent
```

The backend owns conversation and protocol reduction. Native UI reads
normalized snapshots directly, with no IPC, JavaScript bindings or frontend
stream reducer. Background jobs apply their results on the window thread.

See [architecture](docs/ARCHITECTURE.md) for module boundaries and validation.

## Releases and automatic updates

`.github/workflows/release.yml` tests the native application and builds
universal/arm64 macOS packages and Windows amd64 packages. Tags use `v<version>`.

Signed updates use the **native** MyGo update plugin:

1. Run `go tool mygo keygen` and keep the private key outside Git.
2. Set the repository variable `MYGO_UPDATER_PUBLIC_KEY`.
3. Set the repository secret `MYGO_UPDATER_PRIVATE_KEY`.

For local signed builds, provide the same environment variables. The
configuration helper stores only the public key in `mygo.json`; the private
key is read by MyGo at build time and is never written to project metadata.
Configured signed builds require the private key.

The build publishes signed archives and `update-<platform>.json` manifests.
The native updater checks, verifies, installs and offers to restart.
Without signing configuration, Check for Updates opens the release downloads
page; development builds do not check for updates.

## Related repositories

- [miya-agents](https://github.com/lsongdev/miya-agents): agent runtime, ACP, MCP, sessions, skills and tools.
- [miya-channels](https://github.com/lsongdev/miya-channels): chat gateway bridging remote channels to ACP agents.

Both remain independent versioned Go modules.
