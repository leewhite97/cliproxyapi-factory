package provider

import (
	"strings"
	"testing"
)

func TestModels(t *testing.T) {
	r := Models()
	if r.Provider != "factory" || len(r.Models) == 0 {
		t.Fatal("no models")
	}
	seen := map[string]bool{}
	for _, m := range r.Models {
		if !strings.HasPrefix(m.ID, "factory/") || seen[m.ID] {
			t.Fatal("model collision")
		}
		seen[m.ID] = true
	}
	if !seen["factory/claude-sonnet-4-6"] || !seen["factory/gpt-5.3-codex"] || !seen["factory/glm-5.2"] {
		t.Fatal("missing route family")
	}
}
