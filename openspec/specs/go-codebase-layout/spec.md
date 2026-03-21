# go-codebase-layout Specification

## Purpose

定义 Go 模块布局、包边界与质量门禁要求。

## Requirements

### Requirement: 模块标识

系统 MUST 在仓库根目录声明单一 Go 模块，模块路径为 `github.com/morefun2602/opencode-go`，并与 `go.mod` 一致。

#### Scenario: 模块路径与 import 一致

- **WHEN** 本仓库源码 import 本仓库内的包
- **THEN** 所有 import 路径 MUST 使用前缀 `github.com/morefun2602/opencode-go/`

### Requirement: 包布局与边界

系统 MUST 将应用入口放在 `cmd/` 下，将不得被外部模块 import 的实现代码放在 `internal/` 下。系统 MUST NOT 在 `main` 包中除装配（wiring）之外放置生产业务逻辑。

#### Scenario: 外部无法编译 internal 包

- **WHEN** 本仓库之外的消费者模块尝试 import `github.com/morefun2602/opencode-go/internal/...`
- **THEN** Go 工具链 MUST 依据 `internal` 可见性规则拒绝构建

### Requirement: 质量门禁命令

系统 MUST 能在模块根目录通过 `go build ./...` 构建，并通过 `go test ./...` 测试，且无需非标准工作目录布局。

#### Scenario: CI 执行标准 Go 命令

- **WHEN** 自动化在模块根目录执行 `go vet ./...` 与 `go test ./...`
- **THEN** 上述命令 MUST 在 `go.mod` 声明的受支持 Go 版本基线下成功完成

### Requirement: 稳定对外 API 面（可选）

若仓库向其他模块导出可复用包，这些包 MUST 位于 `pkg/` 或文档化的顶层包；否则系统 MUST NOT 在 CLI 与 `http-api` 等规范定义的契约之外暴露稳定的库级 API。

#### Scenario: 避免意外暴露库 API

- **WHEN** 项目按其他能力仅交付二进制与 HTTP 服务
- **THEN** 非测试应用代码 MUST 保留在 `cmd/` 与 `internal/` 下，除非另有明确文档说明

### Requirement: 横切能力包布局

系统 MUST 为 MCP、内置工具、技能、插件、ACP 适配层划分独立 `internal` 包（例如 `internal/mcp`、`internal/tool`、`internal/skill`、`internal/plugin`、`internal/acp`），且 MUST 禁止这些包直接 import `cmd` 或 HTTP 具体 handler 包。

#### Scenario: 依赖无环

- **WHEN** 执行 `go build ./...`
- **THEN** MUST NOT 出现违反上述方向的循环依赖

### Requirement: 契约测试目录

系统 MUST 为 HTTP 与 CLI 的关键用户故事保留 `test` 或 `integration` 测试入口；新增能力 MUST 附带至少一条可自动化执行的契约或集成测试。

#### Scenario: CI 执行测试命令

- **WHEN** CI 运行 `go test ./...`
- **THEN** 新增包 MUST NOT 引入无法在无网络环境下运行的默认测试（除非显式 build tag）
