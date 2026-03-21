# Delta: http-api

本文件为相对 `openspec/specs/http-api/spec.md` 的增量需求。

## ADDED Requirements

### Requirement: 会话集合查询

系统 MUST 提供已文档化的 HTTP 端点用于列出会话（至少支持按工作区或项目过滤中的一种）；响应 MUST 为 JSON 且 MUST 包含稳定字段（会话 id、创建时间等，具体字段以 OpenAPI 或文档为准）。

#### Scenario: 列表成功

- **WHEN** 客户端请求会话列表且凭据有效
- **THEN** 响应 MUST 为 200 且 MUST 包含会话条目数组或分页包装

### Requirement: 消息分页查询

系统 MUST 提供按会话查询消息历史的端点，且 MUST 支持分页参数（例如 cursor/limit）；消息顺序 MUST 与因果顺序一致。

#### Scenario: 分页参数生效

- **WHEN** 客户端传入分页参数请求消息
- **THEN** 返回集 MUST 受分页约束且 MUST 不包含其他会话消息

### Requirement: OpenAPI 或机器可读契约

系统 MUST 为公开 HTTP API 提供 OpenAPI 文档或等价的机器可读契约（路径可固定，例如 `/openapi.json`），且 MUST 与实现同步更新。

#### Scenario: 可获取契约

- **WHEN** 客户端请求契约端点
- **THEN** 响应 MUST 为 200 且 MUST 可解析为 OpenAPI 或文档化格式
