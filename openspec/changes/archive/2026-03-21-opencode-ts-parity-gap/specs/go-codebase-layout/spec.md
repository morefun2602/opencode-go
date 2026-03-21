# Delta: go-codebase-layout

本文件为相对 `openspec/specs/go-codebase-layout/spec.md` 的增量需求。

## ADDED Requirements

### Requirement: 横切能力包布局

系统 MUST 为 MCP、内置工具、技能、插件、ACP 适配层划分独立 `internal` 包（名称以实现为准，例如 `internal/mcp`、`internal/tool`、`internal/skill`、`internal/plugin`、`internal/acp`），且 MUST 禁止这些包直接 import `cmd` 或 HTTP 具体 handler 包。

#### Scenario: 依赖无环

- **WHEN** 执行 `go build ./...`
- **THEN** MUST NOT 出现违反上述方向的循环依赖（以模块图或 lint 规则验证）

### Requirement: 契约测试目录

系统 MUST 为 HTTP 与 CLI 的关键用户故事保留 `test` 或 `integration` 测试入口；新增能力 MUST 附带至少一条可自动化执行的契约或集成测试（可在后续任务中补齐具体路径）。

#### Scenario: CI 执行测试命令

- **WHEN** CI 运行 `go test ./...`
- **THEN** 新增包 MUST 不引入无法在无网络环境下运行的默认测试（除非显式 build tag）
