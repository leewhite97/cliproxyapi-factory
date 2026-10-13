package main

import (
	"encoding/json"
	"factory.local/cliproxyapi-factory/internal/protocol"
	"factory.local/cliproxyapi-factory/internal/provider"
	"gopkg.in/yaml.v3"
	"sync"
)

var configMu sync.RWMutex
var modelCatalog = &provider.CatalogCache{}
var compatibility = protocol.Compatibility{}

func currentCompatibility() protocol.Compatibility {
	configMu.RLock()
	defer configMu.RUnlock()
	c := compatibility
	c.SensitiveWords = append([]string(nil), c.SensitiveWords...)
	return c
}

func configure(raw []byte) error {
	var req struct {
		ConfigYAML []byte `json:"config_yaml"`
	}
	if len(raw) > 0 {
		if e := json.Unmarshal(raw, &req); e != nil {
			return e
		}
	}
	var cfg struct {
		Path          string                 `yaml:"catalog_cache_path"`
		Compatibility protocol.Compatibility `yaml:"compatibility"`
	}
	if e := yaml.Unmarshal(req.ConfigYAML, &cfg); e != nil {
		return e
	}
	if e := cfg.Compatibility.Validate(); e != nil {
		return e
	}
	configMu.Lock()
	defer configMu.Unlock()
	compatibility = cfg.Compatibility
	if modelCatalog.Path != cfg.Path {
		modelCatalog = &provider.CatalogCache{Path: cfg.Path}
	}
	return nil
}
