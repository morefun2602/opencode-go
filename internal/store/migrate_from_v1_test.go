package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateFromV1AddsMessageVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "legacy.db")
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
  created_at INTEGER NOT NULL
);`,
		`CREATE INDEX idx_messages_session ON messages(session_id, turn_seq);`,
		`PRAGMA user_version = 1`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var v int
	if err := st.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != schemaVersion {
		t.Fatalf("want user_version %d, got %d", schemaVersion, v)
	}
}
