# Delta: cli-and-config

本文件为相对 `openspec/specs/cli-and-config/spec.md` 的增量需求。

## ADDED Requirements

### Requirement: 子命令集扩展

系统 MUST 为与上游对齐的核心工作流提供已文档化的子命令（除 `serve` 外，至少规划 **会话/项目/工具/技能** 相关入口之一的具体命名在实现中固定）；未知子命令 MUST 以退出码 2 报告。

#### Scenario: 未知子命令

- **WHEN** 用户输入未定义的子命令
- **THEN** 进程 MUST 打印用法或错误且 MUST 以退出码 2 退出

### Requirement: 交互式模式（TUI/REPL）

系统 MUST 提供非 HTTP 的交互式运行模式（终端 UI 或 REPL 之一）以缩小与上游 CLI 体验差距；该模式 MUST 复用与 `serve` 相同的配置加载规则。

#### Scenario: 交互模式启动

- **WHEN** 用户调用文档化的交互子命令且配置有效
- **THEN** 系统 MUST 进入交互循环且 MUST 响应中断信号按文档退出

### Requirement: Go 扩展键命名

新增仅 Go 实现使用的配置键 MUST 位于顶层并在文档中列出；MUST NOT 占用上游已定义键名。

#### Scenario: 新键不冲突

- **WHEN** 配置包含新增 Go 扩展字段
- **THEN** 解析 MUST 成功且 MUST 不影响上游键语义
