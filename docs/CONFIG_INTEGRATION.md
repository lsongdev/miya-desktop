# Miya Desktop - Config Integration Plan

## Goal

`miya-desktop` should become the primary UI for editing Miya configuration while preserving standalone operation for:

- `miya-agents`
- `miya-channels`

The shared config file is currently:

```text
~/.miya/config.json
```

Desktop should edit that config through backend APIs, not frontend localStorage.

## Config Domains

### Providers

Providers define model API credentials and base URLs.

```json
{
  "providers": {
    "deepseek": {
      "type": "openai",
      "apiKey": "...",
      "apiBase": "https://api.deepseek.com"
    }
  }
}
```

Near-term desktop work:

- move provider CRUD from localStorage to Go backend
- mask API keys in UI
- later move secrets to OS credential storage

### Agents

Agents are externally callable ACP endpoints.

```json
{
  "agents": [
    {
      "id": "miya",
      "name": "Miya Agents",
      "type": "stdio",
      "command": "miya",
      "args": ["acp"]
    }
  ]
}
```

Desktop should support multiple configured ACP agents so users can switch between Miya Agents, opencode, Codex, Claude, and future remote endpoints.

### Profiles

Profiles bind a provider/model/workspace for `miya-agents` runtime behavior.

```json
{
  "profiles": {
    "default": {
      "provider": "deepseek",
      "model": "deepseek-chat",
      "workspace": "~/.miya/workspace",
      "maxTokens": 8192,
      "contextWindowTokens": 128000,
      "contextWarnRatio": 0.8
    }
  }
}
```

Desktop should distinguish:

- Profile: persisted `miya-agents` runtime config
- Agent: active ACP endpoint/connection

### MCP Servers

MCP server config now supports:

```json
{
  "mcpServers": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "~"]
    },
    "remote": {
      "type": "sse",
      "url": "https://example.com/sse",
      "headers": {
        "Authorization": "Bearer ..."
      }
    }
  }
}
```

Desktop should provide MCP CRUD with:

- type selector: stdio, http, sse
- command/args/env for stdio
- url/headers for http/sse
- test connection action

### Channels

Channels config belongs to `miya-channels`, but Desktop should be able to edit and launch it.

```json
{
  "channels": {
    "telegram": {
      "token": "..."
    }
  }
}
```

Desktop integration should eventually support:

- channel CRUD
- channel status
- start/stop `miya-channels`
- inspect remote conversations

## Backend API Shape

Suggested Wails methods:

```text
LoadMiyaConfig() Config
SaveMiyaConfig(config) error

ListProviders()
UpsertProvider(id, provider)
DeleteProvider(id)

ListAgentProfiles()
UpsertAgentProfile(id, agent)
DeleteAgentProfile(id)

ListMcpServers()
UpsertMcpServer(id, server)
DeleteMcpServer(id)
TestMcpServer(id or server)

ListChannels()
UpsertChannel(id, rawConfig)
DeleteChannel(id)
```

## Migration Strategy

1. Keep existing frontend localStorage as temporary fallback.
2. Add backend config APIs for `~/.miya/config.json`.
3. Load settings page data from backend.
4. Write changes back to backend config.
5. Remove localStorage persistence for providers/agents.
6. Add secret storage for API keys and channel tokens.

## Safety

- Do not log API keys, tokens, or headers.
- Mask secret fields in UI.
- Treat MCP server config as executable/trusted input.
- Add explicit confirmation before deleting providers, MCP servers, or channels.
