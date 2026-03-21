package store

import (
	"database/sql"
	"fmt"
)

const (
	schemaVersion      = 4
	maxSchemaSupported = 4
)

func migrate(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v > maxSchemaSupported {
		return fmt.Errorf("数据库 schema 版本 %d 高于本程序支持的上限 %d：请升级 opencode-go 或备份后降级数据", v, maxSchemaSupported)
	}
	if v == schemaVersion {
		return nil
	}
	if v == 0 {
		stmts := []string{
			`CREATE TABLE IF NOT EXISTS workspaces (id TEXT NOT NULL PRIMARY KEY);`,
			`CREATE TABLE IF NOT EXISTS sessions (
  id TEXT NOT NULL PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  title TEXT DEFAULT '',
  archived INTEGER DEFAULT 0,
  parent_id TEXT DEFAULT '',
  parent_message_seq INTEGER DEFAULT 0
);`,
			`CREATE TABLE IF NOT EXISTS messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  workspace_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  role TEXT NOT NULL,
  body TEXT NOT NULL,
  turn_seq INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  message_version INTEGER NOT NULL DEFAULT 1,
  parts TEXT NOT NULL DEFAULT '[]',
  model TEXT NOT NULL DEFAULT '',
  cost_prompt_tokens INTEGER NOT NULL DEFAULT 0,
  cost_completion_tokens INTEGER NOT NULL DEFAULT 0,
  finish_reason TEXT NOT NULL DEFAULT '',
  tool_call_id TEXT NOT NULL DEFAULT ''
);`,
			`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, turn_seq);`,
			`CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id, created_at DESC);`,
		}
		for _, s := range stmts {
			if _, err := db.Exec(s); err != nil {
				return err
			}
		}
		if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
			return err
		}
		return nil
	}
	if v == 1 {
		if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN message_version INTEGER NOT NULL DEFAULT 1`); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id, created_at DESC)`); err != nil {
			return err
		}
		v = 2
	}
	if v == 2 {
		cols := []string{
			`ALTER TABLE messages ADD COLUMN parts TEXT NOT NULL DEFAULT '[]'`,
			`ALTER TABLE messages ADD COLUMN model TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE messages ADD COLUMN cost_prompt_tokens INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE messages ADD COLUMN cost_completion_tokens INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE messages ADD COLUMN finish_reason TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE messages ADD COLUMN tool_call_id TEXT NOT NULL DEFAULT ''`,
		}
		for _, s := range cols {
			if _, err := db.Exec(s); err != nil {
				return err
			}
		}
		if _, err := db.Exec(`UPDATE messages SET parts = json_array(json_object('type','text','text',body)) WHERE parts = '[]' AND body != ''`); err != nil {
			return err
		}
		if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
			return err
		}
		return nil
	}
	if v == 3 {
		cols := []string{
			`ALTER TABLE sessions ADD COLUMN title TEXT DEFAULT ''`,
			`ALTER TABLE sessions ADD COLUMN archived INTEGER DEFAULT 0`,
			`ALTER TABLE sessions ADD COLUMN parent_id TEXT DEFAULT ''`,
			`ALTER TABLE sessions ADD COLUMN parent_message_seq INTEGER DEFAULT 0`,
		}
		for _, s := range cols {
			if _, err := db.Exec(s); err != nil {
				return err
			}
		}
		if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("无法迁移：未知 schema 版本 %d", v)
}
