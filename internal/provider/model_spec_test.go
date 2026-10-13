package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiscoveredModelDoesNotInventCapabilities(t *testing.T) {
	for _, m := range mergedCatalog([]string{"gpt-future", "claude-future"}).Models {
		if strings.HasSuffix(m.ID, "-future") && (m.ContextLength != 0 || m.MaxCompletionTokens != 0 || m.Thinking != nil || len(m.SupportedInputModalities) != 1 || m.SupportedInputModalities[0] != "text") {
			t.Fatalf("invented capability: %+v", m)
		}
	}
}
func TestSpecificReasoningDeclarationsMatchValidation(t *testing.T) {
	for _, m := range mergedCatalog([]string{"gpt-6.1-sol", "glm-5.3", "claude-opus-5-5"}).Models {
		if m.Name != "gpt-6.1-sol" && m.Name != "glm-5.3" && m.Name != "claude-opus-5-5" {
			continue
		}
		if m.Thinking == nil {
			t.Fatalf("missing declared known rule: %s", m.Name)
		}
		if m.Thinking.ZeroAllowed {
			t.Fatalf("declared thinking off for thinking-only model: %s", m.Name)
		}
		for _, level := range m.Thinking.Levels {
			original, _ := json.Marshal(map[string]any{"reasoning_effort": level})
			var e error
			if strings.HasPrefix(m.Name, "claude-") {
				_, e = normalizeClaudeParameters(m.Name, "openai", original, []byte(`{}`))
			} else {
				fmt := "openai-response"
				if m.Name == "glm-5.3" {
					fmt = "openai"
				}
				_, e = normalizeFactoryParameters(m.Name, "openai", fmt, original, []byte(`{}`))
			}
			if e != nil {
				t.Fatalf("declared level %s rejected for %s: %v", level, m.Name, e)
			}
		}
	}
}
