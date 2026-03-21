## 1. ReAct 循环增强

- [x] 1.1 在 `internal/runtime/engine.go` 中实现 doom loop 检测：维护最近 3 次 tool_call 签名的滑动窗口，连续相同时通过 Permission.Ask 通知用户
- [x] 1.2 在 `internal/tools/` 新建 `compaction.go`，实现真正的上下文压缩：token 溢出检测、LLM 摘要生成、历史消息替换、近期消息保留
- [x] 1.3 在 `internal/runtime/engine.go` 中实现 ContextOverflow 错误恢复：捕获 Provider 的上下文溢出错误，触发 compaction，压缩后自动重试
- [x] 1.4 在 `internal/runtime/engine.go` 中实现 Permission/Question Rejected 处理：将拒绝信息作为 tool_result 回注，不终止循环
- [x] 1.5 为 doom loop、compaction、error recovery 编写单元测试

## 2. 增强重试逻辑

- [x] 2.1 在 `internal/tools/` 新建 `retry.go`，实现增强重试模块：retry-after 头解析、指数退避策略、错误类型分类（RateLimit/Timeout 可重试，Auth 不重试）
- [x] 2.2 重构 `internal/runtime/engine.go` 中现有的 `callWithRetry`，使用新的增强重试模块
- [x] 2.3 编写重试模块的单元测试

## 3. 新增内置工具 — multiedit

- [x] 3.1 在 `internal/tool/` 新建 `multiedit.go`，实现 multiedit 工具：接受文件路径和多组 `{old_string, new_string}` 替换对，原子性执行
- [x] 3.2 在 `internal/tool/builtin.go` 中注册 multiedit 工具，标签为 `["write"]`
- [x] 3.3 编写 multiedit 工具的单元测试（成功、old_string 不存在、old_string 不唯一）

## 4. 新增内置工具 — plan_enter / plan_exit

- [x] 4.1 在 `internal/tool/` 新建 `plan.go`，实现 plan_enter 工具：切换当前会话模式为 plan
- [x] 4.2 在 `plan.go` 中实现 plan_exit 工具：通过 Question.Ask 确认后切换回 build 模式
- [x] 4.3 在 `internal/tool/builtin.go` 中注册两个工具，标签为 `["interact"]`
- [x] 4.4 在 `internal/runtime/mode.go` 中确保模式切换逻辑支持运行时动态变更
- [x] 4.5 编写 plan 模式切换工具的单元测试

## 5. 新增内置工具 — batch

- [x] 5.1 在 `internal/tool/` 新建 `batch.go`，实现 batch 工具：接受工具调用数组（最多 25 个），read 标签并发执行、write/execute 标签串行执行
- [x] 5.2 在 `internal/tool/builtin.go` 中注册 batch 工具，标签为 `["execute"]`
- [x] 5.3 编写 batch 工具的单元测试（并发读、串行写、超出上限）

## 6. 新增内置工具 — skill

- [x] 6.1 在 `internal/tool/` 新建 `skill_tool.go`，实现 skill 工具：列出可用 Skill、加载指定 Skill 内容
- [x] 6.2 在 `internal/tool/builtin.go` 中注册 skill 工具，标签为 `["read"]`
- [x] 6.3 编写 skill 工具的单元测试

## 7. 新增内置工具 — ls

- [x] 7.1 在 `internal/tool/` 新建 `ls.go`，实现 ls 工具：返回目录树结构，尊重 `.gitignore` 和忽略模式
- [x] 7.2 在 `internal/tool/builtin.go` 中注册 ls 工具，标签为 `["read"]`
- [x] 7.3 编写 ls 工具的单元测试（正常列出、忽略模式、路径越界）

## 8. SubAgent 增强 — task 工具

- [x] 8.1 重构 `internal/tool/task.go`，扩展 schema 新增 `task_id`（可选）、`subagent_type`（可选）、`description`（可选）参数
- [x] 8.2 实现 task_id 恢复逻辑：有 task_id 时通过 Store 查找已有子会话并复用，无 task_id 时创建新会话并返回 task_id
- [x] 8.3 实现 subagent_type 参数：根据类型选择对应 Mode 的工具集
- [x] 8.4 实现 description 参数：存储到子会话元数据
- [x] 8.5 编写 task 工具增强功能的单元测试

## 9. Skill 系统增强

- [x] 9.1 重构 `internal/skill/skill.go`，增强发现机制：支持 `.cursor/skills/`、`.agents/skills/` 等多路径递归扫描
- [x] 9.2 实现 SKILL.md 的 YAML frontmatter 解析：提取 name、description、triggers 元数据
- [x] 9.3 实现同名技能覆盖优先级：项目级 > 用户级 > 配置路径
- [x] 9.4 编写 Skill 发现增强的单元测试

## 10. MCP OAuth 认证

- [x] 10.1 在 `internal/mcp/` 新建 `oauth.go`，实现 OAuthClientProvider：授权码流程、本地回调服务器
- [x] 10.2 实现 OAuth token 存储：token 持久化到 `~/.opencode/mcp-auth/`，文件权限 0600
- [x] 10.3 实现 token 自动刷新：access_token 过期时使用 refresh_token 获取新 token
- [x] 10.4 实现 OAuth Dynamic Client Registration（RFC 7591）
- [x] 10.5 修改 `internal/mcp/client.go`，在连接失败（401）时自动触发 OAuth 认证流程
- [x] 10.6 编写 MCP OAuth 的单元测试

## 11. Snapshot 模块

- [x] 11.1 新建 `internal/snapshot/` 包，实现 Snapshot 核心接口：Track、Patch、Restore、Diff
- [x] 11.2 实现基于 git diff/stash 的快照存储后端
- [x] 11.3 实现快照与会话 ID / 步骤标识的关联存储
- [x] 11.4 实现非 git 目录的优雅降级（跳过并警告）
- [x] 11.5 在 `internal/runtime/engine.go` 的 ReAct 循环中集成 Snapshot：step 开始前 Track、step 完成后 Patch
- [x] 11.6 编写 Snapshot 模块的单元测试

## 12. 会话增强 — Revert 与摘要

- [x] 12.1 增强 `internal/store/` 的 Revert 实现：集成 Snapshot 恢复文件状态
- [x] 12.2 实现 unrevert 操作：撤销 revert，恢复被删除的消息和文件状态
- [x] 12.3 实现会话摘要生成：基于 Snapshot diff 和工具调用结果在 step 完成后生成增量摘要
- [x] 12.4 编写会话增强功能的单元测试

## 13. 文件监控模块

- [x] 13.1 新建 `internal/filewatcher/` 包，基于 fsnotify 实现文件变更监控服务
- [x] 13.2 实现 `.gitignore` 和忽略模式过滤
- [x] 13.3 在 write、edit、apply_patch 工具中添加 `file.changed` 事件发布
- [x] 13.4 实现与 Snapshot 模块的集成：订阅文件变更事件标记脏状态
- [x] 13.5 在 go.mod 中添加 `github.com/fsnotify/fsnotify` 依赖
- [x] 13.6 编写文件监控模块的单元测试

## 14. LSP 集成

- [x] 14.1 新建 `internal/lsp/` 包，实现 LSP 客户端协议基础：JSON-RPC 2.0 over stdio、initialize/initialized 握手、shutdown/exit
- [x] 14.2 实现 `textDocument/publishDiagnostics` 通知接收与缓存
- [x] 14.3 实现 `textDocument/definition` 请求
- [x] 14.4 实现 `textDocument/references` 请求
- [x] 14.5 实现 `textDocument/documentSymbol` 请求
- [x] 14.6 在 `internal/tool/` 新建 `lsp.go`，实现 lsp 工具：接受操作类型和参数，委托 LSP 客户端执行
- [x] 14.7 在 `internal/tool/builtin.go` 中注册 lsp 工具，标签为 `["read"]`
- [x] 14.8 编写 LSP 客户端和工具的单元测试

## 15. 集成验证

- [x] 15.1 运行全部单元测试确保通过
- [x] 15.2 通过 `go vet ./...` 和 `golangci-lint run` 检查代码质量
- [ ] 15.3 手动测试 ReAct 循环增强功能（doom loop、compaction）
- [ ] 15.4 手动测试新工具（multiedit、batch、ls、skill、plan）
- [ ] 15.5 手动测试 task 工具的 task_id 恢复功能
