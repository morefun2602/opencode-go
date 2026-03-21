# skill-discovery Specification

## Purpose

远程技能发现模块，通过 HTTP 从配置的 URL 拉取技能索引并下载技能文件到本地缓存，使 Agent 可使用远程发布的技能集。

## Requirements

### Requirement: 索引拉取

系统 MUST 实现 `Discovery.Pull(url string) ([]string, error)` 函数。该函数 MUST 请求 `{url}/index.json`，解析为技能索引结构（包含技能名称和文件列表），过滤出包含 `SKILL.md` 的技能条目。

#### Scenario: 有效索引

- **WHEN** 远程 `index.json` 包含 3 个技能条目且均有 `SKILL.md`
- **THEN** `Pull` MUST 返回 3 个本地缓存目录路径

#### Scenario: 索引中缺少 SKILL.md

- **WHEN** 某技能条目的 files 数组不包含 `SKILL.md`
- **THEN** 该条目 MUST 被跳过并记录 warning

#### Scenario: 索引不可达

- **WHEN** HTTP 请求失败
- **THEN** `Pull` MUST 返回空数组和错误日志，MUST NOT panic

### Requirement: 文件下载与缓存

系统 MUST 将远程技能文件下载到 `{CacheDir}/skills/{skill-name}/` 目录。已存在的文件 MUST NOT 重复下载。下载 MUST 支持并发（建议 skill 级别 4 并发、文件级别 8 并发）。

#### Scenario: 缓存命中

- **WHEN** 本地缓存已有 `foo/SKILL.md`
- **THEN** 系统 MUST NOT 重新下载该文件

#### Scenario: 部分下载失败

- **WHEN** 某技能的某个文件下载失败
- **THEN** 系统 MUST 记录错误日志
- **AND** 若 `SKILL.md` 未成功下载，该技能目录 MUST NOT 包含在返回结果中

### Requirement: 索引数据结构

索引 JSON 格式 MUST 为：
```json
{
  "skills": [
    { "name": "skill-name", "files": ["SKILL.md", "references/api.md"] }
  ]
}
```

#### Scenario: 解析索引

- **WHEN** 远程返回有效 JSON
- **THEN** 系统 MUST 正确解析 `skills` 数组中的 `name` 和 `files` 字段
