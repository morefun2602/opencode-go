package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func DoSearch(ctx context.Context, baseURL, query string, timeout, maxOut int) (string, error) {
	if baseURL == "" {
		return "", fmt.Errorf("websearch: no search URL configured")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("websearch: invalid URL: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("websearch: HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, int64(maxOut)+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	s := string(b)
	if len(b) > maxOut {
		s = s[:maxOut] + "\n…truncated"
	}
	return s, nil
}
