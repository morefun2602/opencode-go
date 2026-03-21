# Design: Agent Mode & Runtime Parity

## D1: Permission Ruleset 数据结构

**决策**：新建 `internal/permission/permission.go`，定义独立的 Permission 包。

```go
type Action string
const (
    ActionAllow Action = "allow"
    ActionDeny  Action = "deny"
    ActionAsk   Action = "ask"
)

type Rule struct {
    Permission string // 工具权限标识（"edit", "bash", "*"）
    Pattern    string // 匹配模式（"*", "prefix*", exact）
    Action     Action
}

type Ruleset []Rule
```

编辑工具映射表：`edit`, `write`, `apply_patch`, `multiedit` → permission `"edit"`。其他工具 permission = 工具名。

`Disabled(toolNames []string, ruleset Ruleset) map[string]bool`：遍历 toolNames，对每个 tool 将其 name 映射为 permission，然后 findLast 匹配 ruleset 中的规则。Action 为 deny 时加入结果集。

`Evaluate(permission, target string, ruleset Ruleset) Action`：findLast 匹配，无匹配默认 allow。

`Merge(defaults, overrides Ruleset) Ruleset`：`append(defaults, overrides...)`。

Pattern 匹配逻辑：`"*"` 匹配所有；`"prefix*"` 用 `strings.HasPrefix`；其余精确匹配。

**理由**：TS 用 `findLast` + pattern matching，Go 等效实现。独立包避免循环依赖。

## D2: Agent 结构体增强

**决策**：修改 `internal/runtime/agent.go`，移除 `ToolPermission`，引入新字段。

```go
type Agent struct {
    Name        string
    Description string
    Prompt      string
    Mode        Mode
    Hidden      bool
    Subagent    bool    // true 表示可被 task 工具调用
    Steps       int     // 0 = 使用 Engine 默认
    Model       string  // "provider/model"，空 = 使用 Engine 默认
    Temperature *float64
    Permission  permission.Ruleset
}
```

新增 `Subagent bool` 字段：`explore` 和 `general` 为 true，`build`/`plan` 为 false，隐藏 Agent 为 false。控制 task 工具的可调用性。

内置 Agent 的 Permission 迁移：
- `AgentBuild`：nil（全部 allow）
- `AgentPlan`：`[{Permission: "edit", Pattern: "*", Action: "deny"}, {Permission: "execute", Pattern: "*", Action: "deny"}]`
  但保留 plan_enter/plan_exit 为 allow，通过追加 allow 规则实现
- `AgentExplore`：使用 Allow 列表模式 `[{"*","*","deny"}, {"read","*","allow"}, {"bash","*","allow"}, {"skill","*","allow"}, {"ls","*","allow"}]`
- `AgentGeneral`：`[{"todowrite","*","deny"}, {"todoread","*","deny"}]`
- 隐藏 Agent：`[{"*","*","deny"}]`

`ToolFilter` 函数改为调用 `permission.Disabled(toolNames, agent.Permission)` 后过滤。Mode Tags 过滤逻辑保留作为 fallback（当 Permission 为 nil 时使用 Mode.Tags）。

**理由**：渐进迁移，Permission 为 nil 时 fallback 到 Mode Tags 保证向后兼容。

## D3: 自定义 Agent 注册

**决策**：修改 `internal/runtime/agent.go`，将 `builtinAgents` 改为可修改的 `agentRegistry`（`sync.Map` 或初始化时的普通 map）。新增 `RegisterAgent(a Agent)` 函数。

在 `wireEngine` 中：

```go
for _, af := range cfg.Agents {
    a := Agent{
        Name:  af.Name,
        Model: af.Model,
        Mode:  ModeBuild, // 默认
    }
    if af.Temp > 0 {
        t := af.Temp
        a.Temperature = &t
    }
    if len(af.Tools) > 0 {
        // 转为 allow-only ruleset
        var rules permission.Ruleset
        rules = append(rules, permission.Rule{Permission: "*", Pattern: "*", Action: permission.ActionDeny})
        for _, t := range af.Tools {
            rules = append(rules, permission.Rule{Permission: t, Pattern: "*", Action: permission.ActionAllow})
        }
        a.Permission = rules
    }
    RegisterAgent(a)
}
```

`AgentFile` 配置结构增强：新增 `Steps int`、`Hidden bool`、`Prompt string`、`Mode string`、`Subagent bool` 字段。

**理由**：配置优先于代码定义，自定义 Agent 可覆盖内置。

## D4: Engine session-level Agent 管理

**决策**：将现有 `tool.GlobalModeSwitch`（`map[sessionID]string`）升级为 `AgentSwitch`（`map[sessionID]Agent`），移入 `internal/runtime/` 包。

```go
type AgentSwitch struct {
    mu     sync.RWMutex
    agents map[string]Agent // sessionID → Agent override
}

func (as *AgentSwitch) Get(sessionID string) (Agent, bool) { ... }
func (as *AgentSwitch) Set(sessionID string, agent Agent) { ... }
func (as *AgentSwitch) Delete(sessionID string)            { ... }
```

Engine 新增 `AgentSwitch *AgentSwitch` 字段。

`collectTools` 改为：

```go
func (e *Engine) collectToolsForSession(sessionID string) []llm.ToolDef {
    agent := e.activeAgent(sessionID)
    // ... 收集并过滤工具
}

func (e *Engine) activeAgent(sessionID string) Agent {
    if e.AgentSwitch != nil {
        if a, ok := e.AgentSwitch.Get(sessionID); ok {
            return a
        }
    }
    if e.Agent.Name != "" {
        return e.Agent
    }
    return AgentBuild
}
```

`plan_enter`/`plan_exit` 改为操作 `AgentSwitch`，设置/清除 session-level Agent override。

CompleteTurn 结束时 defer 清理 `AgentSwitch.Delete(sessionID)`。

**理由**：session-level override 允许运行时模式切换，同时保持 Engine 级默认 Agent 不变。

## D5: Engine Agent 感知 — 模型选择与轮数

**决策**：在 CompleteTurn/CompleteTurnStream 的 LLM 调用前增加模型选择逻辑。

```go
func (e *Engine) resolveProvider(agent Agent) (llm.Provider, string) {
    if agent.Model != "" && e.Router != nil {
        parts := strings.SplitN(agent.Model, "/", 2)
        if len(parts) == 2 {
            if prov, ok := e.Providers.Get(parts[0]); ok {
                return prov, parts[1]
            }
        }
    }
    return e.LLM, ""
}
```

轮数限制：

```go
func (e *Engine) maxRounds(agent Agent) int {
    if agent.Steps > 0 {
        return agent.Steps
    }
    if e.MaxToolRounds > 0 {
        return e.MaxToolRounds
    }
    return 25
}
```

Temperature：在调用 `prov.Chat` / `prov.ChatStream` 时，若 `agent.Temperature != nil`，通过 `llm.ChatOptions{Temperature: agent.Temperature}` 传入。

**理由**：Agent 级别的模型和参数覆盖是 TS 的核心特性，Go 必须支持。

## D6: task 工具 subagent_type 实现

**决策**：修改 `internal/tool/task.go`，`registerTask` 新增 `agentLookup func(name string) (Agent, bool)` 参数。

```go
func registerTask(reg, runner, lookup, workspaceID, maxDepth, agentLookup) {
    // ...
    Fn: func(ctx, args) (string, error) {
        agentType, _ := args["subagent_type"].(string)
        var agent Agent
        if agentType != "" {
            a, ok := agentLookup(agentType)
            if !ok {
                return "", fmt.Errorf("unknown agent: %q, available: %v", agentType, listSubagentNames())
            }
            if a.Hidden || !a.Subagent {
                return "", fmt.Errorf("agent %q cannot be used as subagent", agentType)
            }
            agent = a
        } else {
            agent, _ = agentLookup("general")
        }
        // 通过 context 传递 agent 给子 Engine
        sub := context.WithValue(ctx, agentKey{}, agent)
        result, err := runner.CompleteTurn(sub, workspaceID, sid, prompt)
    }
}
```

Engine 的 CompleteTurn 在入口处检查 context 中是否有 Agent override：

```go
if ctxAgent, ok := ctx.Value(agentKey{}).(Agent); ok {
    // 使用 ctxAgent 替代 e.Agent 作为本次执行的 Agent
}
```

**理由**：通过 context 传递 Agent 避免修改 TaskRunner 接口签名，保持向后兼容。

## D7: ToolFilter 迁移

**决策**：`ToolFilter` 改为同时支持 Permission Ruleset 和 Mode Tags。

```go
func ToolFilter(agent Agent, allTools []llm.ToolDef) []llm.ToolDef {
    if len(agent.Permission) > 0 {
        // 使用 Permission Ruleset
        names := make([]string, len(allTools))
        for i, t := range allTools { names[i] = t.Name }
        disabled := permission.Disabled(names, agent.Permission)
        var out []llm.ToolDef
        for _, t := range allTools {
            if !disabled[t.Name] { out = append(out, t) }
        }
        return out
    }
    // fallback: 使用 Mode Tags（向后兼容）
    // ... 现有逻辑
}
```

**理由**：渐进迁移，Permission Ruleset 优先，nil 时 fallback 到 Tags。

## D8: plan_enter/plan_exit 联动

**决策**：修改 `internal/tool/plan.go`，将 `GlobalModeSwitch` 替换为对 `AgentSwitch` 的操作。

plan_enter：`AgentSwitch.Set(sessionID, AgentPlan)`
plan_exit：`AgentSwitch.Delete(sessionID)` — 恢复到 Engine 默认 Agent

`plan.go` 的 `registerPlan` 函数新增 `agentSwitch *AgentSwitch` 参数（从 Engine 传入），替代 `GlobalModeSwitch`。

**理由**：去掉全局状态，改为 Engine 实例级管理。

## D9: Confirm 注入

**决策**：在 `wireEngine` 中根据运行模式注入 `Engine.Confirm`。

REPL 模式：

```go
engine.Confirm = func(name string, args map[string]any) (bool, error) {
    fmt.Printf("Tool %s wants to execute with args: %v\nAllow? [y/n]: ", name, args)
    var answer string
    fmt.Scanln(&answer)
    return strings.ToLower(answer) == "y", nil
}
```

HTTP API 模式：通过 Permission/Question 异步机制处理（已有 `tools.Questions`）。

**理由**：Engine 需要 Confirm 才能完整实现 permission ask 行为。

## D10: AgentFile 配置增强

**决策**：修改 `internal/config/config.go` 的 `AgentFile`：

```go
type AgentFile struct {
    Name        string   `json:"name"`
    Tools       []string `json:"tools"`
    Model       string   `json:"model"`
    Temp        float64  `json:"temperature"`
    Steps       int      `json:"steps"`
    Prompt      string   `json:"prompt"`
    Mode        string   `json:"mode"`
    Hidden      bool     `json:"hidden"`
    Subagent    bool     `json:"subagent"`
    Description string   `json:"description"`
}
```

**理由**：与 Agent 结构体字段对齐，支持完整的自定义 Agent 配置。
