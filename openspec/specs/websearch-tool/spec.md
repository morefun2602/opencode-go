# websearch-tool Specification

## Purpose

定义 websearch 工具的网络搜索能力与可配置的搜索后端。

## Requirements

### Requirement: 网络搜索

websearch 工具 MUST 接受 `query` 参数，通过可配置的搜索后端执行查询并返回结果摘要。默认后端 MUST 为可配置的 HTTP API 端点。

#### Scenario: 搜索成功

- **WHEN** 调用 websearch 并传入合法查询字符串
- **THEN** 工具 MUST 返回包含搜索结果标题、摘要与 URL 的文本

#### Scenario: 搜索后端不可用

- **WHEN** 搜索后端返回非 2xx 或超时
- **THEN** 工具 MUST 返回错误信息而非空结果

### Requirement: 搜索后端配置

搜索后端 URL MUST 通过 `x_opencode_go.websearch_url` 配置。未配置时工具 MUST 返回提示要求配置搜索后端。

#### Scenario: 未配置后端

- **WHEN** websearch_url 未配置且模型调用 websearch
- **THEN** 工具 MUST 返回配置缺失的错误提示
