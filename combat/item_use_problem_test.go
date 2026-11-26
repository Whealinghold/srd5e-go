package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/item-use-effects.md FR-007 - an ItemUseError renders a lowercase text naming the field (the text is not a contract), matches only an ItemUseError of the same field and value, and the field names print in snake_case.
// Path: unhappy (invalid input).
func TestItemUseErrorRenders(t *testing.T) {
	t.Parallel()
	var err error = &combat.ItemUseError{Field: combat.ItemUseFieldDistanceFeet, Value: -3}
	if err.Error() == "" || (&combat.ItemUseError{}).Error() == "" {
		t.Error("an ItemUseError renders no text")
	}
	if errors.Is(err, errors.New("other")) || (*combat.ItemUseError)(nil).Is(&combat.ItemUseError{}) || (&combat.ItemUseError{}).Is((*combat.ItemUseError)(nil)) {
		t.Error("an ItemUseError matches something it should not")
	}
	if errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldDistanceFeet, Value: -4}) || errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldKind, Value: -3}) {
		t.Error("an ItemUseError matches another field or value")
	}
	names := map[combat.ItemUseField]string{combat.ItemUseFieldKind: "kind", combat.ItemUseFieldThrower: "thrower", combat.ItemUseFieldTarget: "target",
		combat.ItemUseFieldDistanceFeet: "distance_feet", combat.ItemUseFieldSight: "sight", combat.ItemUseFieldCover: "cover", combat.ItemUseFieldAttempt: "attempt",
		combat.ItemUseFieldElapsed: "elapsed", combat.ItemUseFieldDamageTaken: "damage_taken", combat.ItemUseFieldAmmunitionPieces: "ammunition_pieces",
		combat.ItemUseFieldUsesRemaining: "uses_remaining", combat.ItemUseFieldOwner: "owner", combat.ItemUseFieldUser: "user", combat.ItemUseFieldPatient: "patient",
		combat.ItemUseFieldStatus: "status", combat.ItemUseFieldPotionUse: "potion_use", combat.ItemUseFieldCoating: "coating", combat.ItemUseFieldEffect: "effect",
		combat.ItemUseFieldDamage: "damage", 0: "unknown", 99: "unknown"}
	for f, want := range names {
		if f.String() != want {
			t.Errorf("field %d = %q, want %q", f, f.String(), want)
		}
	}
}
