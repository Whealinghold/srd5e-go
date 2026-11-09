package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-paladin.md FR-124 - every Paladin event is a past-tense value that implements PaladinEvent and Event, carries the Character id it concerns and exposes the data of what happened through its
// accessors, as new slices where it returns one.
// Path: per case (happy).
func TestPaladinEventsCarryTheCharacterIDAndData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := pldPaladin(t, 20)
	id := c.ID()
	lay := must(useLayOnHands(t).Run(ctx, character.UseLayOnHandsCommand{Character: c, HitPoints: 7, Remove: []content.ConditionKind{cPoisoned}})).(character.UseLayOnHandsApplied).Events()[0].(character.LayOnHandsUsed)
	if lay.HitPoints() != 7 || lay.Remaining() != 88 || len(lay.Conditions()) != 1 || lay.Conditions()[0] != cPoisoned {
		t.Errorf("LayOnHandsUsed = %+v", lay)
	}
	channel := must(useChannelDivinity(t).Run(ctx, character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelDivineSense})).(character.UsePaladinChannelDivinityApplied).Events()[0].(character.PaladinChannelDivinityUsed)
	if channel.Option() != "divine_sense" || channel.Remaining() != 2 {
		t.Errorf("PaladinChannelDivinityUsed = %+v", channel)
	}
	cast := must(useFreeCast(t).Run(ctx, character.UseFreeCastCommand{Character: c, Feature: character.FreeCastFaithfulSteed})).(character.UseFreeCastApplied).Events()[0].(character.FreeCastSpent)
	if cast.Feature() != character.FreeCastFaithfulSteed || cast.Spell() != "find_steed" {
		t.Errorf("FreeCastSpent = %+v", cast)
	}
	nimbus := must(useHolyNimbus(t).Run(ctx, character.UseHolyNimbusCommand{Character: c, Subclass: oath, Action: character.HolyNimbusActivate, Charisma: modifierOf(t, 3), Proficiency: character.ProficiencyBonusFor(character.MustLevel(20))}))
	activated := nimbus.(character.UseHolyNimbusApplied).Events()[0].(character.HolyNimbusActivated)
	restored := must(useHolyNimbus(t).Run(ctx, character.UseHolyNimbusCommand{Character: nimbusSpent(t), Subclass: oath, Action: character.HolyNimbusRestoreWithSlot,
		Spellcasting: castingState(t, classLevelsOf(t, "paladin", 20), nil), Pool: character.PoolSpellcasting})).(character.UseHolyNimbusApplied).Events()[1].(character.HolyNimbusRestored)
	smite := must(applySmite(t).Run(ctx, character.ApplySmiteOfProtectionCommand{Character: c, Subclass: oath, DivineSmiteCast: true, Charisma: modifierOf(t, 3)})).(character.ApplySmiteOfProtectionApplied).Events()[0].(character.SmiteOfProtectionTriggered)
	ended := must2events(character.EndPaladinEffect(id, pldState(t, 20, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3, DivineSenseActive: true}), character.PaladinEffectDivineSense, character.PaladinEndElapsed))
	choice := must(chooseOption(t).Run(ctx, character.ChoosePaladinOptionCommand{Character: pldPaladin(t, 2), Choice: character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{pldGuidance(t), pldFlame(t)}}})).(character.ChoosePaladinOptionAccepted).Events()[0].(character.PaladinChoiceMade)
	one := must(character.StartPaladin(pldPaladin(t, 1), pldProgression(t), character.ClassEntryMulticlass)).(character.StartPaladinStarted).Events()[0].(character.PaladinLevelOneGained)
	adv := pldAdvanced(t, pldPaladin(t, 3), 2, oath).Events()[0].(character.PaladinAdvanced)
	rest := character.ApplyPaladinRest(id, pldProgression(t), character.MustPaladinLevel(5), pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 1, PaladinsSmite: character.LongRestUseSpent}), character.LongRest).Events()[0].(character.PaladinResourcesRestored)
	if rest.LayOnHands() != 15 || rest.ChannelDivinityUses() != 1 || len(rest.FreeCasts()) != 1 || rest.HolyNimbus() {
		t.Errorf("PaladinResourcesRestored = %+v", rest)
	}
	if adv.Level() != 3 || len(adv.FeaturesGained()) != 2 || len(adv.AlwaysPreparedSpellsGranted()) != 2 || len(adv.PendingChoices()) != 0 {
		t.Errorf("PaladinAdvanced = %+v", adv)
	}
	adv.FeaturesGained()[0] = "x"
	adv.AlwaysPreparedSpellsGranted()[0] = "x"
	if adv.FeaturesGained()[0] == "x" || adv.AlwaysPreparedSpellsGranted()[0] == "x" {
		t.Error("PaladinAdvanced returned shared storage")
	}
	if one.Entry() != character.ClassEntryMulticlass || choice.Choice() != "blessed_warrior" || ended.Effect() != character.PaladinEffectDivineSense || ended.Reason() != character.PaladinEndElapsed {
		t.Errorf("events = %+v, %+v, %+v", one, choice, ended)
	}
	for _, e := range []character.PaladinEvent{lay, channel, cast, activated, restored, smite, ended, choice, adv, rest} {
		var asEvent character.Event = e
		if asEvent.CharacterID() != id {
			t.Errorf("%T carries the id %v, want %v", e, asEvent.CharacterID(), id)
		}
	}
	if one.CharacterID() != id {
		t.Error("PaladinLevelOneGained carries another id")
	}
}

// must2events returns the PaladinEffectEnded of an EndPaladinEffect call that was accepted.
func must2events(_ character.PaladinState, ev character.PaladinEvent, reason character.PaladinReason) character.PaladinEffectEnded {
	if reason != 0 {
		panic("the effect was not ended")
	}
	return ev.(character.PaladinEffectEnded)
}
