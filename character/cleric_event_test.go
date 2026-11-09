package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/class-cleric.md FR-125, FR-136 - every Cleric event is a ClericEvent, an Event carrying the id of the character; the family is switchable
// exhaustively, each accessor returns what the operation produced and the slice accessors return copies.
// Path: happy / unhappy (immutability).
func TestClericEventsAreAFamily(t *testing.T) {
	t.Parallel()
	c := clcCleric(t, 20)
	var events []character.Event
	chosen := must(character.ChooseBlessedStrikes(c, []string{"divine_strike"})).(character.ChooseBlessedStrikesChosen)
	events = append(events, chosen.Events()...)
	adv := must(character.AdvanceCleric(clcWith(t, clcCleric(t, 5), clcUses(2, 2)), 4, clcProgression(t), clcLifeDomain(t))).(character.AdvanceClericApplied)
	events = append(events, adv.Events()...)
	spark := must(character.NewUseClericChannelDivinity(clcProgression(t), dicetest.NewFixedRoller(5, 5, 5, 5)))
	applied := must(spark.Run(t.Context(), character.UseClericChannelDivinityCommand{Character: c,
		Effect: character.DivineSparkEffect{Target: character.DivineSparkTarget{Key: "t", Visible: true}, Mode: character.SparkHealing}})).(character.UseClericChannelDivinityApplied)
	events = append(events, applied.Events()...)
	rest := character.ApplyClericRest(c.ID(), must(character.NewClericResources(3, 0, character.DivineInterventionUnavailableUntilLongRest{})), character.LongRest)
	events = append(events, rest.Events()...)
	di := must(character.NewUseDivineIntervention(dicetest.NewFixedRoller()))
	used := must(di.Run(t.Context(), character.UseDivineInterventionCommand{Character: c, Spell: clcClericSpell(t, "cure_wounds", 1)})).(character.UseDivineInterventionApplied)
	events = append(events, used.Events()...)

	seen := map[string]bool{}
	for _, e := range events {
		ce, ok := e.(character.ClericEvent)
		if !ok || e.CharacterID() != c.ID() {
			t.Errorf("%T is not a ClericEvent of the character", e)
			continue
		}
		switch v := ce.(type) {
		case character.ClericOptionChosen:
			seen["option"] = v.Feature() == "blessed_strikes" && v.Option() == "divine_strike"
		case character.ClericAdvanced:
			seen["advanced"] = v.Level() == 5 && len(v.FeaturesGained()) == 1 && len(v.AlwaysPreparedSpellsGranted()) == 2 && len(v.PendingChoices()) == 0 && v.ChannelDivinityUsesGained() == 0
			v.FeaturesGained()[0] = "changed"
			seen["advancedCopy"] = v.FeaturesGained()[0] == "cleric_sear_undead"
		case character.ClericChannelDivinityUsed:
			seen["used"] = v.Effect() == "divine_spark" && v.Remaining() == 3
		case character.ClericChannelDivinityRegained:
			seen["regained"] = v.Uses() == 3
		case character.DivineInterventionRecharged:
			seen["recharged"] = true
		case character.DivineInterventionUsed:
			seen["intervention"] = v.Spell() == "cure_wounds" && v.LongRestsToGo() == 0
		}
	}
	for _, k := range []string{"option", "advanced", "advancedCopy", "used", "regained", "recharged", "intervention"} {
		if !seen[k] {
			t.Errorf("event %q missing or wrong", k)
		}
	}
}
