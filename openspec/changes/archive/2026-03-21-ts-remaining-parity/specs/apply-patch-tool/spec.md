# Capability: apply-patch-tool

## ADDED Requirements

### Requirement: Unified Diff 应用

apply_patch 工具 MUST 接受 `patch` 参数（unified diff 格式字符串），解析其中的文件路径与 hunk，将补丁应用到工作区文件。

#### Scenario: 单文件补丁成功

- **WHEN** patch 包含对单个文件的合法 unified diff 且上下文行匹配
- **THEN** 工具 MUST 将变更写入该文件并返回成功

#### Scenario: 上下文不匹配

- **WHEN** patch 中的上下文行与文件实际内容不匹配
- **THEN** 工具 MUST 返回错误且 MUST NOT 修改文件

#### Scenario: 多文件补丁

- **WHEN** patch 包含对多个文件的 diff
- **THEN** 工具 MUST 依次应用每个文件的变更，任一文件失败时 MUST 回滚已应用的变更

### Requirement: 路径限制

apply_patch 中涉及的文件路径 MUST 受 `ResolveUnder` 限制，禁止修改工作区根之外的文件。

#### Scenario: 越界路径被拒绝

- **WHEN** patch 中的文件路径解析后超出工作区根
- **THEN** 工具 MUST 返回路径错误且 MUST NOT 执行任何 I/O
