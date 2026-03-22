# Agent Core / ReAct Parity Checklist (opencode-go vs opencode)

该清单用于追踪 `opencode-go` 在 Agent Core、Agent Runtime、ReAct 循环领域与上游的能力差异收敛进度。

## Legend

- `DONE`: 已实现并有回归测试
- `PARTIAL`: 已有能力，但语义/链路/测试不完整
- `TODO`: 尚未实现

## Agent Core

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| Agent 定义（name/mode/prompt/permission） | DONE | 保持 | `internal/runtime/agent.go` |
| activeAgent 优先级（subagent > switch > default） | DONE | 补回归测试 | `internal/runtime/engine.go` |
| plan mode session 级切换 | DONE | 保持 | `internal/runtime/agent_switch.go`, `internal/tool/plan.go` |

## ReAct Runtime

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| tool_calls 循环与终止条件 | DONE | 保持 | `internal/runtime/engine.go` |
| ask 在无 Confirm 时语义一致（默认拒绝） | PARTIAL | DONE | `internal/runtime/engine.go` |
| doom loop 保护 | DONE | 配置化 + 观测增强 | `internal/runtime/engine.go`, `internal/config/config.go` |
| DoomLoopWindow 配置化 | TODO | DONE | `internal/config/config.go`, `internal/cli/wire.go` |
| max rounds 与 maxStepsWarning | DONE | 边界测试增强 | `internal/runtime/engine.go` |

## Context / Compaction / Summary

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| ContextOverflow 触发压缩 | DONE | provider 一致性强化 | `internal/runtime/engine.go` |
| auto compaction | DONE | fallback 与事件增强 | `internal/runtime/engine.go` |
| SessionSummary step 生命周期接入 | TODO | DONE | `internal/tools/summary.go`, `internal/runtime/engine.go` |

## Observability

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| session/tool/retry 事件 | PARTIAL | 保持 | `internal/runtime/engine.go` |
| round_start/round_finish/blocked/compact 事件 | TODO | DONE | `internal/runtime/engine.go` |

## Tests / Spec

| Capability | Current | Target | Primary Files |
| --- | --- | --- | --- |
| ReAct 关键路径单测 | PARTIAL | DONE | `internal/runtime/engine_test.go` |
| mode/subagent 优先级回归矩阵 | TODO | DONE | `internal/runtime/engine_test.go`, `internal/tool/task_test.go` |
| ask 语义一致性条款收敛 | PARTIAL | DONE | `openspec/specs/agent-runtime/spec.md`, `openspec/specs/tool-permissions/spec.md` |
