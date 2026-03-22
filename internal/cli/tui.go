package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/tui"
)

func newTUICmd() *cobra.Command {
	var (
		configFile string
		themeFlag  string
		model      string
		sessionID  string
		cont       bool
	)
	c := &cobra.Command{
		Use:   "tui",
		Short: "启动 Bubble Tea 终端界面",
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
			eng, st, err := wireEngine(cfg, log)
			if err != nil {
				return fmt.Errorf("initializing engine: %w", err)
			}
			defer st.Close()

			// 将 --model flag 应用到 Engine
			if model != "" {
				eng.SetModel(model)
			}

			theme := tui.ResolveTheme(themeFlag)

			m := tui.New(eng, st, cfg.WorkspaceID, theme)
			p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())

			eng.Confirm = func(name string, a map[string]any) (bool, error) {
				ch := make(chan bool, 1)
				p.Send(tui.NewConfirmRequest(name, a, ch))
				return <-ch, nil
			}

			m.SetProgram(p)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().StringVar(&themeFlag, "theme", "dark", "主题名称或JSON文件路径 ("+strings.Join(tui.ThemeNames(), "|")+")")
	c.Flags().StringVarP(&model, "model", "m", "", "模型 (provider/model)")
	c.Flags().StringVarP(&sessionID, "session", "s", "", "继续的会话 ID")
	c.Flags().BoolVarP(&cont, "continue", "c", false, "继续上次会话")

	return c
}
