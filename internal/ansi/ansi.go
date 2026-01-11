package ansi

import (
	"io"
	"strings"
)

const (
	ResetSeq = "\x1b[0m"
)

// Connection is a minimal interface for what ansi.Print needs to know about the connection.
// This avoids a circular dependency on the nodes package.
type Connection interface {
	IsUTF8() bool
}

// PrepareForOutput processes raw ANSI data for display.
// It detects if the data is likely CP437 and converts it to UTF-8 if necessary.
func PrepareForOutput(data []byte, forceUTF8 bool) []byte {
	// Remove SAUCE record
	cleanData := StripSauce(data)

	var s string
	if forceUTF8 {
		// If the client needs UTF-8 (like a modern terminal), decode CP437 to UTF-8
		s = DecodeCP437(cleanData)
	} else {
		// If the client is legacy (like SyncTerm), send raw bytes
		s = string(cleanData)
	}

	// Normalize line endings:
	// 1. Replace CRLF with LF (to avoid double CRs if we just did LF->CRLF)
	// 2. Replace LF with CRLF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")

	return []byte(s)
}

// Print writes the prepared ANSI data to the given writer.
// It automatically strips SAUCE metadata, normalizes line endings,
// and appends a reset sequence to restore terminal state.
//
// It uses the provided Connection to determine if UTF-8 conversion is needed.
func Print(w io.Writer, data []byte, conn Connection) (int, error) {
	prepared := PrepareForOutput(data, conn.IsUTF8())
	prepared = append(prepared, []byte(ResetSeq)...)
	return w.Write(prepared)
}
