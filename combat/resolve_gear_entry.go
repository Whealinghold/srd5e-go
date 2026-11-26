package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// ResolveGearEntryInput holds the values NewResolveGearEntryCommand validates. Build it with a
// keyed literal.
type ResolveGearEntryInput struct {
	// Gears and Ledger are the deployed gear and the entries made so far.
	Gears  DeployedGears
	Ledger EntryLedger
	// GearID is the gear the creature sets off.
	GearID DeployedGearID
	// Creature entered the area, or ended its turn there, and Trigger says which.
	Creature CombatantID
	Trigger  AreaTrigger
	// Stamp is the turn in progress, whoever's it is.
	Stamp TurnStamp
	// HoldID is the id of the hold a Hunting Trap begins on a creature it catches; it is read
	// for a failed Hunting Trap save only, but never blank.
	HoldID HoldID
	// Attempt is how the creature makes the Dexterity saving throw, with its modifiers and
	// circumstances. It is ignored for Oil, which allows no save, but never nil.
	Attempt d20.SaveAttempt
}

// ResolveGearEntryCommand is the immutable input of ResolveGearEntry.Run. Build it with
// NewResolveGearEntryCommand; the zero value is invalid.
type ResolveGearEntryCommand struct {
	in ResolveGearEntryInput
}

// NewResolveGearEntryCommand returns the command. The attempt's circumstances are copied.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a blank
// gear id, hold id or a zero creature, or GearProblemInvalidValue for a trigger that is not
// valid, the zero stamp or an attempt that is not a d20.RollAttempt, d20.ChooseToFail or
// d20.AutomaticFailure. A stamp that is not the creature's turn for an ending of a turn is
// answered by Run with ResolveGearEntryInvalid. It never panics. It returns GearProblemLimitExceeded for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewResolveGearEntryCommand(in ResolveGearEntryInput) (ResolveGearEntryCommand, error) {
	attempt, ok := checkSave(in.Attempt)
	switch {
	case in.GearID == "" || in.HoldID == "" || in.Creature.IsZero():
		return ResolveGearEntryCommand{}, gearInvalid(GearProblemBlankID)
	case !in.Trigger.Valid() || in.Stamp.IsZero() || !ok:
		return ResolveGearEntryCommand{}, gearInvalid(GearProblemInvalidValue)
	case attemptOverCap(in.Attempt) != nil:
		return ResolveGearEntryCommand{}, gearInvalid(GearProblemLimitExceeded)
	}
	in.Attempt = attempt
	return ResolveGearEntryCommand{in}, nil
}

// ResolveGearEntryResult is the closed set of outcomes of ResolveGearEntry.Run. Variants:
// ResolveGearEntrySaved, ResolveGearEntryFailed, ResolveGearEntryDamaged,
// ResolveGearEntryRejected, ResolveGearEntryInvalid.
type ResolveGearEntryResult interface{ isResolveGearEntryResult() }

// gearEntryState is what the three results that set gear off carry.
type gearEntryState struct {
	gears  DeployedGears
	ledger EntryLedger
	test   GearTest
	events []GearEvent
}

// ResolveGearEntrySaved: the creature passed the saving throw. A Hunting Trap is sprung all the
// same.
type ResolveGearEntrySaved struct{ gearEntryState }

// ResolveGearEntryFailed: the creature failed the saving throw and suffers the effects, which
// the caller applies.
type ResolveGearEntryFailed struct {
	gearEntryState
	outcome GearEntryFailed
}

// ResolveGearEntryDamaged: the creature takes damage with no saving throw (burning Oil).
type ResolveGearEntryDamaged struct {
	gearEntryState
	outcome GearEntryDamaged
}

// Gears returns the gear after the entry: a Hunting Trap that was set off is sprung.
func (s gearEntryState) Gears() DeployedGears { return s.gears }

// Ledger returns the ledger with the entry recorded.
func (s gearEntryState) Ledger() EntryLedger { return s.ledger }

// Events returns a copy of the events: one HuntingTrapSprung for a Hunting Trap, else none.
func (s gearEntryState) Events() []GearEvent { return slices.Clone(s.events) }

// Test returns the saving throw as d20 resolved it; ok is false for Oil, which has none.
func (s gearEntryState) Test() (GearTest, bool) { return s.test, s.test != nil }

// Outcome returns GearEntrySaved.
func (ResolveGearEntrySaved) Outcome() GearEntryOutcome { return GearEntrySaved{} }

// Outcome returns the GearEntryFailed, whose Effects the caller applies.
func (r ResolveGearEntryFailed) Outcome() GearEntryOutcome { return r.outcome }

// Outcome returns the GearEntryDamaged, whose Effects the caller applies.
func (r ResolveGearEntryDamaged) Outcome() GearEntryOutcome { return r.outcome }

// ResolveGearEntryRejected: the entry sets nothing off and nothing was rolled or recorded.
type ResolveGearEntryRejected struct{ reason GearRejection }

// Reason returns why: unknownGear, triggerNotApplicable, oilNotBurning, trapAlreadySprung or
// alreadyTriggeredThisTurn.
func (r ResolveGearEntryRejected) Reason() GearRejection { return r.reason }

// ResolveGearEntryInvalid: the command holds a value the rules cannot use (an ending of a turn
// stamped with another creature's turn).
type ResolveGearEntryInvalid struct{ problem GearProblem }

// Problem returns what is wrong.
func (r ResolveGearEntryInvalid) Problem() GearProblem { return r.problem }

func (ResolveGearEntrySaved) isResolveGearEntryResult()    {}
func (ResolveGearEntryFailed) isResolveGearEntryResult()   {}
func (ResolveGearEntryDamaged) isResolveGearEntryResult()  {}
func (ResolveGearEntryRejected) isResolveGearEntryResult() {}
func (ResolveGearEntryInvalid) isResolveGearEntryResult()  {}

// ResolveGearEntry is the use case of a creature setting deployed gear off: it makes the
// saving throw of Caltrops, Ball Bearings and a Hunting Trap (and the Hunting Trap's 1d4 on a
// failure) through the dice port, or finds the damage of burning Oil, and returns the effects
// without applying them (SRD 5.2.1, pp. 96-98). It composes the pure d20 functions and never
// calls d20.Resolver or another use case, so no srd5e.d20_test signal fires. Create it with
// NewResolveGearEntry; the zero value is not usable.
type ResolveGearEntry struct {
	gearUse
	flow attackFlow
}

// NewResolveGearEntry returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveGearEntry(roller Roller, opts ...Option) (ResolveGearEntry, error) {
	if roller == nil {
		return ResolveGearEntry{}, invalidInput("gear entry without a dice roller")
	}
	return ResolveGearEntry{newGearUse(opts), attackFlow{roller}}, nil
}

// Run assesses the entry and, when it is not refused, makes the saving throw: only its d20s,
// then for a failed Hunting Trap save one 1d4. A passed or failed save is a normal result. The
// entry is recorded in the ledger and a Hunting Trap springs either way. A hold the trap
// catches is in the effects, for the caller to begin.
//
// It records the shared signals and srd5e.combat.gear.entries. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case or command, and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression; the failure is recorded once and never logged.
func (u ResolveGearEntry) Run(ctx context.Context, cmd ResolveGearEntryCommand) (ResolveGearEntryResult, error) {
	if cmd.in.GearID == "" || u.flow.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveGearEntry, spanResolveGearEntry}
	gear, found := cmd.in.Gears.Find(cmd.in.GearID)
	res, err := run(ctx, u.in, op, func(context.Context) (ResolveGearEntryResult, error) {
		if !found {
			return ResolveGearEntryRejected{GearRejectionUnknownGear}, nil
		}
		return u.resolve(cmd.in, gear)
	}, func(r ResolveGearEntryResult) description {
		switch v := r.(type) {
		case ResolveGearEntrySaved:
			return description{Class: resultApplied, Name: valueResultSaved, Events: widen(v.events)}
		case ResolveGearEntryFailed:
			return description{Class: resultApplied, Name: valueResultFailed, Events: widen(v.events)}
		case ResolveGearEntryDamaged:
			return description{Class: resultApplied, Name: valueResultDamaged, Events: widen(v.events)}
		case ResolveGearEntryRejected:
			return description{Class: resultRefused, Name: v.reason.String()}
		case ResolveGearEntryInvalid:
			return description{Class: resultInvalid, Name: v.problem.String()}
		default:
			panic(fmt.Errorf("combat: unreachable gear entry result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if found {
		if entry := entryValue(res); entry != "" {
			u.recordEntry(ctx, gear.kind, entry)
		}
	}
	return res, nil
}

// entryValue returns the value of srd5e.combat.gear.entry for a result, "" for none.
func entryValue(r ResolveGearEntryResult) string {
	switch v := r.(type) {
	case ResolveGearEntrySaved:
		return valueGearEntrySavePassed
	case ResolveGearEntryFailed:
		return valueGearEntrySaveFailed
	case ResolveGearEntryDamaged:
		return valueGearEntryDamaged
	case ResolveGearEntryRejected:
		if v.reason == GearRejectionAlreadyTriggeredThisTurn {
			return valueGearEntryAlreadyTriggered
		}
		if v.reason == GearRejectionOilNotBurning || v.reason == GearRejectionTrapAlreadySprung {
			return valueGearEntryNotActive
		}
		if v.reason == GearRejectionTriggerNotApplicable {
			return valueGearEntryNotApplicable
		}
	}
	return ""
}

// resolve is the work of Run for gear that exists.
func (u ResolveGearEntry) resolve(in ResolveGearEntryInput, gear DeployedGear) (ResolveGearEntryResult, error) {
	rules := AreaEntryRules{}
	assessment, err := rules.Assess(gear, in.Creature, in.Trigger, in.Stamp, in.Ledger)
	if p, ok := gearProblem(err); ok {
		return ResolveGearEntryInvalid{p}, nil
	}
	switch a := assessment.(type) {
	case GearEntryRefused:
		return ResolveGearEntryRejected(a), nil
	case GearEntryDamageDue:
		return ResolveGearEntryDamaged{gearEntryState{gears: in.Gears, ledger: a.ledger}, a.Outcome()}, nil
	case GearEntryRollRequired:
		test, err := u.flow.gearTest(d20.SavingThrow, in.Attempt, a.dc)
		if err != nil {
			return nil, err
		}
		saved := test.Outcome() == d20.Success
		damage := 0
		if !saved && gear.kind == DeployedGearKindHuntingTrap {
			if damage, err = u.flow.rollTrapDamage(); err != nil {
				return nil, err
			}
		}
		state := gearEntryState{gears: in.Gears, ledger: a.ledger, test: test}
		if gear.kind == DeployedGearKindHuntingTrap {
			sprung, err := in.Gears.Spring(gear.id, in.Creature)
			if err != nil {
				return nil, err
			}
			applied := sprung.(DeployedGearsApplied)
			state.gears, state.events = applied.gears, applied.events
		}
		outcome := gear.entryOutcome(in.Creature, in.HoldID, saved, damage)
		if failed, ok := outcome.(GearEntryFailed); ok {
			return ResolveGearEntryFailed{state, failed}, nil
		}
		return ResolveGearEntrySaved{state}, nil
	default:
		panic(unreachableError("gear entry assessment", assessment))
	}
}
