package truncate

import (
	"strings"
	"testing"
)

func TestTruncateNoOp(t *testing.T) {
	r := Truncate("short output", DefaultOptions())
	if r.Truncated {
		t.Error("should not be truncated")
	}
	if r.Output != "short output" {
		t.Errorf("output should be unchanged, got: %s", r.Output)
	}
}

func TestTruncateByLines(t *testing.T) {
	lines := make([]string, 3000)
	for i := range lines {
		lines[i] = "x"
	}
	input := strings.Join(lines, "\n")

	r := Truncate(input, Options{MaxLines: 100, MaxBytes: 1 << 20, Direction: Tail})
	if !r.Truncated {
		t.Error("should be truncated")
	}
	outLines := strings.Split(r.Output, "\n")
	if len(outLines) < 100 {
		t.Errorf("should have at least 100 lines, got %d", len(outLines))
	}
}

func TestTruncateByBytes(t *testing.T) {
	input := strings.Repeat("a", 100000)
	r := Truncate(input, Options{MaxLines: 100000, MaxBytes: 1000, Direction: Head})
	if !r.Truncated {
		t.Error("should be truncated")
	}
	if len(r.Output) > 1100 {
		t.Errorf("output should be around 1000 bytes, got %d", len(r.Output))
	}
}

func TestTruncateHead(t *testing.T) {
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	input := strings.Join(lines, "\n")

	r := Truncate(input, Options{MaxLines: 3, MaxBytes: 1 << 20, Direction: Head})
	if !r.Truncated {
		t.Error("should be truncated")
	}
	if !strings.Contains(r.Output, "line1") {
		t.Error("head truncation should keep first lines")
	}
}

func TestTruncateTail(t *testing.T) {
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	input := strings.Join(lines, "\n")

	r := Truncate(input, Options{MaxLines: 3, MaxBytes: 1 << 20, Direction: Tail})
	if !r.Truncated {
		t.Error("should be truncated")
	}
	if !strings.Contains(r.Output, "line5") {
		t.Error("tail truncation should keep last lines")
	}
}
