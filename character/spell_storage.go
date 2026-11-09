package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// RingOfSpellStoringCapacity is the levels of spells the ring holds (SRD 5.2.1,
// "Ring of Spell Storing", p. 239).
const RingOfSpellStoringCapacity = 5

func ringSlotLevel(l content.SpellLevel) bool { return l >= 1 && l <= RingOfSpellStoringCapacity }

// StoredSpell is a spell in the ring with the numbers of the creature that cast
// it into the ring, as supplied and never recomputed. Comparable; the zero value
// is invalid.
type StoredSpell struct {
	spell     content.SpellID
	slotLevel content.SpellLevel
	saveDC    SpellSaveDC
	attack    SpellAttackBonus
	ability   ability.Ability
}

// NewStoredSpell returns the stored spell. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell or ability, or a slot level outside
// 1 to 5.
func NewStoredSpell(spell content.SpellID, slotLevel content.SpellLevel, saveDC SpellSaveDC, attack SpellAttackBonus, a ability.Ability) (StoredSpell, error) {
	if !spell.Valid() || !ringSlotLevel(slotLevel) || !a.Valid() {
		return StoredSpell{}, invalidInput("stored spell: invalid spell or ability, or slot level outside 1 to 5")
	}
	return StoredSpell{spell, slotLevel, saveDC, attack, a}, nil
}

// SpellID returns the spell.
func (s StoredSpell) SpellID() content.SpellID { return s.spell }

// SlotLevel returns the level of the slot the caster used; it decides the space used.
func (s StoredSpell) SlotLevel() content.SpellLevel { return s.slotLevel }

// SaveDC returns the original caster's spell save DC.
func (s StoredSpell) SaveDC() SpellSaveDC { return s.saveDC }

// AttackBonus returns the original caster's spell attack bonus.
func (s StoredSpell) AttackBonus() SpellAttackBonus { return s.attack }

// Ability returns the original caster's spellcasting ability.
func (s StoredSpell) Ability() ability.Ability { return s.ability }

// SpellCastIntoRing is a spell cast into a Ring of Spell Storing, before the ring
// decides whether it fits. Construct it with a keyed literal.
type SpellCastIntoRing struct {
	Spell       content.SpellID
	SlotLevel   content.SpellLevel
	SaveDC      SpellSaveDC
	AttackBonus SpellAttackBonus
	Ability     ability.Ability
}

// SpellStorage is the spells in a Ring of Spell Storing, in order. Immutable:
// every operation returns a new value and the exposed list is a copy. The zero
// value is an empty ring.
type SpellStorage struct{ spells []StoredSpell }

// NewSpellStorage returns the ring holding spells. It returns an error matching
// srderr.ErrInvalidInput for a zero stored spell or levels over 5 in total.
func NewSpellStorage(spells []StoredSpell) (SpellStorage, error) {
	used := 0
	for _, s := range spells {
		if !ringSlotLevel(s.slotLevel) {
			return SpellStorage{}, invalidInput("spell storage: zero stored spell")
		}
		used += int(s.slotLevel)
	}
	if used > RingOfSpellStoringCapacity {
		return SpellStorage{}, invalidInput("spell storage: more than 5 levels")
	}
	return SpellStorage{spells: slices.Clone(spells)}, nil
}

// Spells returns a new slice of the stored spells in order.
func (s SpellStorage) Spells() []StoredSpell { return slices.Clone(s.spells) }

// UsedLevels returns the levels the stored spells take.
func (s SpellStorage) UsedLevels() int {
	n := 0
	for _, sp := range s.spells {
		n += int(sp.slotLevel)
	}
	return n
}

// Equal reports whether s and o hold the same spells in the same order.
func (s SpellStorage) Equal(o SpellStorage) bool { return slices.Equal(s.spells, o.spells) }

// SpellStorageRejection says why initial contents were refused. The zero value is not a reason.
type SpellStorageRejection int

// SpellStorageLevelsDoNotMatch: the spells' slot levels do not sum to the rolled levels.
const SpellStorageLevelsDoNotMatch SpellStorageRejection = 1

// SpellStorageInitialResult is the closed set of outcomes of SpellStorage.Initial.
// Variants: SpellStorageInitialised, SpellStorageRejected.
type SpellStorageInitialResult interface{ isSpellStorageInitialResult() }

// SpellStorageInitialised: the ring's starting contents.
type SpellStorageInitialised struct{ storage SpellStorage }

// Storage returns the ring.
func (r SpellStorageInitialised) Storage() SpellStorage { return r.storage }

// SpellStorageRejected: the contents do not match.
type SpellStorageRejected struct{ reason SpellStorageRejection }

// Reason returns SpellStorageLevelsDoNotMatch.
func (r SpellStorageRejected) Reason() SpellStorageRejection { return r.reason }

func (SpellStorageInitialised) isSpellStorageInitialResult() {}
func (SpellStorageRejected) isSpellStorageInitialResult()    {}

// Initial returns the ring a GM fills with spells whose slot levels sum exactly to
// levels, the 1d6 - 1 of RollSpellStoringRingLevels. A different sum is
// SpellStorageRejected. It returns an error matching srderr.ErrInvalidInput for
// levels outside 0 to 5 or an invalid stored spell. The receiver is ignored.
func (SpellStorage) Initial(levels int, spells []StoredSpell) (SpellStorageInitialResult, error) {
	if levels < 0 || levels > RingOfSpellStoringCapacity {
		return nil, invalidInput("initial spell storage: levels outside 0 to 5")
	}
	st, err := NewSpellStorage(spells)
	if err != nil {
		return nil, err
	}
	if st.UsedLevels() != levels {
		return SpellStorageRejected{reason: SpellStorageLevelsDoNotMatch}, nil
	}
	return SpellStorageInitialised{storage: st}, nil
}

// SpellStorageStoreResult is the closed set of outcomes of SpellStorage.Store.
// Variants: SpellStorageStored, SpellStorageSpellExpended, SpellStorageNotStorable.
type SpellStorageStoreResult interface{ isSpellStorageStoreResult() }

// SpellStorageStored: the ring took the spell.
type SpellStorageStored struct {
	storage SpellStorage
	event   RingSpellStored
}

// Storage returns the new ring.
func (r SpellStorageStored) Storage() SpellStorage { return r.storage }

// Event returns the RingSpellStored event.
func (r SpellStorageStored) Event() RingSpellStored { return r.event }

// SpellStorageSpellExpended: the ring was too full, so the spell is expended
// without effect; the ring is unchanged.
type SpellStorageSpellExpended struct{ spell SpellCastIntoRing }

// Spell returns the spell lost.
func (r SpellStorageSpellExpended) Spell() SpellCastIntoRing { return r.spell }

// SpellStorageNotStorable: only a spell of level 1 through 5 can be cast into the
// ring, so a cantrip or a level 6 or higher slot is not storable.
type SpellStorageNotStorable struct{ slotLevel content.SpellLevel }

// SlotLevel returns the slot level refused.
func (r SpellStorageNotStorable) SlotLevel() content.SpellLevel { return r.slotLevel }

func (SpellStorageStored) isSpellStorageStoreResult()        {}
func (SpellStorageSpellExpended) isSpellStorageStoreResult() {}
func (SpellStorageNotStorable) isSpellStorageStoreResult()   {}

// Store puts a spell cast into the ring at the end of the list when its slot level
// fits the space left (the slot level, not the spell's level, decides). Storing
// needs no Attunement. It returns an error matching srderr.ErrInvalidInput for an
// invalid spell or ability, a zero holder or a blank instance.
func (s SpellStorage) Store(holder Identifier, instance MagicItemInstanceID, cast SpellCastIntoRing) (SpellStorageStoreResult, error) {
	if !cast.Spell.Valid() || !cast.Ability.Valid() || holder.IsZero() || !instance.Valid() {
		return nil, invalidInput("store spell: invalid spell or ability, zero holder or blank instance")
	}
	if !ringSlotLevel(cast.SlotLevel) {
		return SpellStorageNotStorable{slotLevel: cast.SlotLevel}, nil
	}
	if s.UsedLevels()+int(cast.SlotLevel) > RingOfSpellStoringCapacity {
		return SpellStorageSpellExpended{spell: cast}, nil
	}
	stored := StoredSpell{cast.Spell, cast.SlotLevel, cast.SaveDC, cast.AttackBonus, cast.Ability}
	return SpellStorageStored{storage: SpellStorage{spells: append(slices.Clone(s.spells), stored)},
		event: RingSpellStored{id: holder, instance: instance, spell: cast.Spell, slotLevel: cast.SlotLevel}}, nil
}

// SpellStorageCastResult is the closed set of outcomes of SpellStorage.Cast.
// Variants: StoredSpellCast, SpellStorageNotUsable, SpellStorageNoSuchSpell.
type SpellStorageCastResult interface{ isSpellStorageCastResult() }

// StoredSpellCast: the wearer cast the spell with the original caster's numbers.
type StoredSpellCast struct {
	spell   StoredSpell
	storage SpellStorage
	event   RingStoredSpellCast
}

// Spell returns the spell with the slot level, save DC, attack bonus and ability
// of the creature that stored it.
func (r StoredSpellCast) Spell() StoredSpell { return r.spell }

// Storage returns the ring without the spell.
func (r StoredSpellCast) Storage() SpellStorage { return r.storage }

// Event returns the RingStoredSpellCast event.
func (r StoredSpellCast) Event() RingStoredSpellCast { return r.event }

// SpellStorageNotUsable: only the attuned wearer can cast; the ring is unchanged.
type SpellStorageNotUsable struct{ access MagicalPropertiesAccessLevel }

// Access returns the access level that was refused.
func (r SpellStorageNotUsable) Access() MagicalPropertiesAccessLevel { return r.access }

// SpellStorageNoSuchSpell: no spell is at the position.
type SpellStorageNoSuchSpell struct{ index int }

// Index returns the position asked for.
func (r SpellStorageNoSuchSpell) Index() int { return r.index }

func (StoredSpellCast) isSpellStorageCastResult()         {}
func (SpellStorageNotUsable) isSpellStorageCastResult()   {}
func (SpellStorageNoSuchSpell) isSpellStorageCastResult() {}

// Cast removes the spell at index when access is AccessFullMagical, freeing its
// levels. The check of access comes first, then the position. It returns an error
// matching srderr.ErrInvalidInput for a zero holder or a blank instance.
func (s SpellStorage) Cast(holder Identifier, instance MagicItemInstanceID, index int, access MagicalPropertiesAccessLevel) (SpellStorageCastResult, error) {
	if holder.IsZero() || !instance.Valid() {
		return nil, invalidInput("cast stored spell: zero holder or blank instance")
	}
	if access != AccessFullMagical {
		return SpellStorageNotUsable{access: access}, nil
	}
	if index < 0 || index >= len(s.spells) {
		return SpellStorageNoSuchSpell{index: index}, nil
	}
	sp := s.spells[index]
	return StoredSpellCast{spell: sp, storage: SpellStorage{spells: slices.Delete(slices.Clone(s.spells), index, index+1)},
		event: RingStoredSpellCast{id: holder, instance: instance, spell: sp.spell, slotLevel: sp.slotLevel}}, nil
}
