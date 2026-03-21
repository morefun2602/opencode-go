# snapshot-restore Specification

## Purpose

定义工作区快照与恢复能力，包括快照追踪、增量记录、快照恢复、差异对比以及与会话 Revert 的集成。

## Requirements

### Requirement: 快照追踪

系统 MUST 提供 Snapshot 模块（`internal/snapshot/`），支持在指定时间点对工作区文件状态创建快照。快照 MUST 记录文件的 git diff 信息。

#### Scenario: 创建快照

- **WHEN** 调用 Snapshot.Track() 并指定会话 ID 和步骤标识
- **THEN** 系统 MUST 保存当前工作区相对于 HEAD 的 diff 并关联到该步骤

#### Scenario: 非 git 目录

- **WHEN** 工作区不是 git 仓库
- **THEN** 系统 MUST 跳过快照操作并记录警告，MUST NOT 导致错误

### Requirement: 快照增量记录

系统 MUST 支持在两个时间点之间记录增量变更（patch），用于追踪每个 ReAct step 的文件变更。

#### Scenario: 记录步骤增量

- **WHEN** 一个 ReAct step 完成后调用 Snapshot.Patch()
- **THEN** 系统 MUST 记录该 step 期间产生的文件变更 diff

### Requirement: 快照恢复

系统 MUST 支持将工作区文件状态恢复到指定快照点。

#### Scenario: 恢复成功

- **WHEN** 调用 Snapshot.Restore() 并指定快照标识
- **THEN** 工作区文件 MUST 恢复到该快照记录时的状态

#### Scenario: 快照不存在

- **WHEN** 指定的快照标识不存在
- **THEN** 系统 MUST 返回明确错误

### Requirement: 快照差异对比

系统 MUST 支持对比两个快照之间的文件差异，返回统一 diff 格式。

#### Scenario: 对比差异

- **WHEN** 调用 Snapshot.Diff() 并指定两个快照标识
- **THEN** 系统 MUST 返回两个时间点之间的 unified diff

### Requirement: Wire 接入 Engine

Snapshot 模块 MUST 在 `wire.go` 中创建实例并注入到 Engine 的 Snapshot 字段。当工作区为 git 仓库时 MUST 启用，否则 MUST 设为 nil。

#### Scenario: git 仓库中启用

- **WHEN** 工作区是 git 仓库且 wire.go 初始化 Engine
- **THEN** Engine.Snapshot MUST 为有效的 Snapshot 服务实例

#### Scenario: 非 git 仓库中禁用

- **WHEN** 工作区不是 git 仓库
- **THEN** Engine.Snapshot MUST 为 nil

### Requirement: 与会话 Revert 集成

Snapshot 模块 MUST 支持会话 Revert 操作：当会话 revert 到某消息位置时，工作区文件 MUST 同步恢复到对应快照。Revert 操作 MUST 在 Store.Revert() 或其调用方中触发 Snapshot.Restore()。

#### Scenario: Revert 恢复文件状态

- **WHEN** 用户 revert 会话到 seq=5 且该位置有关联快照
- **THEN** 系统 MUST 调用 Snapshot.Restore() 恢复工作区文件到 seq=5 时的状态

#### Scenario: Snapshot 不可用时 Revert 仅回退消息

- **WHEN** 用户 revert 会话但 Snapshot 不可用（非 git 仓库或 Engine.Snapshot 为 nil）
- **THEN** Revert MUST 仅回退消息，MUST NOT 报错

#### Scenario: 无关联快照时跳过恢复

- **WHEN** 用户 revert 到某位置但该位置无关联快照
- **THEN** Revert MUST 仅回退消息并记录警告
