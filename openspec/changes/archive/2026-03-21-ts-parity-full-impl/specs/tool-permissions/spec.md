# Capability: tool-permissions

## ADDED Requirements

### Requirement: 三态权限模型

系统 MUST 为每个工具支持 `allow`（默认）、`ask`、`deny` 三种权限，通过 `x_opencode_go.permissions` 配置。

#### Scenario: deny 阻止执行

- **WHEN** 工具权限配置为 `deny` 且模型请求调用该工具
- **THEN** Engine MUST 返回拒绝类 tool_result 且 MUST NOT 执行工具逻辑

#### Scenario: allow 直接执行

- **WHEN** 工具权限配置为 `allow`（或未配置，即默认）
- **THEN** Engine MUST 直接执行工具逻辑而无需确认

### Requirement: ask 交互确认

当工具权限为 `ask` 时，Engine MUST 通过 `Confirm` 回调请求用户确认；用户拒绝时 MUST 返回拒绝类 tool_result 而非终止整个循环。

#### Scenario: REPL 中用户确认

- **WHEN** 工具权限为 `ask` 且在 REPL 模式下
- **THEN** 系统 MUST 在 stderr/stdout 提示用户确认工具调用细节（工具名 + 参数摘要），用户同意后执行

#### Scenario: HTTP 模式下 ask 降级

- **WHEN** 工具权限为 `ask` 但调用来源为 HTTP API（无交互界面）
- **THEN** 系统 MUST 将 `ask` 视为 `allow`（HTTP 端无法交互确认）

### Requirement: Confirm 回调注入

Engine MUST 接受可选的 `Confirm` 函数，由调用方（REPL / HTTP handler）在构造时注入。未注入时所有 `ask` 权限 MUST 降级为 `allow`。

#### Scenario: 未注入 Confirm

- **WHEN** Engine 未设置 Confirm 回调且权限为 `ask`
- **THEN** 工具 MUST 直接执行（等同 allow）
