## ADDED Requirements

### Requirement: JSONC 配置格式支持

系统 MUST 支持 JSONC 格式的配置文件（允许 `//` 和 `/* */` 注释及尾逗号）。解析时 MUST 先去除注释和尾逗号后再按标准 JSON 解析。

#### Scenario: 带注释的配置文件

- **WHEN** 配置文件包含 `// comment` 行注释
- **THEN** 系统 MUST 正确解析配置内容，忽略注释

#### Scenario: 带尾逗号的配置文件

- **WHEN** 配置文件中数组或对象最后一个元素后有逗号
- **THEN** 系统 MUST 正确解析配置内容

### Requirement: 默认模型配置

系统 MUST 支持 `model` 配置项（字符串，`"provider/model"` 格式），指定默认使用的模型。未配置时 MUST 回退到第一个可用 Provider 的第一个模型。

#### Scenario: 指定默认模型

- **WHEN** 配置 `model: "anthropic/claude-sonnet-4-20250514"`
- **THEN** Engine MUST 使用该模型作为普通会话的默认模型

### Requirement: 小模型配置

系统 MUST 支持 `small_model` 配置项（字符串，`"provider/model"` 格式），指定用于 compaction/title/summary 等内部任务的小模型。

#### Scenario: 指定小模型

- **WHEN** 配置 `small_model: "openai/gpt-4o-mini"`
- **THEN** compaction/title/summary 任务 MUST 使用该模型

### Requirement: InstructionPrompt 文件配置

系统 MUST 支持 `instructions` 配置项（字符串数组），每项可为文件路径或 URL。文件路径 MUST 相对于工作区根目录解析。URL MUST 通过 HTTP GET 加载。

#### Scenario: 文件路径指令

- **WHEN** instructions 包含 `"docs/prompt.md"`
- **THEN** 系统 MUST 读取工作区下 `docs/prompt.md` 内容注入系统提示

#### Scenario: URL 指令

- **WHEN** instructions 包含 `"https://example.com/prompt.txt"`
- **THEN** 系统 MUST 通过 HTTP GET 获取内容注入系统提示

### Requirement: Compaction 配置

系统 MUST 支持 `compaction` 配置项，包含以下子字段：
- `auto`（bool，默认 true）：是否自动触发 compaction
- `reserved`（int，默认 20000）：预留 token 数
- `prune`（bool，默认 true）：是否在 compaction 前裁剪旧 tool 输出

#### Scenario: 关闭自动 compaction

- **WHEN** 配置 `compaction: {auto: false}`
- **THEN** Engine MUST NOT 在溢出时自动触发 compaction

#### Scenario: 自定义 reserved

- **WHEN** 配置 `compaction: {reserved: 30000}`
- **THEN** IsOverflow MUST 使用 30000 作为保留量

### Requirement: LSP 服务器配置

系统 MUST 支持 `lsp` 配置项，包含 `servers` 数组，每项包含 `language`（语言标识）、`command`（启动命令）、`args`（参数数组）字段。

#### Scenario: 配置 Go LSP

- **WHEN** 配置 `lsp: {servers: [{language: "go", command: "gopls", args: []}]}`
- **THEN** 系统 MUST 使用 gopls 作为 Go 文件的语言服务器

## MODIFIED Requirements

### Requirement: 与上游配置一致

对于两种实现中均存在的设置，系统 MUST 使用与上游 OpenCode 相同的配置文件名、文件格式与顶层键路径。Go 扩展键也 MUST 位于顶层并文档化。新增键：`providers`（OpenAI/Anthropic 配置含 api_key/base_url/model）、`default_provider`、`default_model`、`max_tool_rounds`、`permissions`（per-tool ask/allow/deny）、`skills_dir`、`mcp_servers`（含 transport/command/url/args 字段）、`model`（默认模型）、`small_model`（小模型）、`instructions`（指令文件/URL 数组）、`compaction`（auto/reserved/prune）、`lsp`（语言服务器配置）。配置文件 MUST 支持 JSONC 格式。

#### Scenario: 新增配置键可加载

- **WHEN** 配置文件包含 `providers.openai.api_key` 键
- **THEN** 系统 MUST 解析并用于初始化 OpenAI 提供商

#### Scenario: JSONC 格式可加载

- **WHEN** 配置文件为 JSONC 格式（含注释和尾逗号）
- **THEN** 系统 MUST 正确解析

#### Scenario: 共享工作区配置可加载

- **WHEN** 用户按兼容性引用放置上游 OpenCode 有效的配置文件
- **THEN** 本实现 MUST 按同优先级解析相同键
