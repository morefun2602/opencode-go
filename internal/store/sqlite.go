package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) CreateSession(ctx context.Context, workspaceID string) (string, error) {
	id := newID()
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO workspaces(id) VALUES (?)`, workspaceID); err != nil {
		return "", err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions(id, workspace_id, created_at) VALUES (?,?,?)`,
		id, workspaceID, time.Now().Unix())
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *SQLite) AppendMessages(ctx context.Context, workspaceID, sessionID string, msgs []MessageRow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var seq int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(turn_seq),0) FROM messages WHERE session_id=? AND workspace_id=?`,
		sessionID, workspaceID).Scan(&seq); err != nil {
		return err
	}
	for _, m := range msgs {
		seq++
		parts := m.Parts
		if parts == "" {
			parts = "[]"
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO messages(workspace_id, session_id, role, body, turn_seq, created_at, message_version, parts, model, cost_prompt_tokens, cost_completion_tokens, finish_reason, tool_call_id) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			workspaceID, sessionID, m.Role, m.Body, seq, time.Now().Unix(), 2,
			parts, m.Model, m.CostPromptTokens, m.CostCompletionTokens, m.FinishReason, m.ToolCallID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLite) ListSessions(ctx context.Context, workspaceID string, limit int) ([]SessionRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workspace_id, created_at, title, archived, parent_id, parent_message_seq FROM sessions WHERE workspace_id=? AND archived=0 ORDER BY created_at DESC LIMIT ?`,
		workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionRow
	for rows.Next() {
		var r SessionRow
		var archived int
		if err := rows.Scan(&r.ID, &r.WorkspaceID, &r.CreatedAt, &r.Title, &archived, &r.ParentID, &r.ParentMessageSeq); err != nil {
			return nil, err
		}
		r.Archived = archived != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLite) ListMessages(ctx context.Context, workspaceID, sessionID string, afterSeq, limit int) ([]MessageRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, role, body, turn_seq, created_at, message_version, parts, model, cost_prompt_tokens, cost_completion_tokens, finish_reason, tool_call_id FROM messages
WHERE workspace_id=? AND session_id=? AND turn_seq > ? ORDER BY turn_seq ASC LIMIT ?`,
		workspaceID, sessionID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MessageRow
	for rows.Next() {
		var m MessageRow
		if err := rows.Scan(&m.ID, &m.Role, &m.Body, &m.TurnSeq, &m.CreatedAt, &m.MessageVersion,
			&m.Parts, &m.Model, &m.CostPromptTokens, &m.CostCompletionTokens, &m.FinishReason, &m.ToolCallID); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *SQLite) SessionExists(ctx context.Context, workspaceID, sessionID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM sessions WHERE workspace_id=? AND id=?`,
		workspaceID, sessionID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *SQLite) Fork(ctx context.Context, workspaceID, sessionID string, seq int) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	id := newID()
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sessions(id, workspace_id, created_at, title, archived, parent_id, parent_message_seq) VALUES (?,?,?,'',0,?,?)`,
		id, workspaceID, now, sessionID, seq); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO messages(workspace_id, session_id, role, body, turn_seq, created_at, message_version, parts, model, cost_prompt_tokens, cost_completion_tokens, finish_reason, tool_call_id)
SELECT workspace_id, ?, role, body, turn_seq, created_at, message_version, parts, model, cost_prompt_tokens, cost_completion_tokens, finish_reason, tool_call_id
FROM messages WHERE session_id=? AND workspace_id=? AND turn_seq<=?`,
		id, sessionID, workspaceID, seq); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *SQLite) Revert(ctx context.Context, workspaceID, sessionID string, seq int) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM messages WHERE session_id=? AND workspace_id=? AND turn_seq>?`,
		sessionID, workspaceID, seq)
	return err
}

func (s *SQLite) SetTitle(ctx context.Context, workspaceID, sessionID, title string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET title=? WHERE id=? AND workspace_id=?`,
		title, sessionID, workspaceID)
	return err
}

func (s *SQLite) SetArchived(ctx context.Context, workspaceID, sessionID string, archived bool) error {
	v := 0
	if archived {
		v = 1
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET archived=? WHERE id=? AND workspace_id=?`,
		v, sessionID, workspaceID)
	return err
}

func (s *SQLite) Usage(ctx context.Context, workspaceID, sessionID string) (int, int, error) {
	var prompt, completion int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_prompt_tokens),0), COALESCE(SUM(cost_completion_tokens),0) FROM messages WHERE session_id=? AND workspace_id=?`,
		sessionID, workspaceID).Scan(&prompt, &completion)
	if err != nil {
		return 0, 0, err
	}
	return prompt, completion, nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
