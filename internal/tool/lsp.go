package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morefun2602/opencode-go/internal/lsp"
	"github.com/morefun2602/opencode-go/internal/tools"
)

// RegisterLSP registers the lsp tool. Called separately because it needs an LSP client.
func RegisterLSP(reg *tools.Registry, client *lsp.Client) {
	reg.Register(tools.Tool{
		Name:        "lsp",
		Description: "Perform LSP operations: diagnostics, definition, references, symbols",
		Tags:        []string{"read"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"diagnostics", "definition", "references", "symbols"},
					"description": "LSP operation to perform",
				},
				"path": map[string]any{"type": "string", "description": "file path"},
				"line": map[string]any{"type": "integer", "description": "line number (0-based, for definition/references)"},
				"character": map[string]any{"type": "integer", "description": "character offset (0-based, for definition/references)"},
			},
			"required": []string{"action", "path"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			if client == nil {
				return "", fmt.Errorf("LSP not available: no language server configured")
			}
			action := fmt.Sprint(args["action"])
			path := fmt.Sprint(args["path"])
			uri := "file://" + path

			switch action {
			case "diagnostics":
				diags := client.GetDiagnostics(uri)
				if len(diags) == 0 {
					return "no diagnostics", nil
				}
				var sb strings.Builder
				for _, d := range diags {
					sev := "info"
					switch d.Severity {
					case 1:
						sev = "error"
					case 2:
						sev = "warning"
					case 3:
						sev = "info"
					case 4:
						sev = "hint"
					}
					fmt.Fprintf(&sb, "%s:%d:%d: [%s] %s\n",
						path, d.Range.Start.Line+1, d.Range.Start.Character+1,
						sev, d.Message)
				}
				return sb.String(), nil

			case "definition":
				line := argInt(args, "line")
				char := argInt(args, "character")
				locs, err := client.Definition(ctx, uri, line, char)
				if err != nil {
					return "", err
				}
				if len(locs) == 0 {
					return "no definition found", nil
				}
				var sb strings.Builder
				for _, l := range locs {
					fmt.Fprintf(&sb, "%s:%d:%d\n", l.URI, l.Range.Start.Line+1, l.Range.Start.Character+1)
				}
				return sb.String(), nil

			case "references":
				line := argInt(args, "line")
				char := argInt(args, "character")
				locs, err := client.References(ctx, uri, line, char)
				if err != nil {
					return "", err
				}
				if len(locs) == 0 {
					return "no references found", nil
				}
				var sb strings.Builder
				for _, l := range locs {
					fmt.Fprintf(&sb, "%s:%d:%d\n", l.URI, l.Range.Start.Line+1, l.Range.Start.Character+1)
				}
				return sb.String(), nil

			case "symbols":
				syms, err := client.DocumentSymbols(ctx, uri)
				if err != nil {
					return "", err
				}
				if len(syms) == 0 {
					return "no symbols found", nil
				}
				var sb strings.Builder
				formatSymbols(&sb, syms, 0)
				return sb.String(), nil

			default:
				return "", fmt.Errorf("unknown LSP action: %s", action)
			}
		},
	})
}

func formatSymbols(sb *strings.Builder, syms []lsp.DocumentSymbol, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, s := range syms {
		fmt.Fprintf(sb, "%s%s (kind=%d) L%d\n", prefix, s.Name, s.Kind, s.Range.Start.Line+1)
		if len(s.Children) > 0 {
			formatSymbols(sb, s.Children, indent+1)
		}
	}
}

func argInt(m map[string]any, k string) int {
	v, ok := m[k]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	default:
		return 0
	}
}
