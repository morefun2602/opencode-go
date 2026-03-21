# 内部包依赖约定

横切能力包（`internal/mcp`、`internal/tool`、`internal/skill`、`internal/plugin`、`internal/acp`、`internal/runtime`、`internal/store`、`internal/llm` 等）**不得** import：

- `github.com/morefun2602/opencode-go/cmd/...`
- 任何直接绑定 HTTP 路由实现的 handler 包（例如为保持单向依赖，业务包不依赖 `internal/server` 的具体类型；组装发生在 `cmd` 或 `internal/cli`）

调用链目标形态：`cli` / `server` → `runtime` → `llm`、**tool 路由**、MCP 客户端门面 → `store`。

## 可选检查

CI 已运行 `go vet ./...`。本地可执行：

```bash
./scripts/check.sh
```
