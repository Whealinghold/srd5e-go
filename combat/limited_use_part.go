package combat

// LairStatus says whether a monster is in its lair, which selects the "in Lair"
// numbers of its stat block (SRD 5.2.1, "Aboleth", p. 258). The host reports it;
// the engine has no map. The zero LairStatus is invalid. Comparable and safe for
// concurrent use.
type LairStatus int

// The two lair statuses.
const (
	// InLair: the monster is in its lair.
	InLair LairStatus = iota + 1
	// OutsideLair: the monster is not in its lair.
	OutsideLair
)

// Valid reports whether s is InLair or OutsideLair.
func (s LairStatus) Valid() bool { return s == InLair || s == OutsideLair }

// String returns "in_lair", "outside_lair" or "unknown".
func (s LairStatus) String() string { return enumName([]string{"in_lair", "outside_lair"}, int(s)) }

// MonsterPartSection is a section of a stat block, in the printed order
// (SRD 5.2.1, "Traits", "Actions", "Bonus Actions", "Reactions", "Legendary
// Actions", pp. 256-257). The zero MonsterPartSection is invalid. Comparable and
// safe for concurrent use.
type MonsterPartSection int

// The sections of a stat block, in the printed order.
const (
	// MonsterPartTraits is the Traits section.
	MonsterPartTraits MonsterPartSection = iota + 1
	// MonsterPartActions is the Actions section.
	MonsterPartActions
	// MonsterPartBonusActions is the Bonus Actions section.
	MonsterPartBonusActions
	// MonsterPartReactions is the Reactions section.
	MonsterPartReactions
	// MonsterPartLegendaryActions is the Legendary Actions section.
	MonsterPartLegendaryActions
)

// Valid reports whether s is one of the five sections.
func (s MonsterPartSection) Valid() bool {
	return s >= MonsterPartTraits && s <= MonsterPartLegendaryActions
}

// String returns "traits", "actions", "bonus_actions", "reactions",
// "legendary_actions" or "unknown".
func (s MonsterPartSection) String() string {
	return enumName([]string{"traits", "actions", "bonus_actions", "reactions", "legendary_actions"}, int(s))
}

// LimitedUsePartRef is the position of a stat-block part: its section and its
// printed position (from 0) in that section's list. It is comparable and usable
// as a map key. The zero LimitedUsePartRef is invalid; build one with
// NewLimitedUsePartRef. Safe for concurrent use.
type LimitedUsePartRef struct {
	section MonsterPartSection
	index   int
}

// NewLimitedUsePartRef returns the reference to part index of section.
//
// It returns an error matching srderr.ErrInvalidInput for an undefined section
// or a negative index. It never panics.
func NewLimitedUsePartRef(section MonsterPartSection, index int) (LimitedUsePartRef, error) {
	if !section.Valid() || index < 0 {
		return LimitedUsePartRef{}, invalidInput("part reference needs a section and a non-negative index")
	}
	return LimitedUsePartRef{section, index}, nil
}

// Section returns the section.
func (r LimitedUsePartRef) Section() MonsterPartSection { return r.section }

// Index returns the printed position in the section.
func (r LimitedUsePartRef) Index() int { return r.index }

// IsZero reports whether r is the invalid zero reference.
func (r LimitedUsePartRef) IsZero() bool { return !r.section.Valid() }

// SpellUseRef is the position of a spell in a stat block's Spellcasting: the
// index of its group and of the spell in that group, both from 0. It is
// comparable. The zero SpellUseRef is group 0, spell 0 and valid. Safe for
// concurrent use.
type SpellUseRef struct{ groupIndex, spellIndex int }

// NewSpellUseRef returns the reference to spell spellIndex of group groupIndex.
//
// It returns an error matching srderr.ErrInvalidInput for a negative index. It
// never panics.
func NewSpellUseRef(groupIndex, spellIndex int) (SpellUseRef, error) {
	if groupIndex < 0 || spellIndex < 0 {
		return SpellUseRef{}, invalidInput("spell reference needs non-negative indexes")
	}
	return SpellUseRef{groupIndex, spellIndex}, nil
}

// GroupIndex returns the index of the spell group.
func (r SpellUseRef) GroupIndex() int { return r.groupIndex }

// SpellIndex returns the index of the spell in its group.
func (r SpellUseRef) SpellIndex() int { return r.spellIndex }

// RechargeReadiness says whether a Recharge part can be used. The zero value is
// invalid. Comparable and safe for concurrent use.
type RechargeReadiness int

// The two readinesses.
const (
	// RechargeReady: the part can be used once.
	RechargeReady RechargeReadiness = iota + 1
	// RechargeSpent: the part was used and has not recharged.
	RechargeSpent
)

// String returns "ready", "spent" or "unknown".
func (r RechargeReadiness) String() string { return enumName([]string{"ready", "spent"}, int(r)) }

// RechargeRollResult says whether a Recharge roll recharged its part. The zero
// value is invalid. Comparable and safe for concurrent use.
type RechargeRollResult int

// The two results of a Recharge roll.
const (
	// RechargeRegained: the roll was in the printed range and the part is ready.
	RechargeRegained RechargeRollResult = iota + 1
	// RechargeNotRegained: the roll was outside the range and the part stays spent.
	RechargeNotRegained
)

// String returns "regained", "not_regained" or "unknown": the value of
// srd5e.combat.recharge_outcome.
func (r RechargeRollResult) String() string {
	return enumName([]string{"regained", "not_regained"}, int(r))
}

// LimitedUseAvailability says whether a limited-use part or spell can be used
// now. The zero value is invalid. Comparable and safe for concurrent use.
type LimitedUseAvailability int

// The three availabilities.
const (
	// UseAvailable: a use or a ready recharge remains.
	UseAvailable LimitedUseAvailability = iota + 1
	// UseExhausted: no use remains, or the part is spent.
	UseExhausted
	// UseNotLimited: the reference has no limited-use entry.
	UseNotLimited
)

// String returns "available", "exhausted", "not_limited" or "unknown".
func (a LimitedUseAvailability) String() string {
	return enumName([]string{"available", "exhausted", "not_limited"}, int(a))
}

// LimitedUseRefusal is why a spend was refused. The zero value is invalid.
// Comparable and safe for concurrent use.
type LimitedUseRefusal int

// The refusals of a spend.
const (
	// LimitedUseRefusedExhausted: no use remains, or the part is spent.
	LimitedUseRefusedExhausted LimitedUseRefusal = iota + 1
	// LimitedUseRefusedNotLimited: the reference has no limited-use entry.
	LimitedUseRefusedNotLimited
)

// String returns "exhausted", "not_limited_use" or "unknown".
func (r LimitedUseRefusal) String() string {
	return enumName([]string{"exhausted", "not_limited_use"}, int(r))
}

// LimitedUseSubject is what an event names: a stat-block part or a spell. It is
// a closed set with the variants PartSubject and SpellSubject; consumers switch
// over it exhaustively.
type LimitedUseSubject interface{ isLimitedUseSubject() }

// PartSubject names a stat-block part.
type PartSubject struct{ Part LimitedUsePartRef }

func (PartSubject) isLimitedUseSubject() {}

// SpellSubject names a spell of the Spellcasting.
type SpellSubject struct{ Spell SpellUseRef }

func (SpellSubject) isLimitedUseSubject() {}
