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

func newImportCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "import <file>",
		Short: "从 JSON 文件导入会话",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
				return err
			}

			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			var payload struct {
				SessionID   string             `json:"session_id"`
				WorkspaceID string             `json:"workspace_id"`
				Messages    []store.MessageRow `json:"messages"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return fmt.Errorf("parse JSON: %w", err)
			}

			path := filepath.Join(cfg.DataDir, "sqlite.db")
			st, err := store.Open(path)
			if err != nil {
				return err
			}
			defer st.Close()

			ws := cfg.WorkspaceID
			if payload.WorkspaceID != "" {
				ws = payload.WorkspaceID
			}

			sid, err := st.CreateSession(cmd.Context(), ws)
			if err != nil {
				return err
			}

			if len(payload.Messages) > 0 {
				if err := st.AppendMessages(cmd.Context(), ws, sid, payload.Messages); err != nil {
					return err
				}
			}

			fmt.Printf("导入成功。新会话 ID: %s (%d 条消息)\n", sid, len(payload.Messages))
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}
