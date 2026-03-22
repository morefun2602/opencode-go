# opencode-go

OpenCode 的 Go 实现，面向企业级 Agent 框架场景（Skills、MCP、ReAct、SubAgent 等）。**配置文件与上游 [OpenCode](https://github.com/anomalyco/opencode) 同格式**，可直接共用 `opencode.json`，降低迁移成本。

## 环境要求

- **Go**：见根目录 [`go.mod`](go.mod)（当前为 **Go 1.25.1**）。若使用更高版本或不同 `GOROOT`，请与 `go version` 一致，避免 `compile: version does not match go tool version`（可用 [gvm](https://github.com/moovweb/gvm) 或官方安装包统一）。

## 构建与测试

推荐使用 **Makefile**：

```bash
make help          # 默认目标：列出所有命令
make check         # 本地 CI：vet + test（同 make ci）
make build         # 输出 bin/opencode
make run           # go run ./cmd/opencode
```

也可直接使用 `go` 或脚本：

```bash
go build -o bin/opencode ./cmd/opencode
go test ./...
go vet ./...
./scripts/ci.sh
```

`make build` 可通过变量覆盖输出路径，例如：`make build OUT=./opencode`。

## 配置

- **格式**：与上游 OpenCode 一致的顶层键（如 `provider`、`model`、`small_model`、`mcp`、`agent`、`permission`、`server` 等），**不再使用** `x_opencode_go`。
- **合并顺序**（后者覆盖前者）：`~/.config/opencode/opencode.json`（或 `.jsonc`）→ `.opencode/opencode.jsonc` / `.json` → 当前目录 `opencode.json` → `OPENCODE_CONFIG` / `--config` 指定文件 → 远程配置（若设置）→ **环境变量** → 命令行标志。
- **常见环境变量**：`OPENAI_API_KEY`、`ANTHROPIC_API_KEY`、`OPENCODE_API_KEY`（Zen / opencode 提供商）等。

示例（节选）：

```json
{
  "model": "opencode/gpt-5-nano",
  "provider": {
    "opencode": {
      "options": { "apiKey": "public", "baseURL": "https://opencode.ai/zen/v1" }
    }
  }
}
```

更完整的键说明见 [docs/CONFIG.md](docs/CONFIG.md)（若与实现不一致，以 [`internal/config/config.go`](internal/config/config.go) 为准）。

## 运行

**HTTP 服务：**

```bash
go run ./cmd/opencode serve --listen 127.0.0.1:8080
# 或: ./bin/opencode serve --listen 127.0.0.1:8080
```

**TUI：**

```bash
go run ./cmd/opencode tui
```

## 文档

| 文档 | 说明 |
|------|------|
| [docs/CONFIG.md](docs/CONFIG.md) | 配置文件与环境变量 |
| [docs/HTTP.md](docs/HTTP.md) | HTTP API（含流式 SSE） |
| [docs/RELEASE.md](docs/RELEASE.md) | 发版与 **BREAKING** 说明模板 |

## 交叉编译（简要）

```bash
GOOS=linux GOARCH=amd64 go build -o opencode-linux-amd64 ./cmd/opencode
GOOS=darwin GOARCH=arm64 go build -o opencode-darwin-arm64 ./cmd/opencode
```

## OpenSpec

变更说明与规格见 `openspec/` 目录（含历史归档与主规格）。
