## ADDED Requirements

### Requirement: Wire 接入 Engine

LSP 模块 MUST 在 `wire.go` 中根据配置（`x_opencode_go.lsp.servers`）创建 LSP 客户端实例，并调用 `tool.RegisterLSP()` 注册 lsp 工具。当无 LSP 配置时 MUST 跳过。

#### Scenario: 有 LSP 配置时注册

- **WHEN** 配置包含 LSP 服务器定义
- **THEN** wire.go MUST 创建 LSP Client 并注册 lsp 工具

#### Scenario: 无 LSP 配置时跳过

- **WHEN** 配置不包含 LSP 服务器定义
- **THEN** wire.go MUST 跳过 LSP 初始化，lsp 工具 MUST NOT 注册

### Requirement: LSP 客户端生命周期管理

LSP 客户端 MUST 随 Engine 生命周期管理：Engine 关闭时 MUST 调用 LSP Client.Close() 优雅关闭语言服务器进程。

#### Scenario: Engine 关闭时关闭 LSP

- **WHEN** Engine 正在关闭
- **THEN** 系统 MUST 调用 LSP Client.Close() 发送 shutdown/exit

#### Scenario: LSP 进程异常退出

- **WHEN** 语言服务器进程意外退出
- **THEN** 后续 lsp 工具调用 MUST 返回错误信息说明 LSP 不可用，MUST NOT panic
