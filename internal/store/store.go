package store

import (
	"context"
)

// SessionRow 会话列表项。
type SessionRow struct {
	ID               string
	WorkspaceID      string
	CreatedAt        int64
	Title            string
	Archived         bool
	ParentID         string
	ParentMessageSeq int
}

// MessageRow 单条消息。
type MessageRow struct {
	ID                   int64
	Role                 string
	Body                 string
	Parts                string // JSON array of Part
	TurnSeq              int
	CreatedAt            int64
	MessageVersion       int
	Model                string
	CostPromptTokens     int
	CostCompletionTokens int
	FinishReason         string
	ToolCallID           string
}

// Store 持久化抽象；按 workspace 隔离。
type Store interface {
	Close() error
	CreateSession(ctx context.Context, workspaceID string) (sessionID string, err error)
	AppendMessages(ctx context.Context, workspaceID, sessionID string, msgs []MessageRow) error
	ListSessions(ctx context.Context, workspaceID string, limit int) ([]SessionRow, error)
	ListMessages(ctx context.Context, workspaceID, sessionID string, afterSeq, limit int) ([]MessageRow, error)
	SessionExists(ctx context.Context, workspaceID, sessionID string) (bool, error)
	Fork(ctx context.Context, workspaceID, sessionID string, seq int) (string, error)
	Revert(ctx context.Context, workspaceID, sessionID string, seq int) error
	Unrevert(ctx context.Context, workspaceID, sessionID string) error
	SetTitle(ctx context.Context, workspaceID, sessionID, title string) error
	SetArchived(ctx context.Context, workspaceID, sessionID string, archived bool) error
	Usage(ctx context.Context, workspaceID, sessionID string) (prompt int, completion int, err error)
}
