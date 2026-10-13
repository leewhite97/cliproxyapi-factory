package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func TestCapture403BoundedPrivateAndRedacted(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FACTORY_DIAGNOSTIC_DIR", dir)
	marker := []byte(`{"expires_at":"` + time.Now().Add(20*time.Minute).UTC().Format(time.RFC3339Nano) + `"}`)
	if err := os.WriteFile(filepath.Join(dir, "capture-control.json"), marker, 0600); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"messages":[{"role":"user","content":"hello fk-test-secret Bearer client-secret"}],"api_key":"private-client-key"}`)
	r := Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "factory/deepseek-v4.1-flash", Format: "openai", SourceFormat: "openai", Payload: body, OriginalRequest: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}}
	s := Service{Host: fakeHost{t, "/api/llm/o/v1/chat/completions", `{"detail":"Forbidden"}`, 403}}
	if _, err := s.Execute(context.Background(), r); err == nil {
		t.Fatal("403 should still fail")
	}
	path := filepath.Join(dir, "factory-403-capture.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("403 capture absent: %v", err)
	}
	if !json.Valid(b) {
		t.Fatal("capture invalid JSON")
	}
	for _, secret := range []string{"fk-test-secret", "client-secret", "private-client-key"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("capture leaked %s", secret)
		}
	}
	var d map[string]json.RawMessage
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"original_request", "plugin_payload", "upstream_request", "model", "route", "captured_at"} {
		if len(d[k]) == 0 {
			t.Fatalf("missing %s", k)
		}
	}
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0600 {
		t.Fatal("capture permissions")
	}
	before := string(b)
	s.Execute(context.Background(), r)
	b, _ = os.ReadFile(path)
	if string(b) != before {
		t.Fatal("second capture overwrote first")
	}
}

type captureDeniedStream struct{ streamFake }

func (f *captureDeniedStream) Open(ctx context.Context, callback, url string, h http.Header, b []byte) (Stream, error) {
	return Stream{Status: 403, ID: "capture-denied"}, nil
}

func TestCaptureStreaming403(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FACTORY_DIAGNOSTIC_DIR", dir)
	marker := []byte(`{"expires_at":"` + time.Now().Add(20*time.Minute).UTC().Format(time.RFC3339Nano) + `"}`)
	os.WriteFile(filepath.Join(dir, "capture-control.json"), marker, 0600)
	f := &captureDeniedStream{streamFake: streamFake{raw: []byte(`{"detail":"Forbidden"}`)}}
	s := Service{Host: f}
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	_, err := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "factory/deepseek-v4.1-flash", Format: "openai", Payload: body, OriginalRequest: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "capture-out"})
	if err == nil || !f.closed {
		t.Fatal("streaming 403 not failed/closed")
	}
	b, err := os.ReadFile(filepath.Join(dir, "factory-403-capture.json"))
	if err != nil || !json.Valid(b) {
		t.Fatalf("streaming capture failed: %v", err)
	}
}

func TestCaptureRequiresActiveMarker(t *testing.T) {
	for _, expiry := range []time.Duration{-time.Minute, 31 * time.Minute} {
		dir := t.TempDir()
		t.Setenv("FACTORY_DIAGNOSTIC_DIR", dir)
		marker := []byte(`{"expires_at":"` + time.Now().Add(expiry).UTC().Format(time.RFC3339Nano) + `"}`)
		os.WriteFile(filepath.Join(dir, "capture-control.json"), marker, 0600)
		s := Service{Host: fakeHost{t, "/api/llm/o/v1/chat/completions", `{"detail":"Forbidden"}`, 403}}
		body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
		s.Execute(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "deepseek-v4.1-flash", Format: "openai", Payload: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}})
		if _, err := os.Stat(filepath.Join(dir, "factory-403-capture.json")); !os.IsNotExist(err) {
			t.Fatal("capture enabled outside approved window")
		}
	}
}
