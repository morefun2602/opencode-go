# Capability: webfetch-tool

## ADDED Requirements

### Requirement: URL 抓取

webfetch 工具 MUST 接受 `url` 参数，通过 HTTP GET 抓取内容并返回纯文本（或简易去标签后的文本）。

#### Scenario: 成功抓取

- **WHEN** 目标 URL 返回 HTTP 200
- **THEN** 工具 MUST 返回响应 body 的文本内容

#### Scenario: 输出截断

- **WHEN** 响应 body 超过 `MaxOutputBytes` 配置
- **THEN** 工具 MUST 截断输出并附加截断提示

### Requirement: 超时与错误

webfetch 工具 MUST 在配置的超时时间内完成请求，超时 MUST 返回超时类错误。非 2xx 状态码 MUST 作为错误返回。

#### Scenario: 请求超时

- **WHEN** 目标 URL 在超时时间内未响应
- **THEN** 工具 MUST 返回超时错误

#### Scenario: HTTP 4xx/5xx

- **WHEN** 目标 URL 返回 404
- **THEN** 工具 MUST 返回包含状态码的错误信息
