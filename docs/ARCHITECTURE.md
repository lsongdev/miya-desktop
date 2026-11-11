# Miya Desktop 原生界面架构

界面全部使用 MyGo 的 Go 原生 UI（`github.com/egoist/mygo/ui`），主窗口通过
`Content: ui.View(state.view)` 渲染。应用不使用 React、Vite、HTML、WebView
或 JavaScript IPC。更新窗口同样使用 `plugins/updater/native`。

## 模块

| 模块 | 职责 |
| --- | --- |
| `main.go` | 原生窗口、应用生命周期、托盘、焦点与通知点击 |
| `native_state.go` | 界面状态、异步任务、会话选择、发送/停止、通知去重 |
| `native_view.go` | 导航、会话列表、首轮设置、多行输入辅助控件 |
| `native_chat.go` | 虚拟化时间线、流式 Markdown、工具、计划和附件 |
| `native_media.go` | 有大小限制的媒体读取、图片加载和原生音频播放 |
| `native_settings.go` | 设置页面、网关控制、注册表、模型发现、微信登录 |
| `native_forms.go` | 原生配置表单、草稿验证、无损编辑与 Provider 重命名 |
| `desktop.go` | 原生通知、外部文件/链接、原生更新插件入口 |
| `app.go` / `registry.go` | 与界面框架无关的应用服务及注册表功能 |
| `internal/agent` | ACP 连接、握手、会话、历史重放、Prompt 和取消 |
| `internal/acpadapter` | ACP 更新归一化 |
| `internal/conversation` | 会话、消息、工具与流式快照 |
| `internal/config` | 共享 Miya 配置及桌面扩展的持久化 |
| `internal/channels` | 内嵌网关与独立通道事件 |
| `mygo.json` | 原生应用打包、平台信息与可选更新公钥 |
| `scripts/configure` | 用 Go 配置发布版本和公钥，不写入私钥 |

## 状态和线程

```text
原生控件
  -> App 应用服务（后台 goroutine）
  -> ACP client
  -> embedded Agent / external stdio Agent

ACP session/update
  -> acpadapter
  -> conversation.Store
  -> conversation.Snapshot
  -> nativeUI.emit
  -> Window.Update
  -> nativeUI.receive
  -> 原生时间线
```

会话内容（历史、压缩、标题）归 agent 运行时所有，桌面端只通过 ACP 访问。
`conversation.Store` 是由 ACP 更新归约出来的内存视图模型，不落盘，也不是
会话的真相来源；重启后通过 `session/list` 和 `session/load` 重建。内嵌网关
与桌面端是同一运行时的平级 ACP 客户端，桌面端托管它的生命周期和配置，但
不接管 channel 会话的路由。

协议解析和流式归约留在后端。原生界面保存展示快照，不自行解析 ACP。
`ui.List` 按消息 ID 维护虚拟化行，自动跟随末尾，用户上滚时保留阅读位置。
`markdown.Stream` 展示流式回复，结束后切换为 `markdown.View`。

界面状态只在窗口线程修改。任务捕获不可变参数，后台工作返回提交函数，
通过 `Window.Update` 更新视图。发送期间阻止切换 Agent 和破坏性会话动作，
Stop 通过独立后台操作调用 ACP 取消。会话草稿按 Agent/session 引用保存。
配置保存采用副本，校验失败不会修改当前配置；重命名 Provider 同步修正引用，
编辑已有项会保留当前表单未覆盖的共享配置字段。

网关状态的 ChannelEvents map 会在锁内复制，防止原生渲染读取变化中的 map。

## 功能覆盖

- 会话搜索、创建、恢复、关闭、删除和跨 Agent 选择。
- 多行编辑器、Enter 发送、Shift+Enter 换行、原生剪贴板和停止。
- Markdown、思考、工具输入输出、计划、错误、上下文用量、模式和停止原因。
- 内嵌、文件和远端图片预览，WAV/MP3/Ogg Vorbis 原生播放，文件/链接打开和原生保存对话框。
- Provider、Profile、外部 Agent、MCP 和 Channel 的原生表单。
- 模型获取、MCP Marketplace、Skills 注册表及安装/删除。
- 首次运行设置、主题、后台完成通知和原生设置持久化。
- Channels 启停、自动运行、修改后重启和微信扫码登录。

## 数据与生命周期

沿用 `~/.miya/config.json`、sessions、workspace 和 logs；界面偏好写入
`~/.miya/desktop.json`。不再依赖浏览器存储。

应用就绪后启动后端并创建原生窗口。窗口关闭时隐藏到托盘，托盘、Dock 和
通知可恢复窗口。退出时取消上下文、停止网关和登录任务并断开 Agent。
无需前端构建或绑定生成。应用和更新窗口均无需系统 WebView。

## 测试

`native_test.go` 使用 MyGo 的 `ui.NewTester`，直接在内存中渲染并操作实际
Go 控件，覆盖设置页面、表单取消/保存、Provider 重命名、多行及中文输入、
时间线区块、剪贴板和通知去重。集成测试从原生输入框经过内嵌 ACP，到本地
模拟 OpenAI 流式服务，再返回原生时间线；停止测试验证取消传递到 HTTP 请求。

`scripts/configure` 测试发布版本、签名配置完整性和失败时的文件保护。
后端保留已有协议、会话和配置测试。

```sh
make test
go test -race ./...
go tool mygo vet
```

按需输出可审阅的原生渲染：

```sh
MIYA_SCREENSHOT_DIR=bin/screenshots go test -run TestNativeScreenshots .
```
