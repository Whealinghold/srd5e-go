package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
)

// CounterCastSpellCommand is the input of CounterCastSpell.Run. Construct it with a keyed literal.
type CounterCastSpellCommand struct {
	// Cast is the cast being countered, built with NewCastObservation.
	Cast CastObservation
	// Counterspeller is the creature casting Counterspell.
	Counterspeller CombatantID
	// DistanceFeet is the distance from the counterspeller to the casting creature, at least 0.
	DistanceFeet int
	// SeesCaster is whether the counterspeller sees the casting creature.
	SeesCaster bool
	// SaveDC is the counterspeller's spell save DC.
	SaveDC character.SpellSaveDC
	// CasterSave is how the casting creature makes its Constitution saving throw: a d20.RollAttempt, or
	// d20.ChooseToFail or d20.AutomaticFailure (no die). Its modifiers are the host's.
	CasterSave d20.SaveAttempt
}

// CounterCastSpellResult is the closed set of outcomes of CounterCastSpell.Run. Variants:
// CounterCastSpellConcluded, CounterCastSpellIneligible.
type CounterCastSpellResult interface{ isCounterCastSpellResult() }

// CounterCastSpellConcluded is a Counterspell that was eligible and whose save was made: Save is the save
// (rolled or forced) and Result is CounterspellCountered, CounterspellResistedResult or CounterspellInvalid.
// On CounterspellCountered the host discards the uncommitted cast (FR-034).
type CounterCastSpellConcluded struct {
	save   CounterspellSave
	result CounterspellResult
}

// Save returns the casting creature's save.
func (r CounterCastSpellConcluded) Save() CounterspellSave { return r.save }

// Result returns the conclusion of the save.
func (r CounterCastSpellConcluded) Result() CounterspellResult { return r.result }

// CounterCastSpellIneligible is a cast Counterspell cannot counter; no die was rolled and nothing changed.
type CounterCastSpellIneligible struct{ ineligible CounterspellIneligible }

// Refusal returns why the cast cannot be countered.
func (r CounterCastSpellIneligible) Refusal() CounterspellRefusal { return r.ineligible.refusal }

func (CounterCastSpellConcluded) isCounterCastSpellResult()  {}
func (CounterCastSpellIneligible) isCounterCastSpellResult() {}

// CounterCastSpell is the use case of Counterspell (SRD 5.2.1, "Counterspell", p. 120): it assesses the cast,
// rolls the casting creature's Constitution save through the dice port with d20.ResolveCheck, and
// concludes. It casts nothing, expends nothing and calls no other use case; casting Counterspell itself
// (FR-038) and the order of a chain of Counterspells (FR-039) are the host's. Create it with
// NewCounterCastSpell; the zero value is not usable. It is safe for concurrent use only if its roller,
// telemetry and clock are.
type CounterCastSpell struct {
	in   instrumenter
	flow attackFlow
}

// NewCounterCastSpell returns the use case rolling through roller. Options: WithTelemetry, WithClock. It
// returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewCounterCastSpell(roller Roller, opts ...Option) (CounterCastSpell, error) {
	if roller == nil {
		return CounterCastSpell{}, invalidInput("counterspell without a dice roller")
	}
	o := newOptions(opts)
	return CounterCastSpell{newInstrumenter(o.tel, o.clock), attackFlow{roller}}, nil
}

// Run counters the cast of cmd (dispel-and-counterspell.md FR-031 to FR-042). An ineligible cast gives
// CounterCastSpellIneligible without a die; otherwise the save is rolled (one d20, two with Advantage or
// Disadvantage; none for a forced failure) and concluded.
//
// It records the shared signals under the operation counter_cast_spell with the result completed (countered),
// counterspell_resisted (applied), counterspell_ineligible (refused) or counterspell_invalid_save (invalid),
// and the event of a countered or resisted cast. It never logs. It returns an error matching
// srderr.ErrInvalidInput, recorded on the span only, for a zero cast, a zero counterspeller, a negative
// distance or a save attempt that is not a roll or a forced failure; an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case; and ctx.Err() unwrapped, before any
// signal, for a cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or answers another expression; the failure is recorded once.
func (u CounterCastSpell) Run(ctx context.Context, cmd CounterCastSpellCommand) (CounterCastSpellResult, error) {
	if u.flow.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationCounterCastSpell, spanCounterCastSpell},
		func(context.Context) (CounterCastSpellResult, error) { return u.counter(cmd) }, describeCounterCast)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// counter is the work of Run: assess, roll the save, conclude.
func (u CounterCastSpell) counter(cmd CounterCastSpellCommand) (CounterCastSpellResult, error) {
	if cmd.Cast.caster.IsZero() {
		return nil, invalidInput("counterspell needs a cast")
	}
	assessment, err := Counterspell{}.Assess(cmd.Cast, cmd.Counterspeller, cmd.DistanceFeet, cmd.SeesCaster, cmd.SaveDC)
	if err != nil {
		return nil, err
	}
	var eligible CounterspellEligible
	switch a := assessment.(type) {
	case CounterspellIneligible:
		return CounterCastSpellIneligible{a}, nil
	case CounterspellEligible:
		eligible = a
	default:
		panic(unreachableError("counterspell assessment", a))
	}
	if err := attemptOverCap(cmd.CasterSave); err != nil {
		return nil, err
	}
	attempt, ok := checkSave(cmd.CasterSave)
	if !ok {
		return nil, invalidInput("the casting creature's save must be a roll or a forced failure")
	}
	save, err := u.flow.resolveSave(attempt, eligible.dc.DifficultyClass())
	if err != nil {
		return nil, err
	}
	var s CounterspellSave
	switch v := save.(type) {
	case UnarmedStrikeSaveRolled:
		s = CounterspellRolledSave{v.result}
	case UnarmedStrikeSaveForced:
		s = CounterspellForcedFailure{v.result}
	default:
		panic(unreachableError("counterspell save", save))
	}
	return CounterCastSpellConcluded{s, Counterspell{}.Conclude(eligible, s)}, nil
}

// describeCounterCast classifies a result (dispel-and-counterspell.md section 7).
func describeCounterCast(r CounterCastSpellResult) description {
	switch v := r.(type) {
	case CounterCastSpellIneligible:
		return description{Class: resultRefused, Name: valueResultCounterspellIneligible}
	case CounterCastSpellConcluded:
		switch c := v.result.(type) {
		case CounterspellCountered:
			return description{Class: resultApplied, Name: valueResultCompleted, Events: []Event{c.event}}
		case CounterspellResistedResult:
			return description{Class: resultApplied, Name: valueResultCounterspellResisted, Events: []Event{c.event}}
		case CounterspellInvalid:
			return description{Class: resultInvalid, Name: valueResultCounterspellInvalidSave}
		default:
			panic(unreachableError("counterspell result", c))
		}
	default:
		panic(unreachableError("counter cast spell result", r))
	}
}
