package scenarios_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func rangerDefinition(t *testing.T) content.ClassDefinition {
	t.Helper()
	bundle := must2(data.NewBundledSource())
	return must2(bundle.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("ranger")))).(content.LookupFound).Entry.(content.ClassDefinition)
}

// The bundled Ranger definition agrees with the rules code at all 20 levels: the Proficiency Bonus of the Features table equals ProficiencyBonusFor, there is no Cantrips column,
// the Prepared Spells equal PreparedLimit and the nine Spell Slot counts equal what SlotsFor derives for the Ranger alone, and the Favored Enemy column equals what RangerProgression reads
// (SRD 5.2.1, "Ranger Features", p. 58). Cites class-ranger.md FR-023, FR-030, FR-032, content-classes-data.md FR-033, FR-035.
func TestRangerTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	def := rangerDefinition(t)
	hunter := def.Subclasses()[0]
	progression := must2(character.NewRangerProgression(def, &hunter))
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must2(def.RowAt(n))
			sc := row.Spellcasting().(content.SlotSpellcastingRow)
			_, hasCantrips := sc.Cantrips()
			slots := character.SlotsFor(character.MustClassLevels([]character.ClassLevel{{Class: "ranger", Level: n}})).Spellcasting()
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() || hasCantrips || sc.PreparedSpells() != character.PreparedLimit(character.CasterRanger, n) {
				t.Errorf("bonus %d, cantrips %v, prepared %d (rules %d)", row.ProficiencyBonus(), hasCantrips, sc.PreparedSpells(), character.PreparedLimit(character.CasterRanger, n))
			}
			for l := 1; l <= 9; l++ {
				if got, want := sc.SpellSlots()[l-1], slots.Maximum(character.MustSlotLevel(l)); got != want {
					t.Errorf("level %d slots = %d, rules = %d", l, got, want)
				}
			}
			v, _ := row.Value("favored_enemy")
			if col, ok := v.(content.ColumnCount); !ok || col.Value() != progression.FavoredEnemyUses(character.MustRangerLevel(n)) {
				t.Errorf("Favored Enemy = %v, progression %d", v, progression.FavoredEnemyUses(character.MustRangerLevel(n)))
			}
		})
	}
}

// A level 11 Hunter chooses Colossus Slayer, spends its Favored Enemy uses, takes a Short and a Long Rest and gets the Hunter rider; a level 10 Ranger uses Tireless with a fixed roller until
// the uses are spent, through the public packages only (SRD 5.2.1, "Level 1: Favored Enemy", "Level 10: Tireless", pp. 58-59; "Ranger Subclass: Hunter", p. 61).
// Cites class-ranger.md FR-033 to FR-036, FR-072, FR-073, FR-075, FR-086, FR-088, FR-110.
func TestRangerFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	def := rangerDefinition(t)
	hunterDef := def.Subclasses()[0]
	progression := must2(character.NewRangerProgression(def, &hunterDef))
	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(10), ability.Dexterity: ability.MustScore(16), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(16), ability.Charisma: ability.MustScore(8),
	}))
	hero := func(level, xp int) character.Character {
		classes := character.MustClassLevels([]character.ClassLevel{{Class: "ranger", Level: level}})
		return must2(character.Restore(character.Parts{
			ID: character.MustIdentifier("hero"), ExperiencePoints: character.MustExperiencePoints(xp), ClassLevels: classes, Species: "human", Background: "soldier",
			AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(90),
			HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"ranger": content.HitDieD10}),
			HeroicInspiration: character.NotInspired, FeatureStates: character.NewRangerCreationChoices(character.WeaponMasteryChoice{}).SeedFeatureStates(character.FeatureStates{}),
		}))
	}
	wis := scores.Modifier(ability.Wisdom)
	// Level 11 Hunter: Hunter’s Prey is owed, the choice is made, and Colossus Slayer then rides on a hit.
	cur := hero(11, 85000)
	summary := func() character.ResolveRangerFeaturesResolved {
		return must2(must2(character.NewResolveRangerFeatures(progression)).Run(ctx, character.ResolveRangerFeaturesCommand{Character: cur, Subclass: character.HunterSubclassID,
			Speed: character.MustSpeedInput(30, true), Wisdom: wis, HuntersMarkDie: dice.D6()})).(character.ResolveRangerFeaturesResolved)
	}
	if !slices.Contains(summary().PendingChoices(), character.RangerPendingHuntersPrey) {
		t.Errorf("Hunter’s Prey is not owed at level 11: %v", summary().PendingChoices())
	}
	chosen := must2(must2(character.NewChooseRangerOption(progression)).Run(ctx, character.ChooseRangerOptionCommand{Character: cur, Subclass: character.HunterSubclassID,
		Choice: character.RangerChooseHuntersPrey{Option: "colossus_slayer"}})).(character.ChooseRangerOptionAccepted)
	cur = chosen.Character()
	if slices.Contains(summary().PendingChoices(), character.RangerPendingHuntersPrey) {
		t.Error("Hunter’s Prey is still owed after the choice")
	}
	facts := character.RangerFacts{Progression: progression, Level: character.MustRangerLevel(11), State: chosen.State(), Subclass: character.HunterSubclassID}
	if rider, ok := facts.ColossusSlayer(character.ColossusSlayerFacts{WeaponHit: true, TargetHitPoints: 5, TargetMaximum: 20}); !ok || rider.Count != 1 || rider.Die != dice.D8() {
		t.Errorf("Colossus Slayer = %+v, %v", rider, ok)
	}
	if got := summary(); got.FavoredEnemyUses() != 4 || got.AttacksPerAttackAction() != 2 || len(got.HunterFeatures()) != 4 {
		t.Errorf("level 11 summary: %d uses, %d attacks, %v", got.FavoredEnemyUses(), got.AttacksPerAttackAction(), got.HunterFeatures())
	}
	// Spend every Favored Enemy use; the fifth cast needs a slot.
	state := chosen.State()
	for i := 0; i < 4; i++ {
		res, ok := character.UseFavoredEnemy(cur.ID(), character.RangerFacts{Progression: progression, Level: character.MustRangerLevel(11), State: state, Subclass: character.HunterSubclassID}).(character.FavoredEnemyUseAccepted)
		if !ok {
			t.Fatalf("use %d rejected", i+1)
		}
		state = res.State()
	}
	if _, ok := character.UseFavoredEnemy(cur.ID(), character.RangerFacts{Progression: progression, Level: character.MustRangerLevel(11), State: state}).(character.FavoredEnemyUseRejected); !ok {
		t.Error("a fifth free cast was accepted")
	}
	short := must2(character.ApplyRangerRest(cur.ID(), character.RangerFacts{Progression: progression, Level: character.MustRangerLevel(11), State: state}, character.ShortRest, 2))
	long := must2(character.ApplyRangerRest(cur.ID(), character.RangerFacts{Progression: progression, Level: character.MustRangerLevel(11), State: short.State()}, character.LongRest, 2))
	if short.State().FavoredEnemySpent() != 4 || len(short.Events()) != 1 || long.State().FavoredEnemySpent() != 0 {
		t.Errorf("after rests: short %d spent %d events, long %d spent", short.State().FavoredEnemySpent(), len(short.Events()), long.State().FavoredEnemySpent())
	}
	// Level 10: Tireless, with a fixed roller, until the three uses (Wisdom +3) are spent.
	tireless := must2(character.NewUseTireless(progression, dicetest.NewFixedRoller(5, 5, 5, 5)))
	c10 := hero(10, 64000)
	for i := 0; i < 3; i++ {
		res, ok := must2(tireless.Run(ctx, character.UseTirelessCommand{Character: c10, Wisdom: wis})).(character.UseTirelessAccepted)
		if !ok || res.TemporaryHitPoints() != 8 {
			t.Fatalf("use %d = %+v", i+1, res)
		}
		c10 = res.Character()
	}
	if r, ok := must2(tireless.Run(ctx, character.UseTirelessCommand{Character: c10, Wisdom: wis})).(character.UseTirelessRejected); !ok || r.Reason != character.RangerNoUsesRemaining {
		t.Errorf("fourth use = %#v", r)
	}
}
