package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseDivineInterventionCommand is the input of UseDivineIntervention.Run: the character, the spell chosen, the chosen
// casting time (nil for the first of the spell's casting times) and the rule data (DefaultClericRuleData for the SRD).
// Construct it with a keyed literal. Run does not alter it.
type UseDivineInterventionCommand struct {
	Character   Character
	Spell       content.Spell
	CastingTime content.CastingTime
	Rules       ClericRuleData
}

// DivineInterventionCast is the free cast Divine Intervention asks the host to perform with CastSpell and
// CastModeFeatureGranted: the spell and the casting time, with no spell slot choice and no Material components needed (SRD
// 5.2.1, "Level 10: Divine Intervention", p. 38). Entitlement is decided; the cast is the host's.
type DivineInterventionCast struct {
	Spell       content.SpellID
	CastingTime content.CastingTime
}

// SpellSlotNeeded reports whether the cast expends a spell slot: never.
func (DivineInterventionCast) SpellSlotNeeded() bool { return false }

// MaterialComponentsNeeded reports whether the cast needs Material components: never.
func (DivineInterventionCast) MaterialComponentsNeeded() bool { return false }

// UseDivineInterventionResult is the closed set of outcomes of UseDivineIntervention.Run. Variants:
// UseDivineInterventionApplied, UseDivineInterventionRejected, UseDivineInterventionNotACleric.
type UseDivineInterventionResult interface{ isUseDivineInterventionResult() }

// UseDivineInterventionApplied is the outcome of a use. Only Run builds one.
type UseDivineInterventionApplied struct {
	character Character
	cast      DivineInterventionCast
	resources ClericResources
	roll      []int
	events    []Event
}

// Character returns the new Character, with Divine Intervention used.
func (r UseDivineInterventionApplied) Character() Character { return r.character }

// Cast returns the free cast the host performs.
func (r UseDivineInterventionApplied) Cast() DivineInterventionCast { return r.cast }

// Resources returns the new resources.
func (r UseDivineInterventionApplied) Resources() ClericResources { return r.resources }

// RechargeRoll returns the recharge dice rolled for a granted spell (2d4 for Wish), as a new slice, and false when none were.
func (r UseDivineInterventionApplied) RechargeRoll() ([]int, bool) {
	return slices.Clone(r.roll), r.roll != nil
}

// Events returns a new slice holding the DivineInterventionUsed event.
func (r UseDivineInterventionApplied) Events() []Event { return slices.Clone(r.events) }

// UseDivineInterventionRejected is the outcome of a use a rule refused: the Character is unchanged and there are no
// events. For not_available, State is the state Divine Intervention is in. Construct it with a keyed literal.
type UseDivineInterventionRejected struct {
	Reason ClericReason
	State  DivineInterventionState
}

// UseDivineInterventionNotACleric is the outcome for a character with no Cleric levels.
type UseDivineInterventionNotACleric struct{}

func (UseDivineInterventionApplied) isUseDivineInterventionResult()    {}
func (UseDivineInterventionRejected) isUseDivineInterventionResult()   {}
func (UseDivineInterventionNotACleric) isUseDivineInterventionResult() {}

// UseDivineIntervention is the use case that uses a Cleric's Divine Intervention and Greater Divine Intervention (SRD 5.2.1,
// "Level 10: Divine Intervention", "Level 20: Greater Divine Intervention", p. 38). It checks the entitlement, rolls the recharge
// dice through the dice port only for a granted spell and emits the srd5e.character.feature_used signals. It holds no rules
// logic. Create it with NewUseDivineIntervention; the zero value is not usable. It is safe for concurrent use only if its roller,
// telemetry and clock are.
type UseDivineIntervention struct {
	roller Roller
	sig    featureSignals
	set    bool
}

// NewUseDivineIntervention returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewUseDivineIntervention(roller Roller, opts ...Option) (UseDivineIntervention, error) {
	if roller == nil {
		return UseDivineIntervention{}, invalidInput("use Divine Intervention without a dice roller")
	}
	return UseDivineIntervention{roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run uses Divine Intervention for cmd.Character. The first reason that holds is reported: below_level_10, not_available,
// not_cleric_spell, spell_level_too_high, reaction_casting_time. A spell an unlocked grant names (Wish at level 20) is exempt
// from the list and level 5 checks and rolls the grant's dice, which start the recharge.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or the zero
// spell, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns an error matching
// srderr.ErrInvalidInput for a casting time the spell does not list, and a *srderr.PortError (error type "dice_roller_error") with
// a nil result when the roller fails, panics or breaks its contract; the Character is unchanged and the failure is recorded once
// and never logged. A refused use is a result.
func (u UseDivineIntervention) Run(ctx context.Context, cmd UseDivineInterventionCommand) (UseDivineInterventionResult, error) {
	if !u.set || cmd.Character.IsZero() || cmd.Spell.ID() == "" {
		return nil, invalidInput("use Divine Intervention: zero use case, character or spell")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseDivineIntervention, ValueClassCleric, ValueFeatureDivineIntervention)
	defer run.end(ctx)
	level, ok := clericLevelOf(cmd.Character)
	if !ok {
		run.rejected(ctx, ClericNotACleric.String())
		return UseDivineInterventionNotACleric{}, nil
	}
	resources, err := readClericResources(cmd.Character, 0)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	verdict, err := (DivineIntervention{}).Check(level, resources.DivineIntervention(), cmd.Spell, cmd.CastingTime, cmd.Rules)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if verdict.Reason != 0 {
		run.rejected(ctx, verdict.Reason.String())
		return UseDivineInterventionRejected{Reason: verdict.Reason, State: resources.DivineIntervention()}, nil
	}
	var rolls []int
	resources.intervention = DivineInterventionUnavailableUntilLongRest{}
	togo := 0
	if verdict.Granted {
		if rolls, err = rollDiceValues(u.roller, verdict.Grant.diceCount, verdict.Grant.die); err == nil {
			var recharging DivineInterventionRecharging
			if recharging, err = (DivineIntervention{}).RechargeAfterGrant(verdict.Grant, rolls); err == nil {
				resources.intervention, togo = recharging, recharging.LongRestsRemaining
			}
		}
		if err != nil {
			if et := srderr.ErrorType(err); et != "" {
				run.portFailure(ctx, err, et)
			} else {
				run.commandError(err)
			}
			return nil, err
		}
	}
	next, err := WithFeatureState(cmd.Character, ClericResourcesKey(), resources)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return UseDivineInterventionApplied{character: next, resources: resources, roll: rolls,
		cast:   DivineInterventionCast{Spell: cmd.Spell.ID(), CastingTime: verdict.CastingTime},
		events: []Event{DivineInterventionUsed{id: next.ID(), spell: cmd.Spell.ID(), longRestsToGo: togo}}}, nil
}
