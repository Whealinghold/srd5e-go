package character_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/class-warlock.md FR-164 - every Warlock event is a past-tense value in the closed WarlockEvent family, also an Event, and carries the CharacterID it was given; this test produces all thirteen and checks the family (the accessors are asserted where the operations produce each event).
// Path: happy.
func TestWarlockEventFamily(t *testing.T) {
	t.Parallel()
	p := warFiend(17)
	known := warState(p, character.WarlockStateParts{Invocations: []content.FeatureOptionID{"pact_of_the_blade", "eldritch_smite", "armor_of_shadows"}, Arcanum: [4]content.SpellID{"", "", "", "power_word_kill"},
		HurlThroughHellSpent: true, MagicalCunningUsed: false})
	st := warCasting(t, 17, 1)
	var events []character.Event
	add := func(tr character.WarlockTransition) { events = append(events, warChanged(t, tr).Events()...) }
	sel, _ := character.SelectInvocations(character.MustWarlockLevel(1), 1, []character.InvocationPick{warPick("pact_of_the_blade", "", "")}, nil, warOptions(t), nil)
	add(character.ChooseInvocations(warID, character.WarlockState{}, sel))
	add(character.ChooseMysticArcanum(warID, character.WarlockState{}, p, warSpell(t, "power_word_kill"), 0))
	add(character.CompleteMagicalCunning(warID, character.WarlockState{}, p, time.Minute, st))
	add(character.UseContactPatron(warID, character.WarlockState{}, p))
	add(character.CastMysticArcanum(warID, known, "power_word_kill"))
	add(character.GrantInvocationSpell(warID, known, "armor_of_shadows", content.LightBright))
	add(character.ResolveDarkOnesBlessing(warID, known, p, warMod(16), character.TargetEnemy, character.BlessingByWarlock, 0))
	add(must(character.ChooseFiendishResilience(warID, known, p, character.ShortRest, content.DamageFire)))
	add(must(character.AddDarkOnesOwnLuck(warID, known, p, warMod(16), d20.SavingThrow, character.LuckNotUsedOnRoll, 4)))
	add(character.UseHurlThroughHell(warID, character.WarlockState{}, p, character.WarlockOtherHit, character.WarlockTurnFacts{}))
	add(character.RestoreHurlThroughHell(warID, known, p, st))
	add(character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, st, content.SizeLarge, character.SmiteApplyProne))
	add(must(character.ApplyWarlockRest(warID, known, character.LongRest)))
	seen := map[string]bool{}
	for _, e := range events {
		we, ok := e.(character.WarlockEvent)
		if !ok || we.CharacterID() != warID {
			t.Errorf("%T = (%v, %v), want a WarlockEvent of %v", e, we, ok, warID)
		}
		seen[fmt.Sprintf("%T", e)] = true
	}
	if len(seen) != 13 {
		t.Errorf("distinct events = %d (%v), want 13", len(seen), seen)
	}
}

// Requirement: doc/specs/class-warlock.md FR-144 - the lists a Warlock event returns are copies: changing the slice read from InvocationsChosen.Picks or WarlockFeaturesRestored.Features changes nothing in the event.
// Path: unhappy (immutability).
func TestWarlockEventListsAreCopies(t *testing.T) {
	t.Parallel()
	sel, _ := character.SelectInvocations(character.MustWarlockLevel(1), 1, []character.InvocationPick{warPick("pact_of_the_blade", "", "")}, nil, warOptions(t), nil)
	chosen := warChanged(t, character.ChooseInvocations(warID, character.WarlockState{}, sel)).Events()[0].(character.InvocationsChosen)
	chosen.Picks()[0].Option = "changed"
	if chosen.Picks()[0].Option != "pact_of_the_blade" {
		t.Error("Picks leaked the event's slice")
	}
	spent := warState(warProfile(5, ""), character.WarlockStateParts{MagicalCunningUsed: true})
	restored := warChanged(t, must(character.ApplyWarlockRest(warID, spent, character.LongRest))).Events()[0].(character.WarlockFeaturesRestored)
	restored.Features()[0] = character.RestoresHurlThroughHell
	if restored.Features()[0] != character.RestoresMagicalCunning {
		t.Error("Features leaked the event's slice")
	}
	if got := warChanged(t, must(character.ApplyWarlockRest(warID, spent, character.LongRest))).Events(); len(got) != 1 {
		t.Errorf("Events must return a copy: %v", got)
	}
}
