package character

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// ErrSlotLevelRange is matched (errors.Is) by the error NewSlotLevel returns for
// a level outside 1 to 9. It also matches srderr.ErrInvalidInput.
var ErrSlotLevelRange = fmt.Errorf("character: slot level out of range 1 to 9: %w", srderr.ErrInvalidInput)

// SlotLevel is the level of a spell slot, 1 to 9; level 0 is a cantrip and never
// uses a slot (SRD 5.2.1, "Spell Level", p. 104). The zero SlotLevel is
// invalid. Comparable and safe for concurrent use.
type SlotLevel struct{ n int }

// NewSlotLevel returns the slot level n. It returns an error matching
// ErrSlotLevelRange (and srderr.ErrInvalidInput) for n outside 1 to 9. It never
// panics.
func NewSlotLevel(n int) (SlotLevel, error) {
	if n < 1 || n > 9 {
		return SlotLevel{}, ErrSlotLevelRange
	}
	return SlotLevel{n: n}, nil
}

// MustSlotLevel is NewSlotLevel for constants known to be valid. It panics with
// the error NewSlotLevel returns, and only then.
func MustSlotLevel(n int) SlotLevel {
	l, err := NewSlotLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 9, or 0 for the zero SlotLevel.
func (l SlotLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero SlotLevel.
func (l SlotLevel) IsZero() bool { return l.n == 0 }

// SpellSlotPool names one of the two pools of slots a character has: the
// Spellcasting pool and the Pact Magic pool (SRD 5.2.1, "Multiclassing", "Pact
// Magic", p. 26). The zero value is invalid. Comparable and safe for concurrent
// use.
type SpellSlotPool int

// The slot pools.
const (
	// PoolSpellcasting is the pool of the Spellcasting feature.
	PoolSpellcasting SpellSlotPool = iota + 1
	// PoolPactMagic is the Warlock's Pact Magic pool.
	PoolPactMagic
)

// Valid reports whether p is one of the two pools.
func (p SpellSlotPool) Valid() bool { return p == PoolSpellcasting || p == PoolPactMagic }

// String returns "spellcasting", "pact_magic" or "unknown".
func (p SpellSlotPool) String() string {
	switch p {
	case PoolSpellcasting:
		return "spellcasting"
	case PoolPactMagic:
		return "pact_magic"
	}
	return "unknown"
}

// SpellSlots is the maximum and remaining Spell Slots of the Spellcasting pool
// for each spell level 1 to 9 (SRD 5.2.1, "Spell Slots", p. 104). It is
// comparable. The zero SpellSlots has no slots and is valid.
type SpellSlots struct{ max, rem [9]int }

// NewSpellSlots returns the slots with the given maximums and remaining counts,
// indexed by spell level minus 1. It returns an error matching
// srderr.ErrInvalidInput when a maximum or remaining count is negative or a
// remaining count exceeds its maximum. It never panics.
func NewSpellSlots(maximum, remaining [9]int) (SpellSlots, error) {
	for i := range maximum {
		if maximum[i] < 0 || remaining[i] < 0 || remaining[i] > maximum[i] {
			return SpellSlots{}, invalidInput("spell slots: remaining outside 0 to maximum")
		}
	}
	return SpellSlots{max: maximum, rem: remaining}, nil
}

// MustSpellSlots is NewSpellSlots for known-valid input. It panics with the
// error NewSpellSlots returns, and only then.
func MustSpellSlots(maximum, remaining [9]int) SpellSlots {
	s, err := NewSpellSlots(maximum, remaining)
	if err != nil {
		panic(err)
	}
	return s
}

// ForLevels returns the full Spellcasting-pool slots (remaining equal to
// maximum) of a character with the given class levels (SRD 5.2.1,
// "Multiclassing", "Spellcasting", pp. 25-26): a single Spellcasting class uses
// its own table (C or D); two or more use the Multiclass Spellcaster table at the
// total spellcaster level (all Bard, Cleric, Druid, Sorcerer and Wizard levels
// plus half of the combined Paladin and Ranger levels, rounded up). Warlock
// levels and classes without Spellcasting add nothing. The receiver is ignored.
func (SpellSlots) ForLevels(l ClassLevels) SpellSlots {
	var max [9]int
	casters, full, half, last := 0, 0, 0, SpellcastingClass(0)
	for _, e := range l.Entries() {
		c, ok := SpellcastingClassFor(e.Class)
		if !ok || c.Feature() != FeatureSpellcasting {
			continue
		}
		casters++
		last = c
		if c.isHalfCaster() {
			half += e.Level
		} else {
			full += e.Level
		}
	}
	switch {
	case casters == 1 && last.isHalfCaster():
		max = slotsAt(halfCasterRows, half)
	case casters == 1:
		max = slotsAt(fullCasterRows, full)
	case casters > 1:
		max = slotsAt(fullCasterRows, full+(half+1)/2)
	}
	return SpellSlots{max: max, rem: max}
}

// Maximum returns the maximum slots of level l; 0 for the zero SlotLevel.
func (s SpellSlots) Maximum(l SlotLevel) int {
	if l.IsZero() {
		return 0
	}
	return s.max[l.n-1]
}

// Remaining returns the unexpended slots of level l; 0 for the zero SlotLevel.
func (s SpellSlots) Remaining(l SlotLevel) int {
	if l.IsZero() {
		return 0
	}
	return s.rem[l.n-1]
}

// Expended returns Maximum minus Remaining for level l.
func (s SpellSlots) Expended(l SlotLevel) int { return s.Maximum(l) - s.Remaining(l) }

// PactMagicSlots is the Warlock's Pact Magic pool: a maximum count of slots, all
// of one Slot Level, and how many remain (SRD 5.2.1, "Warlock", Pact Magic, p.
// 71). It is comparable. The zero PactMagicSlots is the empty pool (no Warlock
// level) and is valid.
type PactMagicSlots struct{ max, level, rem int }

// pactMagicFor returns the full Pact Magic pool of a Warlock level (Table E).
func pactMagicFor(warlockLevel int) PactMagicSlots {
	n, lvl := pactMagicAt(warlockLevel)
	return PactMagicSlots{max: n, level: lvl, rem: n}
}

// HasPactMagic reports whether the pool has any slot (the character has a
// Warlock level).
func (p PactMagicSlots) HasPactMagic() bool { return p.max > 0 }

// Count returns the maximum number of slots.
func (p PactMagicSlots) Count() int { return p.max }

// Level returns the Slot Level of every slot; the zero SlotLevel for an empty pool.
func (p PactMagicSlots) Level() SlotLevel { return SlotLevel{n: p.level} }

// Remaining returns the unexpended slots.
func (p PactMagicSlots) Remaining() int { return p.rem }

// SpellSlotPools is both pools of a character: the Spellcasting pool and the
// Pact Magic pool, which stays separate (SRD 5.2.1, "Multiclassing", "Pact
// Magic", p. 26). It is comparable. The zero value has no slots.
type SpellSlotPools struct {
	spellcasting SpellSlots
	pactMagic    PactMagicSlots
}

// SlotsFor returns the full slots (remaining equal to maximum) of both pools for
// the given class levels. It spends nothing and reads no state. The Pact Magic
// pool is the zero value when there is no Warlock level.
func SlotsFor(l ClassLevels) SpellSlotPools {
	return SpellSlotPools{spellcasting: SpellSlots{}.ForLevels(l), pactMagic: pactMagicFor(l.LevelIn(CasterWarlock.ClassID()))}
}

// Spellcasting returns the Spellcasting pool.
func (p SpellSlotPools) Spellcasting() SpellSlots { return p.spellcasting }

// PactMagic returns the Pact Magic pool; the zero PactMagicSlots when there is
// no Warlock level.
func (p SpellSlotPools) PactMagic() PactMagicSlots { return p.pactMagic }

// SlotPoolView is a read-only view of one pool's slots by level. It is
// comparable. The zero value reads zero everywhere.
type SlotPoolView struct{ max, rem [9]int }

// Maximum returns the maximum slots of level l; 0 for the zero SlotLevel.
func (v SlotPoolView) Maximum(l SlotLevel) int {
	if l.IsZero() {
		return 0
	}
	return v.max[l.n-1]
}

// Remaining returns the unexpended slots of level l; 0 for the zero SlotLevel.
func (v SlotPoolView) Remaining(l SlotLevel) int {
	if l.IsZero() {
		return 0
	}
	return v.rem[l.n-1]
}

// Expended returns Maximum minus Remaining for level l.
func (v SlotPoolView) Expended(l SlotLevel) int { return v.Maximum(l) - v.Remaining(l) }
