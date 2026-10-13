package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"factory.local/cliproxyapi-factory/internal/protocol"
	"factory.local/cliproxyapi-factory/internal/redact"
	"factory.local/cliproxyapi-factory/internal/sse"
	"fmt"
	tr "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
	"net/http"
	"strings"
)

type Stream struct {
	Status  int
	Headers http.Header
	ID      string
}
type Chunk struct {
	Payload []byte
	Error   string
	Done    bool
}
type StreamHost interface {
	Open(context.Context, string, string, http.Header, []byte) (Stream, error)
	Read(context.Context, string) (Chunk, error)
	Close(context.Context, string) error
	Emit(context.Context, string, []byte) error
	End(context.Context, string, string)
}

func (s *Service) ExecuteStream(ctx context.Context, r Request) (http.Header, error) {
	s.init()
	host, ok := s.Host.(StreamHost)
	if !ok {
		return nil, fmt.Errorf("factory: streaming transport unavailable")
	}
	if r.StreamID == "" {
		return nil, fmt.Errorf("factory: output stream ID missing")
	}
	c, route, model, b, e := s.prepare(r, true)
	if e != nil {
		return nil, e
	}
	h := protocol.HeadersWithSession(c.Key, c.Org, route, requestSession(r))
	h.Set("Accept", "text/event-stream")
	up, e := host.Open(ctx, r.HostCallbackID, c.Endpoint+route.Path, h, b)
	if e != nil {
		return nil, transportError(e, c.Key)
	}
	if up.Status < 200 || up.Status >= 300 {
		defer host.Close(ctx, up.ID)
		var body []byte
		for len(body) < 4096 {
			ch, e := host.Read(ctx, up.ID)
			if e != nil {
				break
			}
			body = append(body, ch.Payload...)
			if ch.Done || ch.Error != "" {
				break
			}
		}
		capture403(up.Status, r, route, b, body, c.Key)
		return nil, upstreamError(up.Status, body, c.Key)
	}
	format := r.Format
	if format == "" {
		format = r.SourceFormat
	}
	if err := s.registerStream(up.ID, host); err != nil {
		host.Close(context.Background(), up.ID)
		return nil, err
	}
	go s.pump(s.ctx, host, r.StreamID, up.ID, route.Format, format, model, r.OriginalRequest, b, c.Key)
	return http.Header{"Content-Type": []string{"text/event-stream"}, "Cache-Control": []string{"no-cache"}}, nil
}
func (s *Service) pump(ctx context.Context, h StreamHost, outID, upID, from, to, model string, original, request []byte, key string) {
	defer s.releaseStream(upID)
	terminal := ""
	finished := false
	defer func() {
		cleanup := context.Background()
		h.Close(cleanup, upID)
		h.End(cleanup, outID, redact.Text(terminal, key))
	}()
	d := &sse.Decoder{}
	var state any
	emit := func(frame []byte) error {
		norm := bytes.ReplaceAll(frame, []byte("\r\n"), []byte("\n"))
		var data [][]byte
		for _, line := range bytes.Split(norm, []byte("\n")) {
			if bytes.HasPrefix(line, []byte("data:")) {
				data = append(data, bytes.TrimSpace(line[5:]))
			}
		}
		payload := bytes.Join(data, []byte("\n"))
		if len(payload) == 0 {
			return nil
		}
		var event map[string]any
		if bytes.Equal(payload, []byte("[DONE]")) {
			finished = true
		} else if json.Unmarshal(payload, &event) != nil {
			return fmt.Errorf("factory: malformed SSE JSON")
		} else {
			typ, _ := event["type"].(string)
			if typ == "error" || typ == "response.failed" || event["error"] != nil {
				return fmt.Errorf("factory: upstream stream error: %s", redact.Text(string(payload), key))
			}
			if isTerminalEvent(typ) {
				finished = true
			}
		}
		var chunks [][]byte
		if from == to {
			if to == "openai" {
				if bytes.Equal(payload, []byte("[DONE]")) {
					return nil
				}
				var compact bytes.Buffer
				if err := json.Compact(&compact, payload); err != nil {
					return err
				}
				chunks = [][]byte{compact.Bytes()}
			} else {
				chunks = [][]byte{frame}
			}
		} else {
			f, t := tr.FromString(from), tr.FromString(to)
			if from == "openai-response" && !registry.HasStreamResponseTransformer(t, f) {
				f = tr.FormatCodex
			}
			if !registry.HasStreamResponseTransformer(t, f) {
				return fmt.Errorf("factory: unsupported streaming translation %s -> %s", from, to)
			}
			chunks = registry.TranslateStream(ctx, f, t, model, original, request, append([]byte("data: "), payload...), &state)
		}
		for _, chunk := range chunks {
			if len(chunk) > 0 {
				if e := h.Emit(ctx, outID, chunk); e != nil {
					return e
				}
			}
		}
		return nil
	}
	for {
		ch, e := h.Read(ctx, upID)
		if e != nil {
			terminal = "factory: stream interrupted: " + redact.Text(e.Error(), key)
			return
		}
		if ch.Error != "" {
			terminal = "factory: stream interrupted: " + redact.Text(ch.Error, key)
			return
		}
		for _, frame := range d.Feed(ch.Payload) {
			if e := emit(frame); e != nil {
				terminal = e.Error()
				return
			}
		}
		if ch.Done {
			if trailing := d.Flush(); len(bytes.TrimSpace(trailing)) > 0 {
				terminal = "factory: incomplete SSE frame"
				return
			}
			if !finished {
				terminal = "factory: stream interrupted before terminal event"
			}
			return
		}
	}
}

var _ = strings.TrimSpace
