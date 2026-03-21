# Capability: cli-and-config (delta)

## ADDED Requirements

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
