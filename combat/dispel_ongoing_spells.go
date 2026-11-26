package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// maxOngoingSpellsPerTarget is the cap of active effects per creature (plan P10): a target holding more
// ongoing spells than this is refused before any die is rolled.
const maxOngoingSpellsPerTarget = 64

// DispelOngoingSpellsCommand is the input of DispelOngoingSpells.Run. Construct it with a keyed literal.
// Run copies the slices and changes nothing in the command.
type DispelOngoingSpellsCommand struct {
	// Registry is the read model of the live spells, built by the host from ActiveEffects.
	Registry OngoingSpellRegistry
	// Target is the creature, object or magical effect the player chose (SRD 5.2.1, "Dispel Magic", p. 124).
	Target DispelTarget
	// CastLevel is the level of the slot Dispel Magic was cast with, 3 to 9.
	CastLevel character.SlotLevel
	// AbilityModifier is the spellcasting ability modifier. The Proficiency Bonus is never added (FR-020).
	AbilityModifier ability.Modifier
	// Modifiers carries other bonuses to the check: only its Others apply (FR-020).
	Modifiers d20.RollModifiers
	// Circumstances give the roll mode of every check, resolved once per check; none means a normal roll.
	Circumstances []modifier.RollCircumstance
}

// DispelOngoingSpellsResult is the closed set of outcomes of DispelOngoingSpells.Run. Variants:
// OngoingSpellsDispelled, DispelOngoingSpellsInvalid.
type DispelOngoingSpellsResult interface{ isDispelOngoingSpellsResult() }

// DispelRolledCheck is the ability check rolled for one spell: the effect it was for and the resolved check.
type DispelRolledCheck struct {
	effect OngoingEffectID
	check  d20.CheckResult
}

// Effect returns the id of the spell the check was for.
func (c DispelRolledCheck) Effect() OngoingEffectID { return c.effect }

// Check returns the ability check d20.ResolveCheck resolved.
func (c DispelRolledCheck) Check() d20.CheckResult { return c.check }

// OngoingSpellsDispelled is a completed Dispel Magic: the plan, the checks rolled in plan order and the
// conclusion (the new registry, the spells that ended and survived, and the events). The host ends the
// ended effects (EndActiveEffects with RemovedByEffect), removes their conditions and, once per caster, calls
// BreakConcentration (FR-025, FR-026). With nothing ended the registry is equal to the input. Not
// comparable with ==.
type OngoingSpellsDispelled struct {
	plan    DispelMagicPlan
	checks  []DispelRolledCheck
	applied DispelMagicApplied
}

// Plan returns the plan Dispel Magic made.
func (r OngoingSpellsDispelled) Plan() DispelMagicPlan { return r.plan }

// Checks returns the checks rolled, in plan order, as a new slice.
func (r OngoingSpellsDispelled) Checks() []DispelRolledCheck { return slices.Clone(r.checks) }

// Conclusion returns the conclusion of the checks.
func (r OngoingSpellsDispelled) Conclusion() DispelMagicApplied { return r.applied }

// DispelOngoingSpellsInvalid is a call the use case could not conclude: Problem says why, either a
// conclusion problem (DispelPlanMismatch and the others cannot happen when the use case rolls its own
// checks) or DispelLimitExceeded, a target that holds more than 64 ongoing spells. No die was rolled for
// DispelLimitExceeded.
type DispelOngoingSpellsInvalid struct{ problem DispelMagicProblem }

// Problem returns why the call was invalid.
func (r DispelOngoingSpellsInvalid) Problem() DispelMagicProblem { return r.problem }

func (OngoingSpellsDispelled) isDispelOngoingSpellsResult()     {}
func (DispelOngoingSpellsInvalid) isDispelOngoingSpellsResult() {}

// DispelOngoingSpells is the use case of Dispel Magic (SRD 5.2.1, "Dispel Magic", p. 124): it plans the
// spells on the target, rolls one ability check through the dice port for each spell that needs one,
// resolves it with d20.ResolveCheck and concludes. It ends nothing itself (the host passes the ended ids to
// EndActiveEffects) and calls no other use case. Create it with NewDispelOngoingSpells; the zero value is
// not usable. It is safe for concurrent use only if its roller, telemetry and clock are.
type DispelOngoingSpells struct {
	in   instrumenter
	flow attackFlow
}

// NewDispelOngoingSpells returns the use case rolling through roller. Options: WithTelemetry, WithClock.
// It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewDispelOngoingSpells(roller Roller, opts ...Option) (DispelOngoingSpells, error) {
	if roller == nil {
		return DispelOngoingSpells{}, invalidInput("dispel magic without a dice roller")
	}
	o := newOptions(opts)
	return DispelOngoingSpells{newInstrumenter(o.tel, o.clock), attackFlow{roller}}, nil
}

// Run dispels the spells of cmd.Target (dispel-and-counterspell.md FR-011 to FR-028). A target holding more
// than 64 spells gives DispelOngoingSpellsInvalid with DispelLimitExceeded before any die; otherwise one
// d20 (or two, with Advantage or Disadvantage) is rolled for each DispelNeedsCheck in plan order.
//
// It records the shared signals under the operation dispel_ongoing_spells with the result completed (a
// spell ended), dispel_nothing_ended (applied), or limit_exceeded or dispel_invalid_checks (invalid), and
// one event per OngoingSpellEnded. It never logs. It returns an error matching srderr.ErrInvalidInput,
// recorded on the span only, for a cast level below 3 or a target without an id; an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case; and ctx.Err() unwrapped, before any
// signal, for a cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or answers another expression; the failure is recorded once and no
// partial result is returned.
func (u DispelOngoingSpells) Run(ctx context.Context, cmd DispelOngoingSpellsCommand) (DispelOngoingSpellsResult, error) {
	if u.flow.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd.Circumstances = slices.Clone(cmd.Circumstances)
	res, err := run(ctx, u.in, combatOperation{valueOperationDispelOngoingSpells, spanDispelOngoingSpells},
		func(context.Context) (DispelOngoingSpellsResult, error) { return u.dispel(cmd) },
		describeDispel)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// dispel is the work of Run: plan, roll, conclude.
func (u DispelOngoingSpells) dispel(cmd DispelOngoingSpellsCommand) (DispelOngoingSpellsResult, error) {
	plan, err := DispelMagic{}.Plan(cmd.Registry, cmd.Target, cmd.CastLevel, cmd.AbilityModifier, cmd.Modifiers)
	if err != nil {
		return nil, err
	}
	if len(plan.steps) > maxOngoingSpellsPerTarget {
		return DispelOngoingSpellsInvalid{DispelLimitExceeded}, nil
	}
	mode := modifier.Resolve(cmd.Circumstances...)
	checks := make(map[OngoingEffectID]d20.CheckResult)
	var rolled []DispelRolledCheck
	for _, step := range plan.steps {
		need, ok := step.(DispelNeedsCheck)
		if !ok {
			continue
		}
		roll, err := u.flow.rollD20(mode)
		if err != nil {
			return nil, err
		}
		// FR-020: the spellcasting ability modifier and the other bonuses; the Proficiency Bonus is never added.
		mods := d20.NewRollModifiers(modifier.New(need.abilityModifier.Value())).WithOthers(need.modifiers.Others()...)
		check, err := d20.ResolveCheck(d20.AbilityCheck, roll, mods, need.dc)
		if err != nil {
			return nil, err
		}
		id := need.spell.Effect().ID()
		checks[id] = check
		rolled = append(rolled, DispelRolledCheck{id, check})
	}
	switch c := (DispelMagic{}).Conclude(cmd.Registry, plan, checks).(type) {
	case DispelMagicApplied:
		return OngoingSpellsDispelled{plan, rolled, c}, nil
	case DispelMagicInvalid:
		return DispelOngoingSpellsInvalid(c), nil
	default:
		panic(unreachableError("dispel magic conclusion", c))
	}
}

// describeDispel classifies a result (dispel-and-counterspell.md section 7).
func describeDispel(r DispelOngoingSpellsResult) description {
	switch v := r.(type) {
	case OngoingSpellsDispelled:
		name := valueResultDispelNothingEnded
		if len(v.applied.ended) > 0 {
			name = valueResultCompleted
		}
		events := make([]Event, len(v.applied.events))
		for i, e := range v.applied.events {
			events[i] = e
		}
		return description{Class: resultApplied, Name: name, Events: events}
	case DispelOngoingSpellsInvalid:
		name := valueResultDispelInvalidChecks
		if v.problem == DispelLimitExceeded {
			name = valueResultLimitExceeded
		}
		return description{Class: resultInvalid, Name: name}
	default:
		panic(unreachableError("dispel ongoing spells result", r))
	}
}
