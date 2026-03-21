package tools

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/morefun2602/opencode-go/internal/llm"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	MaxAttempts    int
	InitialDelay  time.Duration
	MaxDelay       time.Duration
	BackoffFactor float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// IsRetryable returns true if the error kind is retryable.
func IsRetryable(kind llm.Kind) bool {
	switch kind {
	case llm.RateLimit, llm.Timeout:
		return true
	case llm.Auth, llm.ContextOverflow:
		return false
	default:
		return false
	}
}

// ExtractRetryAfter parses retry-after from an error or HTTP response header.
// Returns 0 if not found.
func ExtractRetryAfter(err error) time.Duration {
	if re, ok := err.(*llm.RetryableError); ok && re.RetryAfter > 0 {
		return time.Duration(re.RetryAfter) * time.Second
	}
	s := err.Error()
	if idx := findRetryAfterInString(s); idx > 0 {
		return time.Duration(idx) * time.Second
	}
	return 0
}

func findRetryAfterInString(s string) int {
	for i := 0; i < len(s); i++ {
		if i+12 < len(s) && s[i:i+12] == "retry-after:" {
			j := i + 12
			for j < len(s) && s[j] == ' ' {
				j++
			}
			k := j
			for k < len(s) && s[k] >= '0' && s[k] <= '9' {
				k++
			}
			if k > j {
				v, _ := strconv.Atoi(s[j:k])
				return v
			}
		}
	}
	return 0
}

// ParseRetryAfterHeader parses the Retry-After HTTP header value.
func ParseRetryAfterHeader(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// ComputeDelay returns the delay for the given attempt (0-indexed).
func ComputeDelay(cfg RetryConfig, attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	return time.Duration(delay)
}

// Do executes fn with retry logic.
func Do(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		kind := llm.Classify(err)
		if !IsRetryable(kind) {
			return err
		}

		if attempt < cfg.MaxAttempts-1 {
			retryAfter := ExtractRetryAfter(err)
			delay := ComputeDelay(cfg, attempt, retryAfter)
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}
