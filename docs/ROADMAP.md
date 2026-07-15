# Miya Desktop - Agent Client 架构规划

## 背景

Miya Desktop 的目标是成为一个完整的 AI Agent 客户端。它通过 ACP (Agent Communication Protocol) 调用不同 Agent，例如 opencode ACP、Codex、Claude、内置的 `miya-agents` agent loop，以及后续的 `miya-channels` 远程聊天入口。

核心产品能力包括：

- 多 Agent 配置、连接和运行实例管理
- 多会话管理、加载、恢复、关闭、删除
- 完整对话流程，包括消息发送、编辑、多行输入、停止、重试、重新生成
- 流式 Markdown 输出
- Tool call、tool update、plan、usage、mode、thought 等 Agent 过程展示
- 与 `miya-agents` 共享 ACP 能力
- 后续接入 `miya-channels`，支持 Telegram、Feishu、WeChat、WeCom 等远程聊天工具调用

## 当前状态

当前代码已经跑通了基础 ACP 链路：

- 后端通过 `acp.DialStdio()` 启动本地 Agent 进程。
- `internal/agent/manager.go` 维护单个 ACP client，并处理 `initialize`、`sessions/create`、`sessions/load`、`sessions/list`、`sessions/close`、`sessions/delete`、`prompt`。
- 后端解析 `session/update`，并通过 Wails `runtime.EventsEmit` 推送给前端。
- 前端 `Chat.jsx` 在组件内拼接流式消息，展示 Markdown、thought、tool call、plan。
- `AgentContext.jsx` 用 localStorage 保存 Agent 命令配置。
- `Settings.jsx` 已有 Agents、Providers、MCP、Appearance 等设置页面雏形。

这说明基础协议和 UI 原型方向可行，但当前架构仍偏向单 Agent demo。

## 主要问题

### 1. 缺少统一领域模型

当前存在三套临时模型：

- ACP 原始 `SessionUpdate`
- Go 后端 `UpdateEvent`
- React 组件内 `message`

前端现在负责猜测 message boundary、拼接 streaming chunk、合并 tool update。这会导致后续接入不同 Agent、消息编辑、会话恢复、远程 channel 时逻辑快速复杂化。

### 2. Agent 管理粒度不足

当前后端只有一个全局 ACP client。完整客户端需要支持：

- 多个 Agent profile
- 多个 Agent runtime
- 每个 runtime 下多个 conversation/session
- 不同 transport：stdio、embedded、remote
- 不同 provider/env/mcp/cwd 配置

### 3. 会话状态不可靠

消息、tool、thought、streaming 状态目前主要存在 React 组件内存中。切换页面、重连、加载历史、远程消息进入时，很难保证状态一致。

### 4. 配置没有进入后端权威层

Providers、MCP、cwd、环境变量、权限策略等配置大多只存在前端 localStorage。后续它们必须参与 Agent 启动、Session 创建和 Prompt 发送。

### 5. ACP 能力没有完整产品化

当前尚未完整利用 ACP 的能力：

- `PromptResponse.stopReason`
- `usage_update`
- `current_mode_update`
- `config_option_update`
- `available_commands_update`
- `request_permission`
- `cancel`
- `resume`
- session mode/config 设置

### 6. miya-channels 需要共享同一套会话模型

`miya-channels` 当前也是独立 ACP worker，并维护 `channel:user -> acp session`。后续不应把它作为特殊逻辑塞进桌面 UI，而应让它成为同一套 Conversation Service 的外部输入/输出通道。

## 目标架构

建议将系统拆成五层。

```text
Frontend UI
  |
  v
Wails App API
  |
  v
Conversation Service
  |
  +--> Agent Runtime Manager
  |      |
  |      +--> ACP Transport: stdio
  |      +--> ACP Transport: embedded
  |      +--> ACP Transport: remote
  |
  +--> Event Store / Message Store
  |
  +--> Channel Connector
```

### 1. Transport 层

Transport 只负责连接 ACP endpoint，不理解业务 UI。

推荐接口：

```go
type Transport interface {
	Connect(ctx context.Context, spec TransportSpec) (*acp.Client, error)
	Close() error
}
```

Transport 类型：

- `stdio`: opencode、codex、claude、miya-agents CLI
- `embedded`: 直接启动内置 `miya-agents` agent loop
- `remote`: 后续连接远程 ACP 服务或 `miya-channels`
- `http/ws`: 预留网络 ACP

### 2. Agent Runtime Manager

负责 Agent profile 和运行实例。

```go
type AgentProfile struct {
	ID         string
	Name       string
	Kind       string
	Command    string
	Args       []string
	Env        map[string]string
	DefaultCwd string
	ProviderID string
	McpServerIDs []string
}

type AgentRuntime struct {
	ID           string
	ProfileID    string
	Status       string
	Capabilities acp.AgentCapabilities
	AgentInfo    *acp.Implementation
}
```

原则：

- Profile 是配置。
- Runtime 是正在运行的连接。
- Conversation 绑定 Runtime，而不是绑定全局 singleton。

### 3. Conversation Service

Conversation Service 是后端状态权威，统一处理桌面会话和远程 channel 会话。

```go
type Conversation struct {
	ID           string
	RuntimeID    string
	ACPSessionID string
	Title        string
	Cwd          string
	Source       ConversationSource
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ConversationSource struct {
	Type      string // desktop, telegram, feishu, wechat, wecom, api
	Channel   string
	AccountID string
	ThreadID  string
}
```

推荐 API：

```text
CreateAgentProfile(profile)
UpdateAgentProfile(profile)
StartAgent(profileId) -> runtimeId
StopAgent(runtimeId)
ListAgentRuntimes()

CreateConversation(runtimeId, options) -> conversation
LoadConversation(conversationId)
ListConversations(filter)
CloseConversation(conversationId)
DeleteConversation(conversationId)

SendMessage(conversationId, blocks)
EditMessage(messageId, blocks)
CancelTurn(conversationId)
SetMode(conversationId, modeId)
SetConfigOption(conversationId, configId, value)
```

### 4. Event Store / Message Store

ACP 原始事件不应直接驱动 UI。推荐流程：

```text
ACP session/update
  -> ACP Adapter Parse
  -> Conversation Reducer Apply
  -> Store Append
  -> Wails Emit conversation:event
  -> React Render
```

统一消息模型：

```go
type Message struct {
	ID             string
	ConversationID string
	TurnID         string
	Role           string // user, assistant, system, tool
	Blocks         []Block
	Status         string // pending, streaming, complete, failed, superseded
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Block struct {
	ID      string
	Type    string // text, markdown, thought, tool_call, plan, image, audio, error
	Content string
	Meta    map[string]any
}

type ToolCallState struct {
	ID        string
	Title     string
	Kind      string
	Status    string
	Input     json.RawMessage
	Output    json.RawMessage
	Locations []acp.ToolCallLocation
}

type Turn struct {
	ID             string
	ConversationID string
	UserMessageID  string
	Status         string
	StopReason     string
	Usage          *acp.UsageUpdate
}
```

Reducer 职责：

- 根据 `messageId` 或本地策略确定 message boundary
- 合并 text streaming chunk
- 将 thought 作为独立 block
- 创建和更新 tool call block
- 保存 plan、usage、mode、config options
- 在 `PromptResponse` 返回时关闭 turn
- 保存 raw ACP event，便于调试和回放

### 5. UI Store 层

React 只负责渲染和交互，不负责协议归一化。

主要组件建议：

- `AgentSwitcher`: 选择 profile/runtime
- `ConversationList`: 会话列表，支持来源和 agent 过滤
- `MessageTimeline`: 渲染后端归一化 timeline
- `MessageComposer`: 多行输入、附件、发送、停止
- `ThoughtBlock`: 可折叠思考过程
- `ToolCallBlock`: 展示状态、输入、输出、位置、错误
- `PlanBlock`: 展示任务计划和状态
- `UsageIndicator`: 展示上下文和费用
- `ChannelInbox`: 后续展示远程 channel 会话

## 与 miya-agents 的关系

`miya-agents` 应作为一个标准 ACP Agent runtime 接入。

推荐两种模式：

1. `stdio` 模式：通过命令启动 `miya-agents acp`。
2. `embedded` 模式：桌面后端直接引入 `miya-agents` 包并运行 agent loop。

无论哪种模式，进入 Miya Desktop 后都应该被抽象成同一个 `AgentRuntime`，避免 UI 感知差异。

## 与 miya-channels 的关系

`miya-channels` 应作为 Channel Connector，而不是单独的会话系统。

目标流程：

```text
Telegram/Feishu/WeChat/WeCom
  -> Channel Connector IncomingMessage
  -> ConversationService.GetOrCreateConversation(source)
  -> ConversationService.SendMessage()
  -> Agent Runtime ACP Prompt
  -> Conversation Event Store
  -> Channel Writer streams assistant text blocks
```

桌面端可以同时查看这些远程会话，因为它们使用相同的 Conversation 和 Message Store。

需要注意：

- Channel 输出通常只需要 assistant text block。
- 桌面端可展示完整 thought/tool/plan。
- 每个 channel user/thread 映射到一个 conversation。
- `/new`、`/stop` 等命令应调用 Conversation Service，而不是直接操作 ACP client。

## 消息编辑设计

消息编辑不建议直接修改历史并覆盖下游状态。推荐使用 turn 分支或 supersede 策略。

简单版本：

1. 用户编辑某条 user message。
2. 将原消息之后的 assistant response 标记为 `superseded`。
3. 创建新的 user message revision。
4. 创建新 turn 并重新 Prompt。

后续可扩展为 conversation branch：

```text
Message A
  -> Assistant B
  -> Edited Message A'
       -> Assistant C
```

UI 初期可只展示当前 active branch。

## 多行输入设计

Composer 应使用 textarea，而不是 input。

推荐交互：

- `Enter`: 发送
- `Shift+Enter`: 换行
- `Cmd/Ctrl+Enter`: 也可发送，作为可配置项
- streaming 中显示 Stop 按钮
- 支持粘贴多行文本

## Tool Call 展示设计

Tool call 应作为 message block，而不是 message 外挂数组。

展示层级：

- 折叠标题：tool title、kind、status、耗时
- 输入：raw input / 参数
- 输出：raw output / terminal / diff / file path
- 位置：文件路径和行号
- 错误：失败原因

对于 edit/delete/execute 这类高风险工具，后续需要结合 ACP permission request 或本地权限策略。

## Provider 和 MCP 配置

Provider 配置应从前端 localStorage 迁移到后端配置服务。

建议：

- 普通配置保存在应用配置目录。
- API Key 使用系统钥匙串。
- Agent 启动时由 Runtime Manager 注入 env。
- MCP server 配置在 Create/Load Session 时传入 ACP。

```go
type ProviderConfig struct {
	ID      string
	Name    string
	Kind    string
	BaseURL string
	KeyRef  string
}

type McpServerConfig struct {
	ID      string
	Name    string
	Type    string // stdio, http, sse
	Command string
	Args    []string
	URL     string
	Env     map[string]string
	Headers map[string]string
}
```

## 分阶段路线

### Phase 1: 架构地基

- 新建后端 conversation/domain 模型。
- 将 ACP update parse 从 `Manager` 拆成 adapter，并补单元测试。
- 后端实现 Conversation Reducer。
- 后端维护 timeline，前端只订阅归一化事件。
- 去掉 `/tmp` 硬编码，cwd 进入会话配置。

### Phase 2: 完整聊天体验

- Composer 改为 textarea，支持多行输入。
- 支持 stop/cancel。
- 展示 `stopReason`、`usage`、`mode`、`config options`。
- Thought、Tool、Plan 改为 block 级展示。
- 支持消息编辑和重新生成。

### Phase 3: 多 Agent 管理

- 实现 AgentProfile 和 AgentRuntime。
- 支持多个 Agent profile。
- Conversation 绑定 runtime。
- 根据 agent capabilities 控制 UI 功能。
- Provider/MCP/env/cwd 配置进入后端。

### Phase 4: 本地持久化

- 保存 conversations、messages、turns、tool calls、raw events。
- 支持离线查看历史。
- 支持 reload/replay conversation timeline。
- 为调试提供 raw ACP event viewer。

### Phase 5: 接入 miya-channels

- 将 `miya-channels` 改造成 Channel Connector。
- Channel message 进入 Conversation Service。
- Channel writer 从 conversation events 流式写回。
- 桌面端支持查看和接管远程会话。

## 近期优先级

建议优先级如下：

1. 后端 Conversation/Message/Block/Turn 模型
2. ACP Adapter + Reducer + 单元测试
3. Chat.jsx 改为消费后端 timeline
4. textarea composer + stop/cancel
5. AgentProfile/Runtime 重构
6. Provider/MCP 配置后端化
7. miya-channels 共享 Conversation Service

最关键的是先把“ACP 事件到稳定 conversation timeline”的后端状态层做出来。这个边界稳定后，多 Agent、消息编辑、远程 channel、内置 miya-agents 都可以沿着同一套模型扩展。
