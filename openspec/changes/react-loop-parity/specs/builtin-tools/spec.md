## MODIFIED Requirements

### Requirement: 工具名解析与路由

#### Scenario: 工具名大小写修复

- **WHEN** LLM 返回的 tool_call 名称为 "Read" 但注册名为 "read"
- **THEN** Router MUST 尝试小写匹配并使用匹配到的工具执行
- **AND** MUST NOT 路由到 invalid 工具

#### Scenario: 大小写修复失败

- **WHEN** LLM 返回的 tool_call 名称为 "UnknownTool" 且小写 "unknowntool" 也不存在
- **THEN** Router MUST 路由到 invalid 工具
