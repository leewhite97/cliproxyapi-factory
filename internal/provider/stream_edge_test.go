package provider

import (
	"context"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"strings"
	"testing"
	"time"
)

func TestMultiLineChatEvent(t *testing.T) {
	f := &streamFake{raw: []byte("data: {\"choices\":\ndata: [{\"index\":0,\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n"), ended: make(chan string, 1)}
	s := Service{Host: f}
	_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "deepseek-v4-pro", Format: "openai", Payload: []byte(`{"messages":[]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-f.ended:
		if e != "" {
			t.Fatal(e)
		}
		if strings.Contains(string(f.out), "\n") {
			t.Fatal("multiline JSON breaks CPA chat SSE envelope")
		}
	case <-time.After(time.Second):
		t.Fatal("stream worker leaked")
	}
}
func TestOpenAIStreamErrorEventStopsForwarding(t *testing.T) {
	f := &streamFake{raw: []byte("data: {\"error\":{\"message\":\"denied fk-test-secret\",\"type\":\"permission_error\"}}\n\ndata: [DONE]\n\n"), ended: make(chan string, 1)}
	s := Service{Host: f}
	_, err := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "deepseek-v4-pro", Format: "openai", Payload: []byte(`{"messages":[]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-f.ended:
		if !strings.Contains(err, "upstream stream error") {
			t.Fatalf("SSE error was not reported as a failure: %q", err)
		}
		if strings.Contains(err, "fk-test-secret") {
			t.Fatal("credential leaked in error")
		}
		if len(f.out) != 0 {
			t.Fatalf("SSE error forwarded as normal content: %s", f.out)
		}
	case <-time.After(time.Second):
		t.Fatal("stream worker leaked")
	}
}

func TestInterruptedStreamCloses(t *testing.T) {
	for i := 0; i < 20; i++ {
		f := &streamFake{raw: []byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"), ended: make(chan string, 1)}
		s := Service{Host: f}
		_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "deepseek-v4-pro", Format: "openai", Payload: []byte(`{"messages":[]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
		if e != nil {
			t.Fatal(e)
		}
		select {
		case e := <-f.ended:
			if !strings.Contains(e, "interrupted") || !f.closed {
				t.Fatal("disconnect not propagated/closed")
			}
		case <-time.After(time.Second):
			t.Fatal("stream worker leaked")
		}
	}
}
