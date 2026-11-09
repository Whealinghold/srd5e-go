package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// AttuneToMagicItemCommand is the input of AttuneToMagicItem.Do: the holder, the
// set of items attuned to now, the item, the candidate's eligibility facts, the
// host's judgement of a described prerequisite (PrerequisiteNotSupplied by
// default), the Short Rest focus and the Attunement limit. Construct it with a
// keyed literal. Do does not alter it.
type AttuneToMagicItemCommand struct {
	CharacterID Identifier
	Set         AttunementSet
	Item        AttunableItem
	Candidate   AttuneCandidate
	Judgement   PrerequisiteJudgement
	Focus       ShortRestFocus
	Limit       AttunementLimit
}

// AttuneToMagicItemResult is the closed set of outcomes of AttuneToMagicItem.Do.
// Variants: AttuneToMagicItemApplied, AttuneToMagicItemRejected.
type AttuneToMagicItemResult interface{ isAttuneToMagicItemResult() }

// AttuneToMagicItemApplied: the creature is attuned.
type AttuneToMagicItemApplied struct {
	set   AttunementSet
	event MagicItemAttuned
}

// Set returns the new set.
func (r AttuneToMagicItemApplied) Set() AttunementSet { return r.set }

// Events returns a new slice holding the MagicItemAttuned event.
func (r AttuneToMagicItemApplied) Events() []MagicItemEvent { return []MagicItemEvent{r.event} }

// AttuneToMagicItemRejected: the attempt was refused; the set is unchanged.
type AttuneToMagicItemRejected struct{ reason AttuneRejection }

// Reason returns the first rejection that applies.
func (r AttuneToMagicItemRejected) Reason() AttuneRejection { return r.reason }

func (AttuneToMagicItemApplied) isAttuneToMagicItemResult()  {}
func (AttuneToMagicItemRejected) isAttuneToMagicItemResult() {}

// AttuneToMagicItem is the use case that attunes a creature to an item at the end
// of a Short Rest. It wraps AttunementSet.Attune and holds no rule. Create it
// with NewAttuneToMagicItem; the zero value is not usable. It is safe for
// concurrent use only if its telemetry and clock are.
type AttuneToMagicItem struct {
	sig operationSignals
	set bool
}

// NewAttuneToMagicItem returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewAttuneToMagicItem(opts ...Option) AttuneToMagicItem {
	return AttuneToMagicItem{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Do attunes. A refused attempt is AttuneToMagicItemRejected, not an error.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case or an invalid command (see AttunementSet.Attune), and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u AttuneToMagicItem) Do(ctx context.Context, cmd AttuneToMagicItemCommand) (AttuneToMagicItemResult, error) {
	if !u.set {
		return nil, invalidInput("zero AttuneToMagicItem; use NewAttuneToMagicItem")
	}
	set, event, rejection, err := cmd.Set.Attune(cmd.CharacterID, cmd.Item, cmd.Candidate, cmd.Judgement, cmd.Focus, cmd.Limit)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAttuneToMagicItem, ValueOperationAttuneMagicItem)
	defer run.end(ctx)
	if rejection != 0 {
		run.finish(ctx, ValueResultRejected, rejection.String())
		return AttuneToMagicItemRejected{reason: rejection}, nil
	}
	run.finish(ctx, ValueResultChanged, "")
	return AttuneToMagicItemApplied{set: set, event: event}, nil
}

// EndMagicItemAttunementCommand is the input of EndMagicItemAttunement.Do: the
// holder, the set, the instance, the item's curse status and whether the curse
// was broken (CurseIntact by default), and the means of ending. Construct it with
// a keyed literal. Do does not alter it.
type EndMagicItemAttunementCommand struct {
	CharacterID Identifier
	Set         AttunementSet
	Instance    MagicItemInstanceID
	Curse       content.CurseStatus
	CurseBreak  CurseBreak
	Means       EndMeans
}

// EndMagicItemAttunementResult is the closed set of outcomes of
// EndMagicItemAttunement.Do. Variants: EndMagicItemAttunementApplied,
// EndMagicItemAttunementRejected.
type EndMagicItemAttunementResult interface{ isEndMagicItemAttunementResult() }

// EndMagicItemAttunementApplied: Attunement ended.
type EndMagicItemAttunementApplied struct {
	set    AttunementSet
	events []MagicItemAttunementEnded
}

// Set returns the new set.
func (r EndMagicItemAttunementApplied) Set() AttunementSet { return r.set }

// Events returns a new slice of one MagicItemAttunementEnded per ended item.
func (r EndMagicItemAttunementApplied) Events() []MagicItemEvent {
	out := make([]MagicItemEvent, 0, len(r.events))
	for _, e := range r.events {
		out = append(out, e)
	}
	return slices.Clip(out)
}

// EndMagicItemAttunementRejected: the end was refused; the set is unchanged.
type EndMagicItemAttunementRejected struct{ reason EndRejection }

// Reason returns the first rejection that applies.
func (r EndMagicItemAttunementRejected) Reason() EndRejection { return r.reason }

func (EndMagicItemAttunementApplied) isEndMagicItemAttunementResult()  {}
func (EndMagicItemAttunementRejected) isEndMagicItemAttunementResult() {}

// EndMagicItemAttunement is the use case that ends Attunement to an item. It
// wraps AttunementSet.End and holds no rule. Create it with
// NewEndMagicItemAttunement; the zero value is not usable. It is safe for
// concurrent use only if its telemetry and clock are.
type EndMagicItemAttunement struct {
	sig operationSignals
	set bool
}

// NewEndMagicItemAttunement returns the use case. Options: WithTelemetry,
// WithClock; the others are ignored.
func NewEndMagicItemAttunement(opts ...Option) EndMagicItemAttunement {
	return EndMagicItemAttunement{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Do ends Attunement. A refused end is EndMagicItemAttunementRejected, not an
// error.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case or an invalid command (see AttunementSet.End), and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u EndMagicItemAttunement) Do(ctx context.Context, cmd EndMagicItemAttunementCommand) (EndMagicItemAttunementResult, error) {
	if !u.set {
		return nil, invalidInput("zero EndMagicItemAttunement; use NewEndMagicItemAttunement")
	}
	set, events, rejection, err := cmd.Set.End(cmd.CharacterID, cmd.Instance, cmd.Curse, cmd.CurseBreak, cmd.Means)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanEndMagicItemAttunement, ValueOperationEndMagicItemAttunement)
	defer run.end(ctx)
	if rejection != 0 {
		run.finish(ctx, ValueResultRejected, rejection.String())
		return EndMagicItemAttunementRejected{reason: rejection}, nil
	}
	run.finish(ctx, ValueResultChanged, "")
	return EndMagicItemAttunementApplied{set: set, events: events}, nil
}
