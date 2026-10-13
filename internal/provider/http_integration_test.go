package provider

import (
	"context"
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fixtureHTTP struct{ server *httptest.Server }

func (h fixtureHTTP) Do(ctx context.Context, method, callback, url string, headers http.Header, body []byte) (int, http.Header, []byte, error) {
	u := strings.Replace(url, "https://api.factory.ai", h.server.URL, 1)
	r, e := http.NewRequestWithContext(ctx, "POST", u, strings.NewReader(string(body)))
	if e != nil {
		return 0, nil, nil, e
	}
	r.Header = headers
	resp, e := h.server.Client().Do(r)
	if e != nil {
		return 0, nil, nil, e
	}
	defer resp.Body.Close()
	b, e := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, b, e
}
func TestHTTPIntegrationRoutes(t *testing.T) {
	cases := []struct{ model, format, path, wire string }{{"claude-sonnet-4-6", "claude", "/api/llm/a/v1/messages", `{"id":"m1","type":"message","role":"assistant","content":[{"type":"text","text":"OK"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":1}}`}, {"gpt-5.3-codex", "openai-response", "/api/llm/o/v1/responses", `{"id":"r1","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"OK"}]}],"usage":{"input_tokens":3,"output_tokens":1}}`}, {"deepseek-v4-pro", "openai", "/api/llm/o/v1/chat/completions", `{"id":"c1","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`}}
	for _, c := range cases {
		t.Run(c.format, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != c.path || r.Method != "POST" {
					t.Error("wrong endpoint")
				}
				if r.Header.Get("Authorization") != "Bearer fk-test-secret" {
					t.Error("missing auth")
				}
				var d map[string]any
				if json.NewDecoder(r.Body).Decode(&d) != nil {
					t.Error("bad request JSON")
				}
				if d["model"] != c.model {
					t.Error("model prefix not stripped")
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, c.wire)
			}))
			defer server.Close()
			s := Service{Host: fixtureHTTP{server}}
			body := []byte(`{"messages":[{"role":"user","content":"hi"}],"input":"hi","max_tokens":128}`)
			res, e := s.Execute(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "factory/" + c.model, Format: c.format, Payload: body, OriginalRequest: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}})
			if e != nil || !strings.Contains(string(res.Payload), "OK") {
				t.Fatalf("HTTP integration failed %v", e)
			}
		})
	}
}
func TestHTTPErrors(t *testing.T) {
	for _, status := range []int{401, 403, 404, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				io.WriteString(w, `{"error":"fk-test-secret"}`)
			}))
			defer server.Close()
			s := Service{Host: fixtureHTTP{server}}
			_, e := s.Execute(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "factory/claude-sonnet-4-6", Format: "claude", Payload: []byte(`{"messages":[]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}})
			se, ok := e.(*StatusError)
			if !ok || se.Status != status || strings.Contains(se.Message, "fk-test-secret") || se.Retryable != (status == 429 || status >= 500) {
				t.Fatalf("incorrect error classification %v", e)
			}
		})
	}
}
func TestHTTPClientCancellation(t *testing.T) {
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	s := Service{Host: fixtureHTTP{server}}
	done := make(chan error, 1)
	go func() {
		_, e := s.Execute(ctx, Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "claude-sonnet-4-6", Format: "claude", Payload: []byte(`{"messages":[]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}})
		done <- e
	}()
	<-entered
	cancel()
	if <-done == nil {
		t.Fatal("cancelled request succeeded")
	}
	<-cancelled
}
