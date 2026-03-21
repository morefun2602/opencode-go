# skills Specification

## Purpose

TBD

## Requirements

### Requirement: 技能发现

系统 MUST 在配置的搜索路径下以及默认搜索路径（`.cursor/skills/`、`.agents/skills/`）下发现技能定义；发现失败 MUST NOT 导致进程崩溃，且 MUST 可记录警告。搜索 MUST 递归扫描所有子目录，仅匹配文件名为 `SKILL.md`（大小写不敏感）的文件。

#### Scenario: 路径存在时加载列表

- **WHEN** 技能目录存在且包含有效技能
- **THEN** 系统 MUST 在运行时提供可枚举的技能元数据列表

#### Scenario: 递归发现

- **WHEN** 技能目录包含多层嵌套子目录
- **THEN** 系统 MUST 递归扫描所有子目录中的 SKILL.md 文件

#### Scenario: 仅匹配 SKILL.md

- **WHEN** 技能目录包含 `SKILL.md` 和 `README.md`
- **THEN** 系统 MUST 仅将 `SKILL.md` 作为技能文件，MUST NOT 加载 `README.md`

#### Scenario: 重复技能 warning

- **WHEN** 两个搜索路径下存在同名技能
- **THEN** 系统 MUST 使用先发现的（项目级优先），MUST 记录 warning 日志包含两个文件路径

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
6. 远程技能缓存目录：由 `config.skills.urls` 拉取

后发现的同名技能 MUST 被先发现的覆盖（项目级优先于用户级）。

#### Scenario: 项目级覆盖用户级

- **WHEN** 项目 `.cursor/skills/foo/SKILL.md` 和用户 `~/.cursor/skills/foo/SKILL.md` 均存在
- **THEN** 系统 MUST 使用项目级的定义

#### Scenario: 多路径发现

- **WHEN** 工作区和用户目录下均有 skills 目录
- **THEN** 系统 MUST 合并发现结果，冲突时项目级优先

#### Scenario: config.skills.paths 额外路径

- **WHEN** `config.skills.paths` 包含 `["~/my-skills", "custom/skills"]`
- **THEN** 系统 MUST 在标准路径之后搜索这些额外路径
- **AND** `~/` MUST 展开为用户主目录
- **AND** 相对路径 MUST 相对于工作区根目录解析

#### Scenario: 额外路径不存在

- **WHEN** `config.skills.paths` 包含不存在的路径
- **THEN** 系统 MUST 记录 warning 并跳过该路径

### Requirement: Skill 元数据解析

系统 MUST 解析 SKILL.md 文件的 YAML frontmatter（如存在），提取 `name`、`description`、`triggers` 等元数据字段。无 frontmatter 时 MUST 使用目录名作为名称、文件首行作为描述。

#### Scenario: 有 frontmatter 的 Skill

- **WHEN** SKILL.md 包含 YAML frontmatter 且定义了 name 和 description
- **THEN** 系统 MUST 使用 frontmatter 中的值

#### Scenario: 无 frontmatter 的 Skill

- **WHEN** SKILL.md 无 frontmatter
- **THEN** 系统 MUST 使用父目录名作为 Skill 名称

### Requirement: 远程技能发现

系统 MUST 支持 `config.skills.urls` 配置项（字符串数组）。对每个 URL，系统 MUST 请求 `{url}/index.json` 获取技能索引，下载技能文件到本地缓存目录，并将缓存目录加入技能搜索范围。

#### Scenario: 远程索引可用

- **WHEN** `config.skills.urls` 包含有效 URL 且远程 `index.json` 返回包含技能条目
- **THEN** 系统 MUST 下载所有包含 `SKILL.md` 的技能到本地缓存
- **AND** 这些技能 MUST 可被 `DiscoverSkills` 发现

#### Scenario: 远程索引不可达

- **WHEN** 远程 URL 请求失败
- **THEN** 系统 MUST 记录 warning 日志
- **AND** MUST NOT 阻塞其他技能的发现

### Requirement: 外部技能禁用

系统 MUST 支持 `OPENCODE_DISABLE_EXTERNAL_SKILLS` 环境变量。当该变量被设置时，系统 MUST NOT 扫描 `.cursor/skills` 和 `.agents/skills` 目录（无论是项目级还是用户级）。

#### Scenario: 禁用外部技能

- **WHEN** 设置 `OPENCODE_DISABLE_EXTERNAL_SKILLS=1`
- **THEN** 系统 MUST NOT 扫描 `.cursor/skills` 和 `.agents/skills` 目录
- **AND** 仅 `config.skills.paths` 和 `config.skills.urls` 来源的技能可用

### Requirement: 权限过滤

系统 MUST 支持基于 Agent 权限配置过滤可用技能列表。权限规则 `deny` 的技能 MUST NOT 出现在 `available()` 返回中，也 MUST NOT 出现在系统提示和工具描述中。

#### Scenario: deny 技能不可见

- **WHEN** Agent 权限配置 `skill: {"internal-*": "deny"}`
- **THEN** 名称匹配 `internal-*` 的技能 MUST NOT 出现在可用列表中

#### Scenario: 无权限配置时全部可用

- **WHEN** Agent 无 skill 权限配置
- **THEN** 所有已发现技能 MUST 出现在可用列表中

### Requirement: 技能目录集合跟踪

系统 MUST 跟踪所有已发现技能的目录路径集合。该集合 MUST 可通过 API 获取，供 SkillTool 列出技能目录下的文件。

#### Scenario: 目录可查询

- **WHEN** 发现了 5 个技能分布在 3 个目录
- **THEN** Dirs 集合 MUST 包含这 3 个目录路径

### Requirement: Location 字段

`Skill` 结构体 MUST 包含 `Location` 字段，值为该 SKILL.md 文件的绝对路径。系统提示的 verbose 格式 MUST 使用 `file://` URI 展示 location。

#### Scenario: Location 赋值

- **WHEN** 从 `/home/user/.cursor/skills/foo/SKILL.md` 加载技能
- **THEN** `Skill.Location` MUST 为该文件的绝对路径

### Requirement: 双模式格式化输出

系统 MUST 提供 `Fmt(skills, verbose bool)` 函数：
- verbose=true：输出 `<available_skills>` XML 格式，每个技能包含 `<name>`、`<description>`、`<location>` 子元素
- verbose=false：输出 Markdown 列表格式 `- **name**: description`

#### Scenario: verbose 模式

- **WHEN** 调用 `Fmt(skills, true)`
- **THEN** 输出 MUST 以 `<available_skills>` 开头，包含 XML 格式技能列表

#### Scenario: concise 模式

- **WHEN** 调用 `Fmt(skills, false)`
- **THEN** 输出 MUST 为 Markdown 列表格式

### ~~Requirement: LoadDir 函数~~ [REMOVED]

`LoadDir` 函数已被移除。所有技能加载 MUST 通过 `DiscoverSkills` 统一处理。

### ~~Requirement: InjectPrompt 函数~~ [REMOVED]

`InjectPrompt` 函数已被移除。技能注入 MUST 通过 `prompt.Build()` + `SkillSummary` 统一处理。
