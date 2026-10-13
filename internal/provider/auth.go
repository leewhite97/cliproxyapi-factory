package provider

import (
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func ParseAuth(r pluginapi.AuthParseRequest) (pluginapi.AuthParseResponse, error) {
	var d map[string]any
	if json.Unmarshal(r.RawJSON, &d) != nil || d["type"] != "factory" {
		return pluginapi.AuthParseResponse{Handled: false}, nil
	}
	c, e := credential(r.RawJSON)
	if e != nil {
		return pluginapi.AuthParseResponse{}, e
	}
	id := r.FileName
	if id == "" {
		id = "factory-account.json"
	}
	label := c.Label
	if label == "" {
		label = id
	}
	return pluginapi.AuthParseResponse{Handled: true, Auth: pluginapi.AuthData{Provider: "factory", ID: id, FileName: id, Label: label, Disabled: c.Disabled, StorageJSON: r.RawJSON, Metadata: map[string]any{"type": "factory", "label": label}, Attributes: map[string]string{"auth_kind": "api_key"}}}, nil
}
