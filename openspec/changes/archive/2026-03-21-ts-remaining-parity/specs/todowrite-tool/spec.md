# Capability: todowrite-tool

## ADDED Requirements

### Requirement: Todo 持久化

todowrite 工具 MUST 接受结构化的 todo 数组参数（每项含 `id`、`content`、`status`），将其写入当前会话关联的 todo 存储。后续 turn 中模型 MUST 可以通过系统提示看到当前 todo 状态。

#### Scenario: 创建并持久化 todo

- **WHEN** 模型调用 todowrite 传入 `[{"id":"1","content":"实现 X","status":"pending"}]`
- **THEN** todo MUST 被持久化到会话关联的存储中，后续系统提示 MUST 包含该 todo

#### Scenario: 更新 todo 状态

- **WHEN** 模型调用 todowrite 传入 `[{"id":"1","status":"completed"}]`（merge 模式）
- **THEN** 该 todo 的状态 MUST 更新为 completed，其余字段保持不变

### Requirement: Todo 注入系统提示

Engine MUST 在构建系统提示时检查当前会话是否有活跃 todo，若有 MUST 将 todo 列表序列化后附加到系统提示中。

#### Scenario: 系统提示包含 todo

- **WHEN** 会话存在未完成的 todo 且新一轮 turn 开始
- **THEN** 系统提示 MUST 包含 todo 列表的文本表示
