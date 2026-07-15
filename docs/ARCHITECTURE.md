# Miya Desktop - 架构文档

## 项目概述

Miya Desktop 是一个基于 [ACP (Agent Communication Protocol)](https://github.com/lsongdev/miya-agents) 的 AI 代理聊天客户端。通过 stdio 与本地运行的 AI 代理进程（如 OpenCode）进行 JSON-RPC 通信，提供原生桌面体验。

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 桌面框架 | Wails | v2.12.0 |
| 后端语言 | Go | 1.25.0 |
| 前端框架 | React | 18.2.x |
| 构建工具 | Vite | 8.0.11 |
| CSS 框架 | Tailwind CSS | 4.2.4 |
| UI 组件库 | shadcn/ui (base-nova) | v4.7.0 |
| Markdown 渲染 | react-markdown + remark-gfm | 10.1.0 |
| ACP 客户端 | miya-agents | v0.0.0-20260612 |

## 项目结构

```
miya-desktop/
├── main.go                     # 入口：嵌入前端 dist，初始化 Wails 应用
├── app.go                      # App 结构体：暴露 Go 方法给前端调用
├── wails.json                  # Wails 项目配置
├── go.mod / go.sum             # Go 依赖管理
├── internal/
│   └── agent/
│       └── manager.go          # 核心：ACP 客户端、会话管理、事件解析
├── frontend/
│   └── src/
│       ├── App.jsx             # 根组件：侧边栏 + 页面路由
│       ├── pages/
│       │   ├── Chat.jsx        # 主聊天页面，流式消息展示
│       │   └── Settings.jsx    # 设置面板（通用、代理、供应商、MCP、外观、关于）
│       ├── components/
│       │   ├── SessionList.jsx # 会话列表侧边栏
│       │   └── MarkdownContent.jsx  # Markdown 渲染组件
│       ├── context/
│       │   ├── AgentContext.jsx     # 代理连接状态，localStorage 持久化
│       │   ├── ProviderContext.jsx  # LLM 供应商 CRUD
│       │   └── ThemeContext.jsx     # 主题切换（亮/暗）
│       └── lib/
│           └── utils.js        # cn() 工具函数（clsx + tailwind-merge）
└── build/                      # 构建配置（图标、平台 plist 等）
```

## 数据流

```
┌──────────────────┐    Wails Bindings    ┌──────────────┐    ACP stdio    ┌──────────────┐
│   Frontend       │ ──────────────────> │   Go Backend  │ ──────────────> │ Agent 进程    │
│   (React)        │ <────────────────── │   (app.go)    │ <────────────── │ (opencode)   │
│                  │  runtime.EventsEmit │               │                 │              │
└──────────────────┘                     └──────────────┘                 └──────────────┘
```

### 通信机制

1. **前端 → Go**: 通过 Wails 绑定调用 Go 方法（如 `CreateSession`、`SendPrompt`、`ListSessions`）
2. **Go → 前端**: 通过 `runtime.EventsEmit` 推送流式事件（`session:update`）

## 核心组件

### 后端 (Go)

#### `internal/agent/manager.go`

核心管理器，负责：

- **ACP 连接**: `acp.DialStdio()` 启动子进程并建立 stdio 通信
- **协议握手**: 发送初始化请求（协议版本 1），接收代理信息和能力
- **会话生命周期**: `sessions/create`、`sessions/load`、`sessions/list`、`sessions/close`、`sessions/delete`
- **流式事件解析**: 解码所有 ACP 事件类型并推送到前端

支持的 ACP 事件类型：

| 事件类型 | 说明 |
|----------|------|
| `user_message_chunk` | 用户消息片段 |
| `agent_message_chunk` | 代理回复片段（文本/图片/音频） |
| `agent_thought_chunk` | 代理思考/推理过程 |
| `tool_call` / `tool_call_update` | 工具调用及状态更新（pending → in-progress → completed/failed） |
| `plan` | 执行计划（带条目完成状态） |
| `usage_update` | Token/上下文用量统计 |
| `current_mode_update` | 代理模式切换 |
| `session_info_update` | 会话标题、更新时间 |

#### `app.go`

Wails 桥接层，将 Go 方法暴露给前端：

```go
func (a *App) CreateSession(workdir string) (*agent.Session, error)
func (a *App) LoadSession(id string) (*agent.Session, error)
func (a *App) ListSessions() ([]*agent.Session, error)
func (a *App) CloseSession(id string) error
func (a *App) DeleteSession(id string) error
func (a *App) SendPrompt(sessionID, prompt string) error
```

### 前端 (React)

#### 路由结构

```
App.jsx
├── Sidebar (可折叠导航)
│   ├── Chat 页面
│   └── Settings 页面
└── 页面内容区
```

#### 关键组件

- **SessionList.jsx**: 会话 CRUD + 侧边栏展示
- **Chat.jsx**: 消息流展示、自动滚动、流式光标动画
- **Settings.jsx**: 6 个设置分区（通用/代理/供应商/MCP/外观/关于）

#### 状态管理

通过 React Context + localStorage 实现持久化：

- `AgentContext`: 代理配置列表、当前选中的代理、连接状态
- `ProviderContext`: LLM 供应商配置（API Key、Base URL）
- `ThemeContext`: 亮/暗主题切换

## 构建与运行

### 开发模式

```bash
wails dev
```

### 生产构建

```bash
wails build
```

输出目录: `build/bin/`

### 平台支持

- **macOS**: 最低 macOS 10.13，支持 High-DPI
- **Windows**: 使用 WebView2，支持 COM 互操作

## 依赖关系

### Go 依赖

- `github.com/lsongdev/miya-agents` — ACP 客户端库（本地 replace）
- `github.com/wailsapp/wails/v2` — Wails 桌面框架

### 前端依赖

- React 18 + ReactDOM
- @base-ui/react (shadcn v4 基础)
- Lucide React (图标)
- react-markdown + remark-gfm (Markdown 渲染)
- Tailwind CSS 4 + tw-animate-css
- Geist Variable 字体
