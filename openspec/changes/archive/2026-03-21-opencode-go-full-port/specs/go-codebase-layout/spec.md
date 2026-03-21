# 能力：go-codebase-layout

## ADDED Requirements

### Requirement: 模块标识

系统应当在仓库根目录声明单一 Go 模块，模块路径为 `github.com/morefun2602/opencode-go`，并与 `go.mod` 一致。

#### Scenario: 模块路径与 import 一致

- **当** 本仓库源码 import 本仓库内的包
- **则** 所有 import 路径应当使用前缀 `github.com/morefun2602/opencode-go/`

### Requirement: 包布局与边界

系统应当将应用入口放在 `cmd/` 下，将**不得被外部模块 import** 的实现代码放在 `internal/` 下。系统不得在 `main` 包中除装配（wiring）之外放置生产业务逻辑。

#### Scenario: 外部无法编译 internal 包

- **当** 本仓库之外的消费者模块尝试 import `github.com/morefun2602/opencode-go/internal/...`
- **则** Go 工具链应当依据 `internal` 可见性规则拒绝构建

### Requirement: 质量门禁命令

系统应当能在模块根目录通过 `go build ./...` 构建，并通过 `go test ./...` 测试，且无需非标准工作目录布局。

#### Scenario: CI 执行标准 Go 命令

- **当** 自动化在模块根目录执行 `go vet ./...` 与 `go test ./...`
- **则** 上述命令应当在 `go.mod` 声明的受支持 Go 版本基线下成功完成

### Requirement: 稳定对外 API 面（可选）

若仓库向其他模块导出可复用包，这些包应当位于 `pkg/` 或文档化的顶层包；否则系统不得在 CLI 与 `http-api` 等规范定义的契约之外暴露稳定的库级 API。

#### Scenario: 避免意外暴露库 API

- **当** 项目按其他能力仅交付二进制与 HTTP 服务
- **则** 非测试应用代码应当保留在 `cmd/` 与 `internal/` 下，除非另有明确文档说明
