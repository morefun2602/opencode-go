## MODIFIED Requirements

### Requirement: 工具写操作事件发布

write、edit、apply_patch 工具在成功执行文件写操作后 MUST 发布 `file.changed` 事件到 Bus，包含变更文件路径。工具 MUST 通过注入的 FileWatcher 引用调用 NotifyChange()。Wire.go MUST 创建 FileWatcher 实例并传递给工具注册函数。

#### Scenario: edit 工具触发事件

- **WHEN** edit 工具成功修改了文件 `main.go`
- **THEN** Bus MUST 收到 `file.changed` 事件，payload 包含文件路径 `main.go`

#### Scenario: write 工具触发事件

- **WHEN** write 工具成功写入新文件
- **THEN** Bus MUST 收到 `file.changed` 事件

#### Scenario: apply_patch 工具触发事件

- **WHEN** apply_patch 工具成功应用补丁修改多个文件
- **THEN** Bus MUST 收到每个被修改文件的 `file.changed` 事件

#### Scenario: FileWatcher 未注入时跳过

- **WHEN** 工具的 FileWatcher 引用为 nil
- **THEN** 工具 MUST 跳过事件发布，MUST NOT 报错
