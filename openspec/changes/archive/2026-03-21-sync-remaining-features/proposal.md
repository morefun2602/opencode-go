## Why

opencode-go 在第一轮同步（sync-missing-features）后已补齐了大量基础模块（doom loop、compaction 基础实现、retry、新工具、LSP/Snapshot/FileWatcher 模块等），但深入对比 TypeScript 版本发现仍存在多项核心功能缺失或未接入。这些缺失集中在三个层面：一是已实现但未接入的模块（Snapshot、LSP、FileWatcher 均未在 wire 中注入、写工具未发布文件变更事件）；二是 Agent 框架核心能力缺失（无系统提示构建、无模型路由、无 Agent 级定义、Compaction 未真正集成到会话流程）；三是工具层的关键增强缺失（无统一输出截断、read 工具无 offset/limit、无 invalid 工具处理畸形调用）。这些直接影响了框架作为企业级 Agent 应用开发核心的可用性。

## What Changes

### 核心 Agent 框架

- 实现系统提示构建模块：按模型选择 base prompt（OpenAI/Anthropic）、注入环境信息（工作目录、平台、日期、git 状态）、支持 InstructionPrompt（AGENTS.md/CLAUDE.md 向上查找 + config.instructions 文件/URL 加载）
- 实现 Provider 路由与模型选择：defaultModel()、getSmallModel()、parseModel("provider/model")，支持 config.model / config.small_model
- 完善 Agent 定义：新增 general、compaction、title、summary 四个 Agent，每个有独立 prompt 和工具权限规则；Agent 级 prompt 可覆盖 provider prompt
- 将 Compaction 真正集成到会话流程：基于 token 的 isOverflow 检测、prune 裁剪旧 tool 输出、使用 compaction agent 生成摘要

### 工具层增强

- 实现统一工具输出截断服务（Truncate）：按行数/字节数截断、支持 head/tail 方向、所有工具统一接入
- read 工具增加 offset/limit 参数支持大文件分段读取
- 实现 invalid 工具处理畸形工具调用，确保 ReAct 循环不中断
- 增强 Provider 消息转换：per-provider 消息规范化（空 content 过滤、toolCallId 规范化）

### 模块接入

- 在 wire.go 中接入 Snapshot 服务到 Engine
- 在 wire.go 中接入 LSP 客户端并注册 lsp 工具
- 在 write/edit/apply_patch 工具中调用 FileWatcher.NotifyChange 发布文件变更事件
- Revert 操作集成 Snapshot.Restore 恢复工作区文件状态

### 配置增强

- 支持 JSONC 格式（注释和尾逗号）
- 支持 config.model / config.small_model 配置项
- 支持 InstructionPrompt 相关配置：config.instructions（文件路径和 URL 数组）
- 增加 config.compaction 配置（auto、reserved、prune）

## Capabilities

### New Capabilities

- `system-prompt`：系统提示构建模块，负责按模型选择 base prompt、注入环境信息、加载 InstructionPrompt（AGENTS.md/CLAUDE.md）、技能列表注入
- `provider-routing`：Provider 路由与模型选择，实现 defaultModel / getSmallModel / parseModel，支持按 session/agent 动态选择模型
- `tool-truncation`：统一工具输出截断服务，按行数/字节数截断，支持方向控制，所有工具统一接入
- `provider-transform`：Provider 消息转换模块，per-provider 消息规范化、cache 控制、providerOptions 映射

### Modified Capabilities

- `agent-modes`：新增 general/compaction/title/summary 四个 Agent 定义，增加 Agent 级 prompt 支持和 Permission 规则式工具过滤
- `react-loop`：Compaction 集成到 ReAct 循环（isOverflow → prune → process）、Snapshot/LSP/FileWatcher 接入点
- `builtin-tools`：新增 invalid 工具、read 增加 offset/limit、统一截断接入
- `cli-and-config`：JSONC 支持、model/small_model/instructions/compaction 等配置项、InstructionPrompt 文件查找
- `snapshot-restore`：wire 接入 Engine、Revert 集成 Snapshot.Restore
- `lsp-integration`：wire 接入 Engine 并注册 lsp 工具
- `file-monitoring`：write/edit/apply_patch 工具发布 file.changed 事件

## Impact

- `internal/runtime/`：Engine 需要支持多 Provider 选择、Agent 定义、Compaction 集成
- `internal/llm/`：新增 provider-routing 和 provider-transform 模块
- `internal/tool/`：所有工具接入 Truncate 服务、新增 invalid 工具、read 增强、write/edit/apply_patch 发布事件
- `internal/cli/wire.go`：接入 Snapshot、LSP、FileWatcher、新 Agent 定义
- `internal/config/`：JSONC 解析、新配置项
- `internal/session/` 或 `internal/runtime/`：新增 system prompt 构建逻辑
- 新增包：`internal/prompt/`（系统提示）、`internal/truncate/`（截断服务）
- 依赖：可能需要 JSONC 解析库
