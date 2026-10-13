package provider

import (
	"context"
	"testing"
	"time"
)

func TestCatalogReadersDoNotWaitForRefresh(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	c := CatalogCache{Fetch: func(context.Context) ([]byte, error) {
		close(entered)
		<-release
		return []byte("| Future | `gpt-future` | text |"), nil
	}}
	done := make(chan struct{})
	go func() { c.Get(context.Background()); close(done) }()
	<-entered
	read := make(chan struct{})
	go func() { c.Get(context.Background()); close(read) }()
	select {
	case <-read:
	case <-time.After(100 * time.Millisecond):
		close(release)
		<-done
		t.Fatal("reader blocked by network refresh")
	}
	close(release)
	<-done
}
