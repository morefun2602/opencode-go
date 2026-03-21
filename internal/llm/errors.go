package llm

import (
	"context"
	"errors"
	"regexp"
	"strconv"
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

var retryAfterRegex = regexp.MustCompile(`retry[_-]?after[:\s=]*(\d+)`)

// Classify 将错误映射为 Kind（启发式）。
// 对于 RateLimit 错误，尝试提取 retry-after 并包装为 RetryableError。
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

// ClassifyWithRetry classifies and wraps RateLimit errors with retry-after metadata.
func ClassifyWithRetry(err error) (Kind, error) {
	k := Classify(err)
	if k == RateLimit {
		retryAfter := extractRetryAfter(err.Error())
		if retryAfter > 0 {
			return k, &RetryableError{Err: err, Kind: k, RetryAfter: retryAfter}
		}
	}
	return k, err
}

func extractRetryAfter(s string) int {
	m := retryAfterRegex.FindStringSubmatch(strings.ToLower(s))
	if len(m) >= 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return v
		}
	}
	return 0
}
