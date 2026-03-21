## 1. Provider Registry 与 OpenAI-Compatible 提供商

- [x] 1.1 在 `internal/llm` 新增 `registry.go`，实现 `ProviderFactory` 注册表与 `Register` / `Get` 函数
- [x] 1.2 为 Provider 接口新增 `Models() []string` 方法，更新 openai / anthropic 实现
- [x] 1.3 新增 `openai_compatible.go`，复用 openai-go SDK 并支持 `base_url` 覆盖
- [x] 1.4 更新 `internal/config` 新增 `providers` 配置结构（name / type / base_url / api_key / models）
- [x] 1.5 更新 `internal/cli/wire.go`，启动时遍历 providers 配置并注册到 registry
- [x] 1.6 编写 provider registry 和 openai-compatible 的单元测试

## 2. 新增工具 — todowrite

- [x] 2.1 在 `internal/tools` 新增 `todowrite.go`，实现 todo 增删改查存储
- [x] 2.2 在 `internal/store` 新增 todo 存储（会话级 key-value 或独立表）
- [x] 2.3 更新 Engine 系统提示构建，注入当前 todo 列表
- [x] 2.4 在 `internal/tool/builtin.go` 注册 todowrite 工具（标签：write）
- [x] 2.5 编写 todowrite 工具单元测试

## 3. 新增工具 — apply_patch

- [x] 3.1 添加 `sourcegraph/go-diff` 依赖
- [x] 3.2 在 `internal/tools` 新增 `apply_patch.go`，解析 unified diff 并应用到文件
- [x] 3.3 实现多文件补丁的原子性（失败时回滚已写入文件）
- [x] 3.4 实现 `ResolveUnder` 路径限制
- [x] 3.5 在 `internal/tool/builtin.go` 注册 apply_patch 工具（标签：write）
- [x] 3.6 编写 apply_patch 工具单元测试（单文件、多文件、越界路径）

## 4. 新增工具 — websearch

- [x] 4.1 在 `internal/tools` 新增 `websearch.go`，调用可配置的搜索 API
- [x] 4.2 更新 `internal/config` 新增 `websearch_url` 配置项
- [x] 4.3 在 `internal/tool/builtin.go` 注册 websearch 工具（标签：read）
- [x] 4.4 编写 websearch 工具单元测试

## 5. 新增工具 — question

- [x] 5.1 在 `internal/tools` 新增 `question.go`，实现向用户提问并阻塞等待回复的逻辑
- [x] 5.2 定义 question 事件类型与等待/回复的 channel 管理
- [x] 5.3 在 `internal/tool/builtin.go` 注册 question 工具（标签：interact）
- [x] 5.4 编写 question 工具单元测试

## 6. Agent 模式系统

- [x] 6.1 在 `internal/runtime` 新增 `mode.go`，定义 Mode 结构体（名称、允许标签集合）和内置模式（build / plan / explore）
- [x] 6.2 为 `tool.Definition` 新增 `Tags []string` 字段
- [x] 6.3 更新所有内置工具注册，声明标签（read / write / execute / interact）
- [x] 6.4 更新 Engine 的 `collectTools` 方法，按当前模式的标签过滤工具
- [x] 6.5 更新 `internal/config` 新增 `agents` 配置项，支持自定义模式定义
- [x] 6.6 编写模式过滤单元测试

## 7. 会话管理增强

- [x] 7.1 数据库迁移 v3 → v4：session 表新增 title / archived / parent_id / parent_message_seq 列
- [x] 7.2 更新 `internal/store` 的 Session 模型与 CRUD 方法
- [x] 7.3 实现 `Fork` 方法（事务内批量复制消息）
- [x] 7.4 实现 `Revert` 方法（事务内删除指定 seq 之后的消息）
- [x] 7.5 实现 `SetTitle` / `SetArchived` 方法
- [x] 7.6 实现 `Usage` 统计方法（聚合 token 计数）
- [x] 7.7 实现自动标题生成（首轮对话后异步调用 LLM）
- [x] 7.8 编写会话管理单元测试（fork / revert / title / usage）

## 8. 事件总线

- [x] 8.1 新增 `internal/bus` 包，实现 channel-based pub/sub
- [x] 8.2 定义事件类型（session.created / session.updated / message.created / tool.start / tool.end / permission.ask / question.ask）
- [x] 8.3 在 Engine / Store 的关键操作中发射事件
- [x] 8.4 在 `internal/server` 新增 `GET /v1/events` SSE 端点，桥接事件总线到 HTTP SSE
- [x] 8.5 编写事件总线单元测试

## 9. HTTP API 扩展

- [x] 9.1 新增 `GET /v1/providers` 和 `GET /v1/providers/{id}/models` 端点
- [x] 9.2 新增 `POST /v1/sessions/{id}/fork` 端点
- [x] 9.3 新增 `POST /v1/sessions/{id}/revert` 端点
- [x] 9.4 新增 `PATCH /v1/sessions/{id}` 端点
- [x] 9.5 新增 `GET /v1/sessions/{id}/usage` 端点
- [x] 9.6 新增 `GET /v1/config` 端点
- [x] 9.7 新增 `POST /v1/permission/reply` 和 `POST /v1/question/reply` 端点
- [x] 9.8 编写新 HTTP 端点的集成测试

## 10. 权限 Pattern 增强

- [x] 10.1 更新 `internal/policy`，支持 `tool_name:pattern` 格式的 glob 规则
- [x] 10.2 实现 `once` / `always` / `reject` 回复语义及会话级缓存
- [x] 10.3 新增 `POST /v1/permission/reply` 的异步回复处理逻辑
- [x] 10.4 编写权限 pattern 匹配单元测试

## 11. 配置增强

- [x] 11.1 更新 `internal/config` 新增 `instructions []string` 字段
- [x] 11.2 实现远程配置拉取（HTTPS GET + JSON 解析 + 合并）
- [x] 11.3 更新 Engine 系统提示构建，注入 instructions 配置内容
- [x] 11.4 编写配置增强单元测试

## 12. TUI — Bubble Tea 应用

- [x] 12.1 添加 bubbletea / lipgloss / glamour 依赖
- [x] 12.2 新增 `internal/tui` 包，实现顶层 Model（Init / Update / View）
- [x] 12.3 实现对话视图组件（消息列表、Markdown 渲染、代码高亮）
- [x] 12.4 实现输入区域组件（多行输入、Enter 发送、Shift+Enter 换行）
- [x] 12.5 实现会话侧边栏组件（列表、搜索、创建、选择）
- [x] 12.6 实现工具确认对话框组件
- [x] 12.7 实现模式切换交互与状态栏
- [x] 12.8 实现 dark / light 主题支持
- [x] 12.9 更新 `internal/cli`，新增 `tui` 子命令作为默认入口
- [x] 12.10 编写 TUI 组件的基础测试

## 13. 集成验证

- [x] 13.1 全量 `go build ./...` 编译通过
- [x] 13.2 全量 `go test ./...` 通过
- [x] 13.3 手动验证 TUI 启动、对话、模式切换流程
- [x] 13.4 手动验证 HTTP API 新端点（providers / fork / revert / events）
