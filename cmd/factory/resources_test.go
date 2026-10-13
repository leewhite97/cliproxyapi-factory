package main

import (
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"strings"
	"testing"
)

func TestEmbeddedScripts(t *testing.T) {
	for _, path := range []string{"quota.js", "panel-auth.js"} {
		request, _ := json.Marshal(pluginapi.ManagementRequest{Method: "GET", Path: "/v0/resource/plugins/factory/" + path})
		value, err := dispatch(pluginabi.MethodManagementHandle, request)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(value)
		var response pluginapi.ManagementResponse
		if err = json.Unmarshal(raw, &response); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 200 || !strings.HasPrefix(response.Headers.Get("Content-Type"), "text/javascript") || len(response.Body) == 0 {
			t.Fatalf("invalid script response for %s", path)
		}
		if !strings.Contains(response.Headers.Get("Content-Security-Policy"), "script-src 'self'") {
			t.Fatal("missing strict script policy")
		}
		for _, forbidden := range []string{"localStorage.setItem", "sessionStorage.setItem", "innerHTML"} {
			if strings.Contains(string(response.Body), forbidden) {
				t.Fatalf("forbidden pattern %s", forbidden)
			}
		}
	}
}
