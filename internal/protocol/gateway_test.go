package protocol

import (
	"encoding/json"
	"testing"
)

func TestGatewayIdentity(t *testing.T) {
	for _, f := range []string{"claude", "openai-response", "openai"} {
		t.Run(f, func(t *testing.T) {
			body := []byte(`{"system":"custom rules","messages":[{"role":"user","content":"Hi"}],"instructions":"custom rules","input":"Hi","tools":[{"name":"get_weather"}]}`)
			b, e := GatewayRequest(body, f)
			if e != nil {
				t.Fatal(e)
			}
			var d map[string]any
			json.Unmarshal(b, &d)
			switch f {
			case "claude":
				if d["system"] != DroidPrefix {
					t.Fatal("required identity stripped")
				}
			case "openai-response":
				if d["instructions"] != DroidPrefix {
					t.Fatal("required identity stripped")
				}
			case "openai":
				m := d["messages"].([]any)
				if len(m) == 0 || m[0].(map[string]any)["role"] != "system" || m[0].(map[string]any)["content"] != DroidPrefix {
					t.Fatal("required identity stripped")
				}
			}
			if len(d["tools"].([]any)) != 1 {
				t.Fatal("tools lost")
			}
		})
	}
}
