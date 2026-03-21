# HTTP API（`/v1`）

## 健康检查

- `GET /v1/health` → `200`，JSON `{"status":"ok"}`

## 会话

- `GET /v1/sessions?limit=` → `{"sessions":[{"id":"...","workspace_id":"...","created_at":<unix>}]}`（按 `workspace.id` 过滤）
- `POST /v1/sessions` → `{"id":"<session-id>"}`
- `GET /v1/sessions/{id}/messages?cursor=&limit=` → `{"messages":[{"id":...,"role":"...","body":"...","parts":"[...]","turn_seq":...,"created_at":...,"model":"...","cost_prompt_tokens":0,"cost_completion_tokens":0,"finish_reason":"...","tool_call_id":"..."}]}`
  - `cursor`：上一页最后一条的 `turn_seq`（严格大于）；首屏传 `0` 或不传。
  - `parts` 为 JSON 字符串，包含结构化的消息部件数组。

## 完成一轮对话（非流式）

- `POST /v1/sessions/{id}/complete`
- 请求体：`{"text":"用户输入"}`
- 响应：`{"messages":[{"role":"assistant","content":"...","parts":[...]}]}`

响应中 `messages` 数组包含 ReAct 循环中产生的最终助手消息。

## 流式（SSE）

- `POST /v1/sessions/{id}/complete?stream=1`（或 `stream=true`）
- `Content-Type: text/event-stream`
- 数据行：`data: {"type":"text","text":"<chunk>"}\n\n`
- 结束：`event: done\ndata: {}\n\n`
- 错误：`data: {"type":"error","message":"<message>"}\n\n`

## OpenAPI

- `GET /v1/openapi.json` → OpenAPI 3 机器可读契约（与实现同步维护）。

## ACP（与 HTTP 共端口）

- `POST /v1/acp/session/event` → 请求体含 `session_id` 与 `event`；未知会话 **404**；成功 **202**。

## 鉴权

若配置了 `server.auth_token`，请求需带：`Authorization: Bearer <token>`。

## 错误体

HTTP 4xx/5xx 时 JSON：

```json
{"error":{"code":"...","message":"..."}}
```
