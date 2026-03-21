package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DiffLine struct {
	Op   byte
	Text string
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

type FilePatch struct {
	OldPath string
	NewPath string
	Hunks   []Hunk
}

func ParsePatch(patch string) ([]FilePatch, error) {
	lines := strings.Split(patch, "\n")
	var result []FilePatch
	var cur *FilePatch
	var hunk *Hunk

	for _, line := range lines {
		if strings.HasPrefix(line, "--- ") {
			if cur != nil {
				if hunk != nil {
					cur.Hunks = append(cur.Hunks, *hunk)
					hunk = nil
				}
				result = append(result, *cur)
			}
			cur = &FilePatch{OldPath: stripDiffPrefix(line[4:])}
			continue
		}
		if strings.HasPrefix(line, "+++ ") && cur != nil {
			cur.NewPath = stripDiffPrefix(line[4:])
			continue
		}
		if strings.HasPrefix(line, "@@") && cur != nil {
			if hunk != nil {
				cur.Hunks = append(cur.Hunks, *hunk)
			}
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			hunk = &h
			continue
		}
		if hunk != nil {
			if len(line) == 0 {
				hunk.Lines = append(hunk.Lines, DiffLine{Op: ' '})
				continue
			}
			op := line[0]
			if op == ' ' || op == '+' || op == '-' {
				text := ""
				if len(line) > 1 {
					text = line[1:]
				}
				hunk.Lines = append(hunk.Lines, DiffLine{Op: op, Text: text})
				continue
			}
			if strings.HasPrefix(line, "\\ No newline") {
				continue
			}
		}
	}
	if cur != nil {
		if hunk != nil {
			cur.Hunks = append(cur.Hunks, *hunk)
		}
		result = append(result, *cur)
	}
	return result, nil
}

func ApplyFilePatches(patches []FilePatch, resolve func(string) (string, error)) error {
	type bak struct {
		path    string
		data    []byte
		existed bool
	}
	var baks []bak

	restore := func() {
		for _, b := range baks {
			if b.existed {
				_ = os.WriteFile(b.path, b.data, 0o644)
			} else {
				_ = os.Remove(b.path)
			}
		}
	}

	for _, fp := range patches {
		p := fp.NewPath
		if p == "/dev/null" {
			p = fp.OldPath
		}
		rp, err := resolve(p)
		if err != nil {
			restore()
			return fmt.Errorf("resolve %q: %w", p, err)
		}

		data, readErr := os.ReadFile(rp)
		baks = append(baks, bak{path: rp, data: data, existed: readErr == nil})

		if fp.NewPath == "/dev/null" {
			if err := os.Remove(rp); err != nil {
				restore()
				return fmt.Errorf("delete %q: %w", p, err)
			}
			continue
		}

		if fp.OldPath == "/dev/null" {
			var lines []string
			for _, h := range fp.Hunks {
				for _, dl := range h.Lines {
					if dl.Op == '+' {
						lines = append(lines, dl.Text)
					}
				}
			}
			if err := os.MkdirAll(filepath.Dir(rp), 0o755); err != nil {
				restore()
				return err
			}
			if err := os.WriteFile(rp, []byte(joinLines(lines)), 0o644); err != nil {
				restore()
				return err
			}
			continue
		}

		if readErr != nil {
			restore()
			return fmt.Errorf("read %q: %w", p, readErr)
		}
		orig := splitLines(string(data))
		result, err := applyHunks(orig, fp.Hunks)
		if err != nil {
			restore()
			return fmt.Errorf("patch %q: %w", p, err)
		}
		if err := os.WriteFile(rp, []byte(joinLines(result)), 0o644); err != nil {
			restore()
			return err
		}
	}
	return nil
}

func applyHunks(orig []string, hunks []Hunk) ([]string, error) {
	var out []string
	idx := 0
	for _, h := range hunks {
		start := h.OldStart - 1
		if start < 0 {
			start = 0
		}
		for idx < start && idx < len(orig) {
			out = append(out, orig[idx])
			idx++
		}
		for _, dl := range h.Lines {
			switch dl.Op {
			case ' ':
				if idx < len(orig) {
					out = append(out, orig[idx])
					idx++
				}
			case '-':
				if idx < len(orig) {
					idx++
				}
			case '+':
				out = append(out, dl.Text)
			}
		}
	}
	for idx < len(orig) {
		out = append(out, orig[idx])
		idx++
	}
	return out, nil
}

func parseHunkHeader(line string) (Hunk, error) {
	var h Hunk
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@@") {
		return h, fmt.Errorf("invalid hunk header: %s", line)
	}
	end := strings.Index(line[2:], "@@")
	if end < 0 {
		return h, fmt.Errorf("invalid hunk header: %s", line)
	}
	inner := strings.TrimSpace(line[2 : 2+end])
	parts := strings.Fields(inner)
	if len(parts) < 2 {
		return h, fmt.Errorf("invalid hunk header: %s", line)
	}
	var err error
	h.OldStart, h.OldCount, err = parseRange(parts[0], "-")
	if err != nil {
		return h, err
	}
	h.NewStart, h.NewCount, err = parseRange(parts[1], "+")
	if err != nil {
		return h, err
	}
	return h, nil
}

func parseRange(s, prefix string) (int, int, error) {
	if !strings.HasPrefix(s, prefix) {
		return 0, 0, fmt.Errorf("expected prefix %q in %q", prefix, s)
	}
	s = s[len(prefix):]
	if i := strings.Index(s, ","); i >= 0 {
		start, err := strconv.Atoi(s[:i])
		if err != nil {
			return 0, 0, err
		}
		count, err := strconv.Atoi(s[i+1:])
		if err != nil {
			return 0, 0, err
		}
		return start, count, nil
	}
	start, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, err
	}
	return start, 1, nil
}

func stripDiffPrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "/dev/null" {
		return p
	}
	if strings.HasPrefix(p, "a/") || strings.HasPrefix(p, "b/") {
		return p[2:]
	}
	return p
}

func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
