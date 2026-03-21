## Why

Go 版 OpenCode 已实现核心 ReAct 循环、两家 LLM 提供商（OpenAI / Anthropic）以及 8 个内置工具，但与上游 TypeScript 原版相比仍存在大量功能缺失：提供商覆盖率低（上游 20+，Go 仅 3）、缺少多个高频工具（todowrite、apply_patch、websearch 等）、无 Agent 模式切换（plan / explore）、会话管理能力薄弱（无 fork/revert/archive/title）、TUI 仅为原始 REPL、HTTP API 路由覆盖不完整、事件通知机制缺失。本变更旨在系统性地补齐这些差距，使 Go 版具备与上游对等的核心用户体验。

## What Changes

### 提供商层
- 新增 **OpenAI-compatible** 通用提供商，支持通过 `base_url` 接入 Azure、Groq、DeepInfra、Together AI 等兼容 API
- 新增 **provider registry**，支持按名称动态注册与查找提供商
- 新增 **model listing**，暴露每个提供商的可用模型列表

### 工具层
- 新增 `todowrite` 工具——持久化任务清单，跨 turn 可见
- 新增 `apply_patch` 工具——接受 unified diff 并应用到文件
- 新增 `websearch` 工具——通过可配置后端执行网络搜索
- 新增 `question` 工具——向用户提问并等待回复（CLI / HTTP 双通道）

### Agent 模式
- 新增 `plan` 模式——禁止写操作工具，仅允许 read/search
- 新增 `explore` 模式——仅允许只读操作，无副作用
- Engine 支持按模式过滤可用工具集

### 会话管理
- 新增 `fork` 操作——从指定消息克隆会话
- 新增 `setTitle` / `setArchived`——会话元数据管理
- 新增 `revert`——回滚到指定消息并删除之后的消息
- 新增 `usage` 统计——按会话统计 token / cost
- 会话 `summary` 自动生成（用 LLM 产生标题）

### 事件总线
- 新增进程内 pub/sub 事件总线（`internal/bus`），用于解耦会话状态变更、工具执行、MCP 通知等事件
- HTTP SSE `/v1/events` 端点，将事件总线暴露给外部客户端

### HTTP API 扩展
- `GET /v1/providers` / `GET /v1/providers/{id}/models` —— 提供商与模型发现
- `POST /v1/sessions/{id}/fork` —— 会话 fork
- `POST /v1/sessions/{id}/revert` —— 会话回滚
- `PATCH /v1/sessions/{id}` —— 更新会话元数据（title / archived）
- `GET /v1/sessions/{id}/usage` —— token / cost 统计
- `GET /v1/events` —— SSE 事件流
- `GET /v1/config` —— 运行时配置读取
- `POST /v1/permission/reply` —— 异步权限回复
- `POST /v1/question/reply` —— question 工具回复

### TUI 增强
- 从原始 REPL 升级为基于 Bubble Tea 的终端 UI
- 对话视图：消息流、Markdown 渲染、代码高亮
- 会话侧边栏：列表、搜索、创建、fork
- 模式切换：build / plan / explore
- 工具确认对话框：展示工具名、参数、确认/拒绝
- 主题系统：至少支持 light / dark 两套

### 权限增强
- 支持 glob pattern 匹配工具名（`bash:*`、`write:/tmp/*`）
- 支持 `once` / `always` / `reject` 回复语义
- 异步权限流（HTTP 模式下 question/permission 通过事件总线回复）

### 配置增强
- `x_opencode_go.agents` 配置：自定义 Agent 模式（名称、权限、模型、温度）
- `x_opencode_go.instructions` 数组：全局系统提示注入
- 远程配置：支持从 `.well-known/opencode` URL 拉取配置

## Capabilities

### New Capabilities
- `openai-compatible-provider`: OpenAI 兼容提供商支持与 provider registry
- `todowrite-tool`: todowrite 工具实现
- `apply-patch-tool`: apply_patch 工具实现
- `websearch-tool`: websearch 工具实现
- `question-tool`: question 工具实现（含 HTTP 异步回复）
- `agent-modes`: Agent 模式系统（build / plan / explore）
- `session-management`: 会话高级管理（fork / revert / title / archive / usage / summary）
- `event-bus`: 进程内事件总线与 SSE 端点
- `tui`: 基于 Bubble Tea 的终端 UI
- `permission-patterns`: glob pattern 权限匹配与异步回复

### Modified Capabilities
- `llm-and-tools`: 新增 provider registry、model listing、按模式过滤工具
- `http-api`: 新增 providers / fork / revert / events / config / permission / question 路由
- `cli-and-config`: 新增 agents / instructions / 远程配置支持
- `builtin-tools`: 调整工具注册以支持 Agent 模式标签

## Impact

- **新增包**：`internal/bus`、`internal/tui`（含子包 views / components / theme）
- **修改包**：`internal/llm`（registry + openai-compatible）、`internal/tool`（新工具 + 模式过滤）、`internal/runtime`（模式支持 + 事件发射）、`internal/server`（新路由）、`internal/store`（会话元数据列 + fork / revert）、`internal/config`（agents / instructions / remote）、`internal/policy`（pattern 匹配）、`internal/cli`（TUI 命令）
- **新增依赖**：`github.com/charmbracelet/bubbletea`、`github.com/charmbracelet/lipgloss`、`github.com/charmbracelet/glamour`（Markdown 渲染）
- **数据库迁移**：v3 → v4，新增 `title`、`archived`、`parent_id`、`parent_message_seq` 列
- **BREAKING**：无公共 API 破坏性变更；新路由为纯新增
