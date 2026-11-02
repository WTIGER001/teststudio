package runner

import "bytes"

type output struct {
	buffer    bytes.Buffer
	line      []byte
	discard   bool
	oversized bool
	truncated bool
	onLine    func(string)
}

func (o *output) Write(data []byte) (int, error) {
	n := len(data)
	if o.onLine != nil {
		for _, b := range data {
			if b == '\n' {
				if !o.discard {
					o.onLine(string(o.line))
				}
				o.line = o.line[:0]
				o.discard = false
				continue
			}
			if !o.discard && len(o.line) < 256<<10 {
				o.line = append(o.line, b)
			} else {
				o.discard = true
				o.oversized = true
				o.line = o.line[:0]
				o.truncated = true
			}
		}
	}
	remaining := (2 << 20) - o.buffer.Len()
	if len(data) > remaining {
		data = data[:max(0, remaining)]
		o.truncated = true
	}
	_, _ = o.buffer.Write(data)
	return n, nil
}
