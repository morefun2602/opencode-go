package skill

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkills_OnlySkillMD(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "my-skill")
	os.MkdirAll(sub, 0o755)

	os.WriteFile(filepath.Join(sub, "SKILL.md"), []byte("---\nname: my-skill\ndescription: test skill\n---\nBody content"), 0o600)
	os.WriteFile(filepath.Join(sub, "README.md"), []byte("# Readme"), 0o600)

	sk, err := DiscoverSkills([]string{dir}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(sk))
	}
	if sk[0].Name != "my-skill" {
		t.Fatalf("expected name 'my-skill', got %q", sk[0].Name)
	}
	if sk[0].Description != "test skill" {
		t.Fatalf("expected description 'test skill', got %q", sk[0].Description)
	}
	if sk[0].Location == "" {
		t.Fatal("expected Location to be set")
	}
}

func TestDiscoverSkills_Priority(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	sub1 := filepath.Join(dir1, "foo")
	sub2 := filepath.Join(dir2, "foo")
	os.MkdirAll(sub1, 0o755)
	os.MkdirAll(sub2, 0o755)

	os.WriteFile(filepath.Join(sub1, "SKILL.md"), []byte("---\nname: foo\ndescription: first\n---\nFirst"), 0o600)
	os.WriteFile(filepath.Join(sub2, "SKILL.md"), []byte("---\nname: foo\ndescription: second\n---\nSecond"), 0o600)

	sk, err := DiscoverSkills([]string{dir1, dir2}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 1 {
		t.Fatalf("expected 1 skill (deduplicated), got %d", len(sk))
	}
	if sk[0].Description != "first" {
		t.Fatalf("expected first path to win, got description %q", sk[0].Description)
	}
}

func TestDiscoverSkills_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "bare-skill")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "SKILL.md"), []byte("Just some content"), 0o600)

	sk, err := DiscoverSkills([]string{dir}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(sk))
	}
	if sk[0].Name != "bare-skill" {
		t.Fatalf("expected name from dir 'bare-skill', got %q", sk[0].Name)
	}
}

func TestDiscoverSkills_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "test")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "skill.md"), []byte("---\nname: lower\ndescription: lowercase\n---\nBody"), 0o600)

	sk, err := DiscoverSkills([]string{dir}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 1 {
		t.Fatalf("expected 1 skill (case-insensitive match), got %d", len(sk))
	}
}

func TestFmt_Verbose(t *testing.T) {
	skills := []Skill{
		{Name: "test", Description: "Test skill", Location: "/tmp/test/SKILL.md"},
	}
	out := Fmt(skills, true)
	if !contains(out, "<available_skills>") {
		t.Fatal("expected XML format")
	}
	if !contains(out, "<name>test</name>") {
		t.Fatal("expected skill name in XML")
	}
}

func TestFmt_Concise(t *testing.T) {
	skills := []Skill{
		{Name: "test", Description: "Test skill", Location: "/tmp/test/SKILL.md"},
	}
	out := Fmt(skills, false)
	if !contains(out, "## Available Skills") {
		t.Fatal("expected markdown header")
	}
	if !contains(out, "- **test**: Test skill") {
		t.Fatal("expected markdown list item")
	}
}

func TestFmt_Empty(t *testing.T) {
	out := Fmt(nil, true)
	if out != "No skills are currently available." {
		t.Fatalf("unexpected output for empty list: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
