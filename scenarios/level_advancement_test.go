package scenarios_test

import (
	"context"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// classesFromWizard decodes the bundled Wizard document with an Evoker subclass added
// and a Fighter derived from it (Hit Point Die D10, Primary Ability Strength or Dexterity).
func classesFromWizard(t *testing.T) (wizard, fighter content.ClassDefinition) {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_wizard.json")
	if err != nil {
		t.Fatal(err)
	}
	decode := func(text string) content.ClassDefinition {
		set, err := content.JSONDecoder{}.Decode([]byte(text))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range set.Entries() {
			if def, ok := e.(content.ClassDefinition); ok {
				return def
			}
		}
		t.Fatal("no class decoded")
		return content.ClassDefinition{}
	}
	subclass := `{"id":"evoker","name":"Evoker","features":[{"id":"evoker_savant","name":"Evocation Savant","level":3,"text":"Text."}]}`
	withSub := strings.Replace(string(raw), `"subclasses": []`, `"subclasses": [`+subclass+`]`, 1)
	wizard = decode(withSub)
	text := regexp.MustCompile(`"wizard(_[a-z_]*)?"`).ReplaceAllString(string(raw), `"fighter$1"`)
	text = strings.Replace(text, `"hitPointDie": "D6"`, `"hitPointDie": "D10"`, 1)
	text = regexp.MustCompile(`(?s)"primaryAbility": \{.*?\}`).ReplaceAllString(text, `"primaryAbility": {"kind":"anyOf","abilities":["strength","dexterity"]}`)
	return wizard, decode(text)
}

// A level 1 Wizard gains levels 2 to 4 with a preview before each: the Evoker is owed at
// level 3 and chosen, level 4 reports an Ability Score Improvement grant that raises
// Constitution (and the Hit Point maximum), and a first level in Fighter is previewed
// with its Spell Slots and Proficiency Bonus (SRD 5.2.1, "Gaining a Level", p. 23;
// "Multiclassing", pp. 24-26; "Ability Score Improvement", p. 87). The feat itself
// (TakeFeat) belongs to the feats rules and is not part of this scenario.
// Cites level-advancement-and-multiclassing.md FR-009, FR-010, FR-014, FR-016, FR-017,
// FR-018, FR-031, FR-034, FR-041, FR-054, FR-108.
func TestLevelAdvancementWizardToFighter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wizardDef, fighterDef := classesFromWizard(t)
	src := contenttest.NewFakeSource(wizardDef, fighterDef)
	created, err := character.New(character.NewParams{
		ID: character.MustIdentifier("pc-1"), Class: wizardDef.ID(), HitPointDie: content.HitDieD6,
		Species: content.MustSpeciesID("human"), Background: content.MustBackgroundID("sage"),
		AbilityScores: scoresFor(t, 13, 12, 15, 15, 12, 10),
	})
	if err != nil {
		t.Fatal(err)
	}
	pc := created.Character()
	pc = pc.AwardExperience(character.MustExperienceAmount(2700)).Character() // the minimum for level 4

	preview, err := character.NewPreviewLevelAdvancement(character.WithSource(src))
	if err != nil {
		t.Fatal(err)
	}
	levelUp, err := character.NewLevelUpCharacter(character.WithSource(src), character.WithRoller(dicetest.NewFixedRoller()))
	if err != nil {
		t.Fatal(err)
	}
	choose, err := character.NewChooseSubclass(character.WithSource(src))
	if err != nil {
		t.Fatal(err)
	}
	for want := 2; want <= 4; want++ {
		res, err := preview.Run(ctx, character.PreviewLevelAdvancementCommand{Character: pc, Class: wizardDef.ID()})
		avail, ok := res.(character.PreviewLevelAdvancementAvailable)
		if err != nil || !ok || avail.Gain.NewClassLevel() != want {
			t.Fatalf("preview to level %d: %#v %v", want, res, err)
		}
		pending := avail.Gain.PendingChoices()
		if want == 3 && len(pending) != 1 {
			t.Fatalf("level 3 pending = %v, want the subclass", pending)
		}
		step, err := levelUp.Run(ctx, character.LevelUpCharacterCommand{Character: pc, Class: wizardDef.ID(), HitPointChoice: character.TakeFixedValue{}})
		if err != nil {
			t.Fatal(err)
		}
		pc = step.(character.LevelUpCharacterSucceeded).Character()
		if want == 3 {
			chosen, err := choose.Run(ctx, character.ChooseSubclassCommand{Character: pc, Class: wizardDef.ID(), Subclass: "evoker"})
			ok, isOK := chosen.(character.ChooseSubclassSucceeded)
			if err != nil || !isOK {
				t.Fatalf("choose Evoker: %#v %v", chosen, err)
			}
			pc = ok.Character()
		}
		if want == 4 && (len(avail.Gain.PendingChoices()) != 1 || avail.Gain.PendingChoices()[0] != character.PendingFeatGrant{
			Class: wizardDef.ID(), ClassLevel: 4, Feature: "wizard_ability_score_improvement", Category: content.FeatGeneral}) {
			t.Fatalf("level 4 pending = %v, want the Ability Score Improvement", avail.Gain.PendingChoices())
		}
	}

	// The Ability Score Improvement raises Constitution 15 to 17: the modifier rises by one.
	before := pc.HitPointMaximum().Value()
	inc, err := character.NewAbilityScoreImprovementOneByTwo(ability.Constitution)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := character.AbilityScoreIncreaser{}.Apply(pc, inc)
	if err != nil {
		t.Fatal(err)
	}
	raised := tr.(character.TransitionChanged[character.AbilityScoreIncreaseRejection])
	if pc = raised.Character(); pc.AbilityScores().Score(ability.Constitution).Value() != 17 || pc.HitPointMaximum().Value() != before+4 {
		t.Errorf("Constitution %d, Hit Points %d (before %d)", pc.AbilityScores().Score(ability.Constitution).Value(), pc.HitPointMaximum().Value(), before)
	}

	// Fighter as a new class: Intelligence 15 and Strength 13 meet both prerequisites.
	pc = pc.AwardExperience(character.MustExperienceAmount(3800)).Character() // 6,500 for level 5
	res, err := preview.Run(ctx, character.PreviewLevelAdvancementCommand{Character: pc, Class: fighterDef.ID()})
	gain, ok := res.(character.PreviewLevelAdvancementAvailable)
	if err != nil || !ok {
		t.Fatalf("Fighter preview: %#v %v", res, err)
	}
	g := gain.Gain
	if _, isNew := g.ClassStatus().(character.AdvancementClassStatusNew); !isNew || g.NewClassLevel() != 1 || g.NewTotalLevel().Value() != 5 ||
		g.HitPointDie() != content.HitDieD10 || g.ProficiencyBonusAfter().Value() != 3 || !g.ProficiencyBonusIncreases() ||
		g.SpellSlotsAfter().Spellcasting().Maximum(character.MustSlotLevel(1)) != 4 {
		t.Errorf("Fighter gain = %+v", g)
	}

	// Extra Attack from two classes does not stack (SRD 5.2.1, "Multiclassing", p. 25).
	if n, err := character.ExtraAttackCombination([]int{2, 2}); err != nil || n != 2 {
		t.Errorf("Extra Attack = %d, %v", n, err)
	}
}
