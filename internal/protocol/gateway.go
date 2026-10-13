package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Fixed wire identity verified against Factory, not client-controlled policy.
// User-defined system instructions remain folded into user content.
const DroidPrefix = "You are Droid, an AI software engineering agent built by Factory."

func GatewayRequest(body []byte, format string) ([]byte, error) {
	b, e := Fold(body, format)
	if e != nil {
		return nil, e
	}
	var d map[string]any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if e = dec.Decode(&d); e != nil {
		return nil, e
	}
	switch format {
	case "claude":
		d["system"] = DroidPrefix
	case "openai-response":
		d["instructions"] = DroidPrefix
	case "openai":
		m, _ := d["messages"].([]any)
		d["messages"] = append([]any{map[string]any{"role": "system", "content": DroidPrefix}}, m...)
	default:
		return nil, fmt.Errorf("factory: unsupported wire format")
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	e = enc.Encode(d)
	return bytes.TrimSpace(out.Bytes()), e
}
