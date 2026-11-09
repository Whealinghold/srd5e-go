package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// WeaponMasteryChoice is the set of weapon kinds a character may use mastery
// properties with (SRD 5.2.1, "Level 1: Weapon Mastery", pp. 29, 48, 54, 58-59,
// 62), kept in a defined order. Build it with CreateWeaponMasteryChoice,
// ExtendWeaponMasteryChoice or ChangeWeaponMasteryChoice; the zero value is the
// empty choice. Immutable. It holds a slice, so compare with Equal, not ==.
type WeaponMasteryChoice struct{ kinds []content.ItemID }

// Kinds returns the kinds in order as a new slice.
func (c WeaponMasteryChoice) Kinds() []content.ItemID { return slices.Clone(c.kinds) }

// Equal reports whether c and o have the same kinds in the same order.
func (c WeaponMasteryChoice) Equal(o WeaponMasteryChoice) bool { return slices.Equal(c.kinds, o.kinds) }

func (c WeaponMasteryChoice) contains(id content.ItemID) bool { return slices.Contains(c.kinds, id) }

// WeaponMasteryAccess says whether a character has a feature that lets them use
// a weapon's mastery property: the closed set NoMasteryFeature and
// MasteryFromChoice.
//
//sumtype:decl
type WeaponMasteryAccess interface {
	// Permits reports whether the mastery property of the weapon kind may be used.
	Permits(weaponID content.ItemID) bool
}

// NoMasteryFeature permits no weapon kind.
type NoMasteryFeature struct{}

// Permits returns false.
func (NoMasteryFeature) Permits(content.ItemID) bool { return false }

// MasteryFromChoice permits exactly the kinds of its Choice.
type MasteryFromChoice struct{ Choice WeaponMasteryChoice }

// Permits reports whether weaponID is in the choice.
func (m MasteryFromChoice) Permits(weaponID content.ItemID) bool { return m.Choice.contains(weaponID) }

// MasteryInPlay says which mastery property an attack uses: the closed set
// WeaponsOwnMastery and ReplacedMastery (Tactical Master, SRD 5.2.1, p. 48).
//
//sumtype:decl
type MasteryInPlay interface{ isMasteryInPlay() }

// WeaponsOwnMastery uses the weapon's own mastery property.
type WeaponsOwnMastery struct{}

// ReplacedMastery uses Property instead of the weapon's own, for one attack
// only; the stored choice does not change.
type ReplacedMastery struct{ Property content.MasteryProperty }

func (WeaponsOwnMastery) isMasteryInPlay() {}
func (ReplacedMastery) isMasteryInPlay()   {}

// masteryInPlay returns the mastery property used for weapon w under inPlay.
func masteryInPlay(w content.Weapon, inPlay MasteryInPlay) content.MasteryProperty {
	if r, ok := inPlay.(ReplacedMastery); ok {
		return r.Property
	}
	return w.Mastery()
}

// MasteryProblemKind says what is wrong with a proposed Weapon Mastery choice.
// The zero value is not a problem.
type MasteryProblemKind int

// The problem kinds.
const (
	MasteryProblemWrongCount MasteryProblemKind = iota + 1
	MasteryProblemDuplicateWeapon
	MasteryProblemNotMelee
	MasteryProblemNotProficient
	MasteryProblemWeaponNotResolved
	MasteryProblemDropsExistingMastery
	MasteryProblemAlreadyChosen
	MasteryProblemNotCurrentlyChosen
	MasteryProblemChangesMoreThanOne
	MasteryProblemWrongMoment
)

func masteryProblemNames() []string {
	return []string{"wrongCount", "duplicateWeapon", "notMelee", "notProficient", "weaponNotResolved",
		"dropsExistingMastery", "alreadyChosen", "notCurrentlyChosen", "changesMoreThanOne", "wrongMoment"}
}

// String returns the short name ("wrongCount", ...) or "unknown".
func (k MasteryProblemKind) String() string {
	if k < 1 || int(k) > len(masteryProblemNames()) {
		return "unknown"
	}
	return masteryProblemNames()[k-1]
}

// MasteryProblem is one problem of a proposed choice. WeaponID is empty where
// the kind names no weapon (wrongCount, changesMoreThanOne, wrongMoment). It has
// no display text.
type MasteryProblem struct {
	Kind     MasteryProblemKind
	WeaponID content.ItemID
}

// WeaponMasteryChoiceResult is the closed outcome of the three choice
// functions: WeaponMasteryChoiceAccepted or WeaponMasteryChoiceRejected.
//
//sumtype:decl
type WeaponMasteryChoiceResult interface{ isWeaponMasteryChoiceResult() }

// WeaponMasteryChoiceAccepted carries the accepted choice.
type WeaponMasteryChoiceAccepted struct{ choice WeaponMasteryChoice }

// Choice returns the accepted choice.
func (a WeaponMasteryChoiceAccepted) Choice() WeaponMasteryChoice { return a.choice }

// WeaponMasteryChoiceRejected carries every problem found, in a defined order.
type WeaponMasteryChoiceRejected struct{ problems []MasteryProblem }

// Problems returns the problems as a new slice.
func (r WeaponMasteryChoiceRejected) Problems() []MasteryProblem { return slices.Clone(r.problems) }

func (WeaponMasteryChoiceAccepted) isWeaponMasteryChoiceResult() {}
func (WeaponMasteryChoiceRejected) isWeaponMasteryChoiceResult() {}

// MasteryEligibility is the rule saying which weapons a class may choose: the
// closed set SimpleOrMartialMeleeEligibility, SimpleOrMartialEligibility and
// ProficientWeaponEligibility.
//
// A nil value, including the zero WeaponMasteryChange.Eligibility, is read as
// SimpleOrMartialEligibility.
//
//sumtype:decl
type MasteryEligibility interface {
	// problem returns the kind a weapon breaks, if any.
	problem(w content.Weapon) (MasteryProblemKind, bool)
}

// SimpleOrMartialMeleeEligibility allows Simple and Martial Melee weapons (the
// Barbarian, SRD 5.2.1, p. 29).
type SimpleOrMartialMeleeEligibility struct{}

func (SimpleOrMartialMeleeEligibility) problem(w content.Weapon) (MasteryProblemKind, bool) {
	return MasteryProblemNotMelee, w.Type() == content.WeaponRanged
}

// SimpleOrMartialEligibility allows every Simple and Martial weapon (the
// Fighter, SRD 5.2.1, p. 48).
type SimpleOrMartialEligibility struct{}

func (SimpleOrMartialEligibility) problem(content.Weapon) (MasteryProblemKind, bool) { return 0, false }

// ProficientWeaponEligibility allows the weapons the character is proficient
// with (Paladin, Ranger, Rogue).
type ProficientWeaponEligibility struct{ Proficiencies WeaponProficiencies }

func (e ProficientWeaponEligibility) problem(w content.Weapon) (MasteryProblemKind, bool) {
	return MasteryProblemNotProficient, e.Proficiencies.StatusFor(w) == WeaponNotProficient
}

// MasteryRetrainRule says how many kinds a class may change after a Long Rest:
// ReplaceOneKind (Barbarian, Fighter) or ReplaceAnyKinds (Paladin, Ranger,
// Rogue). The zero value is not a rule.
type MasteryRetrainRule int

const (
	// ReplaceOneKind allows changing at most one kind.
	ReplaceOneKind MasteryRetrainRule = iota + 1
	// ReplaceAnyKinds allows changing any number of kinds.
	ReplaceAnyKinds
)

// MasteryChangeMoment is when a change is attempted. The host maps its rest
// kind to it. The zero value is not a moment.
type MasteryChangeMoment int

const (
	// MasteryChangeShortRest is a Short Rest: a change is refused.
	MasteryChangeShortRest MasteryChangeMoment = iota + 1
	// MasteryChangeLongRest is a Long Rest: a change is allowed.
	MasteryChangeLongRest
)

// MasterySwap replaces the kind Remove with the kind Add.
type MasterySwap struct{ Remove, Add content.ItemID }

// WeaponMasteryChange is the input of ChangeWeaponMasteryChoice. Construct it
// with a keyed literal.
type WeaponMasteryChange struct {
	Current     WeaponMasteryChoice
	Swaps       []MasterySwap
	Moment      MasteryChangeMoment
	Rule        MasteryRetrainRule
	Resolved    map[content.ItemID]content.Weapon
	Eligibility MasteryEligibility
}

// orSimpleOrMartial reads a nil rule as SimpleOrMartialEligibility, which allows
// every weapon, so a nil argument never panics.
func orSimpleOrMartial(e MasteryEligibility) MasteryEligibility {
	if e == nil {
		return SimpleOrMartialEligibility{}
	}
	return e
}

func choiceRejected(problems []MasteryProblem) WeaponMasteryChoiceResult {
	return WeaponMasteryChoiceRejected{problems: problems}
}

func choiceAccepted(kinds []content.ItemID) WeaponMasteryChoiceResult {
	return WeaponMasteryChoiceAccepted{choice: WeaponMasteryChoice{kinds: slices.Clone(kinds)}}
}

// unresolved returns one weaponNotResolved problem per distinct id of ids
// without an entry in resolved, in order.
func unresolved(ids []content.ItemID, resolved map[content.ItemID]content.Weapon) []MasteryProblem {
	var out []MasteryProblem
	seen := map[content.ItemID]bool{}
	for _, id := range ids {
		if _, ok := resolved[id]; !ok && !seen[id] {
			out = append(out, MasteryProblem{Kind: MasteryProblemWeaponNotResolved, WeaponID: id})
		}
		seen[id] = true
	}
	return out
}

// candidateProblems lists the duplicateWeapon problem of each repeated kind and
// the eligibility problem of each distinct kind at its first occurrence.
func candidateProblems(candidates []content.ItemID, resolved map[content.ItemID]content.Weapon, eligibility MasteryEligibility) []MasteryProblem {
	eligibility = orSimpleOrMartial(eligibility)
	var out []MasteryProblem
	seen := map[content.ItemID]bool{}
	for _, id := range candidates {
		if seen[id] {
			out = append(out, MasteryProblem{Kind: MasteryProblemDuplicateWeapon, WeaponID: id})
			continue
		}
		seen[id] = true
		if kind, bad := eligibility.problem(resolved[id]); bad {
			out = append(out, MasteryProblem{Kind: kind, WeaponID: id})
		}
	}
	return out
}

// CreateWeaponMasteryChoice accepts exactly allowedCount distinct kinds that
// satisfy eligibility, in the order given, or rejects with every problem found:
// weaponNotResolved alone when any candidate has no entry in resolved,
// otherwise wrongCount first and then, per candidate in order, duplicateWeapon
// and notMelee or notProficient. It only reads its arguments. It returns an
// error matching srderr.ErrInvalidInput, naming allowedCount, for a count below
// 1, and no other error.
func CreateWeaponMasteryChoice(candidates []content.ItemID, resolved map[content.ItemID]content.Weapon, allowedCount int, eligibility MasteryEligibility) (WeaponMasteryChoiceResult, error) {
	if allowedCount < 1 {
		return nil, invalidInput("allowedCount below 1")
	}
	if missing := unresolved(candidates, resolved); len(missing) > 0 {
		return choiceRejected(missing), nil
	}
	var problems []MasteryProblem
	if len(candidates) != allowedCount {
		problems = append(problems, MasteryProblem{Kind: MasteryProblemWrongCount})
	}
	problems = append(problems, candidateProblems(candidates, resolved, eligibility)...)
	if len(problems) > 0 {
		return choiceRejected(problems), nil
	}
	return choiceAccepted(candidates), nil
}

// ExtendWeaponMasteryChoice accepts the full proposed list when it holds exactly
// newAllowedCount distinct kinds, contains every kind of current (as sets) and
// satisfies eligibility; the accepted choice keeps the order of proposed.
// Problems, in order: weaponNotResolved alone; otherwise wrongCount, then
// dropsExistingMastery once per dropped kind in the order of current, then the
// per-candidate problems of CreateWeaponMasteryChoice. It only reads its
// arguments. It returns an error matching srderr.ErrInvalidInput, naming
// newAllowedCount, for a count below 1, and no other error.
func ExtendWeaponMasteryChoice(current WeaponMasteryChoice, proposed []content.ItemID, newAllowedCount int, resolved map[content.ItemID]content.Weapon, eligibility MasteryEligibility) (WeaponMasteryChoiceResult, error) {
	if newAllowedCount < 1 {
		return nil, invalidInput("newAllowedCount below 1")
	}
	if missing := unresolved(proposed, resolved); len(missing) > 0 {
		return choiceRejected(missing), nil
	}
	var problems []MasteryProblem
	if len(proposed) != newAllowedCount {
		problems = append(problems, MasteryProblem{Kind: MasteryProblemWrongCount})
	}
	for _, id := range current.kinds {
		if !slices.Contains(proposed, id) {
			problems = append(problems, MasteryProblem{Kind: MasteryProblemDropsExistingMastery, WeaponID: id})
		}
	}
	problems = append(problems, candidateProblems(proposed, resolved, eligibility)...)
	if len(problems) > 0 {
		return choiceRejected(problems), nil
	}
	return choiceAccepted(proposed), nil
}

// ChangeWeaponMasteryChoice replaces kinds of req.Current after a Long Rest. A
// Short Rest is rejected with wrongMoment alone. Otherwise problems come in the
// order: weaponNotResolved alone (for the added kinds); changesMoreThanOne under
// ReplaceOneKind; then per swap notCurrentlyChosen, alreadyChosen and
// duplicateWeapon; then notMelee or notProficient once per added kind that
// raised no earlier problem. The accepted choice replaces each removed kind in
// place with its added kind. An empty swap list is accepted unchanged. It only
// reads its arguments and returns no error.
func ChangeWeaponMasteryChoice(req WeaponMasteryChange) (WeaponMasteryChoiceResult, error) {
	if req.Moment != MasteryChangeLongRest {
		return choiceRejected([]MasteryProblem{{Kind: MasteryProblemWrongMoment}}), nil
	}
	adds := make([]content.ItemID, len(req.Swaps))
	removed := map[content.ItemID]content.ItemID{}
	for i, s := range req.Swaps {
		adds[i] = s.Add
		removed[s.Remove] = s.Add
	}
	if missing := unresolved(adds, req.Resolved); len(missing) > 0 {
		return choiceRejected(missing), nil
	}
	var problems []MasteryProblem
	if req.Rule == ReplaceOneKind && len(req.Swaps) > 1 {
		problems = append(problems, MasteryProblem{Kind: MasteryProblemChangesMoreThanOne})
	}
	flagged := map[content.ItemID]bool{}
	seenRemove, seenAdd := map[content.ItemID]bool{}, map[content.ItemID]bool{}
	add := func(kind MasteryProblemKind, id content.ItemID) {
		problems = append(problems, MasteryProblem{Kind: kind, WeaponID: id})
		flagged[id] = true
	}
	for _, s := range req.Swaps {
		if !req.Current.contains(s.Remove) {
			add(MasteryProblemNotCurrentlyChosen, s.Remove)
		}
		if _, byAnotherPair := removed[s.Add]; req.Current.contains(s.Add) && !byAnotherPair {
			add(MasteryProblemAlreadyChosen, s.Add)
		}
		if seenRemove[s.Remove] {
			add(MasteryProblemDuplicateWeapon, s.Remove)
		}
		if seenAdd[s.Add] {
			add(MasteryProblemDuplicateWeapon, s.Add)
		}
		seenRemove[s.Remove], seenAdd[s.Add] = true, true
	}
	eligibility := orSimpleOrMartial(req.Eligibility)
	checked := map[content.ItemID]bool{}
	for _, id := range adds {
		if flagged[id] || checked[id] {
			continue
		}
		checked[id] = true
		if kind, bad := eligibility.problem(req.Resolved[id]); bad {
			problems = append(problems, MasteryProblem{Kind: kind, WeaponID: id})
		}
	}
	if len(problems) > 0 {
		return choiceRejected(problems), nil
	}
	next := slices.Clone(req.Current.kinds)
	for i, id := range next {
		if replacement, ok := removed[id]; ok {
			next[i] = replacement
		}
	}
	return choiceAccepted(next), nil
}

// WeaponMasteryChoiceState is the FeatureState holding a character's
// WeaponMasteryChoice under WeaponMasteryChoiceKey. Immutable; compares by
// value.
type WeaponMasteryChoiceState struct{ choice WeaponMasteryChoice }

// NewWeaponMasteryChoiceState returns the state holding choice.
func NewWeaponMasteryChoiceState(choice WeaponMasteryChoice) WeaponMasteryChoiceState {
	return WeaponMasteryChoiceState{choice: WeaponMasteryChoice{kinds: choice.Kinds()}}
}

// Choice returns the held choice.
func (s WeaponMasteryChoiceState) Choice() WeaponMasteryChoice {
	return WeaponMasteryChoice{kinds: s.choice.Kinds()}
}

// Equal reports whether other is a WeaponMasteryChoiceState with an equal choice.
func (s WeaponMasteryChoiceState) Equal(other FeatureState) bool {
	o, ok := other.(WeaponMasteryChoiceState)
	return ok && s.choice.Equal(o.choice)
}

// WeaponMasteryChoiceKey returns the FeatureStateKey named weapon_mastery_choice
// under which the five weapon-mastery classes store the choice and read it back.
func WeaponMasteryChoiceKey() FeatureStateKey[WeaponMasteryChoiceState] {
	return MustFeatureStateKey[WeaponMasteryChoiceState]("weapon_mastery_choice")
}

// WeaponMasteryAccessFor returns MasteryFromChoice for the choice stored under
// WeaponMasteryChoiceKey, or NoMasteryFeature when nothing is stored. It returns
// an error matching srderr.ErrInvalidInput only when the name holds a state of
// another type.
func WeaponMasteryAccessFor(states FeatureStates) (WeaponMasteryAccess, error) {
	s, ok, err := ReadFeatureState(states, WeaponMasteryChoiceKey())
	if err != nil {
		return nil, err
	}
	if !ok {
		return NoMasteryFeature{}, nil
	}
	return MasteryFromChoice{Choice: s.Choice()}, nil
}
