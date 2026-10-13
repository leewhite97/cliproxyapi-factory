package provider

import (
	"bytes"
	"encoding/json"
	"factory.local/cliproxyapi-factory/internal/protocol"
	"factory.local/cliproxyapi-factory/internal/redact"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Opt-in, one-shot capture. A private marker enables at most a 30-minute window.
// Disabled by default; never store credential storage, headers, or callback IDs.
func capture403(status int, r Request, route protocol.Route, wire, upstreamError []byte, key string) {
	if status != 403 {
		return
	}
	dir := os.Getenv("FACTORY_DIAGNOSTIC_DIR")
	if dir == "" {
		dir = "/CLIProxyAPI/logs/factory-cache"
	}
	marker := filepath.Join(dir, "capture-control.json")
	st, err := os.Lstat(marker)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || st.Size() > 4096 {
		return
	}
	f, err := os.Open(marker)
	if err != nil {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(f, 4097))
	f.Close()
	if err != nil || len(raw) > 4096 {
		return
	}
	var control struct {
		ExpiresAt time.Time `json:"expires_at"`
	}
	if json.Unmarshal(raw, &control) != nil {
		return
	}
	now := time.Now().UTC()
	if !now.Before(control.ExpiresAt) || control.ExpiresAt.After(now.Add(30*time.Minute)) {
		return
	}
	if len(r.OriginalRequest)+len(r.Payload)+len(wire) > 8*1024*1024 {
		return
	}
	clean := func(b []byte) any {
		if len(b) == 0 {
			return nil
		}
		var v any
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if d.Decode(&v) != nil {
			return "[INVALID JSON OMITTED]"
		}
		return sanitizeCapture(v, key)
	}
	record := map[string]any{"captured_at": now.Format(time.RFC3339Nano), "model": r.Model, "source_format": r.SourceFormat, "plugin_format": r.Format, "route": route, "status": status, "original_request": clean(r.OriginalRequest), "plugin_payload": clean(r.Payload), "upstream_request": clean(wire), "upstream_error": redact.Text(string(upstreamError), key)}
	b, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return
	}
	output := filepath.Join(dir, "factory-403-capture.json")
	f, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return
	}
	// Keep the exclusive filename even if writing fails: fail closed, never retry
	// and accidentally capture another user's request in this diagnostic window.
	f.Write(b)
	f.Sync()
	f.Close()
}

func sanitizeCapture(v any, key string) any {
	switch x := v.(type) {
	case string:
		return redact.Text(x, key)
	case []any:
		for i := range x {
			x[i] = sanitizeCapture(x[i], key)
		}
		return x
	case map[string]any:
		for k, y := range x {
			normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(k, "_", ""), "-", ""))
			switch normalized {
			case "authorization", "apikey", "accesstoken", "refreshtoken", "password", "secret", "clientsecret":
				x[k] = "[REDACTED]"
			default:
				x[k] = sanitizeCapture(y, key)
			}
		}
		return x
	default:
		return v
	}
}
