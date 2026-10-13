package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Built-in Claude nonstream translators consume complete Messages SSE.
// Built-in Codex nonstream translators consume a terminal Responses event.
func nativeNonStreamInput(from, to string, body []byte) ([]byte, error) {
	if from == to || from == "openai" {
		if !json.Valid(body) {
			return nil, fmt.Errorf("factory: invalid upstream JSON")
		}
		return body, nil
	}
	var terminal []byte
	stopped := false
	for _, line := range bytes.Split(body, []byte("\n")) {
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(line[5:])
		if bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		var event struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &event) != nil {
			return nil, fmt.Errorf("factory: invalid upstream SSE JSON")
		}
		switch event.Type {
		case "error", "response.failed":
			return nil, fmt.Errorf("factory: upstream stream failed")
		case "message_stop":
			stopped = true
		case "response.completed", "response.incomplete":
			terminal = bytes.Clone(data)
		}
	}
	switch from {
	case "claude":
		if !stopped {
			return nil, fmt.Errorf("factory: Messages stream interrupted before terminal event")
		}
		return body, nil
	case "openai-response":
		if len(terminal) == 0 {
			return nil, fmt.Errorf("factory: Responses stream interrupted before terminal event")
		}
		return terminal, nil
	}
	return nil, fmt.Errorf("factory: unsupported nonstream protocol")
}
