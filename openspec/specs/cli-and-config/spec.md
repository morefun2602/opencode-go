# cli-and-config Specification

## Purpose

定义 CLI 入口、子命令、配置加载与优先级规则。

## Requirements

### Requirement: 与上游配置一致

对于两种实现中均存在的设置，系统 MUST 使用与上游 OpenCode 相同的配置文件名、文件格式与顶层键路径。仅 Go 实现使用的设置 MUST 在 `x_opencode_go` 命名空间下。新增以下 `x_opencode_go` 键：`providers`（OpenAI/Anthropic 配置含 api_key/base_url/model）、`default_provider`、`default_model`、`max_tool_rounds`、`permissions`（per-tool ask/allow/deny）、`skills_dir`、`mcp_servers`（含 transport/command/url/args 字段）。

#### Scenario: 新增配置键可加载

- **WHEN** 配置文件包含 `x_opencode_go.providers.openai.api_key` 键
- **THEN** 系统 MUST 解析并用于初始化 OpenAI 提供商

#### Scenario: 共享工作区配置可加载

- **WHEN** 用户按兼容性引用放置上游 OpenCode 有效的配置文件
- **THEN** 本实现 MUST 按同优先级解析相同键

### Requirement: 配置优先级

系统 MUST 按以下严格顺序应用配置（后者覆盖前者）：二进制内嵌默认值；已发现的配置文件中的值；环境变量；命令行 flag。具体环境变量名与 flag 名 MUST 写在实现文档中，且在未标记 BREAKING 的同一主版本内 MUST 保持稳定。

#### Scenario: 环境变量覆盖文件

- **WHEN** 同一逻辑项同时在配置文件与本规范允许的环境变量中设置
- **THEN** 进程 MUST 采用环境变量的值

### Requirement: CLI 帮助与用法错误

系统 MUST 为根命令及每个子命令提供 `-h` / `--help`。对用法或 flag 校验错误以退出码 2 退出。新增以下子命令：`sessions list`、`tools list`、`skills list`、`project`（占位）、`repl`（交互式 agent 循环）。

#### Scenario: 非法 flag 退出码为 2

- **WHEN** 用户传入未定义的 flag
- **THEN** 进程 MUST 向标准错误输出诊断信息并以退出码 2 退出

#### Scenario: 新子命令可用

- **WHEN** 用户运行 `opencode-go sessions list`
- **THEN** 系统 MUST 列出本地 SQLite 中的会话

### Requirement: 运行退出码

系统 MUST 将分类错误映射到稳定退出码：0 成功；2 用法/校验；1 内部或未分类失败；前台命令被用户中断（SIGINT）时为 130（在平台可区分时）。额外码 MUST 有文档说明。

#### Scenario: 运行中被中断

- **WHEN** 用户发送中断信号以取消正在执行的前台命令
- **THEN** 进程 MUST 以退出码 130 结束，除非平台无法区分中断与其他信号

### Requirement: 从 CLI 启动 HTTP 服务模式

系统 MUST 提供已文档化的子命令或 flag 组合，用于启动 `http-api` 规范中定义的 HTTP API 服务，绑定地址与鉴权等参数来自同一套配置优先级规则。

#### Scenario: 启动服务与配置一致

- **WHEN** 用户使用有效配置调用已文档化的 serve/start 类命令
- **THEN** HTTP 服务器 MUST 按 `http-api` 要求监听，并在启动时记录生效的绑定地址

### Requirement: 子命令集扩展

系统 MUST 为与上游对齐的核心工作流提供已文档化的子命令（除 `serve` 外，至少规划会话/项目/工具/技能相关入口之一）；未知子命令 MUST 以退出码 2 报告。

#### Scenario: 未知子命令

- **WHEN** 用户输入未定义的子命令
- **THEN** 进程 MUST 打印用法或错误且 MUST 以退出码 2 退出

### Requirement: 交互式模式（TUI/REPL）

系统 MUST 提供非 HTTP 的交互式运行模式（终端 UI 或 REPL 之一）以缩小与上游 CLI 体验差距；该模式 MUST 复用与 `serve` 相同的配置加载规则。

#### Scenario: 交互模式启动

- **WHEN** 用户调用文档化的交互子命令且配置有效
- **THEN** 系统 MUST 进入交互循环且 MUST 响应中断信号按文档退出

### Requirement: Go 扩展键命名空间

新增仅 Go 实现使用的配置键 MUST 位于 `x_opencode_go` 命名空间（或后续与上游约定的前缀），且 MUST 在文档中列出；MUST NOT 占用上游已定义键名。

#### Scenario: 新键不冲突

- **WHEN** 配置包含 `x_opencode_go` 下新字段
- **THEN** 解析 MUST 成功且 MUST 不影响上游键语义

### Requirement: REPL agent 循环

`repl` 子命令 MUST 启动交互式 agent 循环：从 stdin 读取用户输入，通过 `wireEngine` 构建 Engine，执行 `CompleteTurn`（ReAct 循环），输出助手最终回复。当工具权限为 `ask` 时 MUST 在终端提示用户确认。

#### Scenario: 基本对话

- **WHEN** 用户在 REPL 中输入文本并回车
- **THEN** 系统 MUST 将输入传给 Engine 并在终端输出助手回复

#### Scenario: 工具确认交互

- **WHEN** 模型请求调用权限为 `ask` 的工具
- **THEN** REPL MUST 在终端显示工具名和参数，等待用户输入 y/n

### Requirement: MCP 服务端配置

`x_opencode_go.mcp_servers` 数组中的每个条目 MUST 支持 `name`、`transport`（stdio/sse/streamable_http）、`command`（stdio 用）、`args`（stdio 用）、`url`（SSE/Streamable HTTP 用）字段。

#### Scenario: MCP 配置解析

- **WHEN** 配置文件包含含 `transport: "sse"` 与 `url` 的 MCP 服务端条目
- **THEN** 系统 MUST 初始化 SSE 传输连接该服务端

### Requirement: Agent 模式配置

系统 MUST 支持 `x_opencode_go.agents` 配置项，为自定义 Agent 模式定义名称、允许的工具列表、模型、温度等参数。

#### Scenario: 自定义 Agent 加载

- **WHEN** 配置包含 `agents: [{name: "review", tools: ["read","grep","glob"], model: "gpt-4o"}]`
- **THEN** 系统 MUST 注册名为 `review` 的 Agent 模式

### Requirement: 全局指令注入

系统 MUST 支持 `x_opencode_go.instructions` 配置项（字符串数组），其内容 MUST 在每个会话的系统提示头部注入。

#### Scenario: 指令注入

- **WHEN** 配置包含 `instructions: ["Always respond in English"]`
- **THEN** 每个会话的系统提示 MUST 以该指令开头

### Requirement: 远程配置

系统 MUST 支持从 `.well-known/opencode` URL 拉取远程配置并与本地配置合并。远程配置优先级 MUST 低于本地配置文件。

#### Scenario: 远程配置合并

- **WHEN** 配置了远程配置 URL 且该 URL 返回有效 JSON
- **THEN** 系统 MUST 将远程配置合并到默认值之上、本地文件之下

### Requirement: TUI 子命令

系统 MUST 新增 `tui` 子命令作为默认交互入口，启动基于 Bubble Tea 的终端 UI。

#### Scenario: 默认启动 TUI

- **WHEN** 用户运行 `opencode-go` 不带子命令
- **THEN** 系统 MUST 启动 TUI 模式
