package character

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ApplyFighterOptionChoiceCommand is the input of ApplyFighterOptionChoice.Run: the
// character, the feats it holds and the choice. Construct it with a keyed literal.
// Run does not alter it.
type ApplyFighterOptionChoiceCommand struct {
	Character Character
	Feats     TakenFeats
	Choice    FighterOptionChoice
}

// ApplyFighterOptionChoiceResult is the closed set of outcomes of
// ApplyFighterOptionChoice.Run. Variants: FighterOptionApplied,
// FighterOptionRejected, FighterOptionContentMissing.
type ApplyFighterOptionChoiceResult interface{ isApplyFighterOptionChoiceResult() }

// FighterOptionApplied is the outcome of a choice the rules allowed. Only Run builds one.
type FighterOptionApplied struct{ accepted FighterChoiceAccepted }

// Character returns the new Character.
func (r FighterOptionApplied) Character() Character { return r.accepted.character }

// Feats returns the new TakenFeats; unchanged for a Weapon Mastery choice.
func (r FighterOptionApplied) Feats() TakenFeats { return r.accepted.feats }

// Events returns a new slice of the events: those of taking the feat, then the Fighter event.
func (r FighterOptionApplied) Events() []Event { return r.accepted.Events() }

// FighterOptionRejected is the outcome of a choice a rule refused; the Character and
// the feats stand and there are no events. Only Run builds one.
type FighterOptionRejected struct{ problems []FighterProblem }

// Problems returns every problem, each with its ids, in the order of the reason table
// of the option, as a new slice. It is never empty.
func (r FighterOptionRejected) Problems() []FighterProblem { return slices.Clone(r.problems) }

// FighterOptionContentMissing is the outcome when the source lacks a feat or weapon
// the choice names; no rule was applied. Only Run builds one.
type FighterOptionContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of every entry not found, without repeats, ordered by
// content kind and then by id.
func (r FighterOptionContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (FighterOptionApplied) isApplyFighterOptionChoiceResult()        {}
func (FighterOptionRejected) isApplyFighterOptionChoiceResult()       {}
func (FighterOptionContentMissing) isApplyFighterOptionChoiceResult() {}

// ApplyFighterOptionChoice is the use case that applies a Fighter's Fighting Style or
// Weapon Mastery choice (SRD 5.2.1, "Level 1: Fighting Style", "Level 1: Weapon
// Mastery", p. 47-48, and "Level 7: Additional Fighting Style", p. 49): it looks up the
// feat or weapons the choice names, asks FighterChoiceRules and emits the
// srd5e.character.feature_used signals. It holds no rules logic and calls no other use
// case. Create it with NewApplyFighterOptionChoice; the zero value is not usable. It is
// safe for concurrent use only if its source, telemetry and clock are.
type ApplyFighterOptionChoice struct {
	progression FighterProgression
	source      ContentSource
	sig         featureSignals
	set         bool
}

// NewApplyFighterOptionChoice returns the use case reading content from source.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression or a nil source.
func NewApplyFighterOptionChoice(progression FighterProgression, source ContentSource, opts ...Option) (ApplyFighterOptionChoice, error) {
	if progression.IsZero() || source == nil {
		return ApplyFighterOptionChoice{}, invalidInput("apply a Fighter option without a Fighter progression or a content source")
	}
	return ApplyFighterOptionChoice{progression: progression, source: source, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// fighterFeatureOf is the srd5e.feature value of a choice.
func fighterFeatureOf(choice FighterOptionChoice) string {
	switch choice.(type) {
	case FighterChooseWeaponMasteries, FighterSwapWeaponMastery:
		return ValueFeatureWeaponMastery
	}
	return ValueFeatureFightingStyle
}

// Run applies cmd.Choice. A character with no Fighter levels is rejected with
// FighterNotAFighter before any lookup. Otherwise every feat and weapon the choice
// names is looked up first; any not found is FighterOptionContentMissing for all of
// them and no rule is applied, even FighterFeatureNotGained. Then FighterChoiceRules
// decides, reporting every problem together.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or a nil or unknown Choice, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "content_error") with a nil result when the source
// fails, panics or breaks its contract; the Character is unchanged and the failure
// is recorded once and never logged.
func (u ApplyFighterOptionChoice) Run(ctx context.Context, cmd ApplyFighterOptionChoiceCommand) (ApplyFighterOptionChoiceResult, error) {
	if err := fighterCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	keys, err := fighterChoiceKeys(cmd.Choice)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanApplyFighterOption, ValueClassFighter, fighterFeatureOf(cmd.Choice))
	defer run.end(ctx)
	fc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return FighterOptionRejected{problems: []FighterProblem{{Reason: reason}}}, nil
	}
	found, missing, err := lookupClassEntries(ctx, u.source, keys)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(missing) > 0 {
		run.contentMissing(ctx)
		return FighterOptionContentMissing{missing: missing}, nil
	}
	facts, err := fighterFactsOf(cmd, fc, found)
	if err != nil {
		if errorType := srderr.ErrorType(err); errorType != "" {
			run.portFailure(ctx, err, errorType)
		} else {
			run.commandError(err)
		}
		return nil, err
	}
	verdict, err := FighterChoiceRules{}.Decide(facts, cmd.Choice)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if refused, ok := verdict.(FighterChoiceRefused); ok {
		run.rejected(ctx, refused.problems[0].Reason.String())
		return FighterOptionRejected{problems: refused.Problems()}, nil
	}
	run.succeeded(ctx)
	return FighterOptionApplied{accepted: verdict.(FighterChoiceAccepted)}, nil
}

// fighterChoiceKeys returns the content keys the choice names, without repeats, in
// the order named. It returns an error matching srderr.ErrInvalidInput for a nil or
// unknown choice and for an invalid feat or item id.
func fighterChoiceKeys(choice FighterOptionChoice) ([]content.Key, error) {
	var keys []content.Key
	var feats []content.FeatID
	var items []content.ItemID
	switch c := choice.(type) {
	case FighterChooseFightingStyle:
		feats = append(feats, c.Choice.Feat())
	case FighterReplaceFightingStyle:
		feats = append(feats, c.Choice.Feat())
	case FighterChooseAdditionalFightingStyle:
		feats = append(feats, c.Choice.Feat())
	case FighterChooseWeaponMasteries:
		items = append(items, c.Kinds...)
	case FighterSwapWeaponMastery:
		items = append(items, c.Remove, c.Add) // the removed kind is only compared, but must be a valid id
	default:
		return nil, invalidInput("apply a Fighter option: nil or unknown choice")
	}
	for _, f := range feats {
		if !f.Valid() {
			return nil, invalidInput("apply a Fighter option naming an invalid feat id")
		}
		keys = append(keys, f.Key())
	}
	for _, i := range items {
		if !i.Valid() {
			return nil, invalidInput("apply a Fighter option naming an invalid item id")
		}
	}
	if swap, ok := choice.(FighterSwapWeaponMastery); ok {
		items = []content.ItemID{swap.Add} // only the added kind is looked up
	}
	for _, i := range items {
		keys = append(keys, i.Key())
	}
	return keys, nil
}

// lookupClassEntries looks up every distinct key and returns the entries found by
// key and the keys not found, without repeats, ordered by kind and then by id. A
// source failure, a panic and a broken contract come back as a *srderr.PortError.
func lookupClassEntries(ctx context.Context, source ContentSource, keys []content.Key) (map[content.Key]content.Entry, []content.MissingContent, error) {
	found := map[content.Key]content.Entry{}
	var missing []content.MissingContent
	for _, k := range keys {
		if _, done := found[k]; done || slices.ContainsFunc(missing, func(m content.MissingContent) bool { return m.Kind() == k.Kind() && m.ID() == k.ID() }) {
			continue
		}
		entry, ok, err := lookupEntry(ctx, source, k)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			found[k] = entry
		} else {
			missing = append(missing, content.NewMissingContent(k.Kind(), k.ID()))
		}
	}
	slices.SortFunc(missing, func(a, b content.MissingContent) int {
		return cmp.Or(cmp.Compare(a.Kind(), b.Kind()), strings.Compare(a.ID().String(), b.ID().String()))
	})
	return found, missing, nil
}

// fighterFactsOf resolves the facts FighterChoiceRules needs from the command, the
// Fighter context and the entries found. A feat key that answered another entry type
// is a contract violation of the source.
func fighterFactsOf(cmd ApplyFighterOptionChoiceCommand, fc fighterContext, found map[content.Key]content.Entry) (FighterChoiceFacts, error) {
	state, ok, err := ReadFeatureState(cmd.Character.FeatureStates(), WeaponMasteryChoiceKey())
	if err != nil {
		return FighterChoiceFacts{}, err
	}
	facts := FighterChoiceFacts{Character: cmd.Character, Feats: cmd.Feats, State: fc.state, Stats: fc.stats, Weapons: map[content.ItemID]content.Weapon{}}
	if ok {
		facts.Masteries = state.Choice()
	}
	for k, e := range found {
		switch v := e.(type) {
		case content.Feat:
			facts.Feat = v
		case content.Weapon:
			facts.Weapons[content.ItemID(k.ID().String())] = v
		}
	}
	if _, isStyle := cmd.Choice.(FighterChooseWeaponMasteries); !isStyle {
		if _, isSwap := cmd.Choice.(FighterSwapWeaponMastery); !isSwap && facts.Feat.ID() == "" {
			return FighterChoiceFacts{}, portViolation(content.Port, "lookup", "a feat key answered with another entry type")
		}
	}
	return facts, nil
}
