package server

import (
	"bytes"
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

func TestE2ESessionAndComplete(t *testing.T) {
	db := filepath.Join(t.TempDir(), "db.sqlite")
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Defaults()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	treg := tools.New(log)
	eng := &runtime.Engine{
		Store: st,
		LLM:   llm.Stub{},
		Tools: &tool.Router{Builtin: treg, Log: log},
		Log:   log,
	}
	h := &Handler{Cfg: cfg, Engine: eng, ACP: &acp.Handler{Workspace: cfg.WorkspaceID, Store: st}}
	srv := httptest.NewServer(AuthMiddleware("", RequestID(NewMux(h))))
	defer srv.Close()

	res, err := http.Post(srv.URL+"/v1/sessions", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sessions: %d", res.StatusCode)
	}
	var sr sessionResp
	if err := json.NewDecoder(res.Body).Decode(&sr); err != nil {
		t.Fatal(err)
	}
	if sr.ID == "" {
		t.Fatal("empty session id")
	}
	body := bytes.NewReader([]byte(`{"text":"hi"}`))
	res2, err := http.Post(srv.URL+"/v1/sessions/"+sr.ID+"/complete", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("complete: %d", res2.StatusCode)
	}
	var cr completeResp
	if err := json.NewDecoder(res2.Body).Decode(&cr); err != nil {
		t.Fatal(err)
	}
	if len(cr.Messages) == 0 || cr.Messages[0].Content == "" {
		t.Fatal("empty reply")
	}
}
