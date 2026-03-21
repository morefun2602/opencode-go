# builtin-tools Specification

## Purpose

TBD

## Requirements

### Requirement: 内置工具注册与 schema

系统 MUST 为每个内置工具提供稳定名称、JSON 参数 schema 以及标签集合（`Tags []string`）。标签 MUST 包括 `read`、`write`、`execute` 中的一个或多个。未通过校验的调用 MUST NOT 执行副作用。所有工具的输出 MUST 经过统一截断服务处理，各工具 MUST NOT 自行实现截断逻辑。

#### Scenario: 工具包含标签

- **WHEN** 注册内置工具 `edit`
- **THEN** 工具定义 MUST 包含 `Tags: ["write"]`

#### Scenario: 非法参数被拒绝

- **WHEN** 模型或调用方传入不符合 schema 的参数
- **THEN** 系统 MUST NOT 执行该工具并 MUST 返回校验错误

#### Scenario: 输出统一截断

- **WHEN** 任何工具返回超过截断限制的输出
- **THEN** Registry MUST 在返回前通过截断服务截断输出

### Requirement: 读文件与列目录类工具

系统 MUST 提供与上游语义对齐的只读类工具（至少包含 **read** 与目录列举能力之一；名称以实现对齐文档为准），且 MUST 将路径解析限制在工作区根或其允许范围内。

#### Scenario: 越界路径被拒绝

- **WHEN** 请求访问工作区根之外的禁止路径
- **THEN** 工具 MUST 失败并返回明确错误

### Requirement: 写文件与编辑类工具

系统 MUST 提供写入与编辑类能力（至少覆盖 **write**、**edit** 或与上游等价的 **apply_patch** 之一的分阶段落地），且 MUST 在覆盖或删除前遵守 `agent-runtime` / 配置中的确认策略（若启用）。

#### Scenario: 需确认时未确认则不写

- **WHEN** 策略要求用户确认写操作且未收到确认
- **THEN** 系统 MUST NOT 提交磁盘写入

### Requirement: 搜索与执行类工具

系统 MUST 规划 **grep** / **glob** 类搜索工具与 **bash**（或受控 shell）类执行工具；执行类工具 MUST 具备超时、输出上限与退出码捕获，且 MUST 将会话关联写入日志。工具 MUST NOT 自行截断输出，MUST 依赖统一截断服务。

#### Scenario: 执行超时

- **WHEN** shell 执行超过配置超时
- **THEN** 系统 MUST 终止该执行并 MUST 向编排层返回超时类错误

#### Scenario: 输出由截断服务处理

- **WHEN** bash 工具返回大量输出
- **THEN** 输出 MUST 由 Registry 层截断服务处理，bash 工具本身 MUST NOT 截断

### Requirement: invalid 工具

系统 MUST 注册名为 `invalid` 的内置工具，接受 `tool`（工具名）和 `error`（错误描述）参数。当 Engine 遇到畸形工具调用（schema 校验失败或参数无法解析）时 MUST 将调用路由到 invalid 工具。该工具 MUST 返回描述性错误信息供模型学习。该工具 MUST NOT 出现在提供给模型的活跃工具列表中。

#### Scenario: 畸形工具调用路由

- **WHEN** 模型返回的 tool_call 参数不符合目标工具的 schema
- **THEN** Engine MUST 将该调用路由到 invalid 工具，传入原始工具名和错误描述

#### Scenario: invalid 工具返回

- **WHEN** invalid 工具被调用
- **THEN** MUST 返回 `"The arguments provided to tool '<tool>' are invalid: <error>"` 格式的字符串

#### Scenario: 不在活跃工具列表

- **WHEN** Engine 收集工具定义传给 Provider
- **THEN** invalid 工具 MUST NOT 包含在工具列表中

### Requirement: read 工具 offset/limit

read 工具 MUST 支持可选的 `offset`（起始行号，1-based）和 `limit`（读取行数）参数。未提供时 MUST 读取全文件。单行超过 2000 字符时 MUST 截断该行并附加提示。

#### Scenario: 部分文件读取

- **WHEN** 调用 read 工具传入 offset=10 和 limit=20
- **THEN** MUST 返回文件第 10 到 29 行的内容

#### Scenario: 仅 offset 无 limit

- **WHEN** 调用 read 工具传入 offset=50 但未传 limit
- **THEN** MUST 返回从第 50 行到文件末尾的内容

#### Scenario: 长行截断

- **WHEN** 文件某行超过 2000 字符
- **THEN** 该行 MUST 被截断到 2000 字符并附加 `... (line truncated)` 提示

#### Scenario: offset 超出文件范围

- **WHEN** offset 大于文件总行数
- **THEN** MUST 返回空内容并附加提示说明文件仅有 N 行

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

#### Scenario: 结构化输出格式

- **WHEN** 调用 skill 工具并传入有效技能名称
- **THEN** 输出 MUST 使用 `<skill_content name="...">` XML 格式
- **AND** 输出 MUST 包含技能正文、base 目录（`file://` URI）、技能目录下的文件列表（`<skill_files>` 子元素）
- **AND** 文件列表 MUST 排除 SKILL.md 本身
- **AND** 文件列表 MUST 最多包含 10 个文件

#### Scenario: 动态工具描述

- **WHEN** 注册 skill 工具时
- **THEN** 工具描述 MUST 根据当前可用技能列表动态生成
- **AND** 描述 MUST 包含每个技能的名称和摘要（concise 格式）
- **AND** 参数 `name` 的描述 MUST 包含示例技能名称（最多 3 个）

#### Scenario: 无可用技能时

- **WHEN** 可用技能列表为空
- **THEN** 工具描述 MUST 说明当前无可用技能

#### Scenario: 权限校验

- **WHEN** 调用 skill 工具加载技能
- **THEN** 系统 MUST 通过 Policy 进行权限检查
- **AND** 权限为 `deny` 时 MUST 拒绝加载
- **AND** 权限为 `ask` 时 MUST 请求用户确认

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

### Requirement: 工具名解析与路由

#### Scenario: 工具名大小写修复

- **WHEN** LLM 返回的 tool_call 名称为 "Read" 但注册名为 "read"
- **THEN** Router MUST 尝试小写匹配并使用匹配到的工具执行
- **AND** MUST NOT 路由到 invalid 工具

#### Scenario: 大小写修复失败

- **WHEN** LLM 返回的 tool_call 名称为 "UnknownTool" 且小写 "unknowntool" 也不存在
- **THEN** Router MUST 路由到 invalid 工具

### Requirement: 标签分类表

内置工具 MUST 按以下分类声明标签：
- `read`：read、glob、grep、webfetch、websearch、skill、ls、lsp
- `write`：write、edit、apply_patch、todowrite、multiedit
- `execute`：bash、task、batch
- `interact`：question、plan_enter、plan_exit

#### Scenario: 标签验证

- **WHEN** 系统启动并注册所有内置工具
- **THEN** 每个工具 MUST 至少有一个标签
