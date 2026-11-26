package scenarios_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The scenarios of feats-behaviour.md cover the feat domain services (taking a
// feat, its effects, uses, replacement and spell change) through the public
// packages only. The five use cases of the spec (TakeFeat, ChangeMagicInitiateSpell,
// ExpendFeatUse, RenewFeatUses, RollFreeCasting) have their own scenarios. No feat is
// bundled yet (Phase 5), so the feats are synthetic values built to the printed
// categories and prerequisites of SRD 5.2.1, pp. 87-88.

func grpD3Feat(id string, cat content.FeatCategory, repeatable bool, prereqs ...content.Prerequisite) content.Feat {
	f := content.FeatFields{Category: cat, Benefits: []content.FeatBenefit{content.MustFeatBenefit("Benefit", "A benefit.")}, Repeatable: repeatable}
	if repeatable {
		f.RepeatableText = "You can take this feat more than once."
	}
	if len(prereqs) > 0 {
		f.Prerequisites = prereqs
	}
	return content.MustFeat(content.FeatID(id), id, content.SourceCitation{}, f)
}

func grpD3LevelReq(n int) content.Prerequisite {
	return grpD3Must(content.NewLevelPrerequisite(n))
}

func grpD3ScoreReq(minimum int, abilities ...ability.Ability) content.Prerequisite {
	return grpD3Must(content.NewAbilityScorePrerequisite(abilities, minimum))
}

func grpD3FeatureReq(name string) content.Prerequisite {
	return grpD3Must(content.NewFeaturePrerequisite(name))
}

// The feats of Table C as the SRD prints them (categories and prerequisites).
func grpD3Alert() content.Feat    { return grpD3Feat("alert", content.FeatOrigin, false) }
func grpD3Savage() content.Feat   { return grpD3Feat("savage_attacker", content.FeatOrigin, false) }
func grpD3Initiate() content.Feat { return grpD3Feat("magic_initiate", content.FeatOrigin, true) }
func grpD3Skilled() content.Feat  { return grpD3Feat("skilled", content.FeatOrigin, true) }
func grpD3ASI() content.Feat {
	return grpD3Feat("ability_score_improvement", content.FeatGeneral, false, grpD3LevelReq(4))
}
func grpD3Grappler() content.Feat {
	return grpD3Feat("grappler", content.FeatGeneral, false, grpD3LevelReq(4), grpD3ScoreReq(13, ability.Strength, ability.Dexterity))
}
func grpD3Archery() content.Feat {
	return grpD3Feat("archery", content.FeatFightingStyle, false, grpD3FeatureReq("Fighting Style Feature"))
}
func grpD3Defense() content.Feat {
	return grpD3Feat("defense", content.FeatFightingStyle, false, grpD3FeatureReq("Fighting Style Feature"))
}
func grpD3Boon(id string, extra ...content.Prerequisite) content.Feat {
	return grpD3Feat(id, content.FeatEpicBoon, false, append([]content.Prerequisite{grpD3LevelReq(19)}, extra...)...)
}

func grpD3Scores(str, dex, con, intl, wis, cha int) ability.Scores {
	return grpD3Must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: grpD3Score(str), ability.Dexterity: grpD3Score(dex), ability.Constitution: grpD3Score(con),
		ability.Intelligence: grpD3Score(intl), ability.Wisdom: grpD3Score(wis), ability.Charisma: grpD3Score(cha),
	}))
}

// grpD3Hero is a Fighter of the given level with the given scores, restored at the
// minimum experience of that level.
func grpD3Hero(id string, level int, scores ability.Scores) character.Character {
	lv := character.MustClassLevels([]character.ClassLevel{{Class: "fighter", Level: level}})
	return grpD3Must(character.Restore(character.Parts{
		ID: character.MustIdentifier(id), ExperiencePoints: character.ExperiencePoints{}.MinimumForLevel(character.MustLevel(level)), ClassLevels: lv,
		Species: "human", Background: "soldier", AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(50),
		HitDicePool:       character.MustHitDicePoolFromLevels(lv, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}),
		HeroicInspiration: character.NotInspired,
	}))
}

func grpD3FightingStyleHeld() character.HeldFeatures {
	return grpD3Must(character.NewHeldFeatures("fighter_fighting_style"))
}

func grpD3FullFeatures() character.HeldFeatures {
	return grpD3Must(character.NewHeldFeatures("fighter_fighting_style", "wizard_spellcasting"))
}

func grpD3Grant(source character.FeatSource, scope character.FeatGrantScope) character.FeatGrant {
	return grpD3Must(character.NewFeatGrant(source, scope))
}

func grpD3AnyGrant() character.FeatGrant {
	return grpD3Grant(character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_ability_score_improvement"}, character.AnyFeat{})
}

func grpD3StyleGrant() character.FeatGrant {
	return grpD3Grant(character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}, character.FeatFromCategory{Category: content.FeatFightingStyle})
}

type grpD3SpellSpec struct {
	id      string
	level   int
	classes []string
}

// grpD3Spells decodes synthetic spells (Illusion, magic action) onto the class lists given.
func grpD3Spells(specs ...grpD3SpellSpec) map[content.SpellID]content.Spell {
	docs := make([]string, len(specs))
	for i, s := range specs {
		quoted := make([]string, len(s.classes))
		for j, c := range s.classes {
			quoted[j] = `"` + c + `"`
		}
		docs[i] = fmt.Sprintf(`{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":[%s],"castingTimes":[{"kind":"magicAction"}],"ritual":false,"range":{"kind":"self"},"components":{"verbal":true,"somatic":false},
"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}`, s.id, s.level, strings.Join(quoted, ","))
	}
	set, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":1,"spells":[` + strings.Join(docs, ",") + `]}`))
	if err != nil {
		panic(err) // a broken fixture
	}
	out := map[content.SpellID]content.Spell{}
	for _, e := range set.Entries() {
		sp := e.(content.Spell)
		out[sp.ID()] = sp
	}
	return out
}

func grpD3InitiateSpells() map[content.SpellID]content.Spell {
	all := []string{"cleric", "druid", "wizard"}
	return grpD3Spells(
		grpD3SpellSpec{"guidance", 0, []string{"cleric", "druid"}}, grpD3SpellSpec{"light", 0, []string{"cleric", "wizard"}},
		grpD3SpellSpec{"mending", 0, all}, grpD3SpellSpec{"fire_bolt", 0, []string{"wizard"}},
		grpD3SpellSpec{"shillelagh", 0, []string{"druid"}}, grpD3SpellSpec{"vicious_mockery", 0, []string{"bard"}},
		grpD3SpellSpec{"bless", 1, []string{"cleric"}}, grpD3SpellSpec{"cure_wounds", 1, []string{"cleric", "druid"}},
		grpD3SpellSpec{"shield", 1, []string{"wizard"}}, grpD3SpellSpec{"magic_missile", 1, []string{"wizard"}},
		grpD3SpellSpec{"sleep", 2, []string{"wizard"}})
}

func grpD3IDsOfSpells(ids ...string) []content.SpellID {
	out := make([]content.SpellID, len(ids))
	for i, id := range ids {
		out[i] = content.SpellID(id)
	}
	return out
}

func grpD3MIChoices(list string, a ability.Ability, cantrips []string, levelOne string) character.MagicInitiateChoices {
	return grpD3Must(character.NewMagicInitiateChoices(content.ClassID(list), a, grpD3IDsOfSpells(cantrips...), content.SpellID(levelOne)))
}

func grpD3ClericInitiate() character.MagicInitiateChoices {
	return grpD3MIChoices("cleric", ability.Wisdom, []string{"guidance", "light"}, "bless")
}

func grpD3WizardInitiate() character.MagicInitiateChoices {
	return grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "fire_bolt"}, "shield")
}

func grpD3Single(a ability.Ability) character.SingleAbilityIncrease {
	return grpD3Must(character.NewSingleAbilityIncrease(a))
}

// grpD3Take runs FeatTakingRules.Take, failing the test on an error.
func grpD3Take(t *testing.T, c character.Character, feats character.TakenFeats, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices, held character.HeldFeatures) character.FeatTaking {
	t.Helper()
	tk, err := character.FeatTakingRules{}.Take(c, feats, feat, grant, choices, held, grpD3InitiateSpells())
	if err != nil {
		t.Fatalf("Take(%v): %v", feat.ID(), err)
	}
	return tk
}

func grpD3TakeChanged(t *testing.T, c character.Character, feats character.TakenFeats, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices, held character.HeldFeatures) character.FeatTakingChanged {
	t.Helper()
	ch, ok := grpD3Take(t, c, feats, feat, grant, choices, held).(character.FeatTakingChanged)
	if !ok {
		t.Fatalf("Take(%v) = %#v, want a change", feat.ID(), grpD3Take(t, c, feats, feat, grant, choices, held))
	}
	return ch
}

func grpD3TakeRejected(t *testing.T, c character.Character, feats character.TakenFeats, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices, held character.HeldFeatures) character.FeatRejection {
	t.Helper()
	rj, ok := grpD3Take(t, c, feats, feat, grant, choices, held).(character.FeatTakingRejected)
	if !ok {
		t.Fatalf("Take(%v) was not rejected", feat.ID())
	}
	return rj.Rejection
}

// grpD3Held takes each (feat, choices) pair in turn at level 20 with Fighting Style and Spellcasting.
func grpD3Held(t *testing.T, pairs ...any) character.TakenFeats {
	t.Helper()
	feats := character.TakenFeats{}
	c := grpD3Hero("held-1", 20, grpD3Scores(14, 14, 10, 10, 10, 10))
	for i := 0; i+1 < len(pairs); i += 2 {
		feats = grpD3TakeChanged(t, c, feats, pairs[i].(content.Feat), grpD3AnyGrant(), pairs[i+1].(character.FeatChoices), grpD3FullFeatures()).Feats()
	}
	return feats
}

// grpD3LevelGained levels a fighter to total level n through the real level-up use case
// over the bundled Fighter and returns the levelled character with the LevelGained event that unlocks a spell change.
func grpD3LevelUp(t *testing.T, id string, n int) (character.Character, character.LevelGained) {
	t.Helper()
	before := grpD3Hero(id, n-1, grpD3Scores(12, 12, 12, 12, 12, 12))
	before = grpD3Must(character.Restore(character.Parts{
		ID: before.ID(), ExperiencePoints: character.ExperiencePoints{}.MinimumForLevel(character.MustLevel(n)), ClassLevels: before.ClassLevels(),
		Species: "human", Background: "soldier", AbilityScores: before.AbilityScores(), HitPointMaximum: before.HitPointMaximum(),
		HitDicePool: before.HitDicePool(), HeroicInspiration: character.NotInspired,
	}))
	uc, err := character.NewLevelUpCharacter(character.WithSource(data.MustBundledSource()), character.WithRoller(dicetest.NewFixedRoller()))
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), character.LevelUpCharacterCommand{Character: before, Class: "fighter", HitPointChoice: character.TakeFixedValue{}})
	if err != nil {
		t.Fatal(err)
	}
	ok, isOK := res.(character.LevelUpCharacterSucceeded)
	if !isOK {
		t.Fatalf("level up to %d = %#v", n, res)
	}
	for _, e := range ok.Events() {
		if lg, is := e.(character.LevelGained); is {
			return ok.Character(), lg
		}
	}
	t.Fatal("the level-up produced no LevelGained")
	return character.Character{}, character.LevelGained{}
}

// A Soldier starts with Savage Attacker, takes Ability Score Improvement at
// level 4, and the rules refuse a second Savage Attacker (even from Versatile),
// the feat below its level and an increase past 20 (SRD 5.2.1, "Soldier" p. 83;
// "Ability Score Improvement" p. 87; "Parts of a Feat" p. 87).
// Requirement: doc/specs/feats-behaviour.md section 8 scenario, FR-001, FR-005, FR-009, FR-015, FR-020, FR-026, FR-044, FR-049, FR-079 - the background grant, one ASI increase applied with FeatTaken first, a non-repeatable feat is refused from any source, unmet and exceeded rules leave the character unchanged.
// Path: happy; unhappy (boundary, rule interaction, state error).
func TestGrpD3SoldierTakesSavageAttackerThenAbilityScoreImprovement(t *testing.T) {
	t.Parallel()
	soldier := grpD3Grant(character.BackgroundFeat{Background: "soldier"}, character.SpecifiedFeat{Feat: "savage_attacker", Required: character.NoRequiredSpellList{}})
	level1 := grpD3Hero("soldier-1", 1, grpD3Scores(15, 12, 14, 10, 10, 10))

	first := grpD3TakeChanged(t, level1, character.TakenFeats{}, grpD3Savage(), soldier, character.NoFeatChoices{}, character.HeldFeatures{})
	feats := first.Feats()
	if !feats.Holds("savage_attacker") || feats.Len() != 1 || feats.TakenFrom(character.BackgroundFeat{Background: "soldier"}) != 1 {
		t.Fatalf("after Savage Attacker: %v", feats.All())
	}
	if ev := first.Events(); len(ev) != 1 {
		t.Fatalf("events = %#v", ev)
	} else if taken, ok := ev[0].(character.FeatTaken); !ok || taken.Feat() != "savage_attacker" || taken.Category() != content.FeatOrigin || taken.CharacterID() != level1.ID() {
		t.Errorf("event = %#v", ev[0])
	}
	// The held feat declares its effect: roll damage twice, once per turn.
	if got := (character.FeatEffects{}).Effects(feats); len(got) != 1 || got[0] != (character.RollDamageTwice{Trigger: character.TriggerWeaponHit, Limit: character.OncePerTurn{}}) {
		t.Errorf("Savage Attacker effects = %#v", got)
	}

	// The same feat again through Versatile is not allowed, whatever its first source.
	versatile := grpD3Grant(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin})
	if rj := grpD3TakeRejected(t, level1, feats, grpD3Savage(), versatile, character.NoFeatChoices{}, character.HeldFeatures{}); rj != (character.FeatNotRepeatable{Feat: "savage_attacker"}) {
		t.Errorf("a second Savage Attacker = %#v", rj)
	}

	// Level 4: ASI by two on Strength 15 gives 17, with FeatTaken before the score change.
	level4 := grpD3Hero("soldier-1", 4, grpD3Scores(15, 12, 14, 10, 10, 10))
	asi := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	took := grpD3TakeChanged(t, level4, feats, grpD3ASI(), grpD3AnyGrant(), asi, character.HeldFeatures{})
	if got := took.Character().AbilityScores().Score(ability.Strength).Value(); got != 17 {
		t.Errorf("Strength after the increase = %d, want 17", got)
	}
	ev := took.Events()
	if len(ev) < 2 {
		t.Fatalf("events = %#v", ev)
	}
	if _, ok := ev[0].(character.FeatTaken); !ok {
		t.Errorf("the first event = %#v, want FeatTaken", ev[0])
	}
	if _, ok := ev[1].(character.AbilityScoreChanged); !ok {
		t.Errorf("the second event = %#v, want AbilityScoreChanged", ev[1])
	}
	if !took.Feats().Holds("ability_score_improvement") || !took.Feats().Holds("savage_attacker") || took.Feats().Len() != 2 {
		t.Errorf("feats after ASI = %v", took.Feats().All())
	}
	if level4.AbilityScores().Score(ability.Strength).Value() != 15 || feats.Len() != 1 {
		t.Error("Take altered its inputs")
	}

	// Below the feat's level, and past the maximum of 20: refused with the character unchanged.
	level3 := grpD3Hero("soldier-1", 3, grpD3Scores(15, 12, 14, 10, 10, 10))
	rj := grpD3TakeRejected(t, level3, feats, grpD3ASI(), grpD3AnyGrant(), asi, character.HeldFeatures{})
	if unmet, ok := rj.(character.FeatPrerequisiteNotMet); !ok || len(unmet.Unmet()) != 1 {
		t.Errorf("ASI at level 3 = %#v", rj)
	}
	nearMax := grpD3Hero("soldier-2", 4, grpD3Scores(19, 12, 14, 10, 10, 10))
	over := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	if rj := grpD3TakeRejected(t, nearMax, character.TakenFeats{}, grpD3ASI(), grpD3AnyGrant(), over, character.HeldFeatures{}); !grpD3IsIncreaseRejected(rj) {
		t.Errorf("ASI over the maximum = %#v", rj)
	}
	// One by two on a score of 18 stops exactly at 20.
	exact := grpD3Hero("soldier-3", 4, grpD3Scores(18, 12, 14, 10, 10, 10))
	if got := grpD3TakeChanged(t, exact, character.TakenFeats{}, grpD3ASI(), grpD3AnyGrant(), over, character.HeldFeatures{}).Character().AbilityScores().Score(ability.Strength).Value(); got != 20 {
		t.Errorf("Strength 18 + 2 = %d, want 20", got)
	}
	two := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementTwoByOne, ability.Dexterity, ability.Constitution))
	dexCon := grpD3Hero("soldier-4", 4, grpD3Scores(10, 19, 10, 10, 10, 10))
	sc := grpD3TakeChanged(t, dexCon, character.TakenFeats{}, grpD3ASI(), grpD3AnyGrant(), two, character.HeldFeatures{}).Character().AbilityScores()
	if sc.Score(ability.Dexterity).Value() != 20 || sc.Score(ability.Constitution).Value() != 11 {
		t.Errorf("Dexterity and Constitution by one = %d, %d; want 20, 11", sc.Score(ability.Dexterity).Value(), sc.Score(ability.Constitution).Value())
	}
}

func grpD3IsIncreaseRejected(r character.FeatRejection) bool {
	_, ok := r.(character.FeatIncreaseRejected)
	return ok
}

// Each source of a feat scopes what it may grant: a background names its feat (and
// for Magic Initiate its list), Versatile and a custom background an Origin feat,
// a class slot any feat or a Fighting Style, and a feat outside the scope is turned
// away with the character unchanged (SRD 5.2.1, pp. 24, 47-49, 83, 86-87).
// Requirement: doc/specs/feats-behaviour.md FR-001 to FR-004, FR-007, FR-076, Table G - SpecifiedFeat, FeatFromCategory and AnyFeat accept exactly their feats; a wrong list is a choices problem; repeatability comes from the feat.
// Path: happy; unhappy (invalid input, rule interaction).
func TestGrpD3FeatGrantScopesFollowTheirSource(t *testing.T) {
	t.Parallel()
	level4 := grpD3Hero("scope-1", 4, grpD3Scores(14, 14, 10, 10, 10, 10))
	level20 := grpD3Hero("scope-2", 20, grpD3Scores(14, 14, 10, 10, 10, 10))
	none := character.TakenFeats{}
	noChoices := character.NoFeatChoices{}

	acolyte := grpD3Grant(character.BackgroundFeat{Background: "acolyte"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: character.RequiredSpellListIs{Class: "cleric"}})
	criminal := grpD3Grant(character.BackgroundFeat{Background: "criminal"}, character.SpecifiedFeat{Feat: "alert", Required: character.NoRequiredSpellList{}})
	custom := grpD3Grant(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	versatile := grpD3Grant(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin})
	invocation := grpD3Grant(character.EldritchInvocationFeat{Option: "lessons_of_the_first_ones"}, character.FeatFromCategory{Category: content.FeatOrigin})
	bonus := grpD3Grant(character.BonusFeat{}, character.AnyFeat{})

	// Happy: the exact feat, the right category, any category.
	for _, c := range []struct {
		name    string
		feat    content.Feat
		grant   character.FeatGrant
		choices character.FeatChoices
		char    character.Character
	}{
		{"Acolyte: Magic Initiate on the Cleric list", grpD3Initiate(), acolyte, grpD3ClericInitiate(), level4},
		{"Criminal: Alert", grpD3Alert(), criminal, noChoices, level4},
		{"custom background: an Origin feat", grpD3Skilled(), custom, grpD3Must(character.NewSkilledChoices([]character.SkilledPick{character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillPerception}, character.SkilledSkill{Skill: content.SkillStealth}})), level4},
		{"Versatile: an Origin feat", grpD3Savage(), versatile, noChoices, level4},
		{"Lessons of the First Ones: an Origin feat", grpD3Alert(), invocation, noChoices, level4},
		{"Fighting Style slot: a Fighting Style", grpD3Archery(), grpD3StyleGrant(), noChoices, level4},
		{"an Ability Score Improvement slot: an Origin feat", grpD3Alert(), grpD3AnyGrant(), noChoices, level4},
		{"an Ability Score Improvement slot: a General feat", grpD3ASI(), grpD3AnyGrant(), grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0)), level4},
		{"an Ability Score Improvement slot: a Fighting Style", grpD3Archery(), grpD3AnyGrant(), noChoices, level4},
		{"an Ability Score Improvement slot: an Epic Boon", grpD3Boon("boon_of_fate"), grpD3AnyGrant(), grpD3Single(ability.Wisdom), level20},
		{"Bonus Feats at Level 20: any feat", grpD3Boon("boon_of_truesight"), bonus, grpD3Single(ability.Wisdom), level20},
	} {
		held := character.HeldFeatures{}
		if c.feat.FeatCategory() == content.FeatFightingStyle {
			held = grpD3FightingStyleHeld()
		}
		if _, ok := grpD3Take(t, c.char, none, c.feat, c.grant, c.choices, held).(character.FeatTakingChanged); !ok {
			t.Errorf("%s: not accepted: %#v", c.name, grpD3Take(t, c.char, none, c.feat, c.grant, c.choices, held))
		}
	}

	// Unhappy: each wrong feat is outside its grant and reported by id.
	for _, c := range []struct {
		name    string
		feat    content.Feat
		grant   character.FeatGrant
		choices character.FeatChoices
	}{
		{"Criminal slot given Savage Attacker", grpD3Savage(), criminal, noChoices},
		{"custom background given a General feat", grpD3ASI(), custom, grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))},
		{"Versatile given an Epic Boon", grpD3Boon("boon_of_fate"), versatile, grpD3Single(ability.Wisdom)},
		{"Fighting Style slot given a General feat", grpD3ASI(), grpD3StyleGrant(), grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))},
		{"Fighting Style slot given an Origin feat", grpD3Alert(), grpD3StyleGrant(), noChoices},
	} {
		rj := grpD3TakeRejected(t, level20, none, c.feat, c.grant, c.choices, grpD3FightingStyleHeld())
		if rj != (character.FeatOutsideGrant{Feat: c.feat.ID()}) {
			t.Errorf("%s: rejection = %#v, want FeatOutsideGrant", c.name, rj)
		}
	}

	// The Acolyte's Magic Initiate must be on the Cleric list; the Wizard list is a choices problem.
	rj := grpD3TakeRejected(t, level4, none, grpD3Initiate(), acolyte, grpD3WizardInitiate(), character.HeldFeatures{})
	bad, ok := rj.(character.FeatChoicesInvalid)
	if !ok || !slices.Contains(bad.Problems(), character.FeatChoiceProblem{Name: character.FeatChoiceSpellList, Kind: character.FeatProblemListNotAllowed}) {
		t.Errorf("Acolyte with the Wizard list = %#v", rj)
	}

	// Repeatability is read from the feat: a repeatable Fighting Style is accepted twice, a plain one is not.
	twice := grpD3Feat("two_weapon_fighting", content.FeatFightingStyle, true, grpD3FeatureReq("Fighting Style Feature"))
	first := grpD3TakeChanged(t, level4, none, twice, grpD3StyleGrant(), noChoices, grpD3FightingStyleHeld())
	if _, ok := grpD3Take(t, level4, first.Feats(), twice, grpD3StyleGrant(), noChoices, grpD3FightingStyleHeld()).(character.FeatTakingChanged); !ok {
		t.Error("a repeatable feat was refused the second time")
	}
	firstArchery := grpD3TakeChanged(t, level4, none, grpD3Archery(), grpD3StyleGrant(), noChoices, grpD3FightingStyleHeld())
	if rj := grpD3TakeRejected(t, level4, firstArchery.Feats(), grpD3Archery(), grpD3StyleGrant(), noChoices, grpD3FightingStyleHeld()); rj != (character.FeatNotRepeatable{Feat: "archery"}) {
		t.Errorf("a second Archery = %#v", rj)
	}

	// A grant that names nothing real cannot be built.
	for name, mk := range map[string]func() (character.FeatGrant, error){
		"no source": func() (character.FeatGrant, error) { return character.NewFeatGrant(nil, character.AnyFeat{}) },
		"no scope":  func() (character.FeatGrant, error) { return character.NewFeatGrant(character.BonusFeat{}, nil) },
		"a blank feat": func() (character.FeatGrant, error) {
			return character.NewFeatGrant(character.BonusFeat{}, character.SpecifiedFeat{})
		},
		"an unnamed class": func() (character.FeatGrant, error) {
			return character.NewFeatGrant(character.ClassFeatureFeat{}, character.AnyFeat{})
		},
		"an unknown category": func() (character.FeatGrant, error) {
			return character.NewFeatGrant(character.BonusFeat{}, character.FeatFromCategory{})
		},
	} {
		if _, err := mk(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Prerequisites are checked against the character before the feat's own benefits
// and only when the feat is taken: Grappler needs level 4 and Strength or
// Dexterity 13, its own increase does not count, Epic Boons need level 19 and
// Spell Recall needs Spellcasting, and a Fighting Style needs the feature
// (SRD 5.2.1, "Parts of a Feat", p. 87; "Grappler"; "Boon of Spell Recall", p. 88).
// Requirement: doc/specs/feats-behaviour.md FR-008 to FR-016, Tables P and N - level, ability-score and feature clauses combine with AND and every unmet clause is reported in order; the increase never counts toward its own prerequisite; "Spellcasting" is not met by Pact Magic.
// Path: happy; unhappy (boundary, rule interaction).
func TestGrpD3PrerequisitesAreCheckedBeforeTheFeatsOwnBenefits(t *testing.T) {
	t.Parallel()
	qual := character.FeatQualification{}
	facts := func(level, str, dex int, features ...content.FeatureID) character.FeatQualificationFacts {
		return character.FeatQualificationFacts{Level: character.MustLevel(level), Scores: grpD3Scores(str, dex, 10, 10, 10, 10), Features: grpD3Must(character.NewHeldFeatures(features...))}
	}
	kinds := func(unmet []content.Prerequisite) string {
		var out []string
		for _, p := range unmet {
			switch p.(type) {
			case content.LevelPrerequisite:
				out = append(out, "level")
			case content.AbilityScorePrerequisite:
				out = append(out, "score")
			case content.FeaturePrerequisite:
				out = append(out, "feature")
			}
		}
		return strings.Join(out, ",")
	}
	for _, c := range []struct {
		name string
		feat content.Feat
		in   character.FeatQualificationFacts
		want string
	}{
		{"Alert has no prerequisite", grpD3Alert(), facts(1, 10, 10), ""},
		{"ASI at level 4", grpD3ASI(), facts(4, 10, 10), ""},
		{"ASI at level 3", grpD3ASI(), facts(3, 10, 10), "level"},
		{"Grappler, Strength 13", grpD3Grappler(), facts(4, 13, 8), ""},
		{"Grappler, Dexterity 13", grpD3Grappler(), facts(4, 8, 13), ""},
		{"Grappler, 12 and 12", grpD3Grappler(), facts(4, 12, 12), "score"},
		{"Grappler, level 3 with 13 and 13", grpD3Grappler(), facts(3, 13, 13), "level"},
		{"Grappler, both unmet, in the feat's order", grpD3Grappler(), facts(3, 12, 12), "level,score"},
		{"Boon at level 18", grpD3Boon("boon_of_fate"), facts(18, 10, 10), "level"},
		{"Boon at level 19", grpD3Boon("boon_of_fate"), facts(19, 10, 10), ""},
		{"Boon at level 20", grpD3Boon("boon_of_fate"), facts(20, 10, 10), ""},
		{"Spell Recall with Spellcasting", grpD3Boon("boon_of_spell_recall", grpD3FeatureReq("Spellcasting Feature")), facts(19, 10, 10, "wizard_spellcasting"), ""},
		{"Spell Recall without it", grpD3Boon("boon_of_spell_recall", grpD3FeatureReq("Spellcasting Feature")), facts(19, 10, 10), "feature"},
		{"Spell Recall at 18 with it", grpD3Boon("boon_of_spell_recall", grpD3FeatureReq("Spellcasting Feature")), facts(18, 10, 10, "wizard_spellcasting"), "level"},
		{"Spell Recall with Pact Magic only", grpD3Boon("boon_of_spell_recall", grpD3FeatureReq("Spellcasting Feature")), facts(19, 10, 10, "warlock_pact_magic"), "feature"},
		{"Archery with Fighting Style", grpD3Archery(), facts(1, 10, 10, "fighter_fighting_style"), ""},
		{"Archery with a Paladin's", grpD3Archery(), facts(2, 10, 10, "paladin_fighting_style"), ""},
		{"Archery without it", grpD3Archery(), facts(4, 10, 10), "feature"},
		{"Archery with only an Additional Fighting Style", grpD3Archery(), facts(7, 10, 10, "champion_additional_fighting_style"), "feature"},
	} {
		if got := kinds(qual.Check(c.feat, c.in)); got != c.want {
			t.Errorf("%s: unmet = %q, want %q", c.name, got, c.want)
		}
	}
	// A clause name outside the printed table is never met, even with Spellcasting held.
	odd := grpD3Feat("odd", content.FeatGeneral, false, grpD3FeatureReq("spellcasting feature"))
	if got := qual.Check(odd, facts(5, 10, 10, "wizard_spellcasting")); len(got) != 1 {
		t.Errorf("an unknown clause name was met: %v", got)
	}

	// The feat's own increase never counts toward its prerequisite (Grappler at 12 and 12 choosing Strength).
	c4 := grpD3Hero("prereq-1", 4, grpD3Scores(12, 12, 10, 10, 10, 10))
	rj := grpD3TakeRejected(t, c4, character.TakenFeats{}, grpD3Grappler(), grpD3AnyGrant(), grpD3Single(ability.Strength), character.HeldFeatures{})
	if unmet, ok := rj.(character.FeatPrerequisiteNotMet); !ok || kinds(unmet.Unmet()) != "score" {
		t.Errorf("Grappler choosing Strength at 12 and 12 = %#v", rj)
	}
	// An Epic Boon through an ASI slot needs level 19.
	c18 := grpD3Hero("prereq-2", 18, grpD3Scores(12, 12, 10, 10, 10, 10))
	if rj := grpD3TakeRejected(t, c18, character.TakenFeats{}, grpD3Boon("boon_of_fate"), grpD3AnyGrant(), grpD3Single(ability.Wisdom), character.HeldFeatures{}); grpD3KindsOfRejection(rj) != "level" {
		t.Errorf("a Boon at level 18 = %#v", rj)
	}
	// Prerequisites are checked only when taking: a later fall in a score keeps the feat.
	strong := grpD3Hero("prereq-3", 4, grpD3Scores(15, 10, 10, 10, 10, 10))
	took := grpD3TakeChanged(t, strong, character.TakenFeats{}, grpD3Grappler(), grpD3AnyGrant(), grpD3Single(ability.Strength), character.HeldFeatures{})
	weakened := grpD3Hero("prereq-3", 4, grpD3Scores(8, 10, 10, 10, 10, 10))
	if !took.Feats().Holds("grappler") || len(character.FeatQualification{}.Check(grpD3Grappler(), character.FeatQualificationFacts{Level: weakened.TotalLevel(), Scores: weakened.AbilityScores()})) == 0 {
		t.Error("the held feat must stay while a fresh check would now fail")
	}

	// An unmet check changes nothing and raises no events.
	feats := character.TakenFeats{}
	if rj := grpD3TakeRejected(t, c4, feats, grpD3Grappler(), grpD3AnyGrant(), grpD3Single(ability.Dexterity), character.HeldFeatures{}); rj == nil {
		t.Error("expected a rejection")
	}
	if feats.Len() != 0 || c4.AbilityScores().Score(ability.Dexterity).Value() != 12 {
		t.Error("a rejection altered the character or the feats")
	}

	// A malformed feature id cannot be held.
	if _, err := character.NewHeldFeatures(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank feature id: %v", err)
	}
	if _, err := content.ParseFeatureID("Not A Feature"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a malformed feature id: %v", err)
	}
}

func grpD3KindsOfRejection(r character.FeatRejection) string {
	if unmet, ok := r.(character.FeatPrerequisiteNotMet); ok {
		var out []string
		for _, p := range unmet.Unmet() {
			if _, isLevel := p.(content.LevelPrerequisite); isLevel {
				out = append(out, "level")
			}
		}
		return strings.Join(out, ",")
	}
	return fmt.Sprintf("%T", r)
}

// When several reasons hold at once the reported one follows a fixed order: no
// behaviour, outside the grant, not repeatable, prerequisite, choices, increase.
// Requirement: doc/specs/feats-behaviour.md FR-018, FR-060, FR-015, Table O - one test per adjacent pair with both conditions true.
// Path: unhappy (rule interaction, missing content).
func TestGrpD3RejectionReasonsFollowTheirOrder(t *testing.T) {
	t.Parallel()
	c3 := grpD3Hero("order-1", 3, grpD3Scores(19, 12, 10, 10, 10, 10))
	c4 := grpD3Hero("order-2", 4, grpD3Scores(20, 12, 10, 10, 10, 10))
	asiOne := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	heldASI := grpD3Held(t, grpD3ASI(), asiOne)
	heldAlert := grpD3Held(t, grpD3Alert(), character.NoFeatChoices{})
	none := character.TakenFeats{}

	// No behaviour, and outside the grant too: the unknown feat wins.
	house := grpD3Feat("house_feat", content.FeatGeneral, false)
	specificAlert := grpD3Grant(character.BackgroundFeat{Background: "criminal"}, character.SpecifiedFeat{Feat: "alert", Required: character.NoRequiredSpellList{}})
	if rj := grpD3TakeRejected(t, c4, none, house, specificAlert, character.NoFeatChoices{}, character.HeldFeatures{}); rj != (character.FeatBehaviourUnknown{Feat: "house_feat"}) {
		t.Errorf("unknown and outside the grant = %#v", rj)
	}
	// Outside the grant and already held: the grant wins.
	if rj := grpD3TakeRejected(t, c4, heldAlert, grpD3Alert(), grpD3Grant(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatGeneral}), character.NoFeatChoices{}, character.HeldFeatures{}); rj != (character.FeatOutsideGrant{Feat: "alert"}) {
		t.Errorf("outside the grant and held = %#v", rj)
	}
	// Already held and the level too low: repeatability wins.
	if rj := grpD3TakeRejected(t, c3, heldASI, grpD3ASI(), grpD3AnyGrant(), asiOne, character.HeldFeatures{}); rj != (character.FeatNotRepeatable{Feat: "ability_score_improvement"}) {
		t.Errorf("held and level 3 = %#v", rj)
	}
	// Level too low and no choice: the prerequisite wins.
	if _, ok := grpD3TakeRejected(t, c3, none, grpD3ASI(), grpD3AnyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}).(character.FeatPrerequisiteNotMet); !ok {
		t.Error("level 3 and a missing choice must report the prerequisite")
	}
	// A wrong choice shape and a score at its maximum: the choices win.
	if _, ok := grpD3TakeRejected(t, c4, none, grpD3ASI(), grpD3AnyGrant(), grpD3Single(ability.Strength), character.HeldFeatures{}).(character.FeatChoicesInvalid); !ok {
		t.Error("a wrong shape at the maximum must report the choices")
	}
	// Everything else valid, only the maximum exceeded.
	if !grpD3IsIncreaseRejected(grpD3TakeRejected(t, c4, none, grpD3ASI(), grpD3AnyGrant(), asiOne, character.HeldFeatures{})) {
		t.Error("only the maximum exceeded must report the increase")
	}
}

// Every feat takes exactly the choices of Table C and no others: the shape must
// match, the increased ability must be in the feat's set, and Epic Boons raise a
// score up to 30 where other feats stop at 20 (SRD 5.2.1, pp. 87-88).
// Requirement: doc/specs/feats-behaviour.md FR-017 to FR-020, FR-023, FR-024, FR-061, Tables C and A - missing and unexpected choices, abilities outside the set, a feat with no behaviour, maxima of 20 and 30.
// Path: happy; unhappy (invalid input, boundary, missing content).
func TestGrpD3EachFeatTakesItsOwnChoices(t *testing.T) {
	t.Parallel()
	asiStr := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	recall := grpD3Boon("boon_of_spell_recall", grpD3FeatureReq("Spellcasting Feature"))
	offense := grpD3Boon("boon_of_irresistible_offense")
	fate := grpD3Boon("boon_of_fate")
	truesight := grpD3Boon("boon_of_truesight")

	type problem = character.FeatChoiceProblemKind
	for _, c := range []struct {
		name    string
		level   int
		scores  ability.Scores
		feat    content.Feat
		choices character.FeatChoices
		want    problem // 0: accepted
		check   func(t *testing.T, got ability.Scores)
	}{
		{"Alert takes no choices", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Alert(), character.NoFeatChoices{}, 0, nil},
		{"Alert with an increase", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Alert(), grpD3Single(ability.Strength), character.FeatProblemUnexpected, nil},
		{"Savage Attacker takes no choices", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Savage(), character.NoFeatChoices{}, 0, nil},
		{"Skilled with Magic Initiate choices", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Skilled(), grpD3ClericInitiate(), character.FeatProblemUnexpected, nil},
		{"Magic Initiate with no choices", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Initiate(), character.NoFeatChoices{}, character.FeatProblemMissing, nil},
		{"Grappler with no choice", 4, grpD3Scores(14, 14, 10, 10, 10, 10), grpD3Grappler(), character.NoFeatChoices{}, character.FeatProblemMissing, nil},
		{"Grappler on Dexterity", 4, grpD3Scores(14, 19, 10, 10, 10, 10), grpD3Grappler(), grpD3Single(ability.Dexterity), 0, func(t *testing.T, s ability.Scores) {
			t.Helper()
			if s.Score(ability.Dexterity).Value() != 20 {
				t.Errorf("Dexterity = %d, want 20", s.Score(ability.Dexterity).Value())
			}
		}},
		{"Grappler on Constitution", 4, grpD3Scores(14, 14, 10, 10, 10, 10), grpD3Grappler(), grpD3Single(ability.Constitution), character.FeatProblemAbilityNotAllowed, nil},
		{"ASI by two", 4, grpD3Scores(18, 10, 10, 10, 10, 10), grpD3ASI(), asiStr, 0, func(t *testing.T, s ability.Scores) {
			t.Helper()
			if s.Score(ability.Strength).Value() != 20 {
				t.Errorf("Strength = %d, want 20", s.Score(ability.Strength).Value())
			}
		}},
		{"Archery takes no choices", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Archery(), character.NoFeatChoices{}, 0, nil},
		{"Defense with an increase", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Defense(), grpD3Single(ability.Strength), character.FeatProblemUnexpected, nil},
		{"Boon of Fate on any ability, to 30", 19, grpD3Scores(10, 10, 29, 10, 10, 10), fate, grpD3Single(ability.Constitution), 0, func(t *testing.T, s ability.Scores) {
			t.Helper()
			if s.Score(ability.Constitution).Value() != 30 {
				t.Errorf("Constitution = %d, want 30", s.Score(ability.Constitution).Value())
			}
		}},
		{"Boon of Truesight raises Wisdom past 20", 19, grpD3Scores(10, 10, 10, 10, 20, 10), truesight, grpD3Single(ability.Wisdom), 0, func(t *testing.T, s ability.Scores) {
			t.Helper()
			if s.Score(ability.Wisdom).Value() != 21 {
				t.Errorf("Wisdom = %d, want 21 (maximum 30, not 20)", s.Score(ability.Wisdom).Value())
			}
		}},
		{"Irresistible Offense on Strength", 19, grpD3Scores(14, 10, 10, 10, 10, 10), offense, grpD3Single(ability.Strength), 0, nil},
		{"Irresistible Offense on Wisdom", 19, grpD3Scores(14, 10, 10, 10, 10, 10), offense, grpD3Single(ability.Wisdom), character.FeatProblemAbilityNotAllowed, nil},
		{"Spell Recall on Strength", 19, grpD3Scores(14, 10, 10, 10, 10, 10), recall, grpD3Single(ability.Strength), character.FeatProblemAbilityNotAllowed, nil},
		{"Spell Recall on Charisma", 19, grpD3Scores(14, 10, 10, 10, 10, 10), recall, grpD3Single(ability.Charisma), 0, nil},
		{"a feat with no behaviour", 4, grpD3Scores(10, 10, 10, 10, 10, 10), grpD3Feat("house_feat", content.FeatGeneral, false), character.NoFeatChoices{}, -1, nil},
	} {
		hero := grpD3Hero("choices-1", c.level, c.scores)
		tk := grpD3Take(t, hero, character.TakenFeats{}, c.feat, grpD3AnyGrant(), c.choices, grpD3FullFeatures())
		if c.want == 0 {
			ch, ok := tk.(character.FeatTakingChanged)
			if !ok {
				t.Errorf("%s: %#v, want accepted", c.name, tk)
				continue
			}
			if c.check != nil {
				c.check(t, ch.Character().AbilityScores())
			}
			continue
		}
		if c.want == -1 {
			if rj, ok := tk.(character.FeatTakingRejected); !ok || rj.Rejection != (character.FeatBehaviourUnknown{Feat: "house_feat"}) {
				t.Errorf("%s: %#v, want FeatBehaviourUnknown", c.name, tk)
			}
			continue
		}
		rj, ok := tk.(character.FeatTakingRejected)
		bad, isBad := rj.Rejection.(character.FeatChoicesInvalid)
		if !ok || !isBad || !slices.ContainsFunc(bad.Problems(), func(p character.FeatChoiceProblem) bool { return p.Kind == c.want }) {
			t.Errorf("%s: %#v, want a %v problem", c.name, tk, c.want)
		}
	}

	// Past the maximum: 29 to 30 is the last Epic Boon point, and 30 stops it; a General feat stops at 20.
	atMax := grpD3Hero("choices-2", 19, grpD3Scores(10, 10, 30, 10, 10, 10))
	if !grpD3IsIncreaseRejected(grpD3TakeRejected(t, atMax, character.TakenFeats{}, fate, grpD3AnyGrant(), grpD3Single(ability.Constitution), character.HeldFeatures{})) {
		t.Error("an Epic Boon over 30 must be rejected")
	}
	atTwenty := grpD3Hero("choices-3", 4, grpD3Scores(20, 10, 10, 10, 10, 10))
	if !grpD3IsIncreaseRejected(grpD3TakeRejected(t, atTwenty, character.TakenFeats{}, grpD3Grappler(), grpD3AnyGrant(), grpD3Single(ability.Strength), character.HeldFeatures{})) {
		t.Error("Grappler over 20 must be rejected")
	}

	// Every problem is listed, not the first: a Bard list and a Strength ability together.
	hero := grpD3Hero("choices-4", 4, grpD3Scores(10, 10, 10, 10, 10, 10))
	both := grpD3MIChoices("bard", ability.Strength, []string{"vicious_mockery", "light"}, "shield")
	rj := grpD3TakeRejected(t, hero, character.TakenFeats{}, grpD3Initiate(), grpD3AnyGrant(), both, character.HeldFeatures{}).(character.FeatChoicesInvalid)
	problems := rj.Problems()
	if !slices.Contains(problems, character.FeatChoiceProblem{Name: character.FeatChoiceSpellList, Kind: character.FeatProblemListNotAllowed}) ||
		!slices.Contains(problems, character.FeatChoiceProblem{Name: character.FeatChoiceSpellcastingAbility, Kind: character.FeatProblemAbilityNotAllowed}) {
		t.Errorf("problems = %v, want the list and the ability both", problems)
	}
	problems[0] = character.FeatChoiceProblem{}
	if rj.Problems()[0] == (character.FeatChoiceProblem{}) {
		t.Error("Problems() must return a copy")
	}

	// The constructors reject what is not an ability choice at all.
	if _, err := character.NewSingleAbilityIncrease(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined ability: %v", err)
	}
	for name, mk := range map[string]func() error{
		"two by one on the same ability": func() error {
			_, err := character.NewAbilityScoreImprovementChoices(character.ImprovementTwoByOne, ability.Strength, ability.Strength)
			return err
		},
		"two by one with no second ability": func() error {
			_, err := character.NewAbilityScoreImprovementChoices(character.ImprovementTwoByOne, ability.Strength, 0)
			return err
		},
		"an undefined form": func() error {
			_, err := character.NewAbilityScoreImprovementChoices(0, ability.Strength, 0)
			return err
		},
		"nine Magic Initiate cantrips": func() error {
			_, err := character.NewMagicInitiateChoices("wizard", ability.Intelligence, grpD3IDsOfSpells("a", "b", "c", "d", "e", "f", "g", "h", "i"), "shield")
			return err
		},
	} {
		if err := mk(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// A Magic Initiate takes two cantrips and a level 1 spell from one class list, and
// each later instance needs a different list; the spells are granted as always
// prepared, the free cast comes back on a Long Rest only, and at each level
// gained one spell may be swapped for another of its level (SRD 5.2.1, "Magic
// Initiate", p. 87).
// Requirement: doc/specs/feats-behaviour.md FR-027 to FR-037, Tables M and U - list and ability sets, the cantrip and level 1 counts, levels and list membership, one instance per list, overlapping spells allowed, Spell Change only for a LevelGained of this character at its current total level and once per level.
// Path: happy; unhappy (invalid input, boundary, rule interaction, state error).
func TestGrpD3MagicInitiateChoicesGrantedSpellsAndSpellChange(t *testing.T) {
	t.Parallel()
	hero := grpD3Hero("mi-1", 4, grpD3Scores(10, 10, 10, 10, 10, 10))
	custom := grpD3Grant(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	versatile := grpD3Grant(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin})

	// Table M: each wrong choice gives its own problem kind.
	for _, c := range []struct {
		name    string
		choices character.MagicInitiateChoices
		want    character.FeatChoiceProblemKind
	}{
		{"a Bard list", grpD3MIChoices("bard", ability.Charisma, []string{"vicious_mockery", "light"}, "shield"), character.FeatProblemListNotAllowed},
		{"Strength as the ability", grpD3MIChoices("wizard", ability.Strength, []string{"light", "fire_bolt"}, "shield"), character.FeatProblemAbilityNotAllowed},
		{"one cantrip", grpD3MIChoices("wizard", ability.Intelligence, []string{"light"}, "shield"), character.FeatProblemWrongCount},
		{"three cantrips", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "fire_bolt", "mending"}, "shield"), character.FeatProblemWrongCount},
		{"the same cantrip twice", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "light"}, "shield"), character.FeatProblemDuplicate},
		{"a level 1 spell as a cantrip", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "magic_missile"}, "shield"), character.FeatProblemNotACantrip},
		{"a cantrip as the level 1 spell", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "fire_bolt"}, "mending"), character.FeatProblemNotLevelOne},
		{"a level 2 spell as the level 1 spell", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "fire_bolt"}, "sleep"), character.FeatProblemNotLevelOne},
		{"a cantrip off the list", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "guidance"}, "shield"), character.FeatProblemNotOnSpellList},
		{"a level 1 spell off the list", grpD3MIChoices("wizard", ability.Intelligence, []string{"light", "fire_bolt"}, "bless"), character.FeatProblemNotOnSpellList},
	} {
		bad, ok := grpD3TakeRejected(t, hero, character.TakenFeats{}, grpD3Initiate(), custom, c.choices, character.HeldFeatures{}).(character.FeatChoicesInvalid)
		if !ok || !slices.ContainsFunc(bad.Problems(), func(p character.FeatChoiceProblem) bool { return p.Kind == c.want }) {
			t.Errorf("%s: problems = %v, want %v", c.name, bad.Problems(), c.want)
		}
	}

	// A valid Cleric pick grants its spells; the level 1 spell is always prepared and uncounted.
	cleric := grpD3TakeChanged(t, hero, character.TakenFeats{}, grpD3Initiate(), custom, grpD3ClericInitiate(), character.HeldFeatures{})
	granted := cleric.GrantedSpells()
	if granted.List() != "cleric" || granted.Ability() != ability.Wisdom || !slices.Equal(granted.Cantrips(), grpD3IDsOfSpells("guidance", "light")) || granted.LevelOneSpell() != "bless" ||
		!granted.LevelOneAlwaysPrepared() || granted.LevelOneCountsAgainstPreparedLimit() || granted.IsZero() {
		t.Errorf("granted spells = %+v", granted)
	}
	if all := cleric.Feats().GrantedSpells(); len(all) != 1 || !all[0].Equal(granted) {
		t.Errorf("TakenFeats.GrantedSpells = %v", all)
	}
	if _, ok := cleric.Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable); !ok {
		t.Error("a new free cast must be available")
	}

	// A second instance on another list is allowed (overlapping spells too); the same list is not.
	wizard := grpD3TakeChanged(t, hero, cleric.Feats(), grpD3Initiate(), versatile, grpD3WizardInitiate(), character.HeldFeatures{})
	if wizard.Feats().Len() != 2 || len(wizard.Feats().GrantedSpells()) != 2 {
		t.Errorf("two instances = %v", wizard.Feats().All())
	}
	druid := grpD3MIChoices("druid", ability.Wisdom, []string{"guidance", "mending"}, "cure_wounds")
	if _, ok := grpD3Take(t, hero, cleric.Feats(), grpD3Initiate(), versatile, druid, character.HeldFeatures{}).(character.FeatTakingChanged); !ok {
		t.Error("a Druid instance overlapping the Cleric's Guidance must be accepted")
	}
	again := grpD3MIChoices("cleric", ability.Charisma, []string{"mending", "light"}, "cure_wounds")
	bad, ok := grpD3TakeRejected(t, hero, cleric.Feats(), grpD3Initiate(), versatile, again, character.HeldFeatures{}).(character.FeatChoicesInvalid)
	if !ok || !slices.ContainsFunc(bad.Problems(), func(p character.FeatChoiceProblem) bool { return p.Kind == character.FeatProblemListAlreadyUsed }) {
		t.Errorf("a second Cleric instance = %v", bad.Problems())
	}

	// The free cast is expended and comes back on a Long Rest only.
	exp, err := character.FeatUses{}.Expend(hero, cleric.Feats(), "magic_initiate")
	if err != nil {
		t.Fatal(err)
	}
	spent := exp.(character.FeatsChanged).Feats()
	for _, c := range []struct {
		renewal character.FeatRenewal
		want    bool // available again
	}{{character.RenewOnShortRestFinished, false}, {character.RenewOnInitiativeRolled, false}, {character.RenewOnLongRestFinished, true}} {
		tr, err := character.FeatUses{}.Renew(hero, spent, c.renewal)
		if err != nil {
			t.Fatal(err)
		}
		_, available := tr.(character.FeatsChanged).Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable)
		if available != c.want {
			t.Errorf("after %v the free cast available = %v, want %v", c.renewal, available, c.want)
		}
	}

	// Spell Change at the level-up of the real level-up use case.
	c5, ev5 := grpD3LevelUp(t, "mi-1", 5)
	if ev5.CharacterID() != c5.ID() || ev5.TotalLevel().Value() != 5 {
		t.Fatalf("level-up event = %v at %d", ev5.CharacterID(), ev5.TotalLevel().Value())
	}
	spells := grpD3InitiateSpells()
	replace := func(c character.Character, feats character.TakenFeats, list, old, repl string, ev character.LevelGained) character.FeatsTransition {
		t.Helper()
		tr, err := character.MagicInitiateSpellChange{}.Replace(c, feats, content.ClassID(list), content.SpellID(old), spells[content.SpellID(repl)], ev)
		if err != nil {
			t.Fatal(err)
		}
		return tr
	}
	feats := grpD3TakeChanged(t, c5, character.TakenFeats{}, grpD3Initiate(), custom, grpD3ClericInitiate(), character.HeldFeatures{}).Feats()
	spentFeats := character.TakenFeats{}
	if tr, err := (character.FeatUses{}).Expend(c5, feats, "magic_initiate"); err != nil {
		t.Fatal(err)
	} else {
		spentFeats = tr.(character.FeatsChanged).Feats()
	}

	swapped, ok := replace(c5, spentFeats, "cleric", "guidance", "mending", ev5).(character.FeatsChanged)
	if !ok {
		t.Fatal("a cantrip swap at the level-up was not accepted")
	}
	if ev := swapped.Events(); len(ev) != 1 || ev[0].(character.FeatSpellReplaced).Old() != "guidance" || ev[0].(character.FeatSpellReplaced).New() != "mending" || ev[0].(character.FeatSpellReplaced).List() != "cleric" || ev[0].CharacterID() != c5.ID() {
		t.Errorf("events = %#v", ev)
	}
	gs := swapped.Feats().GrantedSpells()[0]
	if !slices.Equal(gs.Cantrips(), grpD3IDsOfSpells("mending", "light")) && !slices.Equal(gs.Cantrips(), grpD3IDsOfSpells("light", "mending")) || gs.LevelOneSpell() != "bless" || gs.List() != "cleric" || gs.Ability() != ability.Wisdom {
		t.Errorf("granted spells after the swap = %+v", gs)
	}
	if _, still := swapped.Feats().UseStateOf("magic_initiate").(character.FeatUseIsExpended); !still {
		t.Error("a swap must leave the free-cast state alone")
	}
	lvl1, ok := replace(c5, feats, "cleric", "bless", "cure_wounds", ev5).(character.FeatsChanged)
	if !ok || lvl1.Feats().GrantedSpells()[0].LevelOneSpell() != "cure_wounds" {
		t.Errorf("a level 1 swap = %#v", lvl1)
	}

	// Invalid swaps.
	for _, c := range []struct {
		name, old, repl string
		want            character.FeatChoiceProblemKind
	}{
		{"a cantrip for a level 1 spell", "guidance", "cure_wounds", character.FeatProblemLevelMismatch},
		{"a spell for itself", "guidance", "guidance", character.FeatProblemDuplicate},
		{"a spell for the feat's other spell", "guidance", "light", character.FeatProblemDuplicate},
		{"a cantrip for one off the list", "guidance", "fire_bolt", character.FeatProblemNotOnSpellList},
	} {
		rej, isRej := replace(c5, feats, "cleric", c.old, c.repl, ev5).(character.FeatsRejected)
		bad, isBad := rej.Rejection.(character.FeatChoicesInvalid)
		if !isRej || !isBad || !slices.ContainsFunc(bad.Problems(), func(p character.FeatChoiceProblem) bool { return p.Kind == c.want }) {
			t.Errorf("%s: %#v, want %v", c.name, rej, c.want)
		}
	}

	// Not allowed now: no event, another character's event, an event for another level.
	_, other := grpD3LevelUp(t, "someone-else", 5)
	c6, ev6 := grpD3LevelUp(t, "mi-1", 6)
	for name, ev := range map[string]character.LevelGained{"no event": {}, "another character's event": other, "an event for another level": ev6} {
		if rej, ok := replace(c5, feats, "cleric", "guidance", "mending", ev).(character.FeatsRejected); !ok || rej.Rejection != (character.SpellChangeNotAllowedNow{}) {
			t.Errorf("%s: %#v, want SpellChangeNotAllowedNow", name, rej)
		}
	}

	// Once per level; a new level allows another swap.
	once := swapped.Feats()
	if rej, ok := replace(c5, once, "cleric", "light", "mending", ev5).(character.FeatsRejected); !ok || rej.Rejection != (character.SpellChangeAlreadyUsed{}) {
		t.Errorf("a second swap at level 5 = %#v", rej)
	}
	next, ok := replace(c6, once, "cleric", "light", "guidance", ev6).(character.FeatsChanged)
	if !ok {
		t.Fatalf("a swap at the next level was refused: %#v", replace(c6, once, "cleric", "light", "guidance", ev6))
	}
	if gs := next.Feats().GrantedSpells()[0]; !slices.Equal(gs.Cantrips(), grpD3IDsOfSpells("mending", "guidance")) && !slices.Equal(gs.Cantrips(), grpD3IDsOfSpells("guidance", "mending")) {
		t.Errorf("cantrips after two swaps = %v", gs.Cantrips())
	}
	// A swap for a spell of another list is not allowed.
	if rej, ok := replace(c6, once, "cleric", "light", "shillelagh", ev6).(character.FeatsRejected); !ok {
		t.Errorf("a Druid-only cantrip on the Cleric list = %#v", rej)
	}

	// The call itself refuses a missing replacement or list as invalid input.
	if _, err := (character.MagicInitiateSpellChange{}).Replace(c5, feats, "cleric", "guidance", content.Spell{}, ev5); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero replacement spell: %v", err)
	}
	if _, err := (character.MagicInitiateSpellChange{}).Replace(c5, feats, "", "guidance", spells["mending"], ev5); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank list: %v", err)
	}
}

// Skilled takes three distinct skills or tools in any mix and is repeatable; a
// held pick is still accepted; Fighting Style feats come through their own choice
// (SRD 5.2.1, "Skilled", "Fighting Style" feats, p. 87).
// Requirement: doc/specs/feats-behaviour.md FR-038 to FR-043, Table S - exactly three distinct picks, any combination, the grants are listed for the host, a second Skilled is accepted, a FightingStyleChoice of the wrong category is outside the grant.
// Path: happy; unhappy (boundary, invalid input, rule interaction).
func TestGrpD3SkilledAndFightingStyleChoices(t *testing.T) {
	t.Parallel()
	skill := func(s content.Skill) character.SkilledPick { return character.SkilledSkill{Skill: s} }
	tool := func(id string) character.SkilledPick { return character.SkilledTool{Tool: content.ItemID(id)} }
	hero := grpD3Hero("skilled-1", 4, grpD3Scores(10, 10, 10, 10, 10, 10))
	custom := grpD3Grant(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})

	var firstGrants character.ProficiencyGrants
	var firstFeats character.TakenFeats
	for _, c := range []struct {
		name   string
		picks  []character.SkilledPick
		skills int
		tools  int
	}{
		{"three skills", []character.SkilledPick{skill(content.SkillInsight), skill(content.SkillPerception), skill(content.SkillStealth)}, 3, 0},
		{"three tools", []character.SkilledPick{tool("thieves_tools"), tool("herbalism_kit"), tool("smiths_tools")}, 0, 3},
		{"two skills and a tool", []character.SkilledPick{skill(content.SkillInsight), skill(content.SkillPerception), tool("thieves_tools")}, 2, 1},
		{"a skill and two tools", []character.SkilledPick{skill(content.SkillInsight), tool("thieves_tools"), tool("herbalism_kit")}, 1, 2},
	} {
		choices, err := character.NewSkilledChoices(c.picks)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		ch := grpD3TakeChanged(t, hero, character.TakenFeats{}, grpD3Skilled(), custom, choices, character.HeldFeatures{})
		g := ch.ProficiencyGrants()
		if len(g.Skills()) != c.skills || len(g.Tools()) != c.tools || g.IsZero() {
			t.Errorf("%s: grants = %v skills, %v tools", c.name, g.Skills(), g.Tools())
		}
		if c.name == "three skills" {
			firstGrants, firstFeats = g, ch.Feats()
		}
	}
	if sk := firstGrants.Skills(); len(sk) == 3 {
		sk[0] = content.SkillAcrobatics
		if firstGrants.Skills()[0] == content.SkillAcrobatics {
			t.Error("Skills() must return a copy")
		}
	}
	if got := firstFeats.ProficiencyGrants(); len(got.Skills()) != 3 {
		t.Errorf("TakenFeats.ProficiencyGrants = %v", got.Skills())
	}

	// A second Skilled with three other picks is accepted, and the grants add up.
	second := grpD3Must(character.NewSkilledChoices([]character.SkilledPick{tool("thieves_tools"), tool("herbalism_kit"), skill(content.SkillHistory)}))
	both := grpD3TakeChanged(t, hero, firstFeats, grpD3Skilled(), custom, second, character.HeldFeatures{})
	if both.Feats().Len() != 2 || len(both.Feats().ProficiencyGrants().Skills()) != 4 || len(both.Feats().ProficiencyGrants().Tools()) != 2 {
		t.Errorf("two Skilled = %d feats, %v skills, %v tools", both.Feats().Len(), both.Feats().ProficiencyGrants().Skills(), both.Feats().ProficiencyGrants().Tools())
	}

	// Two, four or no picks, and a repeat, are invalid input of the picks.
	for name, picks := range map[string][]character.SkilledPick{
		"two picks":        {skill(content.SkillInsight), skill(content.SkillStealth)},
		"four picks":       {skill(content.SkillInsight), skill(content.SkillStealth), skill(content.SkillHistory), skill(content.SkillArcana)},
		"no picks":         nil,
		"a repeated skill": {skill(content.SkillInsight), skill(content.SkillInsight), skill(content.SkillStealth)},
		"a repeated tool":  {tool("thieves_tools"), tool("thieves_tools"), skill(content.SkillStealth)},
	} {
		_, err := character.NewSkilledChoices(picks)
		var ce *character.FeatChoicesError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ce) || len(ce.Problems()) == 0 {
			t.Errorf("%s: error = %v, want a FeatChoicesError", name, err)
		}
	}
	func() {
		defer func() {
			if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
				t.Errorf("MustSkilledChoices panic = %v", r)
			}
		}()
		character.MustSkilledChoices(skill(content.SkillInsight))
	}()

	// A Fighting Style comes through its own choice, with the grant of the class feature.
	styleSource := character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}
	choice := grpD3Must(character.NewFightingStyleChoice("archery"))
	if choice.Feat() != "archery" {
		t.Errorf("choice feat = %v", choice.Feat())
	}
	id, grant, choices, err := choice.ToTakeFeat(styleSource)
	if err != nil || id != "archery" || grant.Scope() != (character.FeatFromCategory{Category: content.FeatFightingStyle}) || choices != (character.NoFeatChoices{}) {
		t.Fatalf("ToTakeFeat = %v, %v, %v, %v", id, grant, choices, err)
	}
	if _, ok := grpD3Take(t, hero, character.TakenFeats{}, grpD3Archery(), grant, choices, grpD3FightingStyleHeld()).(character.FeatTakingChanged); !ok {
		t.Error("a Fighting Style choice was not accepted")
	}
	// A General feat named by a FightingStyleChoice is outside the grant.
	asiChoice := grpD3Must(character.NewFightingStyleChoice("ability_score_improvement"))
	_, asiGrant, asiChoices, err := asiChoice.ToTakeFeat(styleSource)
	if err != nil {
		t.Fatal(err)
	}
	asiBy := grpD3Hero("skilled-2", 4, grpD3Scores(10, 10, 10, 10, 10, 10))
	if rj := grpD3TakeRejected(t, asiBy, character.TakenFeats{}, grpD3ASI(), asiGrant, asiChoices, grpD3FightingStyleHeld()); rj != (character.FeatOutsideGrant{Feat: "ability_score_improvement"}) {
		t.Errorf("a General feat as a Fighting Style = %#v", rj)
	}
	if _, err := character.NewFightingStyleChoice(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank Fighting Style feat: %v", err)
	}
}

// A character holding a feat gets exactly the effects the SRD prints for it, in
// the order the feats are held, and none when no feat is held (SRD 5.2.1,
// pp. 87-88).
// Requirement: doc/specs/feats-behaviour.md FR-044, FR-049, Table E - each feat declares its effects with its printed field values and usage limit; no feat means no effects; Magic Initiate and Skilled declare none here (their spells and proficiencies are separate).
// Path: happy; unhappy (empty and minimal).
func TestGrpD3HeldFeatsDeclareTheirPrintedEffects(t *testing.T) {
	t.Parallel()
	dim := [2]content.LightLevel{content.LightDim, content.LightDarkness}
	fateUse := grpD3Must(character.NewUseRenewedBy(character.RenewOnInitiativeRolled, character.RenewOnShortRestFinished, character.RenewOnLongRestFinished))
	asiStr := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	none := character.NoFeatChoices{}
	for _, c := range []struct {
		name    string
		feat    content.Feat
		choices character.FeatChoices
		want    []character.FeatEffect
	}{
		{"Alert", grpD3Alert(), none, []character.FeatEffect{character.InitiativeProficiency{}, character.InitiativeSwap{Timing: character.AfterRollingInitiative, Partner: character.WillingAllyInSameCombat, BlockedByCondition: content.ConditionIncapacitated}}},
		{"Savage Attacker", grpD3Savage(), none, []character.FeatEffect{character.RollDamageTwice{Trigger: character.TriggerWeaponHit, Limit: character.OncePerTurn{}}}},
		{"Ability Score Improvement", grpD3ASI(), asiStr, nil},
		{"Grappler", grpD3Grappler(), grpD3Single(ability.Strength), []character.FeatEffect{character.UnarmedStrikeDamageAndGrapple{Limit: character.OncePerTurn{}}, character.AdvantageAgainstGrappledByYou{}, character.GrappleMovementDiscount{TargetSize: character.SameSizeOrSmaller}}},
		{"Archery", grpD3Archery(), none, []character.FeatEffect{character.AttackRollBonus{Bonus: grpD3Mod(2), WeaponType: content.WeaponRanged}}},
		{"Great Weapon Fighting", grpD3Feat("great_weapon_fighting", content.FeatFightingStyle, false, grpD3FeatureReq("Fighting Style Feature")), none, []character.FeatEffect{
			character.DamageDieMinimum{Minimum: 3, AppliesToRolledAtMost: 2, WeaponType: content.WeaponMelee, Hands: character.TwoHands, AnyOfProperties: [2]content.WeaponProperty{content.WeaponPropertyTwoHanded, content.WeaponPropertyVersatile}}}},
		{"Two-Weapon Fighting", grpD3Feat("two_weapon_fighting", content.FeatFightingStyle, false, grpD3FeatureReq("Fighting Style Feature")), none, []character.FeatEffect{character.ExtraAttackAbilityModifierToDamage{RequiredProperty: content.WeaponPropertyLight}}},
		{"Boon of Combat Prowess", grpD3Boon("boon_of_combat_prowess"), grpD3Single(ability.Wisdom), []character.FeatEffect{character.MissBecomesHit{Limit: character.OnceUntilStartOfYourNextTurn{}}}},
		{"Boon of Dimensional Travel", grpD3Boon("boon_of_dimensional_travel"), grpD3Single(ability.Wisdom), []character.FeatEffect{character.BlinkSteps{DistanceFeet: 30, After: [2]character.FeatTriggerAction{character.TriggerAttackAction, character.TriggerMagicAction}, Destination: character.UnoccupiedSpaceYouCanSee}}},
		{"Boon of Fate", grpD3Boon("boon_of_fate"), grpD3Single(ability.Wisdom), []character.FeatEffect{character.ImproveFate{RangeFeet: 60, Dice: grpD3D4s(2), Use: fateUse}}},
		{"Boon of Irresistible Offense", grpD3Boon("boon_of_irresistible_offense"), grpD3Single(ability.Strength), []character.FeatEffect{
			character.IgnoreResistance{DamageTypes: [3]content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing}},
			character.OverwhelmingStrike{Ability: ability.Strength, Trigger: character.TriggerNatural20OnAttackRoll, DamageType: character.SameAsAttackDamageType{}}}},
		{"Boon of Spell Recall", grpD3Boon("boon_of_spell_recall", grpD3FeatureReq("Spellcasting Feature")), grpD3Single(ability.Intelligence), []character.FeatEffect{character.FreeCasting{SlotLevelsFrom: character.MustSlotLevel(1), SlotLevelsTo: character.MustSlotLevel(4), Die: grpD3D4s(1)}}},
		{"Boon of the Night Spirit", grpD3Boon("boon_of_the_night_spirit"), grpD3Single(ability.Wisdom), []character.FeatEffect{
			character.MergeWithShadows{Requires: dim, Grants: content.ConditionInvisible, Cost: character.TriggerBonusAction, EndsAfter: [3]character.FeatTriggerAction{character.TriggerAction, character.TriggerBonusAction, character.TriggerReaction}},
			character.ShadowyForm{Requires: dim, ResistanceToAllExcept: [2]content.DamageType{content.DamagePsychic, content.DamageRadiant}}}},
		{"Boon of Truesight", grpD3Boon("boon_of_truesight"), grpD3Single(ability.Wisdom), []character.FeatEffect{character.TruesightSense{RangeFeet: 60}}},
		{"Magic Initiate declares none here", grpD3Initiate(), grpD3ClericInitiate(), nil},
	} {
		feats := grpD3Held(t, c.feat, c.choices)
		got := character.FeatEffects{}.Effects(feats)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: effects = %#v, want %#v", c.name, got, c.want)
		}
	}
	if got := (character.FeatEffects{}).Effects(character.TakenFeats{}); len(got) != 0 {
		t.Errorf("no feats gave effects %#v", got)
	}
	// Effects follow the order the feats are held.
	two := grpD3Held(t, grpD3Savage(), none, grpD3Alert(), none)
	if got := (character.FeatEffects{}).Effects(two); len(got) != 3 || got[0] != (character.RollDamageTwice{Trigger: character.TriggerWeaponHit, Limit: character.OncePerTurn{}}) {
		t.Errorf("effects of two feats = %#v", got)
	}
}

// Alert adds the Proficiency Bonus to Initiative, Defense gives +1 Armor Class only
// in armor, Great Weapon Fighting treats a 1 or 2 as a 3, and Overwhelming
// Strike adds the current ability score (SRD 5.2.1, pp. 87-88).
// Requirement: doc/specs/feats-behaviour.md FR-045 to FR-048 - Alert's Initiative contribution equals the Proficiency Bonus for the total level and zero without Alert; Defense needs Light, Medium or Heavy armor and a Shield alone gives none; Great Weapon Fighting treats 1 and 2 as 3 and rejects a die below 1; Overwhelming Strike reads the increased ability's current score.
// Path: happy; unhappy (boundary, rule interaction, invalid input).
func TestGrpD3FeatEffectsInitiativeArmorDamageDiceAndOverwhelmingStrike(t *testing.T) {
	t.Parallel()
	none := character.NoFeatChoices{}
	alert := grpD3Held(t, grpD3Alert(), none)
	for level, want := range map[int]int{1: 2, 4: 2, 5: 3, 8: 3, 9: 4, 12: 4, 13: 5, 16: 5, 17: 6, 20: 6} {
		pb := character.ProficiencyBonusFor(character.MustLevel(level))
		if got := (character.FeatEffects{}).InitiativeModifier(alert, pb).Value(); got != want {
			t.Errorf("Alert at level %d adds %d to Initiative, want %d", level, got, want)
		}
	}
	if got := (character.FeatEffects{}).InitiativeModifier(character.TakenFeats{}, character.ProficiencyBonusFor(character.MustLevel(20))).Value(); got != 0 {
		t.Errorf("no Alert adds %d, want 0", got)
	}

	defense := grpD3Held(t, grpD3Defense(), none)
	for name, worn := range map[string][]content.ArmorCategory{
		"Light": {content.ArmorLight}, "Medium": {content.ArmorMedium}, "Heavy": {content.ArmorHeavy},
		"Heavy and a Shield": {content.ArmorHeavy, content.ArmorShield},
	} {
		got := (character.FeatEffects{}).ArmorClassBonuses(defense, worn)
		if len(got) != 1 || got[0].Modifier().Value() != 1 {
			t.Errorf("Defense in %s = %v, want one +1", name, got)
		} else if src, ok := got[0].Source().(character.FeatureFlatBonusSource); !ok || src.Feature != "defense" {
			t.Errorf("Defense bonus source = %#v", got[0].Source())
		}
	}
	for name, worn := range map[string][]content.ArmorCategory{"nothing worn": nil, "a Shield alone": {content.ArmorShield}} {
		if got := (character.FeatEffects{}).ArmorClassBonuses(defense, worn); len(got) != 0 {
			t.Errorf("Defense with %s = %v, want none", name, got)
		}
	}
	if got := (character.FeatEffects{}).ArmorClassBonuses(grpD3Held(t, grpD3Alert(), none), []content.ArmorCategory{content.ArmorHeavy}); len(got) != 0 {
		t.Errorf("armor bonus without Defense = %v", got)
	}

	for rolled, want := range map[int]int{1: 3, 2: 3, 3: 3, 4: 4, 6: 6, 12: 12} {
		if got, err := (character.GreatWeaponFighting{}).Treat(rolled); err != nil || got != want {
			t.Errorf("Great Weapon Fighting on a %d = %d, %v; want %d", rolled, got, err, want)
		}
	}
	for _, bad := range []int{0, -1} {
		if _, err := (character.GreatWeaponFighting{}).Treat(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Treat(%d) error = %v, want ErrInvalidInput", bad, err)
		}
	}

	// Overwhelming Strike adds the increased ability's score after the boon raised it.
	hero := grpD3Hero("strike-1", 20, grpD3Scores(14, 10, 10, 10, 10, 10))
	took := grpD3TakeChanged(t, hero, character.TakenFeats{}, grpD3Boon("boon_of_irresistible_offense"), grpD3AnyGrant(), grpD3Single(ability.Strength), character.HeldFeatures{})
	var strike character.OverwhelmingStrike
	for _, e := range (character.FeatEffects{}).Effects(took.Feats()) {
		if o, ok := e.(character.OverwhelmingStrike); ok {
			strike = o
		}
	}
	if strike.Ability != ability.Strength || strike.ExtraDamage(took.Character().AbilityScores()) != 15 {
		t.Errorf("Overwhelming Strike = %v, %d; want Strength and 15 extra damage", strike.Ability, strike.ExtraDamage(took.Character().AbilityScores()))
	}
	if strike.ExtraDamage(grpD3Scores(8, 10, 10, 10, 10, 10)) != 8 {
		t.Error("the extra damage must follow the current score, not the one at taking time")
	}
}

// Improve Fate and a Magic Initiate free cast are used and renewed on their own
// triggers; expending a feat that is not held or has no limited use is refused, and
// renewing nothing changes nothing (SRD 5.2.1, "Boon of Fate", p. 88).
// Requirement: doc/specs/feats-behaviour.md FR-032, FR-050 to FR-052, Table U - the free use starts available, expending twice is refused, each trigger renews exactly its feats, renewal of nothing reports nothing, FeatNotHeld and FeatHasNoLimitedUse are separate refusals, inputs are unchanged.
// Path: happy; unhappy (state error, empty case, immutability).
func TestGrpD3FeatUsesAreExpendedAndRenewedOnTheirTriggers(t *testing.T) {
	t.Parallel()
	hero := grpD3Hero("uses-1", 20, grpD3Scores(10, 10, 10, 10, 10, 10))
	feats := grpD3Held(t, grpD3Boon("boon_of_fate"), grpD3Single(ability.Wisdom), grpD3Initiate(), grpD3ClericInitiate(), grpD3Alert(), character.NoFeatChoices{})
	expend := func(c character.Character, fs character.TakenFeats, id content.FeatID) character.FeatsTransition {
		t.Helper()
		tr, err := character.FeatUses{}.Expend(c, fs, id)
		if err != nil {
			t.Fatal(err)
		}
		return tr
	}

	one, ok := expend(hero, feats, "boon_of_fate").(character.FeatsChanged)
	if !ok {
		t.Fatal("expending an available Improve Fate failed")
	}
	if _, expended := one.Feats().UseStateOf("boon_of_fate").(character.FeatUseIsExpended); !expended {
		t.Errorf("state after expending = %v", one.Feats().UseStateOf("boon_of_fate"))
	}
	if _, untouched := one.Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable); !untouched {
		t.Error("expending one feat's use touched another's")
	}
	if ev := one.Events(); len(ev) != 1 || ev[0].(character.FeatUseExpended).Feat() != "boon_of_fate" || ev[0].CharacterID() != hero.ID() {
		t.Errorf("events = %#v", ev)
	}
	if _, still := feats.UseStateOf("boon_of_fate").(character.FeatUseIsAvailable); !still {
		t.Error("Expend altered its input")
	}

	if rej, ok := expend(hero, one.Feats(), "boon_of_fate").(character.FeatsRejected); !ok || rej.Rejection != (character.FeatUseAlreadyExpended{Feat: "boon_of_fate"}) {
		t.Errorf("expending twice = %#v", rej)
	}
	if rej, ok := expend(hero, feats, "boon_of_truesight").(character.FeatsRejected); !ok || rej.Rejection != (character.FeatNotHeld{Feat: "boon_of_truesight"}) {
		t.Errorf("a feat not held = %#v", rej)
	}
	if rej, ok := expend(hero, feats, "alert").(character.FeatsRejected); !ok || rej.Rejection != (character.FeatHasNoLimitedUse{Feat: "alert"}) {
		t.Errorf("Alert = %#v", rej)
	}

	// Improve Fate renews on each of its three triggers; the free cast only on a Long Rest.
	both := expend(hero, expend(hero, feats, "boon_of_fate").(character.FeatsChanged).Feats(), "magic_initiate").(character.FeatsChanged).Feats()
	for _, c := range []struct {
		renewal character.FeatRenewal
		fate    bool
		cast    bool
	}{
		{character.RenewOnInitiativeRolled, true, false}, {character.RenewOnShortRestFinished, true, false}, {character.RenewOnLongRestFinished, true, true},
	} {
		tr, err := character.FeatUses{}.Renew(hero, both, c.renewal)
		if err != nil {
			t.Fatal(err)
		}
		ch := tr.(character.FeatsChanged)
		_, fate := ch.Feats().UseStateOf("boon_of_fate").(character.FeatUseIsAvailable)
		_, cast := ch.Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable)
		if fate != c.fate || cast != c.cast {
			t.Errorf("after %v: Improve Fate available %v, free cast available %v; want %v, %v", c.renewal, fate, cast, c.fate, c.cast)
		}
		ev := ch.Events()
		if len(ev) != 1 {
			t.Fatalf("after %v: events = %#v", c.renewal, ev)
		}
		renewed := ev[0].(character.FeatUsesRenewed)
		if renewed.Renewal() != c.renewal || renewed.CharacterID() != hero.ID() || slices.Contains(renewed.Feats(), "magic_initiate") == !c.cast {
			t.Errorf("after %v: renewed %v", c.renewal, renewed.Feats())
		}
	}
	// Nothing expended, nothing listed: the same state and no events.
	idle, err := character.FeatUses{}.Renew(hero, feats, character.RenewOnLongRestFinished)
	if err != nil {
		t.Fatal(err)
	}
	if ch, ok := idle.(character.FeatsChanged); !ok || len(ch.Events()) != 0 || !ch.Feats().Equal(feats) {
		t.Errorf("renewing nothing = %#v", idle)
	}
}

// Boon of Spell Recall may spare a slot of level 1 to 4: one roll of 1d4 spares it
// exactly when the roll equals the slot's level; higher slots never roll (SRD
// 5.2.1, "Boon of Spell Recall", Free Casting, p. 88).
// Requirement: doc/specs/feats-behaviour.md FR-053, FR-054, Table F - spared exactly when the roll equals the level for levels 1 to 4; levels 5 to 9 do not apply; a roll outside 1 to 4 and a slot level outside 1 to 9 are invalid input.
// Path: happy; unhappy (boundary, invalid input).
func TestGrpD3FreeCastingSparesASlotOnlyOnAMatchingRoll(t *testing.T) {
	t.Parallel()
	rule := character.FreeCastingRule{}
	for level := 1; level <= 4; level++ {
		for roll := 1; roll <= 4; roll++ {
			want := character.FreeCastingSpent
			if roll == level {
				want = character.FreeCastingSpared
			}
			got, err := rule.Resolve(character.MustSlotLevel(level), roll)
			if err != nil || got != want {
				t.Errorf("slot %d, roll %d = %v, %v; want %v", level, roll, got, err, want)
			}
		}
	}
	for level := 1; level <= 9; level++ {
		if got, want := rule.Applies(character.MustSlotLevel(level)), level <= 4; got != want {
			t.Errorf("Applies(%d) = %v, want %v", level, got, want)
		}
	}
	for _, level := range []int{5, 9} {
		_, err := rule.Resolve(character.MustSlotLevel(level), 1)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Resolve for a level %d slot error = %v, want ErrInvalidInput (no roll is made)", level, err)
		}
	}
	for _, roll := range []int{0, 5, -1} {
		if _, err := rule.Resolve(character.MustSlotLevel(2), roll); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("a roll of %d error = %v, want ErrInvalidInput", roll, err)
		}
	}
	for _, level := range []int{0, 10} {
		if _, err := character.NewSlotLevel(level); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewSlotLevel(%d) error = %v, want ErrInvalidInput", level, err)
		}
	}
	if rule.Applies(character.SlotLevel{}) {
		t.Error("the zero slot level must not apply")
	}
}

// At level 20 a character earns a bonus feat for every 30,000 XP beyond the level
// 20 threshold; each bonus feat taken uses one up (SRD 5.2.1, "Epic Boon" / "Level
// 20", p. 24).
// Requirement: doc/specs/feats-behaviour.md FR-062, FR-084 - BonusFeats.Remaining is the feats earned minus those taken from the BonusFeat source, never below zero, and none without the option.
// Path: happy; unhappy (boundary, empty and minimal).
func TestGrpD3BonusFeatsAtLevelTwenty(t *testing.T) {
	t.Parallel()
	base := character.ExperiencePoints{}.MinimumForLevel(character.MustLevel(20)).Value()
	xp := func(extra int) character.ExperiencePoints { return character.MustExperiencePoints(base + extra) }
	hero := grpD3Hero("bonus-1", 20, grpD3Scores(10, 10, 10, 10, 10, 10))
	bonus := grpD3Grant(character.BonusFeat{}, character.AnyFeat{})
	taken := grpD3TakeChanged(t, hero, character.TakenFeats{}, grpD3Alert(), bonus, character.NoFeatChoices{}, character.HeldFeatures{}).Feats()
	elsewhere := grpD3TakeChanged(t, hero, character.TakenFeats{}, grpD3Savage(), grpD3AnyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}).Feats()
	if taken.TakenFrom(character.BonusFeat{}) != 1 || elsewhere.TakenFrom(character.BonusFeat{}) != 0 {
		t.Fatalf("TakenFrom = %d / %d, want 1 / 0", taken.TakenFrom(character.BonusFeat{}), elsewhere.TakenFrom(character.BonusFeat{}))
	}
	for _, c := range []struct {
		name   string
		option character.BonusFeatsAtLevel20
		xp     character.ExperiencePoints
		feats  character.TakenFeats
		want   int
	}{
		{"the threshold itself", character.BonusFeatsFromXp{}, xp(0), character.TakenFeats{}, 0},
		{"one point short of a feat", character.BonusFeatsFromXp{}, xp(29999), character.TakenFeats{}, 0},
		{"one feat earned", character.BonusFeatsFromXp{}, xp(30000), character.TakenFeats{}, 1},
		{"three earned", character.BonusFeatsFromXp{}, xp(90000), character.TakenFeats{}, 3},
		{"three earned, one taken", character.BonusFeatsFromXp{}, xp(90000), taken, 2},
		{"one earned, one taken", character.BonusFeatsFromXp{}, xp(30000), taken, 0},
		{"one earned, a feat from another source", character.BonusFeatsFromXp{}, xp(30000), elsewhere, 1},
		{"no bonus feats option", character.NoBonusFeats{}, xp(900000), character.TakenFeats{}, 0},
		{"no option at all", nil, xp(900000), character.TakenFeats{}, 0},
		{"more taken than earned never goes below zero", character.BonusFeatsFromXp{}, xp(0), taken, 0},
	} {
		if got := (character.BonusFeats{}).Remaining(c.option, c.xp, c.feats); got != c.want {
			t.Errorf("%s: remaining = %d, want %d", c.name, got, c.want)
		}
	}
}

// A Fighter swaps a Fighting Style, a Magic Initiate is replaced by another feat,
// and a feat that gave an ability score increase cannot be replaced (SRD 5.2.1,
// "Fighting Style", p. 47; "Wish", p. 175).
// Requirement: doc/specs/feats-behaviour.md FR-082, FR-083, FR-057, FR-061, FR-064 - the old feat is removed first so repeatability and prerequisites are seen without it; FeatReplaced then FeatTaken; FeatNotHeld and FeatNotReplaceable are refusals; TakenFeats validation, copies and equality.
// Path: happy; unhappy (state error, invalid input, immutability, determinism).
func TestGrpD3ReplacingAFeatAndTheTakenFeatsValueObject(t *testing.T) {
	t.Parallel()
	hero := grpD3Hero("replace-1", 8, grpD3Scores(14, 14, 10, 10, 10, 10))
	held := grpD3FightingStyleHeld()
	none := character.NoFeatChoices{}
	replace := func(feats character.TakenFeats, old string, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices) character.FeatTaking {
		t.Helper()
		tk, err := character.FeatReplacement{}.Replace(hero, feats, content.FeatID(old), feat, grant, choices, held, grpD3InitiateSpells())
		if err != nil {
			t.Fatal(err)
		}
		return tk
	}

	feats := grpD3TakeChanged(t, hero, character.TakenFeats{}, grpD3Archery(), grpD3StyleGrant(), none, held).Feats()
	swapped, ok := replace(feats, "archery", grpD3Defense(), grpD3StyleGrant(), none).(character.FeatTakingChanged)
	if !ok {
		t.Fatal("a Fighting Style swap was refused")
	}
	ev := swapped.Events()
	if len(ev) != 2 || ev[0].(character.FeatReplaced).Old() != "archery" || ev[0].(character.FeatReplaced).New() != "defense" || ev[0].CharacterID() != hero.ID() || ev[1].(character.FeatTaken).Feat() != "defense" {
		t.Errorf("events = %#v", ev)
	}
	if swapped.Feats().Holds("archery") || !swapped.Feats().Holds("defense") || swapped.Feats().Len() != 1 || !feats.Holds("archery") {
		t.Errorf("feats after the swap = %v (input holds archery: %v)", swapped.Feats().All(), feats.Holds("archery"))
	}
	if _, ok := replace(feats, "archery", grpD3Archery(), grpD3StyleGrant(), none).(character.FeatTakingChanged); !ok {
		t.Error("a feat replaced by itself must be accepted, as the old one is removed first")
	}

	// Refusals: not held, an increase cannot be undone, the new feat outside its grant or already held.
	asi := grpD3Must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	rich := grpD3Held(t, grpD3Archery(), none, grpD3ASI(), asi, grpD3Initiate(), grpD3ClericInitiate(), grpD3Alert(), none)
	for _, c := range []struct {
		name  string
		old   string
		feat  content.Feat
		grant character.FeatGrant
		want  character.FeatRejection
	}{
		{"the old feat is not held", "defense", grpD3Defense(), grpD3StyleGrant(), character.FeatNotHeld{Feat: "defense"}},
		{"the old feat gave an increase", "ability_score_improvement", grpD3Archery(), grpD3AnyGrant(), character.FeatNotReplaceable{Feat: "ability_score_improvement"}},
		{"the new feat is outside the grant", "archery", grpD3Alert(), grpD3StyleGrant(), character.FeatOutsideGrant{Feat: "alert"}},
		{"the new feat is held and not repeatable", "archery", grpD3Alert(), grpD3AnyGrant(), character.FeatNotRepeatable{Feat: "alert"}},
		{"the new feat has no behaviour", "archery", grpD3Feat("house_feat", content.FeatGeneral, false), grpD3AnyGrant(), character.FeatBehaviourUnknown{Feat: "house_feat"}},
	} {
		if rj := replace(rich, c.old, c.feat, c.grant, none).(character.FeatTakingRejected).Rejection; rj != c.want {
			t.Errorf("%s: %#v, want %#v", c.name, rj, c.want)
		}
	}
	// A Magic Initiate replaced by another feat loses its use state with it (the Wish route).
	used, err := character.FeatUses{}.Expend(hero, rich, "magic_initiate")
	if err != nil {
		t.Fatal(err)
	}
	gone := replace(used.(character.FeatsChanged).Feats(), "magic_initiate", grpD3Savage(), grpD3AnyGrant(), none).(character.FeatTakingChanged)
	if gone.Feats().Holds("magic_initiate") || gone.Feats().UseStateOf("magic_initiate") != nil || !gone.Feats().Holds("savage_attacker") {
		t.Errorf("feats after replacing Magic Initiate = %v", gone.Feats().All())
	}
	if _, err := (character.FeatReplacement{}).Replace(hero, feats, "", grpD3Alert(), grpD3AnyGrant(), none, held, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank old feat id: %v", err)
	}
	if _, err := (character.FeatReplacement{}).Replace(character.Character{}, feats, "archery", grpD3Defense(), grpD3StyleGrant(), none, held, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero character: %v", err)
	}

	// TakenFeats.Replacing is the pure transition: the Repeatable flag of the new feat decides.
	defenseTaken := grpD3Must(character.NewTakenFeat("defense", content.FeatFightingStyle, character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}, none))
	if got, rj := feats.Replacing("archery", defenseTaken, false); rj != nil || !got.Holds("defense") || got.Holds("archery") {
		t.Errorf("Replacing = %v, %v", got.All(), rj)
	}
	if _, rj := feats.Replacing("alert", defenseTaken, false); rj != (character.FeatNotHeld{Feat: "alert"}) {
		t.Errorf("Replacing a feat not held = %#v", rj)
	}

	// Validation of the value objects.
	soldier := character.BackgroundFeat{Background: "soldier"}
	if _, err := character.NewTakenFeat("alert", content.FeatOrigin, soldier, grpD3Single(ability.Strength)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a shape that does not match the feat: %v", err)
	}
	if _, err := character.NewTakenFeat("house_feat", content.FeatOrigin, soldier, none); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a feat with no behaviour: %v", err)
	}
	a := grpD3Must(character.NewTakenFeat("magic_initiate", content.FeatOrigin, soldier, grpD3ClericInitiate()))
	b := grpD3Must(character.NewTakenFeat("magic_initiate", content.FeatOrigin, soldier, grpD3MIChoices("cleric", ability.Charisma, []string{"mending", "light"}, "cure_wounds")))
	if _, err := character.NewTakenFeats(a, b); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("two Magic Initiate on one list: %v", err)
	}
	if _, err := character.NewTakenFeats(character.TakenFeat{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero taken feat: %v", err)
	}
	alertTaken := grpD3Must(character.NewTakenFeat("alert", content.FeatOrigin, soldier, none))
	set := grpD3Must(character.NewTakenFeats(alertTaken, a))
	all := set.All()
	all[0] = character.TakenFeat{}
	if !set.Holds("alert") || set.Len() != 2 || set.TakenFrom(soldier) != 2 {
		t.Error("TakenFeats changed when a returned slice was mutated")
	}
	again := grpD3Must(character.NewTakenFeats(alertTaken, a))
	if !set.Equal(again) || set.Equal(character.TakenFeats{}) || !alertTaken.Equal(alertTaken) || alertTaken.Equal(a) {
		t.Error("equal feats must be Equal and different ones not")
	}
}

func grpD3Mod(n int) modifier.Modifier { return modifier.New(n) }

func grpD3D4s(n int) dice.Expression { return dice.MustExpression(n, dice.D4(), 0) }
