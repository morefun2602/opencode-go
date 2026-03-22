package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
	rtAgent "github.com/morefun2602/opencode-go/internal/runtime"
)

func newDebugCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "debug",
		Short: "调试与排错工具",
	}
	c.AddCommand(newDebugConfigCmd())
	c.AddCommand(newDebugPathsCmd())
	c.AddCommand(newDebugAgentCmd())
	return c
}

func newDebugConfigCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "config",
		Short: "显示解析后的配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(cfg)
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}

func newDebugPathsCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "paths",
		Short: "显示全局路径",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return err
			}
			home, _ := os.UserHomeDir()
			cacheDir, _ := os.UserCacheDir()
			configDir, _ := os.UserConfigDir()

			abs := func(p string) string {
				a, err := filepath.Abs(p)
				if err != nil {
					return p
				}
				return a
			}

			fmt.Printf("data:      %s\n", abs(cfg.DataDir))
			fmt.Printf("config:    %s\n", configDir)
			fmt.Printf("cache:     %s\n", cacheDir)
			fmt.Printf("home:      %s\n", home)
			fmt.Printf("workspace: %s\n", abs(cfg.WorkspaceRoot))
			fmt.Printf("go:        %s\n", runtime.GOROOT())
			if cfg.ConfigPath != "" {
				fmt.Printf("config-file: %s\n", abs(cfg.ConfigPath))
			}
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}

func newDebugAgentCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "agent [name]",
		Short: "显示 agent 配置详情",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				agents := rtAgent.ListAgents()
				for _, a := range agents {
					sub := ""
					if a.Subagent {
						sub = " [subagent]"
					}
					fmt.Printf("%-15s %s%s\n", a.Name, a.Description, sub)
				}
				return nil
			}
			a, ok := rtAgent.GetAgent(args[0])
			if !ok {
				return fmt.Errorf("agent %q not found", args[0])
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"name":        a.Name,
				"description": a.Description,
				"mode":        a.Mode.Name,
				"subagent":    a.Subagent,
				"hidden":      a.Hidden,
				"steps":       a.Steps,
				"model":       a.Model,
				"temperature": a.Temperature,
			})
		},
	}
	return c
}
