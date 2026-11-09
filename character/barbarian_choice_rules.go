package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// BarbarianOptionChoice is a Barbarian-specific choice a player makes, the command of
// ApplyBarbarianOptionChoice: a closed set. It has no variant for Ability Score Improvement, Epic
// Boon, subclass or the Primal Knowledge skill, which belong to level advancement and character
// creation. Variants: BarbarianChooseWeaponMasteries, BarbarianSwapWeaponMastery.
type BarbarianOptionChoice interface{ isBarbarianOptionChoice() }

// BarbarianChooseWeaponMasteries is the choice of the kinds of Simple or Martial Melee weapon to
// have Weapon Mastery with (SRD 5.2.1, "Level 1: Weapon Mastery", p. 29): all the kinds, in order,
// exactly as many as the level allows. When a choice is already held the list must keep every kind
// of it and add the new ones. Construct it with a keyed literal.
type BarbarianChooseWeaponMasteries struct{ Kinds []content.ItemID }

// BarbarianSwapWeaponMastery changes one chosen kind to another after a Long Rest. RestApplied must
// be the event of a Long Rest of this character. Construct it with a keyed literal.
type BarbarianSwapWeaponMastery struct {
	Remove, Add content.ItemID
	RestApplied RestApplied
}

func (BarbarianChooseWeaponMasteries) isBarbarianOptionChoice() {}
func (BarbarianSwapWeaponMastery) isBarbarianOptionChoice()     {}

// BarbarianProblem is one rule problem of an option choice, with the weapon that causes it: Item for a
// weapon problem, empty where the reason names none. It has no display text. Construct it with a keyed
// literal.
type BarbarianProblem struct {
	Reason BarbarianReason
	Item   content.ItemID
}

// BarbarianChoiceVerdict is the closed outcome of BarbarianChoiceRules.Decide. Variants:
// BarbarianChoiceAccepted, BarbarianChoiceRefused.
type BarbarianChoiceVerdict interface{ isBarbarianChoiceVerdict() }

// BarbarianChoiceAccepted is a choice the rules allow: the new Character (with its Weapon Mastery
// choice) and the event. Only Decide builds one.
type BarbarianChoiceAccepted struct {
	character Character
	events    []Event
}

// Character returns the new Character.
func (a BarbarianChoiceAccepted) Character() Character { return a.character }

// Events returns a new slice of the events.
func (a BarbarianChoiceAccepted) Events() []Event { return slices.Clone(a.events) }

// BarbarianChoiceRefused is a choice a rule refused; nothing changed and there are no events. Only
// Decide builds one.
type BarbarianChoiceRefused struct{ problems []BarbarianProblem }

// Problems returns every problem found, in the order of the reason table of the option, as a new slice.
// It is never empty.
func (r BarbarianChoiceRefused) Problems() []BarbarianProblem { return slices.Clone(r.problems) }

func (BarbarianChoiceAccepted) isBarbarianChoiceVerdict() {}
func (BarbarianChoiceRefused) isBarbarianChoiceVerdict()  {}

// BarbarianChoiceFacts is everything BarbarianChoiceRules needs, resolved by the use case: the
// character, the stats of its Barbarian level, its Weapon Mastery choice and the weapons the choice
// names. Construct it with a keyed literal.
type BarbarianChoiceFacts struct {
	Character Character
	Stats     BarbarianLevelStats
	Masteries WeaponMasteryChoice
	// Weapons holds the weapons the choice names that exist; an id missing from it is not a weapon.
	Weapons map[content.ItemID]content.Weapon
}

// BarbarianChoiceRules decides the Weapon Mastery choices of a Barbarian. It delegates to the rules that
// own them (ExtendWeaponMasteryChoice, ChangeWeaponMasteryChoice with SimpleOrMartialMeleeEligibility)
// and only maps their outcomes to BarbarianReason; it restates none of them. It is stateless; the zero
// value is ready to use. Safe for concurrent use.
type BarbarianChoiceRules struct{}

// barbarianReasonOrder returns the reasons choice reports, in the order they are reported.
func barbarianReasonOrder(choice BarbarianOptionChoice) []BarbarianReason {
	if _, ok := choice.(BarbarianChooseWeaponMasteries); ok {
		return []BarbarianReason{BarbarianWrongMasteryCount, BarbarianDuplicateMastery, BarbarianNotSimpleOrMartialMeleeWeapon, BarbarianDropsExistingMastery}
	}
	return []BarbarianReason{BarbarianEventNotForCharacter, BarbarianNoLongRestFinished, BarbarianNotCurrentlyChosen, BarbarianAlreadyChosen, BarbarianNotSimpleOrMartialMeleeWeapon}
}

// Decide checks choice against facts and, when no rule refuses it, applies it. The problems of one
// choice are reported together, each with the weapon that causes it, in the order of the reason table
// of the option.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or stats and for a nil or
// unknown choice. The inputs are unchanged. It never panics.
func (BarbarianChoiceRules) Decide(facts BarbarianChoiceFacts, choice BarbarianOptionChoice) (BarbarianChoiceVerdict, error) {
	if facts.Character.IsZero() || facts.Stats.IsZero() || choice == nil {
		return nil, invalidInput("Barbarian choice with a zero character or stats, or no choice")
	}
	var (
		verdict BarbarianChoiceVerdict
		err     error
	)
	switch c := choice.(type) {
	case BarbarianChooseWeaponMasteries:
		verdict, err = facts.chooseMasteries(c)
	case BarbarianSwapWeaponMastery:
		verdict, err = facts.swapMastery(c)
	default:
		return nil, invalidInput("unknown Barbarian option choice")
	}
	if err != nil {
		return nil, err
	}
	if refused, ok := verdict.(BarbarianChoiceRefused); ok {
		order := barbarianReasonOrder(choice)
		problems := slices.Clone(refused.problems)
		slices.SortStableFunc(problems, func(a, b BarbarianProblem) int { return slices.Index(order, a.Reason) - slices.Index(order, b.Reason) })
		return BarbarianChoiceRefused{problems: problems}, nil
	}
	return verdict, nil
}

// barbarianMasteryReason maps the problem kind of the Weapon Mastery rules to a Barbarian reason.
func barbarianMasteryReason(k MasteryProblemKind) BarbarianReason {
	switch k {
	case MasteryProblemNotMelee, MasteryProblemNotProficient, MasteryProblemWeaponNotResolved:
		return BarbarianNotSimpleOrMartialMeleeWeapon // the last is an item found that is not a weapon
	case MasteryProblemDuplicateWeapon:
		return BarbarianDuplicateMastery
	case MasteryProblemDropsExistingMastery:
		return BarbarianDropsExistingMastery
	case MasteryProblemAlreadyChosen:
		return BarbarianAlreadyChosen
	case MasteryProblemNotCurrentlyChosen:
		return BarbarianNotCurrentlyChosen
	case MasteryProblemWrongMoment:
		return BarbarianNoLongRestFinished
	case MasteryProblemWrongCount, MasteryProblemChangesMoreThanOne:
		return BarbarianWrongMasteryCount
	}
	return BarbarianNotSimpleOrMartialMeleeWeapon
}

func barbarianMasteryProblems(in []MasteryProblem) []BarbarianProblem {
	out := make([]BarbarianProblem, len(in))
	for i, p := range in {
		out[i] = BarbarianProblem{Reason: barbarianMasteryReason(p.Kind), Item: p.WeaponID}
	}
	return out
}

// acceptMasteries stores choice as the Weapon Mastery choice of the new Character.
func (f BarbarianChoiceFacts) acceptMasteries(choice WeaponMasteryChoice, event Event) (BarbarianChoiceVerdict, error) {
	next, err := WithFeatureState(f.Character, WeaponMasteryChoiceKey(), NewWeaponMasteryChoiceState(choice))
	if err != nil {
		return nil, err
	}
	return BarbarianChoiceAccepted{character: next, events: []Event{event}}, nil
}

func (f BarbarianChoiceFacts) chooseMasteries(c BarbarianChooseWeaponMasteries) (BarbarianChoiceVerdict, error) {
	// Extending an empty choice is creating one, so one call serves level 1 and every later level.
	res, err := ExtendWeaponMasteryChoice(f.Masteries, c.Kinds, f.Stats.WeaponMastery(), f.Weapons, SimpleOrMartialMeleeEligibility{})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case WeaponMasteryChoiceRejected:
		return BarbarianChoiceRefused{problems: barbarianMasteryProblems(v.problems)}, nil
	case WeaponMasteryChoiceAccepted:
		return f.acceptMasteries(v.choice, BarbarianWeaponMasteriesChosen{id: f.Character.ID(), kinds: v.choice.Kinds()})
	}
	return nil, invalidInput("unknown Weapon Mastery choice result")
}

func (f BarbarianChoiceFacts) swapMastery(c BarbarianSwapWeaponMastery) (BarbarianChoiceVerdict, error) {
	moment := MasteryChangeLongRest
	var problems []BarbarianProblem
	if c.RestApplied.CharacterID() != f.Character.ID() {
		problems = append(problems, BarbarianProblem{Reason: BarbarianEventNotForCharacter})
	} else if c.RestApplied.BenefitsOf() != LongRest {
		moment = MasteryChangeShortRest
	}
	res, err := ChangeWeaponMasteryChoice(WeaponMasteryChange{
		Current: f.Masteries, Swaps: []MasterySwap{{Remove: c.Remove, Add: c.Add}}, Moment: moment, Rule: ReplaceOneKind,
		Resolved: f.Weapons, Eligibility: SimpleOrMartialMeleeEligibility{},
	})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case WeaponMasteryChoiceRejected:
		problems = append(problems, barbarianMasteryProblems(v.problems)...)
	case WeaponMasteryChoiceAccepted:
		if len(problems) == 0 {
			return f.acceptMasteries(v.choice, BarbarianWeaponMasterySwapped{id: f.Character.ID(), removed: c.Remove, added: c.Add})
		}
	}
	return BarbarianChoiceRefused{problems: problems}, nil
}
