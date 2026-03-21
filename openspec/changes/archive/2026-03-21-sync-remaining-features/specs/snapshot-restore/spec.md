## ADDED Requirements

### Requirement: Wire 接入 Engine

Snapshot 模块 MUST 在 `wire.go` 中创建实例并注入到 Engine 的 Snapshot 字段。当工作区为 git 仓库时 MUST 启用，否则 MUST 设为 nil。

#### Scenario: git 仓库中启用

- **WHEN** 工作区是 git 仓库且 wire.go 初始化 Engine
- **THEN** Engine.Snapshot MUST 为有效的 Snapshot 服务实例

#### Scenario: 非 git 仓库中禁用

- **WHEN** 工作区不是 git 仓库
- **THEN** Engine.Snapshot MUST 为 nil

## MODIFIED Requirements

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
