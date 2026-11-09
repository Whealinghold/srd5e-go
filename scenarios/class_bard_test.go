package scenarios_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// The bundled Bard definition agrees with the rules code at all 20 levels: the Proficiency Bonus of the Features table equals ProficiencyBonusFor, the Bardic Die column equals
// BardicDieAt and maps to the kernel Die with the same sides, and the Prepared Spells and Cantrips columns equal PreparedLimit and CantripLimit (SRD 5.2.1, "Bard Features",
// p. 31). Cites class-bard.md FR-023, FR-125 and content-classes-data.md FR-033.
func TestBardTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle := must2(data.NewBundledSource())
	def := must2(bundle.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("bard")))).(content.LookupFound).Entry.(content.ClassDefinition)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must2(def.RowAt(n))
			level := character.MustBardLevel(n)
			dieColumn, _ := row.Value("bardic_die")
			die, isDice := dieColumn.(content.ColumnDice)
			sc, isSlots := row.Spellcasting().(content.SlotSpellcastingRow)
			cantrips, _ := sc.Cantrips()
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() {
				t.Errorf("Proficiency Bonus = %d", row.ProficiencyBonus())
			}
			if !isDice || die.Count() != 1 || die.Sides() != character.BardicDieAt(level).Sides() || character.BardicDieAt(level).Die().Sides() != die.Sides() {
				t.Errorf("Bardic Die column = %v, rules = %v", dieColumn, character.BardicDieAt(level))
			}
			if !isSlots || sc.PreparedSpells() != character.PreparedLimit(character.CasterBard, n) || cantrips != character.CantripLimit(character.CasterBard, n) {
				t.Errorf("prepared %d (rules %d), cantrips %d (rules %d)", sc.PreparedSpells(), character.PreparedLimit(character.CasterBard, n), cantrips, character.CantripLimit(character.CasterBard, n))
			}
		})
	}
}

// A Bard of level 5 with Charisma 16 (three uses) seeded by BardCreationChoices grants Bardic Inspiration, which another creature uses on a failed check; Cutting Words (College of
// Lore) spends a use; a Short Rest restores them from level 5 and a Long Rest at any level; Countercharm rerolls a failed saving throw with Advantage and spends nothing; at level 14
// Peerless Skill adds the die and spends a use only on a success; Superior Inspiration at level 18 restores two uses on rolling Initiative (SRD 5.2.1, "Bardic Inspiration", pp. 31-32;
// "Level 7: Countercharm", p. 33; "Level 3: Cutting Words", "Level 14: Peerless Skill", p. 35). Cites class-bard.md FR-026, FR-030, FR-038, FR-039, FR-043, FR-072, FR-081, FR-084, FR-087.
func TestBardFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(10), ability.Dexterity: ability.MustScore(14), ability.Constitution: ability.MustScore(12),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(16),
	}))
	cha := scores.Modifier(ability.Charisma)
	choices := character.NewBardCreationChoices(cha)
	hero := func(level, xp int) character.Character {
		classes := character.MustClassLevels([]character.ClassLevel{{Class: "bard", Level: level}})
		return must2(character.Restore(character.Parts{
			ID: character.MustIdentifier("bard"), ExperiencePoints: character.MustExperiencePoints(xp), ClassLevels: classes, Species: "human", Background: "entertainer",
			AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(40),
			HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"bard": content.HitDieD8}),
			HeroicInspiration: character.NotInspired, FeatureStates: choices.SeedFeatureStates(character.FeatureStates{}),
		}))
	}
	pool := func(c character.Character) int {
		p, _, _ := character.ReadFeatureState(c.FeatureStates(), character.BardicInspirationPoolKey())
		return p.Remaining()
	}
	failedCheck := func(natural, dc int) character.AdjustableCheck {
		return character.AdjustableCheck{Result: must2(d20.ResolveCheck(d20.AbilityCheck, d20.MustRoll(modifier.Normal, natural), d20.NewRollModifiers(modifier.New(0)), d20.DifficultyClass(dc)))}
	}

	c := hero(5, 6500)
	if pool(c) != 3 {
		t.Fatalf("seeded pool = %d, want 3", pool(c))
	}
	// Grant, then the recipient uses the d8 within the hour and fails to reach the DC, which still expends the die.
	granting, reason := character.GrantBardicInspiration(c.ID(), character.MustBardLevel(5), must2(pooled(c)), cha, character.InspirationRecipientOther{})
	if reason != 0 || granting.Grant().Die() != character.BardicD8 || granting.Pool().Remaining() != 2 {
		t.Fatalf("grant = %+v, %v", granting, reason)
	}
	inspiration := must2(character.NewUseBardicInspiration(dicetest.NewFixedRoller(2)))
	used := must2(inspiration.Run(ctx, character.UseBardicInspirationCommand{ID: character.MustIdentifier("friend"), Grant: granting.Grant(), Elapsed: 50 * time.Minute, Failed: failedCheck(8, 15)}))
	if applied, ok := used.(character.UseBardicInspirationApplied); !ok || applied.Applied().Result().Total() != 10 || applied.Applied().Result().Outcome() != d20.Failure {
		t.Errorf("Bardic Inspiration = %#v", used)
	}

	// Cutting Words (College of Lore) spends a use; no use is left after the pool is spent.
	cutting := must2(character.NewUseCuttingWords(dicetest.NewFixedRoller(5)))
	cmd := character.UseCuttingWordsCommand{Character: c, Subclass: character.CollegeOfLoreSubclassID, Trigger: character.CuttingWordsDamage{Total: 9}}
	spent := must2(cutting.Run(ctx, cmd)).(character.UseCuttingWordsApplied)
	if spent.Applied().Damage() != 4 || pool(spent.Character()) != 2 || pool(c) != 3 {
		t.Errorf("Cutting Words damage %d, pool %d (input %d)", spent.Applied().Damage(), pool(spent.Character()), pool(c))
	}

	// Rests: a Short Rest restores every use at level 5 (and none at level 4); a Long Rest restores them at any level.
	low := must2(character.NewBardicInspirationPool(3, 0))
	for _, k := range []struct {
		level int
		rest  character.RestKind
		want  character.BardReason
	}{{5, character.ShortRest, 0}, {4, character.ShortRest, character.BardBelowRequiredLevel}, {4, character.LongRest, 0}} {
		got, reason := character.RecoverBardicInspiration(c.ID(), character.MustBardLevel(k.level), low, cha, character.BardRecoveryRest{Kind: k.rest})
		if reason != k.want || (reason == 0 && got.Pool().Remaining() != 3) {
			t.Errorf("level %d rest %v: reason %v, pool %d", k.level, k.rest, reason, got.Pool().Remaining())
		}
	}

	// Countercharm (level 7): a failed saving throw against a Charmed effect is rerolled with Advantage; the pool is untouched.
	seven := hero(7, 23000)
	save := must2(d20.ResolveCheck(d20.SavingThrow, d20.MustRoll(modifier.Normal, 4), d20.NewRollModifiers(modifier.New(1)), d20.DifficultyClass(14)))
	counter := must2(character.NewUseCountercharm(dicetest.NewFixedRoller(4, 15)))
	reroll := must2(counter.Run(ctx, character.UseCountercharmCommand{Character: seven, Failed: save, Conditions: []content.ConditionKind{content.ConditionCharmed}})).(character.UseCountercharmRerolled)
	if reroll.Rerolled().Outcome() != d20.Success || reroll.Rerolled().Roll().Natural() != 15 || pool(seven) != 3 {
		t.Errorf("Countercharm = %v, natural %d", reroll.Rerolled().Outcome(), reroll.Rerolled().Roll().Natural())
	}

	// Peerless Skill (level 14, d10): a roll that reaches the DC spends a use; one that does not spends none.
	fourteen := hero(14, 195000)
	peerless := func(roll int) character.UsePeerlessSkillApplied {
		uc := must2(character.NewUsePeerlessSkill(dicetest.NewFixedRoller(roll)))
		return must2(uc.Run(ctx, character.UsePeerlessSkillCommand{Character: fourteen, Subclass: character.CollegeOfLoreSubclassID, Failed: failedCheck(10, 15)})).(character.UsePeerlessSkillApplied)
	}
	if hit, miss := peerless(5), peerless(4); pool(hit.Character()) != 2 || pool(miss.Character()) != 3 || !hit.Applied().Spent() || miss.Applied().Spent() {
		t.Errorf("Peerless Skill pools = %d and %d", pool(hit.Character()), pool(miss.Character()))
	}

	// Superior Inspiration (level 18): rolling Initiative restores uses until two remain.
	if got, reason := character.RecoverBardicInspiration(c.ID(), character.MustBardLevel(18), must2(character.NewBardicInspirationPool(3, 0)), cha, character.BardRecoveryInitiative{}); reason != 0 || got.Pool().Remaining() != 2 {
		t.Errorf("Superior Inspiration = %d, %v", got.Pool().Remaining(), reason)
	}

	// Advancing: level 20 grants Power Word Heal and Power Word Kill and the second target.
	twenty := must2(character.AdvanceBard(hero(20, 355000), 19, character.CollegeOfLoreSubclassID)).(character.AdvanceBardApplied)
	if len(twenty.AlwaysPreparedSpellsGranted()) != 2 || !character.AllowsSecondTarget(twenty.Level(), "power_word_kill") {
		t.Errorf("level 20 = %v", twenty.AlwaysPreparedSpellsGranted())
	}
}

func pooled(c character.Character) (character.BardicInspirationPool, error) {
	p, _, err := character.ReadFeatureState(c.FeatureStates(), character.BardicInspirationPoolKey())
	return p, err
}
