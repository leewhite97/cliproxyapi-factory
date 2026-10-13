package protocol

import (
	"testing"
)

func TestRoute(t *testing.T) {
	for _, c := range []struct{ id, format, path, provider string }{{"claude-sonnet-4-6", "claude", "/api/llm/a/v1/messages", "anthropic"}, {"gpt-5.3-codex", "openai-response", "/api/llm/o/v1/responses", "openai"}, {"glm-5.1", "openai", "/api/llm/o/v1/chat/completions", "fireworks"}, {"minimax-m3", "claude", "/api/llm/a/v1/messages", "fireworks"}} {
		t.Run(c.id, func(t *testing.T) {
			r, e := Resolve(c.id)
			if e != nil || r.Format != c.format || r.Path != c.path || r.Provider != c.provider {
				t.Fatalf("got %+v %v", r, e)
			}
		})
	}
	if _, e := Resolve("unknown-model"); e == nil {
		t.Fatal("unknown model accepted")
	}
}
