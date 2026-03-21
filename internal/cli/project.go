package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "project",
		Short: "项目级命令（占位；后续对齐上游 project 行为）",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "project: not implemented yet\n")
			return nil
		},
	}
}
