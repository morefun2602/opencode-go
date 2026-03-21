package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/morefun2602/opencode-go/internal/acp"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/store"
	"github.com/morefun2602/opencode-go/internal/tool"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func TestListSessionsAndOpenAPI(t *testing.T) {
	db := filepath.Join(t.TempDir(), "db.sqlite")
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Defaults()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	treg := tools.New(log)
	eng := &runtime.Engine{Store: st, LLM: llm.Stub{}, Tools: &tool.Router{Builtin: treg, Log: log}, Log: log}
	h := &Handler{Cfg: cfg, Engine: eng, ACP: &acp.Handler{Workspace: cfg.WorkspaceID, Store: st}}
	srv := httptest.NewServer(AuthMiddleware("", RequestID(NewMux(h))))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/v1/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("openapi: %d", res.StatusCode)
	}

	res2, err := http.Post(srv.URL+"/v1/sessions", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	res3, err := http.Get(srv.URL + "/v1/sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("list sessions: %d", res3.StatusCode)
	}
	var lr listSessionsResp
	if err := json.NewDecoder(res3.Body).Decode(&lr); err != nil {
		t.Fatal(err)
	}
	if len(lr.Sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(lr.Sessions))
	}
}
