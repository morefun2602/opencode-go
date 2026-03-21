# persistence Specification

## Purpose

定义持久化层的存储、迁移与数据完整性要求。

## Requirements

### Requirement: 会话状态持久存储

在启用持久化时，系统 MUST 将会话元数据与消息历史写入 SQLite 持久本地存储。消息行 MUST 包含 `parts`（JSON 序列化的结构化部件数组）、`model`、`cost_prompt_tokens`、`cost_completion_tokens`、`finish_reason`、`tool_call_id` 列，使消息可被完整还原。

#### Scenario: 重启后含结构化消息的会话恢复

- **WHEN** 某工作区启用持久化且含 tool_call/tool_result 消息，进程重启后再次启动
- **THEN** 系统 MUST 重新加载全部消息且 `parts` JSON MUST 可被反序列化为原始结构

### Requirement: Schema 迁移

系统 MUST 为数据库 schema 维护版本，并在启动时自动应用向前迁移。v2→v3 迁移 MUST 添加 `parts`、`model`、`cost_prompt_tokens`、`cost_completion_tokens`、`finish_reason`、`tool_call_id` 列，并将现有 `body` 内容包装为 `[{"type":"text","text":"<body>"}]` 写入 `parts`。若磁盘上 schema 版本高于二进制所支持的上限，系统 MUST 拒绝启动并给出明确错误信息。

#### Scenario: v2 → v3 迁移

- **WHEN** 打开 schema v2 的数据库
- **THEN** 迁移后所有既有消息 MUST 有有效 `parts` JSON，新增列 MUST 存在，`body` 列 MUST 保持不变

#### Scenario: 旧二进制拒绝新库

- **WHEN** 数据库 schema 版本大于二进制支持的最大版本
- **THEN** 进程 MUST 以非零状态退出并打印错误

### Requirement: 数据完整性

更新关联行（消息与会话）的写操作 MUST 在事务层面原子化。一轮 ReAct 循环产生的全部消息（user + assistant + tool 多条）MUST 在单一事务中写入。失败事务产生的部分写入 MUST NOT 使存储处于不一致状态。

#### Scenario: ReAct 循环消息原子写入

- **WHEN** 一轮 ReAct 循环产生 5 条消息且第 4 条写入时发生 IO 错误
- **THEN** 全部 5 条消息 MUST NOT 被提交，存储 MUST 回滚到事务前状态

### Requirement: 工作区隔离

不同工作区根路径的数据 MUST 相互隔离：对某一工作区的查询 MUST NOT 返回属于另一工作区标识的行。

#### Scenario: 查询限定在工作区

- **WHEN** 运行时为工作区 A 加载历史
- **THEN** 结果集 MUST 排除存储在工作区 B 下的消息

### Requirement: 上游逻辑实体映射

系统 MUST 维护与上游 Drizzle schema 语义等价的核心实体（会话、消息、turn、附件元数据等）；物理表名 MAY 不同，但 MUST 提供映射文档与迁移测试覆盖不变量。

#### Scenario: 映射文档存在

- **WHEN** 维护者查阅仓库文档
- **THEN** MUST 能找到「上游实体 → 本地表/列」映射说明

### Requirement: 大版本迁移与用户数据

当实体模型发生 BREAKING 变更时，系统 MUST 提供向前迁移脚本且 MUST 在启动时检测不兼容数据并给出明确错误与备份建议。

#### Scenario: 不兼容数据拒绝启动

- **WHEN** 磁盘数据版本高于实现支持
- **THEN** 进程 MUST 拒绝启动并 MUST 提示升级或恢复备份

### Requirement: 索引与查询性能

对会话列表与消息分页查询，系统 MUST 创建必要索引或等价查询计划，使常见查询在文档化数据规模下 MUST 在可接受时间内完成。

#### Scenario: 分页查询使用索引

- **WHEN** 对大型会话执行分页加载
- **THEN** 查询 MUST NOT 全表扫描无界增长
