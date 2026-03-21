# Proposal: OpenCode 完整 Go 实现

## Why

上游 OpenCode 以 TypeScript/Bun 生态为主，团队希望在本仓库交付**可独立分发、易运维、符合 Go 惯例**的实现，以便统一二进制发布、降低运行时依赖，并与现有 Go 基础设施对齐。`opencode-go` 目前以 OpenSpec 与工具链为主，尚未承载业务代码，正是从零按 Go 最佳实践建模的时机。

## What Changes

- 在本仓库引入**新的 Go 模块与代码树**（`cmd/`、`internal/` 等），不是逐文件机械翻译，而是按域拆分包、明确边界与接口。
- 交付与上游**功能域对等**的实现路径：**CLI + 首版即提供的 HTTP API**、核心运行时装配、LLM/工具集成、持久化等；具体行为以本变更下的 **spec** 为准，避免“只翻译语句”导致结构腐烂。
- 建立 **Go 工程基线**：格式化、`go test`/`go vet`、模块边界、错误处理与日志约定、最小可复现的构建与发布流程（细节在 `design.md`）。
- **配置与上游一致**：**配置文件名与键名**与上游 OpenCode 对齐（见 `design.md` 与 `cli-and-config` spec），便于共用配置；Go 专有扩展须在 spec 中显式列出。
- **BREAKING**：对协议、HTTP 路由或与上游共用的配置键，任何**有意偏离**均需在对应 spec 中显式记录并说明迁移方式；除已约定可对齐的部分外，本仓库作为新代码基线，不保证与既有非 Go 发行版在**代码目录、二进制名**等其他方面完全一致。

## Capabilities

### New Capabilities

- `go-codebase-layout`：Go 模块与目录约定（例如 `cmd/*`、`internal/*` 分层）、可见性、可测试性、与 OpenSpec/CI 的衔接方式。
- `cli-and-config`：命令行界面、配置文件与环境变量、帮助与退出码约定；**须规定与上游一致的配置文件名与键名**及优先级；与用户操作路径相关的需求。
- `agent-runtime`：会话/任务生命周期、消息与编排、取消与超时、与“智能体主循环”相关的可观察行为（不含具体模型供应商细节）。
- `llm-and-tools`：模型提供商抽象、流式响应、工具/MCP 调用与结果回注；外部系统交互的契约与失败语义。
- `persistence`：本地状态与存储后端（如 SQLite/文件）、迁移与完整性；与重启恢复相关的需求。
- `http-api`：面向编辑器/远程客户端的 HTTP 服务（路由、请求/响应契约、流式与错误语义）、监听与关闭、**鉴权与 TLS 策略**（首版需明确基线，可与仅本地监听默认安全模型配合）。

### Modified Capabilities

- （无）——`openspec/specs/` 下尚无既有能力；本变更仅新增能力。

## Impact

- **代码**：`opencode-go` 将从以 OpenSpec 为主，扩展为包含 Go 源码、构建脚本与测试的主工程；Cursor/OpenSpec 工作流继续用于规格驱动开发。**Go module**：`github.com/morefun2602/opencode-go`。
- **API/协议**：首版**必须**暴露 HTTP API（见 `http-api` spec）；与编辑器/远程集成相关的契约不在 `llm-and-tools` 中混写业务路由细节；实现约定见 `design.md`。
- **依赖**：引入 Go 工具链与所选第三方库（ORM/驱动/HTTP 等），具体选型不在本提案展开。
- **运维/发布**：新增多平台二进制或容器产物时的发布渠道与版本策略，需在后续设计与任务中明确。
