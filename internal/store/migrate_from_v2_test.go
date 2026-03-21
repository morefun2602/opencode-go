package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateFromV2AddsPartsColumns(t *testing.T) {
	p := filepath.Join(t.TempDir(), "v2.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		`CREATE TABLE workspaces (id TEXT NOT NULL PRIMARY KEY);`,
		`CREATE TABLE sessions (
  id TEXT NOT NULL PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  created_at INTEGER NOT NULL
);`,
		`CREATE TABLE messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  workspace_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  role TEXT NOT NULL,
  body TEXT NOT NULL,
  turn_seq INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  message_version INTEGER NOT NULL DEFAULT 1
);`,
		`INSERT INTO workspaces(id) VALUES ('ws1')`,
		`INSERT INTO sessions(id, workspace_id, created_at) VALUES ('s1', 'ws1', 100)`,
		`INSERT INTO messages(workspace_id, session_id, role, body, turn_seq, created_at) VALUES ('ws1', 's1', 'user', 'hello', 1, 100)`,
		`INSERT INTO messages(workspace_id, session_id, role, body, turn_seq, created_at) VALUES ('ws1', 's1', 'assistant', 'world', 2, 101)`,
		`PRAGMA user_version = 2`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	st, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	var v int
	if err := st.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 4 {
		t.Fatalf("want user_version 4, got %d", v)
	}

	var parts string
	err = st.db.QueryRow(`SELECT parts FROM messages WHERE body='hello'`).Scan(&parts)
	if err != nil {
		t.Fatal(err)
	}
	if parts == "[]" || parts == "" {
		t.Fatalf("expected parts to be migrated from body, got %q", parts)
	}
	t.Logf("migrated parts: %s", parts)
}
