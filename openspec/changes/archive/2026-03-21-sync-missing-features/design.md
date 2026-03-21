## Context

opencode-go 是 opencode（TypeScript）的 Go 完整翻译版本，定位为企业级 Agent 应用开发框架核心。当前 Go 版本已建立基础架构：ReAct 循环（`internal/runtime/engine.go`）、LLM Provider 抽象（`internal/llm/`）、12 个内置工具（`internal/tool/`）、SQLite 持久化（`internal/store/`）、事件总线（`internal/bus/`）、MCP 客户端（`internal/mcp/`）、HTTP API（`internal/server/`）、TUI（`internal/tui/`）。

通过与 TypeScript 版本的完整对比，识别出以下类别的功能缺失：
- ReAct 循环缺少鲁棒性机制（doom loop、压缩、错误恢复）
- 工具集不完整（缺 5 个工具：multiedit、plan、batch、skill、ls）
- SubAgent 无法恢复已有会话
- MCP 无 OAuth 支持
- 缺少 LSP、Snapshot、文件监控三个独立模块

现有代码库遵循清晰的包分层：`cmd/` → `internal/cli/` → `internal/runtime/` → `internal/tool/` + `internal/llm/` + `internal/store/`。新功能需融入此结构。

## Goals / Non-Goals

**Goals:**

- 补齐 Go 版本与 TypeScript 版本之间的全部核心功能差距
- 增强 ReAct 循环的鲁棒性：doom loop 检测、真正的上下文压缩、结构化错误恢复
- 补全工具集：multiedit、plan_enter/plan_exit、batch、skill、ls
- 增强 SubAgent：支持 task_id 恢复、subagent_type、description
- 新增 LSP 集成、Snapshot、文件监控三个模块
- MCP OAuth 认证支持
- 所有实现遵循 Go 最佳工程实践，而非逐行翻译 TypeScript

**Non-Goals:**

- 不移植 TypeScript 的 Effect 运行时模式（Go 使用显式依赖注入）
- 不移植 TS 特有的 Provider（Azure、Google、Mistral 等）——仅保持 OpenAI + Anthropic + OpenAI-Compatible
- 不移植 Share/分享功能（企业级需求另行设计）
- 不移植 Control Plane / Workspace Server（当前单进程模型足够）
- 不移植 PTY 模块（Go 的 bash 工具已覆盖执行需求）
- 不移植 codesearch 工具（依赖外部 Exa API，非核心）
- 不重构现有已运行的模块（如 TUI、HTTP API），除非功能补齐所需

## Decisions

### D1: Doom loop 检测 — 滑动窗口方式

**选择**：在 ReAct 循环内维护最近 N 次（默认 3）tool_call 的签名（工具名+参数 hash），当连续 N 次相同时通过 Permission.Ask 通知用户，由用户决定是否继续。

**备选**：(A) 直接终止循环——过于粗暴，可能误杀合理重试。(B) 计数器限制单工具调用次数——无法区分相同参数和不同参数。

**理由**：TS 版本使用相同策略（processor.ts L151-176），经验证有效。滑动窗口避免误报，Permission.Ask 保留用户控制权。

### D2: 上下文压缩 — LLM 摘要 + Replay

**选择**：当 LLM 返回 ContextOverflow 错误时，触发 `Compaction.Process()`：(1) 用 LLM 对历史消息生成摘要 (2) 清除旧消息 (3) 用摘要替代 (4) 重放最近 N 条消息继续对话。

**备选**：(A) 简单截断——丢失关键上下文。(B) 基于 embedding 的选择性保留——实现复杂，效果不确定。

**理由**：TS 版本（session/compaction.ts）验证此方案可行。LLM 摘要保留关键语义，replay 保持对话连贯性。

### D3: 新工具放置 — 继续使用 internal/tool 包

**选择**：所有新工具（multiedit、plan、batch、skill、ls）作为独立 .go 文件放入 `internal/tool/`，在 `builtin.go` 中注册。共享逻辑放 `internal/tools/`。

**理由**：与现有工具保持一致的组织结构。每个工具一个文件便于维护。

### D4: LSP 集成 — 独立包 + 工具暴露

**选择**：新建 `internal/lsp/` 包，封装 LSP 客户端协议实现。通过 `internal/tool/lsp.go` 暴露为工具供 Agent 调用。LSP 服务端进程生命周期由 Engine 管理。

**备选**：(A) 使用第三方 Go LSP 库——现有库（gopls 内部包）不适合作为客户端复用。(B) 嵌入 gopls——仅支持 Go，不通用。

**理由**：自行实现 LSP 客户端部分（initialize、textDocument/diagnostics、textDocument/definition 等）协议量有限（~10 个方法），且可精确控制生命周期。

### D5: Snapshot — git stash/worktree 方式

**选择**：新建 `internal/snapshot/` 包，使用 `git stash` 或 `git diff` 保存文件状态差异，支持 track（保存当前状态）、patch（应用增量）、restore（恢复到指定快照）、diff（对比两个快照）。

**备选**：(A) 基于文件系统复制——对大仓库不可行。(B) 基于 git worktree——隔离性好但资源消耗大。

**理由**：git diff/stash 轻量级，绝大多数项目已有 git，无额外依赖。

### D6: 文件监控 — fsnotify

**选择**：新建 `internal/filewatcher/` 包，使用 `github.com/fsnotify/fsnotify` 监控工作区文件变更。write/edit/apply_patch 工具执行后主动发布 `file.changed` 事件到 Bus。

**理由**：fsnotify 是 Go 生态最成熟的跨平台文件监控库。

### D7: Task 工具恢复 — task_id 参数

**选择**：扩展 task 工具的 schema，新增 `task_id`（可选）、`subagent_type`（可选）、`description`（可选）参数。当提供 task_id 时，通过 Store 查找已有子会话并复用；未提供时行为不变（创建新会话）。

**理由**：与 TS 版本对齐，支持长期子任务的断点续传。

### D8: MCP OAuth — 本地回调服务器

**选择**：在 `internal/mcp/` 新增 oauth.go，实现 OAuth 2.0 授权码流程：启动临时本地 HTTP 服务器接收回调、存储 token 到文件系统、自动刷新。

**理由**：TS 版本的成熟实现证明此方案可行。本地回调服务器避免用户手动复制 token。

### D9: Batch 工具 — 受控并发

**选择**：batch 工具接受工具调用数组（最多 25 个），使用 `errgroup` 并发执行，收集全部结果后统一返回。只读工具（read 标签）无限制，写工具串行执行。

**理由**：并发执行读操作提高效率，串行写操作保证一致性。

## Risks / Trade-offs

- **[风险] LSP 实现复杂度** → 缓解：首版仅实现 5 个核心方法（initialize、diagnostics、definition、references、documentSymbol），后续迭代扩展
- **[风险] Snapshot 依赖 git** → 缓解：在非 git 项目中优雅降级（禁用 snapshot 功能并警告）
- **[风险] 上下文压缩摘要质量** → 缓解：保留最近 N 条消息不压缩，仅压缩早期历史；使用与主对话相同的 Provider
- **[风险] OAuth 流程的安全性** → 缓解：本地回调仅监听 localhost、token 加密存储、短有效期
- **[权衡] batch 并发 vs 串行** → 写工具串行执行牺牲速度换取一致性，但读操作并发带来显著提速
- **[权衡] 新增 3 个依赖（LSP 协议、fsnotify、OAuth 库）** → 均为成熟库，维护风险低
