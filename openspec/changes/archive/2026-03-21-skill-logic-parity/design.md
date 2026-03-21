## Decisions

### D1: Skill 数据结构增强 — 增加 Location 字段

**选择**：在 `Skill` 结构体中增加 `Location string` 字段，存储 SKILL.md 文件的绝对路径。`Path` 字段保留兼容，`Location` 用于 verbose 格式输出。在 `Fmt()` verbose 模式中以 `file://` URI 格式展示。

**理由**：TS 版本 `Skill.Info` 包含 `location` 字段，在系统提示的 XML verbose 格式中使用 `pathToFileURL(skill.location).href`。Go 版本当前只有 `Path` 但未用于输出。保持两个字段是因为 `Path` 已在多处使用。

**影响**：`internal/skill/skill.go`。

---

### D2: DiscoverSkills 重构 — 收窄匹配范围

**选择**：修改 `DiscoverSkills` 仅匹配文件名为 `SKILL.md`（大小写不敏感），移除对任意 `.md` 文件的匹配。同时增加 `slog.Logger` 参数，在发现同名技能时输出 warning 日志。

**理由**：TS 版本仅扫描 `**/SKILL.md` 模式。当前 Go 版本匹配任意 `.md` 文件会导致误加载 README.md 等非技能文件。

**影响**：`internal/skill/skill.go`、`internal/cli/wire.go`（传入 logger）。

---

### D3: 双模式 Fmt 函数 — verbose XML / concise Markdown

**选择**：新增 `Fmt(skills []Skill, verbose bool) string` 函数。verbose=true 时输出 `<available_skills>` XML 格式（包含 name、description、location），verbose=false 时输出 Markdown 列表。现有 `SkillSummary` 改为调用 `Fmt(skills, true)` 并前缀说明文本。工具描述中使用 `Fmt(skills, false)`。

**理由**：TS 版本使用 verbose XML 格式在系统提示中注入技能列表（模型对 XML 结构化信息理解更好），在工具描述中使用 concise Markdown。Go 版本当前 `SkillSummary` 只有 Markdown 格式。

**影响**：`internal/skill/skill.go`（新增 Fmt）、`internal/prompt/prompt.go`（SkillSummary 改用 Fmt）。

---

### D4: SkillTool 输出增强 — `<skill_content>` XML 格式

**选择**：重构 `registerSkillTool` 返回格式为 `<skill_content name="...">` XML 块，包含：技能正文、base 目录（`file://` URI）、`<skill_files>` 子元素列出技能目录下文件（排除 SKILL.md，最多 10 个）。文件列表通过 `os.ReadDir` 递归获取。

**理由**：TS 版本的 SkillTool 返回结构化 XML 输出，包含文件列表，方便模型知道技能目录下有哪些参考文件可以读取。Go 版本当前仅返回纯 Body 文本。

**影响**：`internal/tool/skill_tool.go`。

---

### D5: SkillTool 动态 description — 根据可用技能生成

**选择**：将 `registerSkillTool` 改为接收技能列表后动态生成 description：包含使用说明和 `Fmt(skills, false)` 生成的 concise 列表。参数 `name` 的描述中包含前 3 个技能名作为示例。无技能时显示 "No skills are currently available"。

**理由**：TS 版本的 SkillTool 动态生成 description，包含可用技能列表和使用指引，帮助模型判断何时调用。Go 版本当前是静态描述。

**影响**：`internal/tool/skill_tool.go`。

---

### D6: Skills 配置结构 — Config 增加 Skills 字段

**选择**：在 `File.Go` 中增加 `Skills SkillsConfig` 字段（JSON 键 `skills`）。`SkillsConfig` 包含 `Paths []string` 和 `URLs []string`。在 `Config` 中增加对应字段。在 `merge` 和 `wireEngine` 中处理路径展开和搜索。

**理由**：TS 版本支持 `config.skills.paths` 和 `config.skills.urls`。Go 版本当前只有 `skills_dir` 单路径。

**影响**：`internal/config/config.go`。

---

### D7: 远程技能发现 — internal/skill/discovery.go

**选择**：新建 `internal/skill/discovery.go`，实现 `Discovery` 结构体。`Pull(url string) ([]string, error)` 方法：请求 `{url}/index.json`，解析为 `Index` 结构体（`Skills []IndexSkill`，每个含 `Name` 和 `Files`），过滤出含 SKILL.md 的条目，并发下载文件到 `{CacheDir}/skills/{name}/`，返回成功的目录列表。使用标准 `net/http` 客户端。

**理由**：TS 版本通过 `Discovery.pull` 支持从远程 URL 拉取技能。Go 版本当前完全没有远程发现能力。使用标准库 HTTP 而非 SDK，因为这是简单的文件下载。

**影响**：`internal/skill/discovery.go`（新建）、`internal/cli/wire.go`（集成调用）。

---

### D8: 外部技能禁用 — 环境变量控制

**选择**：在 `wireEngine` 中检查 `os.Getenv("OPENCODE_DISABLE_EXTERNAL_SKILLS")`。当值非空时，跳过 `.cursor/skills` 和 `.agents/skills` 搜索路径（包括项目级和用户级）。仅保留 `config.skills.paths`、`config.skills.urls` 和 `skills_dir` 来源。

**理由**：TS 版本通过 `Flag.OPENCODE_DISABLE_EXTERNAL_SKILLS` 控制外部技能发现。Go 版本需要同等能力。

**影响**：`internal/cli/wire.go`。

---

### D9: skills list CLI 对齐 — 使用多路径发现

**选择**：修改 `skills_cli.go` 的 `list` 命令：构建与 `wireEngine` 相同的搜索路径（项目级 + 用户级 + config.skills.paths + skills_dir），调用 `DiscoverSkills`。输出格式改为 `name: description`。

**理由**：当前 CLI 只扫描 `DataDir/skills`，与 Agent 实际可用技能不一致，用户 debug 困难。

**影响**：`internal/cli/skills_cli.go`。

---

### D10: 清理遗留代码 — 移除 InjectPrompt 和 LoadDir

**选择**：移除 `skill.InjectPrompt` 和 `skill.LoadDir` 函数。所有调用点已在前序变更或本次变更中迁移到 `DiscoverSkills` + `prompt.Build()`。确保没有外部引用后安全删除。

**理由**：`InjectPrompt` 是早期实现，已被 `prompt.Build()` + `SkillSummary` 替代。`LoadDir` 只扫描单层目录的 `.md` 文件，功能已被 `DiscoverSkills` 完全覆盖。

**影响**：`internal/skill/skill.go`。

## File Changes

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/skill/skill.go` | 修改 | Skill 结构体增加 Location；DiscoverSkills 收窄为仅 SKILL.md + 日志 + 重复 warning；新增 Fmt 双模式；移除 InjectPrompt/LoadDir |
| `internal/skill/discovery.go` | 新建 | Discovery 远程技能发现模块（Pull、Index 结构、HTTP 下载缓存） |
| `internal/tool/skill_tool.go` | 修改 | 输出 `<skill_content>` XML 格式；动态 description；文件列表 |
| `internal/prompt/prompt.go` | 修改 | SkillSummary 改用 Fmt verbose 模式；增加 skill 说明前缀 |
| `internal/config/config.go` | 修改 | 增加 SkillsConfig（Paths/URLs）、File.Go.Skills、Config.Skills、merge 逻辑 |
| `internal/cli/wire.go` | 修改 | 集成 config.skills.paths/urls、OPENCODE_DISABLE_EXTERNAL_SKILLS 检查、Discovery 调用 |
| `internal/cli/skills_cli.go` | 修改 | 使用多路径发现替代 LoadDir、输出格式增强 |
| `internal/skill/skill_test.go` | 修改 | 增加 DiscoverSkills 测试（SKILL.md 匹配、同名覆盖、多路径优先级） |
| `internal/skill/discovery_test.go` | 新建 | Discovery.Pull 测试（httptest 模拟服务端） |
| `internal/tool/skill_tool_test.go` | 新建 | SkillTool 输出格式和动态 description 测试 |
