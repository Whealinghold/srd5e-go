package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// MonsterCastRefusal is why a cast was refused. The values are in the order the
// first that applies is returned (FR-031); MonsterCastRefusalNotInProgress belongs to
// completing a casting only. The zero value is invalid. Comparable and safe for
// concurrent use.
type MonsterCastRefusal int

// The refusals, in precedence order.
const (
	// MonsterCastRefusalNotInSpellList: the reference names no spell of the trait.
	MonsterCastRefusalNotInSpellList MonsterCastRefusal = iota + 1
	// MonsterCastRefusalRefMismatch: the completion is not a ListCast of the stored reference.
	MonsterCastRefusalRefMismatch
	// MonsterCastRefusalAlreadyInProgress: a casting of 1 minute or more is in progress.
	MonsterCastRefusalAlreadyInProgress
	// MonsterCastRefusalCannotTakeActions: the monster cannot take actions.
	MonsterCastRefusalCannotTakeActions
	// MonsterCastRefusalComponentsUnrecognized: the components statement is outside the grammar.
	MonsterCastRefusalComponentsUnrecognized
	// MonsterCastRefusalVerbalMissing: a Verbal component is needed and the monster cannot speak.
	MonsterCastRefusalVerbalMissing
	// MonsterCastRefusalSomaticMissing: a Somatic component is needed and the monster cannot gesture.
	MonsterCastRefusalSomaticMissing
	// MonsterCastRefusalSelfOnlyViolated: a "self only" spell was aimed at another creature.
	MonsterCastRefusalSelfOnlyViolated
	// MonsterCastRefusalMissingSaveDC: the spell forces a saving throw and no DC is printed.
	MonsterCastRefusalMissingSaveDC
	// MonsterCastRefusalMissingSpellAttackBonus: the spell makes a spell attack roll and no bonus is printed.
	MonsterCastRefusalMissingSpellAttackBonus
	// MonsterCastRefusalUsesExhausted: no use of the spell remains.
	MonsterCastRefusalUsesExhausted
	// MonsterCastRefusalNotInProgress: there is no casting to complete.
	MonsterCastRefusalNotInProgress
)

// String returns the snake_case name or "unknown".
func (r MonsterCastRefusal) String() string {
	return enumName([]string{"not_in_spell_list", "casting_ref_mismatch", "casting_already_in_progress", "cannot_take_actions",
		"components_statement_unrecognized", "verbal_component_missing", "somatic_component_missing", "self_only_violated",
		"missing_save_dc", "missing_spell_attack_bonus", "uses_exhausted", "casting_not_in_progress"}, int(r))
}

// MonsterCastReport is what a finished cast hands to the effects: the spell, the
// level it was cast at and scales with, the printed save DC and spell attack bonus
// as printed, the ability and the monster's Modifier in it, whom the cast may affect
// and the entry's note text (SRD 5.2.1, "Spellcasting", p. 257). Comparable and safe
// for concurrent use. The zero value is invalid.
type MonsterCastReport struct {
	spell                     content.SpellID
	castAtLevel, scalingLevel int
	saveDC, attackBonus       int
	hasSaveDC, hasAttackBonus bool
	spellAbility              ability.Ability
	abilityModifier           ability.Modifier
	targetLimit               TargetLimit
	restrictionNote           string
}

// Spell returns the spell id.
func (r MonsterCastReport) Spell() content.SpellID { return r.spell }

// CastAtLevel returns the level the spell was cast at and false for a cantrip.
func (r MonsterCastReport) CastAtLevel() (int, bool) { return r.castAtLevel, r.castAtLevel > 0 }

// ScalingLevel returns the level the spell scales with, 1 for a cantrip.
func (r MonsterCastReport) ScalingLevel() int { return r.scalingLevel }

// SaveDC returns the printed spell save DC, unchanged, and whether one is printed.
func (r MonsterCastReport) SaveDC() (d20.DifficultyClass, bool) {
	return d20.DifficultyClass(r.saveDC), r.hasSaveDC
}

// AttackModifiers returns the modifiers of a spell attack roll: one ability slot
// holding the printed spell attack bonus, no Proficiency Bonus and no others, so
// their sum is the printed bonus. It returns false when no bonus is printed.
func (r MonsterCastReport) AttackModifiers() (d20.RollModifiers, bool) {
	if !r.hasAttackBonus {
		return d20.RollModifiers{}, false
	}
	return d20.NewRollModifiers(modifier.New(r.attackBonus)), true
}

// ToCastReport converts the report to a character.CastReport (FR-030, amendment A1): source
// character.CastSourceMonster, no casting class, no slot, the cast and scaling levels as they
// are, and the printed save DC and spell attack bonus exactly as printed, never recomputed from
// an ability and a Proficiency Bonus. A figure that is not printed stays absent. It returns an
// error matching srderr.ErrInvalidInput for the zero MonsterCastReport or a printed DC below 1.
// It never panics.
func (r MonsterCastReport) ToCastReport() (character.CastReport, error) {
	p := character.CastReportParts{Spell: r.spell, Source: character.CastSourceMonster, CastAtLevel: r.castAtLevel, ScalingLevel: r.scalingLevel}
	if r.hasSaveDC {
		dc, err := character.NewSpellSaveDC(r.saveDC)
		if err != nil {
			return character.CastReport{}, err
		}
		p.SaveDC = &dc
	}
	if r.hasAttackBonus {
		bonus := character.NewSpellAttackBonus(r.attackBonus)
		p.AttackBonus = &bonus
	}
	return character.NewCastReport(p)
}

// Ability returns the spellcasting ability used.
func (r MonsterCastReport) Ability() ability.Ability { return r.spellAbility }

// AbilityModifier returns the Modifier of the monster's score in the spellcasting
// ability, for a spell that adds the spellcasting ability modifier.
func (r MonsterCastReport) AbilityModifier() ability.Modifier { return r.abilityModifier }

// TargetLimit returns whom the cast may affect.
func (r MonsterCastReport) TargetLimit() TargetLimit { return r.targetLimit }

// RestrictionNote returns the entry's printed note unchanged, "" when none. A
// Temporary Hit Points waiver is carried in this text.
func (r MonsterCastReport) RestrictionNote() string { return r.restrictionNote }

// MonsterSpellcastingTransition is the new states a cast or a change of
// Concentration leaves, with the events that produced them in the order they
// happened, and the limited-use events of the same call in their own list. The zero
// value is the zero states and no events.
type MonsterSpellcastingTransition struct {
	state            MonsterSpellcastingState
	limitedUse       LimitedUseState
	events           []MonsterSpellcastingEvent
	limitedUseEvents []MonsterLimitedUseEvent
}

// State returns the new spellcasting state.
func (t MonsterSpellcastingTransition) State() MonsterSpellcastingState { return t.state }

// LimitedUse returns the new limited-use state.
func (t MonsterSpellcastingTransition) LimitedUse() LimitedUseState { return t.limitedUse }

// Events returns a copy of the spellcasting events.
func (t MonsterSpellcastingTransition) Events() []MonsterSpellcastingEvent {
	return slices.Clone(t.events)
}

// LimitedUseEvents returns a copy of the limited-use events.
func (t MonsterSpellcastingTransition) LimitedUseEvents() []MonsterLimitedUseEvent {
	return slices.Clone(t.limitedUseEvents)
}

// MonsterCastOutcome is the closed set of outcomes of a cast. Variants:
// MonsterCastPerformed, MonsterCastStarted, MonsterCastRefused. Consumers switch over
// it exhaustively.
type MonsterCastOutcome interface{ isMonsterCastOutcome() }

// MonsterCastPerformed: the cast finished. Report is for the effects and Transition
// holds the new states and events.
type MonsterCastPerformed struct {
	Report     MonsterCastReport
	Transition MonsterSpellcastingTransition
}

func (MonsterCastPerformed) isMonsterCastOutcome() {}

// MonsterCastStarted: a casting of 1 minute or more began; no use is spent.
type MonsterCastStarted struct{ Transition MonsterSpellcastingTransition }

func (MonsterCastStarted) isMonsterCastOutcome() {}

// MonsterCastRefused: a rule refused the cast. Both states are unchanged and nothing
// was emitted.
type MonsterCastRefused struct{ Reason MonsterCastRefusal }

func (MonsterCastRefused) isMonsterCastOutcome() {}
