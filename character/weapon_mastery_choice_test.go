package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

type mp = character.MasteryProblem

func problem(kind character.MasteryProblemKind, id string) mp {
	return mp{Kind: kind, WeaponID: content.ItemID(id)}
}

// choiceOutcome returns the accepted kinds (as names) or the problems of a result.
func choiceOutcome(t *testing.T, res character.WeaponMasteryChoiceResult, err error) ([]content.ItemID, []mp) {
	t.Helper()
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	switch r := res.(type) {
	case character.WeaponMasteryChoiceAccepted:
		return r.Choice().Kinds(), nil
	case character.WeaponMasteryChoiceRejected:
		return nil, r.Problems()
	}
	t.Fatalf("unknown result %#v", res)
	return nil, nil
}

func choiceOf(t *testing.T, names ...string) character.WeaponMasteryChoice {
	t.Helper()
	ws := weaponFixtures(t)
	res, err := character.CreateWeaponMasteryChoice(ids(names...), resolvedOf(ws, names...), len(names), character.SimpleOrMartialEligibility{})
	kinds, _ := choiceOutcome(t, res, err)
	if kinds == nil {
		t.Fatal("fixture choice rejected")
	}
	return res.(character.WeaponMasteryChoiceAccepted).Choice()
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-014 to FR-021, table 4.3a (create rows) -
// create accepts exactly allowedCount distinct eligible kinds in the order given, or
// rejects with every problem in the specified order (SRD 5.2.1, "Level 1: Weapon
// Mastery", pp. 29, 48, 54, 58-59, 62).
// Path: per case.
func TestCreateWeaponMasteryChoice(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	rogue := character.ProficientWeaponEligibility{Proficiencies: rogueRules()}
	melee := character.SimpleOrMartialMeleeEligibility{}
	resolved := func(names ...string) map[content.ItemID]content.Weapon { return resolvedOf(ws, names...) }
	cases := []struct {
		name, requirement, path string
		candidates              []string
		resolved                map[content.ItemID]content.Weapon
		count                   int
		eligibility             character.MasteryEligibility
		wantKinds               []string
		wantProblems            []mp
	}{
		{"Barbarian level 1", "FR-014, FR-017", "happy", []string{"greataxe", "handaxe"}, resolved("greataxe", "handaxe"), 2, melee, []string{"greataxe", "handaxe"}, nil},
		{"Barbarian ranged", "FR-017", "unhappy (invalid input)", []string{"greataxe", "shortbow"}, resolved("greataxe", "shortbow"), 2, melee, nil, []mp{problem(character.MasteryProblemNotMelee, "shortbow")}},
		{"Barbarian one kind", "FR-015", "unhappy (boundary)", []string{"greataxe"}, resolved("greataxe"), 2, melee, nil, []mp{{Kind: character.MasteryProblemWrongCount}}},
		{"Barbarian three kinds", "FR-015", "unhappy (boundary)", []string{"greataxe", "handaxe", "mace"}, resolved("greataxe", "handaxe", "mace"), 2, melee, nil, []mp{{Kind: character.MasteryProblemWrongCount}}},
		{"Barbarian empty", "FR-015", "unhappy (empty case)", nil, resolved(), 2, melee, nil, []mp{{Kind: character.MasteryProblemWrongCount}}},
		{"Duplicate", "FR-016", "unhappy (invalid input)", []string{"handaxe", "handaxe"}, resolved("handaxe"), 2, melee, nil, []mp{problem(character.MasteryProblemDuplicateWeapon, "handaxe")}},
		{"Fighter level 1", "FR-018", "happy", []string{"longsword", "longbow", "dagger"}, resolved("longsword", "longbow", "dagger"), 3, character.SimpleOrMartialEligibility{}, []string{"longsword", "longbow", "dagger"}, nil},
		{"Paladin proficient", "FR-019", "happy", []string{"longsword", "javelin"}, resolved("longsword", "javelin"), 2, character.ProficientWeaponEligibility{Proficiencies: character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponSimple), character.CategoryProficiency(content.WeaponMartial))}, []string{"longsword", "javelin"}, nil},
		{"Rogue unproficient", "FR-019", "unhappy (invalid input)", []string{"dagger", "greatsword"}, resolved("dagger", "greatsword"), 2, rogue, nil, []mp{problem(character.MasteryProblemNotProficient, "greatsword")}},
		{"Ranger missing weapon", "FR-020", "unhappy (missing content)", []string{"longbow", "ghost"}, resolved("longbow"), 2, rogue, nil, []mp{problem(character.MasteryProblemWeaponNotResolved, "ghost")}},
		{"Missing content first", "FR-020", "unhappy (missing content)", []string{"greatsword", "greatsword", "ghost"}, resolved("greatsword"), 2, rogue, nil, []mp{problem(character.MasteryProblemWeaponNotResolved, "ghost")}},
		{"Several rule problems", "FR-021", "unhappy (invalid input)", []string{"greatsword", "greatsword"}, resolved("greatsword"), 2, rogue, nil, []mp{problem(character.MasteryProblemNotProficient, "greatsword"), problem(character.MasteryProblemDuplicateWeapon, "greatsword")}},
		{"wrong count first", "FR-021", "unhappy (rule interaction)", []string{"greatsword"}, resolved("greatsword"), 2, rogue, nil, []mp{{Kind: character.MasteryProblemWrongCount}, problem(character.MasteryProblemNotProficient, "greatsword")}},
		{"unresolved kind once", "FR-020", "unhappy (missing content)", []string{"ghost", "ghost"}, resolved(), 2, melee, nil, []mp{problem(character.MasteryProblemWeaponNotResolved, "ghost")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := character.CreateWeaponMasteryChoice(ids(c.candidates...), c.resolved, c.count, c.eligibility)
			kinds, problems := choiceOutcome(t, res, err)
			if !slices.Equal(kinds, ids(c.wantKinds...)) || !slices.Equal(problems, c.wantProblems) {
				t.Errorf("kinds = %v, problems = %v; want %v, %v [%s, %s]", kinds, problems, c.wantKinds, c.wantProblems, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-022, FR-023, FR-125, table 4.3a (extend rows) -
// extend takes the full proposed list, keeps its order, and reports problems in order.
// Path: per case.
func TestExtendWeaponMasteryChoice(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	current := choiceOf(t, "longsword", "longbow", "dagger")
	resolved := func(names ...string) map[content.ItemID]content.Weapon { return resolvedOf(ws, names...) }
	cases := []struct {
		name, requirement, path string
		proposed                []string
		resolved                map[content.ItemID]content.Weapon
		wantKinds               []string
		wantProblems            []mp
	}{
		{"one more", "FR-022", "happy", []string{"longsword", "longbow", "dagger", "spear"}, resolved("longsword", "longbow", "dagger", "spear"), []string{"longsword", "longbow", "dagger", "spear"}, nil},
		{"reordered", "FR-022", "unhappy (boundary)", []string{"spear", "dagger", "longbow", "longsword"}, resolved("longsword", "longbow", "dagger", "spear"), []string{"spear", "dagger", "longbow", "longsword"}, nil},
		{"drops one", "FR-023", "unhappy (rule interaction)", []string{"longsword", "longbow", "spear", "mace"}, resolved("longsword", "longbow", "spear", "mace"), nil, []mp{problem(character.MasteryProblemDropsExistingMastery, "dagger")}},
		{"too few", "FR-015", "unhappy (boundary)", []string{"longsword", "longbow", "dagger"}, resolved("longsword", "longbow", "dagger"), nil, []mp{{Kind: character.MasteryProblemWrongCount}}},
		{"unresolved", "FR-020", "unhappy (missing content)", []string{"longsword", "longbow", "dagger", "ghost"}, resolved("longsword", "longbow", "dagger"), nil, []mp{problem(character.MasteryProblemWeaponNotResolved, "ghost")}},
		{"wrong count and dropped", "FR-125", "unhappy (rule interaction)", []string{"longsword", "longbow", "spear"}, resolved("longsword", "longbow", "spear"), nil, []mp{{Kind: character.MasteryProblemWrongCount}, problem(character.MasteryProblemDropsExistingMastery, "dagger")}},
		{"current kind repeated", "FR-023", "unhappy (invalid input)", []string{"longsword", "longsword", "longbow", "dagger"}, resolved("longsword", "longbow", "dagger"), nil, []mp{problem(character.MasteryProblemDuplicateWeapon, "longsword")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := character.ExtendWeaponMasteryChoice(current, ids(c.proposed...), 4, c.resolved, character.SimpleOrMartialEligibility{})
			kinds, problems := choiceOutcome(t, res, err)
			if !slices.Equal(kinds, ids(c.wantKinds...)) || !slices.Equal(problems, c.wantProblems) {
				t.Errorf("kinds = %v, problems = %v; want %v, %v [%s, %s]", kinds, problems, c.wantKinds, c.wantProblems, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-024 to FR-027, FR-125, table 4.3a (change rows) -
// a change is accepted after a Long Rest only, replaces kinds in place, and reports
// problems in order.
// Path: per case.
func TestChangeWeaponMasteryChoice(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	current3 := choiceOf(t, "longsword", "longbow", "dagger")
	current2 := choiceOf(t, "longsword", "dagger")
	all := resolvedOf(ws, "longsword", "longbow", "dagger", "rapier", "spear", "greataxe", "greatsword")
	swap := func(remove, add string) character.MasterySwap {
		return character.MasterySwap{Remove: content.ItemID(remove), Add: content.ItemID(add)}
	}
	long, short := character.MasteryChangeLongRest, character.MasteryChangeShortRest
	one, many := character.ReplaceOneKind, character.ReplaceAnyKinds
	anyKind := character.SimpleOrMartialEligibility{}
	cases := []struct {
		name, requirement, path string
		current                 character.WeaponMasteryChoice
		swaps                   []character.MasterySwap
		moment                  character.MasteryChangeMoment
		rule                    character.MasteryRetrainRule
		eligibility             character.MasteryEligibility
		wantKinds               []string
		wantProblems            []mp
	}{
		{"Fighter swap, Long Rest", "FR-024, FR-027", "happy", current3, []character.MasterySwap{swap("dagger", "rapier")}, long, one, anyKind, []string{"longsword", "longbow", "rapier"}, nil},
		{"Fighter swap in place, first", "FR-027", "unhappy (boundary)", current3, []character.MasterySwap{swap("longsword", "spear")}, long, one, anyKind, []string{"spear", "longbow", "dagger"}, nil},
		{"Fighter swap, two", "FR-025", "unhappy (boundary)", current3, []character.MasterySwap{swap("dagger", "rapier"), swap("longsword", "spear")}, long, one, anyKind, nil, []mp{{Kind: character.MasteryProblemChangesMoreThanOne}}},
		{"Fighter swap, Short Rest", "FR-024", "unhappy (state error)", current3, []character.MasterySwap{swap("dagger", "rapier")}, short, one, anyKind, nil, []mp{{Kind: character.MasteryProblemWrongMoment}}},
		{"Fighter swap, not chosen", "FR-027", "unhappy (invalid input)", current3, []character.MasterySwap{swap("greataxe", "rapier")}, long, one, anyKind, nil, []mp{problem(character.MasteryProblemNotCurrentlyChosen, "greataxe")}},
		{"Fighter swap, already chosen", "FR-027", "unhappy (invalid input)", current3, []character.MasterySwap{swap("dagger", "longsword")}, long, one, anyKind, nil, []mp{problem(character.MasteryProblemAlreadyChosen, "longsword")}},
		{"Paladin swap, same kind twice", "FR-027", "unhappy (invalid input)", current2, []character.MasterySwap{swap("dagger", "rapier"), swap("dagger", "spear")}, long, many, anyKind, nil, []mp{problem(character.MasteryProblemDuplicateWeapon, "dagger")}},
		{"same kind added twice", "FR-027", "unhappy (invalid input)", current2, []character.MasterySwap{swap("dagger", "rapier"), swap("longsword", "rapier")}, long, many, anyKind, nil, []mp{problem(character.MasteryProblemDuplicateWeapon, "rapier")}},
		{"Paladin swap both", "FR-026", "happy", current2, []character.MasterySwap{swap("dagger", "rapier"), swap("longsword", "spear")}, long, many, anyKind, []string{"spear", "rapier"}, nil},
		{"swap order does not matter", "FR-027", "unhappy (boundary)", current2, []character.MasterySwap{swap("longsword", "spear"), swap("dagger", "rapier")}, long, many, anyKind, []string{"spear", "rapier"}, nil},
		{"exchange two chosen kinds", "FR-027", "unhappy (rule interaction)", current2, []character.MasterySwap{swap("dagger", "longsword"), swap("longsword", "dagger")}, long, many, anyKind, []string{"dagger", "longsword"}, nil},
		{"Ranger swap, Short Rest", "FR-024", "unhappy (state error)", current2, []character.MasterySwap{swap("dagger", "rapier")}, short, many, anyKind, nil, []mp{{Kind: character.MasteryProblemWrongMoment}}},
		{"No change", "FR-025", "unhappy (empty case)", current3, nil, long, one, anyKind, []string{"longsword", "longbow", "dagger"}, nil},
		{"Short Rest and two swaps", "FR-125", "unhappy (rule interaction)", current3, []character.MasterySwap{swap("dagger", "rapier"), swap("longsword", "spear")}, short, one, anyKind, nil, []mp{{Kind: character.MasteryProblemWrongMoment}}},
		{"Two swaps and not chosen", "FR-125", "unhappy (rule interaction)", current3, []character.MasterySwap{swap("greataxe", "rapier"), swap("dagger", "spear")}, long, one, anyKind, nil, []mp{{Kind: character.MasteryProblemChangesMoreThanOne}, problem(character.MasteryProblemNotCurrentlyChosen, "greataxe")}},
		{"Unresolved add first", "FR-020", "unhappy (missing content)", current3, []character.MasterySwap{swap("greataxe", "ghost")}, long, many, anyKind, nil, []mp{problem(character.MasteryProblemWeaponNotResolved, "ghost")}},
		{"eligibility on the added kind", "FR-125", "unhappy (invalid input)", current2, []character.MasterySwap{swap("dagger", "greatsword")}, long, many, character.ProficientWeaponEligibility{Proficiencies: character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponSimple))}, nil, []mp{problem(character.MasteryProblemNotProficient, "greatsword")}},
		{"ranged add for a melee class", "FR-017", "unhappy (invalid input)", current2, []character.MasterySwap{swap("dagger", "longbow")}, long, one, character.SimpleOrMartialMeleeEligibility{}, nil, []mp{problem(character.MasteryProblemNotMelee, "longbow")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := character.WeaponMasteryChange{Current: c.current, Swaps: slices.Clone(c.swaps), Moment: c.moment, Rule: c.rule, Resolved: all, Eligibility: c.eligibility}
			before := c.current.Kinds()
			res, err := character.ChangeWeaponMasteryChoice(req)
			kinds, problems := choiceOutcome(t, res, err)
			if !slices.Equal(kinds, ids(c.wantKinds...)) || !slices.Equal(problems, c.wantProblems) {
				t.Errorf("kinds = %v, problems = %v; want %v, %v [%s, %s]", kinds, problems, c.wantKinds, c.wantProblems, c.requirement, c.path)
			}
			if !slices.Equal(c.current.Kinds(), before) || !slices.Equal(req.Swaps, c.swaps) {
				t.Errorf("inputs altered [FR-029]")
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-028, table 4.3a "Count 0" - an allowedCount below 1
// is an input error naming the argument, not a panic, with no result.
// Path: unhappy (invalid input).
func TestWeaponMasteryChoiceRejectsCountBelowOne(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		res, err := character.CreateWeaponMasteryChoice(nil, nil, n, character.SimpleOrMartialEligibility{})
		wantInvalid(t, err, "create")
		if res != nil {
			t.Errorf("create result = %#v, want nil", res)
		}
		res, err = character.ExtendWeaponMasteryChoice(character.WeaponMasteryChoice{}, nil, n, nil, character.SimpleOrMartialEligibility{})
		wantInvalid(t, err, "extend")
		if res != nil {
			t.Errorf("extend result = %#v, want nil", res)
		}
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-028, FR-105, FR-029 - Kinds returns a copy, Equal compares
// kinds and order by value, a rejection's Problems is a copy, and the inputs are not altered.
// Path: unhappy (immutability).
func TestWeaponMasteryChoiceIsImmutable(t *testing.T) {
	t.Parallel()
	c := choiceOf(t, "longsword", "dagger")
	k := c.Kinds()
	k[0] = "mutated"
	if got := c.Kinds(); got[0] != "longsword" {
		t.Errorf("Kinds exposes the choice: %v", got)
	}
	if !c.Equal(choiceOf(t, "longsword", "dagger")) || c.Equal(choiceOf(t, "dagger", "longsword")) || c.Equal(choiceOf(t, "longsword")) {
		t.Error("Equal must compare kinds and order by value")
	}
	ws := weaponFixtures(t)
	res, _ := character.CreateWeaponMasteryChoice(ids("greataxe"), resolvedOf(ws, "greataxe"), 2, character.SimpleOrMartialEligibility{})
	rej := res.(character.WeaponMasteryChoiceRejected)
	rej.Problems()[0] = mp{}
	if rej.Problems()[0].Kind != character.MasteryProblemWrongCount {
		t.Error("Problems exposes the rejection")
	}
	cand := ids("greataxe", "handaxe")
	resolved := resolvedOf(ws, "greataxe", "handaxe")
	res, _ = character.CreateWeaponMasteryChoice(cand, resolved, 2, character.SimpleOrMartialEligibility{})
	cand[0] = "x"
	if got := res.(character.WeaponMasteryChoiceAccepted).Choice().Kinds(); got[0] != "greataxe" || len(resolved) != 2 {
		t.Errorf("accepted choice shares the candidate slice: %v", got)
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-001 to FR-006 - Permits is true exactly for the chosen
// kinds, NoMasteryFeature permits nothing (SRD 5.2.1, "Mastery Properties", p. 90).
// Path: unhappy (state error, empty and minimal case).
func TestWeaponMasteryAccessPermits(t *testing.T) {
	t.Parallel()
	choice := choiceOf(t, "longsword", "dagger")
	cases := []struct {
		name   string
		access character.WeaponMasteryAccess
		id     content.ItemID
		want   bool
	}{
		{"chosen", character.MasteryFromChoice{Choice: choice}, "dagger", true},
		{"not chosen", character.MasteryFromChoice{Choice: choice}, "club", false},
		{"no feature, chosen kind", character.NoMasteryFeature{}, "dagger", false},
		{"empty choice", character.MasteryFromChoice{}, "dagger", false},
	}
	for _, c := range cases {
		if got := c.access.Permits(c.id); got != c.want {
			t.Errorf("%s: Permits = %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-123 - the state implements FeatureState under the key
// weapon_mastery_choice, round-trips through FeatureStates, compares by value, and an
// absent key gives NoMasteryFeature.
// Path: happy.
func TestWeaponMasteryChoiceState(t *testing.T) {
	t.Parallel()
	key := character.WeaponMasteryChoiceKey()
	if key.Name() != "weapon_mastery_choice" || key != character.WeaponMasteryChoiceKey() {
		t.Errorf("key = %q", key.Name())
	}
	choice := choiceOf(t, "longsword", "dagger")
	state := character.NewWeaponMasteryChoiceState(choice)
	states, err := character.SetFeatureState(character.FeatureStates{}, key, state)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := character.ReadFeatureState(states, key)
	if err != nil || !ok || !got.Choice().Equal(choice) {
		t.Fatalf("round trip = %v, %v, %v", got, ok, err)
	}
	if !state.Equal(character.NewWeaponMasteryChoiceState(choiceOf(t, "longsword", "dagger"))) || state.Equal(character.NewWeaponMasteryChoiceState(choiceOf(t, "dagger"))) || state.Equal(nil) {
		t.Error("state must compare by value")
	}
	if _, isFeature := any(state).(character.FeatureState); !isFeature {
		t.Error("state is not a FeatureState")
	}
	state.Choice().Kinds()[0] = "mutated"
	if state.Choice().Kinds()[0] != "longsword" {
		t.Error("Choice exposes the state")
	}
	access, err := character.WeaponMasteryAccessFor(states)
	if err != nil || !access.Permits("dagger") || access.Permits("club") {
		t.Errorf("access = %#v, %v", access, err)
	}
	access, err = character.WeaponMasteryAccessFor(character.FeatureStates{})
	if _, none := access.(character.NoMasteryFeature); err != nil || !none {
		t.Errorf("absent key gave %#v, %v; want NoMasteryFeature", access, err)
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-123 - a name holding a state of another type under the
// weapon mastery key is an input error, not a silent NoMasteryFeature.
// Path: unhappy (state error).
func TestWeaponMasteryAccessForForeignState(t *testing.T) {
	t.Parallel()
	foreign := character.MustFeatureStateKey[character.SpentHitDice]("weapon_mastery_choice")
	states, err := character.SetFeatureState(character.FeatureStates{}, foreign, character.SpentHitDice{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := character.WeaponMasteryAccessFor(states); err == nil {
		t.Error("a foreign state under the key must be an error")
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-021 - the problem kind names are the short names.
// Path: happy.
func TestMasteryProblemKindString(t *testing.T) {
	t.Parallel()
	names := []string{"wrongCount", "duplicateWeapon", "notMelee", "notProficient", "weaponNotResolved", "dropsExistingMastery", "alreadyChosen", "notCurrentlyChosen", "changesMoreThanOne", "wrongMoment"}
	for i, want := range names {
		if got := character.MasteryProblemKind(i + 1).String(); got != want {
			t.Errorf("kind %d = %q, want %q", i+1, got, want)
		}
	}
	if character.MasteryProblemKind(0).String() != "unknown" || character.MasteryProblemKind(11).String() != "unknown" {
		t.Error("out-of-range kinds must be unknown")
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-029 - no input makes the three choice functions panic or
// return an error, and a rejection never carries a choice.
// Path: unhappy (invalid input).
func FuzzChangeWeaponMasteryChoice(f *testing.F) {
	f.Add("dagger", "rapier", uint8(2), uint8(1), uint8(1))
	f.Add("ghost", "ghost", uint8(0), uint8(0), uint8(9))
	ws := weaponFixtures(f)
	resolved := resolvedOf(ws, "longsword", "dagger", "rapier", "greataxe", "longbow")
	f.Fuzz(func(t *testing.T, remove, add string, moment, rule, swaps uint8) {
		current := choiceOf(t, "longsword", "dagger")
		var list []character.MasterySwap
		for range int(swaps) % 4 {
			list = append(list, character.MasterySwap{Remove: content.ItemID(remove), Add: content.ItemID(add)})
		}
		res, err := character.ChangeWeaponMasteryChoice(character.WeaponMasteryChange{
			Current: current, Swaps: list, Moment: character.MasteryChangeMoment(moment % 4), Rule: character.MasteryRetrainRule(rule % 4),
			Resolved: resolved, Eligibility: character.ProficientWeaponEligibility{Proficiencies: rogueRules()},
		})
		if err != nil || res == nil {
			t.Fatalf("result = %v, error = %v", res, err)
		}
		if rej, ok := res.(character.WeaponMasteryChoiceRejected); ok && len(rej.Problems()) == 0 {
			t.Error("a rejection without a problem")
		}
	})
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-029 - a nil eligibility is read as
// SimpleOrMartialEligibility and never panics.
// Path: unhappy (nil input).
func TestWeaponMasteryChoiceNilEligibility(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	names := []string{"longsword", "dagger"}
	res, err := character.CreateWeaponMasteryChoice(ids(names...), resolvedOf(ws, names...), 2, nil)
	if kinds, _ := choiceOutcome(t, res, err); len(kinds) != 2 {
		t.Errorf("create kinds = %v, want 2", kinds)
	}
	cur := choiceOf(t, "longsword")
	res, err = character.ExtendWeaponMasteryChoice(cur, ids(names...), 2, resolvedOf(ws, names...), nil)
	if kinds, _ := choiceOutcome(t, res, err); len(kinds) != 2 {
		t.Errorf("extend kinds = %v, want 2", kinds)
	}
	res, err = character.ChangeWeaponMasteryChoice(character.WeaponMasteryChange{
		Current:  cur,
		Swaps:    []character.MasterySwap{{Remove: "longsword", Add: "dagger"}},
		Moment:   character.MasteryChangeLongRest,
		Rule:     character.ReplaceOneKind,
		Resolved: resolvedOf(ws, "dagger"),
	})
	if kinds, _ := choiceOutcome(t, res, err); len(kinds) != 1 || kinds[0] != "dagger" {
		t.Errorf("change kinds = %v, want [dagger]", kinds)
	}
}
