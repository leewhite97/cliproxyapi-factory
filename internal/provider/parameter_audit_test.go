package provider

// Audit harness only: exercises the real prepare path without networking.
import (
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"os"
	"testing"
)

func TestPiParameterAuditWire(t *testing.T) {
	var snapshot struct {
		Models []string `json:"models"`
	}
	raw, err := os.ReadFile("testdata/audit-models.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	out, err := os.OpenFile(t.TempDir()+"/pi-parameter-wire.jsonl", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	type fixture struct {
		name  string
		patch map[string]any
	}
	for _, model := range snapshot.Models {
		for _, format := range []string{"openai", "claude", "openai-response"} {
			base := map[string]any{"model": model, "stream": true}
			switch format {
			case "openai":
				base["messages"] = []any{map[string]any{"role": "developer", "content": "You are a helpful coding assistant. Follow the supplied tools."}, map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Reply OK."}}}}
				base["max_tokens"] = 4096
			case "claude":
				base["system"] = "Follow the supplied tools."
				base["messages"] = []any{map[string]any{"role": "user", "content": "Reply OK."}}
				base["max_tokens"] = 4096
			default:
				base["instructions"] = "Follow the supplied tools."
				base["input"] = []any{map[string]any{"role": "user", "content": "Reply OK."}}
				base["max_output_tokens"] = 4096
			}
			fixtures := []fixture{{"baseline", nil}}
			for _, level := range []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"} {
				if format == "openai" {
					fixtures = append(fixtures, fixture{"reasoning_" + level, map[string]any{"reasoning_effort": level}})
				}
				if format == "openai-response" {
					fixtures = append(fixtures, fixture{"reasoning_" + level, map[string]any{"reasoning": map[string]any{"effort": level}}})
				}
				if format == "claude" {
					fixtures = append(fixtures, fixture{"adaptive_" + level, map[string]any{"thinking": map[string]any{"type": "adaptive"}, "output_config": map[string]any{"effort": level}}})
				}
			}
			for _, p := range []fixture{
				{"thinking_enabled", map[string]any{"thinking": map[string]any{"type": "enabled", "budget_tokens": 1024}}},
				{"thinking_disabled", map[string]any{"thinking": map[string]any{"type": "disabled"}}},
				{"temperature", map[string]any{"temperature": 0.7}}, {"top_p", map[string]any{"top_p": 0.9}},
				{"both_sampling", map[string]any{"temperature": 0.7, "top_p": 0.9}},
				{"max_completion_tokens", map[string]any{"max_completion_tokens": 64000}},
				{"max_output_tokens", map[string]any{"max_output_tokens": 64000}},
				{"stream_usage", map[string]any{"stream_options": map[string]any{"include_usage": true}}},
				{"store_false", map[string]any{"store": false}}, {"store_true", map[string]any{"store": true}},
				{"parallel_tools", map[string]any{"parallel_tool_calls": false}},
				{"json_object", map[string]any{"response_format": map[string]any{"type": "json_object"}}},
				{"stop", map[string]any{"stop": []any{"END"}}},
				{"metadata", map[string]any{"metadata": map[string]any{"audit": "synthetic"}}},
			} {
				fixtures = append(fixtures, p)
			}
			schema := map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}, "additionalProperties": false}
			tool := map[string]any{"type": "function", "function": map[string]any{"name": "get_weather", "description": "Synthetic audit tool", "parameters": schema, "strict": true}}
			choices := []any{"auto", "none", "required", map[string]any{"type": "function", "function": map[string]any{"name": "get_weather"}}}
			if format == "claude" {
				tool = map[string]any{"name": "get_weather", "description": "Synthetic audit tool", "input_schema": schema}
				choices = []any{map[string]any{"type": "auto"}, map[string]any{"type": "none"}, map[string]any{"type": "any"}, map[string]any{"type": "tool", "name": "get_weather"}}
			}
			if format == "openai-response" {
				tool = map[string]any{"type": "function", "name": "get_weather", "description": "Synthetic audit tool", "parameters": schema, "strict": true}
				choices = []any{"auto", "none", "required", map[string]any{"type": "function", "name": "get_weather"}}
			}
			for i, c := range choices {
				name := []string{"tool_auto", "tool_none", "tool_required", "tool_named"}[i]
				fixtures = append(fixtures, fixture{name, map[string]any{"tools": []any{tool}, "tool_choice": c}})
			}
			for _, f := range fixtures {
				body := map[string]any{}
				for k, v := range base {
					body[k] = v
				}
				for k, v := range f.patch {
					body[k] = v
				}
				payload, _ := json.Marshal(body)
				_, route, _, wire, e := (&Service{}).prepare(Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: model, Format: format, Payload: payload, OriginalRequest: payload, StorageJSON: json.RawMessage(`{"type":"factory","api_key":"fk-synthetic-audit-not-a-real-key"}`)}}, true)
				row := map[string]any{"model": model, "source_format": format, "upstream_format": route.Format, "case": f.name, "input": body}
				if e != nil {
					row["error"] = e.Error()
				} else {
					var d any
					if e = json.Unmarshal(wire, &d); e != nil {
						t.Fatal(e)
					}
					row["output"] = d
				}
				if e = enc.Encode(row); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
	t.Logf("Audited %d models across three source protocols; results are wire transformations, not live acceptance", len(snapshot.Models))
}
