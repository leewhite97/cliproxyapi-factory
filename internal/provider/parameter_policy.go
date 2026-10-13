package provider

import (
	"encoding/json"
	"math"
)

// Conflicting aliases are rejected; an explicit caller limit wins over translator defaults.
func normalizeOutputLimit(in, out map[string]any, targetFormat string) error {
	var limit any
	var limitValue int64
	for _, field := range []string{"max_output_tokens", "max_tokens", "max_completion_tokens"} {
		if value, exists := in[field]; exists {
			number, ok := value.(json.Number)
			if !ok {
				return &StatusError{"invalid_parameter", "factory: " + field + " must be a positive integer", 400, false}
			}
			n, err := number.Int64()
			if err != nil {
				f, floatErr := number.Float64()
				if floatErr != nil || f <= 0 || f >= float64(math.MaxInt64) || math.Trunc(f) != f {
					return &StatusError{"invalid_parameter", "factory: " + field + " must be a positive integer", 400, false}
				}
				n = int64(f)
			}
			if n <= 0 {
				return &StatusError{"invalid_parameter", "factory: " + field + " must be a positive integer", 400, false}
			}
			if limit != nil && n != limitValue {
				return &StatusError{"invalid_parameter", "factory: conflicting output token limit aliases", 400, false}
			}
			limit, limitValue = value, n
		}
	}
	if limit != nil {
		for _, field := range []string{"max_output_tokens", "max_tokens", "max_completion_tokens"} {
			delete(out, field)
		}
		field := "max_tokens"
		if targetFormat == "openai-response" {
			field = "max_output_tokens"
		}
		out[field] = limit
	}

	return nil
}
