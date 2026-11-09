package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ChooseRangerOptionCommand is the input of ChooseRangerOption.Run: the Ranger, the feats it holds, the subclass it holds ("" for
// none), its skill proficiencies and the choice. Construct it with a keyed literal. Run does not alter it.
type ChooseRangerOptionCommand struct {
	Character Character
	Feats     TakenFeats
	Subclass  content.SubclassID
	Skills    SkillProficiencies
	Choice    RangerChoice
}

// ChooseRangerOptionResult is the closed set of outcomes of ChooseRangerOption.Run. Variants: ChooseRangerOptionAccepted,
// ChooseRangerOptionRejected, ChooseRangerOptionNotARanger.
type ChooseRangerOptionResult interface{ isChooseRangerOptionResult() }

// ChooseRangerOptionAccepted is the outcome of a choice the rules allowed. Only Run builds one.
type ChooseRangerOptionAccepted struct{ accepted RangerChoiceAccepted }

// Character returns the new Character.
func (r ChooseRangerOptionAccepted) Character() Character { return r.accepted.character }

// Feats returns the new TakenFeats; unchanged unless a Fighting Style feat was taken.
func (r ChooseRangerOptionAccepted) Feats() TakenFeats { return r.accepted.feats }

// State returns the new RangerState.
func (r ChooseRangerOptionAccepted) State() RangerState { return r.accepted.state }

// Grant returns the FeatGrant for the host's TakeFeat after a RangerChooseFeat, and the zero grant after any other choice.
func (r ChooseRangerOptionAccepted) Grant() FeatGrant { return r.accepted.grant }

// Events returns a new slice of the events: those of taking the feat, then the RangerChoiceMade or RangerChoiceReplaced event.
func (r ChooseRangerOptionAccepted) Events() []Event { return r.accepted.Events() }

// ChooseRangerOptionRejected is the outcome of a choice a rule refused: the Character and the feats stand and there are no
// events. Only Run builds one.
type ChooseRangerOptionRejected struct{ reasons []RangerReason }

// Reasons returns every reason that holds in declaration order, as a new slice. It is never empty.
func (r ChooseRangerOptionRejected) Reasons() []RangerReason { return slices.Clone(r.reasons) }

// ChooseRangerOptionNotARanger is the outcome for a character with no Ranger levels.
type ChooseRangerOptionNotARanger struct{}

func (ChooseRangerOptionAccepted) isChooseRangerOptionResult()   {}
func (ChooseRangerOptionRejected) isChooseRangerOptionResult()   {}
func (ChooseRangerOptionNotARanger) isChooseRangerOptionResult() {}

// ChooseRangerOption is the use case that applies a Ranger's choice: Weapon Mastery, Deft Explorer, Fighting Style and Druidic
// Warrior, Expertise, Hunter’s Prey, Defensive Tactics and the feat grants (SRD 5.2.1, "Ranger Class Features", pp. 58-59, and
// "Ranger Subclass: Hunter", p. 61). RangerChoiceRules and the feat rules decide; it holds no rules logic. The feat, weapons and
// cantrips are passed as content values by the caller, so it reads no content source. Create it with NewChooseRangerOption; the
// zero value is not usable. It is safe for concurrent use only if its telemetry and clock are.
type ChooseRangerOption struct{ rangerUse }

// NewChooseRangerOption returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression.
func NewChooseRangerOption(p RangerProgression, opts ...Option) (ChooseRangerOption, error) {
	u, err := newRangerUse("choose a Ranger option", p, opts)
	return ChooseRangerOption{u}, err
}

// Run applies cmd.Choice. Every reason that holds is reported, in declaration order.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a nil or
// unknown Choice, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. A refused choice is a result.
func (u ChooseRangerOption) Run(ctx context.Context, cmd ChooseRangerOptionCommand) (ChooseRangerOptionResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	feature := rangerChoiceFeature(cmd.Choice)
	if feature == "" {
		return nil, invalidInput("choose a Ranger option: nil or unknown choice")
	}
	ctx, run := u.sig.start(ctx, SpanChooseRangerOption, ValueClassRanger, feature)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return ChooseRangerOptionNotARanger{}, nil
	}
	masteries, _, err := ReadFeatureState(cmd.Character.FeatureStates(), WeaponMasteryChoiceKey())
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	verdict, err := RangerChoiceRules{}.Decide(RangerChoiceFacts{Character: cmd.Character, Feats: cmd.Feats, Skills: cmd.Skills,
		Ranger: u.facts(level, state, cmd.Subclass), Masteries: masteries.Choice()}, cmd.Choice)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if refused, isRefused := verdict.(RangerChoiceRefused); isRefused {
		run.rejected(ctx, refused.reasons[0].String())
		return ChooseRangerOptionRejected{reasons: refused.Reasons()}, nil
	}
	run.succeeded(ctx)
	return ChooseRangerOptionAccepted{accepted: verdict.(RangerChoiceAccepted)}, nil
}
