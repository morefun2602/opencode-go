package tool

import "fmt"

// ErrUnknown 未知工具名（内置与 MCP 均未解析）。
type ErrUnknown struct {
	Name string
}

func (e *ErrUnknown) Error() string {
	return fmt.Sprintf("unknown tool: %q", e.Name)
}
