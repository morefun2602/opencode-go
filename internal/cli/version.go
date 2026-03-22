package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	CommitSHA = "unknown"
	BuildDate = "unknown"
)

func newVersionCmd() *cobra.Command {
	var short bool
	c := &cobra.Command{
		Use:   "version",
		Short: "显示版本号",
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Println(Version)
				return
			}
			fmt.Printf("opencode %s (commit: %s, built: %s)\n", Version, CommitSHA, BuildDate)
		},
	}
	c.Flags().BoolVarP(&short, "short", "s", false, "仅输出版本号")
	return c
}
