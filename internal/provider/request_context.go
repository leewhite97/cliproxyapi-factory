package provider

import (
	"context"
	"errors"
	"factory.local/cliproxyapi-factory/internal/redact"
)

// Only the host's canonical session is reused; account selection stays in CPA.
func requestSession(r Request) string {
	s, _ := r.Metadata["canonical_session_id"].(string)
	if len(s) == 0 || len(s) > 256 {
		return ""
	}
	for _, ch := range s {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch == '.' || ch == ':' || ch == '/') {
			return ""
		}
	}
	return s
}
func transportError(e error, key string) *StatusError {
	if errors.Is(e, context.Canceled) {
		return &StatusError{"request_cancelled", "factory: request cancelled", 499, false}
	}
	if errors.Is(e, context.DeadlineExceeded) {
		return &StatusError{"request_timeout", "factory: request timeout", 504, true}
	}
	return &StatusError{"network_error", redact.Text(e.Error(), key), 502, true}
}
