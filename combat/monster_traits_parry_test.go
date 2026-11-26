package combat_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
)

func mtParryTrait(tb testing.TB, bonus int) combat.ParryTrait {
	tb.Helper()
	return luMust(combat.NewParryTrait(luRef(tb, combat.MonsterPartReactions, 0), bonus))
}

// Requirement: doc/specs/monster-traits.md FR-075 - Parry is available only for a hit by a melee attack roll while holding a weapon; the reason is the first failing one.
// Path: per case.
func TestAssessParry(t *testing.T) {
	t.Parallel()
	hit, miss := mtAttack(t, 15, 0, 14), mtAttack(t, 5, 0, 14)
	cases := []struct {
		name  string
		trait combat.ParryTrait
		facts combat.ParryFacts
		want  combat.ParryOffer
	}{
		{"available", mtParryTrait(t, 2), combat.ParryFacts{Attack: hit, IsMeleeAttackRoll: true, HoldingWeapon: true}, combat.ParryAvailable{}},
		{"missed", mtParryTrait(t, 2), combat.ParryFacts{Attack: miss, IsMeleeAttackRoll: true, HoldingWeapon: true}, combat.ParryUnavailable{Reason: combat.ParryAttackMissed}},
		{"zero attack", mtParryTrait(t, 2), combat.ParryFacts{IsMeleeAttackRoll: true, HoldingWeapon: true}, combat.ParryUnavailable{Reason: combat.ParryAttackMissed}},
		{"ranged", mtParryTrait(t, 2), combat.ParryFacts{Attack: hit, HoldingWeapon: true}, combat.ParryUnavailable{Reason: combat.ParryNotMeleeAttackRoll}},
		{"no weapon", mtParryTrait(t, 2), combat.ParryFacts{Attack: hit, IsMeleeAttackRoll: true}, combat.ParryUnavailable{Reason: combat.ParryNotHoldingWeapon}},
		{"first failing reason", mtParryTrait(t, 2), combat.ParryFacts{Attack: miss}, combat.ParryUnavailable{Reason: combat.ParryAttackMissed}},
		{"ranged and unarmed", mtParryTrait(t, 2), combat.ParryFacts{Attack: hit}, combat.ParryUnavailable{Reason: combat.ParryNotMeleeAttackRoll}},
		{"zero trait", combat.ParryTrait{}, combat.ParryFacts{Attack: hit, IsMeleeAttackRoll: true, HoldingWeapon: true}, combat.ParryUnavailable{Reason: combat.ParryNoTrait}},
	}
	for _, c := range cases {
		if got := combat.AssessParry(c.trait, c.facts); got != c.want {
			t.Errorf("%s: %#v, want %#v", c.name, got, c.want)
		}
	}
	for i, w := range []string{"no_trait", "attack_missed", "not_melee_attack_roll", "not_holding_weapon", "unknown"} {
		if combat.ParryUnavailableReason(i+1).String() != w {
			t.Errorf("reason %d", i+1)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-076, FR-077, table PT-1 - re-resolving against the raised Armor Class: +2 turns 15 vs 14 into a miss, 16 vs 14 still hits, +5 turns 18 into a miss, a natural 20 stays a Critical Hit, the original is unchanged.
// Path: per case.
func TestRespondToParry(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                          string
		natural, bonusToRoll, ac, par int
		miss, critical                bool
	}{
		{"Parry +2 turns a hit", 12, 3, 14, 2, true, false},
		{"Parry exactly meets", 13, 3, 14, 2, false, false},
		{"Parry +5", 15, 3, 14, 5, true, false},
		{"Parry +5 at 19", 16, 3, 14, 5, false, false},
		{"natural 20", 20, -8, 14, 5, false, true},
	}
	for _, c := range cases {
		original := mtAttack(t, c.natural, c.bonusToRoll, c.ac)
		if !original.Hit() {
			t.Fatalf("%s: the original does not hit", c.name)
		}
		r, err := combat.RespondToParry(mtParryTrait(t, c.par), mtID(), original)
		if err != nil || r.BecameMiss() != c.miss || r.Result().Hit() == c.miss || r.Result().IsCriticalHit() != c.critical {
			t.Fatalf("%s: %+v, %v", c.name, r, err)
		}
		if r.Result().Target() != d20.ArmorClass(c.ac+c.par) || r.Result().Total() != original.Total() || !original.Equal(mtAttack(t, c.natural, c.bonusToRoll, c.ac)) {
			t.Errorf("%s: target %v total %v", c.name, r.Result().Target(), r.Result().Total())
		}
		if e := r.Event(); e.Bonus() != c.par || e.BecameMiss() != c.miss || e.Kind() != "parry_used" {
			t.Errorf("%s: event %+v", c.name, e)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-078 - responding to a miss, the zero result, a zero trait or subject is an error; the Armor Class sum saturates.
// Path: unhappy (invalid input).
func TestRespondToParryInvalid(t *testing.T) {
	t.Parallel()
	hit := mtAttack(t, 15, 0, 14)
	for name, f := range map[string]func() (combat.ParryResponse, error){
		"miss": func() (combat.ParryResponse, error) {
			return combat.RespondToParry(mtParryTrait(t, 2), mtID(), mtAttack(t, 5, 0, 14))
		},
		"natural 1": func() (combat.ParryResponse, error) {
			return combat.RespondToParry(mtParryTrait(t, 2), mtID(), mtAttack(t, 1, 99, 1))
		},
		"zero result": func() (combat.ParryResponse, error) {
			return combat.RespondToParry(mtParryTrait(t, 2), mtID(), d20.AttackRollResult{})
		},
		"zero trait":   func() (combat.ParryResponse, error) { return combat.RespondToParry(combat.ParryTrait{}, mtID(), hit) },
		"zero subject": func() (combat.ParryResponse, error) { return combat.RespondToParry(mtParryTrait(t, 2), "", hit) },
	} {
		if _, err := f(); !isInvalid(err) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	r, err := combat.RespondToParry(mtParryTrait(t, 5), mtID(), mtAttack(t, 20, 0, math.MaxInt))
	if err != nil || r.Result().Target() != d20.ArmorClass(math.MaxInt) || !r.Result().IsCriticalHit() {
		t.Errorf("saturating AC: %v %v", r.Result().Target(), err)
	}
}
