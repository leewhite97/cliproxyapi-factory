package provider

import (
	"context"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"strings"
	"testing"
	"time"
)

func TestChatSSEHostContract(t *testing.T) {
	f := &streamFake{raw: []byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"OK\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"), ended: make(chan string, 1)}
	s := Service{Host: f}
	_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "glm-5.1", Format: "openai", Payload: []byte(`{"messages":[{"role":"user","content":"hi"}]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-f.ended:
		if strings.Contains(string(f.out), "data:") || strings.Contains(string(f.out), "[DONE]") {
			t.Fatal("CPA chat handler adds SSE envelope; plugin must emit JSON chunks")
		}
	case <-time.After(time.Second):
		t.Fatal("hung")
	}
}
