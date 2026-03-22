# Capability: cli-and-config (delta)

## MODIFIED Requirements

### Requirement: 与上游配置一致

对于两种实现中均存在的设置，系统 MUST 使用与上游 OpenCode 相同的配置文件名、文件格式与顶层键路径。Go 扩展键也 MUST 位于顶层并文档化。新增键：`providers`（OpenAI/Anthropic 配置含 api_key/base_url/model）、`default_provider`、`default_model`、`max_tool_rounds`、`permissions`（per-tool ask/allow/deny）、`skills_dir`、`mcp_servers`（含 transport/command/url/args 字段）。

#### Scenario: 新增配置键可加载

- **WHEN** 配置文件包含 `providers.openai.api_key` 键
- **THEN** 系统 MUST 解析并用于初始化 OpenAI 提供商

#### Scenario: 共享工作区配置可加载

- **WHEN** 用户按兼容性引用放置上游 OpenCode 有效的配置文件
- **THEN** 本实现 MUST 按同优先级解析相同键

### Requirement: CLI 帮助与用法错误

系统 MUST 为根命令及每个子命令提供 `-h` / `--help`。对用法或 flag 校验错误以退出码 2 退出。新增以下子命令：`sessions list`、`tools list`、`skills list`、`project`（占位）、`repl`（交互式 agent 循环）。

#### Scenario: 非法 flag 退出码为 2

- **WHEN** 用户传入未定义的 flag
- **THEN** 进程 MUST 向标准错误输出诊断信息并以退出码 2 退出

#### Scenario: 新子命令可用

- **WHEN** 用户运行 `opencode-go sessions list`
- **THEN** 系统 MUST 列出本地 SQLite 中的会话

## ADDED Requirements

### Requirement: REPL agent 循环

`repl` 子命令 MUST 启动交互式 agent 循环：从 stdin 读取用户输入，通过 `wireEngine` 构建 Engine，执行 `CompleteTurn`（ReAct 循环），输出助手最终回复。当工具权限为 `ask` 时 MUST 在终端提示用户确认。

#### Scenario: 基本对话

- **WHEN** 用户在 REPL 中输入文本并回车
- **THEN** 系统 MUST 将输入传给 Engine 并在终端输出助手回复

#### Scenario: 工具确认交互

- **WHEN** 模型请求调用权限为 `ask` 的工具
- **THEN** REPL MUST 在终端显示工具名和参数，等待用户输入 y/n

### Requirement: MCP 服务端配置

`mcp_servers` 数组中的每个条目 MUST 支持 `name`、`transport`（stdio/sse/streamable_http）、`command`（stdio 用）、`args`（stdio 用）、`url`（SSE/Streamable HTTP 用）字段。

#### Scenario: MCP 配置解析

- **WHEN** 配置文件包含含 `transport: "sse"` 与 `url` 的 MCP 服务端条目
- **THEN** 系统 MUST 初始化 SSE 传输连接该服务端
