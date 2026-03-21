# Delta: llm-and-tools

本文件为相对 `openspec/specs/llm-and-tools/spec.md` 的增量需求。

## ADDED Requirements

### Requirement: 多提供商注册表

系统 MUST 维护可扩展的 LLM 提供商注册表，且 MUST 允许通过配置选择提供商；注册表 MUST 与 `mcp-integration` 及内置工具路由在依赖上保持单向（提供商实现 MUST NOT 直接依赖 HTTP handler）。

#### Scenario: 切换提供商

- **WHEN** 配置更改提供商标识并重启或热加载（若支持）
- **THEN** 补全请求 MUST 路由到新提供商或 MUST 返回明确错误

### Requirement: 工具来源统一路由

系统 MUST 将内置工具与 MCP 工具统一纳入同一调用接口，使模型产生的 tool_calls 能解析到唯一实现；解析失败 MUST 返回结构化错误。

#### Scenario: 未知工具名

- **WHEN** 模型请求未注册工具名
- **THEN** 系统 MUST NOT 执行并 MUST 返回错误给编排层

### Requirement: 失败分类与可重试提示

系统 MUST 将提供商与工具错误分类（例如超时、429、认证失败）；对可重试错误，响应或日志 MUST 携带可重试提示（若 `agent-runtime` 启用重试策略）。

#### Scenario: 超时可识别

- **WHEN** 提供商请求超时
- **THEN** 错误类型 MUST 与超时分类一致且 MUST 可被上层识别
