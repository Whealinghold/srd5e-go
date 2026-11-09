package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ApplyBarbarianOptionChoiceCommand is the input of ApplyBarbarianOptionChoice.Run: the character and
// the choice. Construct it with a keyed literal. Run does not alter it.
type ApplyBarbarianOptionChoiceCommand struct {
	Character Character
	Choice    BarbarianOptionChoice
}

// ApplyBarbarianOptionChoiceResult is the closed set of outcomes of ApplyBarbarianOptionChoice.Run.
// Variants: BarbarianOptionApplied, BarbarianOptionRejected, BarbarianOptionContentMissing.
type ApplyBarbarianOptionChoiceResult interface{ isApplyBarbarianOptionChoiceResult() }

// BarbarianOptionApplied is the outcome of a choice the rules allowed. Only Run builds one.
type BarbarianOptionApplied struct{ accepted BarbarianChoiceAccepted }

// Character returns the new Character.
func (r BarbarianOptionApplied) Character() Character { return r.accepted.character }

// Events returns a new slice of the events.
func (r BarbarianOptionApplied) Events() []Event { return r.accepted.Events() }

// BarbarianOptionRejected is the outcome of a choice a rule refused; the Character stands and there
// are no events. Only Run builds one.
type BarbarianOptionRejected struct{ problems []BarbarianProblem }

// Problems returns every problem, each with its weapon, in the order of the reason table of the
// option, as a new slice. It is never empty.
func (r BarbarianOptionRejected) Problems() []BarbarianProblem { return slices.Clone(r.problems) }

// BarbarianOptionContentMissing is the outcome when the source lacks a weapon the choice names; no
// rule was applied. Only Run builds one.
type BarbarianOptionContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of every entry not found, without repeats, ordered by id.
func (r BarbarianOptionContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (BarbarianOptionApplied) isApplyBarbarianOptionChoiceResult()        {}
func (BarbarianOptionRejected) isApplyBarbarianOptionChoiceResult()       {}
func (BarbarianOptionContentMissing) isApplyBarbarianOptionChoiceResult() {}

// ApplyBarbarianOptionChoice is the use case that applies a Barbarian's Weapon Mastery choice (SRD 5.2.1,
// "Level 1: Weapon Mastery", p. 29): it looks up the weapons the choice names, asks
// BarbarianChoiceRules and emits the srd5e.character.feature_used signals. It holds no rules logic and
// calls no other use case. Create it with NewApplyBarbarianOptionChoice; the zero value is not usable.
// It is safe for concurrent use only if its source, telemetry and clock are.
type ApplyBarbarianOptionChoice struct {
	progression BarbarianProgression
	source      ContentSource
	sig         featureSignals
	set         bool
}

// NewApplyBarbarianOptionChoice returns the use case reading content from source. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for the zero progression or a nil source.
func NewApplyBarbarianOptionChoice(progression BarbarianProgression, source ContentSource, opts ...Option) (ApplyBarbarianOptionChoice, error) {
	if progression.IsZero() || source == nil {
		return ApplyBarbarianOptionChoice{}, invalidInput("apply a Barbarian option without a Barbarian progression or a content source")
	}
	return ApplyBarbarianOptionChoice{progression: progression, source: source, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run applies cmd.Choice. A character with no Barbarian levels is rejected with
// BarbarianNotABarbarian before any lookup. Otherwise every weapon the choice names is looked up first;
// any not found is BarbarianOptionContentMissing for all of them and no rule is applied. Then
// BarbarianChoiceRules decides, reporting every problem together.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero
// Character or a nil or unknown Choice or an invalid item id, and ctx.Err() unwrapped, before any
// signal, when ctx is already cancelled. It returns a *srderr.PortError (error type "content_error")
// with a nil result when the source fails, panics or breaks its contract; the Character is unchanged
// and the failure is recorded once and never logged.
func (u ApplyBarbarianOptionChoice) Run(ctx context.Context, cmd ApplyBarbarianOptionChoiceCommand) (ApplyBarbarianOptionChoiceResult, error) {
	if err := barbarianCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	keys, err := barbarianChoiceKeys(cmd.Choice)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanApplyBarbarianOption, ValueClassBarbarian, ValueFeatureBarbarianWeaponMastery)
	defer run.end(ctx)
	bc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return BarbarianOptionRejected{problems: []BarbarianProblem{{Reason: reason}}}, nil
	}
	found, missing, err := lookupClassEntries(ctx, u.source, keys)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(missing) > 0 {
		run.contentMissing(ctx)
		return BarbarianOptionContentMissing{missing: missing}, nil
	}
	facts := BarbarianChoiceFacts{Character: cmd.Character, Stats: bc.stats, Weapons: map[content.ItemID]content.Weapon{}}
	state, ok, err := ReadFeatureState(cmd.Character.FeatureStates(), WeaponMasteryChoiceKey())
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if ok {
		facts.Masteries = state.Choice()
	}
	for k, e := range found {
		if w, isWeapon := e.(content.Weapon); isWeapon {
			facts.Weapons[content.ItemID(k.ID().String())] = w
		}
	}
	verdict, err := BarbarianChoiceRules{}.Decide(facts, cmd.Choice)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if refused, isRefused := verdict.(BarbarianChoiceRefused); isRefused {
		run.rejected(ctx, refused.problems[0].Reason.String())
		return BarbarianOptionRejected{problems: refused.Problems()}, nil
	}
	run.succeeded(ctx)
	return BarbarianOptionApplied{accepted: verdict.(BarbarianChoiceAccepted)}, nil
}

// barbarianChoiceKeys returns the content keys the choice names, without repeats, in the order named.
// It returns an error matching srderr.ErrInvalidInput for a nil or unknown choice and for an invalid
// item id.
func barbarianChoiceKeys(choice BarbarianOptionChoice) ([]content.Key, error) {
	var items []content.ItemID
	switch c := choice.(type) {
	case BarbarianChooseWeaponMasteries:
		items = append(items, c.Kinds...)
	case BarbarianSwapWeaponMastery:
		if !c.Remove.Valid() {
			return nil, invalidInput("apply a Barbarian option naming an invalid item id") // the removed kind is only compared, but must be a valid id
		}
		items = append(items, c.Add) // only the added kind is looked up
	default:
		return nil, invalidInput("apply a Barbarian option: nil or unknown choice")
	}
	keys := make([]content.Key, 0, len(items))
	for _, i := range items {
		if !i.Valid() {
			return nil, invalidInput("apply a Barbarian option naming an invalid item id")
		}
		keys = append(keys, i.Key())
	}
	return keys, nil
}
