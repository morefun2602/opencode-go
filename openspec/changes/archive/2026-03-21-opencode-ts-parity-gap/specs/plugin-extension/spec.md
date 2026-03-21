# Capability: plugin-extension

## ADDED Requirements

### Requirement: 首版不依赖 Go plugin 动态库

系统 MUST NOT 将 `plugin` 包动态加载作为**唯一**扩展机制；首版 MUST 支持至少一种跨平台扩展方式（例如编译期注册、子进程协议），与 `design.md` 决策一致。

#### Scenario: Windows 可加载扩展路径存在

- **WHEN** 用户在 Windows 上启用扩展
- **THEN** 系统 MUST 在不依赖 `plugin.Open` 的前提下完成扩展加载或文档化等价工作流

### Requirement: 扩展生命周期

扩展 MUST 具备明确的启动、停止与错误边界；扩展崩溃或返回错误时 MUST NOT 拖垮主进程（隔离策略以实现为准）。

#### Scenario: 扩展错误隔离

- **WHEN** 扩展在调用中 panic 或进程退出
- **THEN** 主服务 MUST 继续运行且 MUST 记录错误

### Requirement: 钩子与能力声明

扩展 MUST 通过显式接口或清单声明其提供的钩子（例如工具注册、配置片段）；未声明的能力 MUST NOT 被隐式假设存在。

#### Scenario: 未声明钩子不调用

- **WHEN** 扩展未注册某钩子
- **THEN** 主程序 MUST NOT 调用该钩子
