package ability

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// Ability is one of the six abilities, in the order the SRD lists them. The
// zero Ability is invalid.
type Ability int

// The six abilities (SRD 5.2.1, "The Six Abilities", p. 5).
const (
	Strength Ability = iota + 1
	Dexterity
	Constitution
	Intelligence
	Wisdom
	Charisma
)

func names() []string {
	return []string{
		Strength:     "strength",
		Dexterity:    "dexterity",
		Constitution: "constitution",
		Intelligence: "intelligence",
		Wisdom:       "wisdom",
		Charisma:     "charisma",
	}
}

// All returns the six abilities in SRD order. Each call returns a new slice.
func All() []Ability {
	return []Ability{Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma}
}

// Valid reports whether a is one of the six abilities.
func (a Ability) Valid() bool { return a >= Strength && a <= Charisma }

// String returns the lowercase name of a, or "unknown" when a is invalid.
func (a Ability) String() string {
	if !a.Valid() {
		return "unknown"
	}
	return names()[a]
}

// ParseAbility parses a lowercase or mixed-case ability name such as
// "strength". It returns an *InvalidError with ReasonUnknownAbility for any
// other text.
func ParseAbility(s string) (Ability, error) {
	for _, a := range All() {
		if strings.EqualFold(s, names()[a]) {
			return a, nil
		}
	}
	return 0, invalid(ReasonUnknownAbility, "no ability named "+srderr.Quote(s))
}
