package modifier

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// The d20 faces accepted by RollMode.Select.
const (
	lowestFace  = 1
	highestFace = 20
)

// RollMode says how many d20s a d20 Test rolls and which one it uses (SRD
// 5.2.1, "Advantage/Disadvantage", p. 8). The zero RollMode is Normal.
type RollMode int

// The roll modes. They are named With... because Advantage and Disadvantage
// are the circumstances that Resolve reads.
const (
	Normal RollMode = iota
	WithAdvantage
	WithDisadvantage
)

func modeNames() []string { return []string{"normal", "advantage", "disadvantage"} }

// String returns the lowercase name of m, or "unknown" when m is undefined.
func (m RollMode) String() string {
	if !m.valid() {
		return "unknown"
	}
	return modeNames()[m]
}

func (m RollMode) valid() bool { return m >= Normal && m <= WithDisadvantage }

// ParseRollMode parses "normal", "advantage" or "disadvantage" (any case). It
// returns an *InvalidError with ReasonUnknownRollMode for any other text.
func ParseRollMode(s string) (RollMode, error) {
	for i, n := range modeNames() {
		if strings.EqualFold(s, n) {
			return RollMode(i), nil
		}
	}
	return 0, invalid(ReasonUnknownRollMode, "no roll mode named "+srderr.Quote(s))
}

// Resolve combines circumstances into one RollMode. Advantage and Disadvantage
// do not stack; whatever the counts, at least one of each cancels to Normal
// ("They Don't Stack", p. 8). The order of cs is irrelevant, cs is not altered,
// and nil entries are ignored. Only the values Advantage{} and Disadvantage{}
// count; pointers to them (*Advantage, *Disadvantage) are ignored too.
func Resolve(cs ...RollCircumstance) RollMode {
	var adv, dis bool
	for _, c := range cs {
		switch c.(type) {
		case Advantage:
			adv = true
		case Disadvantage:
			dis = true
		}
	}
	switch {
	case adv && !dis:
		return WithAdvantage
	case dis && !adv:
		return WithDisadvantage
	}
	return Normal
}

// DiceCount returns how many d20s m rolls: 1 for Normal, 2 otherwise. An
// undefined m returns 1.
func (m RollMode) DiceCount() int {
	if m == WithAdvantage || m == WithDisadvantage {
		return 2
	}
	return 1
}

// Select returns the d20 number used from rolled: the only value for Normal,
// the higher for advantage, the lower for disadvantage. rolled is not altered.
//
// It returns an errors.Join of *InvalidError values, all reported at once:
// ReasonRollCountMismatch when len(rolled) differs from DiceCount, and
// ReasonFaceOutOfRange for each value outside 1 to 20. An undefined m is
// rejected with ReasonUnknownRollMode.
func (m RollMode) Select(rolled []int) (int, error) {
	if !m.valid() {
		return 0, invalid(ReasonUnknownRollMode, "undefined roll mode "+strconv.Itoa(int(m)))
	}
	var errs []error
	if n := m.DiceCount(); len(rolled) != n {
		errs = append(errs, invalid(ReasonRollCountMismatch,
			m.String()+" needs "+strconv.Itoa(n)+" d20, got "+strconv.Itoa(len(rolled))))
	}
	for _, r := range rolled {
		if r < lowestFace || r > highestFace {
			errs = append(errs, invalid(ReasonFaceOutOfRange, strconv.Itoa(r)+" is not a d20 face"))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return 0, err
	}
	switch m {
	case WithAdvantage:
		return max(rolled[0], rolled[1]), nil
	case WithDisadvantage:
		return min(rolled[0], rolled[1]), nil
	case Normal:
	}
	return rolled[0], nil
}
