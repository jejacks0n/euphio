package telnet

import (
	"bytes"
	"io"
)

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	// If there are no IAC bytes, just write directly
	if bytes.IndexByte(p, IAC) == -1 {
		return w.w.Write(p)
	}

	// Otherwise, we need to escape IAC -> IAC IAC
	var buf bytes.Buffer
	// Pre-allocate to avoid too many reallocations (length + approx 10% overhead guess)
	buf.Grow(len(p) + len(p)/10)

	for _, b := range p {
		buf.WriteByte(b)
		if b == IAC {
			buf.WriteByte(IAC)
		}
	}

	// Write the escaped buffer
	// Note: The return value n should technically be the number of bytes from p consumed.
	// Since we consume all of p to write to buf, if the write to w.w succeeds, we return len(p).
	_, err = w.w.Write(buf.Bytes())
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// WriteCommand sends a Telnet command sequence.
// It automatically prepends IAC.
// Example: WriteCommand(WILL, ECHO) sends IAC WILL ECHO
func (w *Writer) WriteCommand(cmds ...byte) error {
	data := make([]byte, 1+len(cmds))
	data[0] = IAC
	copy(data[1:], cmds)
	_, err := w.w.Write(data)
	return err
}

// WriteSubNegotiation sends a sub-negotiation sequence.
// It automatically wraps the data in IAC SB ... IAC SE.
func (w *Writer) WriteSubNegotiation(option byte, data []byte) error {
	// IAC SB OPTION <DATA> IAC SE
	buf := make([]byte, 0, 5+len(data))
	buf = append(buf, IAC, SB, option)
	buf = append(buf, data...)
	buf = append(buf, IAC, SE)

	_, err := w.w.Write(buf)
	return err
}
