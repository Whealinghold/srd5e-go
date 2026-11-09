package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A level 7 Life Domain Cleric, built over the bundled Cleric definition and seeded by ClericCreationChoices, advances to its levels, uses Divine
// Spark, Turn Undead with Sear Undead and Preserve Life until Channel Divinity is spent, regains uses on a Short and a Long Rest (SRD 5.2.1, "Level 2:
// Channel Divinity", "Level 3: Preserve Life", pp. 37, 40); a level 20 Cleric then uses Divine Intervention with Wish, which recharges only after 2d4 Long
// Rests (SRD 5.2.1, "Level 20: Greater Divine Intervention", p. 38). Cites class-cleric.md FR-039, FR-052, FR-064, FR-095, FR-105.
func TestClericFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle := must2(data.NewBundledSource())
	def := must2(bundle.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("cleric")))).(content.LookupFound).Entry.(content.ClassDefinition)
	progression := must2(character.NewClericProgression(def))
	life := def.Subclasses()[0]

	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(10), ability.Dexterity: ability.MustScore(10), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(16), ability.Charisma: ability.MustScore(10),
	}))
	choices := must2(character.NewClericCreationChoices([]string{"protector"}, nil, nil))
	hero := func(level, xp int) character.Character {
		classes := character.MustClassLevels([]character.ClassLevel{{Class: "cleric", Level: level}})
		return must2(character.Restore(character.Parts{
			ID: character.MustIdentifier("hero"), ExperiencePoints: character.MustExperiencePoints(xp), ClassLevels: classes, Species: "human", Background: "acolyte",
			AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(60),
			HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"cleric": content.HitDieD8}),
			HeroicInspiration: character.NotInspired, FeatureStates: choices.SeedFeatureStates(character.FeatureStates{}),
		}))
	}

	// Level 6 to 7: Channel Divinity rises to 3 and the Life Domain grants its level 7 spells.
	c := hero(7, 23000)
	adv := must2(character.AdvanceCleric(c, 6, progression, &life)).(character.AdvanceClericApplied)
	c = adv.Character()
	if adv.Resources().ChannelDivinityMaximum() != 3 || len(adv.AlwaysPreparedSpells()) != 2 || adv.PendingChoices()[0] != character.ClericPendingBlessedStrikes {
		t.Fatalf("level 7 advance = %+v", adv)
	}
	dc := character.Stats(character.CasterCleric, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(c.TotalLevel())).SaveDC()
	wis := ability.MustScore(16).Modifier()
	run := func(roller *dicetest.FixedRoller, cmd character.UseClericChannelDivinityCommand) character.UseClericChannelDivinityResult {
		cmd.Character, cmd.Subclass, cmd.SpellSaveDC, cmd.Wisdom = c, "life_domain", dc, wis
		return must2(must2(character.NewUseClericChannelDivinity(progression, roller)).Run(ctx, cmd))
	}
	spark := run(dicetest.NewFixedRoller(3, 8), character.UseClericChannelDivinityCommand{Effect: character.DivineSparkEffect{
		Target: character.DivineSparkTarget{Key: "ally", Visible: true, DistanceFeet: 30}, Mode: character.SparkDamage, DamageType: content.DamageRadiant}}).(character.UseClericChannelDivinityApplied)
	if got := spark.Outcome().(character.DivineSparkOutcome).Resolution(); got.DamageOnFailedSave() != 14 || got.DamageOnSuccessfulSave() != 7 { // 3 + 8 + Wisdom +3
		t.Errorf("Divine Spark = %d / %d", got.DamageOnFailedSave(), got.DamageOnSuccessfulSave())
	}
	c = spark.Character()
	turn := run(dicetest.NewFixedRoller(2, 4, 6), character.UseClericChannelDivinityCommand{Effect: character.TurnUndeadEffect{
		Targets: []character.UndeadTarget{{Key: "z1", Type: content.CreatureTypeUndead, DistanceFeet: 30}}, SearUndead: character.SearUndeadUse}}).(character.UseClericChannelDivinityApplied)
	if sear, ok := turn.Outcome().(character.TurnUndeadOutcome).SearUndead(); !ok || sear.Total() != 12 {
		t.Errorf("Sear Undead = %+v", sear)
	}
	c = turn.Character()
	preserve := run(dicetest.NewFixedRoller(), character.UseClericChannelDivinityCommand{Effect: character.PreserveLifeEffect{
		Targets: []character.PreserveLifeTarget{{Key: "a", CurrentHitPoints: 4, HitPointMaximum: 20, DistanceFeet: 10, Allocation: 6}}}}).(character.UseClericChannelDivinityApplied)
	if preserve.Outcome().(character.PreserveLifeOutcome).Pool() != 35 || preserve.Resources().ChannelDivinityRemaining() != 0 {
		t.Errorf("Preserve Life = %+v", preserve.Resources())
	}
	c = preserve.Character()
	if r, ok := run(dicetest.NewFixedRoller(), character.UseClericChannelDivinityCommand{Effect: character.PreserveLifeEffect{}}).(character.UseClericChannelDivinityRejected); !ok || r.Reason != character.ClericNoUsesRemaining {
		t.Errorf("fourth use = %#v", r)
	}

	// Rests: a Short Rest regains one use, a Long Rest the rest.
	res := preserve.Resources()
	short := character.ApplyClericRest(c.ID(), res, character.ShortRest)
	long := character.ApplyClericRest(c.ID(), short.Resources(), character.LongRest)
	if short.Resources().ChannelDivinityRemaining() != 1 || long.Resources().ChannelDivinityRemaining() != 3 {
		t.Errorf("after rests = %d, %d", short.Resources().ChannelDivinityRemaining(), long.Resources().ChannelDivinityRemaining())
	}

	// Level 20: Divine Intervention with Wish recharges after 2d4 Long Rests.
	c = hero(20, 355000)
	wish := wishScenarioSpell(t)
	use := must2(character.NewUseDivineIntervention(dicetest.NewFixedRoller(2, 3)))
	applied := must2(use.Run(ctx, character.UseDivineInterventionCommand{Character: c, Spell: wish, Rules: character.DefaultClericRuleData()})).(character.UseDivineInterventionApplied)
	if roll, ok := applied.RechargeRoll(); !ok || len(roll) != 2 || applied.Resources().DivineIntervention() != (character.DivineInterventionRecharging{LongRestsRemaining: 5}) {
		t.Fatalf("Wish = %+v", applied.Resources())
	}
	state := applied.Resources()
	for i := 0; i < 5; i++ {
		state = character.ApplyClericRest(c.ID(), state, character.LongRest).Resources()
	}
	if state.DivineIntervention() != (character.DivineInterventionAvailable{}) {
		t.Errorf("after five Long Rests = %+v", state.DivineIntervention())
	}
}

func wishScenarioSpell(t *testing.T) content.Spell {
	t.Helper()
	doc := `{"schemaVersion":1,"spells":[{"id":"wish","name":"Wish","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":9,"school":"Conjuration",` +
		`"classes":["wizard"],"castingTimes":[{"kind":"magicAction"}],"ritual":false,"range":{"kind":"self"},"components":{"verbal":true,"somatic":false},` +
		`"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}]}`
	return must2(content.JSONDecoder{}.Decode([]byte(doc))).Entries()[0].(content.Spell)
}
