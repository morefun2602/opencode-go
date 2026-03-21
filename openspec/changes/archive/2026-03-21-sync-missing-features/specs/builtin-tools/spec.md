## ADDED Requirements

### Requirement: multiedit 工具

系统 MUST 注册名为 `multiedit` 的内置工具，接受文件路径和多组 `{old_string, new_string}` 替换对，在单次调用中对同一文件执行多个位置的编辑。该工具标签 MUST 为 `["write"]`。每组替换的 old_string MUST 在文件中唯一存在。

#### Scenario: 多位置编辑成功

- **WHEN** 调用 multiedit 传入 3 组替换对且所有 old_string 均在文件中唯一存在
- **THEN** 系统 MUST 按顺序执行所有替换并返回成功

#### Scenario: 某组 old_string 不存在

- **WHEN** 某组替换的 old_string 在文件中不存在
- **THEN** 系统 MUST 返回错误指明哪一组替换失败，且 MUST NOT 执行任何替换（原子性）

#### Scenario: old_string 不唯一

- **WHEN** 某组 old_string 在文件中出现多次
- **THEN** 系统 MUST 返回错误指明歧义，且 MUST NOT 执行任何替换

### Requirement: plan_enter 工具

系统 MUST 注册名为 `plan_enter` 的内置工具，调用时将当前会话的 agent 模式切换为 plan 模式。该工具标签 MUST 为 `["interact"]`。

#### Scenario: 切换到 plan 模式

- **WHEN** 当前模式为 build 且调用 plan_enter
- **THEN** 会话模式 MUST 切换为 plan，后续工具调用 MUST 仅包含 plan 模式允许的工具

#### Scenario: 已在 plan 模式

- **WHEN** 当前模式已是 plan 且调用 plan_enter
- **THEN** 系统 MUST 返回提示信息说明已在 plan 模式

### Requirement: plan_exit 工具

系统 MUST 注册名为 `plan_exit` 的内置工具，调用时将当前会话的 agent 模式从 plan 模式切换回 build 模式。切换前 MUST 通过 Question.Ask 询问用户确认。该工具标签 MUST 为 `["interact"]`。

#### Scenario: 退出 plan 模式

- **WHEN** 当前模式为 plan 且调用 plan_exit 且用户确认
- **THEN** 会话模式 MUST 切换为 build

#### Scenario: 用户拒绝退出

- **WHEN** 调用 plan_exit 但用户拒绝确认
- **THEN** 模式 MUST 保持 plan 不变

### Requirement: batch 工具

系统 MUST 注册名为 `batch` 的内置工具，接受一组工具调用定义（最多 25 个），并发执行所有调用并统一返回结果。标签为 `["read"]` 的工具 MUST 并发执行，标签包含 `["write"]` 或 `["execute"]` 的工具 MUST 串行执行。该工具标签 MUST 为 `["execute"]`。

#### Scenario: 并发读操作

- **WHEN** batch 中包含 5 个 read 和 3 个 glob 调用
- **THEN** 这 8 个只读调用 MUST 并发执行

#### Scenario: 串行写操作

- **WHEN** batch 中包含 2 个 edit 调用
- **THEN** 这 2 个写调用 MUST 串行执行

#### Scenario: 超出上限

- **WHEN** batch 中包含超过 25 个调用
- **THEN** 系统 MUST 返回错误

### Requirement: skill 工具

系统 MUST 注册名为 `skill` 的内置工具，接受技能名称参数，加载指定 Skill 的指令和资源内容并返回给 Agent。该工具标签 MUST 为 `["read"]`。

#### Scenario: 加载已存在的 Skill

- **WHEN** 调用 skill 工具并传入有效技能名称
- **THEN** 系统 MUST 返回该 Skill 的完整指令内容

#### Scenario: Skill 不存在

- **WHEN** 调用 skill 工具但指定名称不存在
- **THEN** 系统 MUST 返回错误列出可用 Skill 列表

### Requirement: ls 工具

系统 MUST 注册名为 `ls` 的内置工具，接受目录路径参数，返回目录树结构。结果 MUST 尊重 `.gitignore` 和配置的忽略模式。该工具标签 MUST 为 `["read"]`。

#### Scenario: 列出目录

- **WHEN** 调用 ls 工具传入有效目录路径
- **THEN** 系统 MUST 返回该目录的树形结构列表

#### Scenario: 忽略模式生效

- **WHEN** 目录包含 `.gitignore` 忽略的文件
- **THEN** 结果 MUST NOT 包含被忽略的文件

#### Scenario: 路径越界

- **WHEN** 传入工作区外的路径
- **THEN** 系统 MUST 返回错误

## MODIFIED Requirements

### Requirement: 标签分类表

内置工具 MUST 按以下分类声明标签：
- `read`：read、glob、grep、webfetch、websearch、skill、ls、lsp
- `write`：write、edit、apply_patch、todowrite、multiedit
- `execute`：bash、task、batch
- `interact`：question、plan_enter、plan_exit

#### Scenario: 标签验证

- **WHEN** 系统启动并注册所有内置工具
- **THEN** 每个工具 MUST 至少有一个标签
