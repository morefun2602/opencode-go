## Why

opencode-go 作为 opencode TypeScript 版本的完整 Go 翻译，目标是成为企业级 Agent 应用开发框架核心。当前 Go 版本已实现基础的 ReAct 循环、LLM 调用、内置工具和会话管理，但与 TypeScript 版本相比仍有大量功能缺失。这些缺失直接影响了框架在 Skills、MCP、ReAct、SubAgent 等核心能力上的完整性，需要系统性地补齐以达到功能对等。

## What Changes

### ReAct 循环与运行时增强
- 新增 **doom loop 检测**：防止连续相同工具调用的无限循环
- 实现 **真正的上下文压缩**：当 token 溢出时通过 LLM 摘要压缩历史，而非仅打日志
- 新增 **结构化错误恢复**：ContextOverflow → 触发压缩、Permission/Question Rejected → blocked 状态
- 增强 **重试逻辑**：支持 retry-after、指数退避、APIError 分类重试

### 工具补全
- 新增 **multiedit 工具**：批量多位置编辑（多组 oldString/newString）
- 新增 **plan 模式工具**：plan_enter/plan_exit，支持运行时模式切换
- 新增 **batch 工具**：并行执行最多 25 个工具调用
- 新增 **skill 工具**：专用工具加载和调用 Skill 指令与资源
- 新增 **ls 工具**：目录树列表，带 ignore 模式支持

### SubAgent 增强
- 新增 **task_id 恢复机制**：通过 task_id 恢复已有子 agent 会话而非每次新建
- 新增 **subagent_type 参数**：支持指定子 agent 类型
- 新增 **description 参数**：子任务描述字段

### MCP 增强
- 新增 **OAuth 认证流程**：OAuthClientProvider、回调处理、客户端信息存储与动态注册

### 会话功能补全
- 实现 **会话摘要**：基于 Snapshot diff 的会话摘要生成
- 增强 **会话 Revert**：依赖 Snapshot 的 revert/unrevert/cleanup
- 实现 **重试模块**：sleep、delay、retryable，支持 retry-after 和退避策略

### 新模块
- 新增 **LSP 集成**：多语言 LSP 客户端，支持 diagnostics、hover、goToDefinition、findReferences、documentSymbol
- 新增 **Snapshot 模块**：基于 git worktree 的文件快照，支持 track/patch/restore/revert/diff
- 新增 **文件监控**：文件变更监听，写操作后发布变更事件，触发 VCS 更新

## Capabilities

### New Capabilities

- `lsp-integration`：多语言 LSP 客户端集成，提供代码智能能力（diagnostics、hover、go-to-definition、references、symbols），并暴露为 lsp 工具供 Agent 调用
- `snapshot-restore`：基于 git worktree 的工作区快照系统，支持 track、patch、restore、revert、diff 操作，为会话 revert 和摘要提供基础
- `file-monitoring`：文件变更监控服务，写操作（write/edit/apply_patch）后发布变更事件，与 VCS 和 Snapshot 集成

### Modified Capabilities

- `react-loop`：新增 doom loop 检测（连续 N 次相同工具+相同输入即终止）、ContextOverflow 错误触发压缩、Permission/Question Rejected 转 blocked 状态、Snapshot 集成（step-start/step-finish 时 track/patch）
- `session-management`：实现真正的上下文压缩（token 溢出 → LLM 摘要 → replay）、会话摘要生成（基于 Snapshot diff）、增强 revert（依赖 Snapshot 的 revert/unrevert/cleanup）、重试模块（retry-after、指数退避）
- `builtin-tools`：新增 multiedit（批量多位置编辑）、plan_enter/plan_exit（运行时模式切换）、batch（并行工具执行）、skill（加载调用 Skill）、ls（目录树列表）五个工具
- `task-tool`：新增 task_id 恢复机制（复用已有子会话）、subagent_type 参数（指定子 agent 类型）、description 参数
- `mcp-integration`：新增 OAuth 认证流程（OAuthClientProvider、回调处理、客户端信息存储、动态注册）
- `skills`：新增 skill 工具集成（通过 tool 调用 Skill，而非仅注入 prompt）、增强发现机制（支持 .claude/skills、.agents/skills 等多路径）

## Impact

### 代码影响
- `internal/runtime/engine.go`：ReAct 循环增强（doom loop、压缩触发、错误恢复、snapshot 集成）
- `internal/tool/`：新增 multiedit.go、plan.go、batch.go、skill_tool.go、ls.go
- `internal/tool/task.go`：重构支持 task_id 恢复和 subagent_type
- `internal/tools/`：新增 retry.go、compaction.go（真正的压缩实现）
- `internal/mcp/`：新增 oauth.go（OAuth 流程）
- 新增 `internal/lsp/`：LSP 客户端模块
- 新增 `internal/snapshot/`：Snapshot 模块
- 新增 `internal/filewatcher/`：文件监控模块
- `internal/store/`：可能新增 snapshot 相关表和迁移
- `internal/skill/`：增强发现机制，新增 skill 工具支持

### API 影响
- HTTP API 可能新增 `/v1/lsp/*` 路由
- HTTP API 可能新增 `/v1/snapshot/*` 路由
- MCP OAuth 回调端点

### 依赖影响
- 可能新增 LSP 协议库依赖
- 可能新增文件监控库依赖（如 fsnotify）
- git 命令行依赖（Snapshot worktree 操作）
