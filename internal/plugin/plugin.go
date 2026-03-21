package plugin

import "context"

// Hook 编译期注册插件钩子（首版不加载 Go plugin 动态库）。
type Hook interface {
	Name() string
	OnStart(ctx context.Context) error
}

var hooks []Hook

// Register 在 init 或 main 中注册静态插件。
func Register(h Hook) {
	hooks = append(hooks, h)
}

// StartAll 启动已注册钩子。
func StartAll(ctx context.Context) error {
	for _, h := range hooks {
		if err := h.OnStart(ctx); err != nil {
			return err
		}
	}
	return nil
}
