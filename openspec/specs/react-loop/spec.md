# react-loop Specification

## Purpose

TBD

## Requirements

### Requirement: 消息历史加载

Engine MUST 在每轮对话开始前从持久化层加载当前会话的全部历史消息（含 system、user、assistant、tool 角色），并将其转换为 Provider 可消费的 `[]Message` 格式。

#### Scenario: 历史加载成功

- **WHEN** 用户在已有若干轮对话的会话中发送新消息
- **THEN** Engine MUST 将所有历史消息与新用户消息一并传给 Provider

### Requirement: 工具定义注入

Engine MUST 在调用 Provider 前收集当前可用的全部工具定义（内置 + MCP），并以 `[]ToolDef` 形式传给 Provider。

#### Scenario: 工具列表包含内置与 MCP 工具

- **WHEN** 会话配置了内置工具与至少一个 MCP 服务端
- **THEN** Provider 收到的 tools 列表 MUST 包含两类工具的定义

### Requirement: tool_calls 解析与执行

当 Provider 返回的 `FinishReason` 为 `"tool_calls"` 时，Engine MUST 遍历响应中的每个 tool_call，通过 `ToolRouter` 解析并执行，将结果构建为 `role=tool` 的消息回注到消息列表。

#### Scenario: 模型请求调用已注册工具

- **WHEN** Provider 返回包含一个已知工具名的 tool_call
- **THEN** Engine MUST 执行该工具并将结果消息追加到消息历史

#### Scenario: 模型请求调用未知工具

- **WHEN** Provider 返回包含未注册工具名的 tool_call
- **THEN** Engine MUST 将包含错误信息的 tool_result 消息追加到消息历史，且 MUST NOT 中断循环

### Requirement: 循环终止条件

Engine MUST 在以下任一条件满足时终止 ReAct 循环：（1）Provider 返回 `FinishReason != "tool_calls"`；（2）循环轮数达到 `MaxToolRounds` 上限。

#### Scenario: 最大轮数保护

- **WHEN** 循环达到 `MaxToolRounds`（默认 25）次且 Provider 仍返回 tool_calls
- **THEN** Engine MUST 终止循环并将当前助手消息作为最终响应

### Requirement: 消息持久化

Engine MUST 在一轮对话（含所有中间 tool_call/result）结束后将全部新增消息原子写入持久化层。

#### Scenario: 含工具调用的完整轮次

- **WHEN** 一轮对话产生用户消息、助手消息（含 tool_calls）、tool_result 消息、最终助手消息共 N 条
- **THEN** 持久化层 MUST 包含全部 N 条新消息且顺序 MUST 与因果顺序一致
