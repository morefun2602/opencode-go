package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/skill"
)

func newSkillsCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "skills",
		Short: "技能管理",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "列出已发现技能（使用与 Agent 相同的多路径发现逻辑）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			log := slog.Default()
			paths := BuildSkillSearchPaths(cfg, log)
			sk, err := skill.DiscoverSkills(paths, log)
			if err != nil {
				return err
			}
			if len(sk) == 0 {
				fmt.Fprintln(os.Stdout, "No skills found.")
				return nil
			}
			for _, s := range sk {
				desc := s.Description
				if desc == "" {
					desc = "(no description)"
				}
				fmt.Fprintf(os.Stdout, "- %s: %s\n", s.Name, desc)
			}
			return nil
		},
	}
	list.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.AddCommand(list)
	return c
}
