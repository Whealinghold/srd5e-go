package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ChooseWarlockMysticArcanumCommand is the input of ChooseWarlockMysticArcanum.Run. Spell is the arcanum spell chosen or the one that
// replaces another of its level; Replacements is how many arcanum spells the Warlock already replaced on gaining this level. Construct
// it with a keyed literal. Run does not alter it.
type ChooseWarlockMysticArcanumCommand struct {
	Character    Character
	Spell        content.SpellID
	Replacements int
}

// ChooseWarlockMysticArcanumResult is the closed set of outcomes of ChooseWarlockMysticArcanum.Run. Variants:
// ChooseWarlockMysticArcanumApplied, ChooseWarlockMysticArcanumRejected, ChooseWarlockMysticArcanumContentMissing.
type ChooseWarlockMysticArcanumResult interface{ isChooseWarlockMysticArcanumResult() }

// ChooseWarlockMysticArcanumApplied is the outcome of a choice the rules allowed.
type ChooseWarlockMysticArcanumApplied struct {
	character Character
	events    []Event
}

// Character returns the new Character holding the new WarlockState.
func (r ChooseWarlockMysticArcanumApplied) Character() Character { return r.character }

// Events returns a new slice holding the MysticArcanumChosen event.
func (r ChooseWarlockMysticArcanumApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseWarlockMysticArcanumRejected is the outcome of a choice a rule refused: the Character is unchanged.
type ChooseWarlockMysticArcanumRejected struct{ Rejection WarlockRejection }

// ChooseWarlockMysticArcanumContentMissing is the outcome when the spell is not in the source.
type ChooseWarlockMysticArcanumContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseWarlockMysticArcanumContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChooseWarlockMysticArcanumApplied) isChooseWarlockMysticArcanumResult()        {}
func (ChooseWarlockMysticArcanumRejected) isChooseWarlockMysticArcanumResult()       {}
func (ChooseWarlockMysticArcanumContentMissing) isChooseWarlockMysticArcanumResult() {}

// ChooseWarlockMysticArcanum is the use case that chooses or replaces a Mystic Arcanum spell (SRD 5.2.1, "Level 11: Mystic Arcanum", p. 72;
// class-warlock.md FR-042, FR-117, FR-118, FR-120): it loads the spell and calls ChooseMysticArcanum. Create it with
// NewChooseWarlockMysticArcanum; the zero value is not usable.
type ChooseWarlockMysticArcanum struct{ warlockUse }

// NewChooseWarlockMysticArcanum returns the use case. Options: WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewChooseWarlockMysticArcanum(source ContentSource, opts ...Option) (ChooseWarlockMysticArcanum, error) {
	u, err := newWarlockUse("choose Mystic Arcanum", source, true, opts)
	return ChooseWarlockMysticArcanum{u}, err
}

// Run chooses the spell. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character,
// and, recorded on the span, for an invalid spell id; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError
// when the source fails. A refusal is a ChooseWarlockMysticArcanumRejected result; a spell not in the source is a ContentMissing result.
func (u ChooseWarlockMysticArcanum) Run(ctx context.Context, cmd ChooseWarlockMysticArcanumCommand) (ChooseWarlockMysticArcanumResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseWarlockMysticArcanum, ValueFeatureMysticArcanum, cmd.Character, false)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return ChooseWarlockMysticArcanumRejected{Rejection: rej}, nil
	}
	if !cmd.Spell.Valid() {
		err := invalidInput("choose Mystic Arcanum with an invalid spell id")
		call.run.commandError(err)
		return nil, err
	}
	entries, missing, err := u.lookupAll(ctx, call, ValueFeatureMysticArcanum, []content.Key{cmd.Spell.Key()})
	if err != nil {
		return nil, err
	}
	if len(missing) != 0 {
		call.run.contentMissing(ctx)
		return ChooseWarlockMysticArcanumContentMissing{missing: missing}, nil
	}
	spell, ok := entries[0].(content.Spell)
	if !ok {
		return nil, call.wrongEntry(ctx, cmd.Spell.Key())
	}
	next, changed, rej, err := call.commit(ctx, cmd.Character, ChooseMysticArcanum(cmd.Character.ID(), call.state, call.profile, spell, cmd.Replacements))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return ChooseWarlockMysticArcanumRejected{Rejection: rej}, nil
	}
	return ChooseWarlockMysticArcanumApplied{character: next, events: changed.Events()}, nil
}
