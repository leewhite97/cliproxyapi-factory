package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFoldPreservesClientIdentity(t *testing.T) {
	const identity = "You are an expert coding assistant operating inside a developer client harness."
	const context = identity
	const rest = " You help users by reading files, executing commands, editing code, and writing new files.\n\nNever change project secrets."
	for _, format := range []string{"openai", "claude", "openai-response"} {
		t.Run(format, func(t *testing.T) {
			d := map[string]any{"messages": []any{map[string]any{"role": "developer", "content": identity + rest}, map[string]any{"role": "user", "content": "hi"}}, "max_tokens": 64000, "tools": []any{map[string]any{"name": "example"}}}
			if format == "claude" {
				d["system"] = identity + rest
				d["messages"] = []any{map[string]any{"role": "user", "content": "hi"}}
			}
			if format == "openai-response" {
				d["instructions"] = identity + rest
				d["input"] = []any{map[string]any{"role": "user", "content": "hi"}}
				delete(d, "messages")
			}
			raw, _ := json.Marshal(d)
			wire, err := Fold(raw, format)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			json.Unmarshal(wire, &got)
			field := "messages"
			if format == "openai-response" {
				field = "input"
			}
			content := got[field].([]any)[0].(map[string]any)["content"].(string)
			if content != "<system>\n"+context+rest+"\n</system>\n\nhi" {
				t.Fatalf("identity not contextualized while retaining instructions: %q", content)
			}
			if len(got["tools"].([]any)) != 1 || got["max_tokens"].(float64) != 64000 {
				t.Fatal("unrelated request fields changed")
			}
		})
	}
	// Never rewrite user input or quoted identity inside other system instructions.
	raw, _ := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "system", "content": "Quoted example: " + identity}, map[string]any{"role": "user", "content": identity}}})
	wire, err := Fold(raw, "openai")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(wire), identity) != 2 {
		t.Fatal("rewrote non-leading or user identity")
	}
}
