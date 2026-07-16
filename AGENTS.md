# AGENTS.md

## Project Context

Miya Desktop is a Wails + React desktop client for AI Agents. Its primary integration protocol is ACP (Agent Communication Protocol). The client should support local and remote agents, full conversation flows, streaming output, tool calls, thoughts, plans, usage, session management, and future channel-based remote access.

Related repositories are part of the same broader Miya AI Agents effort:

- `miya/miya-desktop`: desktop client and integration shell.
- `miya/miya-agents`: standalone AI agent loop and ACP implementation.
- `miya/miya-channels`: standalone multi-platform chat gateway that bridges chat platforms to ACP agents.

`miya-agents` and `miya-channels` are independent projects, but they are intended to be integrated into `miya-desktop`. If integration reveals protocol, runtime, state-management, or API issues in those repositories, it is acceptable to improve those repositories too, as long as the changes are scoped and compatible with their standalone use.

## Architecture Direction

Use `docs/ROADMAP.md` as the current architecture plan.

The preferred direction is:

- Treat ACP as the shared protocol boundary.
- Keep the backend as the authority for conversations, messages, turns, tool calls, and streaming state.
- Avoid putting protocol reduction logic in React components.
- Normalize ACP updates into a stable conversation timeline before emitting UI events.
- Support multiple Agent profiles and runtime instances instead of a single global ACP client.
- Treat `miya-channels` as a future Channel Connector that feeds the same Conversation Service, not as a separate conversation model.

## Repository Boundaries

Default working directory is `miya-desktop`. Most changes should happen here.

Changes to sibling repositories are allowed when needed for integration:

- Change `miya-agents` if ACP types, server/client behavior, embedded agent loop integration, session handling, or event semantics need improvement.
- Change `miya-channels` if channel session mapping, streaming fanout, ACP client handling, or connector APIs need improvement.
- Preserve each sibling repository's ability to run independently.
- Do not make cross-repo changes casually. Document why the sibling change is required.
- If a change spans repositories, verify each changed repository independently where practical.

## Development Practices

- Prefer small, staged changes that keep the app runnable.
- Preserve user changes in the worktree. Do not reset or revert unrelated edits.
- Use `rg` for search.
- Use `apply_patch` for manual edits.
- Keep frontend state simple; move protocol and stream reduction logic into backend services.
- Add tests around reducers, ACP adapters, and state transitions before large UI work.
- Keep docs updated when architecture or integration contracts change.

## Current Stack

- Desktop: Wails v2
- Backend: Go
- Frontend: React + Vite
- Styling: Tailwind CSS + local UI components
- Protocol: ACP via `github.com/lsongdev/miya-agents/acp`

## Important Current Files

- `internal/agent/manager.go`: current ACP client manager and update parser.
- `app.go`: Wails API surface exposed to React.
- `frontend/src/pages/Chat.jsx`: current chat UI and in-component streaming reducer.
- `frontend/src/context/AgentContext.jsx`: current Agent configuration and connection state.
- `frontend/src/components/SessionList.jsx`: current session list and create/load/delete controls.
- `docs/ARCHITECTURE.md`: current architecture summary.
- `docs/IMPROVEMENTS.md`: known cleanup and quality issues.
- `docs/ROADMAP.md`: target architecture and phased implementation plan.

## Near-Term Priorities

1. Define backend domain models for AgentProfile, AgentRuntime, Conversation, Message, Block, ToolCall, and Turn.
2. Extract ACP update parsing into an adapter package with unit tests.
3. Implement a backend Conversation Reducer that converts ACP updates into a stable timeline.
4. Update React to consume backend timeline events instead of reducing ACP events locally.
5. Replace single-line input with a textarea composer supporting multi-line messages.
6. Add stop/cancel, stopReason, usage, mode/config updates, and better tool call blocks.
7. Introduce multi-Agent profile/runtime management.
8. Move Provider and MCP configuration into backend-managed configuration.
9. Integrate `miya-channels` through the shared Conversation Service.

## UI Guidelines

- Build the real tool UI, not a marketing landing page.
- Keep operational screens dense, scannable, and predictable.
- Use existing local UI components and lucide icons.
- Use textarea for chat composition.
- Tool calls, thoughts, plans, and usage should be first-class timeline blocks.
- Avoid nested cards and decorative-only visual elements.

## Integration Notes

ACP events should be treated as raw protocol input. The app should store or preserve enough raw event data for debugging, but UI rendering should use normalized state.

Recommended flow:

```text
ACP session/update
  -> ACP Adapter
  -> Conversation Reducer
  -> Message Store
  -> Wails Event
  -> React Timeline
```

For `miya-channels`, the long-term flow should be:

```text
Channel IncomingMessage
  -> ConversationService.GetOrCreateConversation(source)
  -> ConversationService.SendMessage()
  -> Agent Runtime ACP Prompt
  -> Conversation Events
  -> Channel Writer
```

## Safety Notes

- API keys and provider secrets should not remain in frontend localStorage long term.
- Prefer system credential storage for secrets.
- High-risk tool calls should eventually route through ACP permission requests or a local permission policy.
- Be careful with commands that modify files outside `miya-desktop`; sibling repository changes should be intentional and described.
