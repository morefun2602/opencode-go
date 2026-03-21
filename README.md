# opencode-go

OpenCode 的 Go 实现（模块：`github.com/morefun2602/opencode-go`）。

## 构建与测试

最低 Go 版本见 `go.mod`（当前为 **Go 1.26**，并声明 `toolchain go1.26.1` 以便自动拉取一致工具链）。

```bash
go build -o opencode ./cmd/opencode
go test ./...
go vet ./...
```

或使用仓库内脚本：

```bash
./scripts/ci.sh
```

### 本地 `compile: version does not match go tool version`

若出现 `GOROOT` 中标准库版本与 `go` 命令版本不一致，请统一安装（例如使用 [gvm](https://github.com/moovweb/gvm) / 官方包），或依赖 CI 中的 `actions/setup-go`。

## 运行 HTTP 服务

```bash
go run ./cmd/opencode serve --listen 127.0.0.1:8080
```

配置优先级：**默认值 → `opencode.json`（或 `OPENCODE_CONFIG`）→ 环境变量 → 命令行参数**。键名与文件名需与上游 OpenCode 对齐，见 [docs/CONFIG.md](docs/CONFIG.md)。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/CONFIG.md](docs/CONFIG.md) | 配置文件与环境变量 |
| [docs/HTTP.md](docs/HTTP.md) | HTTP API（含流式 SSE） |
| [docs/RELEASE.md](docs/RELEASE.md) | 发版与 **BREAKING** 说明模板 |

## 多平台二进制（简要）

```bash
GOOS=linux GOARCH=amd64 go build -o opencode-linux-amd64 ./cmd/opencode
GOOS=darwin GOARCH=arm64 go build -o opencode-darwin-arm64 ./cmd/opencode
```

版本与 tag 策略与 CI 发布流水线可按团队规范在后续接入。

## OpenSpec

变更 `openspec/changes/opencode-go-full-port/` 含提案、设计、规格与任务清单。
