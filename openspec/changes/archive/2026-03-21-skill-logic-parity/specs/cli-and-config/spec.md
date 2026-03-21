## ADDED Requirements

### Requirement: Skills 配置结构

系统 MUST 支持 `x_opencode_go.skills` 配置项，包含以下子字段：
- `paths`（`[]string`，可选）：额外技能搜索路径，支持 `~/` 展开和相对路径
- `urls`（`[]string`，可选）：远程技能索引 URL

#### Scenario: 配置 skills.paths

- **WHEN** 配置 `skills: {paths: ["~/my-skills", "custom/skills"]}`
- **THEN** 系统 MUST 在标准路径之后搜索这些额外路径

#### Scenario: 配置 skills.urls

- **WHEN** 配置 `skills: {urls: ["https://example.com/.well-known/skills/"]}`
- **THEN** 系统 MUST 从该 URL 拉取远程技能索引

## MODIFIED Requirements

### Requirement: skills list 子命令

#### Scenario: 使用多路径发现

- **WHEN** 用户运行 `opencode-go skills list`
- **THEN** 系统 MUST 使用与 Engine 相同的多路径发现逻辑（`.cursor/skills`、`.agents/skills`、config.skills.paths 等）
- **AND** MUST NOT 仅扫描 `DataDir/skills`
- **AND** 输出 MUST 包含技能名称和描述
