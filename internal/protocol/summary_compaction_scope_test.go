package protocol

import (
	"encoding/json"
	"testing"
)

func TestGatewayRequestPreservesAllSummaryRoles(t *testing.T) {
	const intro = "The conversation history before this point was compacted into the following summary:"
	envelope := intro + "\n\n<summary>\nunchanged body\n</summary>"
	for _, format := range []string{"claude", "openai", "openai-response"} {
		t.Run(format, func(t *testing.T) {
			field, blockType := "messages", "text"
			if format == "openai-response" {
				field, blockType = "input", "input_text"
			}
			original := []any{
				map[string]any{"role": "user", "content": envelope},
				map[string]any{"role": "assistant", "content": envelope},
				map[string]any{"role": "user", "content": []any{
					map[string]any{"type": blockType, "text": "Quoted: " + envelope},
					map[string]any{"type": blockType, "text": intro},
					map[string]any{"type": "tool_result", "content": envelope, "tool_use_id": "fixture"},
				}},
			}
			body, _ := json.Marshal(map[string]any{field: original})
			got, err := GatewayRequest(body, format)
			if err != nil {
				t.Fatal(err)
			}
			var d map[string]any
			if err := json.Unmarshal(got, &d); err != nil {
				t.Fatal(err)
			}
			messages := d[field].([]any)
			if format == "openai" {
				messages = messages[1:]
			}
			if messages[0].(map[string]any)["content"] != envelope {
				t.Fatal("summary envelope changed")
			}
			wantUntouched, _ := json.Marshal(original[1:])
			gotUntouched, _ := json.Marshal(messages[1:])
			if string(wantUntouched) != string(gotUntouched) {
				t.Fatal("assistant text, quoted text, plain text, or tool results changed")
			}
		})
	}
}
