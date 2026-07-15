<p align="center">
  <img src="build/appicon.png" alt="Miya" width="96" height="96">
</p>

<h1 align="center">Miya Desktop</h1>

<p align="center">
  ACP-native desktop workspace for AI agents, local runtimes, external ACP endpoints, MCP tools, and remote-control channels.
</p>

<p align="center">
  <a href="https://github.com/lsongdev/miya-desktop/actions/workflows/release.yml"><img src="https://github.com/lsongdev/miya-desktop/actions/workflows/release.yml/badge.svg" alt="Release"></a>
  <a href="https://github.com/lsongdev/miya-desktop/releases"><img src="https://img.shields.io/github/v/release/lsongdev/miya-desktop?include_prereleases&label=release" alt="Latest release"></a>
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/Wails-2.12-red" alt="Wails 2.12">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows-blue" alt="Platforms">
</p>

The app embeds the Miya agent runtime and channel gateway directly into the desktop binary, while still keeping ACP as the protocol boundary for external agents such as OpenCode, Codex, Claude-compatible tools, or any other ACP endpoint.

## Highlights

- **ACP-first agent client**: connect to embedded Miya agents or external stdio ACP agents.
- **Multi-agent sessions**: view and switch sessions across enabled agents from one session list.
- **Streaming conversations**: render assistant output, Markdown, tool calls, thoughts, plans, usage, and stop reasons.
- **Embedded runtime**: `miya-agents` and `miya-channels` are compiled into the desktop app for a single-app install experience.
- **Configurable providers and profiles**: manage providers, agent profiles, MCP servers, and channels from Settings.
- **Remote-control channels**: run the embedded channel service to bridge chat platforms into ACP agents.
- **Native desktop shell**: Wails + Go backend with a React frontend.

## Product Scope

Miya Desktop is designed as an operator-focused client rather than a generic chat UI. The core workflow is:

1. Configure providers, profiles, MCP servers, and ACP agent endpoints.
2. Enable one or more agents.
3. Create or resume sessions from the left session list.
4. Interact with agents through a streaming timeline that exposes tool activity and reasoning artifacts.
5. Optionally enable Channels so remote chat tools can control the same agent runtime.

## Architecture

```text
React UI
  -> Wails API
  -> Go backend
  -> ACP client manager
  -> embedded miya-agents or external ACP endpoint
```

For the built-in Miya agent, the desktop app uses an in-process ACP transport. This keeps the same ACP client/server contract without launching a separate `miya-agent` process.

```text
desktop acp.Client
  -> in-process ACP transport
  -> miya-agents acp.Server
  -> miya-agents runtime
```

Channels are also embedded:

```text
miya-channels app.Run
  -> channel connector
  -> ACP client
  -> embedded or external agent
```

The related projects can still run independently:

- [miya-agents](https://github.com/lsongdev/miya-agents): agent loop, ACP implementation, MCP support, sessions, tools.
- [miya-channels](https://github.com/lsongdev/miya-channels): chat platform gateway that bridges channel messages to ACP agents.

## Configuration

Miya Desktop reads and writes the shared Miya config file:

```text
~/.miya/config.json
```

Important top-level sections:

- `agents`: external stdio ACP endpoints used by the desktop client.
- `profiles`: Miya agent runtime profiles.
- `providers`: model provider credentials and base URLs.
- `mcpServers`: MCP tool server definitions.
- `channels`: channel gateway configuration.
- `channelsEnabled`: desktop preference for auto-starting the embedded channel service.

The built-in Miya agent is owned by Miya Desktop and does not need to be declared in `agents`.

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

Miya Desktop injects the built-in `Miya Agents` endpoint at runtime. This keeps the config file focused on external ACP tools.

## Development

Prerequisites:

- Go
- Node.js
- Wails CLI

Install frontend dependencies:

```bash
cd frontend
npm install
```

Run the desktop app in development mode:

```bash
wails dev
```

Build a local production app:

```bash
wails build -platform darwin/arm64 -clean -trimpath
```

Run backend tests:

```bash
go test ./...
```

Run frontend build validation:

```bash
cd frontend
npm run build
```

## Release

GitHub Actions provides a release workflow:

```text
.github/workflows/release.yml
```

The workflow checks out:

- `miya-desktop`
- `miya-agents`
- `miya-channels`

It builds Wails apps and uploads prerelease artifacts:

```text
miya-desktop-darwin-universal.zip
miya-desktop-windows-amd64.zip
```

Manual release trigger:

```bash
gh workflow run Release --field version=v0.1.0-test.3
```

## Repository Layout

```text
app.go                         Wails API surface
internal/agent                 ACP client manager and conversation bridge
internal/agentclient           embedded/external ACP client factory
internal/acpadapter            ACP update normalization
internal/channels              embedded channel service controller
internal/config                shared Miya config service
frontend/src/pages/Chat.jsx    main conversation UI
frontend/src/pages/Settings.jsx configuration UI
docs/                          architecture and roadmap notes
```

## Status

Miya Desktop is in preview. The current focus is stabilizing the agent runtime integration, session model, channel service, and configuration experience before expanding platform packaging and deeper permission controls.
