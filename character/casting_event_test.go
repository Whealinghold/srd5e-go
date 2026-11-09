package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/casting-and-concentration.md FR-119, FR-141, Section 5 - the end and break reasons have fixed
// snake_case names, CastingCompleted is an end reason and not a break reason, and every casting event is a
// character.Event and a CastingEvent (an exhaustive switch over the family).
// Path: happy.
func TestConcentrationReasonsAndEventFamily(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.ConcentrationEndReason]string{
		character.EndedIncapacitated: "incapacitated", character.EndedDead: "dead", character.EndedByCaster: "ended",
		character.EndedFailedSave: "failed_save", character.EndedAnotherEffect: "another_effect", character.EndedCastingCompleted: "casting_completed", 0: "unknown", 7: "unknown",
	} {
		if r.String() != want {
			t.Errorf("end reason %d = %q, want %q", r, r.String(), want)
		}
	}
	breaks := []character.ConcentrationBreakReason{character.BreakIncapacitated, character.BreakDead, character.BreakEnded, character.BreakFailedSave, character.BreakAnotherEffect}
	for _, b := range breaks {
		if !b.Valid() || b.String() != character.ConcentrationEndReason(b).String() {
			t.Errorf("break reason %d = %q", b, b.String())
		}
	}
	if character.ConcentrationBreakReason(character.EndedCastingCompleted).Valid() || character.ConcentrationBreakReason(0).Valid() || character.ConcentrationBreakReason(0).String() != "unknown" {
		t.Error("CastingCompleted or the zero value is a break reason [FR-119]")
	}
	var events = []character.CastingEvent{character.SpellCast{}, character.CastingCountered{}, character.ConcentrationBegan{}, character.ConcentrationEnded{}, character.CastingFailed{}}
	for _, e := range events {
		var _ character.Event = e
		switch e.(type) {
		case character.SpellCast, character.CastingCountered, character.ConcentrationBegan, character.ConcentrationEnded, character.CastingFailed:
		default:
			t.Errorf("%T is outside the family", e)
		}
	}
	if _, ok := any(character.SpellSlotExpended{}).(character.CastingEvent); ok {
		t.Error("SpellSlotExpended is a CastingEvent [FR-141]")
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-082, FR-141, Section 5 - a SpellCast event reports the spell, the
// class (when there is one), the source and the levels, and every event carries the character id.
// Path: happy.
func TestSpellCastEventFields(t *testing.T) {
	t.Parallel()
	slot := mustCast(t, castCase{spell: l1(t), slot: choice(2)}).Events()[1].(character.SpellCast)
	cls, hasClass := slot.Class()
	at, hasAt := slot.CastAtLevel()
	sc, hasSc := slot.ScalingLevel()
	if slot.Spell() != preparedL1 || cls != character.CasterWizard || !hasClass || slot.Source() != character.CastSourceSlot || at != 2 || !hasAt || sc != 2 || !hasSc || slot.CharacterID() != caster {
		t.Errorf("event = %+v", slot)
	}
	scroll := mustCast(t, castCase{noClass: true, mode: character.CastModeSpellScroll, terms: terms13(t, 4), spell: castSpell(t, spellID(0, 8), 0)}).Events()[0].(character.SpellCast)
	_, hasClass = scroll.Class()
	_, hasAt = scroll.CastAtLevel()
	if sc, _ := scroll.ScalingLevel(); hasClass || hasAt || sc != 4 {
		t.Errorf("scroll event = %+v, want no class, no castAt, scaling 4", scroll)
	}
	failed := character.ConcentrationRules{}.BreakConcentration(castingStarted(t, wizard5(t)), character.BreakDead).Events()[1].(character.CastingFailed)
	if failed.CharacterID() != caster || failed.Spell() != preparedL1 || failed.Reason() != character.EndedDead {
		t.Errorf("failed = %+v", failed)
	}
}
