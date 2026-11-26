package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
)

// DealDamageToCombatantCommand is the immutable input of DealDamageToCombatant.Run. Build it with
// NewDealDamageToCombatantCommand; the zero value is invalid.
type DealDamageToCombatantCommand struct {
	encounter     Encounter
	target        CombatantID
	damage        []Damage
	adjustments   []DamageAdjustment
	delivery      DamageDelivery
	hitKind       HitKind
	concentration ConcentrationInput
}

// NewDealDamageToCombatantCommand returns the command, taking its own copy of the lists. The damage is what the
// host rolled once, one Damage per component, for a source that is not an attack: a spell's damage after the
// save, a hazard, an effect. Several targets are several calls (SRD 5.2.1, "Damage against Multiple Targets",
// p. 16). It returns an error matching srderr.ErrInvalidInput for a zero encounter, a blank id, an empty damage
// list, a zero or negative Damage, a nil adjustment or a delivery or hit kind that is not defined. It never
// panics.
func NewDealDamageToCombatantCommand(encounter Encounter, target CombatantID, damage []Damage, adjustments []DamageAdjustment,
	delivery DamageDelivery, hitKind HitKind) (DealDamageToCombatantCommand, error) {
	if encounter.IsZero() || target.IsZero() {
		return DealDamageToCombatantCommand{}, invalidInput("damage needs an encounter and a target")
	}
	if err := checkDamageInput(target, damage, adjustments, delivery, hitKind); err != nil {
		return DealDamageToCombatantCommand{}, err
	}
	return DealDamageToCombatantCommand{encounter, target, slices.Clone(damage), slices.Clone(adjustments), delivery, hitKind, NoConcentrationTracking{}}, nil
}

// WithTargetConcentration returns the command with the Concentration of the damaged combatant, the target, as the
// host holds it (FR-078); a command has NoConcentrationTracking until it is given one. It returns an error
// matching srderr.ErrInvalidInput for a nil input or a zero TracksConcentration, and never panics.
func (c DealDamageToCombatantCommand) WithTargetConcentration(in ConcentrationInput) (DealDamageToCombatantCommand, error) {
	if err := validConcentrationInput(in); err != nil {
		return DealDamageToCombatantCommand{}, err
	}
	c.concentration = in
	return c, nil
}

// DealDamageToCombatantResult is the closed set of outcomes of DealDamageToCombatant.Run. Variants:
// DealDamageToCombatantCompleted, DealDamageToCombatantRejected, DealDamageToCombatantInvalid.
type DealDamageToCombatantResult interface{ isDealDamageToCombatantResult() }

// DealDamageToCombatantCompleted: the damage was applied. It holds the new encounter, the events in order and the
// damage resolutions, one per Damage.
type DealDamageToCombatantCompleted struct {
	resolutionCompleted
	resolutions []DamageResolution
	concentration
}

// DealDamageToCombatantRejected: the encounter ended or the target is dead; the encounter is as it came in.
type DealDamageToCombatantRejected struct{ resolutionRejected }

// DealDamageToCombatantInvalid: the host named a combatant the encounter does not hold; the encounter is as it
// came in.
type DealDamageToCombatantInvalid struct{ resolutionInvalid }

func (DealDamageToCombatantCompleted) isDealDamageToCombatantResult() {}
func (DealDamageToCombatantRejected) isDealDamageToCombatantResult()  {}
func (DealDamageToCombatantInvalid) isDealDamageToCombatantResult()   {}

// Resolutions returns a copy of the damage resolutions, equal to what the encounter returns for the same inputs.
func (c DealDamageToCombatantCompleted) Resolutions() []DamageResolution {
	return slices.Clone(c.resolutions)
}

// Concentration returns how the Concentration of the target came out: ConcentrationNotChecked when the host gave
// none, none was in progress or no damage was taken.
func (c DealDamageToCombatantCompleted) Concentration() ConcentrationCheckOutcome { return c.outcome }

// TargetSpellcasting returns the new SpellcastingState of the target and whether the host gave a
// TracksConcentration; it is the state of the next call.
func (c DealDamageToCombatantCompleted) TargetSpellcasting() (character.SpellcastingState, bool) {
	return c.targetSpellcasting()
}

// DealDamageToCombatant is the use case of damage that is not an attack (SRD 5.2.1, "Damage Rolls", p. 16). It
// plays the role of ApplyDamageToCombatant and adds the prompts of the conditions the damage caused. The source
// is not an attacker of the encounter: no attack is made and no attacker's condition is read (FR-077). Create it
// with NewDealDamageToCombatant.
type DealDamageToCombatant struct{ resolutionUse }

// NewDealDamageToCombatant returns the use case. It takes the roller first, as the other use cases that roll do,
// because the Concentration save of a later revision of the spec is rolled through it; this revision rolls no
// die. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewDealDamageToCombatant(roller Roller, opts ...Option) (DealDamageToCombatant, error) {
	if roller == nil {
		return DealDamageToCombatant{}, invalidInput("damage needs a dice roller")
	}
	return DealDamageToCombatant{newResolutionUse(roller, opts)}, nil
}

// Run applies the damage in one command to the encounter, so Death Saving Throw failures and the Massive Damage
// test happen once, and counts one srd5e.combat.deaths per death. It returns DealDamageToCombatantRejected for an
// ended encounter or a dead target and DealDamageToCombatantInvalid for an unknown one. It returns the errors
// described for the package.
func (u DealDamageToCombatant) Run(ctx context.Context, cmd DealDamageToCombatantCommand) (DealDamageToCombatantResult, error) {
	if u.flow.roller == nil || len(cmd.damage) == 0 {
		return nil, invalidInput("zero use case or command")
	}
	op := combatOperation{valueOperationDealDamageToCombatant, spanDealDamageToCombatant}
	return execute(ctx, u.resolutionUse, op, func() (DealDamageToCombatantResult, error) { return u.deal(cmd) })
}

// deal is the work of Run.
func (u DealDamageToCombatant) deal(cmd DealDamageToCombatantCommand) (DealDamageToCombatantResult, error) {
	if s := preconditions(cmd.encounter, cmd.target); s.stopped() {
		if s.problem != nil {
			return DealDamageToCombatantInvalid{resolutionInvalid{s.problem, cmd.encounter}}, nil
		}
		return DealDamageToCombatantRejected{resolutionRejected{s.rejection, cmd.encounter}}, nil
	}
	tr, err := cmd.encounter.ApplyDamage(cmd.target, cmd.damage, cmd.adjustments, cmd.delivery, cmd.hitKind)
	if err != nil {
		return nil, err
	}
	r := &resolution{enc: cmd.encounter}
	resolutions := r.mustCommit(tr)
	r.prompts(cmd.encounter)
	conc, err := u.checkConcentration(r, cmd.target, cmd.concentration, tr.Events())
	if err != nil {
		return nil, err
	}
	r.concentrated(cmd.target, conc)
	if r.overLimit() {
		return DealDamageToCombatantInvalid{resolutionInvalid{LimitExceededProblem{}, cmd.encounter}}, nil
	}
	if r.exitInvalid() {
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return DealDamageToCombatantCompleted{resolutionCompleted{r.enc, r.events}, resolutions, conc}, nil
}
