package provider

import (
	"context"
	"fmt"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogLastSuccess(t *testing.T) {
	now := time.Unix(1000, 0)
	fail := false
	calls := 0
	p := filepath.Join(t.TempDir(), "catalog.json")
	c := CatalogCache{Path: p, Now: func() time.Time { return now }, Fetch: func(context.Context) ([]byte, error) {
		calls++
		if fail {
			return nil, fmt.Errorf("offline")
		}
		return []byte("| New | `gpt-future` | 1x |"), nil
	}}
	has := func(r pluginapi.ModelResponse) bool {
		for _, m := range r.Models {
			if m.ID == "factory/gpt-future" {
				return true
			}
		}
		return false
	}
	if !has(c.Get(context.Background())) {
		t.Fatal("discovery ignored")
	}
	c.Get(context.Background())
	if calls != 1 {
		t.Fatal("TTL ignored")
	}
	now = now.Add(25 * time.Hour)
	fail = true
	if !has(c.Get(context.Background())) {
		t.Fatal("failure discarded successful catalog")
	}
	fresh := CatalogCache{Path: p, Now: func() time.Time { return now }, Fetch: func(context.Context) ([]byte, error) { return nil, fmt.Errorf("offline") }}
	if !has(fresh.Get(context.Background())) {
		t.Fatal("disk cache not restored")
	}
}
