package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func featProblemsOf(t *testing.T, err error) []character.FeatChoiceProblem {
	t.Helper()
	var ce *character.FeatChoicesError
	if !errors.As(err, &ce) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("err = %v, want a *FeatChoicesError matching ErrInvalidInput", err)
	}
	return ce.Problems()
}

// Requirement: per case
// Path: per case.
func TestAbilityScoreImprovementChoices(t *testing.T) {
	t.Parallel()
	str, dex, con := ability.Strength, ability.Dexterity, ability.Constitution
	for _, c := range []struct {
		name, requirement, path string
		kind                    character.AbilityScoreImprovementKind
		first, second           ability.Ability
		want                    []ability.Ability
		problem                 character.FeatChoiceProblem
	}{
		{"one by two", featsSpec + ` FR-020; SRD 5.2.1, "Ability Score Improvement", p. 87`, "happy", character.ImprovementOneByTwo, str, 0, []ability.Ability{str}, character.FeatChoiceProblem{}},
		{"one by two ignores the second", featsSpec + " FR-020", "happy", character.ImprovementOneByTwo, str, con, []ability.Ability{str}, character.FeatChoiceProblem{}},
		{"two by one", featsSpec + " FR-020", "happy", character.ImprovementTwoByOne, dex, con, []ability.Ability{dex, con}, character.FeatChoiceProblem{}},
		{"two by one, the higher ability first", featsSpec + " FR-020", "happy", character.ImprovementTwoByOne, con, dex, []ability.Ability{con, dex}, character.FeatChoiceProblem{}},
		{"a kind past the two forms", featsSpec + " FR-020", "unhappy (invalid input)", 3, str, dex, nil, character.FeatChoiceProblem{Name: character.FeatChoiceIncreasedAbility, Kind: character.FeatProblemAbilityNotAllowed}},
		{"two equal abilities", featsSpec + " FR-020", "unhappy (invalid input)", character.ImprovementTwoByOne, str, str, nil, character.FeatChoiceProblem{Name: character.FeatChoiceIncreasedAbility, Kind: character.FeatProblemDuplicate}},
		{"invalid first ability", featsSpec + " FR-020", "unhappy (invalid input)", character.ImprovementOneByTwo, 0, 0, nil, character.FeatChoiceProblem{Name: character.FeatChoiceIncreasedAbility, Kind: character.FeatProblemAbilityNotAllowed}},
		{"invalid second ability", featsSpec + " FR-020", "unhappy (invalid input)", character.ImprovementTwoByOne, str, 0, nil, character.FeatChoiceProblem{Name: character.FeatChoiceIncreasedAbility, Kind: character.FeatProblemAbilityNotAllowed}},
		{"invalid kind", featsSpec + " FR-020", "unhappy (invalid input)", 0, str, dex, nil, character.FeatChoiceProblem{Name: character.FeatChoiceIncreasedAbility, Kind: character.FeatProblemAbilityNotAllowed}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewAbilityScoreImprovementChoices(c.kind, c.first, c.second)
			if c.want == nil {
				if p := featProblemsOf(t, err); !slices.Equal(p, []character.FeatChoiceProblem{c.problem}) {
					t.Errorf("problems = %v, want %v [%s]", p, c.problem, c.requirement)
				}
				func() {
					defer func() {
						if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
							t.Errorf("Must did not panic with the error: %v", r)
						}
					}()
					character.MustAbilityScoreImprovementChoices(c.kind, c.first, c.second)
				}()
				return
			}
			if err != nil || !slices.Equal(got.Abilities(), c.want) || got.Kind() != c.kind {
				t.Fatalf("= %v, %v, want %v", got, err, c.want)
			}
			if got != character.MustAbilityScoreImprovementChoices(c.kind, c.first, c.second) {
				t.Error("Must differs from New")
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-020 - the choice converts to the Ability Score Improvement increase of the right shape (2 to one ability, or 1 to each of two, maximum 20); the zero value is invalid.
// Path: happy.
func TestAbilityScoreImprovementChoicesIncrease(t *testing.T) {
	t.Parallel()
	one, _ := character.MustAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0).Increase()
	two, _ := character.MustAbilityScoreImprovementChoices(character.ImprovementTwoByOne, ability.Dexterity, ability.Constitution).Increase()
	if !one.Equal(character.MustAbilityScoreIncrease(map[ability.Ability]int{ability.Strength: 2}, 20)) ||
		!two.Equal(character.MustAbilityScoreIncrease(map[ability.Ability]int{ability.Dexterity: 1, ability.Constitution: 1}, 20)) {
		t.Errorf("increases = %v %v", one, two)
	}
	if _, err := (character.AbilityScoreImprovementChoices{}).Increase(); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero value: %v", err)
	}
	if (character.AbilityScoreImprovementChoices{}).Abilities() != nil {
		t.Error("the zero value names no ability")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-023, FR-024 - a single increase names a valid ability only.
// Path: unhappy (invalid input).
func TestSingleAbilityIncrease(t *testing.T) {
	t.Parallel()
	if got, err := character.NewSingleAbilityIncrease(ability.Wisdom); err != nil || got.Ability() != ability.Wisdom {
		t.Errorf("= %v, %v", got, err)
	}
	_, err := character.NewSingleAbilityIncrease(0)
	if p := featProblemsOf(t, err); !slices.Equal(p, []character.FeatChoiceProblem{{Name: character.FeatChoiceIncreasedAbility, Kind: character.FeatProblemAbilityNotAllowed}}) {
		t.Errorf("problems = %v", p)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-019 - FeatChoicesError returns a copy of its problems, matches by problems and satisfies ErrInvalidInput.
// Path: unhappy (immutability).
func TestFeatChoicesError(t *testing.T) {
	t.Parallel()
	_, err := character.NewSkilledChoices(nil)
	var ce *character.FeatChoicesError
	if !errors.As(err, &ce) {
		t.Fatal(err)
	}
	ce.Problems()[0] = character.FeatChoiceProblem{}
	if ce.Problems()[0].Name != character.FeatChoiceSkilledPicks {
		t.Error("Problems must return a copy")
	}
	_, err2 := character.NewSkilledChoices([]character.SkilledPick{})
	_, other := character.NewAbilityScoreImprovementChoices(0, 0, 0)
	if !errors.Is(err, err2) || errors.Is(err, other) || errors.Is(err, errors.New("x")) || ce.Error() == "" {
		t.Error("Is must compare the problems")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-027 to FR-029 (SRD 5.2.1, "Magic Initiate", p. 87) - the constructor copies its cantrips, keeps the list, ability and spells, and refuses an invalid id, ability or list and more than 8 cantrips; the spell Change levels are a bit set.
// Path: per case.
func TestMagicInitiateChoices(t *testing.T) {
	t.Parallel()
	cantrips := spellIDsOf("guidance", "light")
	c, err := character.NewMagicInitiateChoices("cleric", ability.Wisdom, cantrips, "bless")
	if err != nil || c.List() != "cleric" || c.Ability() != ability.Wisdom || c.LevelOneSpell() != "bless" || !slices.Equal(c.Cantrips(), cantrips) {
		t.Fatalf("= %v, %v", c, err)
	}
	cantrips[0] = "x"
	c.Cantrips()[1] = "y"
	if !slices.Equal(c.Cantrips(), spellIDsOf("guidance", "light")) {
		t.Error("the choices must copy their cantrips")
	}
	many := make([]content.SpellID, 9)
	for i := range many {
		many[i] = "guidance"
	}
	for name, run := range map[string]func() error{
		"invalid list": func() error {
			_, e := character.NewMagicInitiateChoices("", ability.Wisdom, cantrips, "bless")
			return e
		},
		"invalid ability": func() error { _, e := character.NewMagicInitiateChoices("cleric", 0, cantrips, "bless"); return e },
		"invalid level 1": func() error {
			_, e := character.NewMagicInitiateChoices("cleric", ability.Wisdom, cantrips, "")
			return e
		},
		"invalid cantrip": func() error {
			_, e := character.NewMagicInitiateChoices("cleric", ability.Wisdom, []content.SpellID{"guidance", ""}, "bless")
			return e
		},
		"nine cantrips": func() error {
			_, e := character.NewMagicInitiateChoices("cleric", ability.Wisdom, many, "bless")
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := character.NewMagicInitiateChoices("cleric", ability.Wisdom, many[:8], "bless"); err != nil {
		t.Errorf("eight cantrips: %v", err)
	}
	if c.SpellChangedAt(character.MustLevel(5)) || c.SpellChangedAt(character.Level{}) {
		t.Error("a new choice has used no Spell Change")
	}
	if !c.Equal(clericInitiate()) || c.Equal(wizardInitiate()) {
		t.Error("Equal must compare the choices")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-038 to FR-041, Table S (SRD 5.2.1, "Skilled", p. 87) - exactly three distinct picks, any mix of skills and tools; the wrong count, a repeat and a blank pick are refused with a typed problem.
// Path: per case.
func TestSkilledChoices(t *testing.T) {
	t.Parallel()
	ins, per, ste := character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillPerception}, character.SkilledSkill{Skill: content.SkillStealth}
	k1, k2, k3 := character.SkilledTool{Tool: "thieves_tools"}, character.SkilledTool{Tool: "herbalism_kit"}, character.SkilledTool{Tool: "gaming_set"}
	for _, c := range []struct {
		name, requirement, path string
		picks                   []character.SkilledPick
		want                    character.FeatChoiceProblemKind // 0: accepted
	}{
		{"three skills", featsSpec + " FR-039, Table S", "happy", []character.SkilledPick{ins, per, ste}, 0},
		{"three tools", featsSpec + " FR-039, Table S", "happy", []character.SkilledPick{k1, k2, k3}, 0},
		{"two skills and a tool", featsSpec + " FR-039, Table S", "happy", []character.SkilledPick{ins, per, k1}, 0},
		{"a skill and two tools", featsSpec + " FR-039, Table S", "happy", []character.SkilledPick{ins, k1, k2}, 0},
		{"two picks", featsSpec + " FR-038, Table S", "unhappy (boundary)", []character.SkilledPick{ins, per}, character.FeatProblemWrongCount},
		{"four picks", featsSpec + " FR-038, Table S", "unhappy (boundary)", []character.SkilledPick{ins, per, ste, k1}, character.FeatProblemWrongCount},
		{"no picks", featsSpec + " FR-038, Table S", "unhappy (boundary)", nil, character.FeatProblemWrongCount},
		{"a repeated skill", featsSpec + " FR-038, Table S", "unhappy (invalid input)", []character.SkilledPick{ins, ins, ste}, character.FeatProblemDuplicate},
		{"a repeated tool", featsSpec + " FR-038, Table S", "unhappy (invalid input)", []character.SkilledPick{k1, k2, k1}, character.FeatProblemDuplicate},
		{"the first and last repeated", featsSpec + " FR-038", "unhappy (invalid input)", []character.SkilledPick{ins, per, ins}, character.FeatProblemDuplicate},
		{"the second and last repeated", featsSpec + " FR-038", "unhappy (invalid input)", []character.SkilledPick{ins, per, per}, character.FeatProblemDuplicate},
		{"a nil pick", featsSpec + " FR-038", "unhappy (invalid input)", []character.SkilledPick{ins, per, nil}, character.FeatProblemMissing},
		{"an invalid skill", featsSpec + " FR-038", "unhappy (invalid input)", []character.SkilledPick{ins, per, character.SkilledSkill{}}, character.FeatProblemMissing},
		{"a blank tool", featsSpec + " FR-038", "unhappy (invalid input)", []character.SkilledPick{ins, per, character.SkilledTool{}}, character.FeatProblemMissing},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewSkilledChoices(c.picks)
			if c.want == 0 {
				if err != nil || !slices.Equal(got.Picks(), c.picks) {
					t.Errorf("= %v, %v [%s]", got, err, c.requirement)
				}
				return
			}
			want := []character.FeatChoiceProblem{{Name: character.FeatChoiceSkilledPicks, Kind: c.want}}
			if p := featProblemsOf(t, err); !slices.Equal(p, want) {
				t.Errorf("problems = %v, want %v [%s]", p, want, c.requirement)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustSkilledChoices must panic")
				}
			}()
			character.MustSkilledChoices(c.picks...)
		})
	}
	if (character.SkilledChoices{}).Picks() != nil {
		t.Error("the zero value has no picks")
	}
	if got := character.MustSkilledChoices(ins, per, ste); !slices.Equal(got.Picks(), []character.SkilledPick{ins, per, ste}) {
		t.Errorf("MustSkilledChoices = %v", got)
	}
	picks := skilledOf(ins, per, ste).Picks()
	picks[0] = nil
	if skilledOf(ins, per, ste).Picks()[0] == nil {
		t.Error("Picks must return a copy")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-038 - NewSkilledChoices never panics and accepts exactly three distinct valid picks, whatever the input.
// Path: unhappy (invalid input).
func FuzzNewSkilledChoices(f *testing.F) {
	f.Add(3, 1, 2, 3, true)
	f.Add(0, 0, 0, 0, false)
	f.Add(4, 1, 1, 1, true)
	f.Fuzz(func(t *testing.T, n, a, b, c int, tool bool) {
		mk := func(v int) character.SkilledPick {
			if tool {
				return character.SkilledTool{Tool: content.ItemID(string(rune('a' + (v%26+26)%26)))}
			}
			return character.SkilledSkill{Skill: content.Skill(v)}
		}
		var picks []character.SkilledPick
		for i := 0; i < n%6; i++ {
			picks = append(picks, mk([]int{a, b, c}[i%3]))
		}
		got, err := character.NewSkilledChoices(picks)
		if err == nil && len(got.Picks()) != 3 {
			t.Fatalf("accepted %d picks", len(got.Picks()))
		}
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("err = %v", err)
		}
	})
}

// Requirement: doc/specs/feats-behaviour.md FR-020 - NewAbilityScoreImprovementChoices never panics and accepts a choice exactly when it converts to an increase.
// Path: unhappy (invalid input).
func FuzzNewAbilityScoreImprovementChoices(f *testing.F) {
	f.Add(1, 1, 2)
	f.Add(2, 3, 3)
	f.Add(0, -1, 9)
	f.Fuzz(func(t *testing.T, kind, first, second int) {
		got, err := character.NewAbilityScoreImprovementChoices(character.AbilityScoreImprovementKind(kind), ability.Ability(first), ability.Ability(second))
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v", err)
			}
			return
		}
		if _, err := got.Increase(); err != nil {
			t.Fatalf("an accepted choice does not convert: %v", err)
		}
	})
}

// Requirement: doc/specs/feats-behaviour.md FR-042, FR-043 (SRD 5.2.1, "Fighting Style", p. 47) - a Fighting Style choice gives its feat, a Fighting Style category scope with the caller's source and no choices; an invalid feat or a nil source is an error.
// Path: per case.
func TestFightingStyleChoice(t *testing.T) {
	t.Parallel()
	choice := must(character.NewFightingStyleChoice("archery"))
	src := character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}
	id, grant, choices, err := choice.ToTakeFeat(src)
	if err != nil || id != "archery" || choice.Feat() != "archery" || choices != (character.NoFeatChoices{}) || grant.Source() != src ||
		grant.Scope() != (character.FeatFromCategory{Category: content.FeatFightingStyle}) {
		t.Errorf("= %v %v %v %v", id, grant, choices, err)
	}
	if _, err := character.NewFightingStyleChoice(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid feat: %v", err)
	}
	if _, _, _, err := (character.FightingStyleChoice{}).ToTakeFeat(src); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero choice: %v", err)
	}
	if _, _, _, err := choice.ToTakeFeat(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil source: %v", err)
	}
	// a General feat chosen as a Fighting Style is refused by TakeFeat with FeatOutsideGrant
	c := charAt(t, 4)
	gid, ggrant, gchoices, _ := must(character.NewFightingStyleChoice("grappler")).ToTakeFeat(src)
	if rj := takeRejected(t, c, character.TakenFeats{}, grapplerFeat(), ggrant, gchoices, character.HeldFeatures{}); rj != (character.FeatOutsideGrant{Feat: gid}) {
		t.Errorf("rejection = %#v", rj)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-019 - each name and kind constant has the camel-case spelling of the spec and values outside them print "unknown".
// Path: per case.
func TestFeatChoiceNamesAndKindsString(t *testing.T) {
	t.Parallel()
	names := []struct {
		name character.FeatChoiceName
		want string
	}{
		{character.FeatChoiceIncreasedAbility, "increasedAbility"}, {character.FeatChoiceSpellList, "spellList"}, {character.FeatChoiceSpellcastingAbility, "spellcastingAbility"},
		{character.FeatChoiceCantrips, "cantrips"}, {character.FeatChoiceLevelOneSpell, "levelOneSpell"}, {character.FeatChoiceSkilledPicks, "skilledPicks"},
		{character.FeatChoiceReplacementSpell, "replacementSpell"}, {character.FeatChoiceFeatChoicesShape, "featChoicesShape"},
	}
	for i, c := range names {
		if c.name.String() != c.want || int(c.name) != i+1 {
			t.Errorf("name %d = %q (%d), want %q", i+1, c.name.String(), int(c.name), c.want)
		}
	}
	kinds := []struct {
		kind character.FeatChoiceProblemKind
		want string
	}{
		{character.FeatProblemMissing, "missing"}, {character.FeatProblemUnexpected, "unexpected"}, {character.FeatProblemAbilityNotAllowed, "abilityNotAllowed"},
		{character.FeatProblemListNotAllowed, "listNotAllowed"}, {character.FeatProblemWrongCount, "wrongCount"}, {character.FeatProblemDuplicate, "duplicate"},
		{character.FeatProblemNotACantrip, "notACantrip"}, {character.FeatProblemNotLevelOne, "notLevelOne"}, {character.FeatProblemNotOnSpellList, "notOnSpellList"},
		{character.FeatProblemLevelMismatch, "levelMismatch"}, {character.FeatProblemListAlreadyUsed, "listAlreadyUsed"},
	}
	for i, c := range kinds {
		if c.kind.String() != c.want || int(c.kind) != i+1 {
			t.Errorf("kind %d = %q (%d), want %q", i+1, c.kind.String(), int(c.kind), c.want)
		}
	}
	if character.FeatChoiceName(0).String() != "unknown" || character.FeatChoiceName(9).String() != "unknown" || character.FeatChoiceProblemKind(0).String() != "unknown" || character.FeatChoiceProblemKind(12).String() != "unknown" {
		t.Error("an undefined value is unknown")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-019 - a nil *FeatChoicesError matches nothing and is matched by nothing.
// Path: unhappy (boundary).
func TestFeatChoicesErrorWithNil(t *testing.T) {
	t.Parallel()
	_, err := character.NewSkilledChoices(nil)
	var ce *character.FeatChoicesError
	errors.As(err, &ce)
	var none *character.FeatChoicesError
	if ce.Is(none) || none.Is(ce) || ce.Is(nil) || !ce.Is(ce) {
		t.Error("Is must refuse a nil error and match an equal one")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-057 - MagicInitiateChoices compares every field: the list, the ability, the level 1 spell and each cantrip, whichever way they differ.
// Path: per case.
func TestMagicInitiateChoicesEqualComparesEveryField(t *testing.T) {
	t.Parallel()
	base := clericInitiate()
	for name, other := range map[string]character.MagicInitiateChoices{
		"a greater list":      miOf("druid", ability.Wisdom, []string{"guidance", "light"}, "bless"),
		"a lesser list":       miOf("bard", ability.Wisdom, []string{"guidance", "light"}, "bless"),
		"a greater ability":   miOf("cleric", ability.Charisma, []string{"guidance", "light"}, "bless"),
		"a lesser ability":    miOf("cleric", ability.Intelligence, []string{"guidance", "light"}, "bless"),
		"a greater level one": miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "cure_wounds"),
		"a lesser level one":  miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "aid"),
		"another cantrip":     miOf("cleric", ability.Wisdom, []string{"guidance", "mending"}, "bless"),
		"one cantrip fewer":   miOf("cleric", ability.Wisdom, []string{"guidance"}, "bless"),
	} {
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("%s: choices compared equal", name)
		}
	}
	if !base.Equal(miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "bless")) {
		t.Error("the same choices must compare equal")
	}
}
