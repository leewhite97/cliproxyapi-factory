package provider

import "testing"

func TestCatalogExcludesDeprecatedExactIDs(t *testing.T) {
	deprecated := []string{"claude-opus-4-6-fast", "claude-opus-4-7-fast", "glm-5.1", "kimi-k2.5", "kimi-k2.6", "kimi-k2.7-code"}
	ids := append(append([]string{}, deprecated...), "glm-5.2", "glm-5.2-fast", "gpt-future-audit")
	for _, response := range []struct {
		name   string
		models map[string]bool
	}{{"static", map[string]bool{}}, {"merged", map[string]bool{}}} {
		result := Models()
		if response.name == "merged" {
			result = mergedCatalog(ids)
		}
		for _, m := range result.Models {
			response.models[m.ID] = true
		}
		for _, id := range deprecated {
			if response.models["factory/"+id] {
				t.Fatalf("%s still advertises deprecated %s", response.name, id)
			}
		}
		if response.name == "merged" && (!response.models["factory/glm-5.2"] || !response.models["factory/glm-5.2-fast"] || !response.models["factory/gpt-future-audit"]) {
			t.Fatal("region-independent discovery was damaged")
		}
	}
}
