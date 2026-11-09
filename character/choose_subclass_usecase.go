package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ChooseSubclassCommand is the input of ChooseSubclass.Run. Construct it with a
// keyed literal. Run does not alter it.
type ChooseSubclassCommand struct {
	Character Character
	Class     content.ClassID
	Subclass  content.SubclassID
}

// ChooseSubclassResult is the closed set of outcomes of ChooseSubclass.Run.
// Variants: ChooseSubclassSucceeded, ChooseSubclassRejected,
// ChooseSubclassContentMissing.
type ChooseSubclassResult interface{ isChooseSubclassResult() }

// ChooseSubclassSucceeded is the outcome of a subclass chosen: the new Character
// and the SubclassChosen event. Only Run builds one.
type ChooseSubclassSucceeded struct {
	character Character
	event     SubclassChosen
}

// Character returns the Character holding the subclass.
func (r ChooseSubclassSucceeded) Character() Character { return r.character }

// Event returns the SubclassChosen event.
func (r ChooseSubclassSucceeded) Event() SubclassChosen { return r.event }

// ChooseSubclassRejected is the outcome of a choice a rule refused.
type ChooseSubclassRejected struct{ Rejection SubclassRejected }

// ChooseSubclassContentMissing is the outcome when the class is unknown to the
// content source. It names the class. Only Run builds one; the zero value names
// nothing.
type ChooseSubclassContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the unknown entries.
func (r ChooseSubclassContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChooseSubclassSucceeded) isChooseSubclassResult()      {}
func (ChooseSubclassRejected) isChooseSubclassResult()       {}
func (ChooseSubclassContentMissing) isChooseSubclassResult() {}

// ChooseSubclass is the use case that gives a Character the subclass of a class:
// it looks up the class, asks SubclassSelection and emits the srd5e.character.*
// signals. It holds no rules logic. Create it with NewChooseSubclass; the zero
// value is not usable. It is safe for concurrent use only if its source,
// telemetry and clock are.
type ChooseSubclass struct {
	source ContentSource
	sig    operationSignals
}

// NewChooseSubclass returns the use case. Options: WithSource (required),
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// when the source is missing.
func NewChooseSubclass(opts ...Option) (ChooseSubclass, error) {
	o := newOptions(opts)
	if o.source == nil {
		return ChooseSubclass{}, invalidInput("subclass choice without a content source")
	}
	return ChooseSubclass{source: o.source, sig: newOperationSignals(o)}, nil
}

// Run applies cmd. Only the class is looked up.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case, a zero Character, an invalid class or subclass
// id, or a stored state under the class's key of another type, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "content_error") with a nil result when the
// source fails, panics or breaks its contract; the failure is recorded once and
// never logged. A refused choice, an unlisted subclass included, is a
// ChooseSubclassRejected result.
func (u ChooseSubclass) Run(ctx context.Context, cmd ChooseSubclassCommand) (ChooseSubclassResult, error) {
	if u.source == nil {
		return nil, invalidInput("zero ChooseSubclass; use NewChooseSubclass")
	}
	if cmd.Character.IsZero() || !cmd.Class.Valid() || !cmd.Subclass.Valid() {
		return nil, invalidInput("subclass choice of a zero character or an invalid id")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAdvancementChooseSubclass, ValueOperationChooseSubclass)
	defer run.end(ctx)

	loaded, err := loadClasses(ctx, u.source, u.sig.tel.Tracer, SpanAdvancementLoadContent, cmd.Character, cmd.Class, false)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(loaded.missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return ChooseSubclassContentMissing{missing: loaded.missing}, nil
	}
	tr, err := SubclassSelection{}.Choose(cmd.Character, loaded.class, cmd.Subclass)
	if err != nil { // a stored state of another type under the key
		run.commandError(err)
		return nil, err
	}
	switch t := tr.(type) {
	case TransitionChanged[SubclassRejected]:
		run.finish(ctx, ValueResultChosen, "")
		return ChooseSubclassSucceeded{character: t.character, event: t.events[0].(SubclassChosen)}, nil
	case TransitionRejected[SubclassRejected]:
		run.finish(ctx, ValueResultRejected, t.rejection.Reason.String())
		return ChooseSubclassRejected{Rejection: t.rejection}, nil
	default:
		panic(invalidInput("unknown transition")) // unreachable: Transition is closed
	}
}
