package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-wizard.md FR-061 to FR-064 - Memorize Spell from Wizard level 5 after a finished Short Rest replaces one prepared spell with a level 1+ spell from the book of a level the Wizard has
// slots for and not already prepared; each cause is a variant in order (level, trigger, not prepared, always prepared, cantrip, not in book, too high, already prepared) and the result is the new prepared list.
// Path: per case (happy / unhappy: invalid input, state error, rule interaction).
func TestMemorizeSpell(t *testing.T) {
	t.Parallel()
	uc := must(character.NewMemorizeSpell(wzSource(t)))
	short, long := wzRest(t, character.ShortRest, false), wzRest(t, character.LongRest, false)
	book := wzBook("a", "b", "c", "d", "hi", "cant")
	always := wzMasteryHeld(t) // one and two are always prepared
	type rej = character.MemorizeRejection
	for _, c := range []struct {
		name     string
		level    int
		trigger  character.RestApplied
		prepared []content.SpellID
		replace  content.SpellID
		next     character.WizardSpellFacts
		res      character.WizardResources
		want     rej
	}{
		{"replace", 5, short, wzIDs("a", "b"), "a", wzFact("c", 1), character.WizardResources{}, rej{}},
		{"highest level", 5, short, wzIDs("a", "b"), "b", wzFact("d", 3), character.WizardResources{}, rej{}},
		{"level 4", 4, short, wzIDs("a", "b"), "a", wzFact("c", 1), character.WizardResources{}, rej{Kind: character.MemorizeFeatureNotAvailable}},
		{"long rest", 5, long, wzIDs("a", "b"), "a", wzFact("c", 1), character.WizardResources{}, rej{Kind: character.MemorizeNotAfterShortRest}},
		{"interrupted long rest", 5, wzRest(t, character.LongRest, true), wzIDs("a", "b"), "a", wzFact("c", 1), character.WizardResources{}, rej{Kind: character.MemorizeNotAfterShortRest}},
		{"not prepared", 5, short, wzIDs("a", "b"), "zz", wzFact("c", 1), character.WizardResources{}, rej{Kind: character.MemorizeNotPrepared, Spell: "zz"}},
		{"always prepared", 18, short, wzIDs("a", "one"), "one", wzFact("c", 1), always, rej{Kind: character.MemorizeAlwaysPreparedNotReplaceable, Spell: "one"}},
		{"new is a cantrip", 5, short, wzIDs("a", "b"), "a", wzFact("cant", 0), character.WizardResources{}, rej{Kind: character.MemorizeNewSpellIsCantrip, Spell: "cant"}},
		{"new not in book", 5, short, wzIDs("a", "b"), "a", wzFact("zz", 1), character.WizardResources{}, rej{Kind: character.MemorizeNewSpellNotInSpellbook, Spell: "zz"}},
		{"new too high", 5, short, wzIDs("a", "b"), "a", wzFact("hi", 4), character.WizardResources{}, rej{Kind: character.MemorizeNewSpellAboveHighestLevel, Spell: "hi", Highest: 3}},
		{"new already prepared", 5, short, wzIDs("a", "b"), "a", wzFact("b", 1), character.WizardResources{}, rej{Kind: character.MemorizeNewSpellAlreadyPrepared, Spell: "b"}},
	} {
		res, err := uc.Run(t.Context(), character.MemorizeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Resources: c.res, Prepared: c.prepared, Spellbook: book,
			Replace: c.replace, New: c.next, Trigger: c.trigger})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.MemorizeSpellRejected); !ok || got.Rejection != c.want {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.MemorizeApplied)
		ev, evOK := got.Events()[0].(character.SpellMemorized)
		want := slices.Clone(c.prepared)
		want[slices.Index(want, c.replace)] = c.next.ID
		if !ok || !slices.Equal(got.Prepared(), want) || !evOK || ev.Replaced() != c.replace || ev.Memorized() != c.next.ID || ev.CharacterID() != wzID {
			t.Errorf("%s = %#v, want %v", c.name, res, want)
		}
		got.Prepared()[0] = "zz"
		if got.Prepared()[0] == "zz" {
			t.Errorf("%s: shared state", c.name)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-041, FR-042, FR-043, FR-044, FR-102 - Ritual Adept is advisory: a Ritual spell in a readable spellbook is eligible whether or not it is prepared; an unreadable book,
// a spell not in the book and a spell without the Ritual tag are each a variant, in that order.
// Path: per case (happy / unhappy: rule interaction, state error).
func TestCheckRitualAdept(t *testing.T) {
	t.Parallel()
	uc := character.NewCheckRitualAdept()
	book := wzBook("ritual", "plain")
	type rej = character.RitualAdeptRejection
	for _, c := range []struct {
		name      string
		spell     character.WizardSpellFacts
		available bool
		want      rej
	}{
		{"eligible", wzFact("ritual", 1, wzRitual), true, rej{}},
		{"unavailable book", wzFact("ritual", 1, wzRitual), false, rej{Kind: character.RitualAdeptSpellbookUnavailable}},
		{"not in book", wzFact("elsewhere", 1, wzRitual), true, rej{Kind: character.RitualAdeptNotInSpellbook}},
		{"not a Ritual", wzFact("plain", 1), true, rej{Kind: character.RitualAdeptNotARitualSpell}},
		{"unavailable and not in book", wzFact("elsewhere", 1), false, rej{Kind: character.RitualAdeptSpellbookUnavailable}},
		{"not in book and not a Ritual", wzFact("elsewhere", 1), true, rej{Kind: character.RitualAdeptNotInSpellbook}},
	} {
		res, err := uc.Run(t.Context(), character.CheckRitualAdeptCommand{ClassLevels: wzLevels(1), Spellbook: book, Spell: c.spell, SpellbookAvailable: c.available})
		if err != nil {
			t.Fatal(err)
		}
		if c.want.Kind == 0 {
			if _, ok := res.(character.RitualAdeptEligible); !ok {
				t.Errorf("%s = %#v", c.name, res)
			}
		} else if got, ok := res.(character.CheckRitualAdeptRejected); !ok || got.Rejection != c.want {
			t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-053 to FR-056 - Scholar takes, from Wizard level 2, one skill of the six in which the character is proficient and gives it as Expertise (stored in the new
// WizardResources); below level 2 it is FeatureNotAvailable, a second choice is Already, Insight is SkillNotEligible and a non-proficient skill is NotProficient, in that order.
// Path: per case (happy / unhappy: state error, invalid input).
func TestChooseScholarExpertise(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseScholarExpertise(wzSource(t)))
	proficient := []content.Skill{content.SkillArcana, content.SkillInsight, content.SkillHistory}
	held := must(uc.Run(t.Context(), character.ChooseScholarExpertiseCommand{CharacterID: wzID, ClassLevels: wzLevels(2), Proficient: proficient, Skill: content.SkillHistory})).(character.ScholarApplied).Resources()
	type rej = character.ScholarRejection
	for _, c := range []struct {
		name      string
		level     int
		resources character.WizardResources
		skill     content.Skill
		want      rej
	}{
		{"level 2", 2, character.WizardResources{}, content.SkillArcana, rej{}},
		{"level 20", 20, character.WizardResources{}, content.SkillArcana, rej{}},
		{"level 1", 1, character.WizardResources{}, content.SkillArcana, rej{Kind: character.ScholarFeatureNotAvailable}},
		{"second choice", 5, held, content.SkillArcana, rej{Kind: character.ScholarAlready}},
		{"Insight", 2, character.WizardResources{}, content.SkillInsight, rej{Kind: character.ScholarSkillNotEligible, Skill: content.SkillInsight}},
		{"Athletics", 2, character.WizardResources{}, content.SkillAthletics, rej{Kind: character.ScholarSkillNotEligible, Skill: content.SkillAthletics}},
		{"not proficient", 2, character.WizardResources{}, content.SkillReligion, rej{Kind: character.ScholarNotProficient, Skill: content.SkillReligion}},
	} {
		res, err := uc.Run(t.Context(), character.ChooseScholarExpertiseCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Resources: c.resources, Proficient: proficient, Skill: c.skill})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.ChooseScholarExpertiseRejected); !ok || got.Rejection != c.want {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.ScholarApplied)
		skill, has := got.Resources().Scholar()
		ev, evOK := got.Events()[0].(character.ScholarExpertiseChosen)
		if !ok || !has || skill != c.skill || got.Skill() != c.skill || !evOK || ev.Skill() != c.skill || ev.CharacterID() != wzID {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
	if _, has := (character.WizardResources{}).Scholar(); has {
		t.Error("the input state must not change")
	}
}
