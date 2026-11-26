package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// MonsterTrait is the closed set of the twelve recurring traits of a stat block
// that this package interprets (SRD 5.2.1, "Traits", p. 256). Each is an immutable,
// comparable value holding the LimitedUsePartRef of its part; a zero trait is
// invalid (see IsZero). Variants: LegendaryResistanceTrait, MagicResistanceTrait,
// UndeadFortitudeTrait, RegenerationTrait, DeathBurstTrait, SwarmTrait,
// PackTacticsTrait, SunlightPenaltyTrait, FlybyTrait, AmorphousTrait,
// SiegeMonsterTrait, ParryTrait. Each embeds traitBase.
type MonsterTrait interface {
	// Part returns the position of the trait's part in the stat block.
	Part() LimitedUsePartRef
	// IsZero reports whether the trait is the invalid zero value.
	IsZero() bool
	isMonsterTrait()
}

// traitBase is the part every trait holds; it puts the trait in MonsterTrait.
type traitBase struct{ part LimitedUsePartRef }

// Part returns the position of the trait's part in the stat block.
func (b traitBase) Part() LimitedUsePartRef { return b.part }

// IsZero reports whether the trait is the invalid zero value.
func (b traitBase) IsZero() bool { return b.part.IsZero() }

func (traitBase) isMonsterTrait() {}

func newTraitBase(part LimitedUsePartRef) (traitBase, error) {
	if part.IsZero() {
		return traitBase{}, invalidInput("a trait needs the reference of its part")
	}
	return traitBase{part}, nil
}

// LegendaryResistanceTrait is Legendary Resistance: when the creature fails a saving
// throw it can choose to succeed instead (SRD 5.2.1, "Aboleth", p. 258). Its uses are
// an X/Day entry of the LimitedUseState, not a field of the trait. The zero value is
// invalid. Construct with NewLegendaryResistanceTrait.
type LegendaryResistanceTrait struct{ traitBase }

// NewLegendaryResistanceTrait returns the trait of the part at part. It returns an
// error matching srderr.ErrInvalidInput for the zero reference. It never panics.
func NewLegendaryResistanceTrait(part LimitedUsePartRef) (LegendaryResistanceTrait, error) {
	b, err := newTraitBase(part)
	return LegendaryResistanceTrait{b}, err
}

// MagicResistanceTrait is Magic Resistance: Advantage on saving throws against spells
// and other magical effects (SRD 5.2.1, "Balor", p. 261). The zero value is invalid.
// Construct with NewMagicResistanceTrait.
type MagicResistanceTrait struct{ traitBase }

// NewMagicResistanceTrait returns the trait of the part at part. It returns an error
// matching srderr.ErrInvalidInput for the zero reference. It never panics.
func NewMagicResistanceTrait(part LimitedUsePartRef) (MagicResistanceTrait, error) {
	b, err := newTraitBase(part)
	return MagicResistanceTrait{b}, err
}

// UndeadFortitudeTrait is Undead Fortitude: a Constitution saving throw keeps the
// creature at 1 Hit Point (SRD 5.2.1, "Zombie", p. 343). The zero value is invalid.
// Construct with NewUndeadFortitudeTrait.
type UndeadFortitudeTrait struct{ traitBase }

// NewUndeadFortitudeTrait returns the trait of the part at part. It returns an error
// matching srderr.ErrInvalidInput for the zero reference. It never panics.
func NewUndeadFortitudeTrait(part LimitedUsePartRef) (UndeadFortitudeTrait, error) {
	b, err := newTraitBase(part)
	return UndeadFortitudeTrait{b}, err
}

// SwarmTrait is the Swarm trait of the swarm stat blocks: the creature shares spaces,
// fits through the opening of a Tiny creature and cannot regain Hit Points or gain
// Temporary Hit Points (SRD 5.2.1, "Swarm of Crawling Claws", p. 278). The zero value
// is invalid. Construct with NewSwarmTrait.
type SwarmTrait struct{ traitBase }

// NewSwarmTrait returns the trait of the part at part. It returns an error matching
// srderr.ErrInvalidInput for the zero reference. It never panics.
func NewSwarmTrait(part LimitedUsePartRef) (SwarmTrait, error) {
	b, err := newTraitBase(part)
	return SwarmTrait{b}, err
}

// PackTacticsTrait is Pack Tactics: Advantage on attack rolls against a creature an
// ally threatens (SRD 5.2.1, "Wolf", p. 364). The zero value is invalid. Construct with
// NewPackTacticsTrait.
type PackTacticsTrait struct{ traitBase }

// NewPackTacticsTrait returns the trait of the part at part. It returns an error
// matching srderr.ErrInvalidInput for the zero reference. It never panics.
func NewPackTacticsTrait(part LimitedUsePartRef) (PackTacticsTrait, error) {
	b, err := newTraitBase(part)
	return PackTacticsTrait{b}, err
}

// FlybyTrait is Flyby: no Opportunity Attack when the creature flies out of an enemy's
// reach (SRD 5.2.1, "Hippogriff", p. 298). The zero value is invalid. Construct with
// NewFlybyTrait.
type FlybyTrait struct{ traitBase }

// NewFlybyTrait returns the trait of the part at part. It returns an error matching
// srderr.ErrInvalidInput for the zero reference. It never panics.
func NewFlybyTrait(part LimitedUsePartRef) (FlybyTrait, error) {
	b, err := newTraitBase(part)
	return FlybyTrait{b}, err
}

// AmorphousTrait is Amorphous: the creature moves through a space as narrow as 1 inch
// at no extra cost (SRD 5.2.1, "Black Pudding", p. 265). The zero value is invalid.
// Construct with NewAmorphousTrait.
type AmorphousTrait struct{ traitBase }

// NewAmorphousTrait returns the trait of the part at part. It returns an error matching
// srderr.ErrInvalidInput for the zero reference. It never panics.
func NewAmorphousTrait(part LimitedUsePartRef) (AmorphousTrait, error) {
	b, err := newTraitBase(part)
	return AmorphousTrait{b}, err
}

// SiegeMonsterTrait is Siege Monster: double damage to objects and structures (SRD
// 5.2.1, "Treant", p. 333). The zero value is invalid. Construct with
// NewSiegeMonsterTrait.
type SiegeMonsterTrait struct{ traitBase }

// NewSiegeMonsterTrait returns the trait of the part at part. It returns an error
// matching srderr.ErrInvalidInput for the zero reference. It never panics.
func NewSiegeMonsterTrait(part LimitedUsePartRef) (SiegeMonsterTrait, error) {
	b, err := newTraitBase(part)
	return SiegeMonsterTrait{b}, err
}

// damageTypeSet is a set of Damage Types as a bit mask, so a trait holding one stays
// comparable.
type damageTypeSet uint16

func (s damageTypeSet) has(t content.DamageType) bool {
	return t.Valid() && s&(1<<(uint(t)-1)) != 0
}

// RegenerationTrait is Regeneration: the creature regains a fixed number of Hit Points
// at the start of each of its turns (SRD 5.2.1, "Oni", p. 312; "Troll", p. 333). With
// no suppressing Damage Type it is the first form (only while the creature has at least
// 1 Hit Point); with suppressing types it is the second (taking one of them stops the
// trait on the next turn, and the creature dies only if it starts its turn with 0 Hit
// Points and does not regenerate). The zero value is invalid. Construct with
// NewRegenerationTrait. It is comparable.
type RegenerationTrait struct {
	traitBase
	amount      int
	suppressing damageTypeSet
	swarm       bool
}

// NewRegenerationTrait returns the Regeneration of amount Hit Points a turn for the
// part at part, suppressed by the Damage Types suppressing (none for the first form).
//
// It returns an error matching srderr.ErrInvalidInput for the zero reference, an amount
// below 1 or above content.MonsterMaxNumber, or a Damage Type that is not valid. A
// repeated type counts once. It never panics.
func NewRegenerationTrait(part LimitedUsePartRef, amount int, suppressing []content.DamageType) (RegenerationTrait, error) {
	b, err := newTraitBase(part)
	if err != nil {
		return RegenerationTrait{}, err
	}
	if amount < 1 || amount > content.MonsterMaxNumber {
		return RegenerationTrait{}, invalidInput("regeneration amount outside 1 to the largest stat block number")
	}
	var set damageTypeSet
	for _, t := range suppressing {
		if !t.Valid() {
			return RegenerationTrait{}, invalidInput("regeneration suppressed by an invalid Damage Type")
		}
		set |= 1 << (uint(t) - 1)
	}
	return RegenerationTrait{traitBase: b, amount: amount, suppressing: set}, nil
}

// Amount returns the Hit Points regained each turn.
func (t RegenerationTrait) Amount() int { return t.amount }

// SuppressingTypes returns, as a new slice and in Damage Type order, the Damage Types
// that stop the trait for a turn; empty for the first form.
func (t RegenerationTrait) SuppressingTypes() []content.DamageType {
	var out []content.DamageType
	for ty := content.DamageAcid; ty <= content.DamageThunder; ty++ {
		if t.suppressing.has(ty) {
			out = append(out, ty)
		}
	}
	return out
}

// SwarmCannotRegain reports whether the monster is also a swarm, which cannot regain
// Hit Points, so the trait never functions (WithSwarm).
func (t RegenerationTrait) SwarmCannotRegain() bool { return t.swarm }

// WithSwarm returns the trait marked as belonging to a swarm, whose Regeneration never
// functions because a swarm cannot regain Hit Points. It returns an error matching
// srderr.ErrInvalidInput for a zero swarm or a zero trait. It never panics.
func (t RegenerationTrait) WithSwarm(s SwarmTrait) (RegenerationTrait, error) {
	if t.IsZero() || s.IsZero() {
		return RegenerationTrait{}, invalidInput("a swarm mark needs a Regeneration and a Swarm")
	}
	t.swarm = true
	return t, nil
}

// hasSuppression reports whether this is the second form.
func (t RegenerationTrait) hasSuppression() bool { return t.suppressing != 0 }

// suppressedBy reports whether any of dealt stops the trait.
func (t RegenerationTrait) suppressedBy(dealt []content.DamageType) bool {
	return slices.ContainsFunc(dealt, t.suppressing.has)
}

// DeathBurstTrait is Death Burst: the creature explodes when it dies, and the saving
// throw of the burst is the part's (SRD 5.2.1, "Magmin", p. 305). The zero value is
// invalid. Construct with NewDeathBurstTrait.
type DeathBurstTrait struct {
	traitBase
	ability ability.Ability
	dc      int
}

// NewDeathBurstTrait returns the trait of the part at part with the first saving
// throw of the part, save.
//
// It returns an error matching srderr.ErrInvalidInput for the zero reference, the zero
// saving throw, an ability that is not valid or a difficulty class below
// content.MonsterMinSaveDC or above content.MonsterMaxNumber. It never panics.
func NewDeathBurstTrait(part LimitedUsePartRef, save content.StatBlockSavingThrow) (DeathBurstTrait, error) {
	b, err := newTraitBase(part)
	if err != nil {
		return DeathBurstTrait{}, err
	}
	if save.IsZero() || !save.Ability().Valid() || save.DC() < content.MonsterMinSaveDC || save.DC() > content.MonsterMaxNumber {
		return DeathBurstTrait{}, invalidInput("death burst needs a saving throw with an ability and a difficulty class in range")
	}
	return DeathBurstTrait{traitBase: b, ability: save.Ability(), dc: save.DC()}, nil
}

// Ability returns the ability of the burst's saving throw.
func (t DeathBurstTrait) Ability() ability.Ability { return t.ability }

// DC returns the difficulty class of the burst's saving throw.
func (t DeathBurstTrait) DC() int { return t.dc }

// SunlightPenaltyScope says which D20 Tests a sunlight penalty covers. The zero value
// is invalid. Comparable and safe for concurrent use.
type SunlightPenaltyScope int

const (
	// SunlightPenaltyAbilityChecksAndAttackRolls: Disadvantage on ability checks and
	// attack rolls (Sunlight Sensitivity, SRD 5.2.1, "Drider", p. 281).
	SunlightPenaltyAbilityChecksAndAttackRolls SunlightPenaltyScope = iota + 1
	// SunlightPenaltyAllD20Tests: Disadvantage on every D20 Test (Sunlight Weakness,
	// SRD 5.2.1, "Shadow", p. 322).
	SunlightPenaltyAllD20Tests
)

// Valid reports whether s is one of the two scopes.
func (s SunlightPenaltyScope) Valid() bool {
	return s == SunlightPenaltyAbilityChecksAndAttackRolls || s == SunlightPenaltyAllD20Tests
}

// String returns "ability_checks_and_attack_rolls", "all_d20_tests" or "unknown".
func (s SunlightPenaltyScope) String() string {
	return enumName([]string{"ability_checks_and_attack_rolls", "all_d20_tests"}, int(s))
}

// SunlightPenaltyTrait is Sunlight Sensitivity or Sunlight Weakness: Disadvantage while
// in sunlight on the D20 Tests its scope covers. The zero value is invalid. Construct
// with NewSunlightPenaltyTrait.
type SunlightPenaltyTrait struct {
	traitBase
	scope SunlightPenaltyScope
}

// NewSunlightPenaltyTrait returns the trait of the part at part with the given scope.
// It returns an error matching srderr.ErrInvalidInput for the zero reference or a scope
// that is not valid. It never panics.
func NewSunlightPenaltyTrait(part LimitedUsePartRef, scope SunlightPenaltyScope) (SunlightPenaltyTrait, error) {
	b, err := newTraitBase(part)
	if err != nil {
		return SunlightPenaltyTrait{}, err
	}
	if !scope.Valid() {
		return SunlightPenaltyTrait{}, invalidInput("a sunlight penalty needs a scope")
	}
	return SunlightPenaltyTrait{b, scope}, nil
}

// Scope returns the D20 Tests the penalty covers.
func (t SunlightPenaltyTrait) Scope() SunlightPenaltyScope { return t.scope }

// ParryTrait is Parry, a Reaction: add a bonus to Armor Class against a melee attack
// roll that hits while the creature holds a weapon (SRD 5.2.1, "Bandit", p. 262). The
// zero value is invalid. Construct with NewParryTrait.
type ParryTrait struct {
	traitBase
	bonus int
}

// NewParryTrait returns the trait of the part at part adding bonus to Armor Class.
// It returns an error matching srderr.ErrInvalidInput for the zero reference or a
// bonus below 1 or above content.MonsterMaxNumber. It never panics.
func NewParryTrait(part LimitedUsePartRef, bonus int) (ParryTrait, error) {
	b, err := newTraitBase(part)
	if err != nil {
		return ParryTrait{}, err
	}
	if bonus < 1 || bonus > content.MonsterMaxNumber {
		return ParryTrait{}, invalidInput("parry bonus outside 1 to the largest stat block number")
	}
	return ParryTrait{b, bonus}, nil
}

// Bonus returns the number added to Armor Class.
func (t ParryTrait) Bonus() int { return t.bonus }
