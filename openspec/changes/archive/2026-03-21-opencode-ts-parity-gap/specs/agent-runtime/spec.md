# Delta: agent-runtime

本文件为相对 `openspec/specs/agent-runtime/spec.md` 的增量需求。

## ADDED Requirements

### Requirement: 压缩与摘要（Compaction）

系统 MUST 支持可配置的对话压缩或摘要策略，使长会话在超过阈值时 MUST 能生成摘要或裁剪上下文且 MUST 保持用户意图可追溯（策略细节以实现为准）。

#### Scenario: 超长会话触发策略

- **WHEN** 会话 token 或消息条数超过配置阈值
- **THEN** 系统 MUST 应用压缩或摘要且 MUST 继续允许新 turn

### Requirement: 重试与回退（Retry / Revert）

系统 MUST 为模型或工具失败提供可配置的重试策略；在支持的场景下 MUST 提供回退到先前消息状态的能力（与 `persistence` 协同），且 MUST NOT 在无确认时丢弃用户数据。

#### Scenario: 重试次数上限

- **WHEN** 连续失败达到配置上限
- **THEN** 系统 MUST 停止重试并 MUST 向用户或客户端报告

### Requirement: 结构化输出模式

当配置或模型支持结构化输出时，系统 MUST 校验输出是否符合声明的 schema；校验失败 MUST 反馈给编排层且 MUST NOT 当作成功完成。

#### Scenario: schema 校验失败

- **WHEN** 模型返回不符合 schema 的 JSON
- **THEN** 系统 MUST 返回校验错误且 MUST NOT 持久化为成功助手消息

### Requirement: 消息模型版本

系统 MUST 为消息存储与 API 暴露版本或 `schema` 字段，以支持与上游 **message v2** 等模型的分阶段对齐；旧客户端 MUST 在未升级时仍能获得向后兼容视图或明确 **BREAKING** 说明。

#### Scenario: 版本字段存在

- **WHEN** 客户端读取消息资源
- **THEN** 响应 MUST 包含版本或模型标识字段或文档化等价物
