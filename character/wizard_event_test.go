package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-wizard.md FR-091 - every Wizard event is a past-tense value that is a WizardEvent and an Event carrying the CharacterID of the command, so two characters' events are distinguishable;
// the accessors return what the use case was given.
// Path: happy / unhappy (immutability).
func TestWizardEventsCarryTheCharacter(t *testing.T) {
	t.Parallel()
	other := character.MustIdentifier("wiz-2")
	src := wzSource(t)
	var events []character.Event
	res, _ := must(character.NewChooseWizardSkills(src)).Run(t.Context(), character.ChooseWizardSkillsCommand{CharacterID: wzID, Skills: []content.Skill{content.SkillArcana, content.SkillHistory}})
	events = append(events, res.(character.WizardSkillsApplied).Events()...)
	eq, _ := must(character.NewChooseWizardEquipment(src)).Run(t.Context(), character.ChooseWizardEquipmentCommand{CharacterID: wzID, Option: "A"})
	events = append(events, eq.(character.WizardEquipmentApplied).Events()...)
	cant, _ := must(character.NewChooseWizardCantrips(src)).Run(t.Context(), character.ChooseWizardCantripsCommand{CharacterID: wzID, ClassLevels: wzLevels(1), Chosen: wzFacts("c", 0, 3)})
	events = append(events, cant.(character.WizardCantripsApplied).Events()...)
	sch, _ := must(character.NewChooseScholarExpertise(src)).Run(t.Context(), character.ChooseScholarExpertiseCommand{CharacterID: wzID, ClassLevels: wzLevels(2), Proficient: []content.Skill{content.SkillArcana}, Skill: content.SkillArcana})
	events = append(events, sch.(character.ScholarApplied).Events()...)
	start, _ := must(character.NewStartSpellbook(src)).Run(t.Context(), character.StartSpellbookCommand{CharacterID: wzID, ClassLevels: wzLevels(1), Spells: wzFacts("a", 1, 6)})
	events = append(events, start.(character.SpellbookChanged).Events()...)
	ar, _ := must(character.NewUseArcaneRecovery(src)).Run(t.Context(), character.UseArcaneRecoveryCommand{CharacterID: wzID, ClassLevels: wzLevels(4), Expended: map[int]int{1: 1}, Chosen: map[int]int{1: 1}, Trigger: wzRest(t, character.ShortRest, false)})
	events = append(events, ar.(character.ArcaneRecoveryApplied).Events()...)
	mem, _ := must(character.NewMemorizeSpell(src)).Run(t.Context(), character.MemorizeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(5), Prepared: wzIDs("a"), Spellbook: wzBook("a", "b"), Replace: "a", New: wzFact("b", 1), Trigger: wzRest(t, character.ShortRest, false)})
	events = append(events, mem.(character.MemorizeApplied).Events()...)
	sm := must(character.NewChooseSpellMastery(src))
	mres, _ := sm.Run(t.Context(), character.ChooseSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzLevels(18), Spellbook: wzBook("one", "two"), Spells: []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2)}})
	events = append(events, mres.(character.SpellMasteryApplied).Events()...)
	rep, _ := must(character.NewReplaceSpellMastery(src)).Run(t.Context(), character.ReplaceSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzLevels(18), Resources: wzMasteryHeld(t), Spellbook: wzBook("one", "two", "o2"), Replace: "one", New: wzFact("o2", 1), Trigger: wzRest(t, character.LongRest, false)})
	events = append(events, rep.(character.SpellMasteryReplaceApplied).Events()...)
	sig, _ := must(character.NewChooseSignatureSpells(src)).Run(t.Context(), character.ChooseSignatureSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(20), Spellbook: wzBook("s1", "s2"), Spells: wzSigFacts()})
	events = append(events, sig.(character.SignatureSpellsApplied).Events()...)
	if len(events) != 10 {
		t.Fatalf("%d events, want 10", len(events))
	}
	for _, e := range events {
		if _, ok := e.(character.WizardEvent); !ok || e.CharacterID() != wzID || e.CharacterID() == other {
			t.Errorf("%T is not a WizardEvent of %v", e, wzID)
		}
		if _, record := e.(character.RecordEvent); record {
			t.Errorf("%T is in the aggregate's closed family", e)
		}
	}
	if sk := events[0].(character.WizardSkillsChosen); len(sk.Skills()) != 2 {
		t.Errorf("skills event = %+v", sk)
	}
	if ae := events[6].(character.SpellMemorized); ae.Replaced() != "a" || ae.Memorized() != "b" {
		t.Errorf("memorized event = %+v", ae)
	}
}
