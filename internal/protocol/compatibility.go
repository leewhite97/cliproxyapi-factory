package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Compatibility is Factory-scoped, never selected from the client's identity.
// SensitiveWords is opt-in; enabling user text may alter code or quoted text.
// Matcher algorithm adapted from CPA v8.0.4 cloak_obfuscate.go (MIT).
type Compatibility struct {
	Mode              string   `yaml:"mode" json:"mode"`
	SensitiveWords    []string `yaml:"sensitive_words" json:"sensitive_words"`
	ObfuscateUserText bool     `yaml:"obfuscate_user_text" json:"obfuscate_user_text"`
}

func (c Compatibility) Validate() error {
	if c.Mode != "" && c.Mode != "factory" && c.Mode != "passthrough" {
		return fmt.Errorf("factory: compatibility mode must be factory or passthrough")
	}
	if c.Mode == "passthrough" && (len(c.SensitiveWords) > 0 || c.ObfuscateUserText) {
		return fmt.Errorf("factory: passthrough cannot enable text obfuscation")
	}
	if len(c.SensitiveWords) > 128 {
		return fmt.Errorf("factory: too many sensitive words")
	}
	for _, w := range c.SensitiveWords {
		if len(w) > 4096 {
			return fmt.Errorf("factory: sensitive word too long")
		}
	}
	return nil
}
func GatewayRequestWithCompatibility(body []byte, format string, c Compatibility) ([]byte, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	if format != "claude" && format != "openai" && format != "openai-response" {
		return nil, fmt.Errorf("factory: unsupported wire format")
	}
	if c.Mode == "passthrough" {
		if !json.Valid(body) {
			return nil, fmt.Errorf("factory: invalid JSON")
		}
		return append([]byte(nil), body...), nil
	}
	b, e := c.obfuscate(body, format)
	if e != nil {
		return nil, e
	}
	return GatewayRequest(b, format)
}
func (c Compatibility) obfuscate(body []byte, format string) ([]byte, error) {
	words := []string{}
	for _, w := range c.SensitiveWords {
		w = strings.TrimSpace(w)
		if utf8.RuneCountInString(w) >= 2 && !strings.Contains(w, "\u200b") {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		return body, nil
	}
	sort.Slice(words, func(i, j int) bool { return len(words[i]) > len(words[j]) })
	for i := range words {
		words[i] = regexp.QuoteMeta(words[i])
	}
	re, e := regexp.Compile("(?i)" + strings.Join(words, "|"))
	if e != nil {
		return nil, e
	}
	change := func(s string) string {
		return re.ReplaceAllStringFunc(s, func(w string) string { _, n := utf8.DecodeRuneInString(w); return w[:n] + "\u200b" + w[n:] })
	}
	var d map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if e := dec.Decode(&d); e != nil || d == nil {
		return nil, fmt.Errorf("factory: invalid request object")
	}
	alter := func(v any) any {
		if s, ok := v.(string); ok {
			return change(s)
		}
		if blocks, ok := v.([]any); ok {
			for _, v := range blocks {
				if m, ok := v.(map[string]any); ok && (m["type"] == "text" || m["type"] == "input_text" || m["type"] == nil) {
					if s, ok := m["text"].(string); ok {
						m["text"] = change(s)
					}
				}
			}
		}
		return v
	}
	top, field := "system", "messages"
	if format == "openai-response" {
		top, field = "instructions", "input"
	}
	if v, ok := d[top]; ok {
		d[top] = alter(v)
	}
	if s, ok := d[field].(string); ok && c.ObfuscateUserText {
		d[field] = change(s)
	}
	if messages, ok := d[field].([]any); ok {
		for _, v := range messages {
			if m, ok := v.(map[string]any); ok {
				role := m["role"]
				if role == "system" || role == "developer" || (role == "user" && c.ObfuscateUserText) {
					m["content"] = alter(m["content"])
				}
			}
		}
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if e = enc.Encode(d); e != nil {
		return nil, e
	}
	return bytes.TrimSpace(out.Bytes()), nil
}
