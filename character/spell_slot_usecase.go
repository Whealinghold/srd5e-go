package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ExpendSpellSlotCommand is the input of ExpendSpellSlot.Do. Construct it with a
// keyed literal. Do does not alter it.
type ExpendSpellSlotCommand struct {
	State SpellcastingState
	Pool  SpellSlotPool
	Level SlotLevel
}

// ExpendSpellSlotResult is the closed set of outcomes of ExpendSpellSlot.Do.
// Variants: ExpendSpellSlotExpended, ExpendSpellSlotRejected.
type ExpendSpellSlotResult interface{ isExpendSpellSlotResult() }

// ExpendSpellSlotExpended: a slot was expended.
type ExpendSpellSlotExpended struct {
	state SpellcastingState
	event SpellSlotExpended
}

// State returns the new state.
func (r ExpendSpellSlotExpended) State() SpellcastingState { return r.state }

// Event returns the SpellSlotExpended event.
func (r ExpendSpellSlotExpended) Event() SpellSlotExpended { return r.event }

// ExpendSpellSlotRejected: no slot was available; the state is unchanged.
type ExpendSpellSlotRejected struct{ reason SlotOperationRejection }

// Reason returns SlotOperationNoSlotAvailable.
func (r ExpendSpellSlotRejected) Reason() SlotOperationRejection { return r.reason }

func (ExpendSpellSlotExpended) isExpendSpellSlotResult() {}
func (ExpendSpellSlotRejected) isExpendSpellSlotResult() {}

// RegainSpellSlotCommand is the input of RegainSpellSlot.Do. Construct it with a
// keyed literal. Do does not alter it.
type RegainSpellSlotCommand struct {
	State SpellcastingState
	Pool  SpellSlotPool
	Level SlotLevel
	Count int
}

// RegainSpellSlotResult is the closed set of outcomes of RegainSpellSlot.Do.
// Variants: RegainSpellSlotRegained, RegainSpellSlotRejected.
type RegainSpellSlotResult interface{ isRegainSpellSlotResult() }

// RegainSpellSlotRegained: slots were restored.
type RegainSpellSlotRegained struct {
	state SpellcastingState
	event SpellSlotsRestored
}

// State returns the new state.
func (r RegainSpellSlotRegained) State() SpellcastingState { return r.state }

// Event returns the SpellSlotsRestored event.
func (r RegainSpellSlotRegained) Event() SpellSlotsRestored { return r.event }

// RegainSpellSlotRejected: no slot of that level was expended; the state is unchanged.
type RegainSpellSlotRejected struct{ reason SlotOperationRejection }

// Reason returns SlotOperationNoExpendedSlot.
func (r RegainSpellSlotRejected) Reason() SlotOperationRejection { return r.reason }

func (RegainSpellSlotRegained) isRegainSpellSlotResult() {}
func (RegainSpellSlotRejected) isRegainSpellSlotResult() {}

// RestoreSpellSlotsCommand is the input of RestoreSpellSlots.Do. Construct it with
// a keyed literal. Do does not alter it.
type RestoreSpellSlotsCommand struct {
	State SpellcastingState
	Rest  RestKind
}

// RestoreSpellSlotsResult is the closed set of outcomes of RestoreSpellSlots.Do.
// Variants: RestoreSpellSlotsRestored.
type RestoreSpellSlotsResult interface{ isRestoreSpellSlotsResult() }

// RestoreSpellSlotsRestored: the state after the rest.
type RestoreSpellSlotsRestored struct {
	state  SpellcastingState
	events []SpellcastingEvent
}

// State returns the new state.
func (r RestoreSpellSlotsRestored) State() SpellcastingState { return r.state }

// Events returns a new slice of the events; none when nothing was expended.
func (r RestoreSpellSlotsRestored) Events() []SpellcastingEvent { return slices.Clone(r.events) }

func (RestoreSpellSlotsRestored) isRestoreSpellSlotsResult() {}

// AdvanceSpellcastingCommand is the input of AdvanceSpellcasting.Do. Construct it
// with a keyed literal. Do does not alter it.
type AdvanceSpellcastingCommand struct {
	State  SpellcastingState
	Levels ClassLevels
}

// AdvanceSpellcastingResult is the closed set of outcomes of
// AdvanceSpellcasting.Do. Variants: AdvanceSpellcastingAdvanced,
// AdvanceSpellcastingRejected.
type AdvanceSpellcastingResult interface{ isAdvanceSpellcastingResult() }

// AdvanceSpellcastingAdvanced: the state follows the new class levels.
type AdvanceSpellcastingAdvanced struct {
	state            SpellcastingState
	spellsToChoose   []ChoiceCount
	cantripsToChoose []ChoiceCount
	events           []SpellcastingEvent
}

// State returns the new state.
func (r AdvanceSpellcastingAdvanced) State() SpellcastingState { return r.state }

// SpellsToChoose returns a new slice of the level 1+ spells each held class may
// still choose.
func (r AdvanceSpellcastingAdvanced) SpellsToChoose() []ChoiceCount {
	return slices.Clone(r.spellsToChoose)
}

// CantripsToChoose returns a new slice of the cantrips each held class may still choose.
func (r AdvanceSpellcastingAdvanced) CantripsToChoose() []ChoiceCount {
	return slices.Clone(r.cantripsToChoose)
}

// Events returns a new slice of the events; a level change raises none today.
func (r AdvanceSpellcastingAdvanced) Events() []SpellcastingEvent { return slices.Clone(r.events) }

// AdvanceSpellcastingRejected: the new levels were refused; the state is unchanged.
type AdvanceSpellcastingRejected struct{ reason SpellcastingAdvanceRejection }

// Reason returns the rejection.
func (r AdvanceSpellcastingRejected) Reason() SpellcastingAdvanceRejection { return r.reason }

func (AdvanceSpellcastingAdvanced) isAdvanceSpellcastingResult() {}
func (AdvanceSpellcastingRejected) isAdvanceSpellcastingResult() {}

// ExpendSpellSlot is the use case that expends one slot from a pool. It wraps
// SpellcastingState.ExpendSlot and applies no casting rule. Create it with
// NewExpendSpellSlot; the zero value is not usable. It is safe for concurrent use
// only if its telemetry and clock are.
type ExpendSpellSlot struct {
	sig spellcastingSignals
	set bool
}

// NewExpendSpellSlot returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored.
func NewExpendSpellSlot(opts ...Option) ExpendSpellSlot {
	return ExpendSpellSlot{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do expends the slot. A pool with no such slot gives ExpendSpellSlotRejected, not
// an error.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state, an invalid pool or a zero level, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled.
func (u ExpendSpellSlot) Do(ctx context.Context, cmd ExpendSpellSlotCommand) (ExpendSpellSlotResult, error) {
	if !u.set || cmd.State.IsZero() || !cmd.Pool.Valid() || cmd.Level.IsZero() {
		return nil, invalidInput("expend spell slot: zero use case or state, invalid pool or zero level")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pool := telemetry.String(AttrPool, cmd.Pool.String())
	ctx, run := u.sig.begin(ctx, SpanExpendSpellSlot, ValueOperationExpendSpellSlot, pool)
	defer run.end(ctx)
	switch r := cmd.State.ExpendSlot(cmd.Pool, cmd.Level).(type) {
	case SlotExpenditureExpended:
		u.sig.slotLevel.Record(ctx, float64(cmd.Level.Value()), pool)
		run.finish(ctx, ValueResultExpended, "")
		return ExpendSpellSlotExpended(r), nil
	case SlotExpenditureRejected:
		run.finish(ctx, ValueResultRejected, r.reason.String())
		return ExpendSpellSlotRejected(r), nil
	default:
		panic(invalidInput("expend spell slot: unreachable outcome"))
	}
}

// RegainSpellSlot is the use case that restores expended slots of one level in a
// pool. It wraps SpellcastingState.RegainSlot. Create it with NewRegainSpellSlot;
// the zero value is not usable. It is safe for concurrent use only if its
// telemetry and clock are.
type RegainSpellSlot struct {
	sig spellcastingSignals
	set bool
}

// NewRegainSpellSlot returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored.
func NewRegainSpellSlot(opts ...Option) RegainSpellSlot {
	return RegainSpellSlot{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do restores up to cmd.Count slots, never above the maximum. A level with none
// expended gives RegainSpellSlotRejected, not an error.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state, an invalid pool, a zero level or a count below 1,
// and ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u RegainSpellSlot) Do(ctx context.Context, cmd RegainSpellSlotCommand) (RegainSpellSlotResult, error) {
	if !u.set || cmd.State.IsZero() || !cmd.Pool.Valid() || cmd.Level.IsZero() || cmd.Count < 1 {
		return nil, invalidInput("regain spell slot: zero use case or state, invalid pool, zero level or count below 1")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanRegainSpellSlot, ValueOperationRegainSpellSlot, telemetry.String(AttrPool, cmd.Pool.String()))
	defer run.end(ctx)
	switch r := cmd.State.RegainSlot(cmd.Pool, cmd.Level, cmd.Count).(type) {
	case SlotRecoveryRegained:
		run.finish(ctx, ValueResultRegained, "")
		return RegainSpellSlotRegained(r), nil
	case SlotRecoveryRejected:
		run.finish(ctx, ValueResultRejected, r.reason.String())
		return RegainSpellSlotRejected(r), nil
	default:
		panic(invalidInput("regain spell slot: unreachable outcome"))
	}
}

// RestoreSpellSlots is the use case that applies a finished rest to the slots: a
// Long Rest restores both pools, a Short Rest only Pact Magic. It wraps
// SpellcastingState.RestoreSlots. Create it with NewRestoreSpellSlots; the zero
// value is not usable. It is safe for concurrent use only if its telemetry and
// clock are.
type RestoreSpellSlots struct {
	sig spellcastingSignals
	set bool
}

// NewRestoreSpellSlots returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewRestoreSpellSlots(opts ...Option) RestoreSpellSlots {
	return RestoreSpellSlots{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do restores the slots. A rest with nothing expended succeeds with no events.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state or an invalid rest kind, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled.
func (u RestoreSpellSlots) Do(ctx context.Context, cmd RestoreSpellSlotsCommand) (RestoreSpellSlotsResult, error) {
	if !u.set || cmd.State.IsZero() || !cmd.Rest.Valid() {
		return nil, invalidInput("restore spell slots: zero use case or state, or invalid rest kind")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanRestoreSpellSlots, ValueOperationRestoreSpellSlots, telemetry.String(AttrRestKind, cmd.Rest.String()))
	defer run.end(ctx)
	state, events := cmd.State.RestoreSlots(cmd.Rest)
	run.record(ctx, u.sig.restored, false, ValueResultRestored, "")
	return RestoreSpellSlotsRestored{state: state, events: events}, nil
}

// AdvanceSpellcasting is the use case that follows a character to new class
// levels. It wraps SpellcastingState.Advance. Create it with
// NewAdvanceSpellcasting; the zero value is not usable. It is safe for concurrent
// use only if its telemetry and clock are.
type AdvanceSpellcasting struct {
	sig spellcastingSignals
	set bool
}

// NewAdvanceSpellcasting returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewAdvanceSpellcasting(opts ...Option) AdvanceSpellcasting {
	return AdvanceSpellcasting{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do advances the state. New levels the rules refuse give
// AdvanceSpellcastingRejected, not an error.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state or zero class levels, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled.
func (u AdvanceSpellcasting) Do(ctx context.Context, cmd AdvanceSpellcastingCommand) (AdvanceSpellcastingResult, error) {
	if !u.set || cmd.State.IsZero() || cmd.Levels.IsZero() {
		return nil, invalidInput("advance spellcasting: zero use case, state or class levels")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanAdvanceSpellcasting, ValueOperationAdvanceSpellcasting)
	defer run.end(ctx)
	switch r := cmd.State.Advance(cmd.Levels).(type) {
	case SpellcastingLevelChanged:
		run.finish(ctx, ValueResultAdvanced, "")
		return AdvanceSpellcastingAdvanced(r), nil
	case SpellcastingLevelChangeRejected:
		run.finish(ctx, ValueResultRejected, r.reason.String())
		return AdvanceSpellcastingRejected(r), nil
	default:
		panic(invalidInput("advance spellcasting: unreachable outcome"))
	}
}
