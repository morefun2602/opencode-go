package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/runtime"
)

func newAgentCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "agent",
		Short: "管理 agents",
	}
	c.AddCommand(newAgentListCmd())
	return c
}

func newAgentListCmd() *cobra.Command {
	var jsonFlag bool
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "列出已注册的 agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			agents := runtime.ListAgents()
			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				var out []map[string]any
				for _, a := range agents {
					out = append(out, map[string]any{
						"name":        a.Name,
						"description": a.Description,
						"mode":        a.Mode.Name,
						"subagent":    a.Subagent,
					})
				}
				return enc.Encode(out)
			}
			for _, a := range agents {
				sub := ""
				if a.Subagent {
					sub = " [subagent]"
				}
				desc := a.Description
				if desc == "" {
					desc = a.Mode.Name
				}
				fmt.Printf("%-15s %-10s %s%s\n", a.Name, a.Mode.Name, desc, sub)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonFlag, "json", false, "JSON 输出")
	return c
}
