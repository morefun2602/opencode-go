# Capability: acp-bridge

## ADDED Requirements

### Requirement: 与 HTTP 共端口与鉴权

ACP 相关端点 MUST 部署在与 `http-api` 相同的监听地址与 TCP 端口上，且 MUST 复用同一套鉴权中间件（Bearer / 共享密钥等）；独立端口或独立凭据集 MUST NOT 作为首版默认要求。

#### Scenario: 未授权访问被拒绝

- **WHEN** 请求访问 ACP 路由且缺少有效凭据（在需要鉴权的绑定模式下）
- **THEN** 服务器 MUST 返回 HTTP 401

### Requirement: 路由与版本

ACP HTTP 路由 MUST 位于与现有 API 一致的版本前缀之下（例如 `/v1/...` 子路径）；路径表 MUST 在文档中列出。

#### Scenario: 路径带版本前缀

- **WHEN** 客户端调用已文档化的 ACP 端点
- **THEN** 请求路径 MUST 包含版本前缀

### Requirement: 与会话模型映射

ACP 事件与会话标识 MUST 映射到 `agent-runtime` 的会话抽象；未知会话 ID MUST 返回 4xx 并 MUST NOT 创建隐式特权会话。

#### Scenario: 未知会话

- **WHEN** 请求引用不存在的会话 ID
- **THEN** 系统 MUST 返回 4xx 且 MUST NOT 泄漏其他会话数据
