package cli

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
)

func newReplCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "repl",
		Short: "交互式 agent 对话（读取 stdin，使用与 serve 相同的配置加载）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return withCode(2, err)
			}
			log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
			eng, st, err := wireEngine(cfg, log)
			if err != nil {
				return err
			}
			defer st.Close()

			eng.Confirm = func(name string, a map[string]any) (bool, error) {
				fmt.Fprintf(os.Stderr, "\n[tool] %s %v\nallow? (y/n): ", name, a)
				sc := bufio.NewScanner(os.Stdin)
				if sc.Scan() {
					return strings.TrimSpace(strings.ToLower(sc.Text())) == "y", nil
				}
				return false, sc.Err()
			}

			ctx := cmd.Context()
			sid, err := eng.Store.CreateSession(ctx, cfg.WorkspaceID)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "session %s (type lines, Ctrl-D to exit)\n", sid)
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				line := sc.Text()
				out, err := eng.CompleteTurn(ctx, cfg.WorkspaceID, sid, line)
				if err != nil {
					fmt.Fprintf(os.Stderr, "err: %v\n", err)
					continue
				}
				fmt.Fprintln(os.Stdout, out)
			}
			return sc.Err()
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}
