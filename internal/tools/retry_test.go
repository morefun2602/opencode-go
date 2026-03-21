package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morefun2602/opencode-go/internal/llm"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		kind llm.Kind
		want bool
	}{
		{llm.RateLimit, true},
		{llm.Timeout, true},
		{llm.Auth, false},
		{llm.ContextOverflow, false},
		{llm.Other, false},
	}
	for _, tt := range tests {
		if got := IsRetryable(tt.kind); got != tt.want {
			t.Errorf("IsRetryable(%v) = %v, want %v", tt.kind, got, tt.want)
		}
	}
}

func TestComputeDelay(t *testing.T) {
	cfg := DefaultRetryConfig()
	d0 := ComputeDelay(cfg, 0, 0)
	if d0 != 1*time.Second {
		t.Errorf("attempt 0 delay = %v, want 1s", d0)
	}
	d1 := ComputeDelay(cfg, 1, 0)
	if d1 != 2*time.Second {
		t.Errorf("attempt 1 delay = %v, want 2s", d1)
	}

	ra := ComputeDelay(cfg, 0, 5*time.Second)
	if ra != 5*time.Second {
		t.Errorf("retry-after should take precedence, got %v", ra)
	}
}

func TestDo_Success(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1}
	calls := 0
	err := Do(context.Background(), cfg, func() error {
		calls++
		if calls < 2 {
			return fmt.Errorf("429 rate limit")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDo_AuthNotRetried(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1}
	calls := 0
	err := Do(context.Background(), cfg, func() error {
		calls++
		return fmt.Errorf("401 unauthorized")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("auth errors should not retry, got %d calls", calls)
	}
}

func TestExtractRetryAfter(t *testing.T) {
	d := ExtractRetryAfter(&llm.RetryableError{Err: fmt.Errorf("err"), RetryAfter: 5})
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}
