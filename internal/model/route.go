package model

import (
	"fmt"
	"strings"
)

type Route struct{ Format, Path, Provider string }

func Resolve(id string) (Route, error) {
	switch {
	case strings.HasPrefix(id, "claude-"):
		return Route{"claude", "/api/llm/a/v1/messages", "anthropic"}, nil
	case strings.HasPrefix(id, "minimax-"):
		return Route{"claude", "/api/llm/a/v1/messages", "fireworks"}, nil
	case strings.HasPrefix(id, "gpt-") || strings.HasSuffix(id, "-codex"):
		return Route{"openai-response", "/api/llm/o/v1/responses", "openai"}, nil
	default:
		for _, p := range []string{"glm-", "kimi-", "deepseek-", "nemotron-"} {
			if strings.HasPrefix(id, p) {
				return Route{"openai", "/api/llm/o/v1/chat/completions", "fireworks"}, nil
			}
		}
	}
	return Route{}, fmt.Errorf("factory: unsupported model")
}
