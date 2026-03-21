## ADDED Requirements

### Requirement: 统一截断服务

系统 MUST 提供 `internal/truncate/` 模块，对工具输出进行统一截断。截断 MUST 支持按行数（默认 2000 行）和字节数（默认 50KB）两个维度，取先达到的上限。

#### Scenario: 按行数截断

- **WHEN** 工具输出为 3000 行
- **THEN** 截断服务 MUST 截取前 2000 行并附加截断提示

#### Scenario: 按字节数截断

- **WHEN** 工具输出为 80KB 但行数在限制内
- **THEN** 截断服务 MUST 按 50KB 截断并附加截断提示

#### Scenario: 未超限不截断

- **WHEN** 工具输出在行数和字节数限制之内
- **THEN** 截断服务 MUST 返回原始输出

### Requirement: 截断方向控制

截断服务 MUST 支持 `head`（保留前部）和 `tail`（保留尾部）两种方向。默认 MUST 为 `tail`。

#### Scenario: head 截断

- **WHEN** 方向为 head
- **THEN** MUST 保留输出的前 N 行/字节

#### Scenario: tail 截断

- **WHEN** 方向为 tail
- **THEN** MUST 保留输出的后 N 行/字节

### Requirement: Registry 层统一接入

`tools.Registry.Run()` 在工具执行返回后 MUST 自动对输出调用截断服务。各工具 MUST NOT 自行实现截断逻辑（移除现有各工具内的局部截断）。

#### Scenario: grep 工具输出截断

- **WHEN** grep 工具返回 5000 行结果
- **THEN** Registry MUST 在返回给 Engine 前截断到 2000 行

#### Scenario: bash 工具输出截断

- **WHEN** bash 工具返回 100KB 输出
- **THEN** Registry MUST 在返回给 Engine 前截断到 50KB

### Requirement: 截断结果元数据

截断服务 MUST 返回是否发生截断的标识。当发生截断时，输出末尾 MUST 附加截断提示信息，告知用户输出已被截断及原始大小。

#### Scenario: 截断提示信息

- **WHEN** 输出被截断
- **THEN** 截断后的输出 MUST 以 `\n...truncated (original: X lines / Y bytes)` 结尾
