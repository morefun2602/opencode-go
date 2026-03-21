# Capability: builtin-tools (delta)

## MODIFIED Requirements

### Requirement: 内置工具注册与 schema

系统 MUST 为每个内置工具提供稳定名称、JSON 参数 schema 以及标签集合（`Tags []string`）。标签 MUST 包括 `read`、`write`、`execute` 中的一个或多个。未通过校验的调用 MUST NOT 执行副作用。

#### Scenario: 工具包含标签

- **WHEN** 注册内置工具 `edit`
- **THEN** 工具定义 MUST 包含 `Tags: ["write"]`

#### Scenario: 非法参数被拒绝

- **WHEN** 模型或调用方传入不符合 schema 的参数
- **THEN** 系统 MUST NOT 执行该工具并 MUST 返回校验错误

## ADDED Requirements

### Requirement: 标签分类表

内置工具 MUST 按以下分类声明标签：
- `read`：read、glob、grep、webfetch、websearch
- `write`：write、edit、apply_patch、todowrite
- `execute`：bash、task
- `interact`：question

#### Scenario: 标签验证

- **WHEN** 系统启动并注册所有内置工具
- **THEN** 每个工具 MUST 至少有一个标签
