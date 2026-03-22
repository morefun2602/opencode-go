package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
)

func newRunCmd() *cobra.Command {
	var (
		configFile string
		model      string
		sessionID  string
		cont       bool
		formatFlag string
	)
	c := &cobra.Command{
		Use:   "run [message...]",
		Short: "用消息运行 opencode（无头模式）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
				return err
			}

			log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
			eng, st, err := wireEngine(cfg, log)
			if err != nil {
				return err
			}
			defer st.Close()

			_ = model // TODO: override model via --model

			eng.Confirm = func(name string, a map[string]any) (bool, error) {
				return true, nil
			}

			ctx := context.Background()
			sid := sessionID
			if sid == "" && cont {
				sessions, _ := st.ListSessions(ctx, cfg.WorkspaceID, 1)
				if len(sessions) > 0 {
					sid = sessions[0].ID
				}
			}
			if sid == "" {
				sid, err = st.CreateSession(ctx, cfg.WorkspaceID)
				if err != nil {
					return err
				}
			}

			text := strings.Join(args, " ")
			reply, err := eng.CompleteTurn(ctx, cfg.WorkspaceID, sid, text)
			if err != nil {
				return err
			}

			if formatFlag == "json" {
				out := map[string]string{"session_id": sid, "reply": reply}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			fmt.Println(reply)
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().StringVarP(&model, "model", "m", "", "模型 (provider/model)")
	c.Flags().StringVarP(&sessionID, "session", "s", "", "继续的会话 ID")
	c.Flags().BoolVarP(&cont, "continue", "c", false, "继续上次会话")
	c.Flags().StringVar(&formatFlag, "format", "default", "输出格式 (default|json)")
	return c
}
