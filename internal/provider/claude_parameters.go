package provider

import "encoding/json"

// Native budget-only requests on 5.5 are a documented effort approximation;
// original logical effort always takes precedence over SDK-created budgets.
func normalizeClaudeParameters(model, sourceFormat string, original, wire []byte) ([]byte, error) {
	in, err := parameterObject(original)
	if err != nil {
		return nil, err
	}
	out, err := parameterObject(wire)
	if err != nil {
		return nil, err
	}
	bad := func(message string) ([]byte, error) {
		return nil, &StatusError{"unsupported_parameter", "factory: " + model + ": " + message, 400, false}
	}
	spec := modelSpec(model)
	modern := spec.ModernClaude
	mini := spec.MiniMax
	sonnet46 := spec.Sonnet46
	if modern {
		for _, field := range []string{"temperature", "top_p"} {
			if _, exists := in[field]; exists {
				return bad(field + " is not supported")
			}
		}
		if choice, ok := out["tool_choice"].(map[string]any); ok && (choice["type"] == "any" || choice["type"] == "tool") {
			return bad("forced tool choice is not supported; use auto or none")
		}
	}
	config, _ := out["output_config"].(map[string]any)
	if config == nil {
		config = map[string]any{}
	}
	originalConfig, _ := in["output_config"].(map[string]any)
	for field, value := range originalConfig {
		config[field] = value
	}
	thinking, _ := out["thinking"].(map[string]any)
	if value, exists := in["thinking"]; exists && value != nil {
		var valid bool
		thinking, valid = value.(map[string]any)
		if !valid {
			return bad("thinking must be an object")
		}
	}
	if thinking == nil {
		thinking = map[string]any{}
	}
	level := ""
	logical := false
	if value, exists := in["reasoning_effort"]; exists {
		var ok bool
		level, ok = value.(string)
		if !ok {
			return bad("reasoning_effort must be a string")
		}
		logical = true
	}
	if reason, ok := in["reasoning"].(map[string]any); ok {
		if value, exists := reason["effort"]; exists {
			s, valid := value.(string)
			if !valid {
				return bad("reasoning effort must be a string")
			}
			if level != "" && level != s {
				return bad("conflicting reasoning effort fields")
			}
			level = s
			logical = true
		}
	}
	if value, exists := originalConfig["effort"]; exists {
		s, valid := value.(string)
		if !valid {
			return bad("output_config.effort must be a string")
		}
		if level != "" && level != s {
			return bad("conflicting reasoning effort fields")
		}
		level = s
		logical = false
	}
	typ, _ := thinking["type"].(string)
	if modern && typ == "enabled" && len(originalConfig) > 0 && originalConfig["effort"] != nil {
		return bad("manual thinking and explicit effort cannot be combined")
	}
	if modern && level == "" && typ == "enabled" {
		n, ok := thinking["budget_tokens"].(json.Number)
		if !ok {
			return bad("legacy enabled thinking requires an integer budget_tokens")
		}
		budget, e := n.Int64()
		if e != nil || budget <= 0 {
			return bad("legacy budget_tokens must be positive")
		}
		// Same ordinal cutoffs as the pinned SDK, with minimal collapsed to low.
		switch {
		case budget <= 1024:
			level = "low"
		case budget <= 8192:
			level = "medium"
		case budget <= 24576:
			level = "high"
		default:
			level = "xhigh"
		}
		logical = true
	}
	if modern && typ == "disabled" {
		if level != "" && level != "none" {
			return bad("disabled thinking conflicts with an active effort")
		}
		level = "none"
		logical = true
	}
	if modern && typ == "between_tools" {
		if model != "claude-sonnet-5-5" {
			return bad("between_tools is not a verified off mode for this model")
		}
		if level != "" {
			return bad("between_tools cannot be combined with an effort")
		}
	}
	if (modern || mini || sonnet46) && level != "" {
		if logical && level == "minimal" {
			level = "low"
		}
		if logical && sonnet46 && level == "xhigh" {
			level = "max"
		} // Matches the existing SDK ordinal mapping.
		if level == "none" && logical {
			switch {
			case model == "claude-sonnet-5-5":
				thinking = map[string]any{"type": "between_tools"}
				delete(config, "effort")
			case modern:
				return bad("disabling thinking is not supported; select an active effort")
			case mini:
				thinking = map[string]any{"type": "adaptive"}
				config["effort"] = "none"
			default:
				thinking = map[string]any{"type": "disabled"}
				delete(config, "effort")
			}
		} else {
			allowed := spec.permitsEffort(level)
			if !allowed {
				return bad("unsupported effort " + level)
			}
			thinking = map[string]any{"type": "adaptive"}
			config["effort"] = level
		}
	}
	if modern && typ != "" && typ != "enabled" && typ != "disabled" && typ != "adaptive" && typ != "between_tools" {
		return bad("unsupported thinking type " + typ)
	}
	if len(thinking) > 0 {
		out["thinking"] = thinking
	}
	if len(config) > 0 {
		out["output_config"] = config
	} else {
		delete(out, "output_config")
	}
	if !modern {
		for _, field := range []string{"temperature", "top_p"} {
			if value, exists := in[field]; exists {
				out[field] = value
			}
		}
		if sonnet46 && in["temperature"] != nil && in["top_p"] != nil {
			return bad("temperature and top_p cannot both be specified")
		}
		if thinking["type"] == "enabled" {
			budget, ok := thinking["budget_tokens"].(json.Number)
			if !ok {
				return bad("enabled thinking requires an integer budget_tokens")
			}
			n, e := budget.Int64()
			if e != nil || n < 1024 {
				return bad("enabled thinking budget_tokens must be at least 1024")
			}
			if limit, ok := out["max_tokens"].(json.Number); ok {
				max, e := limit.Int64()
				if e != nil || n >= max {
					return bad("max_tokens must exceed the thinking budget")
				}
			}
			if temp, exists := in["temperature"]; exists {
				number, ok := temp.(json.Number)
				if !ok {
					return bad("temperature must be numeric")
				}
				f, e := number.Float64()
				if e != nil || f != 1 {
					return bad("manual thinking requires temperature=1 or omitted")
				}
			}
		}
	}
	return json.Marshal(out)
}
