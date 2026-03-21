package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newToolsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tools",
		Short: "内置工具名列表",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "列出内置工具（与 tool.RegisterBuiltin 一致）",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := []string{"read", "write", "glob", "grep", "bash"}
			for _, n := range names {
				_, _ = fmt.Fprintln(os.Stdout, n)
			}
			return nil
		},
	}
	c.AddCommand(list)
	return c
}
