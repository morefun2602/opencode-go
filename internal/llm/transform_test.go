package llm

import (
	"testing"
)

func TestTransformMessagesAnthropic(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", Content: "", Parts: nil},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi", Parts: []Part{
			{Type: "tool_call", ToolCallID: "call@123!abc", ToolName: "read"},
		}},
	}

	result := TransformMessages(msgs, "anthropic")

	if len(result) != 2 {
		t.Fatalf("expected 2 messages (empty assistant filtered), got %d", len(result))
	}

	if result[1].Parts[0].ToolCallID == "call@123!abc" {
		t.Error("ToolCallID should have been sanitized")
	}

	for _, c := range result[1].Parts[0].ToolCallID {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			t.Errorf("invalid character in sanitized ToolCallID: %c", c)
		}
	}
}

func TestTransformMessagesPassthrough(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "test"},
	}

	result := TransformMessages(msgs, "openai")
	if len(result) != 1 || result[0].Content != "test" {
		t.Error("openai should pass through unchanged")
	}
}
