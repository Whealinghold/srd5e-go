package dice

import "strings"

// notation holds the digit runs and sign found by scan.
type notation struct {
	count, sides, modifier string
	negative               bool
}

// minusSign is U+2212, the sign the SRD prints for negative modifiers.
const minusSign = "−"

func digitsEnd(s string, i int) int {
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// scan splits dice notation into tokens: [count]d|D sides [sign modifier],
// where the sign is +, - or U+2212 and spaces are allowed on both sides of the
// sign or on neither. It reports false for anything else.
func scan(s string) (notation, bool) {
	var n notation
	i := digitsEnd(s, 0)
	n.count = s[:i]
	if i >= len(s) || (s[i] != 'd' && s[i] != 'D') {
		return n, false
	}
	i++
	j := digitsEnd(s, i)
	if j == i {
		return n, false
	}
	n.sides, i = s[i:j], j
	if i == len(s) {
		return n, true
	}

	spaced := s[i] == ' '
	if spaced {
		i++
	}
	switch {
	case strings.HasPrefix(s[i:], "+"):
		i++
	case strings.HasPrefix(s[i:], "-"):
		i, n.negative = i+1, true
	case strings.HasPrefix(s[i:], minusSign):
		i, n.negative = i+len(minusSign), true
	default:
		return n, false
	}
	if spaced {
		if i >= len(s) || s[i] != ' ' {
			return n, false
		}
		i++
	}
	j = digitsEnd(s, i)
	if j == i || j != len(s) {
		return n, false
	}
	n.modifier = s[i:j]
	return n, true
}
