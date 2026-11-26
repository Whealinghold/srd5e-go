package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// maxOrdealPenalty is the largest penalty of an ordeal (SRD 5.2.1, "Raise Dead", p. 157).
const maxOrdealPenalty = 4

// ReviveOrdeal is the penalty to D20 Tests a creature carries after Raise Dead or
// Resurrection, from 0 to 4, shrinking by 1 for each Long Rest finished (SRD 5.2.1,
// "Raise Dead", p. 157). Build it with NewReviveOrdeal or FreshOrdeal. The zero
// value is no ordeal. Immutable, comparable with == and safe for concurrent use.
type ReviveOrdeal struct{ penalty int }

// NewReviveOrdeal returns the ordeal of the given penalty, 0 to 4.
//
// It returns a *RestorationRangeError (matching srderr.ErrInvalidInput) outside
// that range, and the zero ReviveOrdeal with it. It never panics.
func NewReviveOrdeal(penalty int) (ReviveOrdeal, error) {
	if penalty < 0 || penalty > maxOrdealPenalty {
		return ReviveOrdeal{}, rangeError("ordeal_penalty", int64(penalty))
	}
	return ReviveOrdeal{penalty}, nil
}

// MustReviveOrdeal is NewReviveOrdeal for known-good values. It panics with the
// error NewReviveOrdeal returns, and only then.
func MustReviveOrdeal(penalty int) ReviveOrdeal {
	o, err := NewReviveOrdeal(penalty)
	if err != nil {
		panic(err)
	}
	return o
}

// FreshOrdeal returns the ordeal a successful Raise Dead or Resurrection gives: 4.
func FreshOrdeal() ReviveOrdeal { return ReviveOrdeal{maxOrdealPenalty} }

// Penalty returns the penalty as a number from 0 to 4.
func (o ReviveOrdeal) Penalty() int { return o.penalty }

// Modifier returns the modifier to every D20 Test: -Penalty, or 0 for none. The
// host adds it to its RollModifiers and to a Death Saving Throw total, together
// with the Exhaustion reduction (the two are summed).
func (o ReviveOrdeal) Modifier() modifier.Modifier { return modifier.New(-o.penalty) }

// AfterRest returns the ordeal after a rest: one less for character.LongRest, never
// below 0, and an equal value for every other kind. The host passes
// RestApplied.BenefitsOf(), so a Long Rest interrupted into Short Rest benefits
// does not reduce it.
func (o ReviveOrdeal) AfterRest(kind character.RestKind) ReviveOrdeal {
	if kind == character.LongRest && o.penalty > 0 {
		return ReviveOrdeal{o.penalty - 1}
	}
	return o
}
