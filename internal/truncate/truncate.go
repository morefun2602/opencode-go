package truncate

import (
	"fmt"
	"strings"
)

type Direction int

const (
	Tail Direction = iota
	Head
)

type Options struct {
	MaxLines  int
	MaxBytes  int
	Direction Direction
}

type Result struct {
	Output    string
	Truncated bool
}

func DefaultOptions() Options {
	return Options{
		MaxLines:  2000,
		MaxBytes:  50 * 1024,
		Direction: Tail,
	}
}

func Truncate(output string, opts Options) Result {
	if opts.MaxLines <= 0 {
		opts.MaxLines = 2000
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 50 * 1024
	}

	if len(output) <= opts.MaxBytes {
		lines := strings.Split(output, "\n")
		if len(lines) <= opts.MaxLines {
			return Result{Output: output, Truncated: false}
		}
	}

	truncated := false
	result := output

	lines := strings.Split(result, "\n")
	if len(lines) > opts.MaxLines {
		truncated = true
		if opts.Direction == Head {
			lines = lines[:opts.MaxLines]
		} else {
			lines = lines[len(lines)-opts.MaxLines:]
		}
		result = strings.Join(lines, "\n")
	}

	if len(result) > opts.MaxBytes {
		truncated = true
		if opts.Direction == Head {
			result = result[:opts.MaxBytes]
		} else {
			if len(result) > opts.MaxBytes {
				result = result[len(result)-opts.MaxBytes:]
			}
		}
	}

	if truncated {
		origLines := strings.Count(output, "\n") + 1
		origBytes := len(output)
		result += fmt.Sprintf("\n...truncated (original: %d lines / %d bytes)", origLines, origBytes)
	}

	return Result{Output: result, Truncated: truncated}
}
