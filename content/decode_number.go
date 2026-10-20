package content

import (
	"encoding/json"
	"strings"
)

// MaxSafeInteger is the largest integer magnitude the decoder accepts, 2^53 - 1
// (9007199254740991): the largest integer a float64 holds exactly. A whole
// number beyond it is outOfRange, so a decoded integer never depends on the
// width of the platform's int (schema FR-021).
const MaxSafeInteger = 1<<53 - 1

// maxExponent bounds the exponent read from a number literal. A larger one
// cannot change the verdict (a non-zero mantissa is then far beyond the safe
// range, or fractional), and the clamp keeps every sum in an int64 whatever
// the platform.
const maxExponent = 1_000_000_000

// numberClass is the verdict of [classifyNumber].
type numberClass uint8

const (
	// numberWhole: the value is a whole number with magnitude at most
	// MaxSafeInteger.
	numberWhole numberClass = iota
	// numberFractional: the value is not a whole number.
	numberFractional
	// numberTooLarge: the value is whole with magnitude above MaxSafeInteger.
	numberTooLarge
)

// classifyNumber decides whether the JSON number literal num is a whole
// number, and its value when it is within the safe range. The decision is made
// on the decimal text (digits and exponent), never on a rounded float64, in
// time linear in the length of the literal: 2, 2.0 and 20e-1 are whole;
// 2.000...0 with any number of zeros is whole; 1e999999999 is too large; 1e-5
// is fractional; -0 is whole and 0. The literal must be a valid JSON number.
func classifyNumber(num json.Number) (int64, numberClass) {
	s, negative := strings.CutPrefix(string(num), "-")
	mantissa, expText := s, ""
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mantissa, expText = s[:i], s[i+1:]
	}
	intPart, frac, _ := strings.Cut(mantissa, ".")
	digits := len(intPart) + len(frac)
	digitAt := func(i int) byte {
		if i < len(intPart) {
			return intPart[i]
		}
		return frac[i-len(intPart)]
	}
	first, last := -1, -1
	for i := range digits {
		if digitAt(i) != '0' {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return 0, numberWhole
	}
	// value = digits[first..last] * 10^e, with no trailing zero in the digits.
	e := parseExponent(expText) - int64(len(frac)) + int64(digits-1-last)
	significant := int64(last - first + 1) // equivalent mutant when 1 is dropped: the guard below only keeps the digits within uint64, and the v > MaxSafeInteger check decides the same verdict
	switch {
	case e < 0:
		return 0, numberFractional
	case significant+e > 16: // 2^53 - 1 has 16 digits; a 17th makes it larger (equivalent mutant at 17: the v check then gives the same verdict)
		return 0, numberTooLarge
	}
	var v uint64
	for i := first; i <= last; i++ {
		v = v*10 + uint64(digitAt(i)-'0')
	}
	for ; e > 0; e-- { // equivalent mutant: e is not negative here, so > and != agree
		v *= 10
	}
	if v > MaxSafeInteger {
		return 0, numberTooLarge
	}
	if negative {
		return -int64(v), numberWhole
	}
	return int64(v), numberWhole
}

// parseExponent reads the optional signed exponent of a literal, clamped to
// plus or minus maxExponent.
func parseExponent(text string) int64 {
	text, negative := strings.CutPrefix(text, "-")
	text = strings.TrimPrefix(text, "+")
	var v int64
	for i := 0; i < len(text) && v < maxExponent; i++ { // equivalent mutants: i only grows from 0 so < and != agree on len(text), and one more digit past maxExponent is clamped
		v = v*10 + int64(text[i]-'0')
	}
	v = min(v, maxExponent)
	if negative {
		return -v
	}
	return v
}
