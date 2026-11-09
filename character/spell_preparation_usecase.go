package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ChooseAdditionalSpellsCommand is the input of ChooseAdditionalSpells.Do. Construct
// it with a keyed literal. Do does not alter it.
type ChooseAdditionalSpellsCommand struct {
	State    SpellcastingState
	Class    SpellcastingClass
	Prepared []content.SpellID
	Cantrips []content.SpellID
	Spells   map[content.SpellID]content.Spell
}

// ChooseAdditionalSpellsResult is the closed set of outcomes of
// ChooseAdditionalSpells.Do. Variants: ChooseAdditionalSpellsChosen,
// ChooseAdditionalSpellsRejected, ChooseAdditionalSpellsContentMissing.
type ChooseAdditionalSpellsResult interface{ isChooseAdditionalSpellsResult() }

// ChooseAdditionalSpellsChosen: the additions were accepted.
type ChooseAdditionalSpellsChosen struct {
	state  SpellcastingState
	events []SpellcastingEvent
}

// State returns the new state.
func (r ChooseAdditionalSpellsChosen) State() SpellcastingState { return r.state }

// Events returns a new slice of the events; none for an empty addition.
func (r ChooseAdditionalSpellsChosen) Events() []SpellcastingEvent { return slices.Clone(r.events) }

// ChooseAdditionalSpellsRejected: a rule refused the additions; the state is unchanged.
type ChooseAdditionalSpellsRejected struct{ rejection PreparationRejection }

// Rejection returns every problem found.
func (r ChooseAdditionalSpellsRejected) Rejection() PreparationRejection { return r.rejection }

// ChooseAdditionalSpellsContentMissing: a spell named is absent from the spells
// supplied; the state is unchanged.
type ChooseAdditionalSpellsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the absent spells, in the order named, without repeats.
func (r ChooseAdditionalSpellsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChooseAdditionalSpellsChosen) isChooseAdditionalSpellsResult()         {}
func (ChooseAdditionalSpellsRejected) isChooseAdditionalSpellsResult()       {}
func (ChooseAdditionalSpellsContentMissing) isChooseAdditionalSpellsResult() {}

// ChangePreparedSpellsCommand is the input of ChangePreparedSpells.Do. Construct
// it with a keyed literal. Do does not alter it.
type ChangePreparedSpellsCommand struct {
	State   SpellcastingState
	Class   SpellcastingClass
	Trigger PreparationTrigger
	Change  PreparationChange
	Spells  map[content.SpellID]content.Spell
}

// ChangePreparedSpellsResult is the closed set of outcomes of
// ChangePreparedSpells.Do. Variants: ChangePreparedSpellsChanged,
// ChangePreparedSpellsRejected, ChangePreparedSpellsContentMissing.
type ChangePreparedSpellsResult interface{ isChangePreparedSpellsResult() }

// ChangePreparedSpellsChanged: the change was accepted.
type ChangePreparedSpellsChanged struct {
	state  SpellcastingState
	events []SpellcastingEvent
}

// State returns the new state.
func (r ChangePreparedSpellsChanged) State() SpellcastingState { return r.state }

// Events returns a new slice of the events; none when nothing changed.
func (r ChangePreparedSpellsChanged) Events() []SpellcastingEvent { return slices.Clone(r.events) }

// ChangePreparedSpellsRejected: a rule refused the change; the state is unchanged.
type ChangePreparedSpellsRejected struct{ rejection PreparationRejection }

// Rejection returns every problem found.
func (r ChangePreparedSpellsRejected) Rejection() PreparationRejection { return r.rejection }

// ChangePreparedSpellsContentMissing: a spell named is absent from the spells
// supplied; the state is unchanged.
type ChangePreparedSpellsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the absent spells, in the order named, without repeats.
func (r ChangePreparedSpellsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChangePreparedSpellsChanged) isChangePreparedSpellsResult()        {}
func (ChangePreparedSpellsRejected) isChangePreparedSpellsResult()       {}
func (ChangePreparedSpellsContentMissing) isChangePreparedSpellsResult() {}

// ChooseAdditionalSpells is the use case that adds spells and cantrips to a
// class's lists. It wraps ChoosePreparation and holds no rules logic. Create it
// with NewChooseAdditionalSpells; the zero value is not usable. It is safe for
// concurrent use only if its telemetry and clock are.
type ChooseAdditionalSpells struct {
	sig spellcastingSignals
	set bool
}

// NewChooseAdditionalSpells returns the use case. Options: WithTelemetry,
// WithClock; the others are ignored.
func NewChooseAdditionalSpells(opts ...Option) ChooseAdditionalSpells {
	return ChooseAdditionalSpells{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do adds the spells and cantrips as ChoosePreparation does.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state or an invalid class, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns an error matching
// ErrClassNotHeld, recorded on the span only, when the state does not hold the class.
func (u ChooseAdditionalSpells) Do(ctx context.Context, cmd ChooseAdditionalSpellsCommand) (ChooseAdditionalSpellsResult, error) {
	if !u.set || cmd.State.IsZero() || !cmd.Class.Valid() {
		return nil, invalidInput("choose additional spells: zero use case or state, or invalid class")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanChooseAdditionalSpells, ValueOperationChooseAdditionalSpells, telemetry.String(AttrClass, cmd.Class.String()))
	defer run.end(ctx)
	out, err := ChoosePreparation(cmd.State, cmd.Class, cmd.Prepared, cmd.Cantrips, cmd.Spells)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	switch o := out.(type) {
	case PreparationAccepted:
		run.record(ctx, u.sig.changes, false, ValueResultChanged, "")
		return ChooseAdditionalSpellsChosen(o), nil
	case PreparationRefused:
		run.record(ctx, u.sig.changes, false, ValueResultRejected, o.rejection.First().String())
		return ChooseAdditionalSpellsRejected(o), nil
	case PreparationContentMissing:
		run.record(ctx, u.sig.changes, false, ValueResultContentMissing, "")
		return ChooseAdditionalSpellsContentMissing(o), nil
	default:
		panic(invalidInput("choose additional spells: unreachable outcome"))
	}
}

// ChangePreparedSpells is the use case that replaces and adds prepared spells,
// cantrips and granted cantrips on a trigger. It wraps ChangePreparation and holds
// no rules logic. Create it with NewChangePreparedSpells; the zero value is not
// usable. It is safe for concurrent use only if its telemetry and clock are.
type ChangePreparedSpells struct {
	sig spellcastingSignals
	set bool
}

// NewChangePreparedSpells returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewChangePreparedSpells(opts ...Option) ChangePreparedSpells {
	return ChangePreparedSpells{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do changes the lists as ChangePreparation does.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state, an invalid class or an invalid trigger, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It
// returns an error matching ErrClassNotHeld, or an invalid-input error for a
// replacement naming another class or an ungranted cantrip, recorded on the span
// only.
func (u ChangePreparedSpells) Do(ctx context.Context, cmd ChangePreparedSpellsCommand) (ChangePreparedSpellsResult, error) {
	if !u.set || cmd.State.IsZero() || !cmd.Class.Valid() || !cmd.Trigger.Valid() {
		return nil, invalidInput("change prepared spells: zero use case or state, or invalid class or trigger")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanChangePreparedSpells, ValueOperationChangePreparedSpells, telemetry.String(AttrClass, cmd.Class.String()))
	defer run.end(ctx)
	out, err := ChangePreparation(cmd.State, cmd.Class, cmd.Trigger, cmd.Change, cmd.Spells)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	switch o := out.(type) {
	case PreparationAccepted:
		run.record(ctx, u.sig.changes, false, ValueResultChanged, "")
		return ChangePreparedSpellsChanged(o), nil
	case PreparationRefused:
		run.record(ctx, u.sig.changes, false, ValueResultRejected, o.rejection.First().String())
		return ChangePreparedSpellsRejected(o), nil
	case PreparationContentMissing:
		run.record(ctx, u.sig.changes, false, ValueResultContentMissing, "")
		return ChangePreparedSpellsContentMissing(o), nil
	default:
		panic(invalidInput("change prepared spells: unreachable outcome"))
	}
}
