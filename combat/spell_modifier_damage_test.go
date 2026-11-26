package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func smCurse(tb testing.TB, caster, target string, e combat.CurseEffect) combat.SpellModifierSource {
	tb.Helper()
	return smSrc(tb, "bestow_curse", caster, target, 0, combat.ChosenCurse{Effect: e})
}

// Requirement: doc/specs/spell-granted-modifiers.md table D rows D9, D10, D12, FR-061 to FR-064 - Bestow Curse with extra damage adds one 1d8 Necrotic component for an attack roll that hits and for a spell's damage, never for other damage, for another attacker, for another target or for another curse effect; the Critical Hit doubles it by the attack's own rule; the extra damage is part of the attack, so a miss produces no event (SRD 5.2.1, "Bestow Curse", p. 113; "Critical Hits", p. 16).
// Path: per case.
func TestQueryDamageRiders(t *testing.T) {
	t.Parallel()
	extra := smCurse(t, "c", "m", combat.CurseExtraDamage{})
	want := []combat.DamageComponent{combat.MustDiceDamage(1, dice.D8(), content.DamageNecrotic, 0, false)}
	cases := []struct {
		name, row, path string
		sources         []combat.SpellModifierSource
		attacker, tgt   string
		ev              combat.DamageEvent
		wantRider       bool
	}{
		{"attack roll hit", "D9", "happy", []combat.SpellModifierSource{extra}, "c", "m", combat.AttackRollHitEvent{}, true},
		{"spell damage (Fireball)", "D9", "unhappy (rule interaction)", []combat.SpellModifierSource{extra}, "c", "m", combat.SpellDamageEvent{}, true},
		{"trap damage", "D12", "unhappy (rule interaction)", []combat.SpellModifierSource{extra}, "c", "m", combat.OtherDamageEvent{}, false},
		{"another creature hits the target", "D12", "unhappy (rule interaction)", []combat.SpellModifierSource{extra}, "d", "m", combat.AttackRollHitEvent{}, false},
		{"the caster hits another creature", "D12", "unhappy (rule interaction)", []combat.SpellModifierSource{extra}, "c", "x", combat.AttackRollHitEvent{}, false},
		{"the cursed creature hits the caster", "FR-010", "unhappy (rule interaction)", []combat.SpellModifierSource{extra}, "m", "c", combat.AttackRollHitEvent{}, false},
		{"a curse of attack rolls adds no damage", "FR-061", "unhappy (boundary)", []combat.SpellModifierSource{smCurse(t, "c", "m", combat.CurseAttackRolls{})}, "c", "m", combat.AttackRollHitEvent{}, false},
		{"a curse on an ability adds no damage", "FR-061", "unhappy (boundary)", []combat.SpellModifierSource{smCurse(t, "c", "m", combat.CurseDodge{})}, "c", "m", combat.SpellDamageEvent{}, false},
		{"no sources", "FR-024", "unhappy (empty and minimal)", nil, "c", "m", combat.AttackRollHitEvent{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.QueryDamageRiders(c.sources, cid(c.attacker), cid(c.tgt), c.ev)
			if err != nil || (len(got) == 1) != c.wantRider || (c.wantRider && got[0] != want[0]) || (!c.wantRider && len(got) != 0) {
				t.Errorf("riders = %v, %v, want rider=%v [%s%s]", got, err, c.wantRider, spellModifiersSpec, c.row)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md D10, FR-062 - on a Critical Hit the extra die is rolled twice by the attack's own rule (AttackDamageComposer doubles the dice of every DiceDamage), with no rule of this spec and no flat modifier (SRD 5.2.1, "Critical Hits", p. 16).
// Path: unhappy (boundary).
func TestBestowCurseDieDoublesOnACriticalHit(t *testing.T) {
	t.Parallel()
	riders, err := combat.QueryDamageRiders([]combat.SpellModifierSource{smCurse(t, "c", "m", combat.CurseExtraDamage{})}, cid("c"), cid("m"), combat.AttackRollHitEvent{})
	if err != nil || len(riders) != 1 {
		t.Fatalf("riders = %v, %v", riders, err)
	}
	for crit, count := range map[bool]int{false: 1, true: 2} {
		e, ok := (combat.AttackDamageComposer{}).Dice(riders[0], crit)
		if !ok || e.Count() != count || e.Die() != dice.D8() || e.Modifier() != 0 || riders[0].Type() != content.DamageNecrotic {
			t.Errorf("dice for critical=%v = %v, %v, want %d d8", crit, e, ok, count)
		}
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-061, FR-084 - a nil event, a zero attacker or a zero target is invalid input.
// Path: unhappy (invalid input).
func TestQueryDamageRidersInvalidInput(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func() ([]combat.DamageComponent, error){
		"nil event": func() ([]combat.DamageComponent, error) {
			return combat.QueryDamageRiders(nil, cid("c"), cid("m"), nil)
		},
		"zero attacker": func() ([]combat.DamageComponent, error) {
			return combat.QueryDamageRiders(nil, "", cid("m"), combat.OtherDamageEvent{})
		},
		"zero target": func() ([]combat.DamageComponent, error) {
			return combat.QueryDamageRiders(nil, cid("c"), "", combat.OtherDamageEvent{})
		},
	} {
		if got, err := f(); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
			t.Errorf("%s = %v, %v, want nil and ErrInvalidInput", name, got, err)
		}
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-065, table D rows D6, D7 - Resistance gives a 1d4 for the chosen type only, once per turn (the host's flag), only on its target; a type outside its choice, an already-reduced turn and another creature give none; a zero creature or an invalid type is invalid input (SRD 5.2.1, "Resistance", p. 158).
// Path: per case.
func TestQueryDamageReduction(t *testing.T) {
	t.Parallel()
	res := []combat.SpellModifierSource{smSrc(t, "resistance", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageFire})}
	cases := []struct {
		name, row, path string
		who             string
		typ             content.DamageType
		already         bool
		want            bool
	}{
		{"Fire damage", "FR-065", "happy", "a", content.DamageFire, false, true},
		{"Cold damage", "D6", "unhappy (boundary)", "a", content.DamageCold, false, false},
		{"already reduced this turn", "D7", "unhappy (state error)", "a", content.DamageFire, true, false},
		{"another creature", "FR-010", "unhappy (rule interaction)", "b", content.DamageFire, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.QueryDamageReduction(res, cid(c.who), c.typ, c.already)
			e, has := got.Dice()
			if err != nil || has != c.want || (has && (e.Count() != 1 || e.Die() != dice.D4() || e.Modifier() != 0)) {
				t.Errorf("reduction = %+v, %v, want %v [%s%s]", got, err, c.want, spellModifiersSpec, c.row)
			}
		})
	}
	if _, err := combat.QueryDamageReduction(res, "", content.DamageFire, false); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero creature = %v", err)
	}
	if _, err := combat.QueryDamageReduction(res, cid("a"), content.DamageType(0), false); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid type = %v", err)
	}
	if got, err := combat.QueryDamageReduction(nil, cid("a"), content.DamageFire, false); err != nil {
		t.Errorf("no sources = %v", err)
	} else if _, has := got.Dice(); has {
		t.Error("no sources must give none")
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-066, D13 - ToAdjustment gives a flat adjustment of -rolled scoped to the chosen type for 1 to 4 and an error matching srderr.ErrInvalidInput for 0, 5, a negative number and for no reduction; it never panics.
// Path: unhappy (invalid input; boundary).
func TestToAdjustment(t *testing.T) {
	t.Parallel()
	red := luMust(combat.QueryDamageReduction([]combat.SpellModifierSource{smSrc(t, "resistance", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageFire})}, cid("a"), content.DamageFire, false))
	for _, n := range []int{1, 2, 3, 4} {
		adj, err := red.ToAdjustment(n)
		if err != nil || adj.Amount() != -n || adj.Scope() != (combat.DamageOfType{Type: content.DamageFire}) {
			t.Errorf("ToAdjustment(%d) = %+v, %v", n, adj, err)
		}
	}
	for _, n := range []int{0, 5, -1, 1 << 20} {
		if _, err := red.ToAdjustment(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ToAdjustment(%d) = %v, want ErrInvalidInput (D13)", n, err)
		}
	}
	if _, err := (combat.SpellDamageReduction{}).ToAdjustment(2); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ToAdjustment on no reduction = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md table D rows D5, D8, FR-066, FR-067 - through the real damage pipeline (ApplyDamage), the Resistance spell's reduction comes before the creature's Resistance: 11 Fire, d4 shows 3, Resistant to Fire gives 8 halved to 4 (the other reading gives 2); a roll of 4 on 3 damage gives 0 (floored); it covers only the chosen type (SRD 5.2.1, "Resistance", p. 158; "Order of Application", "No Stacking", p. 17).
// Path: per case.
func TestResistanceSpellInTheDamagePipeline(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, row, path string
		damage          combat.Damage
		rolled          int
		resistant       bool
		wantDealt       int
	}{
		{"11 Fire, rolled 3, Resistant to Fire", "D5", "unhappy (rule interaction)", fire(11), 3, true, 4},
		{"11 Fire, rolled 3, not Resistant", "FR-066", "happy", fire(11), 3, false, 8},
		{"3 Fire, rolled 4", "D8", "unhappy (boundary)", fire(3), 4, false, 0},
		{"Cold damage is not reduced", "D6", "unhappy (boundary)", combat.MustDamage(11, content.DamageCold), 3, false, 11},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			red := luMust(combat.QueryDamageReduction([]combat.SpellModifierSource{smSrc(t, "resistance", "c", "hero", 0, combat.ChosenResistanceType{Type: content.DamageFire})}, subject(), content.DamageFire, false))
			adj := luMust(red.ToAdjustment(c.rolled))
			var res []combat.DamageScope
			if c.resistant {
				res = fireScope()
			}
			st := status(t, 50, 50, 0, combat.Alive{}, combat.FallsUnconscious, affinities(t, res, nil, nil))
			cmd := luMust(combat.NewApplyDamageCommand(subject(), st, []combat.Damage{c.damage}, []combat.DamageAdjustment{adj}, combat.OtherSource, combat.NormalHit, nil))
			out, err := combat.NewApplyDamage().Run(context.Background(), cmd)
			applied, ok := out.(combat.DamageApplied)
			if err != nil || !ok || len(applied.Resolutions()) != 1 || applied.Resolutions()[0].Dealt() != c.wantDealt {
				t.Errorf("ApplyDamage = %+v, %v, want %d dealt [%s%s]", out, err, c.wantDealt, spellModifiersSpec, c.row)
			}
		})
	}
}
