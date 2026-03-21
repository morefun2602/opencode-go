## ADDED Requirements

### Requirement: MCP OAuth 认证

系统 MUST 支持通过 OAuth 2.0 授权码流程对需要认证的 MCP 服务端进行身份验证。系统 MUST 实现 OAuthClientProvider 接口，管理 token 的获取、存储和刷新。

#### Scenario: 首次 OAuth 认证

- **WHEN** 连接需要 OAuth 认证的 MCP 服务端且无已存储 token
- **THEN** 系统 MUST 启动本地回调服务器、打开浏览器引导用户授权、接收回调并存储 token

#### Scenario: Token 自动刷新

- **WHEN** 已存储的 access_token 已过期但 refresh_token 有效
- **THEN** 系统 MUST 自动使用 refresh_token 获取新 access_token

#### Scenario: Token 完全失效

- **WHEN** access_token 和 refresh_token 均已过期
- **THEN** 系统 MUST 重新启动完整 OAuth 授权流程

### Requirement: OAuth 回调处理

系统 MUST 在 OAuth 流程中启动临时本地 HTTP 服务器（默认端口可配置），监听授权回调。回调处理完成后 MUST 关闭临时服务器。

#### Scenario: 回调成功

- **WHEN** OAuth 授权服务器重定向到本地回调 URL 并携带有效授权码
- **THEN** 系统 MUST 用授权码交换 token 并存储

#### Scenario: 回调超时

- **WHEN** 用户未在超时时间内完成授权
- **THEN** 系统 MUST 关闭回调服务器并返回超时错误

### Requirement: OAuth 凭证存储

系统 MUST 将 OAuth token（access_token、refresh_token、过期时间）安全存储到文件系统。token 文件 MUST 仅限当前用户可读。

#### Scenario: Token 持久化

- **WHEN** OAuth 流程成功获取 token
- **THEN** 系统 MUST 将 token 存储到 `~/.opencode/mcp-auth/` 目录下，文件权限 MUST 为 0600

### Requirement: MCP 客户端动态注册

系统 MUST 支持 OAuth Dynamic Client Registration（RFC 7591），允许 MCP 客户端向授权服务器自动注册。

#### Scenario: 动态注册成功

- **WHEN** MCP 服务端支持动态注册且客户端首次连接
- **THEN** 系统 MUST 自动注册客户端并保存 client_id/client_secret

## MODIFIED Requirements

### Requirement: MCP 客户端连接与发现

系统 MUST 支持作为 MCP **客户端**连接已配置的 MCP 服务端（传输方式以实现为准，例如 stdio 或 HTTP），并在连接成功后 MUST 拉取工具/资源清单；连接失败时 MUST 返回可分类错误并记录日志，且 MUST NOT 静默忽略。当 MCP 服务端要求 OAuth 认证时，系统 MUST 自动触发 OAuth 认证流程。

#### Scenario: 配置存在时尝试连接

- **WHEN** 配置中声明至少一个 MCP 服务端且进程启动或显式连接
- **THEN** 系统 MUST 建立连接或返回明确错误，且工具清单 MUST 可被内部注册表消费

#### Scenario: 需要 OAuth 认证的服务端

- **WHEN** MCP 服务端返回 401 未授权错误
- **THEN** 系统 MUST 自动启动 OAuth 认证流程，成功后重试连接
