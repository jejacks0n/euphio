package telnet

import (
	"bytes"
	"io"
)

type CommandHandler interface {
	HandleCommand(cmd, option byte)
	HandleSubNegotiation(option byte, data []byte)
}

type Reader struct {
	r       io.Reader
	buf     bytes.Buffer // Buffer for incoming data from the socket
	dataBuf bytes.Buffer // Buffer for processed user data waiting to be read
	handler CommandHandler
}

func NewReader(r io.Reader, handler CommandHandler) *Reader {
	return &Reader{
		r:       r,
		handler: handler,
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	// If we have processed data ready, return it
	if r.dataBuf.Len() > 0 {
		return r.dataBuf.Read(p)
	}

	// Read more data from the underlying reader
	buf := make([]byte, 4096)
	n, err = r.r.Read(buf)
	if n > 0 {
		r.buf.Write(buf[:n])
		r.processCommands()
	}

	// If we still have no data after processing, and there was no error,
	// we might need to read again (handled by the caller usually, but
	// strictly speaking Read should block if there is no data but no error).
	// However, for now, let's just return what we have.

	if r.dataBuf.Len() > 0 {
		return r.dataBuf.Read(p)
	}

	return 0, err
}

func (r *Reader) processCommands() {
	for {
		// Look for IAC
		data := r.buf.Bytes()
		iacIndex := bytes.IndexByte(data, IAC)

		if iacIndex == -1 {
			// No IAC found, everything is data
			r.dataBuf.Write(r.buf.Next(r.buf.Len()))
			return
		}

		// Write data before IAC to dataBuf
		if iacIndex > 0 {
			r.dataBuf.Write(r.buf.Next(iacIndex))
			// Refresh data slice after consuming bytes
			data = r.buf.Bytes()
		}

		// We are now at IAC. Check if we have enough bytes to determine the command.
		if len(data) < 2 {
			// Not enough data yet, wait for more
			return
		}

		commandCode := data[1]

		// Handle escaped IAC (IAC IAC) -> single byte 255
		if commandCode == IAC {
			r.dataBuf.WriteByte(IAC)
			r.buf.Next(2) // Consume IAC IAC
			continue
		}

		// Handle Commands
		switch commandCode {
		case WILL, WONT, DO, DONT:
			if len(data) < 3 {
				// Need option byte
				return
			}
			option := data[2]
			if r.handler != nil {
				r.handler.HandleCommand(commandCode, option)
			}
			r.buf.Next(3) // Consume IAC COMMAND OPTION

		case SB:
			// Sub-negotiation: IAC SB OPTION ... IAC SE
			// Find IAC SE
			seIndex := bytes.Index(data, []byte{IAC, SE})
			if seIndex == -1 {
				// Complete sub-negotiation not received yet
				return
			}

			// Extract sub-negotiation data
			// IAC (0) SB (1) OPTION (2) ... IAC (seIndex) SE (seIndex+1)
			// Length to consume is seIndex + 2

			option := data[2]
			subData := data[3:seIndex]
			if r.handler != nil {
				r.handler.HandleSubNegotiation(option, subData)
			}

			r.buf.Next(seIndex + 2)

		default:
			// Simple commands (IAC NOP, IAC GA, etc.)
			if r.handler != nil {
				r.handler.HandleCommand(commandCode, 0)
			}
			r.buf.Next(2) // Consume IAC COMMAND
		}
	}
}
