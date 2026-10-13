package provider

import (
	"context"
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"net/http"
	"testing"
)

type fakeHost struct {
	t          *testing.T
	path, wire string
	status     int
}

func (f fakeHost) Do(ctx context.Context, method, callback, url string, h http.Header, b []byte) (int, http.Header, []byte, error) {
	if url != "https://api.factory.ai"+f.path {
		f.t.Fatalf("wrong endpoint %s", url)
	}
	if h.Get("Authorization") != "Bearer fk-test-secret" {
		f.t.Fatal("missing auth")
	}
	var d map[string]any
	json.Unmarshal(b, &d)
	prefix := "You are Droid, an AI software engineering agent built by Factory."
	if (d["system"] != nil && d["system"] != prefix) || (d["instructions"] != nil && d["instructions"] != prefix) {
		f.t.Fatal("custom system not folded, or fixed identity incorrect")
	}
	return f.status, http.Header{}, []byte(f.wire), nil
}
func TestExecuteRoutes(t *testing.T) {
	for _, c := range []struct{ model, path, format, wire string }{{"claude-sonnet-4-6", "/api/llm/a/v1/messages", "claude", `{"id":"msg1","type":"message","role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"OK"}],"stop_reason":"end_turn","usage":{"input_tokens":5,"output_tokens":1}}`}, {"gpt-5.3-codex", "/api/llm/o/v1/responses", "openai-response", `{"id":"resp1","object":"response","status":"completed","output":[{"id":"m1","type":"message","role":"assistant","content":[{"type":"output_text","text":"OK","annotations":[]}]}],"usage":{"input_tokens":5,"output_tokens":1,"total_tokens":6}}`}, {"glm-5.1", "/api/llm/o/v1/chat/completions", "openai", `{"id":"chat1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`}} {
		t.Run(c.model, func(t *testing.T) {
			s := Service{Host: fakeHost{t, c.path, c.wire, 200}}
			body := []byte(`{"messages":[{"role":"system","content":"rules"},{"role":"user","content":"hi"}],"max_tokens":128}`)
			if c.format == "claude" {
				body = []byte(`{"system":"rules","messages":[{"role":"user","content":"hi"}],"max_tokens":128}`)
			}
			if c.format == "openai-response" {
				body = []byte(`{"instructions":"rules","input":"hi"}`)
			}
			out, e := s.Execute(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: c.model, Format: c.format, SourceFormat: c.format, Payload: body, OriginalRequest: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}})
			if e != nil || !json.Valid(out.Payload) {
				t.Fatalf("execute failed %v", e)
			}
		})
	}
}
