# builtin-tools Specification

## Purpose

TBD

## Requirements

### Requirement: 内置工具注册与 schema

系统 MUST 为每个内置工具提供稳定名称、JSON 参数 schema 以及标签集合（`Tags []string`）。标签 MUST 包括 `read`、`write`、`execute` 中的一个或多个。未通过校验的调用 MUST NOT 执行副作用。

#### Scenario: 工具包含标签

- **WHEN** 注册内置工具 `edit`
- **THEN** 工具定义 MUST 包含 `Tags: ["write"]`

#### Scenario: 非法参数被拒绝

- **WHEN** 模型或调用方传入不符合 schema 的参数
- **THEN** 系统 MUST NOT 执行该工具并 MUST 返回校验错误

### Requirement: 读文件与列目录类工具

系统 MUST 提供与上游语义对齐的只读类工具（至少包含 **read** 与目录列举能力之一；名称以实现对齐文档为准），且 MUST 将路径解析限制在工作区根或其允许范围内。

#### Scenario: 越界路径被拒绝

- **WHEN** 请求访问工作区根之外的禁止路径
- **THEN** 工具 MUST 失败并返回明确错误

### Requirement: 写文件与编辑类工具

系统 MUST 提供写入与编辑类能力（至少覆盖 **write**、**edit** 或与上游等价的 **apply_patch** 之一的分阶段落地），且 MUST 在覆盖或删除前遵守 `agent-runtime` / 配置中的确认策略（若启用）。

#### Scenario: 需确认时未确认则不写

- **WHEN** 策略要求用户确认写操作且未收到确认
- **THEN** 系统 MUST NOT 提交磁盘写入

### Requirement: 搜索与执行类工具

系统 MUST 规划 **grep** / **glob** 类搜索工具与 **bash**（或受控 shell）类执行工具；执行类工具 MUST 具备超时、输出上限与退出码捕获，且 MUST 将会话关联写入日志。

#### Scenario: 执行超时

- **WHEN** shell 执行超过配置超时
- **THEN** 系统 MUST 终止该执行并 MUST 向编排层返回超时类错误

### Requirement: 标签分类表

内置工具 MUST 按以下分类声明标签：
- `read`：read、glob、grep、webfetch、websearch
- `write`：write、edit、apply_patch、todowrite
- `execute`：bash、task
- `interact`：question

#### Scenario: 标签验证

- **WHEN** 系统启动并注册所有内置工具
- **THEN** 每个工具 MUST 至少有一个标签
