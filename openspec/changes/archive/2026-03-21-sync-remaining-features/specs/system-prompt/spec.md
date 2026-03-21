## ADDED Requirements

### Requirement: 模型专用 Base Prompt

系统 MUST 提供 `internal/prompt/` 模块，根据 Provider 类型（OpenAI / Anthropic）返回对应的基础系统提示文本。当 Agent 定义了自定义 prompt 时，Agent prompt MUST 覆盖 provider base prompt。

#### Scenario: OpenAI 模型 base prompt

- **WHEN** 当前模型属于 OpenAI provider
- **THEN** 系统 MUST 使用 OpenAI 专用的 base prompt 文本

#### Scenario: Anthropic 模型 base prompt

- **WHEN** 当前模型属于 Anthropic provider
- **THEN** 系统 MUST 使用 Anthropic 专用的 base prompt 文本

#### Scenario: Agent prompt 覆盖

- **WHEN** 当前 Agent 定义了自定义 prompt
- **THEN** 系统 MUST 使用 Agent prompt 替代 provider base prompt

### Requirement: 环境信息注入

系统 MUST 在系统提示中注入运行时环境信息，包括：工作目录（workspace root）、操作系统平台、当前日期、git 分支名称和状态。

#### Scenario: 环境信息包含工作目录

- **WHEN** 构建系统提示
- **THEN** 提示 MUST 包含当前 workspace root 路径

#### Scenario: 环境信息包含 git 状态

- **WHEN** 工作区是 git 仓库
- **THEN** 提示 MUST 包含当前分支名称

#### Scenario: 非 git 目录

- **WHEN** 工作区不是 git 仓库
- **THEN** 提示 MUST 跳过 git 相关信息，MUST NOT 报错

### Requirement: InstructionPrompt 加载

系统 MUST 支持从多个来源加载用户指令并注入系统提示：（1）向上查找 AGENTS.md / CLAUDE.md / CONTEXT.md 文件；（2）config.instructions 中配置的文件路径；（3）config.instructions 中配置的 URL（HTTP GET）。

#### Scenario: AGENTS.md 自动发现

- **WHEN** 工作区或其祖先目录包含 AGENTS.md 文件
- **THEN** 系统 MUST 将其内容注入系统提示

#### Scenario: config.instructions 文件路径

- **WHEN** config.instructions 包含本地文件路径
- **THEN** 系统 MUST 读取文件内容并注入系统提示

#### Scenario: config.instructions URL

- **WHEN** config.instructions 包含以 http:// 或 https:// 开头的 URL
- **THEN** 系统 MUST 通过 HTTP GET 获取内容并注入系统提示

#### Scenario: 文件不存在时跳过

- **WHEN** 配置的文件路径不存在
- **THEN** 系统 MUST 跳过该路径，MUST NOT 报错

### Requirement: 技能列表注入

系统 MUST 在系统提示中仅注入技能的名称和描述摘要（而非完整正文）。技能完整内容 MUST 通过 skill 工具按需加载。

#### Scenario: 技能摘要注入

- **WHEN** 有 3 个可用技能
- **THEN** 系统提示 MUST 包含 3 个技能的名称和描述，MUST NOT 包含技能完整正文

### Requirement: 系统提示组装顺序

系统 MUST 按以下固定顺序组装系统提示：（1）Model/Agent base prompt → （2）环境信息 → （3）InstructionPrompt → （4）技能列表摘要。

#### Scenario: 组装顺序正确

- **WHEN** 所有组件均可用
- **THEN** 最终系统提示 MUST 按 base prompt、环境信息、InstructionPrompt、技能列表的顺序排列
