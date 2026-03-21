# edit-tool Specification

## Purpose

TBD

## Requirements

### Requirement: 精确文本替换

edit 工具 MUST 接受 `path`、`old_string`、`new_string` 三个参数，在文件中将 `old_string` 精确替换为 `new_string`。

#### Scenario: 唯一匹配替换成功

- **WHEN** `old_string` 在目标文件中恰好出现一次
- **THEN** 工具 MUST 将该处替换为 `new_string` 并返回成功

#### Scenario: old_string 不存在

- **WHEN** `old_string` 在目标文件中不存在
- **THEN** 工具 MUST 返回错误且 MUST NOT 修改文件

#### Scenario: old_string 出现多次

- **WHEN** `old_string` 在目标文件中出现多于一次
- **THEN** 工具 MUST 返回错误提示"非唯一匹配"且 MUST NOT 修改文件

### Requirement: 工作区路径限制

edit 工具的 `path` 参数 MUST 受 `ResolveUnder` 限制，禁止访问工作区根之外的文件。

#### Scenario: 越界路径被拒绝

- **WHEN** `path` 解析后超出工作区根
- **THEN** 工具 MUST 返回路径错误且 MUST NOT 执行任何 I/O
