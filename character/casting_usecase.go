package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// CastSpellCommand is the input of CastSpell.Do. Construct it with a keyed
// literal. Do does not alter it. The optional fields are zero (or nil) when absent.
type CastSpellCommand struct {
	State SpellcastingState
	// Class is the casting class; absent only for CastModeSpellScroll, CastModeDuplicated
	// and a class-less CastModeFeatureGranted cast.
	Class SpellcastingClass
	// Stats are the casting stats for the report: required for a class-less feature-granted
	// cast, optional with a class (their ability MUST then be the class's), absent otherwise.
	Stats *SpellcastingStats
	Spell content.Spell
	Mode  CastMode
	// Slot is the slot a CastModeNormal cast of a level 1+ spell spends; nil otherwise.
	Slot SpellSlotChoice
	// CastingTimeMode picks one of the spell's casting time modes; the zero mode is the first.
	CastingTimeMode content.CastingTimeMode
	Context         CastingContext
	// Terms are required with CastModeSpellScroll and CastModeDuplicated and absent otherwise.
	Terms *SpellCastTerms
	// FeatureCastLevel (1 to 9, at least the spell's level) is the level a feature-granted
	// cast is made at; 0 means the spell's own level.
	FeatureCastLevel int
	// CountsAsSlotSpell applies the one-slot-per-turn rule to a feature-granted cast.
	CountsAsSlotSpell bool
}

// CastSpellResult is the closed set of outcomes of CastSpell.Do. Variants:
// CastSpellCast, CastSpellStarted, CastSpellRejected.
type CastSpellResult interface{ isCastSpellResult() }

// CastSpellCast: the spell took effect.
type CastSpellCast struct {
	report CastReport
	state  SpellcastingState
	events []Event
}

// Report returns the report of the cast.
func (r CastSpellCast) Report() CastReport { return r.report }

// State returns the new state.
func (r CastSpellCast) State() SpellcastingState { return r.state }

// Events returns a new slice of the events in order: SpellSlotExpended, ConcentrationEnded,
// ConcentrationBegan, SpellCast, as they apply.
func (r CastSpellCast) Events() []Event { return slices.Clone(r.events) }

// CastSpellStarted: a longer casting began; no slot is spent until CompleteCasting.
type CastSpellStarted struct {
	state  SpellcastingState
	events []Event
}

// State returns the new state, holding the CastingInProgress.
func (r CastSpellStarted) State() SpellcastingState { return r.state }

// Events returns a new slice of the events: ConcentrationEnded (if any) then ConcentrationBegan.
func (r CastSpellStarted) Events() []Event { return slices.Clone(r.events) }

// CastSpellRejected: a rule refused the cast; the state is unchanged.
type CastSpellRejected struct{ rejection CastRejection }

// Rejection returns every reason, in the fixed order.
func (r CastSpellRejected) Rejection() CastRejection { return r.rejection }

func (CastSpellCast) isCastSpellResult()     {}
func (CastSpellStarted) isCastSpellResult()  {}
func (CastSpellRejected) isCastSpellResult() {}

// CastSpell is the use case that casts a spell (SRD 5.2.1, "Casting Spells", pp.
// 104-106). Create it with NewCastSpell; the zero value is not usable. It is safe
// for concurrent use only if its telemetry and clock are.
type CastSpell struct {
	sig castingSignals
	set bool
}

// NewCastSpell returns the use case. Options: WithTelemetry, WithClock; the others are ignored.
func NewCastSpell(opts ...Option) CastSpell {
	return CastSpell{sig: newCastingSignals(newOptions(opts)), set: true}
}

// Do casts the spell. A cast the rules refuse gives CastSpellRejected; one with a
// casting time of a minute or more (or a Ritual) gives CastSpellStarted.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or contradictory input (see CastSpellCommand), and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled.
func (u CastSpell) Do(ctx context.Context, cmd CastSpellCommand) (CastSpellResult, error) {
	if !u.set {
		return nil, invalidInput("cast spell: zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := spellCasting{}.cast(castInput{state: cmd.State, class: cmd.Class, stats: cmd.Stats, spell: cmd.Spell, mode: cmd.Mode, slot: cmd.Slot,
		timeMode: cmd.CastingTimeMode, ctx: cmd.Context, terms: cmd.Terms, featureLevel: cmd.FeatureCastLevel, countsAsSlot: cmd.CountsAsSlotSpell})
	if err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanCastSpell, ValueOperationCastSpell, castAttrs(castSourceFor(cmd.Mode, cmd.Spell), cmd.Class)...)
	defer run.end(ctx)
	var result CastSpellResult
	var res, rejection string
	switch o := out.(type) {
	case castPerformed:
		res, result = ValueResultCast, CastSpellCast(o)
		u.sig.recordSlot(ctx, o.report)
	case castBegun:
		res, result = ValueResultStarted, CastSpellStarted(o)
	case castRefused:
		res, rejection, result = ValueResultRejected, o.rejection.First().String(), CastSpellRejected(o)
	default:
		panic(invalidInput("cast spell: unreachable outcome"))
	}
	run.record(ctx, u.sig.cast, false, res, rejection)
	u.sig.tel.Logger.Log(ctx, telemetry.LevelDebug, LogSpellCastResolved, func() []telemetry.Attr {
		return []telemetry.Attr{telemetry.String(AttrResult, res)}
	})
	return result, nil
}

// castAttrs are the attributes of a cast: its source and, when there is one, its class.
func castAttrs(source CastSource, class SpellcastingClass) []telemetry.Attr {
	attrs := []telemetry.Attr{telemetry.String(AttrCastSource, source.String())}
	if class.Valid() {
		attrs = append(attrs, telemetry.String(AttrClass, class.String()))
	}
	return attrs
}

// recordSlot records the level of the slot a cast spent, when it spent one.
func (s castingSignals) recordSlot(ctx context.Context, r CastReport) {
	if pool, level, ok := r.Slot(); ok {
		s.slotLevel.Record(ctx, float64(level.Value()), telemetry.String(AttrPool, pool.String()))
	}
}

// CompleteCastingCommand is the input of CompleteCasting.Do. Construct it with a
// keyed literal. Do does not alter it.
type CompleteCastingCommand struct {
	State   SpellcastingState
	Spell   content.Spell
	Context CastingContext
	// Slot replaces the slot chosen at the start; nil keeps it. Only for a casting with a slot.
	Slot SpellSlotChoice
	// Stats and Terms are as in CastSpellCommand, for the report of the cast.
	Stats *SpellcastingStats
	Terms *SpellCastTerms
	// FeatureCastLevel and CountsAsSlotSpell are as in CastSpellCommand, for a feature-granted casting.
	FeatureCastLevel  int
	CountsAsSlotSpell bool
}

// CompleteCastingResult is the closed set of outcomes of CompleteCasting.Do.
// Variants: CompleteCastingCompleted, CompleteCastingRejected.
type CompleteCastingResult interface{ isCompleteCastingResult() }

// CompleteCastingCompleted: the spell took effect and the casting is over.
type CompleteCastingCompleted struct {
	report CastReport
	state  SpellcastingState
	events []Event
}

// Report returns the report of the cast.
func (r CompleteCastingCompleted) Report() CastReport { return r.report }

// State returns the new state.
func (r CompleteCastingCompleted) State() SpellcastingState { return r.state }

// Events returns a new slice of the events in order.
func (r CompleteCastingCompleted) Events() []Event { return slices.Clone(r.events) }

// CompleteCastingRejected: nothing to complete, or a re-check failed; the state is
// unchanged and the casting stays in progress.
type CompleteCastingRejected struct{ rejection CastRejection }

// Rejection returns every reason, in the fixed order.
func (r CompleteCastingRejected) Rejection() CastRejection { return r.rejection }

func (CompleteCastingCompleted) isCompleteCastingResult() {}
func (CompleteCastingRejected) isCompleteCastingResult()  {}

// CompleteCasting is the use case that finishes a casting of a minute or more.
// Create it with NewCompleteCasting; the zero value is not usable. It is safe for
// concurrent use only if its telemetry and clock are.
type CompleteCasting struct {
	sig castingSignals
	set bool
}

// NewCompleteCasting returns the use case. Options: WithTelemetry, WithClock; the others are ignored.
func NewCompleteCasting(opts ...Option) CompleteCasting {
	return CompleteCasting{sig: newCastingSignals(newOptions(opts)), set: true}
}

// Do completes the casting. Nothing in progress, or a re-check the rules fail,
// gives CompleteCastingRejected, not an error.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a spell other than the one being cast or contradictory input,
// and ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u CompleteCasting) Do(ctx context.Context, cmd CompleteCastingCommand) (CompleteCastingResult, error) {
	if !u.set {
		return nil, invalidInput("complete casting: zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := spellCasting{}.complete(completeInput{state: cmd.State, spell: cmd.Spell, ctx: cmd.Context, slot: cmd.Slot, stats: cmd.Stats, terms: cmd.Terms,
		featureLevel: cmd.FeatureCastLevel, countsAsSlot: cmd.CountsAsSlotSpell})
	if err != nil {
		return nil, err
	}
	var attrs []telemetry.Attr
	if c, ok := cmd.State.CastingInProgress(); ok {
		attrs = castAttrs(c.Source(), c.Class())
	}
	ctx, run := u.sig.begin(ctx, SpanCompleteCasting, ValueOperationCompleteCasting, attrs...)
	defer run.end(ctx)
	switch o := out.(type) {
	case castPerformed:
		u.sig.recordSlot(ctx, o.report)
		run.record(ctx, u.sig.cast, false, ValueResultCast, "")
		return CompleteCastingCompleted(o), nil
	case castRefused:
		run.record(ctx, u.sig.cast, false, ValueResultRejected, o.rejection.First().String())
		return CompleteCastingRejected(o), nil
	default:
		panic(invalidInput("complete casting: unreachable outcome"))
	}
}

// BreakConcentrationCommand is the input of BreakConcentration.Do. Construct it
// with a keyed literal. Do does not alter it.
type BreakConcentrationCommand struct {
	State  SpellcastingState
	Reason ConcentrationBreakReason
}

// BreakConcentrationResult is the closed set of outcomes of BreakConcentration.Do.
// Variants: BreakConcentrationEnded, BreakConcentrationCastingFailed,
// BreakConcentrationUnchanged.
type BreakConcentrationResult interface{ isBreakConcentrationResult() }

// BreakConcentrationEnded: Concentration ended; no casting was in progress.
type BreakConcentrationEnded struct {
	state  SpellcastingState
	events []CastingEvent
}

// State returns the new state.
func (r BreakConcentrationEnded) State() SpellcastingState { return r.state }

// Events returns a new slice of the events: ConcentrationEnded.
func (r BreakConcentrationEnded) Events() []CastingEvent { return slices.Clone(r.events) }

// BreakConcentrationCastingFailed: Concentration ended and the casting in progress failed.
type BreakConcentrationCastingFailed struct {
	state   SpellcastingState
	events  []CastingEvent
	failure CastingFailed
}

// State returns the new state.
func (r BreakConcentrationCastingFailed) State() SpellcastingState { return r.state }

// Events returns a new slice of the events: ConcentrationEnded then CastingFailed.
func (r BreakConcentrationCastingFailed) Events() []CastingEvent { return slices.Clone(r.events) }

// CastingFailed returns the failed casting, repeated from Events.
func (r BreakConcentrationCastingFailed) CastingFailed() CastingFailed { return r.failure }

// BreakConcentrationUnchanged: no Concentration was in progress; the state is unchanged.
type BreakConcentrationUnchanged struct{ state SpellcastingState }

// State returns the unchanged state.
func (r BreakConcentrationUnchanged) State() SpellcastingState { return r.state }

func (BreakConcentrationEnded) isBreakConcentrationResult()         {}
func (BreakConcentrationCastingFailed) isBreakConcentrationResult() {}
func (BreakConcentrationUnchanged) isBreakConcentrationResult()     {}

// BreakConcentration is the use case that ends Concentration for a reason. It
// returns the transition of ConcentrationRules.BreakConcentration unchanged.
// Create it with NewBreakConcentration; the zero value is not usable. It is safe
// for concurrent use only if its telemetry and clock are.
type BreakConcentration struct {
	sig castingSignals
	set bool
}

// NewBreakConcentration returns the use case. Options: WithTelemetry, WithClock; the others are ignored.
func NewBreakConcentration(opts ...Option) BreakConcentration {
	return BreakConcentration{sig: newCastingSignals(newOptions(opts)), set: true}
}

// Do ends the Concentration. With none in progress it succeeds with
// BreakConcentrationUnchanged.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state or an invalid reason, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled.
func (u BreakConcentration) Do(ctx context.Context, cmd BreakConcentrationCommand) (BreakConcentrationResult, error) {
	if !u.set || cmd.State.IsZero() || !cmd.Reason.Valid() {
		return nil, invalidInput("break concentration: zero use case or state, or invalid reason")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanBreakConcentration, ValueOperationBreakConcentration, telemetry.String(AttrConcentrationReason, cmd.Reason.String()))
	defer run.end(ctx)
	t := ConcentrationRules{}.BreakConcentration(cmd.State, cmd.Reason)
	switch failed, casting := t.CastingFailed(); {
	case casting:
		run.finish(ctx, ValueResultCastingFailed, "")
		return BreakConcentrationCastingFailed{state: t.state, events: t.events, failure: failed}, nil
	case len(t.events) > 0:
		run.finish(ctx, ValueResultEnded, "")
		return BreakConcentrationEnded{state: t.state, events: t.events}, nil
	}
	run.finish(ctx, ValueResultNone, "")
	return BreakConcentrationUnchanged{state: t.state}, nil
}

// BeginConcentrationCommand is the input of BeginConcentration.Do. Construct it
// with a keyed literal. Do does not alter it.
type BeginConcentrationCommand struct {
	State SpellcastingState
	// Subject is a ConcentrationSubjectFeature for a non-spell effect or a
	// ConcentrationSubjectSpell for a held spell that is not cast.
	Subject ConcentrationSubject
}

// BeginConcentrationResult is the closed set of outcomes of BeginConcentration.Do.
// Variants: BeginConcentrationBegan, BeginConcentrationCastingFailed.
type BeginConcentrationResult interface{ isBeginConcentrationResult() }

// BeginConcentrationBegan: Concentration began; no casting was in progress.
type BeginConcentrationBegan struct {
	state  SpellcastingState
	events []CastingEvent
}

// State returns the new state.
func (r BeginConcentrationBegan) State() SpellcastingState { return r.state }

// Events returns a new slice of the events: ConcentrationEnded (if any) then ConcentrationBegan.
func (r BeginConcentrationBegan) Events() []CastingEvent { return slices.Clone(r.events) }

// BeginConcentrationCastingFailed: Concentration began and the casting in progress failed.
type BeginConcentrationCastingFailed struct {
	state   SpellcastingState
	events  []CastingEvent
	failure CastingFailed
}

// State returns the new state.
func (r BeginConcentrationCastingFailed) State() SpellcastingState { return r.state }

// Events returns a new slice of the events: ConcentrationEnded, CastingFailed, ConcentrationBegan.
func (r BeginConcentrationCastingFailed) Events() []CastingEvent { return slices.Clone(r.events) }

// CastingFailed returns the failed casting, repeated from Events.
func (r BeginConcentrationCastingFailed) CastingFailed() CastingFailed { return r.failure }

func (BeginConcentrationBegan) isBeginConcentrationResult()         {}
func (BeginConcentrationCastingFailed) isBeginConcentrationResult() {}

// BeginConcentration is the use case that begins Concentration on an effect that
// is not a cast (SRD 5.2.1, "Concentration", p. 179). Create it with
// NewBeginConcentration; the zero value is not usable. It is safe for concurrent
// use only if its telemetry and clock are.
type BeginConcentration struct {
	sig castingSignals
	set bool
}

// NewBeginConcentration returns the use case. Options: WithTelemetry, WithClock; the others are ignored.
func NewBeginConcentration(opts ...Option) BeginConcentration {
	return BeginConcentration{sig: newCastingSignals(newOptions(opts)), set: true}
}

// Do begins the Concentration, ending any in progress and failing a casting in progress.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero state or a missing or invalid subject, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled.
func (u BeginConcentration) Do(ctx context.Context, cmd BeginConcentrationCommand) (BeginConcentrationResult, error) {
	if !u.set || cmd.State.IsZero() || !validSubject(cmd.Subject) {
		return nil, invalidInput("begin concentration: zero use case or state, or missing or invalid subject")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanBeginConcentration, ValueOperationBeginConcentration)
	defer run.end(ctx)
	state, events, failed := beginConcentration(cmd.State, cmd.Subject)
	if failed.spell.Valid() {
		run.finish(ctx, ValueResultCastingFailed, "")
		return BeginConcentrationCastingFailed{state: state, events: events, failure: failed}, nil
	}
	run.finish(ctx, ValueResultBegan, "")
	return BeginConcentrationBegan{state: state, events: events}, nil
}

// validSubject reports whether s is a spell or feature subject with a valid id.
func validSubject(s ConcentrationSubject) bool {
	switch v := s.(type) {
	case ConcentrationSubjectSpell:
		return v.spell.Valid()
	case ConcentrationSubjectFeature:
		return v.feature.Valid()
	}
	return false
}
