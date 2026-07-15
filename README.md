# Miya Desktop

Miya Desktop is an ACP-native desktop workspace for AI agents. It brings local agent runtimes, external ACP agents, session management, MCP tools, and remote-control channels into a single client built for day-to-day agent work.

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

- `agents`: ACP endpoints used by the desktop client.
- `profiles`: Miya agent runtime profiles.
- `providers`: model provider credentials and base URLs.
- `mcpServers`: MCP tool server definitions.
- `channels`: channel gateway configuration.
- `channelsEnabled`: desktop preference for auto-starting the embedded channel service.

Example agent endpoint:

```json
{
  "id": "miya",
  "name": "Miya Agents",
  "enabled": true,
  "type": "builtin",
  "command": "miya-agent",
  "args": ["acp"]
}
```

Existing development configs such as `go -C .../miya-agents run . acp` are recognized as the built-in Miya agent when the endpoint id is `miya`.

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

GitHub Actions provides a macOS release workflow:

```text
.github/workflows/release.yml
```

The workflow checks out:

- `miya-desktop`
- `miya-agents`
- `miya-channels`

It builds a macOS universal Wails app and uploads a prerelease artifact:

```text
miya-desktop-darwin-universal.zip
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
