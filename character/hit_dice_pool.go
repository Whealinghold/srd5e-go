package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// HitDiceCount is a number of Hit Point Dice of one type.
type HitDiceCount struct {
	Die   content.HitPointDie
	Count int
}

// HitDicePool is the Hit Point Dice a character has, counted by die type (SRD
// 5.2.1, "Multiclassing", "Hit Points and Hit Point Dice", p. 25): the dice
// held, not those spent. Entries are ordered by ascending faces and hold at
// least 1 die each, so equal pools are Equal. It is immutable; it holds a
// slice, so compare two with Equal, not ==. The zero HitDicePool is the empty
// pool. Safe for concurrent use.
type HitDicePool struct{ entries []HitDiceCount }

// NewHitDicePool returns the pool holding counts. An entry with a count of 0 is
// dropped, so equal pools compare equal; the total need not match any level.
//
// It returns an error matching srderr.ErrInvalidInput for a negative count or
// an invalid die. It never panics.
func NewHitDicePool(counts map[content.HitPointDie]int) (HitDicePool, error) {
	for die, n := range counts {
		if !die.Valid() {
			return HitDicePool{}, invalidInput("invalid Hit Point Die")
		}
		if n < 0 {
			return HitDicePool{}, invalidInput("negative Hit Dice count")
		}
	}
	return poolOf(counts), nil
}

// poolOf builds the ordered pool of counts that are already valid.
func poolOf(counts map[content.HitPointDie]int) HitDicePool {
	var entries []HitDiceCount
	for die, n := range counts {
		if n > 0 {
			entries = append(entries, HitDiceCount{Die: die, Count: n})
		}
	}
	slices.SortFunc(entries, func(a, b HitDiceCount) int { return a.Die.Faces() - b.Die.Faces() })
	return HitDicePool{entries: entries}
}

// HitDicePoolFromLevels builds the pool of levels: one die per class level, the
// same die type pooled into one count and different dice kept apart (SRD 5.2.1,
// "Multiclassing", p. 25). dice gives the Hit Point Die of each class; an entry
// for a class not in levels is ignored. The total count equals the total level.
//
// It returns an error matching srderr.ErrInvalidInput when a class in levels
// has no valid die in dice. It never panics.
func HitDicePoolFromLevels(levels ClassLevels, dice map[content.ClassID]content.HitPointDie) (HitDicePool, error) {
	counts := make(map[content.HitPointDie]int)
	for _, e := range levels.entries {
		die, ok := dice[e.Class]
		if !ok || !die.Valid() {
			return HitDicePool{}, invalidInput("class without a Hit Point Die")
		}
		counts[die] += e.Level
	}
	return NewHitDicePool(counts)
}

// MustHitDicePoolFromLevels is HitDicePoolFromLevels for known-valid input. It
// panics with the error HitDicePoolFromLevels returns, and only then.
func MustHitDicePoolFromLevels(levels ClassLevels, dice map[content.ClassID]content.HitPointDie) HitDicePool {
	p, err := HitDicePoolFromLevels(levels, dice)
	if err != nil {
		panic(err)
	}
	return p
}

// Entries returns a new slice of the counts, ordered by ascending faces.
func (p HitDicePool) Entries() []HitDiceCount { return slices.Clone(p.entries) }

// Count returns how many dice of type die the pool holds.
func (p HitDicePool) Count(die content.HitPointDie) int {
	for _, e := range p.entries {
		if e.Die == die {
			return e.Count
		}
	}
	return 0
}

// Total returns the number of dice held.
func (p HitDicePool) Total() int {
	n := 0
	for _, e := range p.entries {
		n += e.Count
	}
	return n
}

// Equal reports whether p and o hold the same counts.
func (p HitDicePool) Equal(o HitDicePool) bool { return slices.Equal(p.entries, o.entries) }

// Plus returns a new pool with one more die of type die (a count of 1 when
// absent), entries in order; the receiver is unchanged. An invalid die leaves
// the pool as it is.
func (p HitDicePool) Plus(die content.HitPointDie) HitDicePool {
	if !die.Valid() {
		return HitDicePool{entries: slices.Clone(p.entries)}
	}
	counts := make(map[content.HitPointDie]int, len(p.entries)+1)
	for _, e := range p.entries {
		counts[e.Die] = e.Count
	}
	counts[die]++
	return poolOf(counts)
}
