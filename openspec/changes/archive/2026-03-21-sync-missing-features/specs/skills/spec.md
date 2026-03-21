## ADDED Requirements

### Requirement: Skill 工具集成

系统 MUST 提供名为 `skill` 的内置工具，允许 Agent 在运行时按名称加载和查看 Skill 内容。Skill 工具 MUST 支持列出所有可用 Skill 和加载指定 Skill 的完整指令。

#### Scenario: 列出可用 Skill

- **WHEN** Agent 调用 skill 工具且未指定名称（或指定 action 为 "list"）
- **THEN** 系统 MUST 返回所有已发现 Skill 的名称和描述列表

#### Scenario: 加载指定 Skill

- **WHEN** Agent 调用 skill 工具并指定有效 Skill 名称
- **THEN** 系统 MUST 返回该 Skill 的完整 Markdown 指令内容

### Requirement: 增强技能发现路径

系统 MUST 按以下顺序搜索技能定义：
1. 项目级：`{workspace}/.cursor/skills/**/SKILL.md`
2. 项目级：`{workspace}/.agents/skills/**/SKILL.md`
3. 用户级：`~/.cursor/skills/**/SKILL.md`
4. 用户级：`~/.agents/skills/**/SKILL.md`
5. 配置指定的额外路径：`config.skills.paths`

后发现的同名技能 MUST 被先发现的覆盖（项目级优先于用户级）。

#### Scenario: 项目级覆盖用户级

- **WHEN** 项目 `.cursor/skills/foo/SKILL.md` 和用户 `~/.cursor/skills/foo/SKILL.md` 均存在
- **THEN** 系统 MUST 使用项目级的定义

#### Scenario: 多路径发现

- **WHEN** 工作区和用户目录下均有 skills 目录
- **THEN** 系统 MUST 合并发现结果，冲突时项目级优先

### Requirement: Skill 元数据解析

系统 MUST 解析 SKILL.md 文件的 YAML frontmatter（如存在），提取 `name`、`description`、`triggers` 等元数据字段。无 frontmatter 时 MUST 使用目录名作为名称、文件首行作为描述。

#### Scenario: 有 frontmatter 的 Skill

- **WHEN** SKILL.md 包含 YAML frontmatter 且定义了 name 和 description
- **THEN** 系统 MUST 使用 frontmatter 中的值

#### Scenario: 无 frontmatter 的 Skill

- **WHEN** SKILL.md 无 frontmatter
- **THEN** 系统 MUST 使用父目录名作为 Skill 名称

## MODIFIED Requirements

### Requirement: 技能发现

系统 MUST 在配置的搜索路径下以及默认搜索路径（`.cursor/skills/`、`.agents/skills/`）下发现技能定义；发现失败 MUST NOT 导致进程崩溃，且 MUST 可记录警告。搜索 MUST 递归扫描所有子目录中的 SKILL.md 文件。

#### Scenario: 路径存在时加载列表

- **WHEN** 技能目录存在且包含有效技能
- **THEN** 系统 MUST 在运行时提供可枚举的技能元数据列表

#### Scenario: 递归发现

- **WHEN** 技能目录包含多层嵌套子目录
- **THEN** 系统 MUST 递归扫描所有子目录中的 SKILL.md 文件
