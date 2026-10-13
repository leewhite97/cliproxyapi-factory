package main

import (
	"context"
	"encoding/json"
	"factory.local/cliproxyapi-factory/internal/provider"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"net/http"
)

type hostTransport struct{}

func hostRequest(method, callback, url string, h http.Header, b []byte) any {
	if method == "" {
		method = "POST"
	}
	return map[string]any{"host_callback_id": callback, "method": method, "url": url, "headers": h, "body": b}
}

func (hostTransport) Do(ctx context.Context, method, callback, url string, h http.Header, b []byte) (int, http.Header, []byte, error) {
	raw, e := callHost(pluginabi.MethodHostHTTPDo, hostRequest(method, callback, url, h, b))
	if e != nil {
		return 0, nil, nil, e
	}
	var r struct {
		StatusCode int
		Headers    http.Header
		Body       []byte
	}
	e = json.Unmarshal(raw, &r)
	return r.StatusCode, r.Headers, r.Body, e
}

func (hostTransport) Open(ctx context.Context, callback, url string, h http.Header, b []byte) (provider.Stream, error) {
	raw, e := callHost(pluginabi.MethodHostHTTPDoStream, hostRequest("POST", callback, url, h, b))
	if e != nil {
		return provider.Stream{}, e
	}
	var r struct {
		Status  int         `json:"status_code"`
		Headers http.Header `json:"headers"`
		ID      string      `json:"stream_id"`
	}
	e = json.Unmarshal(raw, &r)
	return provider.Stream{Status: r.Status, Headers: r.Headers, ID: r.ID}, e
}

func (hostTransport) Read(ctx context.Context, id string) (provider.Chunk, error) {
	raw, e := callHost(pluginabi.MethodHostHTTPStreamRead, map[string]any{"stream_id": id})
	if e != nil {
		return provider.Chunk{}, e
	}
	var ch provider.Chunk
	e = json.Unmarshal(raw, &ch)
	return ch, e
}

func (hostTransport) Close(ctx context.Context, id string) error {
	_, e := callHost(pluginabi.MethodHostHTTPStreamClose, map[string]any{"stream_id": id})
	return e
}

func (hostTransport) Emit(ctx context.Context, id string, b []byte) error {
	_, e := callHost(pluginabi.MethodHostStreamEmit, map[string]any{"stream_id": id, "payload": b})
	return e
}

func (hostTransport) End(ctx context.Context, id, e string) {
	callHost(pluginabi.MethodHostStreamClose, map[string]any{"stream_id": id, "error": e})
}
