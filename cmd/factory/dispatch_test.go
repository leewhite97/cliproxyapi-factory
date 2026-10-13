package main

import (
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"testing"
)

func TestMetadataWireCompatibility(t *testing.T) {
	b, _ := json.Marshal(registration())
	var r struct {
		Metadata pluginapi.Metadata `json:"metadata"`
	}
	if json.Unmarshal(b, &r) != nil || r.Metadata.GitHubRepository == "" || r.Metadata.Name == "" {
		t.Fatal("host cannot decode plugin metadata")
	}
}
