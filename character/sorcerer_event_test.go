package character_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-sorcerer.md FR-131 - every Sorcerer event is a past-tense value in the closed SorcererEvent family, also an Event, and carries the CharacterID; the accessors are asserted where the operations produce each
// event (the Innate Sorcery, Font of Magic, Metamagic, rest and Draconic Sorcery tests), and this test produces all fifteen and checks the family.
// Path: happy.
func TestSorcererEventFamily(t *testing.T) {
	t.Parallel()
	p := sorDraconic(20)
	rich := sorWith(p, 10, []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}, func(s *character.SorcererStateParts) {
		s.CreatedSlots = [5]int{0, 0, 1, 0, 0}
		s.ElementalAffinity = content.DamageFire
		s.DragonWingsUsed = true
	})
	var events []character.Event
	events = append(events, sorChanged(t, character.ActivateInnateSorcery(sorID, sorFresh(p), p, character.InnateSorceryNormalUse)).Events()...)
	events = append(events, sorChanged(t, character.EndInnateSorcery(sorID, sorWith(p, 0, nil, func(s *character.SorcererStateParts) { s.InnateSorceryActive = true }))).Events()...)
	events = append(events, must(character.ConvertSpellSlot(sorID, rich, p, character.ConvertCreatedSlot{}, 3)).(character.SorcererSlotsChanged).Events()...)
	events = append(events, sorChanged(t, must(character.CreateSlot(sorID, rich, p, 1, character.SlotCreationRule{Cost: 2, MinSorcererLevel: 2}, true))).Events()...)
	events = append(events, sorChanged(t, must(character.SpendCreatedSlot(sorID, rich, sorCasting(false), 3, 3))).Events()...)
	events = append(events, sorChanged(t, must(character.ApplySorcererRest(sorID, rich, p, character.LongRest, 0))).Events()...)
	events = append(events, sorChanged(t, must(character.ApplySorcererRest(sorID, sorWith(p, 5, nil, nil), p, character.ShortRest, 10))).Events()...)
	events = append(events, character.ApplyMetamagic(sorID, rich, p, sorMod(16), character.MetamagicRequest{Effect: character.MetamagicTwinned, Cost: 1, Facts: sorFacts(nil)}).(character.MetamagicApplied).Events()...)
	events = append(events, sorChanged(t, character.ChooseMetamagic(sorID, rich, p, sorSelection(t, 10, character.MetamagicCareful, character.MetamagicTwinned, character.MetamagicDistant, character.MetamagicSubtle))).Events()...)
	events = append(events, sorChanged(t, character.ChooseElementalAffinity(sorID, sorFresh(p), p, content.DamageAcid)).Events()...)
	events = append(events, sorChanged(t, character.ApplyElementalAffinity(sorID, rich, p, sorMod(16), []content.DamageType{content.DamageFire}, false)).Events()...)
	for _, a := range []character.DragonWingsAction{character.DragonWingsActivate, character.DragonWingsDismiss, character.DragonWingsRestoreUse} {
		s := sorWith(p, 20, nil, func(s *character.SorcererStateParts) {
			s.DragonWingsUsed, s.DragonWingsActive = a != character.DragonWingsActivate, a == character.DragonWingsDismiss
		})
		events = append(events, sorChanged(t, must(character.UseDragonWings(sorID, s, p, a))).Events()...)
	}
	events = append(events, sorChanged(t, character.UseDragonCompanion(sorID, sorFresh(p), p, true)).Events()...)
	seen := map[string]bool{}
	for _, e := range events {
		se, ok := e.(character.SorcererEvent)
		if !ok || se.CharacterID() != sorID {
			t.Errorf("%T = (%v, %v), want a SorcererEvent of %v", e, se, ok, sorID)
		}
		seen[sorTypeName(e)] = true
	}
	if len(seen) != 15 {
		t.Errorf("distinct events = %d (%v), want 15", len(seen), seen)
	}
}

func sorTypeName(v any) string { return fmt.Sprintf("%T", v) }
