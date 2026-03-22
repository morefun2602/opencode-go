package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuthInvalidateToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	p := NewOAuthProvider("srv", OAuthConfig{}, nil)
	if err := p.saveToken(&OAuthToken{
		AccessToken: "abc",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("saveToken failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".opencode", "mcp-auth", "srv.json")); err != nil {
		t.Fatalf("token file should exist: %v", err)
	}
	if err := p.InvalidateToken(); err != nil {
		t.Fatalf("invalidate failed: %v", err)
	}
	if _, err := p.loadToken(); err == nil {
		t.Fatal("expected loadToken to fail after invalidate")
	}
}
