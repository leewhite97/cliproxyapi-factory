package provider

import (
	"context"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"strings"
	"testing"
	"time"
)

func TestClaudeStreamToOpenAI(t *testing.T) {
	wire := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"model\":\"claude-sonnet-4-6\",\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	f := &streamFake{raw: []byte(wire), ended: make(chan string, 1)}
	s := Service{Host: f}
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "claude-sonnet-4-6", Format: "openai", Payload: body, OriginalRequest: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-f.ended:
		if e != "" || !strings.Contains(string(f.out), "Hello") {
			t.Fatalf("translated stream lost: %s %s", e, f.out)
		}
	case <-time.After(time.Second):
		t.Fatal("hung")
	}
}

func TestOpenAIResponseIncompleteToOpenAI(t *testing.T) {
	wire := "event: response.text.delta\ndata: {\"type\":\"response.text.delta\",\"delta\":\"partial text\"}\n\nevent: response.incomplete\ndata: {\"type\":\"response.incomplete\",\"response\":{\"id\":\"r1\",\"model\":\"gpt-5.3-codex\",\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"},\"usage\":{\"input_tokens\":10,\"output_tokens\":20,\"total_tokens\":30}}}\n\n"
	f := &streamFake{raw: []byte(wire), ended: make(chan string, 1)}
	s := Service{Host: f}
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "gpt-5.3-codex", Format: "openai", Payload: body, OriginalRequest: body, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-f.ended:
		t.Logf("ended error: %q", e)
		t.Logf("emitted chunks: %s", string(f.out))
		if e != "" {
			t.Fatalf("expected clean termination on response.incomplete, got: %s", e)
		}
		if !strings.Contains(string(f.out), "length") {
			t.Errorf("expected finish_reason length in output, got: %s", string(f.out))
		}
	case <-time.After(time.Second):
		t.Fatal("hung")
	}
}
