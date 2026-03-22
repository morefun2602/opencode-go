package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/store"
)

func newStatsCmd() *cobra.Command {
	var (
		configFile string
		days       int
	)
	c := &cobra.Command{
		Use:   "stats",
		Short: "显示 token 使用统计",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
				return err
			}
			path := filepath.Join(cfg.DataDir, "sqlite.db")
			st, err := store.Open(path)
			if err != nil {
				return err
			}
			defer st.Close()

			prompt, completion, err := st.TotalUsage(cmd.Context(), cfg.WorkspaceID, days)
			if err != nil {
				return err
			}

			period := "all time"
			if days > 0 {
				period = fmt.Sprintf("last %d days", days)
			}

			fmt.Printf("Token usage (%s):\n", period)
			fmt.Printf("  Prompt tokens:     %d\n", prompt)
			fmt.Printf("  Completion tokens: %d\n", completion)
			fmt.Printf("  Total tokens:      %d\n", prompt+completion)
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().IntVar(&days, "days", 0, "最近 N 天的统计（默认全部）")
	return c
}
