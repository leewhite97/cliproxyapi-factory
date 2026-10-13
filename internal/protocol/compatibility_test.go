package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigurableObfuscationScope(t *testing.T) {
	raw := []byte(`{"system":"ClientName must stay helpful","messages":[{"role":"user","content":[{"type":"text","text":"ClientName example"},{"type":"tool_result","content":"ClientName exact"}]}],"tools":[{"name":"ClientName","input_schema":{"type":"object"}}]}`)
	for _, user := range []bool{false, true} {
		wire, err := GatewayRequestWithCompatibility(raw, "claude", Compatibility{SensitiveWords: []string{"clientname"}, ObfuscateUserText: user})
		if err != nil {
			t.Fatal(err)
		}
		var d map[string]any
		json.Unmarshal(wire, &d)
		blocks := d["messages"].([]any)[0].(map[string]any)["content"].([]any)
		if !strings.Contains(blocks[0].(map[string]any)["text"].(string), "C\u200blientName") {
			t.Fatal("system words not obfuscated")
		}
		text := blocks[1].(map[string]any)["text"].(string)
		if strings.Contains(text, "\u200b") != user {
			t.Fatal("user scope not explicit")
		}
		if blocks[2].(map[string]any)["content"] != "ClientName exact" || d["tools"].([]any)[0].(map[string]any)["name"] != "ClientName" {
			t.Fatal("tool data changed")
		}
	}
}
func TestCompatibilityModes(t *testing.T) {
	raw := []byte(`{"system":"Original identity","messages":[{"role":"user","content":"hello"}]}`)
	wire, e := GatewayRequestWithCompatibility(raw, "claude", Compatibility{Mode: "passthrough"})
	if e != nil || string(wire) != string(raw) {
		t.Fatal("passthrough changed request")
	}
	_, e = GatewayRequestWithCompatibility(raw, "claude", Compatibility{Mode: "typo"})
	if e == nil {
		t.Fatal("invalid mode accepted")
	}
}
func TestObfuscationIdempotentAndLiteral(t *testing.T) {
	c := Compatibility{SensitiveWords: []string{"a.b", "ClientName", " ", "x", "ClientName", "\u200bword"}}
	raw := []byte(`{"system":"a.b ClientName A.B clientname","messages":[]}`)
	one, e := c.obfuscate(raw, "claude")
	if e != nil {
		t.Fatal(e)
	}
	two, e := c.obfuscate(one, "claude")
	if e != nil || string(one) != string(two) {
		t.Fatal("obfuscation not idempotent")
	}
	if !strings.Contains(string(one), "a\u200b.b") {
		t.Fatal("literal matching failed")
	}
}
