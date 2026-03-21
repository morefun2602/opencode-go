# permission Specification

## Purpose

引入结构化的 Permission Ruleset 系统，替代简单的 Deny/Allow 列表，支持 pattern 通配符匹配和 ask 交互行为，与 TypeScript 参考实现对齐。

## Requirements

### Requirement: Rule 数据结构

系统 MUST 定义 `Rule` 结构体，包含：
- `Permission string` — 权限标识（对应工具名或工具组，如 `"edit"` 匹配所有编辑类工具）
- `Pattern string` — 匹配模式（如 `"*"` 通配所有、`"internal-*"` 前缀匹配）
- `Action string` — 行为：`"allow"`、`"deny"` 或 `"ask"`

`Ruleset` MUST 为 `[]Rule` 类型。

#### Scenario: Rule 结构

- **WHEN** 创建 Rule `{Permission: "edit", Pattern: "*", Action: "deny"}`
- **THEN** 该规则 MUST 表示拒绝所有编辑操作

### Requirement: Disabled 函数

系统 MUST 实现 `Disabled(toolNames []string, ruleset Ruleset) map[string]bool` 函数。对每个工具名，使用 `ruleset` 中最后匹配的规则确定行为。`Action` 为 `"deny"` 的工具 MUST 出现在返回的 disabled 集合中。

编辑类工具（edit、write、apply_patch、multiedit）MUST 映射到统一的 permission 名 `"edit"`。其他工具使用自身 name 作为 permission。

#### Scenario: 编辑工具统一映射

- **WHEN** Ruleset 包含 `{Permission: "edit", Pattern: "*", Action: "deny"}`
- **THEN** Disabled MUST 返回包含 edit、write、apply_patch、multiedit 的集合

#### Scenario: 精确工具名匹配

- **WHEN** Ruleset 包含 `{Permission: "bash", Pattern: "*", Action: "deny"}`
- **THEN** Disabled MUST 仅包含 bash

#### Scenario: 最后匹配规则优先

- **WHEN** Ruleset 为 `[{Permission: "*", Pattern: "*", Action: "deny"}, {Permission: "read", Pattern: "*", Action: "allow"}]`
- **THEN** read MUST NOT 在 Disabled 中（后规则覆盖）
- **AND** 其他所有工具 MUST 在 Disabled 中

### Requirement: Pattern 通配符匹配

`Pattern` 字段 MUST 支持：
- `"*"` — 匹配所有
- `"prefix*"` — 前缀匹配（如 `"internal-*"` 匹配 `"internal-tool1"`）
- 精确匹配 — 完全相等

#### Scenario: 前缀匹配

- **WHEN** Rule 的 Pattern 为 `"internal-*"` 且工具名为 `"internal-debug"`
- **THEN** 该 Rule MUST 匹配

#### Scenario: 精确匹配

- **WHEN** Rule 的 Pattern 为 `"bash"` 且工具名为 `"bash"`
- **THEN** 该 Rule MUST 匹配

### Requirement: Merge 函数

系统 MUST 实现 `Merge(defaults, overrides Ruleset) Ruleset` 函数。结果为 defaults 追加 overrides，后出现的规则优先生效。

#### Scenario: 覆盖默认规则

- **WHEN** defaults deny "edit"，overrides allow "edit"
- **THEN** Merge 结果中 "edit" MUST 为 allow（overrides 排在后面，findLast 匹配）

### Requirement: Evaluate 函数

系统 MUST 实现 `Evaluate(permission, target string, ruleset Ruleset) Action` 函数，返回匹配到的 Action（allow/deny/ask）。无匹配规则时默认为 `"allow"`。

#### Scenario: 无匹配规则

- **WHEN** Ruleset 为空且查询 "read"
- **THEN** Evaluate MUST 返回 "allow"

#### Scenario: ask 行为

- **WHEN** Ruleset 包含 `{Permission: "bash", Pattern: "*", Action: "ask"}`
- **THEN** Evaluate("bash", "rm -rf /", ruleset) MUST 返回 "ask"

### Requirement: 内置 Agent Permission 迁移

所有内置 Agent 的 `ToolPermission` MUST 迁移为 `Permission Ruleset`：
- `AgentBuild`：空 Ruleset（全部 allow）
- `AgentPlan`：`[{Permission: "edit", Pattern: "*", Action: "deny"}, {Permission: "bash", Pattern: "*", Action: "deny"}]` + Mode Tags 等效
- `AgentExplore`：`[{Permission: "*", Pattern: "*", Action: "deny"}, {Permission: "read", Pattern: "*", Action: "allow"}, {Permission: "bash", Pattern: "*", Action: "allow"}]`
- `AgentGeneral`：`[{Permission: "todowrite", Pattern: "*", Action: "deny"}]`
- 隐藏 Agent（compaction/title/summary）：`[{Permission: "*", Pattern: "*", Action: "deny"}]`

#### Scenario: plan Agent 迁移后等效

- **WHEN** 使用 plan Agent 的 Permission Ruleset 过滤工具
- **THEN** 结果 MUST 与现有 ModePlan Tags 过滤结果一致（排除 write/execute 标签工具）
