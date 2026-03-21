## MODIFIED Requirements

### Requirement: skill 工具

#### Scenario: 结构化输出格式

- **WHEN** 调用 skill 工具并传入有效技能名称
- **THEN** 输出 MUST 使用 `<skill_content name="...">` XML 格式
- **AND** 输出 MUST 包含技能正文、base 目录（`file://` URI）、技能目录下的文件列表（`<skill_files>` 子元素）
- **AND** 文件列表 MUST 排除 SKILL.md 本身
- **AND** 文件列表 MUST 最多包含 10 个文件

#### Scenario: 动态工具描述

- **WHEN** 注册 skill 工具时
- **THEN** 工具描述 MUST 根据当前可用技能列表动态生成
- **AND** 描述 MUST 包含每个技能的名称和摘要（concise 格式）
- **AND** 参数 `name` 的描述 MUST 包含示例技能名称（最多 3 个）

#### Scenario: 无可用技能时

- **WHEN** 可用技能列表为空
- **THEN** 工具描述 MUST 说明当前无可用技能

#### Scenario: 权限校验

- **WHEN** 调用 skill 工具加载技能
- **THEN** 系统 MUST 通过 Policy 进行权限检查
- **AND** 权限为 `deny` 时 MUST 拒绝加载
- **AND** 权限为 `ask` 时 MUST 请求用户确认
