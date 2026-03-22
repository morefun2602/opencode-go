package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/store"
)

func newExportCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "export [sessionID]",
		Short: "将会话导出为 JSON",
		Args:  cobra.MaximumNArgs(1),
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

			var sessionID string
			if len(args) > 0 {
				sessionID = args[0]
			} else {
				sessions, _ := st.ListSessions(cmd.Context(), cfg.WorkspaceID, 1)
				if len(sessions) == 0 {
					return fmt.Errorf("没有可导出的会话")
				}
				sessionID = sessions[0].ID
			}

			msgs, err := st.ListMessages(cmd.Context(), cfg.WorkspaceID, sessionID, 0, 100000)
			if err != nil {
				return err
			}

			out := map[string]any{
				"session_id":   sessionID,
				"workspace_id": cfg.WorkspaceID,
				"messages":     msgs,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}
