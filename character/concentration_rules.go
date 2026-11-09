package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ConcentrationBreakTransition is the result of ConcentrationRules.BreakConcentration:
// the new state, the events in order (ConcentrationEnded, then CastingFailed when
// a casting was in progress) and that CastingFailed on its own. It holds a
// slice, so compare two with reflect.DeepEqual.
type ConcentrationBreakTransition struct {
	state  SpellcastingState
	events []CastingEvent
	failed CastingFailed
}

// State returns the new state.
func (t ConcentrationBreakTransition) State() SpellcastingState { return t.state }

// Events returns a new slice of the events; none when no Concentration was in progress.
func (t ConcentrationBreakTransition) Events() []CastingEvent { return slices.Clone(t.events) }

// CastingFailed returns the failed casting, repeated from Events, and whether a
// casting was in progress.
func (t ConcentrationBreakTransition) CastingFailed() (CastingFailed, bool) {
	return t.failed, t.failed.spell.Valid()
}

// CounterCastTransition is the result of ConcentrationRules.CounterCast: the new
// state and the events in order. It holds a slice, so compare two with
// reflect.DeepEqual.
type CounterCastTransition struct {
	state  SpellcastingState
	events []CastingEvent
}

// State returns the new state.
func (t CounterCastTransition) State() SpellcastingState { return t.state }

// Events returns a new slice of the events.
func (t CounterCastTransition) Events() []CastingEvent { return slices.Clone(t.events) }

// ConcentrationRules is the stateless pure domain service for ending
// Concentration. The zero value is ready to use. It records nothing and calls no
// use case.
type ConcentrationRules struct{}

// BreakConcentration ends the Concentration in progress for reason (SRD 5.2.1,
// "Concentration", p. 179). A casting in progress is cleared with no slot
// spent and reported as CastingFailed (SRD 5.2.1, "Longer Casting Times", p.
// 105). With no Concentration in progress, or an invalid reason, it returns the
// state unchanged, no events and no CastingFailed. It never panics.
func (ConcentrationRules) BreakConcentration(state SpellcastingState, reason ConcentrationBreakReason) ConcentrationBreakTransition {
	conc, ok := state.Concentration()
	if !ok || !reason.Valid() {
		return ConcentrationBreakTransition{state: state}
	}
	end := ConcentrationEndReason(reason)
	t := ConcentrationBreakTransition{state: state, events: []CastingEvent{ConcentrationEnded{id: state.id, subject: conc.Subject(), reason: end}}}
	t.state.conc = Concentration{}
	if c, going := state.CastingInProgress(); going {
		t.failed = CastingFailed{id: state.id, spell: c.Spell(), reason: end}
		t.events = append(t.events, t.failed)
		t.state.casting = CastingInProgress{}
	}
	return t
}

// CounterCast applies a countered cast of spell: an earlier Concentration still
// ends when the spell requires Concentration, because it ends "the moment you
// start casting"; no slot is expended, no Concentration begins and one
// CastingCountered follows (SRD 5.2.1, "Counterspell", p. 120; "Concentration",
// p. 179; the earlier Concentration ending is a package interpretation).
//
// It returns an error matching srderr.ErrInvalidInput for a zero state, an invalid
// spell or a state with a casting in progress (a countered longer casting is
// BreakConcentration with BreakEnded). It never panics.
func (ConcentrationRules) CounterCast(state SpellcastingState, spell content.Spell) (CounterCastTransition, error) {
	if _, going := state.CastingInProgress(); going || state.IsZero() || !spell.ID().Valid() {
		return CounterCastTransition{}, invalidInput("counter cast: zero state, invalid spell or a casting in progress")
	}
	t := CounterCastTransition{state: state}
	if conc, ok := state.Concentration(); ok && spell.RequiresConcentration() {
		t.events = append(t.events, ConcentrationEnded{id: state.id, subject: conc.Subject(), reason: EndedAnotherEffect})
		t.state.conc = Concentration{}
	}
	t.events = append(t.events, CastingCountered{id: state.id, spell: spell.ID()})
	return t, nil
}

// beginConcentration begins Concentration on subject as an effect: any
// Concentration in progress ends with EndedAnotherEffect, a casting in progress
// fails, and ConcentrationBegan follows, in that order. failed is the
// CastingFailed also present in events, zero when no casting was in progress.
func beginConcentration(state SpellcastingState, subject ConcentrationSubject) (next SpellcastingState, events []CastingEvent, failed CastingFailed) {
	next = state
	if conc, ok := state.Concentration(); ok {
		events = append(events, ConcentrationEnded{id: state.id, subject: conc.Subject(), reason: EndedAnotherEffect})
	}
	if c, going := state.CastingInProgress(); going {
		failed = CastingFailed{id: state.id, spell: c.Spell(), reason: EndedAnotherEffect}
		events = append(events, failed)
		next.casting = CastingInProgress{}
	}
	next.conc = Concentration{subject: subject, kind: ConcentrationEffect}
	events = append(events, ConcentrationBegan{id: state.id, subject: subject, kind: ConcentrationEffect})
	return next, events, failed
}
