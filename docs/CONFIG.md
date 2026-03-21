# 配置说明

## 与上游对齐

- **配置文件名**：默认在进程工作目录查找 `opencode.json`；或通过 `OPENCODE_CONFIG` / `--config` 指定路径。
- **格式**：JSON（与上游一致时勿改为 YAML，除非上游亦支持）。
- **兼容性引用**：字段 `upstream_compat_ref`（环境变量 `OPENCODE_UPSTREAM_COMPAT_REF`）用于标注所跟踪的上游版本或提交范围。

## 顶层键（示例）

| 键 | 说明 |
|----|------|
| `upstream_compat_ref` | 上游兼容性引用 |
| `server.listen` | 监听地址，如 `127.0.0.1:8080` |
| `server.auth_token` | Bearer 鉴权（非 loopback 监听时必填） |
| `workspace.id` | 工作区 ID，用于数据隔离 |
| `x_opencode_go.data_dir` | 本地数据目录（SQLite 等） |
| `x_opencode_go.llm_timeout` | 如 `"60s"` |
| `x_opencode_go.workspace_root` | 工具读写的文件系统根（默认 `.`） |
| `x_opencode_go.require_write_confirm` | `write` 工具是否要求 `confirm: true` |
| `x_opencode_go.bash_timeout_sec` | `bash` 工具超时（秒） |
| `x_opencode_go.max_output_bytes` | 工具输出截断上限 |
| `x_opencode_go.compaction_turns` | 超过则记录压缩阈值日志 |
| `x_opencode_go.llm_max_retries` | 对可重试 LLM 错误的最大重试次数（不含首次） |
| `x_opencode_go.structured_output_schema` | 非空时校验助手输出为 JSON |
| `x_opencode_go.mcp_servers` | MCP 服务端列表（见下文） |
| `x_opencode_go.mcp_tool_prefix` | MCP 工具名前缀（默认 `mcp.`） |
| `x_opencode_go.default_provider` | 默认 LLM 提供商（`openai` / `anthropic` / `stub`） |
| `x_opencode_go.default_model` | 覆盖提供商默认模型 |
| `x_opencode_go.max_tool_rounds` | ReAct 循环中最大工具调用轮次（默认 25） |
| `x_opencode_go.skills_dir` | 技能目录路径（默认 `<data_dir>/skills`） |
| `x_opencode_go.providers` | 提供商配置（见下文） |
| `x_opencode_go.permissions` | 工具权限（见下文） |

## providers 配置

```json
{
  "x_opencode_go": {
    "providers": {
      "openai": {
        "api_key": "sk-...",
        "base_url": "https://api.openai.com/v1",
        "model": "gpt-4o"
      },
      "anthropic": {
        "api_key": "sk-ant-...",
        "model": "claude-sonnet-4-20250514"
      }
    },
    "default_provider": "anthropic"
  }
}
```

## permissions 配置

每个工具名映射到 `"allow"` / `"ask"` / `"deny"`。默认 `allow`。

```json
{
  "x_opencode_go": {
    "permissions": {
      "bash": "ask",
      "write": "ask"
    }
  }
}
```

`ask` 模式下 REPL 会在执行前提示用户确认；HTTP 模式下降级为 `allow`。

## mcp_servers 配置

```json
{
  "x_opencode_go": {
    "mcp_servers": [
      { "name": "my-mcp", "command": "npx", "args": ["-y", "@some/mcp-server"], "transport": "stdio" },
      { "name": "remote", "url": "https://mcp.example.com/rpc", "transport": "streamable_http" },
      { "name": "legacy", "url": "https://mcp.example.com/sse", "transport": "sse" }
    ]
  }
}
```

支持三种传输：`stdio`、`sse`、`streamable_http`。省略 `transport` 时按 `command` 存在推断 `stdio`，否则推断 `streamable_http`。

## 环境变量（覆盖配置文件）

| 变量 | 对应配置 |
|------|----------|
| `OPENCODE_CONFIG` | 配置文件路径 |
| `OPENCODE_UPSTREAM_COMPAT_REF` | `upstream_compat_ref` |
| `OPENCODE_SERVER_LISTEN` | `server.listen` |
| `OPENCODE_AUTH_TOKEN` | `server.auth_token` |
| `OPENCODE_WORKSPACE_ID` | `workspace.id` |
| `OPENCODE_DATA_DIR` | `x_opencode_go.data_dir` |
| `OPENCODE_LLM_TIMEOUT` | `x_opencode_go.llm_timeout` |
| `OPENAI_API_KEY` | `x_opencode_go.providers.openai.api_key`（仅在配置文件未设置时） |
| `ANTHROPIC_API_KEY` | `x_opencode_go.providers.anthropic.api_key`（仅在配置文件未设置时） |

命令行 `serve` 的 `--listen`、`--token`、`--data-dir`、`--workspace` 在环境变量之后再次覆盖。
