package cli

import (
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/tui"
)

func newTUICmd() *cobra.Command {
	var (
		configFile string
		themeFlag  string
	)
	c := &cobra.Command{
		Use:   "tui",
		Short: "启动 Bubble Tea 终端界面",
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
				return true, nil
			}

			theme := tui.Dark
			if themeFlag == "light" {
				theme = tui.Light
			}

			m := tui.New(eng, st, cfg.WorkspaceID, theme)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().StringVar(&themeFlag, "theme", "dark", "主题 (dark|light)")
	return c
}
