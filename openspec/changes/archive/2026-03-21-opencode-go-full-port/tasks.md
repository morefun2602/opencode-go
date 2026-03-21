# 任务清单：OpenCode Go 完整实现

## 1. 仓库与模块骨架

- [x] 1.1 确认根目录 `go.mod` 模块路径为 `github.com/morefun2602/opencode-go`，并建立 `cmd/opencode` 与 `internal/` 下各包目录（config、cli、runtime、llm、tools、store、server）的空包与最小 `main` 入口
- [x] 1.2 在 `README` 或 `docs/` 中说明标准构建命令（`go build ./...`、`go test ./...`）与最低 Go 版本，满足 `go-codebase-layout` 规范

## 2. 配置与 CLI 基线

- [x] 2.1 实现配置模型与加载：按 `design.md` 优先级（默认值 → 文件 → 环境变量 → flag），并落实 `cli-and-config` 中与上游一致的**文件名、格式与键路径**及**兼容性引用版本**说明
- [x] 2.2 实现根命令与子命令的 `-h/--help`，用法/校验错误退出码 2，以及文档化的运行退出码（含 SIGINT→130，若平台适用）
- [x] 2.3 实现用于启动 HTTP 服务的子命令或 flag 组合，绑定与鉴权参数走同一套配置优先级

## 3. 持久化层

- [x] 3.1 定义 `store` 接口并实现基于 SQLite 的持久化（优先 `modernc.org/sqlite` 或设计选定驱动），满足 `persistence` 中的会话/消息与工作区隔离
- [x] 3.2 实现 schema 版本、启动时向前迁移；磁盘 schema 新于二进制支持时拒绝启动并给出明确错误
- [x] 3.3 对关联写操作使用事务，保证 turn 级写入原子性（用户消息与助手消息不半提交）

## 4. LLM 与工具

- [x] 4.1 实现 `internal/llm` 提供商抽象与注册机制，出站请求使用 `net/http`、可配置超时与 `context` 取消，满足 `llm-and-tools`
- [x] 4.2 在支持流式时实现分片消费与向消费者增量转发（CLI/HTTP），避免无谓全量缓冲
- [x] 4.3 实现工具注册、参数 schema 校验、失败时结构化错误回传与日志（含关联 ID），不静默忽略工具/MCP 失败

## 5. 智能体运行时

- [x] 5.1 实现会话生命周期（创建/选择/关闭、稳定会话 ID、进程内唯一），满足 `agent-runtime`
- [x] 5.2 将长时操作接入 `context` 取消，取消后不再输出该 turn 的后续助手内容并向上返回取消
- [x] 5.3 保证同 turn 因果顺序持久化（先用户后助手），并为主要生命周期与工具事件输出结构化日志（稳定字段名）

## 6. HTTP API

- [x] 6.1 实现 `internal/server`：`http.Server`、版本前缀路由（如 `/v1`）、JSON 错误体（`code`/`message`）、默认 loopback 与非 loopback 时强制鉴权的启动校验
- [x] 6.2 实现至少一种鉴权（Bearer 或共享密钥 via 头/查询参数），配置与 `cli-and-config` 优先级一致
- [x] 6.3 实现优雅关闭（SIGTERM 等）、监听关闭与宽限期内排空进行中的请求
- [x] 6.4 若提供流式 HTTP 输出，文档化内容类型、分块/SSE 格式与结束条件，并提供非流式完整响应的替代路径

## 7. 集成与质量

- [x] 7.1 将 HTTP handler 通过窄接口依赖 runtime/store，完成与 `agent-runtime` / `llm-and-tools` / `persistence` 行为对齐的端到端路径（至少一条可重复执行的集成场景）
- [x] 7.2 添加 `go vet`、`go test ./...` 至 CI（或等价脚本），并覆盖关键规范场景（配置优先级、鉴权失败 401、迁移拒绝等）的自动化测试

## 8. 发布与文档

- [x] 8.1 编写或更新发行说明模板：标注 **BREAKING**（配置键、API 前缀、DB schema）；说明默认绑定 loopback 与扩大暴露时的安全要求
- [x] 8.2 列出多平台二进制或容器构建与版本策略（与 `proposal`/`design` 一致），便于后续发布流水线接入
