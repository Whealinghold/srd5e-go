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

// Requirement: per case
// Path: per case.
func TestNewTakenFeat(t *testing.T) {
	t.Parallel()
	src := character.BackgroundFeat{Background: "soldier"}
	for _, c := range []struct {
		name, requirement, path string
		id                      content.FeatID
		cat                     content.FeatCategory
		source                  character.FeatSource
		choices                 character.FeatChoices
		ok                      bool
		limited                 bool
	}{
		{"Alert", featsSpec + " FR-061, Table C", "happy", "alert", content.FeatOrigin, src, character.NoFeatChoices{}, true, false},
		{"Magic Initiate starts with its free cast available", featsSpec + " FR-032, FR-061", "happy", "magic_initiate", content.FeatOrigin, src, clericInitiate(), true, true},
		{"Boon of Fate starts with Improve Fate available", featsSpec + " FR-050, FR-061", "happy", "boon_of_fate", content.FeatEpicBoon, src, singleOf(ability.Wisdom), true, true},
		{"Skilled", featsSpec + " FR-061", "happy", "skilled", content.FeatOrigin, src, skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillArcana}, character.SkilledSkill{Skill: content.SkillHistory}), true, false},
		{"a feat with no behaviour", featsSpec + " FR-061", "unhappy (invalid input)", "house_feat", content.FeatGeneral, src, character.NoFeatChoices{}, false, false},
		{"the wrong choices shape", featsSpec + " FR-061, Table C", "unhappy (invalid input)", "alert", content.FeatOrigin, src, singleOf(ability.Wisdom), false, false},
		{"missing choices", featsSpec + " FR-061, Table C", "unhappy (invalid input)", "grappler", content.FeatGeneral, src, character.NoFeatChoices{}, false, false},
		{"nil choices", featsSpec + " FR-061", "unhappy (invalid input)", "alert", content.FeatOrigin, src, nil, false, false},
		{"an invalid category", featsSpec + " FR-061", "unhappy (invalid input)", "alert", 0, src, character.NoFeatChoices{}, false, false},
		{"a nil source", featsSpec + " FR-061", "unhappy (invalid input)", "alert", content.FeatOrigin, nil, character.NoFeatChoices{}, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewTakenFeat(c.id, c.cat, c.source, c.choices)
			if !c.ok {
				if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
					t.Errorf("= %v, %v [%s]", got, err, c.requirement)
				}
				return
			}
			if err != nil || got.IsZero() || got.Feat() != c.id || got.Category() != c.cat || got.Source() != c.source || (got.Use() != nil) != c.limited {
				t.Errorf("= %+v, %v [%s]", got, err, c.requirement)
			}
			if c.limited {
				if _, ok := got.Use().(character.FeatUseIsAvailable); !ok {
					t.Errorf("use = %v", got.Use())
				}
			}
			if !got.Equal(got) || got.Equal(character.TakenFeat{}) {
				t.Error("Equal")
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-057, FR-061, FR-062 - TakenFeats copies its feats on construction and on read, refuses a zero feat, more than 64 feats and two Magic Initiate entries with one spell list, counts the feats taken from a source, and compares by value.
// Path: per case.
func TestTakenFeats(t *testing.T) {
	t.Parallel()
	soldier := character.BackgroundFeat{Background: "soldier"}
	versatile := character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}
	alert := must(character.NewTakenFeat("alert", content.FeatOrigin, soldier, character.NoFeatChoices{}))
	savage := must(character.NewTakenFeat("savage_attacker", content.FeatOrigin, versatile, character.NoFeatChoices{}))
	cleric := must(character.NewTakenFeat("magic_initiate", content.FeatOrigin, soldier, clericInitiate()))
	input := []character.TakenFeat{alert, savage}
	feats := must(character.NewTakenFeats(input...))
	input[0] = savage
	feats.All()[1] = alert
	if got := feats.All(); !got[0].Equal(alert) || !got[1].Equal(savage) || feats.Len() != 2 {
		t.Error("TakenFeats must copy on construction and on read")
	}
	if feats.TakenFrom(soldier) != 1 || feats.TakenFrom(versatile) != 1 || feats.TakenFrom(character.BonusFeat{}) != 0 || feats.TakenFrom(character.BackgroundFeat{Background: "sage"}) != 0 {
		t.Error("TakenFrom must match on the kind and names of the source")
	}
	if !feats.Holds("alert") || feats.Holds("grappler") || (character.TakenFeats{}).Holds("alert") || (character.TakenFeats{}).Len() != 0 {
		t.Error("Holds")
	}
	if !feats.Equal(must(character.NewTakenFeats(alert, savage))) || feats.Equal(must(character.NewTakenFeats(savage, alert))) || feats.Equal(character.TakenFeats{}) {
		t.Error("Equal must compare the feats in order")
	}
	if _, err := character.NewTakenFeats(alert, character.TakenFeat{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero feat: %v", err)
	}
	if _, err := character.NewTakenFeats(cleric, cleric); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("two Magic Initiate on one list: %v", err)
	}
	if _, err := character.NewTakenFeats(cleric, must(character.NewTakenFeat("magic_initiate", content.FeatOrigin, versatile, wizardInitiate()))); err != nil {
		t.Errorf("two lists: %v", err)
	}
	many := slices.Repeat([]character.TakenFeat{alert}, 65)
	if _, err := character.NewTakenFeats(many...); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("65 feats: %v", err)
	}
	if _, err := character.NewTakenFeats(many[:64]...); err != nil {
		t.Errorf("64 feats: %v", err)
	}
}

// Requirement: per case
// Path: per case.
func TestTakenFeatsReplacing(t *testing.T) {
	t.Parallel()
	src := character.BackgroundFeat{Background: "soldier"}
	mk := func(id string, cat content.FeatCategory, c character.FeatChoices) character.TakenFeat {
		return must(character.NewTakenFeat(content.FeatID(id), cat, src, c))
	}
	alert, savage, archery, defense := mk("alert", content.FeatOrigin, character.NoFeatChoices{}), mk("savage_attacker", content.FeatOrigin, character.NoFeatChoices{}),
		mk("archery", content.FeatFightingStyle, character.NoFeatChoices{}), mk("defense", content.FeatFightingStyle, character.NoFeatChoices{})
	feats := must(character.NewTakenFeats(alert, archery))
	for _, c := range []struct {
		name, requirement, path string
		old                     content.FeatID
		repl                    character.TakenFeat
		repeatable              bool
		want                    []character.TakenFeat
		rejection               character.FeatRejection
	}{
		{"a Fighting Style for another", featsSpec + ` FR-083; SRD 5.2.1, "Fighting Style" (Fighter), p. 47`, "happy", "archery", defense, false, []character.TakenFeat{alert, defense}, nil},
		{"a feat by itself", featsSpec + " FR-083", "unhappy (boundary)", "archery", archery, false, []character.TakenFeat{alert, archery}, nil},
		{"the old feat is not held", featsSpec + " FR-083", "unhappy (state error)", "defense", savage, false, nil, character.FeatNotHeld{Feat: "defense"}},
		{"another held feat, not repeatable", featsSpec + " FR-083", "unhappy (state error)", "archery", alert, false, nil, character.FeatNotRepeatable{Feat: "alert"}},
		{"another held feat, repeatable", featsSpec + " FR-083", "happy", "archery", alert, true, []character.TakenFeat{alert, alert}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, rj := feats.Replacing(c.old, c.repl, c.repeatable)
			if rj != c.rejection {
				t.Fatalf("rejection = %#v, want %#v [%s]", rj, c.rejection, c.requirement)
			}
			want, _ := character.NewTakenFeats(c.want...)
			if rj == nil && !got.Equal(want) {
				t.Errorf("feats = %v, want %v", got.All(), c.want)
			}
			if rj != nil && !got.Equal(feats) {
				t.Error("a rejection must return the feats unchanged")
			}
			if !feats.Equal(must(character.NewTakenFeats(alert, archery))) {
				t.Error("Replacing altered its receiver")
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-057 - TakenFeat compares the feat, the category, the source and the use state, whichever way each differs.
// Path: per case.
func TestTakenFeatEqualComparesEveryField(t *testing.T) {
	t.Parallel()
	soldier := character.BackgroundFeat{Background: "soldier"}
	mk := func(id string, cat content.FeatCategory, src character.FeatSource) character.TakenFeat {
		return must(character.NewTakenFeat(content.FeatID(id), cat, src, character.NoFeatChoices{}))
	}
	base := mk("defense", content.FeatGeneral, soldier)
	for name, other := range map[string]character.TakenFeat{
		"a greater feat":     mk("savage_attacker", content.FeatGeneral, soldier),
		"a lesser feat":      mk("alert", content.FeatGeneral, soldier),
		"a greater category": mk("defense", content.FeatEpicBoon, soldier),
		"a lesser category":  mk("defense", content.FeatOrigin, soldier),
		"another source":     mk("defense", content.FeatGeneral, character.CustomBackgroundFeat{}),
	} {
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("%s: feats compared equal", name)
		}
	}
	fate := must(character.NewTakenFeat("boon_of_fate", content.FeatEpicBoon, soldier, singleOf(ability.Wisdom)))
	spent := expend(t, must(character.NewTakenFeats(fate)), "boon_of_fate").(character.FeatsChanged).Feats().All()[0]
	if fate.Equal(spent) || spent.Equal(fate) || !spent.Equal(spent) {
		t.Error("the use state is part of the value")
	}
	if !(character.TakenFeat{}).IsZero() || base.IsZero() {
		t.Error("IsZero")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-083 - Replacing finds the old feat wherever it stands, the first in the order taken.
// Path: unhappy (boundary).
func TestTakenFeatsReplacingTheFirstFeat(t *testing.T) {
	t.Parallel()
	src := character.BackgroundFeat{Background: "soldier"}
	alert := must(character.NewTakenFeat("alert", content.FeatOrigin, src, character.NoFeatChoices{}))
	savage := must(character.NewTakenFeat("savage_attacker", content.FeatOrigin, src, character.NoFeatChoices{}))
	feats := must(character.NewTakenFeats(alert, savage))
	got, rj := feats.Replacing("alert", savage, true)
	if rj != nil || !got.Equal(must(character.NewTakenFeats(savage, savage))) {
		t.Errorf("= %v, %v", got.All(), rj)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-040 - ProficiencyGrants is empty only when it grants no skill and no tool.
// Path: unhappy (boundary).
func TestProficiencyGrantsIsZero(t *testing.T) {
	t.Parallel()
	skillsOnly := heldFeats(t, skilledFeat(), skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillArcana}, character.SkilledSkill{Skill: content.SkillHistory})).ProficiencyGrants()
	toolsOnly := heldFeats(t, skilledFeat(), skilledOf(character.SkilledTool{Tool: "thieves_tools"}, character.SkilledTool{Tool: "herbalism_kit"}, character.SkilledTool{Tool: "gaming_set"})).ProficiencyGrants()
	if skillsOnly.IsZero() || toolsOnly.IsZero() || !(character.ProficiencyGrants{}).IsZero() || !(character.TakenFeats{}).ProficiencyGrants().IsZero() {
		t.Error("IsZero must be true only when nothing is granted")
	}
}
