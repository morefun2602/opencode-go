# Tool Parity Checklist (opencode-go vs opencode)

该清单用于持续跟踪 `opencode-go` 在 Tool 子系统与上游 `opencode` 的能力对齐进度。

## Legend

- `DONE`: 已实现并有测试覆盖
- `PARTIAL`: 已有基础能力，但关键链路/边界未闭环
- `TODO`: 尚未实现

## Core Tool Runtime

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| Registry 统一执行入口 | DONE | 保持 | `internal/tools/registry.go` |
| Tool 参数统一 schema 校验 | TODO | DONE | `internal/tools/registry.go` |
| 统一错误格式（tool + validation） | PARTIAL | DONE | `internal/tools/registry.go`, `internal/runtime/engine.go` |
| 工具输出统一截断 | DONE | 保持 | `internal/tools/registry.go`, `internal/truncate/` |
| Tool 调用审计（allow/ask/deny） | PARTIAL | DONE | `internal/runtime/engine.go`, `internal/policy/policy.go` |

## Builtin Tools

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| `todowrite` | DONE | 保持 | `internal/tool/todowrite.go` |
| `todoread` | TODO | DONE | `internal/tool/todoread.go`, `internal/tool/builtin.go` |
| `plan_enter/plan_exit` | DONE | 保持 | `internal/tool/plan.go`, `internal/runtime/engine.go` |
| `apply_patch`/`edit`/`write` 路由策略 | TODO | DONE | `internal/runtime/engine.go` |
| 自定义工具目录加载（`.opencode/tool|tools`） | TODO | DONE | `internal/tool/custom_loader.go`, `internal/cli/wire.go` |

## MCP Integration

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| MCP tools: list + call | DONE | 保持 | `internal/mcp/client.go` |
| MCP resources: list + read | TODO | DONE | `internal/mcp/client.go` |
| OAuth token 获取/刷新模块 | PARTIAL | DONE | `internal/mcp/oauth.go` |
| OAuth 执行链路接入（wire + remote transport） | TODO | DONE | `internal/cli/wire.go`, `internal/mcp/client.go` |
| 401 自动恢复重试 | TODO | DONE | `internal/mcp/client.go` |

## Security & Policy

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| workspace 内路径约束 | DONE | 保持 | `internal/tool/paths.go` |
| policy allow/ask/deny | DONE | 保持 | `internal/policy/policy.go` |
| 按参数模式权限判定（path/cmd） | PARTIAL | DONE | `internal/runtime/engine.go`, `internal/policy/policy.go` |
| external_directory 语义权限点 | TODO | DONE | `internal/runtime/engine.go`, `internal/tool/bash.go` |

## Tests & Docs

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| Tool runtime 单测 | PARTIAL | DONE | `internal/tool/*_test.go`, `internal/runtime/*_test.go` |
| MCP OAuth + resources 测试 | TODO | DONE | `internal/mcp/*_test.go` |
| Parity 文档与 OpenSpec 对齐 | PARTIAL | DONE | `docs/PARITY.md`, `openspec/specs/*` |
