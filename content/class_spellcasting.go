package content

import "github.com/Whealinghold/srd5e-go/ability"

// ClassSpellLevels is the number of spell levels a slot caster has columns
// for: 1 to 9 (SRD 5.2.1, the Features tables of the casters).
const ClassSpellLevels = 9

// ClassSpellcastingTrait is the spellcasting a class has (SRD 5.2.1, the
// "Spellcasting" and "Pact Magic" features): a closed set with two variants,
// [SlotSpellcastingTrait] and [PactSpellcastingTrait]. It is named "Trait"
// because ClassSpellcasting is the spellcasting state of a character.
// Consumers switch over it exhaustively. The variants are immutable,
// comparable and safe for concurrent use.
type ClassSpellcastingTrait interface {
	// Ability returns the spellcasting ability.
	Ability() ability.Ability
	isClassSpellcastingTrait()
}

// SlotSpellcastingTrait is spellcasting with spell slots. The zero value is
// invalid; build it with [NewSlotSpellcastingTrait].
type SlotSpellcastingTrait struct {
	ability  ability.Ability
	cantrips bool
}

// NewSlotSpellcastingTrait returns slot spellcasting with the given ability;
// cantrips says whether the class knows cantrips.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid ability. It never panics.
func NewSlotSpellcastingTrait(a ability.Ability, cantrips bool) (SlotSpellcastingTrait, error) {
	if !a.Valid() {
		return SlotSpellcastingTrait{}, blockError("SlotSpellcastingTrait", BlockInvalidMember)
	}
	return SlotSpellcastingTrait{ability: a, cantrips: cantrips}, nil
}

// Ability returns the spellcasting ability.
func (t SlotSpellcastingTrait) Ability() ability.Ability { return t.ability }

// Cantrips reports whether the class knows cantrips.
func (t SlotSpellcastingTrait) Cantrips() bool { return t.cantrips }

func (SlotSpellcastingTrait) isClassSpellcastingTrait() {}

// PactSpellcastingTrait is Pact Magic. The zero value is invalid; build it
// with [NewPactSpellcastingTrait].
type PactSpellcastingTrait struct{ ability ability.Ability }

// NewPactSpellcastingTrait returns Pact Magic with the given ability.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid ability. It never panics.
func NewPactSpellcastingTrait(a ability.Ability) (PactSpellcastingTrait, error) {
	if !a.Valid() {
		return PactSpellcastingTrait{}, blockError("PactSpellcastingTrait", BlockInvalidMember)
	}
	return PactSpellcastingTrait{ability: a}, nil
}

// Ability returns the spellcasting ability.
func (t PactSpellcastingTrait) Ability() ability.Ability { return t.ability }

func (PactSpellcastingTrait) isClassSpellcastingTrait() {}

func validSpellcastingTrait(t ClassSpellcastingTrait) bool {
	switch v := t.(type) {
	case SlotSpellcastingTrait:
		_, err := NewSlotSpellcastingTrait(v.ability, v.cantrips)
		return err == nil
	case PactSpellcastingTrait:
		_, err := NewPactSpellcastingTrait(v.ability)
		return err == nil
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}

// SpellcastingRow is the spellcasting part of one row of the Features table:
// a closed set with two variants, [SlotSpellcastingRow] and
// [PactSpellcastingRow]. Consumers switch over it exhaustively. The variants
// are immutable, comparable and safe for concurrent use.
type SpellcastingRow interface {
	// PreparedSpells returns the Prepared Spells column of the row.
	PreparedSpells() int
	isSpellcastingRow()
}

// SlotSpellcastingRow is the spellcasting part of a row of a slot caster: the
// Cantrips column when the class has one, the Prepared Spells column and nine
// slot counts. A printed "—" is stored as 0. The zero value is a valid row of
// no cantrips, no prepared spells and no slots.
type SlotSpellcastingRow struct {
	cantrips    int
	hasCantrips bool
	prepared    int
	slots       [ClassSpellLevels]int
}

func slotRow(cantrips int, hasCantrips bool, prepared int, slots []int) (SlotSpellcastingRow, error) {
	const block = "SlotSpellcastingRow"
	if len(slots) != ClassSpellLevels {
		return SlotSpellcastingRow{}, blockError(block, BlockOutOfRange)
	}
	if cantrips < 0 || cantrips > ClassMaxCount || prepared < 0 || prepared > ClassMaxCount {
		return SlotSpellcastingRow{}, blockError(block, BlockOutOfRange)
	}
	r := SlotSpellcastingRow{cantrips: cantrips, hasCantrips: hasCantrips, prepared: prepared}
	for i, n := range slots {
		if n < 0 || n > ClassMaxCount {
			return SlotSpellcastingRow{}, blockError(block, BlockOutOfRange)
		}
		r.slots[i] = n
	}
	return r, nil
}

// NewSlotSpellcastingRow returns a row with no Cantrips column, preparedSpells
// and the slots of spell levels 1 to 9.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a count outside 0 to [ClassMaxCount] or a slot list that
// is not exactly nine long. It never panics.
func NewSlotSpellcastingRow(preparedSpells int, spellSlots []int) (SlotSpellcastingRow, error) {
	return slotRow(0, false, preparedSpells, spellSlots)
}

// NewSlotSpellcastingRowWithCantrips is like [NewSlotSpellcastingRow] for a
// class that has a Cantrips column.
func NewSlotSpellcastingRowWithCantrips(cantrips, preparedSpells int, spellSlots []int) (SlotSpellcastingRow, error) {
	return slotRow(cantrips, true, preparedSpells, spellSlots)
}

// Cantrips returns the Cantrips column and true, or 0 and false when the class
// has none.
func (r SlotSpellcastingRow) Cantrips() (int, bool) { return r.cantrips, r.hasCantrips }

// PreparedSpells returns the Prepared Spells column.
func (r SlotSpellcastingRow) PreparedSpells() int { return r.prepared }

// SpellSlots returns the slots of spell levels 1 to 9, index 0 being level 1.
func (r SlotSpellcastingRow) SpellSlots() [ClassSpellLevels]int { return r.slots }

func (SlotSpellcastingRow) isSpellcastingRow() {}

// PactSpellcastingRow is the spellcasting part of a Warlock row: Cantrips,
// Prepared Spells, the number of Pact Magic slots and their spell level. The
// zero value is invalid; build it with [NewPactSpellcastingRow].
type PactSpellcastingRow struct {
	cantrips, prepared, slots, slotLevel int
}

// NewPactSpellcastingRow returns a Pact Magic row.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for cantrips or preparedSpells outside 0 to
// [ClassMaxCount], spellSlots outside 1 to [ClassMaxCount] or slotLevel
// outside 1 to 9. It never panics.
func NewPactSpellcastingRow(cantrips, preparedSpells, spellSlots, slotLevel int) (PactSpellcastingRow, error) {
	if cantrips < 0 || cantrips > ClassMaxCount || preparedSpells < 0 || preparedSpells > ClassMaxCount ||
		spellSlots < 1 || spellSlots > ClassMaxCount || slotLevel < 1 || slotLevel > ClassSpellLevels {
		return PactSpellcastingRow{}, blockError("PactSpellcastingRow", BlockOutOfRange)
	}
	return PactSpellcastingRow{cantrips: cantrips, prepared: preparedSpells, slots: spellSlots, slotLevel: slotLevel}, nil
}

// Cantrips returns the Cantrips column.
func (r PactSpellcastingRow) Cantrips() int { return r.cantrips }

// PreparedSpells returns the Prepared Spells column.
func (r PactSpellcastingRow) PreparedSpells() int { return r.prepared }

// SpellSlots returns the number of Pact Magic slots.
func (r PactSpellcastingRow) SpellSlots() int { return r.slots }

// SlotLevel returns the spell level of the Pact Magic slots, 1 to 9.
func (r PactSpellcastingRow) SlotLevel() int { return r.slotLevel }

func (PactSpellcastingRow) isSpellcastingRow() {}

func validSpellcastingRow(r SpellcastingRow) bool {
	switch v := r.(type) {
	case SlotSpellcastingRow:
		_, err := slotRow(v.cantrips, v.hasCantrips, v.prepared, v.slots[:])
		return err == nil
	case PactSpellcastingRow:
		_, err := NewPactSpellcastingRow(v.cantrips, v.prepared, v.slots, v.slotLevel)
		return err == nil
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}
