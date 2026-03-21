package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/skill"
)

func newSkillsCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "skills",
		Short: "技能（data_dir/skills 下 *.md）",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "列出已发现技能",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			dir := filepath.Join(cfg.DataDir, "skills")
			sk, err := skill.LoadDir(dir)
			if err != nil {
				return err
			}
			for _, s := range sk {
				_, _ = fmt.Fprintln(os.Stdout, s.Name)
			}
			return nil
		},
	}
	list.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.AddCommand(list)
	return c
}
