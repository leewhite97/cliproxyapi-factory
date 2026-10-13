package sse

import (
	"bytes"
	"testing"
)

func TestFragmentedFrames(t *testing.T) {
	d := &Decoder{}
	raw := []byte("event: test\r\ndata: {\"x\":1}\r\n\r\ndata: [DONE]\n\n")
	var frames [][]byte
	for _, b := range raw {
		frames = append(frames, d.Feed([]byte{b})...)
	}
	if len(frames) != 2 || !bytes.Equal(bytes.Join(frames, nil), raw) {
		t.Fatal("lost frame bytes")
	}
}
