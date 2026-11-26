package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// maxMaximumIncreases is the cap on the entries of one HitPointMaximumIncreases
// ledger: the active effects per creature (plan P10).
const maxMaximumIncreases = 64

// HitPointMaximumIncrease is one increase to a creature's Hit Point maximum from a
// spell (SRD 5.2.1, "Aid", p. 107; "Heroes' Feast", p. 140). It is not Temporary Hit
// Points. Build it with NewHitPointMaximumIncrease, AidIncrease or HeroesFeastIncrease.
// The zero value is invalid. Immutable, comparable and safe for concurrent use.
type HitPointMaximumIncrease struct {
	id     OngoingEffectID
	spell  content.SpellID
	amount int
}

// NewHitPointMaximumIncrease returns the increase the effect effectID, granted by
// spell, gives.
//
// It returns an error matching srderr.ErrInvalidInput for a blank effectID, an
// invalid spell id, and (as a *RestorationRangeError) an amount below 1. It never panics.
func NewHitPointMaximumIncrease(effectID OngoingEffectID, spell content.SpellID, amount int) (HitPointMaximumIncrease, error) {
	switch {
	case strings.TrimSpace(string(effectID)) == "":
		return HitPointMaximumIncrease{}, invalidInput("a Hit Point maximum increase needs an effect id")
	case !spell.Valid():
		return HitPointMaximumIncrease{}, invalidInput("a Hit Point maximum increase needs a valid spell id")
	case amount < 1:
		return HitPointMaximumIncrease{}, rangeError("increase_amount", int64(amount))
	}
	return HitPointMaximumIncrease{effectID, spell, amount}, nil
}

// ID returns the id of the ongoing effect that holds the increase.
func (i HitPointMaximumIncrease) ID() OngoingEffectID { return i.id }

// Spell returns the spell that grants the increase.
func (i HitPointMaximumIncrease) Spell() content.SpellID { return i.spell }

// Amount returns the Hit Points the increase adds, at least 1 when valid.
func (i HitPointMaximumIncrease) Amount() int { return i.amount }

// IsZero reports whether i is the zero, invalid value.
func (i HitPointMaximumIncrease) IsZero() bool { return i == HitPointMaximumIncrease{} }

// AidIncrease returns the increase of Aid cast with a spell slot of level castLevel:
// 5 for each slot level above 1 (5 at level 2, 40 at level 9).
//
// It returns a *RestorationRangeError (matching srderr.ErrInvalidInput) for a
// castLevel below 2 or above 9, and an error for a blank effectID. It never panics.
func AidIncrease(effectID OngoingEffectID, castLevel int) (HitPointMaximumIncrease, error) {
	if castLevel < 2 || castLevel > 9 {
		return HitPointMaximumIncrease{}, rangeError("cast_level", int64(castLevel))
	}
	return NewHitPointMaximumIncrease(effectID, content.SpellID("aid"), 5*(castLevel-1))
}

// HeroesFeastIncrease returns the increase of Heroes' Feast from the 2d10 the host
// rolled: the total of the roll. The package rolls nothing.
//
// It returns an error matching srderr.ErrInvalidInput for any roll that is not
// exactly 2d10 with modifier 0 (the zero dice.Result included), and for a blank
// effectID. It never panics.
func HeroesFeastIncrease(effectID OngoingEffectID, roll dice.Result) (HitPointMaximumIncrease, error) {
	e := roll.Expression()
	if e.Count() != 2 || e.Die() != dice.D10() || e.Modifier() != 0 {
		return HitPointMaximumIncrease{}, invalidInput("Heroes' Feast increases by exactly 2d10")
	}
	return NewHitPointMaximumIncrease(effectID, content.SpellID("heroes_feast"), roll.Total())
}

// HitPointMaximumIncreases is the host-held ledger of the increases on one
// creature, with distinct effect ids and at most 64 entries. Build it with
// NewHitPointMaximumIncreases. The zero value is the empty ledger. Immutable and
// safe for concurrent use; compare with Equal.
type HitPointMaximumIncreases struct{ entries []HitPointMaximumIncrease }

// NewHitPointMaximumIncreases returns the ledger holding entries, copied.
//
// It returns an error matching srderr.ErrInvalidInput for a zero entry, two entries
// with one effect id, and more than 64 entries (a *RestorationRangeError). It never panics.
func NewHitPointMaximumIncreases(entries []HitPointMaximumIncrease) (HitPointMaximumIncreases, error) {
	if len(entries) > maxMaximumIncreases {
		return HitPointMaximumIncreases{}, rangeError("increase_entries", int64(len(entries)))
	}
	for i, e := range entries {
		if e.IsZero() || e.amount < 1 {
			return HitPointMaximumIncreases{}, invalidInput("the ledger holds an invalid increase")
		}
		if slices.ContainsFunc(entries[:i], func(o HitPointMaximumIncrease) bool { return o.id == e.id }) {
			return HitPointMaximumIncreases{}, invalidInput("two increases share an effect id")
		}
	}
	return HitPointMaximumIncreases{slices.Clone(entries)}, nil
}

// Entries returns a copy of the entries in the order they were added.
func (l HitPointMaximumIncreases) Entries() []HitPointMaximumIncrease { return slices.Clone(l.entries) }

// Len returns the number of entries.
func (l HitPointMaximumIncreases) Len() int { return len(l.entries) }

// Equal reports whether l and o hold the same entries in the same order.
func (l HitPointMaximumIncreases) Equal(o HitPointMaximumIncreases) bool {
	return slices.Equal(l.entries, o.entries)
}

// Total returns the effective total (SRD 5.2.1, "Combining Spell Effects", p. 106):
// the largest amount of each spell, summed over the distinct spells. A sum that
// would pass the largest int stops there.
func (l HitPointMaximumIncreases) Total() int {
	largest := map[content.SpellID]int{}
	for _, e := range l.entries {
		largest[e.spell] = max(largest[e.spell], e.amount)
	}
	total := 0
	for _, a := range largest {
		total = satAdd(total, a)
	}
	return total
}

func (l HitPointMaximumIncreases) find(id OngoingEffectID) bool {
	return slices.ContainsFunc(l.entries, func(e HitPointMaximumIncrease) bool { return e.id == id })
}

func (l HitPointMaximumIncreases) with(i HitPointMaximumIncrease) HitPointMaximumIncreases {
	return HitPointMaximumIncreases{append(slices.Clone(l.entries), i)}
}

func (l HitPointMaximumIncreases) without(id OngoingEffectID) HitPointMaximumIncreases {
	return HitPointMaximumIncreases{slices.DeleteFunc(slices.Clone(l.entries), func(e HitPointMaximumIncrease) bool { return e.id == id })}
}
