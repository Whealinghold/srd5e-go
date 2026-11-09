package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/wish-spell.md FR-061 - the five Wish events are the members of the closed family WishEvent, each an Event that
// carries its character id, and a host switches over the family exhaustively.
// Path: happy.
func TestWishEventsAreTheClosedFamilyOfEvent(t *testing.T) {
	t.Parallel()
	cast, sp := slotCast(t, 2, 2)
	active := stressed(t, wishWizard(t), 4, true, false)
	damage, err := wishResolver(t, scriptedRoller(3, 4)).RollWishStressDamage(t.Context(), character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: sp})
	if err != nil {
		t.Fatal(err)
	}
	app, err := character.ApplyWish(wishWizard(t), must(character.NewDuplicateSpell(levelSpell(t, 2))), character.WishStressRoll{})
	if err != nil {
		t.Fatal(err)
	}
	stress, err := character.ApplyWish(wishWizard(t), must(character.NewInstantHealth(1)), wishRoll(t, 3, 20))
	if err != nil {
		t.Fatal(err)
	}
	var events []character.WishEvent
	for _, e := range append(append(append(app.Events(), stress.Events()...), damage.(character.RollWishStressDamageRolled).Events()...),
		character.WishAfterDay(caster, wishStress(t, 1, false, false), character.WishOrdinaryDay).Events()...) {
		events = append(events, e.(character.WishEvent))
	}
	rest := character.NewRestTaker().TakeRest(t.Context(), simpleRest(t, character.LongRest, ran(8, 0))).(character.TakeRestApplied).Applied()
	tr, err := character.WishAfterRest(restHero, wishStress(t, 0, true, false), rest)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range tr.Events() {
		events = append(events, e.(character.WishEvent))
	}
	seen := map[string]bool{}
	for _, e := range events {
		var asEvent character.Event = e
		switch v := e.(type) {
		case character.WishDuplicated:
			seen["duplicated"] = v.Spell() == spellID(2, 5) && v.Level() == content.SpellLevel(2) && asEvent.CharacterID() == caster
		case character.WishStressEndured:
			seen["endured"] = v.Days() == 3 && v.Percentile() == 20 && v.LockedOut() && asEvent.CharacterID() == caster
		case character.WishWeaknessEnded:
			seen["weakness"] = asEvent.CharacterID() == caster
		case character.WishCastDamageEnded:
			seen["castDamage"] = asEvent.CharacterID() == restHero
		case character.WishStressDamageRolled:
			seen["damage"] = v.Level() == 2 && v.Amount() == 7 && asEvent.CharacterID() == caster
		default:
			t.Errorf("unexpected WishEvent %T", e)
		}
	}
	for _, k := range []string{"duplicated", "endured", "weakness", "castDamage", "damage"} {
		if !seen[k] {
			t.Errorf("event %s missing or wrong in %v", k, seen)
		}
	}
}
