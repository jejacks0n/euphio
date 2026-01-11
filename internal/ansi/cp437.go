package ansi

import "strings"

// CP437ToUnicode maps the upper 128 bytes of CP437 to Unicode runes.
// Indices 0-127 correspond to 0x80-0xFF.
var cp437ToUnicode = [128]rune{
	'\u00C7', '\u00FC', '\u00E9', '\u00E2', '\u00E4', '\u00E0', '\u00E5', '\u00E7', // 80-87
	'\u00EA', '\u00EB', '\u00E8', '\u00EF', '\u00EE', '\u00EC', '\u00C4', '\u00C5', // 88-8F
	'\u00C9', '\u00E6', '\u00C6', '\u00F4', '\u00F6', '\u00F2', '\u00FB', '\u00F9', // 90-97
	'\u00FF', '\u00D6', '\u00DC', '\u00A2', '\u00A3', '\u00A5', '\u20A7', '\u0192', // 98-9F
	'\u00E1', '\u00ED', '\u00F3', '\u00FA', '\u00F1', '\u00D1', '\u00AA', '\u00BA', // A0-A7
	'\u00BF', '\u2310', '\u00AC', '\u00BD', '\u00BC', '\u00A1', '\u00AB', '\u00BB', // A8-AF
	'\u2591', '\u2592', '\u2593', '\u2502', '\u2524', '\u2561', '\u2562', '\u2556', // B0-B7
	'\u2555', '\u2563', '\u2551', '\u2557', '\u255D', '\u255C', '\u255B', '\u2510', // B8-BF
	'\u2514', '\u2534', '\u252C', '\u251C', '\u2500', '\u253C', '\u255E', '\u255F', // C0-C7
	'\u255A', '\u2554', '\u2569', '\u2566', '\u2560', '\u2550', '\u256C', '\u2567', // C8-CF
	'\u2568', '\u2564', '\u2565', '\u2559', '\u2558', '\u2552', '\u2553', '\u256B', // D0-D7
	'\u256A', '\u2518', '\u250C', '\u2588', '\u2584', '\u258C', '\u2590', '\u2580', // D8-DF
	'\u03B1', '\u00DF', '\u0393', '\u03C0', '\u03A3', '\u03C3', '\u00B5', '\u03C4', // E0-E7
	'\u03A6', '\u0398', '\u03A9', '\u03B4', '\u221E', '\u03C6', '\u03B5', '\u2229', // E8-EF
	'\u2261', '\u00B1', '\u2265', '\u2264', '\u2320', '\u2321', '\u00F7', '\u2248', // F0-F7
	'\u00B0', '\u2219', '\u00B7', '\u221A', '\u207F', '\u00B2', '\u25A0', '\u00A0', // F8-FF
}

// DecodeCP437 converts CP437 encoded bytes to a UTF-8 string.
func DecodeCP437(data []byte) string {
	var sb strings.Builder
	sb.Grow(len(data))

	for _, b := range data {
		if b < 0x80 {
			sb.WriteByte(b)
		} else {
			sb.WriteRune(cp437ToUnicode[b-0x80])
		}
	}
	return sb.String()
}
