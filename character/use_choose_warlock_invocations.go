package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ChooseWarlockInvocationsCommand is the input of ChooseWarlockInvocations.Run. Picks is the full set of Eldritch Invocations the
// Warlock holds afterwards, as many as its level entitles it to; KnownCantrips are the facts of the Warlock cantrips the character
// knows. Construct it with a keyed literal. Run does not alter it.
type ChooseWarlockInvocationsCommand struct {
	Character     Character
	Picks         []InvocationPick
	KnownCantrips []CantripFacts
}

// ChooseWarlockInvocationsResult is the closed set of outcomes of ChooseWarlockInvocations.Run. Variants:
// ChooseWarlockInvocationsApplied, ChooseWarlockInvocationsRejected, ChooseWarlockInvocationsContentMissing.
type ChooseWarlockInvocationsResult interface{ isChooseWarlockInvocationsResult() }

// ChooseWarlockInvocationsApplied is the outcome of a selection the rules allowed.
type ChooseWarlockInvocationsApplied struct {
	character Character
	events    []Event
}

// Character returns the new Character holding the new WarlockState.
func (r ChooseWarlockInvocationsApplied) Character() Character { return r.character }

// Events returns a new slice holding the InvocationsChosen event.
func (r ChooseWarlockInvocationsApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseWarlockInvocationsRejected is the outcome of a selection a rule refused.
type ChooseWarlockInvocationsRejected struct{ Rejection WarlockRejection }

// ChooseWarlockInvocationsContentMissing is the outcome when the warlock class is not in the source.
type ChooseWarlockInvocationsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseWarlockInvocationsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChooseWarlockInvocationsApplied) isChooseWarlockInvocationsResult()        {}
func (ChooseWarlockInvocationsRejected) isChooseWarlockInvocationsResult()       {}
func (ChooseWarlockInvocationsContentMissing) isChooseWarlockInvocationsResult() {}

// ChooseWarlockInvocations is the use case that sets the Eldritch Invocations a Warlock knows (SRD 5.2.1, "Level 1: Eldritch
// Invocations", p. 70; class-warlock.md FR-043 to FR-053), checking them against the options and the Invocations column of the class
// definition. Create it with NewChooseWarlockInvocations; the zero value is not usable.
type ChooseWarlockInvocations struct{ warlockUse }

// NewChooseWarlockInvocations returns the use case. Options: WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewChooseWarlockInvocations(source ContentSource, opts ...Option) (ChooseWarlockInvocations, error) {
	u, err := newWarlockUse("choose Eldritch Invocations", source, true, opts)
	return ChooseWarlockInvocations{u}, err
}

// Run sets the invocations. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError when the source fails. A refused
// selection (WarlockInvocationSelectionInvalid with every problem, WarlockNotAWarlock) is a ChooseWarlockInvocationsRejected result.
func (u ChooseWarlockInvocations) Run(ctx context.Context, cmd ChooseWarlockInvocationsCommand) (ChooseWarlockInvocationsResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseWarlockInvocations, ValueFeatureEldritchInvocations, cmd.Character, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return ChooseWarlockInvocationsRejected{Rejection: rej}, nil
	}
	if call.missing(ctx) {
		return ChooseWarlockInvocationsContentMissing{missing: warlockMissingContent()}, nil
	}
	entitled := 0
	if row, err := call.def.RowAt(call.profile.level.n); err == nil {
		if v, err := row.Value("eldritch_invocations"); err == nil {
			if n, ok := v.(content.ColumnCount); ok {
				entitled = n.Value()
			}
		}
	}
	sel, problems := SelectInvocations(call.profile.level, entitled, cmd.Picks, call.state.Invocations(), eldritchInvocationOptions(call.def), cmd.KnownCantrips)
	if len(problems) != 0 {
		rej := WarlockRejection{Kind: WarlockInvocationSelectionInvalid, Problems: problems}
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseWarlockInvocationsRejected{Rejection: rej}, nil
	}
	next, changed, rej, err := call.commit(ctx, cmd.Character, ChooseInvocations(cmd.Character.ID(), call.state, sel))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return ChooseWarlockInvocationsRejected{Rejection: rej}, nil
	}
	return ChooseWarlockInvocationsApplied{character: next, events: changed.Events()}, nil
}

// open starts the span, reads the profile and the state of c and, when load is true and c is a Warlock, looks the class up in a
// SpanLoadContent child span. It sits in this file because it is the one place that wraps a content port failure (architecture.md
// section 9, A6). The caller defers call.run.end(ctx) right after open, whatever err is. A source failure, panic or broken contract
// comes back as a *srderr.PortError, recorded once; another type of state under a key comes back as an error matching
// srderr.ErrInvalidInput, recorded on the span only.
func (u warlockUse) open(ctx context.Context, span, feature string, c Character, load bool) (context.Context, warlockCall, error) {
	ctx, run := u.sig.start(ctx, span, ValueClassWarlock, feature)
	call := warlockCall{run: run}
	p, isWarlock, err := WarlockProfileOf(c)
	if err == nil && isWarlock {
		call.state, _, err = ReadFeatureState(c.FeatureStates(), WarlockStateKey())
		if err == nil {
			call.pact, _, err = ReadFeatureState(c.FeatureStates(), WarlockPactStateKey())
		}
	}
	if err != nil {
		run.commandError(err)
		return ctx, call, err
	}
	call.profile = p
	if !isWarlock {
		call.stage = warlockNotAWarlock
		return ctx, call, nil
	}
	if !load {
		return ctx, call, nil
	}
	lctx, lspan := u.sig.tel.Tracer.Start(ctx, SpanLoadContent, telemetry.String(AttrClass, ValueClassWarlock), telemetry.String(AttrFeature, feature))
	defer lspan.End()
	entry, found, err := lookupEntry(lctx, u.source, WarlockClassID.Key())
	if err == nil && found {
		def, ok := entry.(content.ClassDefinition)
		if !ok {
			err = srderr.ContractViolation(content.Port, "lookup", "the warlock key answered with another entry type")
		}
		call.def = def
	}
	if err != nil {
		lspan.RecordError(err)
		lspan.SetStatus(telemetry.StatusError)
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return ctx, call, err
	}
	if !found {
		call.stage = warlockMissing
	}
	return ctx, call, nil
}

// lookupAll looks the keys up in one SpanLoadContent child span. It returns the entries in the order of keys (nil for one not found) and
// the content that was not found. A source failure, panic or broken contract comes back as a *srderr.PortError, recorded once.
func (u warlockUse) lookupAll(ctx context.Context, call warlockCall, feature string, keys []content.Key) ([]content.Entry, []content.MissingContent, error) {
	lctx, lspan := u.sig.tel.Tracer.Start(ctx, SpanLoadContent, telemetry.String(AttrClass, ValueClassWarlock), telemetry.String(AttrFeature, feature))
	defer lspan.End()
	entries := make([]content.Entry, len(keys))
	var missing []content.MissingContent
	for i, key := range keys {
		entry, found, err := lookupEntry(lctx, u.source, key)
		if err != nil {
			lspan.RecordError(err)
			lspan.SetStatus(telemetry.StatusError)
			call.run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, nil, err
		}
		if !found {
			missing = append(missing, content.NewMissingContent(key.Kind(), key.ID()))
			continue
		}
		entries[i] = entry
	}
	return entries, missing, nil
}

// wrongEntry records the contract violation of a key that answered with another entry type and returns it.
func (call warlockCall) wrongEntry(ctx context.Context, key content.Key) error {
	err := srderr.ContractViolation(content.Port, "lookup", "the "+key.ID().String()+" key answered with another entry type")
	call.run.portFailure(ctx, err, srderr.ErrorType(err))
	return err
}
