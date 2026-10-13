package provider

import (
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"testing"
)

func prepareParameterFixture(t *testing.T, model, format, body string) (map[string]any, error) {
	t.Helper()
	_, _, _, wire, err := (&Service{}).prepare(Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: model, Format: format, SourceFormat: format, Payload: []byte(body), OriginalRequest: []byte(body), StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-parameter-fixture"}`)}}, true)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = json.Unmarshal(wire, &out)
	return out, err
}
func TestResponsesSupportedFieldsPreserved(t *testing.T) {
	for _, c := range []struct {
		field string
		value any
	}{{"parallel_tool_calls", false}, {"store", true}, {"temperature", 0.7}, {"top_p", 0.9}} {
		t.Run(c.field, func(t *testing.T) {
			body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "Reply OK"}}, c.field: c.value}
			raw, _ := json.Marshal(body)
			out, err := prepareParameterFixture(t, "gpt-5.3-codex", "openai", string(raw))
			if err != nil {
				t.Fatal(err)
			}
			if out[c.field] != c.value {
				t.Fatalf("explicit %s overridden: got %v want %v", c.field, out[c.field], c.value)
			}
		})
	}
}

func TestResponsesUnsupportedParametersFailLocally(t *testing.T) {
	for _, patch := range []string{`"temperature":0.7`, `"top_p":0.9`, `"reasoning_effort":"none"`} {
		_, err := prepareParameterFixture(t, "gpt-6.1-sol", "openai", `{"messages":[{"role":"user","content":"hi"}],`+patch+`}`)
		if status, ok := err.(*StatusError); !ok || status.Status != 400 {
			t.Fatalf("unsupported input should fail locally with 400: %v", err)
		}
	}
}
func TestResponsesMinimalMapsToLowestEffort(t *testing.T) {
	out, err := prepareParameterFixture(t, "gpt-5.3-codex", "openai", `{"messages":[{"role":"user","content":"hi"}],"reasoning_effort":"minimal"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out["reasoning"].(map[string]any)["effort"] != "low" {
		t.Fatal("logical minimal not mapped to supported lowest effort")
	}
}

func TestFactoryLimitAliasesAndConflicts(t *testing.T) {
	out, err := prepareParameterFixture(t, "deepseek-v4.1-flash", "openai", `{"messages":[{"role":"user","content":"hi"}],"max_output_tokens":2048}`)
	if err != nil {
		t.Fatal(err)
	}
	if out["max_tokens"] != float64(2048) || out["max_output_tokens"] != nil {
		t.Fatal("Responses limit alias leaked into Chat route")
	}
	for _, patch := range []string{`"max_tokens":2048,"max_completion_tokens":4096`, `"max_completion_tokens":-1`, `"max_completion_tokens":"4096"`} {
		_, err = prepareParameterFixture(t, "gpt-5.3-codex", "openai", `{"messages":[{"role":"user","content":"hi"}],`+patch+`}`)
		if status, ok := err.(*StatusError); !ok || status.Status != 400 {
			t.Fatalf("invalid/conflicting limit was not rejected: %v", err)
		}
	}
}

func TestStructuredOutputNotSilentlyDropped(t *testing.T) {
	body := `{"messages":[{"role":"user","content":"Return a JSON object"}],"response_format":{"type":"json_object"}}`
	out, err := prepareParameterFixture(t, "gpt-5.3-codex", "openai", body)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := out["text"].(map[string]any)
	format, _ := text["format"].(map[string]any)
	if format["type"] != "json_object" {
		t.Fatal("JSON object constraint lost")
	}
	_, err = prepareParameterFixture(t, "claude-sonnet-5-5", "openai", body)
	if status, ok := err.(*StatusError); !ok || status.Status != 400 {
		t.Fatalf("unimplemented JSON object mapping should fail clearly: %v", err)
	}
}

func TestCoreReasoningCapabilities(t *testing.T) {
	for _, model := range []string{"kimi-k3", "nemotron-3-ultra"} {
		out, err := prepareParameterFixture(t, model, "openai", `{"messages":[{"role":"user","content":"hi"}],"reasoning_effort":"minimal"}`)
		if err != nil {
			t.Fatal(err)
		}
		if out["reasoning_effort"] != "low" {
			t.Fatal("minimal not mapped to lowest supported level")
		}
	}
	for _, extra := range []string{`"reasoning_effort":"none"`, `"thinking":{"type":"disabled"}`} {
		_, err := prepareParameterFixture(t, "glm-5.3", "openai", `{"messages":[{"role":"user","content":"hi"}],`+extra+`}`)
		if status, ok := err.(*StatusError); !ok || status.Status != 400 {
			t.Fatalf("thinking-only model must reject off locally: %v", err)
		}
	}
}

func TestHostConvertedPayloadKeepsOriginalReasoning(t *testing.T) {
	original := []byte(`{"input":"hi","reasoning":{"effort":"high","summary":"detailed"},"max_output_tokens":2048}`)
	_, _, _, wire, err := (&Service{}).prepare(Request{ExecutorRequest: pluginapi.ExecutorRequest{Model: "gpt-5.3-codex", Format: "openai", SourceFormat: "openai-response", Payload: []byte(`{"messages":[{"role":"user","content":"hi"}]}`), OriginalRequest: original, StorageJSON: []byte(`{"type":"factory","api_key":"fk-test-parameter-fixture"}`)}}, true)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	json.Unmarshal(wire, &out)
	reasoning := out["reasoning"].(map[string]any)
	if reasoning["effort"] != "high" || reasoning["summary"] != "detailed" {
		t.Fatalf("original intent lost across host conversion: %v", reasoning)
	}
}

func TestResponsesSamplingDoesNotInventReasoning(t *testing.T) {
	out, err := prepareParameterFixture(t, "gpt-5.3-codex", "openai", `{"messages":[{"role":"user","content":"hi"}],"temperature":0.7}`)
	if err != nil {
		t.Fatal(err)
	}
	reasoning, _ := out["reasoning"].(map[string]any)
	if reasoning["effort"] != nil {
		t.Fatal("sampling request acquired an unrequested thinking effort")
	}
	_, err = prepareParameterFixture(t, "gpt-5.3-codex", "openai", `{"messages":[{"role":"user","content":"hi"}],"temperature":0.7,"reasoning_effort":"medium"}`)
	if status, ok := err.(*StatusError); !ok || status.Status != 400 {
		t.Fatalf("incompatible explicit reasoning/sampling not rejected: %v", err)
	}
}

func TestResponsesOutputLimitPreserved(t *testing.T) {
	out, err := prepareParameterFixture(t, "gpt-5.3-codex", "openai", `{"messages":[{"role":"user","content":"Reply OK"}],"max_completion_tokens":2048}`)
	if err != nil {
		t.Fatal(err)
	}
	if out["max_output_tokens"] != float64(2048) {
		t.Fatalf("client output limit lost: %v", out["max_output_tokens"])
	}
}
