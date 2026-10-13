package provider

import (
	"context"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"strings"
	"testing"
)

func TestParseCredential(t *testing.T) {
	r, e := ParseAuth(pluginapi.AuthParseRequest{FileName: "factory-test.json", RawJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)})
	if e != nil || !r.Handled || r.Auth.Provider != "factory" || r.Auth.ID != "factory-test.json" {
		t.Fatalf("bad parse: %v", e)
	}
	for _, v := range r.Auth.Metadata {
		if strings.Contains(v.(string), "fk-test-secret") {
			t.Fatal("secret metadata")
		}
	}
	if r.Auth.Attributes["auth_kind"] != "api_key" {
		t.Fatal("wrong credential class")
	}
	r, e = ParseAuth(pluginapi.AuthParseRequest{RawJSON: []byte(`{"type":"codex","access_token":"other"}`)})
	if e != nil || r.Handled {
		t.Fatal("captured native auth")
	}
}
func TestToolTranslation(t *testing.T) {
	for _, route := range []string{"claude", "openai-response", "openai"} {
		b, e := translateRequest("openai", route, "test-model", []byte(`{"messages":[{"role":"user","content":"weather?"}],"tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}],"tool_choice":{"type":"function","function":{"name":"get_weather"}}}`), false)
		if e != nil || !strings.Contains(string(b), "get_weather") || !strings.Contains(string(b), "city") {
			t.Fatalf("tools lost on %s: %v", route, e)
		}
	}
}
func TestErrorMapping(t *testing.T) {
	for _, status := range []int{401, 403, 404, 429, 500, 503} {
		e := upstreamError(status, []byte(`{"error":"fk-test-secret"}`), "fk-test-secret")
		if e.Status != status || strings.Contains(e.Message, "fk-test-secret") || e.Retryable != (status == 429 || status >= 500) {
			t.Fatalf("bad mapping %d", status)
		}
	}
}

var _ = context.Background
