package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// featsSpec is the spec every feats-behaviour test cites.
const featsSpec = "doc/specs/feats-behaviour.md"

// featOf is a synthetic feat of id and category, with the given prerequisites.
func featOf(id string, cat content.FeatCategory, repeatable bool, prereqs ...content.Prerequisite) content.Feat {
	f := content.FeatFields{Category: cat, Benefits: []content.FeatBenefit{content.MustFeatBenefit("Benefit", "A benefit.")}, Repeatable: repeatable}
	if repeatable {
		f.RepeatableText = "You can take this feat more than once."
	}
	if len(prereqs) > 0 {
		f.Prerequisites = prereqs
	}
	return content.MustFeat(content.FeatID(id), id, content.SourceCitation{}, f)
}

func levelReq(n int) content.Prerequisite { return must(content.NewLevelPrerequisite(n)) }

func scoreReq(min int, abilities ...ability.Ability) content.Prerequisite {
	return must(content.NewAbilityScorePrerequisite(abilities, min))
}

func featureReq(name string) content.Prerequisite { return must(content.NewFeaturePrerequisite(name)) }

// The feats of table C as the SRD prints them (categories and prerequisites).
func alertFeat() content.Feat    { return featOf("alert", content.FeatOrigin, false) }
func savageFeat() content.Feat   { return featOf("savage_attacker", content.FeatOrigin, false) }
func initiateFeat() content.Feat { return featOf("magic_initiate", content.FeatOrigin, true) }
func skilledFeat() content.Feat  { return featOf("skilled", content.FeatOrigin, true) }
func asiFeat() content.Feat {
	return featOf("ability_score_improvement", content.FeatGeneral, false, levelReq(4))
}
func grapplerFeat() content.Feat {
	return featOf("grappler", content.FeatGeneral, false, levelReq(4), scoreReq(13, ability.Strength, ability.Dexterity))
}
func archeryFeat() content.Feat {
	return featOf("archery", content.FeatFightingStyle, false, featureReq("Fighting Style Feature"))
}
func defenseFeat() content.Feat {
	return featOf("defense", content.FeatFightingStyle, false, featureReq("Fighting Style Feature"))
}
func boonFeat(id string, extra ...content.Prerequisite) content.Feat {
	return featOf(id, content.FeatEpicBoon, false, append([]content.Prerequisite{levelReq(19)}, extra...)...)
}
func recallFeat() content.Feat {
	return boonFeat("boon_of_spell_recall", featureReq("Spellcasting Feature"))
}

// charAt is a Fighter of the given level with the given scores (str, dex, con, int, wis, cha).
func charAt(t *testing.T, level int, scores ...int) character.Character {
	t.Helper()
	s := flatScores(t, 10)
	if len(scores) == 6 {
		s = scoresOf(t, scores[0], scores[1], scores[2], scores[3], scores[4], scores[5])
	}
	return restored(t, 0, 50, s, cl("fighter", level))
}

// fightingStyle is the feature a Fighter of level 1 holds.
func fightingStyleHeld(t *testing.T) character.HeldFeatures {
	t.Helper()
	return must(character.NewHeldFeatures(content.FeatureID("fighter_fighting_style")))
}

// grantOf builds a grant of scope from source.
func grantOf(source character.FeatSource, scope character.FeatGrantScope) character.FeatGrant {
	return must(character.NewFeatGrant(source, scope))
}

var (
	anyGrant = func() character.FeatGrant {
		return grantOf(character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_asi"}, character.AnyFeat{})
	}
	soldierSrc = character.BackgroundFeat{Background: "soldier"}
)

// miSpells are the synthetic spells of the Magic Initiate tests.
func miSpells(t *testing.T) map[content.SpellID]content.Spell {
	t.Helper()
	allThree := []string{"cleric", "druid", "wizard"}
	return spellMap(t,
		spellSpec{"guidance", 0, []string{"cleric", "druid"}}, spellSpec{"light", 0, []string{"cleric", "wizard"}},
		spellSpec{"mending", 0, []string{"cleric", "wizard", "druid"}}, spellSpec{"fire_bolt", 0, []string{"wizard"}},
		spellSpec{"shillelagh", 0, []string{"druid"}}, spellSpec{"vicious_mockery", 0, []string{"bard"}},
		spellSpec{"bless", 1, []string{"cleric"}}, spellSpec{"cure_wounds", 1, []string{"cleric", "druid"}},
		spellSpec{"shield", 1, []string{"wizard"}}, spellSpec{"magic_missile", 1, []string{"wizard"}},
		spellSpec{"cantrip_a", 0, allThree}, spellSpec{"cantrip_b", 0, allThree}, spellSpec{"cantrip_c", 0, allThree},
		spellSpec{"lvl1_a", 1, allThree}, spellSpec{"lvl1_b", 1, allThree}, spellSpec{"lvl1_c", 1, allThree},
		spellSpec{"sleep", 2, []string{"wizard"}}, spellSpec{"hold_person", 2, []string{"cleric", "wizard"}})
}

func spellIDsOf(ids ...string) []content.SpellID {
	out := make([]content.SpellID, len(ids))
	for i, id := range ids {
		out[i] = content.SpellID(id)
	}
	return out
}

// miOf builds Magic Initiate choices.
func miOf(list string, ab ability.Ability, cantrips []string, levelOne string) character.MagicInitiateChoices {
	return must(character.NewMagicInitiateChoices(content.ClassID(list), ab, spellIDsOf(cantrips...), content.SpellID(levelOne)))
}

func clericInitiate() character.MagicInitiateChoices {
	return miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "bless")
}

func wizardInitiate() character.MagicInitiateChoices {
	return miOf("wizard", ability.Intelligence, []string{"light", "fire_bolt"}, "shield")
}

func singleOf(a ability.Ability) character.SingleAbilityIncrease {
	return must(character.NewSingleAbilityIncrease(a))
}

func skilledOf(picks ...character.SkilledPick) character.SkilledChoices {
	return must(character.NewSkilledChoices(picks))
}

// takeOK runs FeatTakingRules.Take and requires a change.
func takeOK(t *testing.T, c character.Character, feats character.TakenFeats, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices, held character.HeldFeatures) character.FeatTakingChanged {
	t.Helper()
	tk, err := character.FeatTakingRules{}.Take(c, feats, feat, grant, choices, held, miSpells(t))
	if err != nil {
		t.Fatalf("Take: %v", err)
	}
	ch, ok := tk.(character.FeatTakingChanged)
	if !ok {
		t.Fatalf("Take = %#v, want a change", tk)
	}
	return ch
}

// takeRejected runs Take and returns the rejection.
func takeRejected(t *testing.T, c character.Character, feats character.TakenFeats, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices, held character.HeldFeatures) character.FeatRejection {
	t.Helper()
	tk, err := character.FeatTakingRules{}.Take(c, feats, feat, grant, choices, held, miSpells(t))
	if err != nil {
		t.Fatalf("Take: %v", err)
	}
	rj, ok := tk.(character.FeatTakingRejected)
	if !ok {
		t.Fatalf("Take = %#v, want a rejection", tk)
	}
	return rj.Rejection
}

// heldFeats builds TakenFeats by taking each (feat, choices) in turn at level 20 with a Fighting Style and Spellcasting.
func heldFeats(t *testing.T, pairs ...any) character.TakenFeats {
	t.Helper()
	feats := character.TakenFeats{}
	c := charAt(t, 20, 14, 14, 10, 10, 10, 10)
	held := must(character.NewHeldFeatures(content.FeatureID("fighter_fighting_style"), content.FeatureID("wizard_spellcasting")))
	for i := 0; i+1 < len(pairs); i += 2 {
		ch := takeOK(t, c, feats, pairs[i].(content.Feat), anyGrant(), pairs[i+1].(character.FeatChoices), held)
		feats = ch.Feats()
	}
	return feats
}

// charWithID is a Fighter of the given level and id with flat scores of 10.
func charWithID(t *testing.T, id string, level int) character.Character {
	t.Helper()
	lv := levels(cl("fighter", level))
	c, err := character.Restore(character.Parts{
		ID: character.MustIdentifier(id), ExperiencePoints: character.ExperiencePoints{}, ClassLevels: lv, Species: "human", Background: "soldier",
		AbilityScores: flatScores(t, 10), HitPointMaximum: character.MustHitPointMaximum(50),
		HitDicePool:       character.MustHitDicePoolFromLevels(lv, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}),
		HeroicInspiration: character.NotInspired,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
