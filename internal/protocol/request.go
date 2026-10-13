package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func newID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic("secure randomness unavailable")
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func Headers(key, org string, r Route) http.Header {
	return HeadersWithSession(key, org, r, "")
}

func HeadersWithSession(key, org string, r Route, session string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+key)
	h.Set("Content-Type", "application/json")
	h.Set("X-Factory-Client", "cli")
	h.Set("X-Client-Version", "0.162.1")
	h.Set("User-Agent", "factory-cli/0.162.1")
	h.Set("x-api-provider", r.Provider)
	if session == "" {
		session = newID()
	}
	h.Set("x-session-id", session)
	h.Set("x-assistant-message-id", newID())
	if org != "" {
		h.Set("X-Factory-Org-Id", org)
	}
	if r.Format == "claude" {
		h.Set("anthropic-version", "2023-06-01")
	}
	if r.Format == "openai-response" {
		h.Set("OpenAI-Platform", "org-bHuLtG1fGmYk5YaOihAAXFBw")
	}
	return h
}
func text(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	a, ok := v.([]any)
	if !ok {
		return "", fmt.Errorf("factory: invalid system content")
	}
	var out []string
	for _, p := range a {
		d, ok := p.(map[string]any)
		if !ok {
			return "", fmt.Errorf("factory: invalid system block")
		}
		s, ok := d["text"].(string)
		if !ok {
			return "", fmt.Errorf("factory: unsupported system block")
		}
		out = append(out, s)
	}
	return strings.Join(out, "\n\n"), nil
}
func Fold(body []byte, format string) ([]byte, error) {
	var d map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if e := dec.Decode(&d); e != nil {
		return nil, fmt.Errorf("factory: invalid JSON")
	}
	var parts []string
	field := "messages"
	top := "system"
	blockType := "text"
	if format == "openai-response" {
		field = "input"
		top = "instructions"
		blockType = "input_text"
	}
	if s, e := text(d[top]); e != nil {
		return nil, e
	} else if s != "" {
		parts = append(parts, s)
	}
	delete(d, top)
	var items []any
	switch v := d[field].(type) {
	case []any:
		items = v
	case string:
		items = []any{map[string]any{"role": "user", "content": v}}
	case nil:
	default:
		return nil, fmt.Errorf("factory: invalid message list")
	}
	kept := make([]any, 0, len(items))
	for _, v := range items {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("factory: invalid message")
		}
		role, _ := m["role"].(string)
		if role == "system" || role == "developer" {
			s, e := text(m["content"])
			if e != nil {
				return nil, e
			}
			if s != "" {
				parts = append(parts, s)
			}
		} else {
			kept = append(kept, v)
		}
	}
	if len(parts) > 0 {
		preamble := "<system>\n" + strings.Join(parts, "\n\n") + "\n</system>\n\n"
		var first map[string]any
		if len(kept) > 0 {
			first, _ = kept[0].(map[string]any)
		}
		if first != nil && first["role"] == "user" {
			switch c := first["content"].(type) {
			case string:
				first["content"] = preamble + c
			case []any:
				first["content"] = append([]any{map[string]any{"type": blockType, "text": preamble}}, c...)
			case nil:
				first["content"] = preamble
			default:
				return nil, fmt.Errorf("factory: invalid user content")
			}
		} else {
			kept = append([]any{map[string]any{"role": "user", "content": preamble}}, kept...)
		}
	}
	d[field] = kept
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	e := enc.Encode(d)
	return bytes.TrimSpace(out.Bytes()), e
}
