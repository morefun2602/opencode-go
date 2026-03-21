package llm

import (
	"context"
	"errors"
	"strings"
)

// Kind 提供商/调用错误分类，供编排重试策略使用。
type Kind int

const (
	Other Kind = iota
	Timeout
	RateLimit
	Auth
)

// Classify 将错误映射为 Kind（启发式）。
func Classify(err error) Kind {
	if err == nil {
		return Other
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return Timeout
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "timeout") || strings.Contains(s, "deadline") {
		return Timeout
	}
	if strings.Contains(s, "429") || strings.Contains(s, "rate") {
		return RateLimit
	}
	if strings.Contains(s, "401") || strings.Contains(s, "403") || strings.Contains(s, "unauthorized") || strings.Contains(s, "auth") {
		return Auth
	}
	return Other
}
