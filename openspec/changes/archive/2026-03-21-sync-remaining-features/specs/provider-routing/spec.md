## ADDED Requirements

### Requirement: 模型引用解析

系统 MUST 提供 `ParseModel(s string) ModelRef` 函数，将 `"provider/model"` 格式字符串解析为 `ModelRef{ProviderID, ModelID}` 结构。无 `/` 时 ProviderID MUST 为空。

#### Scenario: 完整格式解析

- **WHEN** 输入为 `"anthropic/claude-sonnet-4-20250514"`
- **THEN** 返回 `ModelRef{ProviderID: "anthropic", ModelID: "claude-sonnet-4-20250514"}`

#### Scenario: 仅 model 名

- **WHEN** 输入为 `"gpt-4o"`
- **THEN** 返回 `ModelRef{ProviderID: "", ModelID: "gpt-4o"}`

### Requirement: 默认模型选择

系统 MUST 提供 `Router.DefaultModel() ModelRef` 方法，按以下优先级选择默认模型：（1）config.model 配置值；（2）第一个已注册 Provider 的第一个模型。

#### Scenario: 使用配置模型

- **WHEN** config.model 为 `"anthropic/claude-sonnet-4-20250514"`
- **THEN** DefaultModel() MUST 返回该模型引用

#### Scenario: 回退到第一个可用

- **WHEN** config.model 未配置
- **THEN** DefaultModel() MUST 返回第一个已注册 Provider 的第一个模型

### Requirement: 小模型选择

系统 MUST 提供 `Router.SmallModel() ModelRef` 方法，用于 compaction/title/summary 等内部任务。按以下优先级选择：（1）config.small_model 配置值；（2）按 Provider 类型使用预定义的小模型优先级列表（Anthropic: claude-haiku 系列，OpenAI: gpt-4o-mini 系列）。

#### Scenario: 使用配置小模型

- **WHEN** config.small_model 为 `"openai/gpt-4o-mini"`
- **THEN** SmallModel() MUST 返回该模型引用

#### Scenario: 按 provider 自动选择

- **WHEN** config.small_model 未配置且仅注册了 Anthropic provider
- **THEN** SmallModel() MUST 返回 Anthropic 的 haiku 系列模型

### Requirement: 模型解析

系统 MUST 提供 `Router.Resolve(ref ModelRef) (Provider, string, error)` 方法，根据 ModelRef 返回对应的 Provider 实例和模型 ID。ProviderID 为空时 MUST 在所有已注册 Provider 中搜索匹配的 modelID。

#### Scenario: 精确匹配

- **WHEN** 传入 `ModelRef{ProviderID: "openai", ModelID: "gpt-4o"}`
- **THEN** MUST 返回 OpenAI Provider 实例和 `"gpt-4o"`

#### Scenario: 模糊搜索

- **WHEN** 传入 `ModelRef{ProviderID: "", ModelID: "claude-sonnet-4-20250514"}`
- **THEN** MUST 在已注册 Provider 中找到匹配的 Anthropic Provider 并返回

#### Scenario: 模型不存在

- **WHEN** 传入不存在的模型引用
- **THEN** MUST 返回明确错误

### Requirement: Engine 集成 Router

Engine MUST 使用 Router 替代直接持有单一 Provider。每次 LLM 调用时 Engine MUST 通过 Router 获取 Provider 和模型 ID。

#### Scenario: 使用默认模型

- **WHEN** Engine 执行普通会话的 CompleteTurn
- **THEN** Engine MUST 通过 Router.DefaultModel() 获取 Provider 和模型

#### Scenario: 使用小模型

- **WHEN** Engine 执行 compaction/title/summary 内部任务
- **THEN** Engine MUST 通过 Router.SmallModel() 获取 Provider 和模型
