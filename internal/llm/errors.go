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
	ContextOverflow
)

// RetryableError wraps an error with retry metadata.
type RetryableError struct {
	Err        error
	Kind       Kind
	RetryAfter int // seconds, 0 if not specified
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

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
	if strings.Contains(s, "context_length") || strings.Contains(s, "context window") ||
		strings.Contains(s, "token limit") || strings.Contains(s, "max_tokens") ||
		strings.Contains(s, "too many tokens") || strings.Contains(s, "context_too_long") {
		return ContextOverflow
	}
	return Other
}
