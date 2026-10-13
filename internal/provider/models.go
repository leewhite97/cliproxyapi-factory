package provider

import (
	_ "embed"
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

//go:embed catalog.json
var catalog []byte

func Models() pluginapi.ModelResponse {
	var entries []struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Input     []string `json:"input"`
		Context   int64    `json:"contextWindow"`
		Tokens    int64    `json:"maxTokens"`
		Reasoning bool     `json:"reasoning"`
	}
	json.Unmarshal(catalog, &entries)
	out := pluginapi.ModelResponse{Provider: "factory"}
	for _, e := range entries {
		if deprecatedModelIDs[e.ID] {
			continue
		}
		m := modelSpec(e.ID).info()
		out.Models = append(out.Models, m)
	}
	return out
}
