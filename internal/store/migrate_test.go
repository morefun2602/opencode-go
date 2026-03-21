package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRejectFutureSchemaVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = Open(p)
	if err == nil {
		t.Fatal("expected error for schema newer than supported")
	}
}
