package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMiddleware401(t *testing.T) {
	h := AuthMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareOK(t *testing.T) {
	h := AuthMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestJSONErrShape(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSONErr(rec, http.StatusBadRequest, "bad", "msg")
	if rec.Code != http.StatusBadRequest {
		t.Fatal(rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"bad"`) {
		t.Fatal(rec.Body.String())
	}
}
