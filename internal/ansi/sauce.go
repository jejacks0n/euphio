package ansi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// SAUCE Record Structure (128 bytes)
// ID      [5]byte  // "SAUCE"
// Version [2]byte  // "00"
// Title   [35]byte
// Author  [20]byte
// Group   [20]byte
// Date    [8]byte  // YYYYMMDD
// FileSize int32
// DataType byte
// FileType byte
// TInfo1   uint16
// TInfo2   uint16
// TInfo3   uint16
// TInfo4   uint16
// Comments byte    // Number of comment lines
// Flags    byte
// Filler   [22]byte

const (
	SauceIDLen  = 5
	SauceRecLen = 128
)

var (
	SauceID    = []byte("SAUCE")
	ErrNoSauce = errors.New("no SAUCE record found")
)

type Sauce struct {
	Title    string
	Author   string
	Group    string
	Date     string
	DataType byte
	FileType byte
	TInfo1   uint16
	TInfo2   uint16
	TInfo3   uint16
	TInfo4   uint16
	Comments []string
	Flags    byte
}

// StripSauce removes the SAUCE record and comments from the data
func StripSauce(data []byte) []byte {
	if len(data) < SauceRecLen {
		return data
	}

	// Check for SAUCE record at the end
	end := len(data)
	recStart := end - SauceRecLen

	if !bytes.Equal(data[recStart:recStart+SauceIDLen], SauceID) {
		return data
	}

	// Parse comments count to find where the actual data ends
	commentsCount := int(data[recStart+104])

	// Comment block is 64 bytes per line + 5 bytes ID "COMNT"
	// But the ID is only present if comments > 0

	trimLen := SauceRecLen
	if commentsCount > 0 {
		// 5 bytes for "COMNT" + (64 * lines)
		trimLen += 5 + (64 * commentsCount)
	}

	// Ensure we don't trim more than we have
	if trimLen > len(data) {
		return []byte{}
	}

	// Check for EOF marker (0x1A) often placed before SAUCE
	// Some editors put it there, some don't.
	// If the byte before the stripped part is 0x1A (SUB), strip it too.
	contentEnd := len(data) - trimLen
	if contentEnd > 0 && data[contentEnd-1] == 0x1A {
		contentEnd--
	}

	return data[:contentEnd]
}

// ParseSauce extracts the SAUCE record from the data
func ParseSauce(data []byte) (*Sauce, error) {
	if len(data) < SauceRecLen {
		return nil, ErrNoSauce
	}

	recStart := len(data) - SauceRecLen
	if !bytes.Equal(data[recStart:recStart+SauceIDLen], SauceID) {
		return nil, ErrNoSauce
	}

	r := bytes.NewReader(data[recStart:])

	// Skip ID (5) and Version (2)
	r.Seek(7, io.SeekStart)

	readString := func(len int) string {
		buf := make([]byte, len)
		r.Read(buf)
		return string(bytes.TrimRight(buf, "\x00 "))
	}

	s := &Sauce{
		Title:  readString(35),
		Author: readString(20),
		Group:  readString(20),
		Date:   readString(8),
	}

	// Skip FileSize (4)
	r.Seek(4, io.SeekCurrent)

	binary.Read(r, binary.LittleEndian, &s.DataType)
	binary.Read(r, binary.LittleEndian, &s.FileType)
	binary.Read(r, binary.LittleEndian, &s.TInfo1)
	binary.Read(r, binary.LittleEndian, &s.TInfo2)
	binary.Read(r, binary.LittleEndian, &s.TInfo3)
	binary.Read(r, binary.LittleEndian, &s.TInfo4)

	var commentsCount byte
	binary.Read(r, binary.LittleEndian, &commentsCount)
	binary.Read(r, binary.LittleEndian, &s.Flags)

	// If there are comments, we need to read them from before the record
	if commentsCount > 0 {
		// Comment block ends right before the SAUCE record
		// Format: "COMNT" + (64 bytes * count)
		commentBlockLen := 5 + (64 * int(commentsCount))
		commentStart := recStart - commentBlockLen

		if commentStart >= 0 && bytes.Equal(data[commentStart:commentStart+5], []byte("COMNT")) {
			s.Comments = make([]string, commentsCount)
			cr := bytes.NewReader(data[commentStart+5:])
			for i := 0; i < int(commentsCount); i++ {
				buf := make([]byte, 64)
				cr.Read(buf)
				s.Comments[i] = string(bytes.TrimRight(buf, "\x00 "))
			}
		}
	}

	return s, nil
}
