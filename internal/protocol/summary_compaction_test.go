package protocol

import (
	"encoding/json"
	"testing"
)

func TestGatewayRequestPreservesSummaryEnvelope(t *testing.T) {
	const intro = "The conversation history before this point was compacted into the following summary:"
	content := intro + "\n\n<summary>\nKeep all decisions, source code, and private paths exactly.\n</summary>\n"
	body, _ := json.Marshal(map[string]any{
		"model": "claude-opus-5-5", "system": "Keep custom instructions.",
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": content},
			map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": "image-fixture"}},
		}}},
		"thinking": map[string]any{"type": "adaptive"}, "output_config": map[string]any{"effort": "high"},
		"max_tokens": 128000, "tools": []any{map[string]any{"name": "read", "input_schema": map[string]any{"type": "object"}}},
	})
	got, err := GatewayRequest(body, "claude")
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]any
	if err := json.Unmarshal(got, &d); err != nil {
		t.Fatal(err)
	}
	blocks := d["messages"].([]any)[0].(map[string]any)["content"].([]any)
	found := false
	for _, value := range blocks {
		block := value.(map[string]any)
		if block["type"] == "text" && block["text"] == content {
			found = true
		}
	}
	if !found {
		t.Fatal("summary text changed")
	}
	if d["system"] != DroidPrefix || d["max_tokens"] != float64(128000) || len(d["tools"].([]any)) != 1 {
		t.Fatal("unrelated fields changed")
	}
	if d["thinking"].(map[string]any)["type"] != "adaptive" || d["output_config"].(map[string]any)["effort"] != "high" {
		t.Fatal("thinking intent changed")
	}
	image := blocks[len(blocks)-1].(map[string]any)
	if image["source"].(map[string]any)["data"] != "image-fixture" {
		t.Fatal("image changed")
	}
}
