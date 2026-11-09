package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/spellcasting.md FR-127 - every spellcasting event is a
// character.Event and a SpellcastingEvent carrying the character id of its state, so
// events of two characters are distinguishable; the family is switched exhaustively.
// Path: happy.
func TestSpellcastingEventsCarryTheCharacterID(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	other := character.MustIdentifier("other")
	mk := func(id character.Identifier) []character.SpellcastingEvent {
		res, err := character.NewCreateSpellcasting().Do(t.Context(), character.CreateSpellcastingCommand{Character: id, Levels: classLevelsOf(t, "wizard", 5), Spells: spells})
		if err != nil {
			t.Fatal(err)
		}
		st := res.(character.CreateSpellcastingCreated).State()
		exp := st.ExpendSlot(character.PoolSpellcasting, slotLevel(1)).(character.SlotExpenditureExpended)
		_, rest := exp.State().RestoreSlots(character.LongRest)
		out, err := character.ChoosePreparation(st, character.CasterWizard, nil, spellIDs(0, 1), spells)
		if err != nil {
			t.Fatal(err)
		}
		return append([]character.SpellcastingEvent{exp.Event()}, append(rest, out.(character.PreparationAccepted).Events()...)...)
	}
	seen := map[string]bool{}
	for _, c := range []struct {
		id     character.Identifier
		events []character.SpellcastingEvent
	}{{caster, mk(caster)}, {other, mk(other)}} {
		if len(c.events) != 3 {
			t.Fatalf("events = %v, want 3", c.events)
		}
		for _, e := range c.events {
			var ev character.Event = e
			if ev.CharacterID() != c.id {
				t.Errorf("event %#v id = %v, want %v", e, ev.CharacterID(), c.id)
			}
			switch e.(type) {
			case character.SpellSlotExpended:
				seen["expended"] = true
			case character.SpellSlotsRestored:
				seen["restored"] = true
			case character.PreparedSpellsChanged:
				seen["changed"] = true
			}
		}
	}
	if len(seen) != 3 {
		t.Errorf("variants seen = %v", seen)
	}
}
