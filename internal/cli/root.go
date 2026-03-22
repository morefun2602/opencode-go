package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Run 执行根命令并返回进程退出码。
func Run() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return exitFromErr(err)
	}
	return 0
}

func newRootCmd() *cobra.Command {
	c := &cobra.Command{
		Use:           "opencode",
		Short:         "OpenCode Go 实现",
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(newServeCmd())
	c.AddCommand(newSessionsCmd())
	c.AddCommand(newToolsCmd())
	c.AddCommand(newSkillsCmd())
	c.AddCommand(newProjectCmd())
	c.AddCommand(newReplCmd())
	c.AddCommand(newTUICmd())
	c.AddCommand(newVersionCmd())
	c.AddCommand(newRunCmd())
	c.AddCommand(newDebugCmd())
	c.AddCommand(newModelsCmd())
	c.AddCommand(newProvidersCmd())
	c.AddCommand(newAgentCmd())
	c.AddCommand(newMCPCmd())
	c.AddCommand(newExportCmd())
	c.AddCommand(newImportCmd())
	c.AddCommand(newStatsCmd())
	c.AddCommand(newDBCmd())
	return c
}
