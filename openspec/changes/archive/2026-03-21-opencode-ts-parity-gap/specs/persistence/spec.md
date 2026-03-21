# Delta: persistence

本文件为相对 `openspec/specs/persistence/spec.md` 的增量需求。

## ADDED Requirements

### Requirement: 上游逻辑实体映射

系统 MUST 维护与上游 Drizzle schema **语义等价**的核心实体（会话、消息、turn、附件元数据等）；物理表名 MAY 不同，但 MUST 提供映射文档与迁移测试覆盖不变量。

#### Scenario: 映射文档存在

- **WHEN** 维护者查阅仓库文档
- **THEN** MUST 能找到「上游实体 → 本地表/列」映射说明

### Requirement: 大版本迁移与用户数据

当实体模型发生 **BREAKING** 变更时，系统 MUST 提供向前迁移脚本且 MUST 在启动时检测不兼容数据并给出明确错误与备份建议。

#### Scenario: 不兼容数据拒绝启动

- **WHEN** 磁盘数据版本高于实现支持
- **THEN** 进程 MUST 拒绝启动并 MUST 提示升级或恢复备份

### Requirement: 索引与查询性能

对会话列表与消息分页查询，系统 MUST 创建必要索引或等价查询计划，使常见查询在文档化数据规模下 MUST 在可接受时间内完成（阈值在实现中定义并测试）。

#### Scenario: 分页查询使用索引

- **WHEN** 对大型会话执行分页加载
- **THEN** 查询 MUST NOT 全表扫描无界增长（以实现与测试验证）
