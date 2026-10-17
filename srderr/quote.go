package srderr

import (
	"strconv"
	"unicode/utf8"
)

const quoteLimit = 32

// Quote returns s as a Go-quoted string for use in error details. Input longer
// than 32 bytes is cut on a rune boundary and marked with "..." inside the
// quotes, so untrusted text is always escaped and valid UTF-8.
func Quote(s string) string {
	if len(s) <= quoteLimit {
		return strconv.Quote(s)
	}
	n := quoteLimit
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return strconv.Quote(s[:n] + "...")
}
