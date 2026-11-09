package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// PreviewLevelAdvancementCommand is the input of PreviewLevelAdvancement.Run.
// Construct it with a keyed literal. Run does not alter it.
type PreviewLevelAdvancementCommand struct {
	Character Character
	Class     content.ClassID
}

// PreviewLevelAdvancementResult is the closed set of outcomes of
// PreviewLevelAdvancement.Run. Variants: PreviewLevelAdvancementAvailable,
// PreviewLevelAdvancementUnavailable, PreviewLevelAdvancementContentMissing.
type PreviewLevelAdvancementResult interface{ isPreviewLevelAdvancementResult() }

// PreviewLevelAdvancementAvailable is the outcome of a level the character can
// gain. Construct it with a keyed literal.
type PreviewLevelAdvancementAvailable struct{ Gain LevelAdvancementGain }

// PreviewLevelAdvancementUnavailable is the outcome of a level a rule refuses.
// Construct it with a keyed literal.
type PreviewLevelAdvancementUnavailable struct{ Block LevelAdvancementBlock }

// PreviewLevelAdvancementContentMissing is the outcome when the chosen class, or
// for a new class a current class, is unknown to the content source. It names at
// least one entry. Only Run builds one; the zero value names nothing.
type PreviewLevelAdvancementContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the unknown entries, ordered by id, without
// duplicates.
func (r PreviewLevelAdvancementContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (PreviewLevelAdvancementAvailable) isPreviewLevelAdvancementResult()      {}
func (PreviewLevelAdvancementUnavailable) isPreviewLevelAdvancementResult()    {}
func (PreviewLevelAdvancementContentMissing) isPreviewLevelAdvancementResult() {}

// PreviewLevelAdvancement is the use case that previews gaining a level in a
// class: it looks up the chosen class (and, for a new class, every current
// class), asks LevelAdvancementPreviewer and emits the srd5e.character.*
// signals. It holds no rules logic and never changes a Character. Create it with
// NewPreviewLevelAdvancement; the zero value is not usable. It is safe for
// concurrent use only if its source, telemetry and clock are.
type PreviewLevelAdvancement struct {
	source ContentSource
	sig    operationSignals
}

// NewPreviewLevelAdvancement returns the use case. Options: WithSource
// (required), WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput when the source is missing.
func NewPreviewLevelAdvancement(opts ...Option) (PreviewLevelAdvancement, error) {
	o := newOptions(opts)
	if o.source == nil {
		return PreviewLevelAdvancement{}, invalidInput("level advancement preview without a content source")
	}
	return PreviewLevelAdvancement{source: o.source, sig: newOperationSignals(o)}, nil
}

// Run previews cmd.Class for cmd.Character. Every lookup completes before any
// rule is evaluated, so a missing class wins over every rejection.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case, a zero Character or an invalid class, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It
// returns a *srderr.PortError (error type "content_error") with a nil result when
// the source fails, panics or breaks its contract; the failure is recorded once
// and never logged. A level the rules refuse is a
// PreviewLevelAdvancementUnavailable result.
func (u PreviewLevelAdvancement) Run(ctx context.Context, cmd PreviewLevelAdvancementCommand) (PreviewLevelAdvancementResult, error) {
	if u.source == nil {
		return nil, invalidInput("zero PreviewLevelAdvancement; use NewPreviewLevelAdvancement")
	}
	if cmd.Character.IsZero() || !cmd.Class.Valid() {
		return nil, invalidInput("level advancement preview of a zero character or an invalid class")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAdvancementPreview, ValueOperationPreview)
	defer run.end(ctx)

	loaded, err := loadClasses(ctx, u.source, u.sig.tel.Tracer, SpanAdvancementLoadContent, cmd.Character, cmd.Class, cmd.Character.classes.LevelIn(cmd.Class) == 0)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(loaded.missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return PreviewLevelAdvancementContentMissing{missing: loaded.missing}, nil
	}
	preview, err := LevelAdvancementPreviewer{}.Preview(cmd.Character, loaded.class, loaded.requirements)
	if err != nil { // unreachable for a decoded class and a checked character
		run.portFailure(ctx, err, "")
		return nil, err
	}
	switch p := preview.(type) {
	case LevelAdvancementGain:
		run.finish(ctx, ValueResultAvailable, "")
		return PreviewLevelAdvancementAvailable{Gain: p}, nil
	case LevelAdvancementBlock:
		run.finish(ctx, ValueResultUnavailable, levelUpRejectionValue(p.rejection))
		return PreviewLevelAdvancementUnavailable{Block: p}, nil
	default:
		panic(invalidInput("unknown preview")) // unreachable: LevelAdvancementPreview is closed
	}
}
