# 插件（编译期注册）

首版通过 `internal/plugin` 的 `Register` 在 **编译期** 挂载钩子，**不**依赖 Go `plugin` 动态库（Windows 不支持且发布约束多）。

- 在 `init` 或 `main` 中调用 `plugin.Register`。
- `OnStart` 在 `serve` / `repl` 等入口通过 `plugin.StartAll` 触发。
- 若未来引入子进程 JSON-RPC 等扩展，应保持与 `design.md` §7 一致并更新本文档。
