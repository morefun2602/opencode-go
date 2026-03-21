package acp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/morefun2602/opencode-go/internal/store"
)

// Handler ACP HTTP 适配：映射到 store 会话存在性检查。
type Handler struct {
	Workspace string
	Store     store.Store
}

type eventReq struct {
	SessionID string          `json:"session_id"`
	Event     json.RawMessage `json:"event"`
}

// SessionEvent POST /v1/acp/session/event — 最小占位：校验会话后 202。
func (h *Handler) SessionEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req eventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.SessionID == "" {
		http.Error(w, "missing session_id", http.StatusBadRequest)
		return
	}
	ok, err := h.Store.SessionExists(r.Context(), h.Workspace, req.SessionID)
	if err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// Bridge 占位：将 ACP 会话 ID 绑定到运行时（后续扩展）。
func Bridge(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ctxKeySession{}, sessionID)
}

type ctxKeySession struct{}
