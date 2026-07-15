# Miya Desktop - 改进建议

## 严重问题 (High Priority)

### 1. 命名不统一

当前存在多处命名冲突：

| 位置 | 当前值 | 建议值 |
|------|--------|--------|
| `go.mod` module | `wails-app` | `miya-desktop` |
| `wails.json` name | `wails-app` | `miya-desktop` |
| `wails.json` outputFilename | `wails-app` | `miya-desktop` |
| `main.go` 窗口标题 | `"wails-app"` | `"Miya Desktop"` |
| macOS bundleId | `com.wails.wails-app` | `com.lsong.miya-desktop` |

### 2. go.mod 中的本地 replace 指令

```go
replace github.com/lsongdev/miya-agents => /Users/Lsong/Projects/miya-agents
```

这会导致其他开发者无法构建项目。应：
- 将 `miya-agents` 发布到 Go module proxy
- 或在发布前移除 replace 指令

### 3. 会话工作目录硬编码为 `/tmp`

`SessionList.jsx` 中：
```javascript
const session = await CreateSession('/tmp')
```

应改为用户可配置的工作目录，或使用 `os.userHomeDir()` / 当前工作目录。

---

## 中等问题 (Medium Priority)

### 4. 移除死代码

以下文件/代码未被使用，应清理：

| 文件 | 原因 |
|------|------|
| `frontend/src/App.css` | 空文件 |
| `frontend/src/pages/Home.jsx` | 未路由 |
| `frontend/src/pages/About.jsx` | 未路由（Settings 中有内联 About） |
| `frontend/src/hooks/useNavigate.js` | 未使用 |
| `frontend/public/fonts/` (Nunito) | 使用的是 Geist 字体 |
| `app.go` 中的 `Greet()` 方法 | Wails 模板默认，未被调用 |

### 5. Provider 配置未生效

`ProviderContext` 管理 LLM 供应商配置（API Key、Base URL），但这些数据从未传递给 Go 后端或代理进程。当前只是 UI 展示，无实际功能。应：
- 将 provider 配置通过启动参数或环境变量传递给 ACP 代理
- 或在 Go 端实现 provider 管理

### 6. 移除生产环境中的 console.log

`Chat.jsx` 中有调试日志：
```javascript
console.log('agent_thought_chunk', data)
console.log('tool_call', data)
console.log('tool_call_update', data)
```

应移除或通过环境变量控制。

### 7. Settings.jsx 中的调试残留

`Settings.jsx` 第 191 行：
```javascript
{(adding || (editing === null && false))}
```

`&& false` 使条件永远为 false，应简化为 `adding`。

### 8. 错误处理改进

- `main.go` 中使用 `println()` 输出错误，GUI 应用中不可见，应使用日志框架
- 无 React Error Boundary，代理进程崩溃时前端无优雅降级
- 流式事件监听缺少超时机制

---

## 低优先级 (Low Priority)

### 9. 包管理器统一

项目中同时存在 `package-lock.json`（npm）和 `pnpm-lock.yaml`（pnpm），应统一使用一个。建议使用 pnpm。

### 10. API Key 安全存储

Provider API Key 以明文存储在 localStorage 中。虽然是桌面应用，但仍建议：
- 使用 Go 后端通过系统钥匙串（macOS Keychain / Windows Credential Manager）存储
- 或至少加密存储

### 11. 添加 React Error Boundary

在 `App.jsx` 外层包装 Error Boundary，捕获组件渲染错误，提供优雅降级 UI。

### 12. 添加 TypeScript 支持

当前前端使用 JSX，无类型检查。建议：
- 逐步迁移到 TSX
- 为 Wails 绑定生成类型定义

### 13. MCP Servers 功能

Settings 中 MCP Servers 分区标记为 "Coming soon..."，如需实现建议：
- 在 Go 端实现 MCP 客户端
- 前端提供 MCP 服务器配置 UI

### 14. 会话本地持久化

当前会话数据依赖远程代理存储。可考虑本地缓存，支持离线查看历史会话。

### 15. tsconfig 引用修正

`tsconfig.node.json` 引用 `vite.config.ts`，实际文件为 `vite.config.js`，应修正。

---

## 建议优先级排序

1. ✅ 统一命名 + 移除 replace 指令
2. ✅ 清理死代码
3. ✅ 修复工作目录硬编码
4. ✅ 移除 console.log + 调试残留
5. ⬜ 实现 Provider 配置传递
6. ⬜ 添加 Error Boundary
7. ⬜ 改进错误处理
8. ⬜ 考虑 TypeScript 迁移
