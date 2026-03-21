# Proposal：与上游 TypeScript OpenCode 的功能对齐（差距收口）

## Why

当前 Go 实现已完成最小可运行路径（配置、SQLite、`serve`、占位 LLM、极简 HTTP），与上游 **`packages/opencode`**（CLI/TUI、丰富工具链、MCP、多模型提供商、会话高级能力、插件与技能等）相比存在系统性差距；若无规格化对齐，后续实现会重复决策、遗漏行为。本变更将差距**显式化为能力清单与规格**，为分阶段实现提供合同。

## What Changes

- 在 OpenSpec 中**新增**若干能力规格，覆盖上游已具备、Go 尚未建模的横切域（MCP、内置工具、技能、插件扩展等）。
- 对 **`openspec/specs/` 下已有六大能力**做**需求级**增补（delta），使 HTTP 面、CLI、运行时、LLM、持久化等与上游语义对齐（不要求逐文件翻译 TS，但**用户可感知行为与集成点**应对齐或可文档化差异）。
- 明确 **Non-goals**（若纳入本提案）：**桌面/Web/企业独占包**（`packages/desktop`、`packages/app` 等）的完整复刻可作为独立变更；本提案以 **`packages/opencode` 核心运行时能力**为对齐参照。
- 若与既有主规格冲突，**BREAKING** 变更须在对应 delta 中标注迁移说明；新增能力默认不破坏已有主规格中已实现的子集（除非显式 **BREAKING**）。

## Capabilities

### New Capabilities

- `mcp-integration`：MCP 客户端/宿主侧能力（与上游 `@modelcontextprotocol/sdk` 用法对齐：连接、发现、工具调用、错误与重试语义）；**不包含**上游仓库内其他产品线的 MCP 管理 UI。
- `builtin-tools`：内置工具族（与上游 `src/tool/*` 对齐的**命名与最小契约**：read、write、edit、grep、bash、apply_patch、glob 等）及**权限/确认**模型；具体是否一期实现全部工具在 `tasks` 中切片。
- `skills`：技能发现、加载顺序、与工具/会话的交互边界（对齐上游 skill 相关行为）。
- `plugin-extension`：插件加载、生命周期与钩子（对齐 `@opencode-ai/plugin` 的**能力级**目标，非实现细节绑定）。
- `acp-bridge`（可选命名）：与 ACP / Agent Client Protocol 相关的会话与事件桥接（若与上游 `src/acp` 等模块对齐）；范围以「可与编辑器/代理协议集成」为下限。

### Modified Capabilities

（以下主规格文件位于 `openspec/specs/<name>/spec.md`，本变更以 **delta spec** 增补/修改需求。）

- `http-api`：扩展 REST/HTTP 面以覆盖上游控制面中的**会话列表、消息查询/分页、流式与错误语义**等（与当前极简 `/v1` 相比为需求级扩张）。
- `llm-and-tools`：多提供商注册、流式、超时与失败分类；与 `builtin-tools` / `mcp-integration` 的调用链划分。
- `agent-runtime`：会话生命周期（Compaction、Retry、Revert、结构化输出、消息模型版本等）与上游语义对齐的**最小可验证需求集**。
- `cli-and-config`：子命令集、交互式 TUI/CLI 模式、与 `serve` 的组合；配置键与上游一致前提下增加**新键**时的命名空间规则。
- `persistence`：持久化实体与迁移策略与上游 **Drizzle schema 语义**对齐（表/字段级映射在 design 中落地，规格中固定**不变量**）。
- `go-codebase-layout`：因新增模块（MCP、工具、插件等）调整目录与边界时的**可见性与依赖方向**约束。

## Impact

- **代码**：`internal/` 将显著扩张；可能新增 `internal/mcp`、`internal/tool`、`internal/skill`、`internal/plugin` 等包（具体以 `design.md` 为准）。
- **依赖**：可能引入 MCP 相关 Go 库、进程执行与沙箱相关依赖；**BREAKING** 时升级 `go.mod` 与发布说明。
- **API**：HTTP 与 CLI 对外行为变更，需同步 `docs/` 与兼容性引用。
- **测试**：每个能力需配套契约/集成测试策略，在 `tasks` 中拆解。
