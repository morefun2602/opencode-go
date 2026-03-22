package cli

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"
)

func newModelsCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "models [provider]",
		Short: "列出可用模型",
		Args:  cobra.MaximumNArgs(1),
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

			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}

			modelMap := eng.ListModels()
			if len(modelMap) == 0 {
				return nil
			}
			providers := make([]string, 0, len(modelMap))
			for name := range modelMap {
				providers = append(providers, name)
			}
			sort.Strings(providers)
			for _, name := range providers {
				if filter != "" && name != filter {
					continue
				}
				models := append([]string(nil), modelMap[name]...)
				sort.Strings(models)
				for _, model := range models {
					fmt.Printf("%-20s %s\n", name, model)
				}
			}

			dm := eng.Router.DefaultModel()
			if dm.ProviderID != "" || dm.ModelID != "" {
				fmt.Printf("\ndefault: %s/%s\n", dm.ProviderID, dm.ModelID)
			}
			sm := eng.Router.SmallModel()
			if sm.ProviderID != "" || sm.ModelID != "" {
				fmt.Printf("small:   %s/%s\n", sm.ProviderID, sm.ModelID)
			}

			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}
