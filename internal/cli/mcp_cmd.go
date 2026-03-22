package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
)

func newMCPCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "mcp",
		Short: "管理 MCP 服务",
	}
	c.AddCommand(newMCPListCmd())
	c.AddCommand(newMCPAddCmd())
	return c
}

func newMCPListCmd() *cobra.Command {
	var configFile string
	var jsonFlag bool
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "列出已配置的 MCP 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return err
			}
			if len(cfg.MCPServers) == 0 {
				fmt.Println("未配置任何 MCP 服务。")
				return nil
			}
			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cfg.MCPServers)
			}
			for _, s := range cfg.MCPServers {
				transport := s.InferTransport()
				endpoint := s.Command
				if s.URL != "" {
					endpoint = s.URL
				}
				fmt.Printf("%-20s %-8s %s\n", s.Name, transport, endpoint)
			}
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().BoolVar(&jsonFlag, "json", false, "JSON 输出")
	return c
}

func newMCPAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "添加 MCP 服务（提示编辑配置文件）",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("在 opencode.json 的 mcp 字段中添加 MCP 服务配置：")
			fmt.Println(`  "mcp": {`)
			fmt.Println(`    "my-server": {`)
			fmt.Println(`      "type": "local",`)
			fmt.Println(`      "command": ["npx", "-y", "@my/mcp-server"]`)
			fmt.Println(`    }`)
			fmt.Println(`  }`)
			return nil
		},
	}
}
