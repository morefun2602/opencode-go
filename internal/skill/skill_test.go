package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	sk, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 1 || sk[0].Name != "a" {
		t.Fatalf("got %+v", sk)
	}
}
