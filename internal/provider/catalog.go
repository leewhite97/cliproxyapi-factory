package provider

import (
	"factory.local/cliproxyapi-factory/internal/protocol"
	"fmt"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"regexp"
	"strings"
)

// Only exact, live-verified retired IDs are excluded; region restrictions are not global.
var deprecatedModelIDs = map[string]bool{
	"claude-opus-4-6-fast": true, "claude-opus-4-7-fast": true, "glm-5.1": true,
	"kimi-k2.5": true, "kimi-k2.6": true, "kimi-k2.7-code": true,
}

var catalogID = regexp.MustCompile("`([^`]+)`")

func ParseModelDocs(body []byte) ([]string, error) {
	seen := map[string]bool{}
	var ids []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 4 {
			continue
		}
		m := catalogID.FindStringSubmatch(cells[2])
		if len(m) != 2 {
			continue
		}
		id := strings.TrimSpace(m[1])
		if _, e := protocol.Resolve(id); e != nil || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("factory: catalog discovery yielded no routable models")
	}
	return ids, nil
}
func mergedCatalog(ids []string) pluginapi.ModelResponse {
	out := Models()
	seen := map[string]bool{}
	for _, m := range out.Models {
		seen[m.ID] = true
	}
	for _, id := range ids {
		wireID := "factory/" + id
		if seen[wireID] || deprecatedModelIDs[id] {
			continue
		}
		if _, e := protocol.Resolve(id); e != nil {
			continue
		}
		out.Models = append(out.Models, modelSpec(id).info())
		seen[wireID] = true
	}
	return out
}
