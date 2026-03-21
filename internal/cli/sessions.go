package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/store"
)

func newSessionsCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "sessions",
		Short: "会话（本地 SQLite）",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "列出当前工作区会话",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			path := filepath.Join(cfg.DataDir, "sqlite.db")
			st, err := store.Open(path)
			if err != nil {
				return err
			}
			defer st.Close()
			rows, err := st.ListSessions(cmd.Context(), cfg.WorkspaceID, 200)
			if err != nil {
				return err
			}
			for _, r := range rows {
				_, _ = fmt.Fprintf(os.Stdout, "%s\t%d\n", r.ID, r.CreatedAt)
			}
			return nil
		},
	}
	list.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.AddCommand(list)
	return c
}
