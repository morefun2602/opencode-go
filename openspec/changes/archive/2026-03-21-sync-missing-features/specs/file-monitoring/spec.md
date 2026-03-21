## ADDED Requirements

### Requirement: 文件变更监控服务

系统 MUST 提供文件监控模块（`internal/filewatcher/`），使用 fsnotify 监控工作区目录的文件创建、修改、删除事件。

#### Scenario: 监控启动

- **WHEN** Engine 初始化时
- **THEN** 文件监控服务 MUST 启动并监控工作区根目录

#### Scenario: 检测外部文件变更

- **WHEN** 工作区内某文件被外部程序修改
- **THEN** 监控服务 MUST 发布 `file.changed` 事件到 Bus

### Requirement: 工具写操作事件发布

write、edit、apply_patch 工具在成功执行文件写操作后 MUST 发布 `file.changed` 事件到 Bus，包含变更文件路径。

#### Scenario: edit 工具触发事件

- **WHEN** edit 工具成功修改了文件 `main.go`
- **THEN** Bus MUST 收到 `file.changed` 事件，payload 包含文件路径 `main.go`

#### Scenario: write 工具触发事件

- **WHEN** write 工具成功写入新文件
- **THEN** Bus MUST 收到 `file.changed` 事件

### Requirement: 忽略模式

文件监控 MUST 尊重 `.gitignore` 和配置的忽略模式，被忽略的文件变更 MUST NOT 触发事件。

#### Scenario: 忽略 node_modules

- **WHEN** `node_modules/` 目录下文件变更
- **THEN** 监控服务 MUST NOT 发布事件

### Requirement: 与 Snapshot 集成

文件变更事件 MUST 可被 Snapshot 模块订阅，用于触发自动快照或标记脏状态。

#### Scenario: 变更触发脏标记

- **WHEN** 文件变更事件发生且 Snapshot 模块已订阅
- **THEN** Snapshot MUST 标记工作区为"已变更"状态
