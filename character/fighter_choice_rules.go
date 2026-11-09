package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FighterOptionChoice is a Fighter-specific choice a player makes, the command of
// ApplyFighterOptionChoice: a closed set. It has no variant for Ability Score
// Improvement, Epic Boon or subclass, which belong to level advancement. Variants:
// FighterChooseFightingStyle, FighterReplaceFightingStyle,
// FighterChooseAdditionalFightingStyle, FighterChooseWeaponMasteries,
// FighterSwapWeaponMastery.
type FighterOptionChoice interface{ isFighterOptionChoice() }

// FighterChooseFightingStyle is the choice of the level 1 Fighting Style feat (SRD
// 5.2.1, "Level 1: Fighting Style", p. 47). Construct it with a keyed literal.
type FighterChooseFightingStyle struct{ Choice FightingStyleChoice }

// FighterReplaceFightingStyle replaces the level 1 Fighting Style with another
// Fighting Style feat after gaining a Fighter level. It names no replaced feat:
// the replaced one is always the level 1 style. LevelGained must be the event of
// gaining the current Fighter level by this character. Construct it with a keyed
// literal.
type FighterReplaceFightingStyle struct {
	Choice      FightingStyleChoice
	LevelGained LevelGained
}

// FighterChooseAdditionalFightingStyle is the Champion's choice of another Fighting
// Style feat at Fighter level 7 (SRD 5.2.1, "Level 7: Additional Fighting Style",
// p. 49). Subclass is the chosen subclass, empty for none. Construct it with a
// keyed literal.
type FighterChooseAdditionalFightingStyle struct {
	Choice   FightingStyleChoice
	Subclass content.SubclassID
}

// FighterChooseWeaponMasteries is the choice of the kinds of Simple or Martial
// weapon to have Weapon Mastery with (SRD 5.2.1, "Level 1: Weapon Mastery", p. 48):
// all the kinds, in order, exactly as many as the level allows. When a choice is
// already held the list must keep every kind of it and add the new ones. Construct
// it with a keyed literal.
type FighterChooseWeaponMasteries struct{ Kinds []content.ItemID }

// FighterSwapWeaponMastery changes one chosen kind to another after a Long Rest.
// RestApplied must be the event of a Long Rest of this character. Construct it
// with a keyed literal.
type FighterSwapWeaponMastery struct {
	Remove, Add content.ItemID
	RestApplied RestApplied
}

func (FighterChooseFightingStyle) isFighterOptionChoice()           {}
func (FighterReplaceFightingStyle) isFighterOptionChoice()          {}
func (FighterChooseAdditionalFightingStyle) isFighterOptionChoice() {}
func (FighterChooseWeaponMasteries) isFighterOptionChoice()         {}
func (FighterSwapWeaponMastery) isFighterOptionChoice()             {}

// FighterProblem is one rule problem of an option choice, with the ids that cause
// it: Feat for a feat problem, Item for a weapon problem, empty where the reason
// names none. It has no display text. Construct it with a keyed literal.
type FighterProblem struct {
	Reason FighterReason
	Feat   content.FeatID
	Item   content.ItemID
}

// FighterChoiceVerdict is the closed outcome of FighterChoiceRules.Decide.
// Variants: FighterChoiceAccepted, FighterChoiceRefused.
type FighterChoiceVerdict interface{ isFighterChoiceVerdict() }

// FighterChoiceAccepted is a choice the rules allow: the new Character (with its
// FighterState and Weapon Mastery choice), the new TakenFeats and the events in
// order: those of taking the feat, then the Fighter event. Only Decide builds one.
type FighterChoiceAccepted struct {
	character Character
	feats     TakenFeats
	events    []Event
}

// Character returns the new Character.
func (a FighterChoiceAccepted) Character() Character { return a.character }

// Feats returns the new TakenFeats; the same feats for a Weapon Mastery choice.
func (a FighterChoiceAccepted) Feats() TakenFeats { return a.feats }

// Events returns a new slice of the events.
func (a FighterChoiceAccepted) Events() []Event { return slices.Clone(a.events) }

// FighterChoiceRefused is a choice a rule refused; nothing changed and there are no
// events. Only Decide builds one.
type FighterChoiceRefused struct{ problems []FighterProblem }

// Problems returns every problem found, in the order of the reason table of the
// option, as a new slice. It is never empty.
func (r FighterChoiceRefused) Problems() []FighterProblem { return slices.Clone(r.problems) }

func (FighterChoiceAccepted) isFighterChoiceVerdict() {}
func (FighterChoiceRefused) isFighterChoiceVerdict()  {}

// FighterChoiceFacts is everything FighterChoiceRules needs, resolved by the use
// case: the character, its feats, its FighterState and Weapon Mastery choice, the
// stats of its Fighter level and the content the choice names. Construct it with a
// keyed literal.
type FighterChoiceFacts struct {
	Character Character
	Feats     TakenFeats
	State     FighterState
	Masteries WeaponMasteryChoice
	Stats     FighterLevelStats
	// Feat is the content of the feat a style choice names; the zero Feat for the other choices.
	Feat content.Feat
	// Weapons holds the weapons the mastery choices name that exist; an id missing from it is not a weapon.
	Weapons map[content.ItemID]content.Weapon
}

// FighterChoiceRules decides the Fighting Style and Weapon Mastery choices of a
// Fighter. It delegates to the rules that own them (FeatTakingRules,
// FeatReplacement, CreateWeaponMasteryChoice, ExtendWeaponMasteryChoice,
// ChangeWeaponMasteryChoice) and only maps their outcomes to FighterReason; it
// restates none of them. It is stateless; the zero value is ready to use. Safe for
// concurrent use.
type FighterChoiceRules struct{}

const (
	fighterStyleFeature  content.FeatureID = "fighter_fighting_style"
	championStyleFeature content.FeatureID = "champion_additional_fighting_style"
)

// fighterReasonOrder returns the reasons choice reports, in the order they are reported.
func fighterReasonOrder(choice FighterOptionChoice) []FighterReason {
	switch choice.(type) {
	case FighterChooseFightingStyle:
		return []FighterReason{FighterFightingStyleAlreadyChosen, FighterNotAFightingStyleFeat, FighterFeatNotRepeatable}
	case FighterReplaceFightingStyle:
		return []FighterReason{FighterNoStyleToReplace, FighterEventNotForCharacter, FighterNoLevelGained, FighterNotAFightingStyleFeat, FighterSameFeat, FighterFeatNotRepeatable}
	case FighterChooseAdditionalFightingStyle:
		return []FighterReason{FighterFeatureNotGained, FighterAdditionalStyleAlreadyChosen, FighterNotAFightingStyleFeat, FighterFeatNotRepeatable}
	case FighterChooseWeaponMasteries:
		return []FighterReason{FighterWrongMasteryCount, FighterDuplicateMastery, FighterNotSimpleOrMartialWeapon, FighterDropsExistingMastery}
	}
	return []FighterReason{FighterEventNotForCharacter, FighterNoLongRestFinished, FighterNotCurrentlyChosen, FighterAlreadyChosen, FighterNotSimpleOrMartialWeapon}
}

// Decide checks choice against facts and, when no rule refuses it, applies it. The
// problems of one choice are reported together, each with the ids that cause it, in
// the order of the reason table of the option.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or
// stats, a nil or unknown choice, a style choice whose facts hold no feat of the
// chosen id, and a Character holding a state of another type under the Fighter
// keys. The inputs are unchanged. It never panics.
func (r FighterChoiceRules) Decide(facts FighterChoiceFacts, choice FighterOptionChoice) (FighterChoiceVerdict, error) {
	if facts.Character.IsZero() || facts.Stats.IsZero() || choice == nil {
		return nil, invalidInput("Fighter choice with a zero character or stats, or no choice")
	}
	var (
		verdict FighterChoiceVerdict
		err     error
	)
	switch c := choice.(type) {
	case FighterChooseFightingStyle:
		verdict, err = r.chooseStyle(facts, c)
	case FighterReplaceFightingStyle:
		verdict, err = r.replaceStyle(facts, c)
	case FighterChooseAdditionalFightingStyle:
		verdict, err = r.additionalStyle(facts, c)
	case FighterChooseWeaponMasteries:
		verdict, err = r.chooseMasteries(facts, c)
	case FighterSwapWeaponMastery:
		verdict, err = r.swapMastery(facts, c)
	default:
		return nil, invalidInput("unknown Fighter option choice")
	}
	if err != nil {
		return nil, err
	}
	if refused, ok := verdict.(FighterChoiceRefused); ok {
		return FighterChoiceRefused{problems: sortFighterProblems(refused.problems, fighterReasonOrder(choice))}, nil
	}
	return verdict, nil
}

// sortFighterProblems orders problems by the position of their reason in order, keeping
// the order of equal reasons.
func sortFighterProblems(problems []FighterProblem, order []FighterReason) []FighterProblem {
	out := slices.Clone(problems)
	slices.SortStableFunc(out, func(a, b FighterProblem) int {
		return slices.Index(order, a.Reason) - slices.Index(order, b.Reason)
	})
	return out
}

func refuseFighterChoice(problems ...FighterProblem) (FighterChoiceVerdict, error) {
	return FighterChoiceRefused{problems: problems}, nil
}

// styleTake takes or replaces the feat of facts.Feat as a Fighting Style under
// source, with the held Fighting Style feature. replaced is the feat to replace, or
// "" to take.
func (FighterChoiceRules) styleTake(facts FighterChoiceFacts, source FeatSource, replaced content.FeatID, scope FeatGrantScope) (FeatTaking, error) {
	grant, err := NewFeatGrant(source, scope)
	if err != nil {
		return nil, err
	}
	held, err := NewHeldFeatures(fighterStyleFeature)
	if err != nil {
		return nil, err
	}
	if replaced != "" {
		return FeatReplacement{}.Replace(facts.Character, facts.Feats, replaced, facts.Feat, grant, NoFeatChoices{}, held, nil)
	}
	return FeatTakingRules{}.Take(facts.Character, facts.Feats, facts.Feat, grant, NoFeatChoices{}, held, nil)
}

// styleOutcome takes (or replaces) the style feat under the Fighting Style scope
// and returns the reasons the feat rules refuse it with, or the change. A feat
// outside the scope also learns whether it would be refused as not Repeatable, so
// both are reported together.
func (r FighterChoiceRules) styleOutcome(facts FighterChoiceFacts, source FeatSource, replaced content.FeatID) ([]FighterReason, FeatTakingChanged, error) {
	if !facts.Feat.ID().Valid() {
		return nil, FeatTakingChanged{}, invalidInput("Fighting Style choice without the feat content")
	}
	taking, err := r.styleTake(facts, source, replaced, FeatFromCategory{Category: content.FeatFightingStyle})
	if err != nil {
		return nil, FeatTakingChanged{}, err
	}
	switch t := taking.(type) {
	case FeatTakingChanged:
		return nil, t, nil
	case FeatTakingRejected:
		switch t.Rejection.(type) {
		case FeatNotRepeatable:
			return []FighterReason{FighterFeatNotRepeatable}, FeatTakingChanged{}, nil
		case FeatNotHeld:
			return []FighterReason{FighterNoStyleToReplace}, FeatTakingChanged{}, nil
		case FeatOutsideGrant:
			reasons := []FighterReason{FighterNotAFightingStyleFeat}
			again, err := r.styleTake(facts, source, replaced, AnyFeat{})
			if err != nil {
				return nil, FeatTakingChanged{}, err
			}
			if rej, ok := again.(FeatTakingRejected); ok {
				if _, repeat := rej.Rejection.(FeatNotRepeatable); repeat {
					reasons = append(reasons, FighterFeatNotRepeatable)
				}
			}
			return reasons, FeatTakingChanged{}, nil
		}
		// Any other refusal (a feat with no behaviour, an unmet prerequisite) means the
		// feat is not one a Fighting Style can be taken with.
		return []FighterReason{FighterNotAFightingStyleFeat}, FeatTakingChanged{}, nil
	}
	return nil, FeatTakingChanged{}, invalidInput("unknown feat taking outcome")
}

func fighterFeatProblems(reasons []FighterReason, feat content.FeatID) []FighterProblem {
	out := make([]FighterProblem, len(reasons))
	for i, r := range reasons {
		out[i] = FighterProblem{Reason: r, Feat: feat}
	}
	return out
}

// accept stores state and, for masteries, the choice in the new Character and
// builds the accepted verdict.
func acceptFighterChoice(c Character, feats TakenFeats, state FighterState, events []Event) (FighterChoiceVerdict, error) {
	next, err := WithFeatureState(c, FighterStateKey(), state)
	if err != nil {
		return nil, err
	}
	return FighterChoiceAccepted{character: next, feats: feats, events: events}, nil
}

func (r FighterChoiceRules) chooseStyle(facts FighterChoiceFacts, c FighterChooseFightingStyle) (FighterChoiceVerdict, error) {
	source := ClassFeatureFeat{Class: FighterClassID, Feature: fighterStyleFeature}
	reasons, change, err := r.styleOutcome(facts, source, "")
	if err != nil {
		return nil, err
	}
	var problems []FighterProblem
	if facts.State.fightingStyle != "" {
		problems = append(problems, FighterProblem{Reason: FighterFightingStyleAlreadyChosen, Feat: facts.State.fightingStyle})
		// A second choice is refused whatever repeatability says.
		reasons = slices.DeleteFunc(reasons, func(r FighterReason) bool { return r == FighterFeatNotRepeatable })
	}
	problems = append(problems, fighterFeatProblems(reasons, c.Choice.Feat())...)
	if len(problems) > 0 {
		return refuseFighterChoice(problems...)
	}
	state := facts.State
	state.fightingStyle = c.Choice.Feat()
	events := append(change.Events(), Event(FighterFightingStyleChosen{id: facts.Character.ID(), feat: state.fightingStyle}))
	return acceptFighterChoice(change.Character(), change.Feats(), state, events)
}

func (r FighterChoiceRules) replaceStyle(facts FighterChoiceFacts, c FighterReplaceFightingStyle) (FighterChoiceVerdict, error) {
	var problems []FighterProblem
	old := facts.State.fightingStyle
	if old == "" {
		problems = append(problems, FighterProblem{Reason: FighterNoStyleToReplace})
	}
	if c.LevelGained.CharacterID() != facts.Character.ID() {
		problems = append(problems, FighterProblem{Reason: FighterEventNotForCharacter})
	} else if c.LevelGained.Class() != FighterClassID || c.LevelGained.ClassLevel() != facts.Stats.level.n {
		problems = append(problems, FighterProblem{Reason: FighterNoLevelGained})
	}
	// With no style to replace the feat rules still run as a plain take, so every problem is reported together.
	source := ClassFeatureFeat{Class: FighterClassID, Feature: fighterStyleFeature}
	reasons, change, err := r.styleOutcome(facts, source, old)
	if err != nil {
		return nil, err
	}
	problems = append(problems, fighterFeatProblems(reasons, c.Choice.Feat())...)
	if old != "" && c.Choice.Feat() == old {
		problems = append(problems, FighterProblem{Reason: FighterSameFeat, Feat: old})
	}
	if len(problems) > 0 {
		return refuseFighterChoice(problems...)
	}
	state := facts.State
	state.fightingStyle = c.Choice.Feat()
	events := append(change.Events(), Event(FighterFightingStyleReplaced{id: facts.Character.ID(), old: old, feat: state.fightingStyle}))
	return acceptFighterChoice(change.Character(), change.Feats(), state, events)
}

func (r FighterChoiceRules) additionalStyle(facts FighterChoiceFacts, c FighterChooseAdditionalFightingStyle) (FighterChoiceVerdict, error) {
	source := ClassFeatureFeat{Class: FighterClassID, Feature: championStyleFeature}
	reasons, change, err := r.styleOutcome(facts, source, "")
	if err != nil {
		return nil, err
	}
	var problems []FighterProblem
	if !isChampion(c.Subclass) || facts.Stats.level.n < championStyleLevel {
		problems = append(problems, FighterProblem{Reason: FighterFeatureNotGained})
	}
	if facts.State.additionalFightingStyle != "" {
		problems = append(problems, FighterProblem{Reason: FighterAdditionalStyleAlreadyChosen, Feat: facts.State.additionalFightingStyle})
	}
	problems = append(problems, fighterFeatProblems(reasons, c.Choice.Feat())...)
	if len(problems) > 0 {
		return refuseFighterChoice(problems...)
	}
	state := facts.State
	state.additionalFightingStyle = c.Choice.Feat()
	events := append(change.Events(), Event(FighterFightingStyleChosen{id: facts.Character.ID(), feat: state.additionalFightingStyle, additional: true}))
	return acceptFighterChoice(change.Character(), change.Feats(), state, events)
}

// fighterMasteryReason maps the problem kind of the Weapon Mastery rules to a Fighter reason.
func fighterMasteryReason(k MasteryProblemKind) FighterReason {
	switch k {
	case MasteryProblemNotMelee, MasteryProblemNotProficient, MasteryProblemWeaponNotResolved:
		return FighterNotSimpleOrMartialWeapon // the last is an item found that is not a weapon
	case MasteryProblemDuplicateWeapon:
		return FighterDuplicateMastery
	case MasteryProblemDropsExistingMastery:
		return FighterDropsExistingMastery
	case MasteryProblemAlreadyChosen:
		return FighterAlreadyChosen
	case MasteryProblemNotCurrentlyChosen:
		return FighterNotCurrentlyChosen
	case MasteryProblemWrongMoment:
		return FighterNoLongRestFinished
	case MasteryProblemWrongCount, MasteryProblemChangesMoreThanOne:
		return FighterWrongMasteryCount
	}
	return FighterNotSimpleOrMartialWeapon
}

func fighterMasteryProblems(in []MasteryProblem) []FighterProblem {
	out := make([]FighterProblem, len(in))
	for i, p := range in {
		out[i] = FighterProblem{Reason: fighterMasteryReason(p.Kind), Item: p.WeaponID}
	}
	return out
}

// acceptMasteries stores choice as the Weapon Mastery choice of the new Character.
func acceptMasteries(facts FighterChoiceFacts, choice WeaponMasteryChoice, event Event) (FighterChoiceVerdict, error) {
	next, err := WithFeatureState(facts.Character, WeaponMasteryChoiceKey(), NewWeaponMasteryChoiceState(choice))
	if err != nil {
		return nil, err
	}
	return FighterChoiceAccepted{character: next, feats: facts.Feats, events: []Event{event}}, nil
}

func (FighterChoiceRules) chooseMasteries(facts FighterChoiceFacts, c FighterChooseWeaponMasteries) (FighterChoiceVerdict, error) {
	// Extending an empty choice is creating one, so one call serves level 1 and every later level.
	res, err := ExtendWeaponMasteryChoice(facts.Masteries, c.Kinds, facts.Stats.WeaponMastery(), facts.Weapons, SimpleOrMartialEligibility{})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case WeaponMasteryChoiceRejected:
		return refuseFighterChoice(fighterMasteryProblems(v.problems)...)
	case WeaponMasteryChoiceAccepted:
		return acceptMasteries(facts, v.choice, FighterWeaponMasteriesChosen{id: facts.Character.ID(), kinds: v.choice.Kinds()})
	}
	return nil, invalidInput("unknown Weapon Mastery choice result")
}

func (FighterChoiceRules) swapMastery(facts FighterChoiceFacts, c FighterSwapWeaponMastery) (FighterChoiceVerdict, error) {
	moment := MasteryChangeLongRest
	var problems []FighterProblem
	if c.RestApplied.CharacterID() != facts.Character.ID() {
		problems = append(problems, FighterProblem{Reason: FighterEventNotForCharacter})
	} else if c.RestApplied.BenefitsOf() != LongRest {
		moment = MasteryChangeShortRest
	}
	res, err := ChangeWeaponMasteryChoice(WeaponMasteryChange{
		Current: facts.Masteries, Swaps: []MasterySwap{{Remove: c.Remove, Add: c.Add}}, Moment: moment, Rule: ReplaceOneKind,
		Resolved: facts.Weapons, Eligibility: SimpleOrMartialEligibility{},
	})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case WeaponMasteryChoiceRejected:
		problems = append(problems, fighterMasteryProblems(v.problems)...)
	case WeaponMasteryChoiceAccepted:
		if len(problems) == 0 {
			return acceptMasteries(facts, v.choice, FighterWeaponMasterySwapped{id: facts.Character.ID(), removed: c.Remove, added: c.Add})
		}
	}
	return refuseFighterChoice(problems...)
}
