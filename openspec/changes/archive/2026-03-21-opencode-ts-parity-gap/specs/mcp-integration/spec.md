# Capability: mcp-integration

## ADDED Requirements

### Requirement: MCP 客户端连接与发现

系统 MUST 支持作为 MCP **客户端**连接已配置的 MCP 服务端（传输方式以实现为准，例如 stdio 或 HTTP），并在连接成功后 MUST 拉取工具/资源清单；连接失败时 MUST 返回可分类错误并记录日志，且 MUST NOT 静默忽略。

#### Scenario: 配置存在时尝试连接

- **WHEN** 配置中声明至少一个 MCP 服务端且进程启动或显式连接
- **THEN** 系统 MUST 建立连接或返回明确错误，且工具清单 MUST 可被内部注册表消费

### Requirement: MCP 工具调用与回注

系统 MUST 将 MCP 暴露的工具纳入统一工具路由，使 `llm-and-tools` 中的模型循环可按名称调用；调用结果 MUST 以结构化形式回注编排层，失败语义 MUST 与内置工具一致（不得静默失败）。

#### Scenario: 调用失败可观测

- **WHEN** 对某 MCP 工具调用因网络或协议错误失败
- **THEN** 智能体循环 MUST 收到失败结果且日志 MUST 包含关联标识

### Requirement: 与内置工具命名空间

系统 MUST 为 MCP 工具名实施可配置的命名空间或前缀策略，以避免与 `builtin-tools` 冲突；冲突解析规则 MUST 文档化。

#### Scenario: 名称冲突可配置

- **WHEN** MCP 工具名与内置工具名相同
- **THEN** 系统 MUST 按文档化规则解析或拒绝并返回错误
