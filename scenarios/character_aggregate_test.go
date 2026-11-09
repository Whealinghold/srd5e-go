package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func scoresFor(t *testing.T, str, dex, con, intl, wis, cha int) ability.Scores {
	t.Helper()
	s, err := ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(str), ability.Dexterity: ability.MustScore(dex), ability.Constitution: ability.MustScore(con),
		ability.Intelligence: ability.MustScore(intl), ability.Wisdom: ability.MustScore(wis), ability.Charisma: ability.MustScore(cha),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A Wizard is created, gains 300 XP and a level (fixed Hit Points), takes a first
// level in a second class once a primary ability is raised, and raises
// Constitution; a level with too little XP changes nothing; Heroic Inspiration is
// granted and expended; a class the source lacks is reported (SRD 5.2.1, "Level
// Advancement", p. 23; "Multiclassing", pp. 24-25; "Heroic Inspiration", p. 8).
// Cites character-aggregate.md FR-011, FR-020, FR-025, FR-026, FR-027, FR-045,
// FR-061, FR-090, FR-091.
func TestCharacterLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wizard := content.MustClassID("wizard")
	created, err := character.New(character.NewParams{
		ID: character.MustIdentifier("pc-1"), Class: wizard, HitPointDie: content.HitDieD6,
		Species: content.MustSpeciesID("human"), Background: content.MustBackgroundID("sage"),
		AbilityScores: scoresFor(t, 8, 14, 17, 15, 12, 10),
	})
	if err != nil {
		t.Fatal(err)
	}
	pc := created.Character()
	if pc.HitPointMaximum().Value() != 9 || len(created.Events()) != 1 { // d6 top face 6 + Constitution +3
		t.Fatalf("created: maximum %d, %d events", pc.HitPointMaximum().Value(), len(created.Events()))
	}

	levelUp, err := character.NewLevelUpCharacter(character.WithSource(data.MustBundledSource()), character.WithRoller(dicetest.NewFixedRoller()))
	if err != nil {
		t.Fatal(err)
	}
	tooSoon, err := levelUp.Run(ctx, character.LevelUpCharacterCommand{Character: pc, Class: wizard, HitPointChoice: character.TakeFixedValue{}})
	if rej, ok := tooSoon.(character.LevelUpCharacterRejected); err != nil || !ok || rej.Rejection != (character.LevelUpInsufficientExperience{}) {
		t.Fatalf("with 0 XP: %#v %v", tooSoon, err)
	}

	award, err := character.NewAwardExperience().Run(ctx, character.AwardExperienceCommand{Character: pc, Amount: character.MustExperienceAmount(300)})
	if err != nil {
		t.Fatal(err)
	}
	pc = award.(character.AwardExperienceSucceeded).Character()
	res, err := levelUp.Run(ctx, character.LevelUpCharacterCommand{Character: pc, Class: wizard, HitPointChoice: character.TakeFixedValue{}})
	if err != nil {
		t.Fatal(err)
	}
	pc = res.(character.LevelUpCharacterSucceeded).Character()
	if pc.TotalLevel().Value() != 2 || pc.HitPointMaximum().Value() != 9+4+3 { // fixed d6 value 4 + Constitution +3
		t.Fatalf("level 2: level %d maximum %d", pc.TotalLevel().Value(), pc.HitPointMaximum().Value())
	}

	// A new class needs 13 in the Primary Ability of both classes: raise Strength first.
	pc = pc.AwardExperience(character.MustExperienceAmount(600)).Character()
	change := character.NewChangeAbilityScore()
	raised, err := change.Run(ctx, character.ChangeAbilityScoreCommand{Character: pc, Ability: ability.Strength, Score: ability.MustScore(13)})
	if err != nil {
		t.Fatal(err)
	}
	pc = raised.(character.ChangeAbilityScoreSucceeded).Character()
	fighter, err := character.NewAdvancementClass(content.MustClassID("fighter"), content.HitDieD10, mustAllOf(t, ability.Strength))
	if err != nil {
		t.Fatal(err)
	}
	intl, err := character.NewAdvancementClass(wizard, content.HitDieD6, mustAllOf(t, ability.Intelligence))
	if err != nil {
		t.Fatal(err)
	}
	gain, _ := character.NewFixedHitPointGain(6)
	tr, err := pc.LevelUp(fighter, gain, map[content.ClassID]content.AbilityRequirement{wizard: intl.PrimaryAbility()})
	if err != nil {
		t.Fatal(err)
	}
	pc = tr.(character.TransitionChanged[character.LevelUpRejection]).Character()
	if pc.ClassLevels().LevelIn(content.MustClassID("fighter")) != 1 || pc.TotalLevel().Value() != 3 || pc.HitDicePool().Total() != 3 {
		t.Fatalf("multiclass: %v", pc.ClassLevels().Entries())
	}

	con, err := change.Run(ctx, character.ChangeAbilityScoreCommand{Character: pc, Ability: ability.Constitution, Score: ability.MustScore(18)})
	if err != nil {
		t.Fatal(err)
	}
	if got := con.(character.ChangeAbilityScoreSucceeded); got.Character().HitPointMaximum().Value() != pc.HitPointMaximum().Value()+3 || len(got.Events()) != 2 {
		t.Errorf("Constitution 17 to 18 at level 3 must add 3: %d, %d events", got.Character().HitPointMaximum().Value(), len(got.Events()))
	}

	// Heroic Inspiration: grant, expend, expend again.
	pc = pc.GrantHeroicInspiration().Character()
	spent, ok := pc.ExpendHeroicInspiration().(character.TransitionChanged[character.HeroicInspirationRejection])
	if !ok || spent.Character().HeroicInspiration() != character.NotInspired {
		t.Fatal("expending Heroic Inspiration must succeed once")
	}
	if _, again := spent.Character().ExpendHeroicInspiration().(character.TransitionRejected[character.HeroicInspirationRejection]); !again {
		t.Error("expending a second time must be refused")
	}

	// A class the source lacks.
	empty, err := character.NewLevelUpCharacter(character.WithSource(contenttest.NewFakeSource()), character.WithRoller(dicetest.NewFixedRoller()))
	if err != nil {
		t.Fatal(err)
	}
	missing, err := empty.Run(ctx, character.LevelUpCharacterCommand{Character: pc, Class: wizard, HitPointChoice: character.TakeFixedValue{}})
	if m, ok := missing.(character.LevelUpCharacterContentMissing); err != nil || !ok || len(m.Missing()) != 1 {
		t.Errorf("missing class: %#v %v", missing, err)
	}
}

func mustAllOf(t *testing.T, a ability.Ability) content.AbilityRequirement {
	t.Helper()
	r, err := content.NewAllOf(a)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
