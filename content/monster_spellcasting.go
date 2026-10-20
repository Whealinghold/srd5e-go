package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// SpellFrequency is how often the spells of a [MonsterSpellGroup] can be cast
// (SRD 5.2.1, "Spellcasting", p. 257), a closed family with three variants:
// [SpellFrequencyAtWill], [SpellFrequencyPerDay] ("N/Day:") and
// [SpellFrequencyPerDayEach] ("N/Day Each:"). Consumers switch over it
// exhaustively. Every variant is immutable, comparable and safe for concurrent
// use.
type SpellFrequency interface{ isSpellFrequency() }

// SpellFrequencyAtWill is "At Will:". The zero value is valid.
type SpellFrequencyAtWill struct{}

func (SpellFrequencyAtWill) isSpellFrequency() {}

// SpellFrequencyPerDay is "N/Day:": the group's spells share n casts a day. The
// zero value is invalid; build it with [NewSpellFrequencyPerDay].
type SpellFrequencyPerDay struct{ count int }

// NewSpellFrequencyPerDay returns the frequency of n casts a day.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange when n is below [MonsterMinUses] or above
// [MonsterMaxNumber]. It never panics.
func NewSpellFrequencyPerDay(n int) (SpellFrequencyPerDay, error) {
	if n < MonsterMinUses || n > MonsterMaxNumber {
		return SpellFrequencyPerDay{}, blockError("SpellFrequencyPerDay", BlockOutOfRange)
	}
	return SpellFrequencyPerDay{count: n}, nil
}

// Count returns the number of casts a day.
func (f SpellFrequencyPerDay) Count() int { return f.count }

func (SpellFrequencyPerDay) isSpellFrequency() {}

// SpellFrequencyPerDayEach is "N/Day Each:": each spell of the group can be
// cast n times a day. The zero value is invalid; build it with
// [NewSpellFrequencyPerDayEach].
type SpellFrequencyPerDayEach struct{ count int }

// NewSpellFrequencyPerDayEach returns the frequency of n casts a day for each
// spell.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange when n is below [MonsterMinUses] or above
// [MonsterMaxNumber]. It never panics.
func NewSpellFrequencyPerDayEach(n int) (SpellFrequencyPerDayEach, error) {
	if n < MonsterMinUses || n > MonsterMaxNumber {
		return SpellFrequencyPerDayEach{}, blockError("SpellFrequencyPerDayEach", BlockOutOfRange)
	}
	return SpellFrequencyPerDayEach{count: n}, nil
}

// Count returns the number of casts a day of each spell.
func (f SpellFrequencyPerDayEach) Count() int { return f.count }

func (SpellFrequencyPerDayEach) isSpellFrequency() {}

// validSpellFrequency reports whether f is a variant its constructor would
// accept.
func validSpellFrequency(f SpellFrequency) bool {
	var err error
	switch v := f.(type) {
	case SpellFrequencyAtWill:
		return true
	case SpellFrequencyPerDay:
		_, err = NewSpellFrequencyPerDay(v.count)
	case SpellFrequencyPerDayEach:
		_, err = NewSpellFrequencyPerDayEach(v.count)
	default:
		return false
	}
	return err == nil
}

// MonsterSpellEntry is one spell of a stat block's Spellcasting: the name
// verbatim, an optional note such as "included in AC" and the level it is cast
// at (SRD 5.2.1, "Spellcasting", p. 257). The zero value is invalid.
// Comparable and safe for concurrent use.
type MonsterSpellEntry struct {
	name        string
	note        string
	castAtLevel int
}

// NewMonsterSpellEntry returns the spell called name, with note ("" for none)
// and castAtLevel, which is 0 (the lowest possible level) or a level from
// [MonsterMinSpellLevel] to [MonsterMaxSpellLevel].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName for a name that is not a valid name, BlockBlankText or
// BlockInvalidCharacter for a non-empty note that breaks the text rule, then
// BlockOutOfRange for a level outside the range. It never panics.
func NewMonsterSpellEntry(name, note string, castAtLevel int) (MonsterSpellEntry, error) {
	const block = "MonsterSpellEntry"
	if err := checkName(block, name); err != nil {
		return MonsterSpellEntry{}, err
	}
	if err := optionalText(block, note); err != nil {
		return MonsterSpellEntry{}, err
	}
	if castAtLevel != 0 && (castAtLevel < MonsterMinSpellLevel || castAtLevel > MonsterMaxSpellLevel) {
		return MonsterSpellEntry{}, blockError(block, BlockOutOfRange)
	}
	return MonsterSpellEntry{name: name, note: note, castAtLevel: castAtLevel}, nil
}

// Name returns the spell name as printed.
func (e MonsterSpellEntry) Name() string { return e.name }

// Note returns the printed note, or "" when there is none.
func (e MonsterSpellEntry) Note() string { return e.note }

// CastAtLevel returns the level the spell is cast at, and false when the
// entry prints none (the lowest possible level).
func (e MonsterSpellEntry) CastAtLevel() (int, bool) { return e.castAtLevel, e.castAtLevel != 0 }

// MonsterSpellGroup is the spells of one frequency in a stat block's
// Spellcasting. The zero value is invalid. It holds a slice, so compare it
// with [MonsterSpellGroup.Equal].
type MonsterSpellGroup struct {
	frequency SpellFrequency
	spells    []MonsterSpellEntry
}

// NewMonsterSpellGroup returns the group of frequency and a copy of spells.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockMissing for
// a nil frequency, BlockInvalidMember for an invalid frequency or a zero
// spell, BlockConstraint for no spell. It never panics.
func NewMonsterSpellGroup(frequency SpellFrequency, spells []MonsterSpellEntry) (MonsterSpellGroup, error) {
	const block = "MonsterSpellGroup"
	switch {
	case frequency == nil:
		return MonsterSpellGroup{}, blockError(block, BlockMissing)
	case !validSpellFrequency(frequency) || slices.Contains(spells, MonsterSpellEntry{}):
		return MonsterSpellGroup{}, blockError(block, BlockInvalidMember)
	case len(spells) == 0:
		return MonsterSpellGroup{}, blockError(block, BlockConstraint)
	}
	return MonsterSpellGroup{frequency: frequency, spells: slices.Clone(spells)}, nil
}

// Frequency returns how often the spells can be cast.
func (g MonsterSpellGroup) Frequency() SpellFrequency { return g.frequency }

// Spells returns the spells in printed order, as a new slice.
func (g MonsterSpellGroup) Spells() []MonsterSpellEntry { return slices.Clone(g.spells) }

// Equal reports whether g and o are the same group.
func (g MonsterSpellGroup) Equal(o MonsterSpellGroup) bool {
	return g.frequency == o.frequency && slices.Equal(g.spells, o.spells)
}

// MonsterSpellcastingFields holds the members of a [MonsterSpellcasting], for
// [NewMonsterSpellcasting]. Construct it with a keyed literal.
type MonsterSpellcastingFields struct {
	// Ability is the spellcasting ability (required).
	Ability ability.Ability
	// SaveDC is the printed spell save DC; it counts only when HasSaveDC is
	// set.
	SaveDC int
	// HasSaveDC says the stat block prints a spell save DC.
	HasSaveDC bool
	// AttackBonus is the printed spell attack bonus; it counts only when
	// HasAttackBonus is set.
	AttackBonus int
	// HasAttackBonus says the stat block prints a spell attack bonus.
	HasAttackBonus bool
	// Components is a printed components statement such as "requiring no
	// Material components"; "" means none.
	Components string
	// Groups are the spell groups in printed order (at least one).
	Groups []MonsterSpellGroup
}

// MonsterSpellcasting is the structured Spellcasting of a stat block part
// (SRD 5.2.1, "Spellcasting", p. 257): the ability, the save DC and attack
// bonus when printed, a printed components statement verbatim and the spell
// groups. The zero value is invalid and means a part has none. It holds a
// slice, so compare it with [MonsterSpellcasting.Equal].
type MonsterSpellcasting struct{ f MonsterSpellcastingFields }

// NewMonsterSpellcasting returns the spellcasting of f, with a copy of its
// groups.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid ability or a zero group, BlockOutOfRange
// for a DC below [MonsterMinSaveDC] or a number above [MonsterMaxNumber],
// BlockBlankText or BlockInvalidCharacter for a non-empty components
// statement that breaks the text rule, then BlockConstraint for no group. It
// never panics.
func NewMonsterSpellcasting(f MonsterSpellcastingFields) (MonsterSpellcasting, error) {
	const block = "MonsterSpellcasting"
	switch {
	case !f.Ability.Valid():
		return MonsterSpellcasting{}, blockError(block, BlockInvalidMember)
	case f.HasSaveDC && (f.SaveDC < MonsterMinSaveDC || f.SaveDC > MonsterMaxNumber),
		f.HasAttackBonus && !monsterNumberOK(f.AttackBonus):
		return MonsterSpellcasting{}, blockError(block, BlockOutOfRange)
	}
	if err := optionalText(block, f.Components); err != nil {
		return MonsterSpellcasting{}, err
	}
	switch {
	case slices.ContainsFunc(f.Groups, func(g MonsterSpellGroup) bool { return g.frequency == nil }):
		return MonsterSpellcasting{}, blockError(block, BlockInvalidMember)
	case len(f.Groups) == 0:
		return MonsterSpellcasting{}, blockError(block, BlockConstraint)
	}
	if !f.HasSaveDC {
		f.SaveDC = 0
	}
	if !f.HasAttackBonus {
		f.AttackBonus = 0
	}
	f.Groups = slices.Clone(f.Groups)
	return MonsterSpellcasting{f: f}, nil
}

// Ability returns the spellcasting ability.
func (s MonsterSpellcasting) Ability() ability.Ability { return s.f.Ability }

// SaveDC returns the printed spell save DC, and false when none is printed.
func (s MonsterSpellcasting) SaveDC() (int, bool) { return s.f.SaveDC, s.f.HasSaveDC }

// AttackBonus returns the printed spell attack bonus, and false when none is
// printed.
func (s MonsterSpellcasting) AttackBonus() (int, bool) { return s.f.AttackBonus, s.f.HasAttackBonus }

// Components returns the printed components statement, or "" when there is
// none.
func (s MonsterSpellcasting) Components() string { return s.f.Components }

// Groups returns the spell groups in printed order, as a new slice.
func (s MonsterSpellcasting) Groups() []MonsterSpellGroup { return slices.Clone(s.f.Groups) }

// IsZero reports whether s is the zero value, meaning no spellcasting.
func (s MonsterSpellcasting) IsZero() bool { return s.f.Ability == 0 }

// Equal reports whether s and o are the same spellcasting.
func (s MonsterSpellcasting) Equal(o MonsterSpellcasting) bool {
	a, b := s.f, o.f
	return a.Ability == b.Ability && a.SaveDC == b.SaveDC && a.HasSaveDC == b.HasSaveDC &&
		a.AttackBonus == b.AttackBonus && a.HasAttackBonus == b.HasAttackBonus && a.Components == b.Components &&
		slices.EqualFunc(a.Groups, b.Groups, MonsterSpellGroup.Equal)
}
