# permission-patterns Specification

## Purpose

定义工具权限的 glob pattern 匹配、回复语义扩展与异步权限回复机制。

## Requirements

### Requirement: Glob Pattern 匹配

权限规则 MUST 支持 glob pattern 匹配工具名与参数。格式为 `tool_name:pattern`，其中 pattern 匹配工具的关键参数值（如文件路径）。未指定 pattern 的规则 MUST 匹配该工具的所有调用。

#### Scenario: 路径 pattern 匹配

- **WHEN** 权限规则为 `write:/tmp/*: allow` 且工具调用 write 的 path 为 `/tmp/foo.txt`
- **THEN** 权限 MUST 匹配为 allow

#### Scenario: 工具名 pattern

- **WHEN** 权限规则为 `bash: ask`
- **THEN** 所有 bash 工具调用 MUST 触发 ask 确认

### Requirement: 回复语义扩展

权限回复 MUST 支持 `once`（仅本次）、`always`（后续同 pattern 不再询问）、`reject`（拒绝且后续同 pattern 自动拒绝）三种语义。

#### Scenario: always 回复

- **WHEN** 用户对 `write:/src/*` 权限询问回复 always
- **THEN** 后续同 pattern 的 write 调用 MUST 自动 allow，不再询问

#### Scenario: reject 回复

- **WHEN** 用户对 `bash` 权限询问回复 reject
- **THEN** 后续同 pattern 的 bash 调用 MUST 自动 deny，不再询问

### Requirement: 异步权限回复端点

系统 MUST 提供 `POST /v1/permission/reply` 端点，接受 `{permission_id, action, scope}` 载荷（scope 为 once/always/reject），将回复传递给等待中的权限检查。

#### Scenario: HTTP 权限回复

- **WHEN** HTTP 客户端 POST 权限回复且 permission_id 匹配
- **THEN** 系统 MUST 将回复传递给阻塞中的权限检查并恢复 ReAct 循环
