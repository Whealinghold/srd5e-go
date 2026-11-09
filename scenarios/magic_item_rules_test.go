package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// miFocus builds a completed Short Rest focused on one item.
func miFocus(t *testing.T, instance string, purpose character.RestPurpose) character.ShortRestFocus {
	t.Helper()
	report := must2(character.NewRestReport(character.ShortRest, character.StandardLongRest, character.NewRestRanItsCourse(1, 0)))
	res := must2(character.ResolveRest(report))
	return must2(character.NewShortRestFocus(character.MagicItemInstanceID(instance), purpose, character.InContact, character.ShortRest, res))
}

// Requirement: doc/specs/magic-item-rules.md section 8 (scenario test) - a Thief (limit 4) attunes four items, a fifth is rejected, one
// ends by a completed Short Rest and a new one attunes; the events carry the character id.
// Path: happy.
func TestScenarioAttunementLimitOfAThief(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), character.MustIdentifier("thief-1")
	limit := must2(character.NewAttunementLimit(4))
	candidate := must2(character.NewAttuneCandidate([]content.ClassID{content.MustClassID("rogue")}, character.CannotCast))
	attune := character.NewAttuneToMagicItem()
	end := character.NewEndMagicItemAttunement()
	set := character.AttunementSet{}
	for _, n := range []string{"a", "b", "c", "d"} {
		item := must2(character.NewAttunableItem(content.MustMagicItemID("item_"+n), character.MagicItemInstanceID(n), content.AttunementRequired{}))
		res := must2(attune.Do(ctx, character.AttuneToMagicItemCommand{CharacterID: who, Set: set, Item: item, Candidate: candidate, Focus: miFocus(t, n, character.RestPurposeAttune), Limit: limit}))
		applied, ok := res.(character.AttuneToMagicItemApplied)
		if !ok || applied.Events()[0].CharacterID() != who {
			t.Fatalf("attune %s: %#v", n, res)
		}
		set = applied.Set()
	}
	fifth := must2(character.NewAttunableItem(content.MustMagicItemID("item_e"), "e", content.AttunementRequired{}))
	attuneFifth := character.AttuneToMagicItemCommand{CharacterID: who, Set: set, Item: fifth, Candidate: candidate, Focus: miFocus(t, "e", character.RestPurposeAttune), Limit: limit}
	if r, ok := must2(attune.Do(ctx, attuneFifth)).(character.AttuneToMagicItemRejected); !ok || r.Reason() != character.AttuneLimitReached {
		t.Fatalf("fifth item: %#v", r)
	}
	ended := must2(end.Do(ctx, character.EndMagicItemAttunementCommand{CharacterID: who, Set: set, Instance: "a", Curse: content.NotCursed,
		Means: character.EndByShortRest{Focus: miFocus(t, "a", character.RestPurposeEndAttunement)}}))
	set = ended.(character.EndMagicItemAttunementApplied).Set()
	attuneFifth.Set = set
	res, ok := must2(attune.Do(ctx, attuneFifth)).(character.AttuneToMagicItemApplied)
	if !ok || res.Set().Len() != 4 || !res.Set().IsAttunedTo("e") || res.Set().IsAttunedTo("a") {
		t.Fatalf("after ending one: %#v", res)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-027, FR-064, FR-065 and table "Item spellcasting ability" - an item's spell uses the user's
// ability (or +0) with the Proficiency Bonus, composed with the spell save DC and attack bonus of spellcasting; two cloaks are worn only with a GM exception.
// Path: happy.
func TestScenarioItemSpellcastingAndWornCloaks(t *testing.T) {
	t.Parallel()
	pb := character.ProficiencyBonusFor(character.MustLevel(5)) // +3
	user := []character.UserSpellAbility{{Ability: ability.Wisdom, Modifier: must2(ability.NewScore(14)).Modifier()}, {Ability: ability.Charisma, Modifier: must2(ability.NewScore(18)).Modifier()}}
	r := character.ItemSpellcastingAbility{}.Resolve(user, ability.Charisma, pb).(character.ItemSpellcastingResolved)
	stats := character.StatsForAbility(must2(character.NewFeatureSpellAbility(r.Ability)), r.Modifier, r.ProficiencyBonus)
	if stats.SaveDC().Value() != 15 || stats.AttackBonus().Value() != 7 {
		t.Errorf("DC %d, attack %d, want 15 and +7", stats.SaveDC().Value(), stats.AttackBonus().Value())
	}
	cloak := func(id string) character.WornItem {
		return must2(character.NewWornItem(character.MagicItemInstanceID(id), character.WornSlotCloak, ""))
	}
	worn := character.WornItems{}
	worn = must2(worn.Wear(cloak("c1"), content.ItemFitAdjusts, 0, 0)).(character.WornItemsWorn).Items
	if got := must2(worn.Wear(cloak("c2"), content.ItemFitAdjusts, 0, 0)); got != (character.WornItemsSlotOccupied{Slot: character.WornSlotCloak}) {
		t.Errorf("second cloak: %#v", got)
	}
	if _, ok := must2(worn.Wear(cloak("c2"), content.ItemFitAdjusts, 0, character.SlotExceptionGMAllows)).(character.WornItemsWorn); !ok {
		t.Error("second cloak with a GM exception")
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-099, FR-101, FR-102, FR-103 - a sentient sword's conflict end to end with a fixed roller: the
// bearer fails the conflict save and meets the demands, the sword takes control for 1d12 hours and is then unable to try again until dawn, and a
// passed repeat save ends the control.
// Path: happy.
func TestScenarioSentientSwordConflict(t *testing.T) {
	t.Parallel()
	ctx, bearer := context.Background(), character.MustIdentifier("bearer-1")
	charisma := must2(ability.NewScore(14)) // +2: DC 14
	save := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	demands := must2(character.NewSentientDemands([]content.SentientDemand{content.SentientDemandKeepMeClose}))

	conflict := must2(character.NewResolveSentientItemConflict(dicetest.NewFixedRoller(13)))
	if _, ok := must2(conflict.Do(ctx, character.ResolveSentientItemConflictCommand{ItemCharisma: charisma, Attempt: save, Demands: demands})).(character.ResolveSentientItemConflictDemandsMade); !ok {
		t.Fatal("a total of 13 against DC 14 meets the demands")
	}
	control := must2(character.NewAttemptSentientItemControl(dicetest.NewFixedRoller(5, 9)))
	taken, ok := must2(control.Do(ctx, character.AttemptSentientItemControlCommand{CharacterID: bearer, ItemCharisma: charisma, Attempt: save})).(character.AttemptSentientItemControlTaken)
	if !ok || taken.Effect().Hours() != 9 || taken.Effect().Condition() != content.ConditionCharmed || taken.Power() != character.ControlPowerSpentUntilNextDawn {
		t.Fatalf("control = %#v", taken)
	}
	again := must2(control.Do(ctx, character.AttemptSentientItemControlCommand{CharacterID: bearer, Power: taken.Power(), ItemCharisma: charisma, Attempt: save}))
	if again != (character.AttemptSentientItemControlRejected{}) {
		t.Errorf("the spent power was accepted: %#v", again)
	}
	repeat := must2(character.NewRepeatSentientControlSave(dicetest.NewFixedRoller(16)))
	ended, ok := must2(repeat.Do(ctx, character.RepeatSentientControlSaveCommand{CharacterID: bearer, Instance: "sword", ItemCharisma: charisma, Attempt: save})).(character.RepeatSentientControlSaveEnded)
	if !ok || ended.Event().CharacterID() != bearer || taken.Power().AfterDawn() != character.ControlPowerAvailable {
		t.Errorf("repeat save = %#v", ended)
	}
}
