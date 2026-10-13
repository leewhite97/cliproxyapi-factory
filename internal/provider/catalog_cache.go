package provider

import (
	"context"
	"encoding/json"
	"factory.local/cliproxyapi-factory/internal/protocol"
	"fmt"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type CatalogCache struct {
	Path   string
	Fetch  func(context.Context) ([]byte, error)
	Now    func() time.Time
	mu     sync.Mutex
	loaded bool
	ids    []string
	next   time.Time
}

func (c *CatalogCache) Get(ctx context.Context) pluginapi.ModelResponse {
	c.mu.Lock()

	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	if !c.loaded {
		c.loaded = true
		if b, e := os.ReadFile(c.Path); e == nil && len(b) < 2*1024*1024 {
			var ids []string
			if json.Unmarshal(b, &ids) == nil {
				for _, id := range ids {
					if _, e := protocol.Resolve(id); e == nil {
						c.ids = append(c.ids, id)
					}
				}
			}
		}
	}
	if now.Before(c.next) {
		ids := append([]string(nil), c.ids...)
		c.mu.Unlock()
		return mergedCatalog(ids)
	}
	c.next = now.Add(5 * time.Minute)
	fetch := c.Fetch
	if fetch == nil {
		fetch = FetchModelDocs
	}
	previous := append([]string(nil), c.ids...)
	path := c.Path
	c.mu.Unlock()
	bounded, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	b, e := fetch(bounded)
	if e != nil {
		return mergedCatalog(previous)
	}
	ids, e := ParseModelDocs(b)
	if e != nil {
		return mergedCatalog(previous)
	}
	c.mu.Lock()
	c.ids = ids
	c.next = now.Add(24 * time.Hour)
	c.mu.Unlock()
	if path != "" {
		data, _ := json.Marshal(ids)
		if f, e := os.CreateTemp(filepath.Dir(path), ".factory-catalog-*"); e == nil {
			name := f.Name()
			_, errWrite := f.Write(data)
			errClose := f.Close()
			if errWrite == nil && errClose == nil {
				os.Rename(name, path)
			}
			os.Remove(name)
		}
	}
	return mergedCatalog(ids)
}

// Public discovery is independent of credential-bearing inference transport.
// It is bounded and carries no Authorization, cookies, or account metadata.
func FetchModelDocs(ctx context.Context) ([]byte, error) {
	r, e := http.NewRequestWithContext(ctx, "GET", "https://docs.factory.ai/models.md", nil)
	if e != nil {
		return nil, e
	}
	r.Header.Set("Accept", "text/markdown,text/plain;q=0.9")
	client := http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 || (r.URL.Hostname() != "docs.factory.ai" && r.URL.Hostname() != "docs.factory.com") || r.URL.Scheme != "https" {
			return fmt.Errorf("factory: unsafe catalog redirect")
		}
		return nil
	}}
	resp, e := client.Do(r)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("factory: public model catalog HTTP %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if len(b) > 2*1024*1024 {
		return nil, fmt.Errorf("factory: catalog too large")
	}
	return b, e
}
