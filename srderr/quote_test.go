package srderr_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Whealinghold/srd5e-go/srderr"
)

func TestQuote(t *testing.T) {
	t.Parallel()
	if got := srderr.Quote("a\nb"); got != `"a\nb"` {
		t.Errorf("control chars not escaped: %s", got)
	}
	long := strings.Repeat("é", 40) // 2-byte runes; byte 32 is a rune boundary, 31 is not
	for _, s := range []string{long, "x" + long} {
		got := srderr.Quote(s)
		if !utf8.ValidString(got) || !strings.HasSuffix(got, `..."`) {
			t.Errorf("Quote(%q) = %s", s, got)
		}
	}
}

func TestQuoteTruncationExact(t *testing.T) {
	t.Parallel()
	// Requirement: none - pins current behaviour of Quote's 32-byte limit.
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"31 bytes kept", strings.Repeat("a", 31), `"` + strings.Repeat("a", 31) + `"`},
		{"32 bytes kept", strings.Repeat("a", 32), `"` + strings.Repeat("a", 32) + `"`},
		{"33 bytes cut", strings.Repeat("a", 33), `"` + strings.Repeat("a", 32) + `..."`},
		{"cut on rune boundary", strings.Repeat("é", 40), `"` + strings.Repeat("é", 16) + `..."`},
		{"cut backs up mid-rune", "x" + strings.Repeat("é", 40), `"x` + strings.Repeat("é", 15) + `..."`},
		{"only continuation bytes", strings.Repeat("\x80", 40), `"..."`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := srderr.Quote(tc.in); got != tc.want {
				t.Fatalf("Quote = %s, want %s", got, tc.want)
			}
		})
	}
}
