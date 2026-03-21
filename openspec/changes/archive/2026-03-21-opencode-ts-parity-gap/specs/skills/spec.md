# Capability: skills

## ADDED Requirements

### Requirement: 技能发现

系统 MUST 在配置的搜索路径下发现技能定义（格式以实现与上游对齐文档为准）；发现失败 MUST NOT 导致进程崩溃，且 MUST 可记录警告。

#### Scenario: 路径存在时加载列表

- **WHEN** 技能目录存在且包含有效技能
- **THEN** 系统 MUST 在运行时提供可枚举的技能元数据列表

### Requirement: 加载顺序与覆盖

当多个技能定义同名或冲突时，系统 MUST 应用文档化的优先级规则（例如更近工作区优先）；最终生效集 MUST 可观测（日志或调试接口）。

#### Scenario: 冲突可解析

- **WHEN** 两处定义同名技能
- **THEN** 系统 MUST 按规则选择其一并 MUST 在调试模式下记录选择原因

### Requirement: 与会话编排集成

系统 MUST 将已加载技能的指令或上下文注入策略与 `agent-runtime` 对齐（例如在会话开始或 turn 边界合并）；注入行为 MUST 可通过配置关闭或限定作用域。

#### Scenario: 关闭技能注入

- **WHEN** 用户或配置禁用技能
- **THEN** 编排 MUST NOT 向模型附加技能内容
