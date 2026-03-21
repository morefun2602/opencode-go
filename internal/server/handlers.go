package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/morefun2602/opencode-go/internal/acp"
	"github.com/morefun2602/opencode-go/internal/bus"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/tools"
)

type Handler struct {
	Cfg    config.Config
	Engine *runtime.Engine
	ACP    *acp.Handler
	Bus    *bus.Bus
}

type sessionResp struct {
	ID string `json:"id"`
}

type completeReq struct {
	Text string `json:"text"`
}

type completeResp struct {
	Messages []llm.Message `json:"messages"`
}

type sessionItem struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	CreatedAt   int64  `json:"created_at"`
}

type listSessionsResp struct {
	Sessions []sessionItem `json:"sessions"`
}

type messageItem struct {
	ID                   int64  `json:"id"`
	Role                 string `json:"role"`
	Body                 string `json:"body"`
	Parts                string `json:"parts"`
	TurnSeq              int    `json:"turn_seq"`
	CreatedAt            int64  `json:"created_at"`
	Model                string `json:"model,omitempty"`
	CostPromptTokens     int    `json:"cost_prompt_tokens,omitempty"`
	CostCompletionTokens int    `json:"cost_completion_tokens,omitempty"`
	FinishReason         string `json:"finish_reason,omitempty"`
	ToolCallID           string `json:"tool_call_id,omitempty"`
}

type listMessagesResp struct {
	Messages []messageItem `json:"messages"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	limit := 100
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := h.Engine.Store.ListSessions(r.Context(), h.Cfg.WorkspaceID, limit)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := listSessionsResp{Sessions: make([]sessionItem, 0, len(rows))}
	for _, row := range rows {
		out.Sessions = append(out.Sessions, sessionItem{ID: row.ID, WorkspaceID: row.WorkspaceID, CreatedAt: row.CreatedAt})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing session id")
		return
	}
	after := 0
	if s := r.URL.Query().Get("cursor"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			after = n
		}
	}
	limit := 100
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	ctx := r.Context()
	ok, err := h.Engine.Store.SessionExists(ctx, h.Cfg.WorkspaceID, id)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if !ok {
		writeJSONErr(w, http.StatusNotFound, "not_found", "unknown session")
		return
	}
	rows, err := h.Engine.Store.ListMessages(ctx, h.Cfg.WorkspaceID, id, after, limit)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := listMessagesResp{Messages: make([]messageItem, 0, len(rows))}
	for _, m := range rows {
		out.Messages = append(out.Messages, messageItem{
			ID: m.ID, Role: m.Role, Body: m.Body, Parts: m.Parts,
			TurnSeq: m.TurnSeq, CreatedAt: m.CreatedAt,
			Model: m.Model, CostPromptTokens: m.CostPromptTokens,
			CostCompletionTokens: m.CostCompletionTokens,
			FinishReason: m.FinishReason, ToolCallID: m.ToolCallID,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) OpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(OpenAPIJSON))
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	id, err := h.Engine.Store.CreateSession(r.Context(), h.Cfg.WorkspaceID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessionResp{ID: id})
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing session id")
		return
	}
	if r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	stream := r.URL.Query().Get("stream") == "1" || strings.EqualFold(r.URL.Query().Get("stream"), "true")
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req completeReq
	if err := json.Unmarshal(b, &req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	ctx := r.Context()
	if stream {
		h.completeStream(w, ctx, id, req.Text)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, h.Cfg.LLMTimeout)
	defer cancel()
	reply, err := h.Engine.CompleteTurn(ctx, h.Cfg.WorkspaceID, id, req.Text)
	if err != nil {
		if ctx.Err() != nil {
			writeJSONErr(w, http.StatusRequestTimeout, "canceled", ctx.Err().Error())
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(completeResp{
		Messages: []llm.Message{{Role: "assistant", Content: reply}},
	})
}

func (h *Handler) completeStream(w http.ResponseWriter, ctx context.Context, sessionID, text string) {
	ctx, cancel := context.WithTimeout(ctx, h.Cfg.LLMTimeout)
	defer cancel()
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fl, ok := w.(http.Flusher)
	if !ok {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "streaming unsupported")
		return
	}
	err := h.Engine.CompleteTurnStream(ctx, h.Cfg.WorkspaceID, sessionID, text, func(s string) error {
		evt, _ := json.Marshal(map[string]string{"type": "text", "text": s})
		_, e := io.WriteString(w, "data: "+string(evt)+"\n\n")
		fl.Flush()
		return e
	})
	if err != nil {
		evt, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		_, _ = io.WriteString(w, "data: "+string(evt)+"\n\n")
		fl.Flush()
		return
	}
	_, _ = io.WriteString(w, "event: done\ndata: {}\n\n")
	fl.Flush()
}

// --- 9.1 Providers and Models ---

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	names := h.Engine.Providers.List()
	type providerItem struct {
		ID string `json:"id"`
	}
	items := make([]providerItem, 0, len(names))
	for _, n := range names {
		items = append(items, providerItem{ID: n})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"providers": items})
}

func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing provider id")
		return
	}
	p, err := h.Engine.Providers.Get(id)
	if err != nil {
		writeJSONErr(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"models": p.Models()})
}

// --- 9.2 Session Fork ---

func (h *Handler) ForkSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing session id")
		return
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req struct {
		MessageSeq int `json:"message_seq"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	newID, err := h.Engine.Store.Fork(r.Context(), h.Cfg.WorkspaceID, id, req.MessageSeq)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sessionResp{ID: newID})
}

// --- 9.3 Session Revert ---

func (h *Handler) RevertSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing session id")
		return
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req struct {
		MessageSeq int `json:"message_seq"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if err := h.Engine.Store.Revert(r.Context(), h.Cfg.WorkspaceID, id, req.MessageSeq); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// --- 9.4 Session Metadata Update ---

func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PATCH")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing session id")
		return
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req struct {
		Title    *string `json:"title"`
		Archived *bool   `json:"archived"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	ctx := r.Context()
	if req.Title != nil {
		if err := h.Engine.Store.SetTitle(ctx, h.Cfg.WorkspaceID, id, *req.Title); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
	}
	if req.Archived != nil {
		if err := h.Engine.Store.SetArchived(ctx, h.Cfg.WorkspaceID, id, *req.Archived); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// --- 9.5 Session Usage ---

func (h *Handler) SessionUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "missing session id")
		return
	}
	prompt, completion, err := h.Engine.Store.Usage(r.Context(), h.Cfg.WorkspaceID, id)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{
		"prompt_tokens":     prompt,
		"completion_tokens": completion,
		"total_tokens":      prompt + completion,
	})
}

// --- 9.6 Config Endpoint ---

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	sanitized := map[string]any{
		"listen":           h.Cfg.Listen,
		"workspace_id":     h.Cfg.WorkspaceID,
		"default_provider": h.Cfg.DefaultProvider,
		"default_model":    h.Cfg.DefaultModel,
		"max_tool_rounds":  h.Cfg.MaxToolRounds,
		"workspace_root":   h.Cfg.WorkspaceRoot,
		"data_dir":         h.Cfg.DataDir,
		"llm_timeout":      h.Cfg.LLMTimeout.String(),
		"bash_timeout_sec": h.Cfg.BashTimeoutSec,
		"max_output_bytes": h.Cfg.MaxOutputBytes,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sanitized)
}

// --- 9.7 Permission and Question Reply ---

func (h *Handler) PermissionReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req struct {
		PermissionID string `json:"permission_id"`
		Action       string `json:"action"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (h *Handler) QuestionReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req struct {
		QuestionID string `json:"question_id"`
		Answer     string `json:"answer"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if !tools.Questions.Reply(req.QuestionID, req.Answer) {
		writeJSONErr(w, http.StatusNotFound, "not_found", "unknown question")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// --- 9.8 SSE Events Endpoint ---

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if h.Bus == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "unavailable", "event bus not configured")
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fl.Flush()

	ch := h.Bus.Subscribe("*")
	defer h.Bus.Unsubscribe("*", ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
			if err != nil {
				return
			}
			fl.Flush()
		}
	}
}

func NewMux(h *Handler) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /v1/health", h.Health)
	m.HandleFunc("GET /v1/openapi.json", h.OpenAPI)
	m.HandleFunc("GET /v1/sessions", h.ListSessions)
	m.HandleFunc("POST /v1/sessions", h.CreateSession)
	m.HandleFunc("GET /v1/sessions/{id}/messages", h.ListMessages)
	m.HandleFunc("POST /v1/sessions/{id}/complete", h.Complete)
	m.HandleFunc("GET /v1/providers", h.ListProviders)
	m.HandleFunc("GET /v1/providers/{id}/models", h.ListModels)
	m.HandleFunc("POST /v1/sessions/{id}/fork", h.ForkSession)
	m.HandleFunc("POST /v1/sessions/{id}/revert", h.RevertSession)
	m.HandleFunc("PATCH /v1/sessions/{id}", h.UpdateSession)
	m.HandleFunc("GET /v1/sessions/{id}/usage", h.SessionUsage)
	m.HandleFunc("GET /v1/config", h.GetConfig)
	m.HandleFunc("POST /v1/permission/reply", h.PermissionReply)
	m.HandleFunc("POST /v1/question/reply", h.QuestionReply)
	m.HandleFunc("GET /v1/events", h.Events)
	if h.ACP != nil {
		m.HandleFunc("POST /v1/acp/session/event", h.ACP.SessionEvent)
	}
	return m
}

func AuthMiddleware(token string, next http.Handler) http.Handler {
	if strings.TrimSpace(token) == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		got = strings.TrimSpace(got)
		if got != token {
			writeJSONErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = time.Now().Format(time.RFC3339Nano)
		}
		ctx := context.WithValue(r.Context(), ctxKeyReqID{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKeyReqID struct{}

func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyReqID{}).(string)
	return v
}
