# Proposal: CLI Commands Parity

## Why

Go 版本仅有 7 个 CLI 命令（serve, sessions list, tools list, skills list, project, repl, tui），而 TS 版本有 23 个。缺失的命令使 Go 版本在日常使用中功能不完整。

## What Changes

补齐 TS 版本中 Go 尚未实现的 CLI 命令，按优先级分批实现：

**P0 — 基础必备：**
- `version` — 版本号显示
- `run` — 无头模式执行（传入消息，获取回复）
- `session list` 增强 + `session delete` — 会话管理完整化
- `debug config` — 显示解析后的配置
- `debug paths` — 显示全局路径

**P1 — 核心能力：**
- `models` — 列出可用模型
- `providers list/login/logout` — AI 提供商凭据管理
- `agent list` — 列出已注册 agents
- `mcp list/add` — MCP 服务管理
- `export` / `import` — 会话导入导出

**P2 — 增强功能：**
- `stats` — Token 用量统计
- `db path/query` — 数据库工具
- `debug agent` — 显示 agent 配置详情
- TUI 默认命令增强（--model, --continue, --session 标志）

## Capabilities (delta specs)

- `cli-commands/spec.md` — MODIFIED — 新增命令定义

## Impact

- `internal/cli/root.go` — 注册新命令
- `internal/cli/version.go` — 新建
- `internal/cli/run.go` — 新建
- `internal/cli/debug.go` — 新建
- `internal/cli/models.go` — 新建
- `internal/cli/providers.go` — 新建
- `internal/cli/agent_cmd.go` — 新建
- `internal/cli/mcp_cmd.go` — 新建
- `internal/cli/export.go` — 新建
- `internal/cli/import.go` — 新建
- `internal/cli/stats.go` — 新建
- `internal/cli/db.go` — 新建
- `internal/cli/sessions.go` — 修改（增加 delete）
- `internal/cli/tui.go` — 修改（增加标志）
