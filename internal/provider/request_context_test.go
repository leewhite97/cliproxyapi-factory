package provider

import (
	"context"
	"encoding/json"
	"factory.local/cliproxyapi-factory/internal/protocol"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"net/http"
	"strings"
	"testing"
)

func TestCanonicalSessionHeaders(t *testing.T) {
	r := Request{ExecutorRequest: pluginapi.ExecutorRequest{Metadata: map[string]any{"canonical_session_id": "session-123"}}}
	route, _ := protocol.Resolve("deepseek-v4.1-flash")
	a := protocol.HeadersWithSession("fk-test", "", route, requestSession(r))
	b := protocol.HeadersWithSession("fk-test", "", route, requestSession(r))
	if a.Get("x-session-id") != "session-123" || b.Get("x-session-id") != "session-123" || a.Get("x-assistant-message-id") == b.Get("x-assistant-message-id") {
		t.Fatal("session/message IDs not independent")
	}
	r.Metadata["canonical_session_id"] = "bad\r\nheader"
	if requestSession(r) != "" {
		t.Fatal("invalid host session accepted")
	}
}
func TestSameFormatTranslationPreservesLargeInteger(t *testing.T) {
	b, e := translateRequest("openai", "openai", "deepseek-v4.1-flash", []byte(`{"messages":[],"metadata":{"integer":9007199254740993}}`), false)
	if e != nil || !strings.Contains(string(b), "9007199254740993") {
		t.Fatal("large number lost")
	}
}
func TestPrepareUsesExplicitCompatibility(t *testing.T) {
	s := Service{Compatibility: func() protocol.Compatibility { return protocol.Compatibility{Mode: "passthrough"} }}
	raw := []byte(`{"messages":[{"role":"system","content":"Original"},{"role":"user","content":"hello"}]}`)
	_, _, _, b, e := s.prepare(Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "deepseek-v4.1-flash", Format: "openai", Payload: raw, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test"}`)}}, false)
	if e != nil {
		t.Fatal(e)
	}
	var d map[string]any
	json.Unmarshal(b, &d)
	if !strings.Contains(string(b), "Original") || strings.Contains(string(b), protocol.DroidPrefix) {
		t.Fatal("passthrough not respected")
	}
}
func TestCancelledRequestIsNotRetryable(t *testing.T) {
	e := transportError(context.Canceled, "fk-test")
	if e.Retryable || e.Code != "request_cancelled" {
		t.Fatal("cancellation treated as retryable upstream failure")
	}
	_ = http.MethodPost
}
