package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
)

func newProvidersCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "providers",
		Aliases: []string{"auth"},
		Short:   "管理 AI 提供商与凭据",
	}
	c.AddCommand(newProvidersListCmd())
	c.AddCommand(newProvidersLoginCmd())
	c.AddCommand(newProvidersLogoutCmd())
	return c
}

func newProvidersListCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "列出已配置的提供商",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return err
			}
			if len(cfg.Providers) == 0 {
				fmt.Println("未配置任何提供商。在 opencode.json 中添加 provider 配置，或设置 OPENAI_API_KEY / ANTHROPIC_API_KEY 环境变量。")
				return nil
			}
			for name, p := range cfg.Providers {
				keyHint := "(未设置)"
				if p.APIKey != "" {
					if len(p.APIKey) >= 10 {
						keyHint = p.APIKey[:4] + "..." + p.APIKey[len(p.APIKey)-4:]
					} else {
						keyHint = "****"
					}
				}
				provType := p.Type
				if provType == "" {
					provType = name
				}
				fmt.Printf("%-20s type=%-12s key=%s\n", name, provType, keyHint)
				if p.BaseURL != "" {
					fmt.Printf("  baseURL: %s\n", p.BaseURL)
				}
			}
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}

func newProvidersLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login [provider]",
		Short: "登录提供商（设置 API key）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("提示：请在 opencode.json 的 providers 段或环境变量中设置 API key。")
			fmt.Println("  例如: ANTHROPIC_API_KEY, OPENAI_API_KEY")
			return nil
		},
	}
}

func newProvidersLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "登出提供商",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("提示：移除 opencode.json 中的 providers 配置或取消环境变量即可登出。")
			return nil
		},
	}
}
