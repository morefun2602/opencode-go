## ADDED Requirements

### Requirement: invalid 工具

系统 MUST 注册名为 `invalid` 的内置工具，接受 `tool`（工具名）和 `error`（错误描述）参数。当 Engine 遇到畸形工具调用（schema 校验失败或参数无法解析）时 MUST 将调用路由到 invalid 工具。该工具 MUST 返回描述性错误信息供模型学习。该工具 MUST NOT 出现在提供给模型的活跃工具列表中。

#### Scenario: 畸形工具调用路由

- **WHEN** 模型返回的 tool_call 参数不符合目标工具的 schema
- **THEN** Engine MUST 将该调用路由到 invalid 工具，传入原始工具名和错误描述

#### Scenario: invalid 工具返回

- **WHEN** invalid 工具被调用
- **THEN** MUST 返回 `"The arguments provided to tool '<tool>' are invalid: <error>"` 格式的字符串

#### Scenario: 不在活跃工具列表

- **WHEN** Engine 收集工具定义传给 Provider
- **THEN** invalid 工具 MUST NOT 包含在工具列表中

### Requirement: read 工具 offset/limit

read 工具 MUST 支持可选的 `offset`（起始行号，1-based）和 `limit`（读取行数）参数。未提供时 MUST 读取全文件。单行超过 2000 字符时 MUST 截断该行并附加提示。

#### Scenario: 部分文件读取

- **WHEN** 调用 read 工具传入 offset=10 和 limit=20
- **THEN** MUST 返回文件第 10 到 29 行的内容

#### Scenario: 仅 offset 无 limit

- **WHEN** 调用 read 工具传入 offset=50 但未传 limit
- **THEN** MUST 返回从第 50 行到文件末尾的内容

#### Scenario: 长行截断

- **WHEN** 文件某行超过 2000 字符
- **THEN** 该行 MUST 被截断到 2000 字符并附加 `... (line truncated)` 提示

#### Scenario: offset 超出文件范围

- **WHEN** offset 大于文件总行数
- **THEN** MUST 返回空内容并附加提示说明文件仅有 N 行

## MODIFIED Requirements

### Requirement: 内置工具注册与 schema

系统 MUST 为每个内置工具提供稳定名称、JSON 参数 schema 以及标签集合（`Tags []string`）。标签 MUST 包括 `read`、`write`、`execute` 中的一个或多个。未通过校验的调用 MUST NOT 执行副作用。所有工具的输出 MUST 经过统一截断服务处理，各工具 MUST NOT 自行实现截断逻辑。

#### Scenario: 工具包含标签

- **WHEN** 注册内置工具 `edit`
- **THEN** 工具定义 MUST 包含 `Tags: ["write"]`

#### Scenario: 非法参数被拒绝

- **WHEN** 模型或调用方传入不符合 schema 的参数
- **THEN** 系统 MUST NOT 执行该工具并 MUST 返回校验错误

#### Scenario: 输出统一截断

- **WHEN** 任何工具返回超过截断限制的输出
- **THEN** Registry MUST 在返回前通过截断服务截断输出

### Requirement: 搜索与执行类工具

系统 MUST 规划 **grep** / **glob** 类搜索工具与 **bash**（或受控 shell）类执行工具；执行类工具 MUST 具备超时、输出上限与退出码捕获，且 MUST 将会话关联写入日志。工具 MUST NOT 自行截断输出，MUST 依赖统一截断服务。

#### Scenario: 执行超时

- **WHEN** shell 执行超过配置超时
- **THEN** 系统 MUST 终止该执行并 MUST 向编排层返回超时类错误

#### Scenario: 输出由截断服务处理

- **WHEN** bash 工具返回大量输出
- **THEN** 输出 MUST 由 Registry 层截断服务处理，bash 工具本身 MUST NOT 截断
