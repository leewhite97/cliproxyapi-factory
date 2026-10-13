package provider

import (
	"bytes"
	"encoding/json"
)

func parameterObject(raw []byte) (map[string]any, error) {
	var d map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if !json.Valid(raw) {
		return nil, &StatusError{"invalid_request", "factory: invalid parameter JSON", 400, false}
	}
	if err := dec.Decode(&d); err != nil || d == nil {
		return nil, &StatusError{"invalid_request", "factory: parameter body must be an object", 400, false}
	}
	return d, nil
}

// Recover client parameter intent after endpoint-specific SDK translation.
func normalizeFactoryParameters(model, sourceFormat, targetFormat string, original, wire []byte) ([]byte, error) {
	spec := modelSpec(model)
	in, err := parameterObject(original)
	if err != nil {
		return nil, err
	}
	out, err := parameterObject(wire)
	if err != nil {
		return nil, err
	}
	if err := normalizeOutputLimit(in, out, targetFormat); err != nil {
		return nil, err
	}
	if rf, ok := in["response_format"].(map[string]any); ok {
		kind, _ := rf["type"].(string)
		if targetFormat == "claude" && kind != "" && kind != "text" {
			return nil, &StatusError{"unsupported_parameter", "factory: OpenAI response_format mapping is unavailable for the Claude route; use native output_config.format instead", 400, false}
		}
		if targetFormat == "openai-response" && kind == "json_object" {
			text, _ := out["text"].(map[string]any)
			if text == nil {
				text = map[string]any{}
			}
			text["format"] = map[string]any{"type": "json_object"}
			out["text"] = text
		}
	}
	if targetFormat == "openai" {
		if spec.MinimalToLow && out["reasoning_effort"] == "minimal" {
			out["reasoning_effort"] = "low"
		}
		if spec.ThinkingOnly {
			thinking, _ := out["thinking"].(map[string]any)
			if out["reasoning_effort"] == "none" || thinking["type"] == "disabled" {
				return nil, &StatusError{"unsupported_parameter", "factory: glm-5.3 is thinking-only and cannot disable reasoning", 400, false}
			}
		}
	}
	if targetFormat == "openai-response" {
		if spec.StrictGPT {
			for _, field := range []string{"temperature", "top_p"} {
				if _, ok := in[field]; ok {
					return nil, &StatusError{"unsupported_parameter", "factory: " + model + " does not support " + field, 400, false}
				}
			}
		}
		if originalReasoning, ok := in["reasoning"].(map[string]any); ok {
			reasoning, _ := out["reasoning"].(map[string]any)
			if reasoning == nil {
				reasoning = map[string]any{}
			}
			for field, value := range originalReasoning {
				reasoning[field] = value
			}
			out["reasoning"] = reasoning
		}
		if effort, ok := in["reasoning_effort"]; ok {
			reasoning, _ := out["reasoning"].(map[string]any)
			if reasoning == nil {
				reasoning = map[string]any{}
			}
			reasoning["effort"] = effort
			out["reasoning"] = reasoning
			delete(out, "reasoning_effort")
		}
		if spec.Codex && (in["temperature"] != nil || in["top_p"] != nil) {
			originalReasoning, _ := in["reasoning"].(map[string]any)
			reasoning, _ := out["reasoning"].(map[string]any)
			if in["reasoning_effort"] == nil && originalReasoning["effort"] == nil {
				// Do not manufacture Codex's medium default when the original caller
				// requested sampling with the native endpoint's default reasoning.
				delete(reasoning, "effort")
				if len(reasoning) == 0 {
					delete(out, "reasoning")
				}
			} else if reasoning["effort"] != "none" {
				return nil, &StatusError{"unsupported_parameter", "factory: gpt-5.3-codex sampling is incompatible with active reasoning; omit sampling or select reasoning none", 400, false}
			}
		}
		if spec.Codex || spec.StrictGPT {
			if reasoning, ok := out["reasoning"].(map[string]any); ok {
				if effort, exists := reasoning["effort"]; exists {
					level, valid := effort.(string)
					if !valid {
						return nil, &StatusError{"invalid_parameter", "factory: reasoning effort must be a string", 400, false}
					}
					// Logical minimum uses the lowest supported thinking level.
					if level == "minimal" {
						level = "low"
					}
					allowed := spec.permitsEffort(level)
					if !allowed {
						return nil, &StatusError{"unsupported_parameter", "factory: " + model + " does not support reasoning effort " + level, 400, false}
					}
					reasoning["effort"] = level
				}
			}
		}
		for _, field := range []string{"parallel_tool_calls", "store", "temperature", "top_p"} {
			if value, ok := in[field]; ok {
				out[field] = value
			}
		}
		for _, field := range []string{"max_output_tokens", "max_tokens", "max_completion_tokens"} {
			if value, ok := in[field]; ok {
				out["max_output_tokens"] = value
				break
			}
		}
	}
	return json.Marshal(out)
}
