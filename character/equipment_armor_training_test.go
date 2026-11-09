package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

const (
	catL = content.ArmorLight
	catM = content.ArmorMedium
	catH = content.ArmorHeavy
	catS = content.ArmorShield
)

type cats = []content.ArmorCategory

// Requirement: per case equipSpec FR-071, FR-072, FR-073, FR-075, FR-076
// (SRD 5.2.1, "Armor Training", p. 177; "Armor", p. 92; Initiative, p. 13) -
// the "Armor Training" rows: Untrained, RollCircumstancesFor and
// WearsUntrainedBodyArmor.
// Path: per case (happy and unhappy rule interaction, empty and minimal).
func TestArmorTrainingRows(t *testing.T) {
	t.Parallel()
	str, dex := []ability.Ability{ability.Strength}, []ability.Ability{ability.Dexterity}
	cases := []struct {
		name             string
		worn, trained    cats
		involved         []ability.Ability
		untrained        cats
		disadvantage     bool
		untrainedBodyArm bool
	}{
		{"H untrained, Strength", cats{catH}, cats{catL, catM, catS}, str, cats{catH}, true, true},
		{"H untrained, Dexterity", cats{catH}, cats{catL, catM, catS}, dex, cats{catH}, true, true},
		{"H trained, Strength", cats{catH}, cats{catH}, str, nil, false, false},
		{"H untrained, Initiative", cats{catH}, cats{catL, catM, catS}, dex, cats{catH}, true, true},
		{"M untrained, Strength", cats{catM}, cats{catL}, str, cats{catM}, true, true},
		{"L untrained, none trained", cats{catL}, nil, dex, cats{catL}, true, true},
		{"H untrained, Constitution", cats{catH}, cats{catL, catM, catS}, []ability.Ability{ability.Constitution}, cats{catH}, false, true},
		{"H untrained, Charisma", cats{catH}, cats{catL, catM, catS}, []ability.Ability{ability.Charisma}, cats{catH}, false, true},
		{"H untrained, no ability", cats{catH}, cats{catL, catM, catS}, nil, cats{catH}, false, true},
		{"nothing worn", nil, nil, str, nil, false, false},
		{"H untrained, both abilities", cats{catH}, nil, []ability.Ability{ability.Strength, ability.Dexterity}, cats{catH}, true, true},
		{"S alone, Strength", cats{catS}, nil, str, cats{catS}, false, false},
		{"S alone, Dexterity", cats{catS}, nil, dex, cats{catS}, false, false},
		{"H and S untrained", cats{catH, catS}, nil, str, cats{catH, catS}, true, true},
		{"H and S, H trained", cats{catH, catS}, cats{catH}, str, cats{catS}, false, false},
		{"H and S, S trained", cats{catH, catS}, cats{catS}, dex, cats{catH}, true, true},
		{"order and duplicates", cats{catS, catH, catS}, cats{catL, catL}, str, cats{catH, catS}, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, err := character.ArmorTrainingRules{}.Assess(c.worn, c.trained)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(e.Untrained(), c.untrained) || e.WearsUntrainedBodyArmor() != c.untrainedBodyArm {
				t.Errorf("Untrained = %v body %v, want %v body %v", e.Untrained(), e.WearsUntrainedBodyArmor(), c.untrained, c.untrainedBodyArm)
			}
			got := e.RollCircumstancesFor(c.involved)
			if c.disadvantage != (len(got) == 1 && got[0] == (modifier.Disadvantage{})) || len(got) > 1 {
				t.Errorf("circumstances = %v, want disadvantage %v", got, c.disadvantage)
			}
		})
	}
}

// Requirement: equipSpec FR-073 - an untrained Shield alone gives no
// circumstance for any ability set.
// Path: unhappy (rule interaction).
func TestUntrainedShieldNeverGivesDisadvantage(t *testing.T) {
	t.Parallel()
	e, err := character.ArmorTrainingRules{}.Assess(cats{catS}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range ability.All() {
		if got := e.RollCircumstancesFor([]ability.Ability{a}); len(got) != 0 {
			t.Errorf("Shield alone with %v = %v", a, got)
		}
	}
	if got := e.RollCircumstancesFor(nil); len(got) != 0 {
		t.Errorf("Shield alone with no ability = %v", got)
	}
}

// Requirement: per case equipSpec FR-074 (SRD 5.2.1, "One at a Time", p. 92) -
// two or three body armors are rejected; an empty worn is valid.
// Path: unhappy (invalid input, boundary).
func TestArmorTrainingOneAtATime(t *testing.T) {
	t.Parallel()
	for _, worn := range []cats{{catL, catH}, {catL, catM, catH}, {catM, catH}, {catL, catM}} {
		_, err := character.ArmorTrainingRules{}.Assess(worn, cats{catL, catM, catH})
		wantInvalid(t, err, "two body armors")
	}
	_, err := character.ArmorTrainingRules{}.Assess(cats{catL, catS, catS}, nil)
	if err != nil {
		t.Errorf("one body armor and a Shield (twice listed) = %v, want valid", err)
	}
	if e, err := (character.ArmorTrainingRules{}).Assess(nil, nil); err != nil || len(e.Untrained()) != 0 {
		t.Errorf("empty worn = %v, %v", e, err)
	}
	_, err = character.ArmorTrainingRules{}.Assess(cats{0}, nil)
	wantInvalid(t, err, "invalid worn category")
	_, err = character.ArmorTrainingRules{}.Assess(nil, cats{content.ArmorCategory(9)})
	wantInvalid(t, err, "invalid trained category")
}

// Requirement: equipSpec FR-076 - a monster is trained in its stat block's
// armor: nothing is untrained and no circumstance arises.
// Path: happy, unhappy (rule interaction).
func TestMonsterTraining(t *testing.T) {
	t.Parallel()
	for _, worn := range []cats{{catM}, {catH, catS}} {
		trained := character.ArmorTrainingRules{}.MonsterTraining(worn)
		if !slices.Equal(trained, worn) {
			t.Fatalf("MonsterTraining(%v) = %v", worn, trained)
		}
		e, err := character.ArmorTrainingRules{}.Assess(worn, trained)
		if err != nil || len(e.Untrained()) != 0 {
			t.Fatalf("Assess = %v, %v", e, err)
		}
		both := []ability.Ability{ability.Strength, ability.Dexterity}
		if got := e.RollCircumstancesFor(both); len(got) != 0 {
			t.Errorf("circumstances = %v", got)
		}
	}
	src := cats{catM}
	got := character.ArmorTrainingRules{}.MonsterTraining(src)
	got[0] = catH
	if src[0] != catM {
		t.Error("MonsterTraining returned the caller's slice")
	}
}

// Requirement: equipSpec FR-077 (SRD 5.2.1, "They Don't Stack", p. 8) - the
// circumstance list is accepted by modifier.Resolve: alone it gives
// disadvantage, with one Advantage it gives normal.
// Path: unhappy (rule interaction).
func TestArmorTrainingResolves(t *testing.T) {
	t.Parallel()
	e, err := character.ArmorTrainingRules{}.Assess(cats{catH}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cs := e.RollCircumstancesFor([]ability.Ability{ability.Strength})
	if got := modifier.Resolve(cs...); got != modifier.WithDisadvantage {
		t.Errorf("alone = %v", got)
	}
	if got := modifier.Resolve(append(cs, modifier.Advantage{})...); got != modifier.Normal {
		t.Errorf("with Advantage = %v", got)
	}
}

// Requirement: equipSpec FR-078 - the effects are an immutable value: inputs
// are copied and unaltered, input order is irrelevant, Untrained is a fresh
// slice, and Equal compares by the untrained categories.
// Path: unhappy (immutability).
func TestArmorTrainingImmutable(t *testing.T) {
	t.Parallel()
	worn, trained := cats{catH, catS}, cats{catL}
	e, err := character.ArmorTrainingRules{}.Assess(worn, trained)
	if err != nil {
		t.Fatal(err)
	}
	worn[0], trained[0] = catM, catH
	if want := (cats{catH, catS}); !slices.Equal(e.Untrained(), want) {
		t.Errorf("after mutating inputs Untrained = %v, want %v", e.Untrained(), want)
	}
	u := e.Untrained()
	u[0] = catL
	if want := (cats{catH, catS}); !slices.Equal(e.Untrained(), want) {
		t.Errorf("after mutating the result Untrained = %v, want %v", e.Untrained(), want)
	}
	a, _ := character.ArmorTrainingRules{}.Assess(cats{catS, catH}, nil)
	b, _ := character.ArmorTrainingRules{}.Assess(cats{catH, catS, catH}, cats{catL})
	c, _ := character.ArmorTrainingRules{}.Assess(cats{catH}, nil)
	if !a.Equal(b) || a.Equal(c) {
		t.Error("Equal should hold for the same untrained set and fail for a different one")
	}
	if !(character.ArmorTrainingEffects{}).Equal(character.ArmorTrainingEffects{}) {
		t.Error("zero effects should be equal")
	}
}
