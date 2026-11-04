package execution

import "bytes"

type Output struct {
	buffer      bytes.Buffer
	Truncated   bool
	OnLine      func(string)
	line        []byte
	discardLine bool
}

func (b *Output) Len() int      { return b.buffer.Len() }
func (b *Output) Bytes() []byte { return b.buffer.Bytes() }

func (b *Output) Write(p []byte) (int, error) {
	n := len(p)
	if b.OnLine != nil {
		for _, value := range p {
			if value == '\n' {
				if !b.discardLine {
					b.OnLine(string(b.line))
				}
				b.line = b.line[:0]
				b.discardLine = false
				continue
			}
			if len(b.line) < 65536 && !b.discardLine {
				b.line = append(b.line, value)
			} else {
				b.discardLine = true
				b.line = b.line[:0]
			}
		}
	}
	remaining := (2 << 20) - b.Len()
	if len(p) > remaining {
		p = p[:max(remaining, 0)]
		b.Truncated = true
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}

func (b *Output) PendingLineBytes() int { return len(b.line) }
