package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/wish-spell.md section 7, FR-066 - the span names, attribute keys and attribute values of the Wish use cases (the effect
// values, the operation values, the new result values and the snake_case of every ResolveWishRejectionKind) are a public contract: their strings are
// pinned here and each appears in doc/observability/signals.md and, for the attribute values, in doc/rules/observability.md.
// Path: happy.
func TestWishSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	pins := []struct{ got, want string }{
		{character.SpanResolveWish, "srd5e.character.resolve_wish"},
		{character.SpanRollWishStressDamage, "srd5e.character.roll_wish_stress_damage"},
		{character.AttrWishEffect, "srd5e.character.wish_effect"},
		{character.AttrWishLockout, "srd5e.character.wish_lockout"},
		{character.ValueOperationResolveWish, "resolve_wish"},
		{character.ValueOperationRollWishStressDamage, "roll_wish_stress_damage"},
		{character.ValueResultDuplicated, "duplicated"},
		{character.ValueResultDamage, "damage"},
		{character.ValueWishEffectDuplicate, "duplicate"},
		{character.ValueWishEffectObjectCreation, "object_creation"},
		{character.ValueWishEffectInstantHealth, "instant_health"},
		{character.ValueWishEffectResistance, "resistance"},
		{character.ValueWishEffectSpellImmunity, "spell_immunity"},
		{character.ValueWishEffectSuddenLearning, "sudden_learning"},
		{character.ValueWishEffectRollRedo, "roll_redo"},
		{character.ValueWishEffectReshapeReality, "reshape_reality"},
	}
	for kind := character.WishCannotCastAgain; kind.Valid(); kind++ {
		pins = append(pins, struct{ got, want string }{kind.String(), kind.String()})
	}
	for _, p := range pins {
		if p.got != p.want {
			t.Errorf("constant = %q, want %q", p.got, p.want)
		}
		if !strings.Contains(string(catalogue), "`"+p.want+"`") {
			t.Errorf("signals.md does not list %q", p.want)
		}
		if !strings.HasPrefix(p.want, "srd5e.") && !strings.Contains(string(rules), "`"+p.want+"`") {
			t.Errorf("observability.md does not list %q", p.want)
		}
	}
	if n := len(pins) - 16; n != 9 {
		t.Errorf("%d rejection kinds pinned, want 9", n)
	}
}
