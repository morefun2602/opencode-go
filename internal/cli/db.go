package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/config"

	_ "modernc.org/sqlite"
)

func newDBCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "db",
		Short: "数据库工具",
	}
	c.AddCommand(newDBPathCmd())
	c.AddCommand(newDBQueryCmd())
	return c
}

func newDBPathCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "path",
		Short: "显示数据库路径",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return err
			}
			abs, _ := filepath.Abs(filepath.Join(cfg.DataDir, "sqlite.db"))
			fmt.Println(abs)
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	return c
}

func newDBQueryCmd() *cobra.Command {
	var (
		configFile string
		formatFlag string
	)
	c := &cobra.Command{
		Use:   "query [sql]",
		Short: "执行 SQL 查询",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile, nil)
			if err != nil {
				return err
			}
			dbPath := filepath.Join(cfg.DataDir, "sqlite.db")
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			query := args[0]
			upper := strings.ToUpper(strings.TrimSpace(query))
			if strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "PRAGMA") {
				return runSelectQuery(db, query, formatFlag)
			}
			result, err := db.Exec(query)
			if err != nil {
				return err
			}
			affected, _ := result.RowsAffected()
			fmt.Printf("OK, %d rows affected\n", affected)
			return nil
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径")
	c.Flags().StringVar(&formatFlag, "format", "tsv", "输出格式 (json|tsv)")
	return c
}

func runSelectQuery(db *sql.DB, query, format string) error {
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	if format == "json" {
		var results []map[string]any
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			row := make(map[string]any)
			for i, col := range cols {
				row[col] = vals[i]
			}
			results = append(results, row)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Println(strings.Join(cols, "\t"))
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		strs := make([]string, len(vals))
		for i, v := range vals {
			strs[i] = fmt.Sprintf("%v", v)
		}
		fmt.Println(strings.Join(strs, "\t"))
	}
	return nil
}
