package character

import (
	"context"
	"slices"
)

// ChoosePaladinOptionCommand is the input of ChoosePaladinOption.Run: the Paladin, the feats it holds and the choice. Construct it
// with a keyed literal. Run does not alter it.
type ChoosePaladinOptionCommand struct {
	Character Character
	Feats     TakenFeats
	Choice    PaladinChoice
}

// ChoosePaladinOptionResult is the closed set of outcomes of ChoosePaladinOption.Run. Variants: ChoosePaladinOptionAccepted,
// ChoosePaladinOptionRejected, ChoosePaladinOptionNotAPaladin.
type ChoosePaladinOptionResult interface{ isChoosePaladinOptionResult() }

// ChoosePaladinOptionAccepted is the outcome of a choice the rules allowed. Only Run builds one.
type ChoosePaladinOptionAccepted struct {
	character Character
	feats     TakenFeats
	state     PaladinState
	events    []Event
}

// Character returns the new Character.
func (r ChoosePaladinOptionAccepted) Character() Character { return r.character }

// Feats returns the new TakenFeats; unchanged for a Blessed Warrior choice.
func (r ChoosePaladinOptionAccepted) Feats() TakenFeats { return r.feats }

// State returns the new PaladinState.
func (r ChoosePaladinOptionAccepted) State() PaladinState { return r.state }

// Events returns a new slice of the events: those of taking the feat, then the PaladinChoiceMade event.
func (r ChoosePaladinOptionAccepted) Events() []Event { return slices.Clone(r.events) }

// ChoosePaladinOptionRejected is the outcome of a choice a rule refused: the Character and the feats stand and there are no events.
// Only Run builds one.
type ChoosePaladinOptionRejected struct{ reasons []PaladinReason }

// Reasons returns every reason that holds in declaration order, as a new slice. It is never empty.
func (r ChoosePaladinOptionRejected) Reasons() []PaladinReason { return slices.Clone(r.reasons) }

// ChoosePaladinOptionNotAPaladin is the outcome for a character with no Paladin levels.
type ChoosePaladinOptionNotAPaladin struct{}

func (ChoosePaladinOptionAccepted) isChoosePaladinOptionResult()    {}
func (ChoosePaladinOptionRejected) isChoosePaladinOptionResult()    {}
func (ChoosePaladinOptionNotAPaladin) isChoosePaladinOptionResult() {}

// ChoosePaladinOption is the use case that applies a Paladin's Fighting Style or Blessed Warrior choice (SRD 5.2.1, "Level 2:
// Fighting Style", p. 54). PaladinChoiceRules and the feat rules decide; it holds no rules logic. The feat and the cantrips are
// passed as content values by the caller, so it reads no content source. Create it with NewChoosePaladinOption; the zero value is
// not usable. It is safe for concurrent use only if its telemetry and clock are.
type ChoosePaladinOption struct{ paladinUse }

// NewChoosePaladinOption returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression.
func NewChoosePaladinOption(p PaladinProgression, opts ...Option) (ChoosePaladinOption, error) {
	u, err := newPaladinUse("choose a Paladin option", p, opts)
	return ChoosePaladinOption{u}, err
}

// Run applies cmd.Choice. Every reason that holds is reported, in declaration order.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a nil or
// unknown Choice, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. A refused choice is a result.
func (u ChoosePaladinOption) Run(ctx context.Context, cmd ChoosePaladinOptionCommand) (ChoosePaladinOptionResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	switch cmd.Choice.(type) {
	case ChoosePaladinFightingStyle, ChooseBlessedWarriorCantrips, ReplaceBlessedWarriorCantrip:
	default:
		return nil, invalidInput("choose a Paladin option: nil or unknown choice")
	}
	ctx, run := u.sig.start(ctx, SpanChoosePaladinOption, ValueClassPaladin, ValueFeatureChooseOption)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return ChoosePaladinOptionNotAPaladin{}, nil
	}
	rules := PaladinChoiceRules{}
	var (
		next    = cmd.Character
		feats   = cmd.Feats
		events  []Event
		reasons []PaladinReason
		choice  string
	)
	switch c := cmd.Choice.(type) {
	case ChoosePaladinFightingStyle:
		changed, rs, err := rules.Style(cmd.Character, cmd.Feats, u.progression, level, state, c.Feat)
		if err != nil {
			run.commandError(err)
			return nil, err
		}
		reasons, choice = rs, "fighting_style_feat"
		if len(rs) == 0 {
			next, feats, events = changed.Character(), changed.Feats(), changed.Events()
			state.style = PaladinFightingStyleFeat{Feat: c.Feat.ID()}
		}
	case ChooseBlessedWarriorCantrips:
		state, reasons = rules.BlessedWarrior(u.progression, level, state, c.Cantrips)
		choice = "blessed_warrior"
	case ReplaceBlessedWarriorCantrip:
		state, reasons = rules.Replace(state, c.Old, c.New)
		choice = "blessed_warrior_cantrip_replaced"
	}
	if len(reasons) > 0 {
		return ChoosePaladinOptionRejected{reasons: paladinReject(ctx, run, reasons)}, nil
	}
	next, err = paladinStore(run, next, state)
	if err != nil {
		return nil, err
	}
	events = append(events, PaladinChoiceMade{id: next.ID(), choice: choice})
	run.succeeded(ctx)
	return ChoosePaladinOptionAccepted{character: next, feats: feats, state: state, events: events}, nil
}
