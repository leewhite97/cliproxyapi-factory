package redact

import (
	"regexp"
	"strings"
)

var keys = regexp.MustCompile(`fk-[A-Za-z0-9_-]+`)
var bearer = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`)

func Text(s string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "[REDACTED]")
		}
	}
	s = keys.ReplaceAllString(s, "[REDACTED]")
	return bearer.ReplaceAllString(s, "Bearer [REDACTED]")
}
