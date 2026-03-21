# 发行说明模板

## 版本 x.y.z（YYYY-MM-DD）

### 破坏性变更（BREAKING）

- （无则写「无」）
- 配置键：列出删除或重命名的 `opencode.json` 键。
- HTTP：列出删除或变更语义的 `/v1/...` 路径。
- 数据库：若 `PRAGMA user_version` / 迁移版本有不可逆变更，说明备份与迁移步骤。

### 安全

- 默认仅 **loopback** 监听；若需 `0.0.0.0` / `::` 监听，必须配置 `server.auth_token` 并在文档中提示暴露面。

### 构建产物

- 多平台二进制 / 容器镜像构建与 tag 策略见 [README.md](../README.md) 与 CI 工作流。
