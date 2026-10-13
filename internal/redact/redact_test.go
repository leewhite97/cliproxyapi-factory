package redact

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	s := Text("Authorization: Bearer fk-test-secret and fk-other-value", "fk-test-secret")
	if strings.Contains(s, "fk-test-secret") || strings.Contains(s, "fk-other-value") {
		t.Fatal("leaked credential")
	}
}
