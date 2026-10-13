package provider

import (
	"fmt"
	"testing"
)

func TestClaudeKnownContracts(t *testing.T) {
	for _, c := range []struct {
		model, format, extra, typ, effort string
		fail                              bool
	}{
		{"claude-sonnet-5-5", "openai", `"reasoning_effort":"none"`, "between_tools", "", false},
		{"claude-opus-5-5", "openai", `"reasoning_effort":"none"`, "", "", true},
		{"claude-sonnet-5-5", "openai", `"reasoning_effort":"minimal"`, "adaptive", "low", false},
		{"minimax-m3", "openai", `"reasoning_effort":"max"`, "adaptive", "max", false},
		{"claude-sonnet-4-6", "openai", `"reasoning_effort":"xhigh"`, "adaptive", "max", false},
		{"claude-sonnet-5-5", "claude", `"thinking":{"type":"enabled","budget_tokens":8192}`, "adaptive", "medium", false},
		{"claude-sonnet-5-5", "claude", `"thinking":{"type":"disabled"}`, "between_tools", "", false},
		{"claude-opus-5-5", "claude", `"thinking":{"type":"disabled"}`, "", "", true},
		{"claude-sonnet-5-5", "claude", `"thinking":{"type":"adaptive"},"output_config":{"effort":"xhigh"}`, "adaptive", "xhigh", false},
		{"claude-sonnet-4-6", "claude", `"thinking":{"type":"adaptive"},"output_config":{"effort":"xhigh"}`, "", "", true},
		{"claude-sonnet-5-5", "openai", `"temperature":0.7`, "", "", true},
		{"claude-sonnet-5-5", "openai", `"top_p":0.9`, "", "", true},
		{"claude-sonnet-5-5", "claude", `"tool_choice":{"type":"any"}`, "", "", true},
		{"claude-sonnet-5-5", "claude", `"tool_choice":{"type":"tool","name":"x"}`, "", "", true},
		{"claude-sonnet-4-5-20250929", "claude", `"thinking":{"type":"enabled","budget_tokens":128}`, "", "", true},
	} {
		t.Run(c.model+"/"+c.extra, func(t *testing.T) {
			body := `{"messages":[{"role":"user","content":"hi"}],"max_tokens":64000,` + c.extra + `}`
			out, err := prepareParameterFixture(t, c.model, c.format, body)
			if c.fail {
				if status, ok := err.(*StatusError); !ok || status.Status != 400 {
					t.Fatalf("expected local 400, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			thinking, _ := out["thinking"].(map[string]any)
			cfg, _ := out["output_config"].(map[string]any)
			if thinking["type"] != c.typ || (c.effort != "" && cfg["effort"] != c.effort) {
				t.Fatalf("wrong contract: %v %v", thinking, cfg)
			}
			if thinking["budget_tokens"] != nil {
				t.Fatal("obsolete manual budget survived adaptive mapping")
			}
		})
	}
}

func TestClaude55LargeBudgetUsesPinnedSDKThreshold(t *testing.T) {
	for _, budget := range []int{32769, 65536} {
		raw := fmt.Sprintf(`{"messages":[{"role":"user","content":"hi"}],"max_tokens":128000,"thinking":{"type":"enabled","budget_tokens":%d}}`, budget)
		out, err := prepareParameterFixture(t, "claude-sonnet-5-5", "claude", raw)
		if err != nil {
			t.Fatal(err)
		}
		if out["output_config"].(map[string]any)["effort"] != "xhigh" {
			t.Fatalf("budget %d was not mapped using SDK xhigh cutoff: %#v", budget, out)
		}
	}
}

func TestClaude55ChatEffortUsesAdaptive(t *testing.T) {
	out, err := prepareParameterFixture(t, "claude-sonnet-5-5", "openai", `{"messages":[{"role":"user","content":"hi"}],"reasoning_effort":"medium","max_tokens":64000}`)
	if err != nil {
		t.Fatal(err)
	}
	thinking, _ := out["thinking"].(map[string]any)
	config, _ := out["output_config"].(map[string]any)
	if thinking["type"] != "adaptive" || thinking["budget_tokens"] != nil || config["effort"] != "medium" {
		t.Fatalf("legacy thinking survived: %v %v", thinking, config)
	}
}
