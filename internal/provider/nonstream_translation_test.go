package provider

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNonStreamToolNativeTranslation(t *testing.T) {
	cases := []struct{ from, body string }{{"claude", "data: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"model\":\"claude-sonnet-4-6\",\"usage\":{\"input_tokens\":5}}}\n\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"tool1\",\"name\":\"get_weather\",\"input\":{}}}\n\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"city\\\":\\\"Paris\\\"}\"}}\n\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":7}}\n\ndata: {\"type\":\"message_stop\"}\n\n"}, {"openai-response", "event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"r1\",\"model\":\"gpt-5.3-codex\",\"status\":\"completed\",\"output\":[{\"type\":\"function_call\",\"call_id\":\"tool1\",\"name\":\"get_weather\",\"arguments\":\"{\\\"city\\\":\\\"Paris\\\"}\"}],\"usage\":{\"input_tokens\":5,\"output_tokens\":7,\"total_tokens\":12}}}\n\n"}}
	for _, c := range cases {
		t.Run(c.from, func(t *testing.T) {
			b, e := response(context.Background(), c.from, "openai", "model", []byte(`{"tools":[{"type":"function","function":{"name":"get_weather"}}]}`), nil, []byte(c.body))
			if e != nil {
				t.Fatal(e)
			}
			var d struct {
				Choices []struct {
					Message struct {
						ToolCalls []struct {
							ID       string                           `json:"id"`
							Function struct{ Name, Arguments string } `json:"function"`
						} `json:"tool_calls"`
					} `json:"message"`
				} `json:"choices"`
			}
			if json.Unmarshal(b, &d) != nil || len(d.Choices) != 1 || len(d.Choices[0].Message.ToolCalls) != 1 {
				t.Fatal("tool lost")
			}
			tool := d.Choices[0].Message.ToolCalls[0]
			var params map[string]string
			json.Unmarshal([]byte(tool.Function.Arguments), &params)
			if tool.Function.Name != "get_weather" || params["city"] != "Paris" || tool.ID == "" {
				t.Fatal("tool data corrupted")
			}
		})
	}
}
