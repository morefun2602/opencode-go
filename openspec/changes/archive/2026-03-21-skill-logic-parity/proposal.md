## Why

深入对比 TypeScript 和 Go 两个版本的 Skill 系统实现后，发现 Go 版本在多项关键能力上与 TS 版本存在显著差距。SkillTool 输出过于简陋（仅返回纯文本 body，无文件列表、无权限校验、无动态 description）；系统提示中的技能摘要缺少 XML verbose 格式；发现路径缺少 `config.skills.paths` 数组和 worktree 向上遍历；无远程技能发现（`config.skills.urls`）；无权限过滤机制；`skills list` CLI 与 Agent 实际使用的多路径发现逻辑不一致；遗留的 `InjectPrompt` 函数未清理。这些差距直接影响 Skill 系统在企业级场景下的完整性和可扩展性。

## What Changes

### 核心能力对齐

- **SkillTool 输出增强**：返回结构化 `<skill_content>` XML 格式，包含技能名称、正文、base 目录、技能目录下的文件列表（最多 10 个）
- **SkillTool 动态 description**：根据当前可用技能列表动态生成工具描述，包含技能名称和摘要，帮助模型理解何时调用
- **SkillTool 权限校验**：加载技能前通过 Policy 进行权限检查（allow/deny/ask）
- **系统提示 XML 格式**：`SkillSummary` 增加 verbose 模式，输出 `<available_skills>` XML 格式（含 name、description、location），系统提示使用 verbose，工具描述使用 concise

### 发现与加载增强

- **`config.skills.paths` 支持**：Config 中增加 `Skills.Paths []string` 字段，支持额外技能搜索路径（支持 `~/` 展开和相对路径解析）
- **Worktree 向上遍历**：从当前目录向上遍历到 worktree 根目录，搜索 `.cursor/skills`、`.agents/skills`
- **远程技能发现**：实现 `Discovery` 模块，支持 `config.skills.urls`，从远程 URL 拉取 `index.json` 索引并下载技能文件到本地缓存
- **外部技能禁用**：支持 `OPENCODE_DISABLE_EXTERNAL_SKILLS` 环境变量，禁用 `.cursor/skills`、`.agents/skills` 下的外部技能

### 正确性修复

- **`skills list` CLI 与 Agent 对齐**：CLI 的 `skills list` 命令使用与 `wireEngine` 相同的多路径发现逻辑，而非仅扫描 `DataDir/skills`
- **`DiscoverSkills` 范围收窄**：仅匹配 `SKILL.md` 文件（与 TS 对齐），不再匹配任意 `.md`
- **重复技能警告**：发现同名技能时记录 warning 日志，包含已存在和重复的路径
- **清理遗留代码**：移除不再使用的 `InjectPrompt` 函数和 `LoadDir` 函数（功能已被 `DiscoverSkills` 覆盖）

### Skill 数据结构增强

- **增加 `Location` 字段**：`Skill` 结构体增加 `Location` 字段（`file://` URI），与 TS 的 `location` 字段对齐
- **Dirs 跟踪**：记录所有已发现技能的目录集合，供 SkillTool 使用

## Capabilities

### New Capabilities

- `skill-discovery`：远程技能发现模块，通过 HTTP 从 URL 拉取索引并下载技能文件到本地缓存

### Modified Capabilities

- `skills`：发现路径增强（config.skills.paths、worktree 遍历、远程 URL）、权限过滤、外部技能禁用、数据结构增强、重复告警、范围收窄、遗留清理
- `builtin-tools`：SkillTool 输出格式（`<skill_content>` XML + 文件列表）、动态 description、权限校验
- `system-prompt`：`SkillSummary` 增加 verbose/concise 双模式（XML vs Markdown）
- `cli-and-config`：`skills list` 使用多路径发现、Config 增加 `skills.paths` 和 `skills.urls`

## Impact

- `internal/skill/skill.go`：数据结构增强、`DiscoverSkills` 重构（收窄范围、worktree 遍历、config.paths）、新增 `Fmt` 双模式输出、移除 `InjectPrompt`/`LoadDir`
- `internal/skill/discovery.go`（新增）：远程技能发现模块
- `internal/tool/skill_tool.go`：输出格式重构（`<skill_content>` XML）、动态 description、权限校验、文件列表
- `internal/prompt/prompt.go`：`SkillSummary` 增加 verbose 参数
- `internal/cli/wire.go`：集成 config.skills.paths、config.skills.urls、外部技能禁用开关
- `internal/cli/skills_cli.go`：使用多路径发现替代 `LoadDir`
- `internal/config/config.go`：增加 `Skills` 配置结构（Paths、URLs）
- 无破坏性变更，全部为增量补齐和增强
