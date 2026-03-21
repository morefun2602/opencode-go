# 持久化：上游逻辑实体 ↔ 本地 SQLite

本文档与 `openspec/specs` 中的持久化规范交叉引用。

| 上游概念 | 本地表 | 主要列 |
|---------|--------|--------|
| Workspace | `workspaces` | `id` |
| Session | `sessions` | `id`, `workspace_id`, `created_at` |
| Message | `messages` | 见下表 |

## messages 表（schema v3）

| 列名 | 类型 | 说明 |
|------|------|------|
| `id` | `INTEGER PRIMARY KEY` | 自增 ID |
| `workspace_id` | `TEXT NOT NULL` | 所属工作区 |
| `session_id` | `TEXT NOT NULL` | 所属会话 |
| `role` | `TEXT NOT NULL` | `user` / `assistant` / `tool` |
| `body` | `TEXT NOT NULL` | 纯文本内容（兼容旧版） |
| `turn_seq` | `INTEGER NOT NULL` | 同一会话内单调递增序号 |
| `created_at` | `INTEGER NOT NULL` | Unix 时间戳 |
| `message_version` | `INTEGER NOT NULL DEFAULT 1` | 消息模型版本 |
| `parts` | `TEXT NOT NULL DEFAULT '[]'` | JSON 数组，结构化消息部件 |
| `model` | `TEXT NOT NULL DEFAULT ''` | 使用的模型名称 |
| `cost_prompt_tokens` | `INTEGER NOT NULL DEFAULT 0` | 输入 token 数 |
| `cost_completion_tokens` | `INTEGER NOT NULL DEFAULT 0` | 输出 token 数 |
| `finish_reason` | `TEXT NOT NULL DEFAULT ''` | `stop` / `tool_calls` / `length` |
| `tool_call_id` | `TEXT NOT NULL DEFAULT ''` | 工具调用关联 ID |

## parts JSON 结构

每个 part 是一个对象，`type` 字段标识类型：

- `{"type":"text","text":"..."}` — 文本内容
- `{"type":"tool_call","tool_call_id":"...","tool_name":"...","args":{...}}` — 工具调用
- `{"type":"tool_result","tool_call_id":"...","result":"...","is_error":false}` — 工具结果

## 迁移

- **v1→v2**：添加 `message_version` 列。
- **v2→v3**：添加 `parts`、`model`、`cost_prompt_tokens`、`cost_completion_tokens`、`finish_reason`、`tool_call_id` 列；已有记录的 `body` 自动包装为 `[{"type":"text","text":"<body>"}]` 格式的 `parts`。

## 不变量

- 消息属于唯一 `(workspace_id, session_id)`；`turn_seq` 在同一会话内单调递增。
- ReAct 循环中产生的所有消息在单一事务中原子写入。
