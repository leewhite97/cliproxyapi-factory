package provider

import (
	"strings"
	"testing"
)

func TestModelDocs(t *testing.T) {
	ids, e := ParseModelDocs([]byte("| Model | Model ID | Multiplier |\n| Claude | `claude-new` | 1x |\n| GPT | `gpt-new` | 1x |\n| GLM | `glm-new` | 1x |\n| MiniMax | `minimax-new` | 1x |\n| Mystery | `unknown-new` | 1x |\n| Duplicate | `gpt-new` | 1x |"))
	if e != nil || strings.Join(ids, ",") != "claude-new,gpt-new,glm-new,minimax-new" {
		t.Fatalf("bad docs parse %v %v", ids, e)
	}
	if _, e := ParseModelDocs([]byte("malformed document")); e == nil {
		t.Fatal("zero-model discovery treated as success")
	}
}
