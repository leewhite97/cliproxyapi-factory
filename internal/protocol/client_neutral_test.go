package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGatewayClientNeutralDefault(t *testing.T) {
	identity := "You are a custom AI assistant operating inside a generic developer client."
	summary := "The conversation history before this point was compacted into the following summary:\n\n<summary>\nkeep body\n</summary>"
	for _, format := range []string{"openai", "claude", "openai-response"} {
		field, top := "messages", "system"
		if format == "openai-response" {
			field, top = "input", "instructions"
		}
		body, _ := json.Marshal(map[string]any{top: identity, field: []any{map[string]any{"role": "user", "content": summary}}})
		wire, err := GatewayRequest(body, format)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		json.Unmarshal(wire, &result)
		if !strings.Contains(string(wire), identity) {
			t.Fatal("client identity text changed")
		}
		parts := result[field].([]any)
		if format == "openai" {
			parts = parts[1:]
		}
		content := parts[0].(map[string]any)["content"].(string)
		if !strings.Contains(content, summary) {
			t.Fatal("user summary changed")
		}
	}
}
