package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-wizard.md FR-004, FR-005, FR-006 - the Wizard skills are exactly 2 distinct skills from the listed seven: other counts are WrongSkillCount(count), a repeat is
// DuplicateSkill, a skill outside the list (Athletics) is SkillNotAllowed naming the first offender; the accepted result carries the skills and one event.
// Path: per case (happy / unhappy: invalid input).
func TestChooseWizardSkills(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseWizardSkills(wzSource(t)))
	arcana, history, athletics, religion := content.SkillArcana, content.SkillHistory, content.SkillAthletics, content.SkillReligion
	type rej = character.SkillsRejection
	for _, c := range []struct {
		name   string
		skills []content.Skill
		want   rej
	}{
		{"two", []content.Skill{arcana, history}, rej{}},
		{"first and last of the list", []content.Skill{arcana, religion}, rej{}},
		{"none", nil, rej{Kind: character.SkillsWrongSkillCount}},
		{"one", []content.Skill{arcana}, rej{Kind: character.SkillsWrongSkillCount, Count: 1}},
		{"three", []content.Skill{arcana, history, religion}, rej{Kind: character.SkillsWrongSkillCount, Count: 3}},
		{"twice", []content.Skill{arcana, arcana}, rej{Kind: character.SkillsDuplicateSkill, Skill: arcana}},
		{"Athletics", []content.Skill{arcana, athletics}, rej{Kind: character.SkillsSkillNotAllowed, Skill: athletics}},
		{"first offender", []content.Skill{athletics, content.SkillStealth}, rej{Kind: character.SkillsSkillNotAllowed, Skill: athletics}},
		{"repeat before list", []content.Skill{athletics, athletics}, rej{Kind: character.SkillsDuplicateSkill, Skill: athletics}},
	} {
		res, err := uc.Run(t.Context(), character.ChooseWizardSkillsCommand{CharacterID: wzID, Skills: c.skills})
		if err != nil {
			t.Fatal(err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.ChooseWizardSkillsRejected); !ok || got.Rejection != c.want {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.WizardSkillsApplied)
		ev, evOK := got.Events()[0].(character.WizardSkillsChosen)
		if !ok || !slices.Equal(got.Skills(), c.skills) || !evOK || ev.CharacterID() != wzID || !slices.Equal(ev.Skills(), c.skills) {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-009, FR-010, FR-011 - option A resolves to the definition's option exactly as held (2 Daggers, the Arcane Focus, Robe, Spellbook, Scholar's Pack and 5 GP),
// option B to 55 GP and no lines; anything else (C, lower case, empty) is UnknownEquipmentOption naming the raw value, not an error.
// Path: per case (happy / unhappy: invalid input).
func TestChooseWizardEquipment(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseWizardEquipment(wzSource(t)))
	def := wzDef(t)
	for option, want := range map[string]content.ClassEquipmentOption{"A": def.StartingEquipment().A(), "B": def.StartingEquipment().B()} {
		res, err := uc.Run(t.Context(), character.ChooseWizardEquipmentCommand{CharacterID: wzID, Option: option})
		got, ok := res.(character.WizardEquipmentApplied)
		if err != nil || !ok || got.Option() != option || got.Equipment().Gold() != want.Gold() || len(got.Equipment().Items()) != len(want.Items()) {
			t.Fatalf("option %s = %#v, %v", option, res, err)
		}
		if ev, evOK := got.Events()[0].(character.WizardEquipmentChosen); !evOK || ev.Option() != option || ev.CharacterID() != wzID {
			t.Errorf("event = %#v", got.Events())
		}
		got.Events()[0] = nil
		if got.Events()[0] == nil {
			t.Error("Events returned a shared slice")
		}
	}
	a := must(uc.Run(t.Context(), character.ChooseWizardEquipmentCommand{CharacterID: wzID, Option: "A"})).(character.WizardEquipmentApplied).Equipment()
	if a.Gold() != 5 || len(a.Items()) != 5 || a.Items()[0].Name() != "Dagger" || a.Items()[0].Quantity() != 2 {
		t.Errorf("option A = %d GP, %v", a.Gold(), a.Items())
	}
	if b := must(uc.Run(t.Context(), character.ChooseWizardEquipmentCommand{CharacterID: wzID, Option: "B"})).(character.WizardEquipmentApplied).Equipment(); b.Gold() != 55 || len(b.Items()) != 0 {
		t.Errorf("option B = %d GP, %v", b.Gold(), b.Items())
	}
	for _, option := range []string{"C", "a", "", "AB"} {
		res, err := uc.Run(t.Context(), character.ChooseWizardEquipmentCommand{CharacterID: wzID, Option: option})
		if got, ok := res.(character.ChooseWizardEquipmentRejected); err != nil || !ok || got.Rejection != (character.EquipmentRejection{Kind: character.EquipmentUnknownEquipmentOption, Option: option}) {
			t.Errorf("option %q = %#v, %v", option, res, err)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-027, FR-028, FR-029, FR-099 - a cantrip choice must be exactly the pending number (3 at level 1, 1 at level 4, none when full), distinct from each
// other and from those known, level 0 and on the Wizard list; each cause has its variant, in order, naming the first offender; the choice is the facts' ids.
// Path: per case (happy / unhappy: invalid input, boundary at the column).
func TestChooseWizardCantrips(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseWizardCantrips(wzSource(t)))
	three := wzFacts("c", 0, 3)
	type rej = character.CantripChoiceRejection
	for _, c := range []struct {
		name    string
		level   int
		current []content.SpellID
		chosen  []character.WizardSpellFacts
		want    rej
	}{
		{"three at level 1", 1, nil, three, rej{}},
		{"one at level 4", 4, wzIDs("c1", "c2", "c3"), []character.WizardSpellFacts{wzFact("c4", 0)}, rej{}},
		{"one at level 10", 10, wzIDs("c1", "c2", "c3", "c4"), []character.WizardSpellFacts{wzFact("c5", 0)}, rej{}},
		{"two at level 1", 1, nil, three[:2], rej{Kind: character.CantripChoiceWrongCantripCount, Allowed: 3}},
		{"four at level 1", 1, nil, append(slices.Clone(three), wzFact("cd", 0)), rej{Kind: character.CantripChoiceWrongCantripCount, Allowed: 3}},
		{"two when full", 3, wzIDs("c1", "c2", "c3"), three[:2], rej{Kind: character.CantripChoiceWrongCantripCount, Allowed: 0}},
		{"repeat", 1, nil, []character.WizardSpellFacts{three[0], three[1], three[0]}, rej{Kind: character.CantripChoiceDuplicateCantrip, Spell: "ca"}},
		{"already known", 4, wzIDs("ca", "cb", "cc"), []character.WizardSpellFacts{three[0]}, rej{Kind: character.CantripChoiceDuplicateCantrip, Spell: "ca"}},
		{"not a cantrip", 1, nil, []character.WizardSpellFacts{three[0], wzFact("s", 1), three[2]}, rej{Kind: character.CantripChoiceNotACantrip, Spell: "s"}},
		{"off the list", 1, nil, []character.WizardSpellFacts{three[0], three[1], wzFact("o", 0, wzOffList)}, rej{Kind: character.CantripChoiceNotOnWizardList, Spell: "o"}},
		{"level 1 spell before off list", 1, nil, []character.WizardSpellFacts{wzFact("o", 0, wzOffList), wzFact("s", 1), three[2]}, rej{Kind: character.CantripChoiceNotACantrip, Spell: "s"}},
	} {
		res, err := uc.Run(t.Context(), character.ChooseWizardCantripsCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Current: c.current, Chosen: c.chosen})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.ChooseWizardCantripsRejected); !ok || got.Rejection != c.want {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.WizardCantripsApplied)
		if !ok || len(got.Cantrips()) != len(c.chosen) || len(got.Events()) != 1 || got.Events()[0].(character.WizardCantripsChosen).CharacterID() != wzID {
			t.Errorf("%s = %#v", c.name, res)
			continue
		}
		got.Cantrips()[0] = "zz"
		got.Events()[0].(character.WizardCantripsChosen).Cantrips()[0] = "zz"
		if got.Cantrips()[0] == "zz" || got.Events()[0].(character.WizardCantripsChosen).Cantrips()[0] == "zz" {
			t.Errorf("%s: an accessor returned shared state", c.name)
		}
	}
	// A multiclass Wizard's cantrips follow the Wizard level, not the total level.
	multi := levels(cl("fighter", 5), cl("wizard", 1))
	if res, _ := uc.Run(t.Context(), character.ChooseWizardCantripsCommand{CharacterID: wzID, ClassLevels: multi, Chosen: three}); typeName(res) != "WizardCantripsApplied" {
		t.Errorf("multiclass = %#v", res)
	}
}
