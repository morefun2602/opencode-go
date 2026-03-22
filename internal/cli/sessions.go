package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/store"
)

func newSessionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "sessions",
		Aliases: []string{"session"},
		Short:   "会话管理（本地 SQLite）",
	}
	c.AddCommand(newSessionListCmd())
	c.AddCommand(newSessionDeleteCmd())
	return c
}

func newSessionListCmd() *cobra.Command {
	var (
		configFile string
		maxCount   int
		formatFlag string
	)
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "列出当前工作区会话",
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
			rows, err := st.ListSessions(cmd.Context(), cfg.WorkspaceID, maxCount)
			if err != nil {
				return err
			}
			if formatFlag == "json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}
			for _, r := range rows {
				title := r.Title
				if title == "" {
					title = "(untitled)"
				}
				ts := time.Unix(r.CreatedAt, 0).Format("2006-01-02 15:04")
				fmt.Fprintf(os.Stdout, "%-34s  %-20s  %s\n", r.ID, title, ts)
			}
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().IntVarP(&maxCount, "max-count", "n", 50, "最大数量")
	c.Flags().StringVar(&formatFlag, "format", "table", "输出格式 (table|json)")
	return c
}

func newSessionDeleteCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "delete <sessionID>",
		Short: "删除会话",
		Args:  cobra.ExactArgs(1),
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

			if err := st.DeleteSession(cmd.Context(), cfg.WorkspaceID, args[0]); err != nil {
				return err
			}
			fmt.Printf("会话 %s 已删除。\n", args[0])
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}
