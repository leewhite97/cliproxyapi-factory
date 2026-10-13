package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHeaders(t *testing.T) {
	r, _ := Resolve("gpt-5.3-codex")
	h := Headers("fk-test-secret", "org-example", r)
	h2 := Headers("fk-test-secret", "", r)
	for k, v := range map[string]string{"Authorization": "Bearer fk-test-secret", "X-Factory-Client": "cli", "X-Client-Version": "0.162.1", "x-api-provider": "openai", "X-Factory-Org-Id": "org-example", "OpenAI-Platform": "org-bHuLtG1fGmYk5YaOihAAXFBw"} {
		if h.Get(k) != v {
			t.Errorf("missing header %s", k)
		}
	}
	if h.Get("x-session-id") == "" || h.Get("x-session-id") == h2.Get("x-session-id") || h.Get("x-assistant-message-id") == h2.Get("x-assistant-message-id") {
		t.Fatal("IDs missing or reused")
	}
	if h2.Get("X-Factory-Org-Id") != "" {
		t.Fatal("unexpected org")
	}
}
func TestFold(t *testing.T) {
	for _, c := range []struct{ format, body, field string }{{"claude", `{"system":[{"type":"text","text":"rules"}],"messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","data":"abc"}}]}],"tools":[{"name":"get_weather"}],"thinking":{"type":"enabled","budget_tokens":1024}}`, "system"}, {"openai", `{"messages":[{"role":"system","content":"rules"},{"role":"user","content":"Hi"}],"reasoning_effort":"high"}`, "messages"}, {"openai-response", `{"instructions":"rules","input":[{"role":"developer","content":"extra rules"},{"role":"user","content":[{"type":"input_image","image_url":"image"}]}],"reasoning":{"effort":"high"}}`, "instructions"}} {
		t.Run(c.format, func(t *testing.T) {
			b, e := Fold([]byte(c.body), c.format)
			if e != nil {
				t.Fatal(e)
			}
			if !json.Valid(b) || !strings.Contains(string(b), "<system>") {
				t.Fatalf("not folded: %s", b)
			}
			var d map[string]any
			json.Unmarshal(b, &d)
			if c.field != "messages" && d[c.field] != nil {
				t.Fatal("system field remained")
			}
			for _, x := range []string{"get_weather", "budget_tokens", "input_image", "image_url", "reasoning_effort"} {
				if strings.Contains(c.body, x) && !strings.Contains(string(b), x) {
					t.Fatalf("lost %s", x)
				}
			}
		})
	}
}
