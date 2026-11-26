package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

func poisonOn(tb testing.TB, pieces int) []combat.BasicPoisonCoating {
	tb.Helper()
	app, err := combat.BasicPoisonRules{}.ApplyToAmmunition(gid("archer"), pieces)
	if err != nil {
		tb.Fatal(err)
	}
	return app.Coatings()
}

func weaponPoison(tb testing.TB) combat.BasicPoisonCoating {
	tb.Helper()
	app, err := combat.BasicPoisonRules{}.ApplyToWeapon(gid("archer"))
	if err != nil {
		tb.Fatal(err)
	}
	return app.Coatings()[0]
}

// Requirement: doc/specs/item-use-effects.md FR-020, FR-026 - ApplyToWeapon returns one potent coating and ApplyToAmmunition 1 to 3 independent potent coatings, each at elapsed zero, with the Bonus Action cost, one vial consumed and one BasicPoisonApplied event; the coatings hold their owner only.
// Path: happy; unhappy (boundary).
func TestBasicPoisonApplication(t *testing.T) {
	t.Parallel()
	rules := combat.BasicPoisonRules{}
	app, err := rules.ApplyToWeapon(gid("archer"))
	if err != nil {
		t.Fatal(err)
	}
	c := app.Coatings()
	if len(c) != 1 || c[0].Object() != combat.BasicPoisonWeapon || c[0].State() != combat.BasicPoisonPotent || c[0].Elapsed() != 0 || !c[0].IsOwnedBy(gid("archer")) || c[0].IsOwnedBy(gid("a")) || c[0].IsOwnedBy(gid("x")) || c[0].IsOwnedBy("") {
		t.Errorf("weapon coatings = %+v", c)
	}
	if app.Cost() != combat.ItemActionCostBonusAction {
		t.Errorf("cost = %v, want bonus action", app.Cost())
	}
	if got, ok := app.Consumption().(combat.ItemConsumed); !ok || got.Count() != 1 || got.Item() != content.MustItemID("poison_basic") {
		t.Errorf("consumption = %#v, want one poison_basic", app.Consumption())
	}
	ev := app.Events()
	if len(ev) != 1 {
		t.Fatalf("events = %v, want one", ev)
	}
	combattest.CheckEvent(t, ev[0], "basic_poison_applied", gid("archer"))
	ev[0] = nil
	c[0] = combat.BasicPoisonCoating{}
	if app.Events()[0] == nil || app.Coatings()[0] == (combat.BasicPoisonCoating{}) {
		t.Error("Events and Coatings return the stored slices, not copies [FR-075]")
	}
	for _, n := range []int{1, 2, 3} {
		if got := poisonOn(t, n); len(got) != n || got[0].Object() != combat.BasicPoisonAmmunitionPiece {
			t.Errorf("%d pieces: %d coatings", n, len(got))
		}
	}
	for _, n := range []int{0, 4, -1, math.MaxInt, math.MinInt} {
		if _, err := rules.ApplyToAmmunition(gid("archer"), n); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldAmmunitionPieces, Value: int64(n)}) {
			t.Errorf("%d pieces: err = %v, want the argument and its value", n, err)
		}
	}
	if _, err := rules.ApplyToWeapon(""); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldOwner}) {
		t.Errorf("zero owner: %v", err)
	}
	if _, err := rules.ApplyToAmmunition("", 1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldOwner}) {
		t.Errorf("zero owner: %v", err)
	}
}

// Requirement: per case
// Path: per case.
func TestBasicPoisonAdvance(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		advance                 time.Duration
		wantState               combat.BasicPoisonState
		wantEvent               bool
	}{
		{"59 seconds stays potent", itemUseSpec + "FR-021 table 4.4a", "unhappy (boundary)", 59 * time.Second, combat.BasicPoisonPotent, false},
		{"60 seconds expires", itemUseSpec + "FR-021 table 4.4a", "unhappy (boundary)", 60 * time.Second, combat.BasicPoisonExpired, true},
		{"zero changes nothing", itemUseSpec + "FR-021", "unhappy (empty)", 0, combat.BasicPoisonPotent, false},
		{"far past the minute expires", itemUseSpec + "FR-021", "happy", math.MaxInt64, combat.BasicPoisonExpired, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			u, err := weaponPoison(t).Advance(c.advance)
			if err != nil || u.Coating().State() != c.wantState || (len(u.Events()) == 1) != c.wantEvent {
				t.Fatalf("Advance = %v, %v, %v [%s]", u.Coating().State(), u.Events(), err, c.requirement)
			}
			if c.wantEvent {
				combattest.CheckEvent(t, u.Events()[0], "basic_poison_spent", gid("archer"))
				if u.Events()[0].(combat.BasicPoisonSpent).Cause() != combat.BasicPoisonExpired {
					t.Error("cause is not expired")
				}
			}
		})
	}
	delivered, _ := weaponPoison(t).AfterWeaponDamage(content.DamagePiercing, 1)
	if u, err := delivered.Coating().Advance(time.Hour); err != nil || u.Coating() != delivered.Coating() || len(u.Events()) != 0 {
		t.Errorf("a delivered coating advanced to %+v with events %v, want unchanged and silent [FR-025]", u.Coating(), u.Events())
	}
	spent, _ := weaponPoison(t).Advance(time.Minute)
	again, err := spent.Coating().Advance(time.Hour)
	if err != nil || again.Coating() != spent.Coating() || len(again.Events()) != 0 {
		t.Errorf("a spent coating advanced to %+v with events %v, want unchanged and silent [FR-025]", again.Coating(), again.Events())
	}
	if _, err := weaponPoison(t).Advance(-1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldElapsed, Value: -1}) {
		t.Errorf("Advance(-1) err = %v [FR-021]", err)
	}
	if _, err := (combat.BasicPoisonCoating{}).Advance(1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldCoating}) {
		t.Errorf("zero coating: %v", err)
	}
	half, _ := weaponPoison(t).Advance(40 * time.Second)
	if u, _ := half.Coating().Advance(19 * time.Second); u.Coating().State() != combat.BasicPoisonPotent || u.Coating().Elapsed() != 59*time.Second {
		t.Errorf("two advances: %+v", u.Coating())
	}
}

// Requirement: per case
// Path: per case.
func TestBasicPoisonExtraDamage(t *testing.T) {
	t.Parallel()
	deliveredU, _ := weaponPoison(t).AfterWeaponDamage(content.DamagePiercing, 1)
	spent, _ := weaponPoison(t).Advance(time.Minute)
	cases := []struct {
		name, requirement, path string
		coat                    combat.BasicPoisonCoating
		typ                     content.DamageType
		want                    combat.BasicPoisonNoExtraReason
	}{
		{"slashing", itemUseSpec + "FR-022", "happy", weaponPoison(t), content.DamageSlashing, 0},
		{"piercing", itemUseSpec + "FR-022", "happy", weaponPoison(t), content.DamagePiercing, 0},
		{"bludgeoning", itemUseSpec + "FR-022 table 4.4a", "unhappy (rule interaction)", weaponPoison(t), content.DamageBludgeoning, combat.BasicPoisonNotPiercingOrSlashing},
		{"thunder", itemUseSpec + "FR-022", "unhappy (rule interaction)", weaponPoison(t), content.DamageThunder, combat.BasicPoisonNotPiercingOrSlashing},
		{"poison", itemUseSpec + "FR-022", "unhappy (rule interaction)", weaponPoison(t), content.DamagePoison, combat.BasicPoisonNotPiercingOrSlashing},
		{"fire", itemUseSpec + "FR-022", "unhappy (rule interaction)", weaponPoison(t), content.DamageFire, combat.BasicPoisonNotPiercingOrSlashing},
		{"expired coating", itemUseSpec + "FR-023 table 4.4a", "unhappy (state error)", spent.Coating(), content.DamagePiercing, combat.BasicPoisonNotPotent},
		{"delivered coating", itemUseSpec + "FR-023", "unhappy (state error)", deliveredU.Coating(), content.DamagePiercing, combat.BasicPoisonNotPotent},
		{"zero coating", itemUseSpec + "FR-023", "unhappy (state error)", combat.BasicPoisonCoating{}, content.DamagePiercing, combat.BasicPoisonNotPotent},
		{"expired and bludgeoning", itemUseSpec + "FR-023", "unhappy (rule interaction)", spent.Coating(), content.DamageBludgeoning, combat.BasicPoisonNotPotent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch v := c.coat.ExtraDamage(c.typ).(type) {
			case combat.BasicPoisonExtraDamage:
				d := v.Damage()
				if c.want != 0 || d.Count() != 1 || d.Die() != dice.D4() || d.Type() != content.DamagePoison || d.Modifier() != 0 || d.AddsAbilityModifier() {
					t.Errorf("extra damage = %+v [%s]", d, c.requirement)
				}
			case combat.BasicPoisonNoExtraDamage:
				if v.Reason() != c.want {
					t.Errorf("no extra damage: %v, want %v [%s]", v.Reason(), c.want, c.requirement)
				}
			}
		})
	}
	c := weaponPoison(t)
	_ = c.ExtraDamage(content.DamageSlashing)
	if c.State() != combat.BasicPoisonPotent {
		t.Error("ExtraDamage spent the coating [FR-022]")
	}
}

// Requirement: per case
// Path: per case.
func TestBasicPoisonAfterWeaponDamage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		typ                     content.DamageType
		taken                   int
		want                    combat.BasicPoisonState
	}{
		{"slashing hit taking 4", itemUseSpec + "FR-024 table 4.4a", "happy", content.DamageSlashing, 4, combat.BasicPoisonDelivered},
		{"piercing hit taking 1", itemUseSpec + "FR-024 table 4.4a", "unhappy (boundary)", content.DamagePiercing, 1, combat.BasicPoisonDelivered},
		{"thunder hit", itemUseSpec + "FR-024", "unhappy (rule interaction)", content.DamageThunder, 6, combat.BasicPoisonPotent},
		{"bludgeoning hit", itemUseSpec + "FR-024 table 4.4a", "unhappy (rule interaction)", content.DamageBludgeoning, 6, combat.BasicPoisonPotent},
		{"piercing hit absorbed whole by Temporary Hit Points", itemUseSpec + "FR-024 table 4.4a", "unhappy (rule interaction)", content.DamagePiercing, 5, combat.BasicPoisonDelivered},
		{"piercing reduced to 0", itemUseSpec + "FR-024 table 4.4a", "unhappy (boundary)", content.DamagePiercing, 0, combat.BasicPoisonPotent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			u, err := weaponPoison(t).AfterWeaponDamage(c.typ, c.taken)
			if err != nil || u.Coating().State() != c.want || (len(u.Events()) == 1) != (c.want == combat.BasicPoisonDelivered) {
				t.Fatalf("AfterWeaponDamage = %v, %v, %v [%s]", u.Coating().State(), u.Events(), err, c.requirement)
			}
			if len(u.Events()) == 1 {
				combattest.CheckEvent(t, u.Events()[0], "basic_poison_spent", gid("archer"))
				if u.Events()[0].(combat.BasicPoisonSpent).Cause() != combat.BasicPoisonDelivered {
					t.Error("cause is not delivered")
				}
			}
		})
	}
	spent, _ := weaponPoison(t).Advance(time.Minute)
	if u, err := spent.Coating().AfterWeaponDamage(content.DamageSlashing, 9); err != nil || u.Coating() != spent.Coating() || len(u.Events()) != 0 {
		t.Errorf("after a hit on an expired coating: %+v, %v [FR-025]", u, err)
	}
	if _, err := weaponPoison(t).AfterWeaponDamage(content.DamageSlashing, -1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldDamageTaken, Value: -1}) {
		t.Errorf("negative damage: %v", err)
	}
	if _, err := (combat.BasicPoisonCoating{}).AfterWeaponDamage(content.DamageSlashing, 1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldCoating}) {
		t.Errorf("zero coating: %v", err)
	}
	u, _ := weaponPoison(t).AfterWeaponDamage(content.DamageSlashing, 4)
	if u2, _ := u.Coating().AfterWeaponDamage(content.DamageSlashing, 4); len(u2.Events()) != 0 {
		t.Error("a delivered coating emitted a second event")
	}
}

// Requirement: doc/specs/item-use-effects.md FR-026, table 4.4a - three arrows, one fired and hitting: that piece is delivered and the two others stay potent, then expire together after 1 minute.
// Path: unhappy (rule interaction).
func TestBasicPoisonCoatingsAreIndependent(t *testing.T) {
	t.Parallel()
	arrows := poisonOn(t, 3)
	hit, _ := arrows[0].AfterWeaponDamage(content.DamagePiercing, 3)
	arrows[0] = hit.Coating()
	states := func() []combat.BasicPoisonState {
		var s []combat.BasicPoisonState
		for _, a := range arrows {
			s = append(s, a.State())
		}
		return s
	}
	if want := []combat.BasicPoisonState{combat.BasicPoisonDelivered, combat.BasicPoisonPotent, combat.BasicPoisonPotent}; !slices.Equal(states(), want) {
		t.Fatalf("states = %v, want %v", states(), want)
	}
	for i := range arrows {
		u, _ := arrows[i].Advance(time.Minute)
		arrows[i] = u.Coating()
	}
	if want := []combat.BasicPoisonState{combat.BasicPoisonDelivered, combat.BasicPoisonExpired, combat.BasicPoisonExpired}; !slices.Equal(states(), want) {
		t.Errorf("states after a minute = %v, want %v", states(), want)
	}
	for _, p := range [][2]string{{combat.BasicPoisonWeapon.String(), "weapon"}, {combat.BasicPoisonAmmunitionPiece.String(), "ammunition_piece"}, {combat.BasicPoisonObject(0).String(), "unknown"},
		{combat.BasicPoisonPotent.String(), "potent"}, {combat.BasicPoisonExpired.String(), "expired"}, {combat.BasicPoisonDelivered.String(), "delivered"}, {combat.BasicPoisonState(0).String(), "unknown"},
		{combat.BasicPoisonNotPiercingOrSlashing.String(), "not_piercing_or_slashing"}, {combat.BasicPoisonNotPotent.String(), "not_potent"}, {combat.BasicPoisonNoExtraReason(0).String(), "unknown"}} {
		if p[0] != p[1] {
			t.Errorf("String() = %q, want %q", p[0], p[1])
		}
	}
}
