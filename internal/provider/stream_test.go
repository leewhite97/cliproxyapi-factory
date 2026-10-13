package provider

import (
	"context"
	"fmt"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"net/http"
	"strings"
	"testing"
	"time"
)

type streamFake struct {
	fakeHost
	raw      []byte
	pos      int
	out      []byte
	ended    chan string
	failEmit bool
	closed   bool
}

func (f *streamFake) Open(ctx context.Context, c, u string, h http.Header, b []byte) (Stream, error) {
	return Stream{200, http.Header{}, "up"}, nil
}
func (f *streamFake) Read(context.Context, string) (Chunk, error) {
	if f.pos >= len(f.raw) {
		return Chunk{Done: true}, nil
	}
	b := f.raw[f.pos]
	f.pos++
	return Chunk{Payload: []byte{b}}, nil
}
func (f *streamFake) Close(context.Context, string) error { f.closed = true; return nil }
func (f *streamFake) Emit(ctx context.Context, id string, b []byte) error {
	if f.failEmit {
		return fmt.Errorf("client disconnected")
	}
	f.out = append(f.out, b...)
	return nil
}
func (f *streamFake) End(ctx context.Context, id, e string) { f.ended <- e }
func TestStreamRoutes(t *testing.T) {
	for _, c := range []struct{ model, format, wire string }{{"claude-sonnet-4-6", "claude", "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"}, {"gpt-5.3-codex", "openai-response", "event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n"}, {"glm-5.1", "openai", "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"OK\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"}} {
		t.Run(c.model, func(t *testing.T) {
			for _, disconnect := range []bool{false, true} {
				f := &streamFake{raw: []byte(c.wire), ended: make(chan string, 1), failEmit: disconnect}
				s := Service{Host: f}
				_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: c.model, Format: c.format, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`), Payload: []byte(`{"messages":[{"role":"user","content":"hi"}],"input":"hi"}`)}, StreamID: "out"})
				if e != nil {
					t.Fatal(e)
				}
				select {
				case e := <-f.ended:
					if disconnect && !strings.Contains(e, "disconnect") {
						t.Fatal("cancel lost")
					}
					expected := c.wire
					if c.format == "openai" {
						expected = strings.TrimPrefix(strings.Split(c.wire, "\n\n")[0], "data: ")
					}
					if !disconnect && (e != "" || string(f.out) != expected) {
						t.Fatalf("lost stream %s %s", e, f.out)
					}
					if !f.closed {
						t.Fatal("upstream not closed")
					}
				case <-time.After(time.Second):
					t.Fatal("stream hung")
				}
			}
		})
	}
}
