// SSE framing adapted from arthur-sommer-etc/cliproxyapi-copilot-plugin (MIT).
package sse

import "bytes"

type Decoder struct{ buffer []byte }

func (d *Decoder) Feed(chunk []byte) [][]byte {
	d.buffer = append(d.buffer, chunk...)
	var out [][]byte
	for {
		a := bytes.Index(d.buffer, []byte("\n\n"))
		b := bytes.Index(d.buffer, []byte("\r\n\r\n"))
		width := 2
		if a < 0 || (b >= 0 && b < a) {
			a = b
			width = 4
		}
		if a < 0 {
			break
		}
		out = append(out, append([]byte(nil), d.buffer[:a+width]...))
		d.buffer = append(d.buffer[:0], d.buffer[a+width:]...)
	}
	return out
}
func (d *Decoder) Flush() []byte { b := d.buffer; d.buffer = nil; return b }
