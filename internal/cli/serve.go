package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/morefun2602/opencode-go/internal/acp"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/server"
)

func newServeCmd() *cobra.Command {
	var (
		configFile string
		listen     string
		token      string
		dataDir    string
		workspace  string
	)
	c := &cobra.Command{
		Use:   "serve",
		Short: "启动 HTTP API 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			overlay := &config.Config{}
			if listen != "" {
				overlay.Listen = listen
			}
			if token != "" {
				overlay.AuthToken = token
			}
			if dataDir != "" {
				overlay.DataDir = dataDir
			}
			if workspace != "" {
				overlay.WorkspaceID = workspace
			}
			cfg, err := config.Load(configFile, overlay)
			if err != nil {
				return withCode(2, err)
			}
			if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
				return err
			}

			log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
			eng, st, err := wireEngine(cfg, log)
			if err != nil {
				return err
			}
			defer st.Close()

			h := &server.Handler{
				Cfg: cfg, Engine: eng,
				ACP: &acp.Handler{Workspace: cfg.WorkspaceID, Store: st},
				Bus: eng.Bus,
			}
			mux := server.NewMux(h)
			handler := server.AuthMiddleware(cfg.AuthToken, server.RequestID(mux))
			srv := &http.Server{
				Addr:              cfg.Listen,
				Handler:           handler,
				ReadHeaderTimeout: 10 * time.Second,
			}
			errCh := make(chan error, 1)
			go func() {
				log.Info("http_listen", "addr", cfg.Listen, "upstream_compat_ref", cfg.UpstreamCompatRef)
				errCh <- srv.ListenAndServe()
			}()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-sigCh:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_ = srv.Shutdown(ctx)
				<-errCh
				if sig == syscall.SIGINT {
					return codeErr{code: 130, err: fmt.Errorf("SIGINT")}
				}
				return nil
			case err := <-errCh:
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					return err
				}
				return nil
			}
		},
	}
	c.Flags().StringVar(&configFile, "config", "", "配置文件路径（默认 opencode.json 或 OPENCODE_CONFIG）")
	c.Flags().StringVar(&listen, "listen", "", "监听地址（覆盖 OPENCODE_SERVER_LISTEN）")
	c.Flags().StringVar(&token, "token", "", "鉴权 token（覆盖 OPENCODE_AUTH_TOKEN）")
	c.Flags().StringVar(&dataDir, "data-dir", "", "数据目录（覆盖 OPENCODE_DATA_DIR）")
	c.Flags().StringVar(&workspace, "workspace", "", "工作区 ID（覆盖 OPENCODE_WORKSPACE_ID）")
	return c
}
