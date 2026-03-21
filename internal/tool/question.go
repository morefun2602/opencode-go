package tool

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerQuestion(reg *tools.Registry) {
	reg.Register(tools.Tool{
		Name:        "question",
		Description: "Ask the user a question and wait for their answer",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{"type": "string", "description": "question to ask the user"},
				"options":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "optional answer choices"},
			},
			"required": []string{"question"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			text := fmt.Sprint(args["question"])
			var opts []string
			if raw, ok := args["options"]; ok && raw != nil {
				b, _ := json.Marshal(raw)
				_ = json.Unmarshal(b, &opts)
			}
			id := newQuestionID()
			return tools.Questions.Ask(ctx, id, text, opts)
		},
	})
}

func newQuestionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "q_" + hex.EncodeToString(b)
}
