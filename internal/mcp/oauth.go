package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// OAuthConfig holds OAuth client configuration for an MCP server.
type OAuthConfig struct {
	AuthorizationURL string `json:"authorization_url"`
	TokenURL         string `json:"token_url"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	Scopes           string `json:"scopes"`
	RedirectPort     int    `json:"redirect_port"`
}

// OAuthToken represents stored OAuth tokens.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired returns true if the access token has expired.
func (t *OAuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// OAuthProvider manages OAuth authentication for MCP servers.
type OAuthProvider struct {
	config   OAuthConfig
	serverID string
	log      *slog.Logger
	mu       sync.Mutex
}

// NewOAuthProvider creates a new OAuth provider.
func NewOAuthProvider(serverID string, config OAuthConfig, log *slog.Logger) *OAuthProvider {
	return &OAuthProvider{
		config:   config,
		serverID: serverID,
		log:      log,
	}
}

// GetToken returns a valid OAuth token, refreshing or re-authenticating as needed.
func (p *OAuthProvider) GetToken(ctx context.Context) (*OAuthToken, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, err := p.loadToken()
	if err == nil && !token.IsExpired() {
		return token, nil
	}

	if err == nil && token.RefreshToken != "" {
		refreshed, refreshErr := p.refreshToken(ctx, token.RefreshToken)
		if refreshErr == nil {
			_ = p.saveToken(refreshed)
			return refreshed, nil
		}
		if p.log != nil {
			p.log.Warn("oauth_refresh_failed", "server", p.serverID, "err", refreshErr)
		}
	}

	newToken, err := p.authorize(ctx)
	if err != nil {
		return nil, fmt.Errorf("oauth authorize: %w", err)
	}
	_ = p.saveToken(newToken)
	return newToken, nil
}

func (p *OAuthProvider) authorize(ctx context.Context) (*OAuthToken, error) {
	port := p.config.RedirectPort
	if port == 0 {
		port = 18923
	}
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code in callback")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		fmt.Fprint(w, "<html><body><h1>Authorization successful!</h1><p>You can close this window.</p></body></html>")
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()
	defer func() {
		_ = srv.Close()
	}()

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s",
		p.config.AuthorizationURL,
		url.QueryEscape(p.config.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(p.config.Scopes))

	if p.log != nil {
		p.log.Info("oauth_authorize", "url", authURL, "server", p.serverID)
	}

	timeout := 5 * time.Minute
	select {
	case code := <-codeCh:
		return p.exchangeCode(ctx, code, redirectURI)
	case err := <-errCh:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("oauth authorization timed out after %v", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *OAuthProvider) exchangeCode(ctx context.Context, code, redirectURI string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
		"client_id":    {p.config.ClientID},
	}
	if p.config.ClientSecret != "" {
		data.Set("client_secret", p.config.ClientSecret)
	}
	return p.tokenRequest(ctx, data)
}

func (p *OAuthProvider) refreshToken(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.config.ClientID},
	}
	if p.config.ClientSecret != "" {
		data.Set("client_secret", p.config.ClientSecret)
	}
	return p.tokenRequest(ctx, data)
}

func (p *OAuthProvider) tokenRequest(ctx context.Context, data url.Values) (*OAuthToken, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed (%d): %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	if result.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	return &OAuthToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresAt:    expiresAt,
	}, nil
}

// DynamicRegister registers a client with the authorization server (RFC 7591).
func (p *OAuthProvider) DynamicRegister(ctx context.Context, registrationURL string) error {
	body := map[string]any{
		"client_name":   "opencode-go",
		"redirect_uris": []string{fmt.Sprintf("http://localhost:%d/callback", p.config.RedirectPort)},
		"grant_types":   []string{"authorization_code", "refresh_token"},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", registrationURL, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dynamic registration failed (%d): %s", resp.StatusCode, data)
	}

	var reg struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return err
	}
	p.config.ClientID = reg.ClientID
	p.config.ClientSecret = reg.ClientSecret
	return nil
}

func (p *OAuthProvider) tokenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".opencode", "mcp-auth", p.serverID+".json")
}

func (p *OAuthProvider) loadToken() (*OAuthToken, error) {
	b, err := os.ReadFile(p.tokenPath())
	if err != nil {
		return nil, err
	}
	var t OAuthToken
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (p *OAuthProvider) saveToken(t *OAuthToken) error {
	path := p.tokenPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
