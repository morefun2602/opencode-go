## ADDED Requirements

### Requirement: 工具调用卡片基础框架

每个 `tool_call` Part MUST 渲染为结构化卡片，包含：
- 工具名称（加粗）
- 关键参数摘要（单行）
- 执行状态图标：⟳ running、✓ done、✗ error
- 可选：结果摘要（来自配对的 tool_result）

卡片 MUST 使用 theme.Border 色边框包裹，与普通文本区分。

#### Scenario: 工具卡片基本渲染

- **WHEN** assistant 消息包含 tool_call Part（name=read, args={path: "/foo/bar.go"}）
- **THEN** TUI MUST 显示带边框的卡片：工具名 "read" + 参数摘要 "path: /foo/bar.go" + 状态图标

### Requirement: bash 工具卡片

bash 工具 MUST 显示：
- 命令文本（截断为单行，最多 80 字符）
- 执行结果：成功时显示输出前 3 行，失败时显示错误

#### Scenario: bash 成功

- **WHEN** bash 工具调用成功且输出为 10 行
- **THEN** 卡片 MUST 显示命令 + 前 3 行输出 + "...7 more lines"

### Requirement: read 工具卡片

read 工具 MUST 显示文件路径，结果区域显示 "(N lines read)"。

#### Scenario: read 文件

- **WHEN** read 工具返回 50 行文件内容
- **THEN** 卡片 MUST 显示 "read" + 文件路径 + "(50 lines read)"

### Requirement: edit/write 工具卡片

edit 和 write 工具 MUST 显示文件路径，结果区域显示操作摘要。

#### Scenario: edit 文件

- **WHEN** edit 工具成功修改文件
- **THEN** 卡片 MUST 显示 "edit" + 文件路径 + ✓

### Requirement: grep/glob 工具卡片

grep 和 glob 工具 MUST 显示搜索模式，结果区域显示匹配数量。

#### Scenario: grep 搜索

- **WHEN** grep 工具搜索 pattern "func.*Test" 返回 5 个匹配
- **THEN** 卡片 MUST 显示 "grep" + pattern + "(5 matches)"

### Requirement: 通用工具卡片 fallback

未特殊处理的工具 MUST 使用通用卡片：显示工具名 + 参数 JSON 摘要（截断为 60 字符）。

#### Scenario: 未知工具

- **WHEN** tool_call 的工具名不在专用列表中
- **THEN** MUST 使用通用卡片渲染

### Requirement: 结果折叠

工具结果超过 5 行时 MUST 默认折叠，仅显示前 3 行 + 折叠提示。

#### Scenario: 长输出折叠

- **WHEN** 工具结果超过 5 行
- **THEN** 卡片 MUST 仅显示前 3 行
- **AND** 显示 "▸ N more lines" 折叠提示
