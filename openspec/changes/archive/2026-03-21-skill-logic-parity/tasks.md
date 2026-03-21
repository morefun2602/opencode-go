## 1. Config 增加 Skills 配置结构（D6）

- [x] 1.1 在 `internal/config/config.go` 新增 `SkillsConfig` 结构体：`Paths []string` (json:"paths")、`URLs []string` (json:"urls")
- [x] 1.2 在 `File.Go` 中增加 `Skills SkillsConfig` 字段 (json:"skills")
- [x] 1.3 在 `Config` 中增加 `Skills SkillsConfig` 字段
- [x] 1.4 在 `merge()` 中增加 Skills 字段合并逻辑：paths 和 urls 非空时覆盖

## 2. Skill 数据结构增强（D1）

- [x] 2.1 在 `internal/skill/skill.go` 的 `Skill` 结构体中增加 `Location string` 字段
- [x] 2.2 在 `DiscoverSkills` 中为每个发现的 Skill 设置 `Location` 为绝对路径

## 3. DiscoverSkills 重构（D2）

- [x] 3.1 修改 `DiscoverSkills` 签名：增加 `log *slog.Logger` 参数
- [x] 3.2 修改文件匹配逻辑：仅匹配文件名为 `SKILL.md`（`strings.EqualFold(name, "SKILL.md")`），移除对任意 `.md` 的匹配
- [x] 3.3 发现同名技能时通过 logger 输出 warning，包含已存在路径和重复路径
- [x] 3.4 编写 `DiscoverSkills` 单元测试：创建临时目录结构，验证仅 SKILL.md 被匹配、同名覆盖行为、多路径优先级

## 4. 双模式 Fmt 函数（D3）

- [x] 4.1 在 `internal/skill/skill.go` 新增 `Fmt(skills []Skill, verbose bool) string` 函数
- [x] 4.2 verbose=true 时输出 `<available_skills>` XML 格式：每个 skill 含 `<name>`、`<description>`、`<location>`（`url.URL{Scheme: "file", Path: s.Location}.String()`）
- [x] 4.3 verbose=false 时输出 Markdown 列表：`- **name**: description`
- [x] 4.4 空列表时返回 "No skills are currently available."
- [x] 4.5 编写 Fmt 的单元测试

## 5. 清理遗留代码（D10）

- [x] 5.1 移除 `skill.InjectPrompt` 函数
- [x] 5.2 移除 `skill.LoadDir` 函数
- [x] 5.3 搜索所有引用确保无外部调用（`wire.go` 中的 `LoadDir` 回退逻辑一并移除）
- [x] 5.4 更新 `skill_test.go`：移除 `LoadDir` 相关测试

## 6. SkillSummary 改用 verbose 模式（D3）

- [x] 6.1 修改 `internal/prompt/prompt.go` 的 `SkillSummary`：调用 `skill.Fmt(skills, true)` 替代手动拼装
- [x] 6.2 在 Fmt 输出前增加说明前缀："Skills provide specialized instructions and workflows for specific tasks.\nUse the `skill` tool to load a skill when a task matches its description."
- [x] 6.3 更新 prompt 相关测试

## 7. SkillTool 输出增强（D4 + D5）

- [x] 7.1 修改 `internal/tool/skill_tool.go`：加载技能时构建 `<skill_content name="...">` XML 输出
- [x] 7.2 在输出中包含技能正文（`s.Body`）、base 目录（`file://` URI of `filepath.Dir(s.Path)`）
- [x] 7.3 实现文件列表扫描：`os.ReadDir` 遍历技能目录，排除 SKILL.md，最多收集 10 个文件路径，包装为 `<skill_files>` XML
- [x] 7.4 修改工具注册：根据技能列表动态生成 description，包含 `skill.Fmt(skills, false)` 和使用说明
- [x] 7.5 参数 `name` 的描述中包含前 3 个技能名作为示例
- [x] 7.6 无技能时 description 显示 "No skills are currently available."
- [x] 7.7 编写 SkillTool 的单元测试：验证输出格式和动态 description

## 8. 远程技能发现模块（D7）

- [x] 8.1 新建 `internal/skill/discovery.go`，定义 `IndexSkill` 结构体（`Name string`、`Files []string`）和 `Index` 结构体（`Skills []IndexSkill`）
- [x] 8.2 实现 `Discovery` 结构体，包含 `CacheDir string`、`Log *slog.Logger`、`Client *http.Client`
- [x] 8.3 实现 `Pull(url string) ([]string, error)`：请求 `{url}/index.json`、解析 Index、过滤含 SKILL.md 的条目
- [x] 8.4 实现文件下载逻辑：并发下载到 `{CacheDir}/skills/{name}/`，已存在文件跳过
- [x] 8.5 下载失败时记录错误日志，SKILL.md 未成功的条目不包含在返回中
- [x] 8.6 编写 `discovery_test.go`：使用 `httptest.NewServer` 模拟远程服务端，测试正常拉取、索引不可达、部分下载失败

## 9. wireEngine 集成增强（D8）

- [x] 9.1 在 `wireEngine` 中检查 `os.Getenv("OPENCODE_DISABLE_EXTERNAL_SKILLS")`：非空时跳过 `.cursor/skills` 和 `.agents/skills` 搜索路径
- [x] 9.2 从 `cfg.Skills.Paths` 读取额外路径：`~/` 展开为 `os.UserHomeDir()`、相对路径相对于 `cfg.WorkspaceRoot` 解析，不存在时 log.Warn 并跳过
- [x] 9.3 构建 Discovery 实例（CacheDir = `filepath.Join(cfg.DataDir, "cache")`），对 `cfg.Skills.URLs` 中的每个 URL 调用 `Pull`，将返回的目录追加到 skillSearchPaths
- [x] 9.4 将 `DiscoverSkills` 调用改为传入 logger
- [x] 9.5 移除 `LoadDir` 回退逻辑

## 10. skills list CLI 对齐（D9）

- [x] 10.1 修改 `internal/cli/skills_cli.go`：构建与 wireEngine 相同的搜索路径（项目级 + 用户级 + config.skills.paths + skills_dir）
- [x] 10.2 调用 `skill.DiscoverSkills(paths, logger)` 替代 `skill.LoadDir`
- [x] 10.3 输出格式改为 `- name: description`，每行一个技能

## 11. 集成验证

- [x] 11.1 运行 `go build ./...` 确保编译通过
- [x] 11.2 运行 `go test ./internal/skill/...` 确保 skill 模块测试通过
- [x] 11.3 运行 `go test ./internal/tool/...` 确保工具模块测试通过
- [x] 11.4 运行 `go vet ./...` 检查代码质量
- [x] 11.5 手动验证：在 `.cursor/skills/test/SKILL.md` 创建测试技能，运行 `skills list` 确认发现
- [x] 11.6 验证外部技能禁用：设置 `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` 后运行 `skills list` 确认外部技能被过滤
