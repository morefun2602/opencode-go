package llm

import (
	"fmt"
	"testing"
	"time"
)

func TestRetryDelayExponential(t *testing.T) {
	d0 := RetryDelay(0, fmt.Errorf("rate limit"))
	if d0 != 1*time.Second {
		t.Errorf("attempt 0: want 1s, got %v", d0)
	}

	d1 := RetryDelay(1, fmt.Errorf("rate limit"))
	if d1 != 2*time.Second {
		t.Errorf("attempt 1: want 2s, got %v", d1)
	}

	d2 := RetryDelay(2, fmt.Errorf("rate limit"))
	if d2 != 4*time.Second {
		t.Errorf("attempt 2: want 4s, got %v", d2)
	}
}

func TestRetryDelayMax(t *testing.T) {
	d := RetryDelay(10, fmt.Errorf("rate limit"))
	if d > 30*time.Second {
		t.Errorf("should cap at 30s, got %v", d)
	}
}

func TestRetryDelayWithRetryAfter(t *testing.T) {
	err := &RetryableError{
		Err:        fmt.Errorf("rate limit"),
		Kind:       RateLimit,
		RetryAfter: 5,
	}
	d := RetryDelay(0, err)
	if d != 5*time.Second {
		t.Errorf("want 5s from retry-after, got %v", d)
	}
}

func TestRetryDelayRetryAfterCapped(t *testing.T) {
	err := &RetryableError{
		Err:        fmt.Errorf("rate limit"),
		Kind:       RateLimit,
		RetryAfter: 60,
	}
	d := RetryDelay(0, err)
	if d != 30*time.Second {
		t.Errorf("should cap at 30s, got %v", d)
	}
}

func TestClassifyWithRetry(t *testing.T) {
	k, wrapped := ClassifyWithRetry(fmt.Errorf("429 rate limit retry-after: 10"))
	if k != RateLimit {
		t.Errorf("want RateLimit, got %v", k)
	}
	re, ok := wrapped.(*RetryableError)
	if !ok {
		t.Fatal("expected RetryableError")
	}
	if re.RetryAfter != 10 {
		t.Errorf("want RetryAfter=10, got %d", re.RetryAfter)
	}
}

func TestClassifyWithRetryNoHeader(t *testing.T) {
	k, wrapped := ClassifyWithRetry(fmt.Errorf("429 too many requests"))
	if k != RateLimit {
		t.Errorf("want RateLimit, got %v", k)
	}
	if _, ok := wrapped.(*RetryableError); ok {
		t.Error("should not wrap when no retry-after")
	}
}

func TestExtractRetryAfter(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"retry-after: 5", 5},
		{"retry_after=10", 10},
		{"RetryAfter: 3", 3},
		{"no info here", 0},
	}
	for _, tt := range tests {
		got := extractRetryAfter(tt.input)
		if got != tt.want {
			t.Errorf("extractRetryAfter(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
