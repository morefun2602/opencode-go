## MODIFIED Requirements

### Requirement: 技能列表注入

#### Scenario: verbose 格式注入系统提示

- **WHEN** 构建系统提示中的技能摘要
- **THEN** 系统 MUST 使用 verbose 模式（`<available_skills>` XML 格式）
- **AND** 每个技能 MUST 包含 `<name>`、`<description>`、`<location>` 子元素

#### Scenario: 技能禁用时不注入

- **WHEN** Agent 权限配置禁用了 skill 工具
- **THEN** 系统提示 MUST NOT 包含任何技能相关内容

#### Scenario: 技能说明前缀

- **WHEN** 系统提示注入技能列表
- **THEN** 技能列表前 MUST 包含说明文本：介绍 skills 的作用并提示使用 skill 工具加载完整指令
