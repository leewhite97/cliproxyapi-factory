package provider

import (
	"context"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"sync"
	"testing"
	"time"
)

type blockingHost struct {
	streamFake
	entered  chan struct{}
	released chan struct{}
	once     sync.Once
}

func (f *blockingHost) Read(ctx context.Context, id string) (Chunk, error) {
	close(f.entered)
	<-f.released
	return Chunk{Done: true}, nil
}
func (f *blockingHost) Close(context.Context, string) error {
	f.once.Do(func() { close(f.released) })
	return nil
}
func TestShutdownDrainsStream(t *testing.T) {
	f := &blockingHost{streamFake: streamFake{ended: make(chan string, 1)}, entered: make(chan struct{}), released: make(chan struct{})}
	s := Service{Host: f}
	_, e := s.ExecuteStream(context.Background(), Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "deepseek-v4-pro", Format: "openai", Payload: []byte(`{"messages":[]}`), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-secret"}`)}, StreamID: "out"})
	if e != nil {
		t.Fatal(e)
	}
	<-f.entered
	done := make(chan struct{})
	go func() { s.Shutdown(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		f.Close(context.Background(), "up")
		<-done
		t.Fatal("shutdown did not close blocking upstream")
	}
	select {
	case <-f.ended:
	case <-time.After(time.Second):
		t.Fatal("shutdown leaked upstream stream worker")
	}
}
