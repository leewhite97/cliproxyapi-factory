package main

import (
	"embed"
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"net/http"
)

//go:embed web/*
var quotaAssets embed.FS

func managementResource(b []byte) (any, error) {
	var req pluginapi.ManagementRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return nil, err
	}
	resources := map[string]struct{ file, contentType string }{
		"/v0/resource/plugins/factory/quota":         {"web/quota.html", "text/html; charset=utf-8"},
		"/v0/resource/plugins/factory/quota.js":      {"web/quota.js", "text/javascript; charset=utf-8"},
		"/v0/resource/plugins/factory/panel-auth.js": {"web/panel-auth.js", "text/javascript; charset=utf-8"},
	}
	asset, ok := resources[req.Path]
	if req.Method != "GET" || !ok {
		return pluginapi.ManagementResponse{StatusCode: 404, Body: []byte("Not found")}, nil
	}
	body, err := quotaAssets.ReadFile(asset.file)
	if err != nil {
		return nil, err
	}
	return pluginapi.ManagementResponse{StatusCode: 200, Headers: http.Header{
		"Content-Type": {asset.contentType}, "Cache-Control": {"no-store"}, "X-Content-Type-Options": {"nosniff"}, "Referrer-Policy": {"no-referrer"},
		"Content-Security-Policy": {"default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'"},
	}, Body: body}, nil
}
