package main

import (
	"encoding/json"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"strings"
	"testing"
)

func TestManagementResource(t *testing.T) {
	v, err := dispatch(pluginabi.MethodManagementRegister, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(v)
	var reg pluginapi.ManagementRegistrationResponse
	if err = json.Unmarshal(data, &reg); err != nil {
		t.Fatal(err)
	}
	if len(reg.Resources) != 3 || reg.Resources[0].Path != "/quota" || reg.Resources[0].Menu == "" {
		t.Fatalf("missing resource: %s", data)
	}
	req, _ := json.Marshal(pluginapi.ManagementRequest{Method: "GET", Path: "/v0/resource/plugins/factory/quota"})
	v, err = dispatch(pluginabi.MethodManagementHandle, req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(v)
	var res pluginapi.ManagementResponse
	if err = json.Unmarshal(data, &res); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || !strings.Contains(string(res.Body), "Factory Quota") {
		t.Fatal("missing page")
	}
	for _, bad := range []string{"localStorage", "https://cdn", "innerHTML", "fk-"} {
		if strings.Contains(string(res.Body), bad) {
			t.Fatalf("unsafe page marker %s", bad)
		}
	}
	req, _ = json.Marshal(pluginapi.ManagementRequest{Method: "GET", Path: "/v0/resource/plugins/factory/not-found"})
	v, err = dispatch(pluginabi.MethodManagementHandle, req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(v)
	json.Unmarshal(data, &res)
	if res.StatusCode != 404 {
		t.Fatal("unknown resource must return 404")
	}
}
