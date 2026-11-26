package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ConcentrationDamageRule says whether damage can break the Concentration of a combatant: a fact the host
// states from any source, for example the Ranger Hunter's Mark from level 13 (combat-action-resolution.md
// FR-158; SRD 5.2.1, "Concentration", p. 179). The zero value is invalid; TracksConcentration reads it as
// DamageMayBreakConcentration.
type ConcentrationDamageRule int

// The two rules.
const (
	// DamageMayBreakConcentration: damage calls for the Constitution saving throw. The default.
	DamageMayBreakConcentration ConcentrationDamageRule = iota + 1
	// DamageCannotBreakConcentration: damage never calls for the save; Incapacitated and death still end
	// Concentration.
	DamageCannotBreakConcentration
)

// Valid reports whether r is one of the two rules.
func (r ConcentrationDamageRule) Valid() bool {
	return r == DamageMayBreakConcentration || r == DamageCannotBreakConcentration
}

// String returns "damage_may_break_concentration", "damage_cannot_break_concentration" or "unknown".
func (r ConcentrationDamageRule) String() string {
	return enumName([]string{"damage_may_break_concentration", "damage_cannot_break_concentration"}, int(r))
}

// ConcentrationInput is what the host tells a use case about the Concentration of the damaged combatant
// (combat-action-resolution.md FR-078, section 5). The caster's state is the host's, not the encounter's.
// Variants: NoConcentrationTracking, TracksConcentration.
type ConcentrationInput interface{ isConcentrationInput() }

// NoConcentrationTracking: the host tracks no Concentration for the combatant, so none is checked. The zero
// value of a command reads as this.
type NoConcentrationTracking struct{}

// TracksConcentration: the host holds the SpellcastingState of the combatant. Build it with
// NewTracksConcentration. Not comparable with ==; its state holds slices.
type TracksConcentration struct {
	state         character.SpellcastingState
	modifiers     d20.RollModifiers
	circumstances []modifier.RollCircumstance
	rule          ConcentrationDamageRule
}

func (NoConcentrationTracking) isConcentrationInput() {}
func (TracksConcentration) isConcentrationInput()     {}

// NewTracksConcentration returns the input: the combatant's SpellcastingState, the modifiers and the
// circumstances of its Constitution saving throw that the engine cannot know (the ability modifier and
// proficiency of the character, for example Advantage from a feature), and the rule; the zero rule is
// DamageMayBreakConcentration. It copies the circumstances. It returns an error matching
// srderr.ErrInvalidInput for a zero state, a nil circumstance or a rule that is not defined. It never panics.
func NewTracksConcentration(state character.SpellcastingState, mods d20.RollModifiers, circumstances []modifier.RollCircumstance,
	rule ConcentrationDamageRule) (TracksConcentration, error) {
	if rule == 0 {
		rule = DamageMayBreakConcentration
	}
	if state.IsZero() || !rule.Valid() || slices.Contains(circumstances, nil) {
		return TracksConcentration{}, invalidInput("tracking Concentration needs a state, valid circumstances and a rule")
	}
	return TracksConcentration{state, mods, slices.Clone(circumstances), rule}, nil
}

// State returns the SpellcastingState the host passed in.
func (t TracksConcentration) State() character.SpellcastingState { return t.state }

// Modifiers returns the modifiers of the save.
func (t TracksConcentration) Modifiers() d20.RollModifiers { return t.modifiers }

// Circumstances returns a copy of the circumstances of the save.
func (t TracksConcentration) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(t.circumstances)
}

// DamageRule returns the rule.
func (t TracksConcentration) DamageRule() ConcentrationDamageRule { return t.rule }

// ConcentrationCheckOutcome is how the Concentration of a damaged combatant came out (combat-action-resolution.md
// FR-079): domain values only (FR-121). Variants: ConcentrationNotChecked, ConcentrationHeld,
// ConcentrationCheckFailed, ConcentrationEndedWithoutSave.
type ConcentrationCheckOutcome interface{ isConcentrationCheckOutcome() }

// ConcentrationNotChecked: nothing was checked: no tracking, no Concentration in progress, no damage taken, or
// damage that cannot break it.
type ConcentrationNotChecked struct{}

// ConcentrationHeld: the Constitution saving throw succeeded; the state is unchanged.
type ConcentrationHeld struct {
	dc   d20.DifficultyClass
	save TestResolution
}

// ConcentrationCheckFailed: the Constitution saving throw failed and Concentration ended. Not comparable with
// ==; it holds a slice.
type ConcentrationCheckFailed struct {
	dc     d20.DifficultyClass
	save   TestResolution
	events []character.CastingEvent
}

// ConcentrationEndedWithoutSave: Concentration ended because the combatant is Incapacitated or dead, with no
// die. Build it with NewConcentrationEndedWithoutSave. Not comparable with ==; it holds a slice.
type ConcentrationEndedWithoutSave struct {
	reason character.ConcentrationBreakReason
	events []character.CastingEvent
}

func (ConcentrationNotChecked) isConcentrationCheckOutcome()       {}
func (ConcentrationHeld) isConcentrationCheckOutcome()             {}
func (ConcentrationCheckFailed) isConcentrationCheckOutcome()      {}
func (ConcentrationEndedWithoutSave) isConcentrationCheckOutcome() {}

// DC returns the Difficulty Class of the save.
func (o ConcentrationHeld) DC() d20.DifficultyClass { return o.dc }

// Save returns the Constitution saving throw.
func (o ConcentrationHeld) Save() TestResolution { return o.save }

// DC returns the Difficulty Class of the save.
func (o ConcentrationCheckFailed) DC() d20.DifficultyClass { return o.dc }

// Save returns the Constitution saving throw.
func (o ConcentrationCheckFailed) Save() TestResolution { return o.save }

// Events returns a copy of the events the break transition returned, in order.
func (o ConcentrationCheckFailed) Events() []character.CastingEvent { return slices.Clone(o.events) }

// Reason returns the reason of the break: Incapacitated or Dead.
func (o ConcentrationEndedWithoutSave) Reason() character.ConcentrationBreakReason { return o.reason }

// Events returns a copy of the events the break transition returned, in order.
func (o ConcentrationEndedWithoutSave) Events() []character.CastingEvent {
	return slices.Clone(o.events)
}

// NewConcentrationEndedWithoutSave returns the outcome for reason and the events of the break transition. It
// returns an error matching srderr.ErrInvalidInput for a reason other than BreakIncapacitated or BreakDead
// (FR-157). It copies events and never panics.
func NewConcentrationEndedWithoutSave(reason character.ConcentrationBreakReason, events []character.CastingEvent) (ConcentrationEndedWithoutSave, error) {
	if reason != character.BreakIncapacitated && reason != character.BreakDead {
		return ConcentrationEndedWithoutSave{}, invalidInput("Concentration ends without a save only when Incapacitated or dead")
	}
	return ConcentrationEndedWithoutSave{reason, slices.Clone(events)}, nil
}
