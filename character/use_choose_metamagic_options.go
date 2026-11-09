package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ChooseMetamagicOptionsCommand is the input of ChooseMetamagicOptions.Run. Chosen is the full set of Metamagic options the
// Sorcerer holds afterwards, as many as its level entitles it to. Construct it with a keyed literal. Run does not alter it.
type ChooseMetamagicOptionsCommand struct {
	Character Character
	Chosen    []content.FeatureOptionID
}

// ChooseMetamagicOptionsResult is the closed set of outcomes of ChooseMetamagicOptions.Run. Variants:
// ChooseMetamagicOptionsApplied, ChooseMetamagicOptionsRejected, ChooseMetamagicOptionsContentMissing.
type ChooseMetamagicOptionsResult interface{ isChooseMetamagicOptionsResult() }

// ChooseMetamagicOptionsApplied is the outcome of a selection the rules allowed.
type ChooseMetamagicOptionsApplied struct {
	character Character
	events    []Event
}

// Character returns the new Character holding the new SorcererState.
func (r ChooseMetamagicOptionsApplied) Character() Character { return r.character }

// Events returns a new slice holding the MetamagicOptionsChosen event.
func (r ChooseMetamagicOptionsApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseMetamagicOptionsRejected is the outcome of a selection a rule refused.
type ChooseMetamagicOptionsRejected struct{ Rejection SorcererRejection }

// ChooseMetamagicOptionsContentMissing is the outcome when the sorcerer class is not in the source.
type ChooseMetamagicOptionsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseMetamagicOptionsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChooseMetamagicOptionsApplied) isChooseMetamagicOptionsResult()        {}
func (ChooseMetamagicOptionsRejected) isChooseMetamagicOptionsResult()       {}
func (ChooseMetamagicOptionsContentMissing) isChooseMetamagicOptionsResult() {}

// ChooseMetamagicOptions is the use case that sets the Metamagic options a Sorcerer knows (SRD 5.2.1, "Level 2: Metamagic",
// p. 66; class-sorcerer.md FR-035, FR-055 to FR-058), checking them against the options of the class definition. Create it with
// NewChooseMetamagicOptions; the zero value is not usable.
type ChooseMetamagicOptions struct{ sorcererUse }

// NewChooseMetamagicOptions returns the use case. Options: WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewChooseMetamagicOptions(source ContentSource, opts ...Option) (ChooseMetamagicOptions, error) {
	u, err := newSorcererUse("choose Metamagic options", source, opts)
	return ChooseMetamagicOptions{u}, err
}

// Run sets the options. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError when the source fails. A
// refused selection (SorcererMetamagicSelectionInvalid with every problem, SorcererMetamagicReplacementInvalid,
// SorcererFeatureNotGained, SorcererNotASorcerer) is a ChooseMetamagicOptionsRejected result.
func (u ChooseMetamagicOptions) Run(ctx context.Context, cmd ChooseMetamagicOptionsCommand) (ChooseMetamagicOptionsResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseMetamagicOptions, ValueFeatureMetamagic, cmd.Character)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notSorcerer(ctx); ok {
		return ChooseMetamagicOptionsRejected{Rejection: rej}, nil
	}
	if call.missing(ctx) {
		return ChooseMetamagicOptionsContentMissing{missing: sorcererMissingContent()}, nil
	}
	if call.profile.level.n < sorcererFontOfMagicLevel {
		rej := notGained("metamagic")
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseMetamagicOptionsRejected{Rejection: rej}, nil
	}
	sel, problems := NewMetamagicSelection(call.profile.level, cmd.Chosen, metamagicOptions(call.def))
	if len(problems) != 0 {
		rej := SorcererRejection{Kind: SorcererMetamagicSelectionInvalid, Problems: problems}
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseMetamagicOptionsRejected{Rejection: rej}, nil
	}
	next, events, rej, err := call.commit(ctx, cmd.Character, ChooseMetamagic(cmd.Character.ID(), call.state, call.profile, sel))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return ChooseMetamagicOptionsRejected{Rejection: rej}, nil
	}
	return ChooseMetamagicOptionsApplied{character: next, events: events}, nil
}

// open starts the span, reads the profile and the state of c and, for a Sorcerer, looks the class up in a SpanLoadContent child
// span. It is shared by the four Sorcerer use cases and sits here because it is the one place that wraps a content port failure
// (architecture.md section 9, A6). The caller defers call.run.end(ctx) right after open, whatever err is. A source failure, panic or
// broken contract comes back as a *srderr.PortError, recorded once; another type of state under a key comes back as an error
// matching srderr.ErrInvalidInput, recorded on the span only.
func (u sorcererUse) open(ctx context.Context, span, feature string, c Character) (context.Context, sorcererCall, error) {
	ctx, run := u.sig.start(ctx, span, ValueClassSorcerer, feature)
	call := sorcererCall{run: run}
	p, isSorcerer, err := SorcererProfileOf(c)
	if err == nil && isSorcerer {
		call.state, call.hasState, err = ReadFeatureState(c.FeatureStates(), SorcererStateKey())
	}
	if err != nil {
		run.commandError(err)
		return ctx, call, err
	}
	call.profile, call.isLevels = p, isSorcerer
	if !isSorcerer {
		call.stage = sorcererNotASorcerer
		return ctx, call, nil
	}
	lctx, lspan := u.sig.tel.Tracer.Start(ctx, SpanLoadContent, telemetry.String(AttrClass, ValueClassSorcerer), telemetry.String(AttrFeature, feature))
	defer lspan.End()
	entry, found, err := lookupEntry(lctx, u.source, SorcererClassID.Key())
	if err == nil && found {
		def, ok := entry.(content.ClassDefinition)
		if !ok {
			err = srderr.ContractViolation(content.Port, "lookup", "the sorcerer key answered with another entry type")
		}
		call.def = def
	}
	if err != nil {
		lspan.RecordError(err)
		lspan.SetStatus(telemetry.StatusError)
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return ctx, call, err
	}
	call.stage = sorcererReady
	if !found {
		call.stage = sorcererMissing
	}
	return ctx, call, nil
}
