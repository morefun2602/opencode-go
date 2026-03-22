# lsp-integration Specification

## Purpose

定义 LSP（Language Server Protocol）客户端集成能力，包括语言服务器连接管理、诊断信息获取、跳转到定义、查找引用、文档符号查询以及通过内置工具暴露 LSP 功能。

## Requirements

### Requirement: LSP 客户端初始化

系统 MUST 提供 LSP 客户端模块（`internal/lsp/`），能够通过 stdio 传输启动和连接语言服务器进程。客户端 MUST 完成 LSP initialize/initialized 握手，并在关闭时发送 shutdown/exit。

#### Scenario: 成功连接语言服务器

- **WHEN** 系统配置了某语言（如 Go）的语言服务器命令
- **THEN** LSP 客户端 MUST 启动子进程、完成握手并进入就绪状态

#### Scenario: 语言服务器不可用

- **WHEN** 配置的语言服务器命令不存在或启动失败
- **THEN** 系统 MUST 返回明确错误且 MUST NOT 导致进程崩溃

### Requirement: 诊断信息获取

LSP 客户端 MUST 支持 `textDocument/publishDiagnostics` 通知的接收与缓存，并提供按文件路径查询当前诊断列表的接口。

#### Scenario: 获取文件诊断

- **WHEN** 对指定文件请求诊断信息
- **THEN** 系统 MUST 返回该文件的诊断列表（包含行号、消息、严重级别）

#### Scenario: 文件无诊断

- **WHEN** 指定文件无任何诊断
- **THEN** 系统 MUST 返回空列表

### Requirement: 跳转到定义

LSP 客户端 MUST 支持 `textDocument/definition` 请求，接受文件路径和位置（行/列），返回定义位置列表。

#### Scenario: 成功跳转

- **WHEN** 对已知符号的位置请求定义
- **THEN** 系统 MUST 返回包含文件路径和行列的定义位置

#### Scenario: 无定义结果

- **WHEN** 对无定义信息的位置请求
- **THEN** 系统 MUST 返回空结果

### Requirement: 查找引用

LSP 客户端 MUST 支持 `textDocument/references` 请求，返回指定符号在项目中的所有引用位置。

#### Scenario: 查找引用成功

- **WHEN** 对某符号请求引用
- **THEN** 系统 MUST 返回所有引用位置的列表

### Requirement: 文档符号

LSP 客户端 MUST 支持 `textDocument/documentSymbol` 请求，返回指定文件的符号树（函数、类、变量等）。

#### Scenario: 获取文档符号

- **WHEN** 对某文件请求文档符号
- **THEN** 系统 MUST 返回层级化的符号列表

### Requirement: LSP 工具暴露

系统 MUST 注册名为 `lsp` 的内置工具，接受操作类型（diagnostics、definition、references、symbols）和文件路径/位置参数，内部委托 LSP 客户端执行。该工具 MUST 标签为 `["read"]`。

#### Scenario: 通过工具获取诊断

- **WHEN** Agent 调用 lsp 工具并指定操作为 diagnostics、文件路径为 "main.go"
- **THEN** 工具 MUST 返回该文件的诊断信息文本

#### Scenario: LSP 未初始化时调用

- **WHEN** Agent 调用 lsp 工具但无可用的语言服务器
- **THEN** 工具 MUST 返回错误信息说明 LSP 不可用

### Requirement: Wire 接入 Engine

LSP 模块 MUST 在 `wire.go` 中根据配置（`lsp.servers`）创建 LSP 客户端实例，并调用 `tool.RegisterLSP()` 注册 lsp 工具。当无 LSP 配置时 MUST 跳过。

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
