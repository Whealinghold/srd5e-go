package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func smAC(tb testing.TB, s ...combat.SpellModifierSource) combat.SpellArmorClassInputs {
	tb.Helper()
	return luMust(combat.QueryArmorClass(s, cid("a"), false))
}

// finalAC is a computed Armor Class with the flat bonuses of in added and then its floor applied: the order of
// character.CalculateDerivedStats (derived-stats.md FR-038, FR-080).
func finalAC(computed int, in combat.SpellArmorClassInputs) int {
	for _, b := range in.FlatBonuses() {
		computed += b.Modifier().Value()
	}
	return in.ApplyFloor(computed)
}

// Requirement: doc/specs/spell-granted-modifiers.md table A rows A1 to A7, A12, FR-040 to FR-046 - flat bonuses of Shield +5, Shield of Faith +2, Haste +2 and Slow -2 add after the base; Barkskin is a floor on the finished Armor Class, applied after every bonus and never as a base; the bonuses are not clamped (SRD 5.2.1, "Shield", pp. 161-162; "Barkskin", p. 112; "Armor Class", p. 177).
// Path: per case.
func TestSpellArmorClassFinal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, row, path string
		computed        int
		spells          []content.SpellID
		want            int
	}{
		{"Shield spell", "A1", "happy", 12, []content.SpellID{"shield"}, 17},
		{"Shield, Shield of Faith, Haste", "A2", "unhappy (rule interaction)", 12, []content.SpellID{"shield", "shield_of_faith", "haste"}, 21},
		{"Slow", "A3", "happy", 12, []content.SpellID{"slow"}, 10},
		{"Barkskin", "A4", "happy", 12, []content.SpellID{"barkskin"}, 17},
		{"Barkskin below a computed 19", "A5", "unhappy (boundary)", 19, []content.SpellID{"barkskin"}, 19},
		{"Barkskin at a computed 17", "FR-043", "unhappy (boundary)", 17, []content.SpellID{"barkskin"}, 17},
		{"Barkskin one below, 16", "FR-043", "unhappy (boundary)", 16, []content.SpellID{"barkskin"}, 17},
		{"Barkskin and Shield of Faith lift 12 to 14, floor raises it", "A6", "unhappy (rule interaction)", 12, []content.SpellID{"barkskin", "shield_of_faith"}, 17},
		{"Barkskin and Shield of Faith on 16, floor not needed", "A7", "unhappy (rule interaction)", 16, []content.SpellID{"barkskin", "shield_of_faith"}, 18},
		{"Barkskin and Shield spell", "A12", "unhappy (rule interaction)", 12, []content.SpellID{"barkskin", "shield"}, 17},
		{"Slow on a low Armor Class is not clamped", "FR-045", "unhappy (boundary)", 1, []content.SpellID{"slow"}, -1},
		{"no spell", "FR-007", "unhappy (empty and minimal)", 12, nil, 12},
		{"a spell with no Armor Class effect", "FR-002", "unhappy (missing content)", 12, []content.SpellID{"longstrider", "fireball"}, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var src []combat.SpellModifierSource
			for i, s := range c.spells {
				src = append(src, smPlain(t, s, i))
			}
			if got := finalAC(c.computed, smAC(t, src...)); got != c.want {
				t.Errorf("final Armor Class = %d, want %d [%s%s]", got, c.want, spellModifiersSpec, c.row)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-040, FR-046, FR-008 - each bonus carries its spell as source, in CastOrder; Shield of Faith and Haste give +4; a second Shield of Faith is one source, so two casts passed as one give +2 and passed as two are rejected as a duplicate.
// Path: unhappy (rule interaction).
func TestSpellArmorClassBonusesCarryTheirSpell(t *testing.T) {
	t.Parallel()
	in := smAC(t, smPlain(t, "haste", 2), smPlain(t, "shield_of_faith", 1), smPlain(t, "shield", 3), smPlain(t, "slow", 4))
	var spells []content.SpellID
	var total int
	for _, b := range in.FlatBonuses() {
		spells = append(spells, b.Source().(character.SpellFlatBonusSource).Spell)
		total += b.Modifier().Value()
	}
	if !slices.Equal(spells, []content.SpellID{"shield_of_faith", "haste", "shield", "slow"}) || total != 2+2+5-2 {
		t.Errorf("bonuses = %v totalling %d", spells, total)
	}
	if got := finalAC(10, smAC(t, smPlain(t, "shield_of_faith", 0), smPlain(t, "haste", 1))); got != 14 {
		t.Errorf("Shield of Faith and Haste give %d, want 14 (FR-046)", got)
	}
	if _, err := combat.QueryArmorClass([]combat.SpellModifierSource{smPlain(t, "shield_of_faith", 0), smSrc(t, "shield_of_faith", "d", "a", 1, combat.NoChoice{})}, cid("a"), false); !errors.Is(err, combat.ErrDuplicateSource) {
		t.Errorf("two Shield of Faith on one creature = %v, want ErrDuplicateSource", err)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-010, FR-084 - a source on another creature gives nothing; a zero creature is invalid input.
// Path: unhappy (rule interaction; invalid input).
func TestQueryArmorClassIsPerCreature(t *testing.T) {
	t.Parallel()
	other := smSrc(t, "shield", "c", "b", 0, combat.NoChoice{})
	if got := smAC(t, other); !got.Equal(combat.SpellArmorClassInputs{}) {
		t.Errorf("a Shield on b gave a's Armor Class %+v", got)
	}
	if _, err := combat.QueryArmorClass(nil, combat.CombatantID(""), false); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero creature = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-041, A8, A9 - Mage Armor gives a base of 13 only when no armor is worn, as a basis with the Dexterity modifier and a Shield allowed (a carried Shield does not stop it); with armor worn the query is neutral (SRD 5.2.1, "Mage Armor", p. 145).
// Path: unhappy (rule interaction; state error).
func TestMageArmorBase(t *testing.T) {
	t.Parallel()
	src := []combat.SpellModifierSource{smPlain(t, "mage_armor", 0)}
	bare := luMust(combat.QueryArmorClass(src, cid("a"), false))
	if v, ok := bare.AlternativeBase(); !ok || v != 13 {
		t.Errorf("A8 base = %d, %v, want 13", v, ok)
	}
	b, ok, err := bare.AlternativeBasis("mage_armor")
	if err != nil || !ok || b.BaseValue() != 13 || !slices.Equal(b.Abilities(), []ability.Ability{ability.Dexterity}) || !b.ShieldAllowed() || b.ID() != "mage_armor" {
		t.Errorf("basis = %+v, %v, %v", b, ok, err)
	}
	if _, _, err := bare.AlternativeBasis("Not Valid"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid feature id = %v, want ErrInvalidInput", err)
	}
	worn := luMust(combat.QueryArmorClass(src, cid("a"), true))
	if _, ok := worn.AlternativeBase(); ok || !worn.Equal(combat.SpellArmorClassInputs{}) {
		t.Errorf("A9 with armor worn gave %+v, want neutral", worn)
	}
	if _, ok, err := worn.AlternativeBasis("mage_armor"); ok || err != nil {
		t.Errorf("A9 basis = %v, %v, want none", ok, err)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-043 - ApplyFloor returns the larger of its argument and the floor and the argument unchanged without a floor, including a negative one and the integer limits.
// Path: unhappy (boundary).
func TestApplyFloor(t *testing.T) {
	t.Parallel()
	bark := smAC(t, smPlain(t, "barkskin", 0))
	if f, ok := bark.Floor(); !ok || f != 17 {
		t.Fatalf("Barkskin floor = %d, %v, want 17", f, ok)
	}
	for in, want := range map[int]int{-5: 17, 0: 17, 16: 17, 17: 17, 18: 18, 1 << 30: 1 << 30} {
		if got := bark.ApplyFloor(in); got != want {
			t.Errorf("Barkskin ApplyFloor(%d) = %d, want %d", in, got, want)
		}
	}
	none := combat.SpellArmorClassInputs{}
	if _, ok := none.Floor(); ok || none.ApplyFloor(-3) != -3 || none.ApplyFloor(12) != 12 {
		t.Error("with no floor ApplyFloor must return its argument")
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-023 - the inputs expose a copy of the bonuses and Equal compares bonuses, base and floor.
// Path: unhappy (immutability).
func TestSpellArmorClassInputsImmutability(t *testing.T) {
	t.Parallel()
	in := smAC(t, smPlain(t, "shield", 0))
	b := in.FlatBonuses()
	b[0], _ = character.NewFlatBonus(modifier.New(99), character.SpellFlatBonusSource{Spell: "shield"})
	if in.FlatBonuses()[0].Modifier().Value() != 5 {
		t.Error("FlatBonuses returned the inputs' own slice")
	}
	if !in.Equal(smAC(t, smPlain(t, "shield", 0))) || in.Equal(smAC(t, smPlain(t, "shield_of_faith", 0))) || in.Equal(smAC(t, smPlain(t, "barkskin", 0))) || smAC(t, smPlain(t, "mage_armor", 0)).Equal(combat.SpellArmorClassInputs{}) {
		t.Error("Equal must compare bonuses, base and floor")
	}
}

func smAttack(tb testing.TB, natural, bonus int, ac d20.ArmorClass) d20.AttackRollResult {
	tb.Helper()
	roll := luMust(d20.NewRoll(modifier.Normal, []int{natural}))
	return luMust(d20.ResolveAttack(roll, d20.NewRollModifiers(modifier.New(bonus)), ac, 20))
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-044, A10, A11 - the Shield spell's +5 applies including to the triggering attack: the same roll and modifiers are judged against the new Armor Class, no die is rolled, a total that now misses is a miss, a natural 20 stays a Critical Hit and a hit, a natural 1 stays a miss, and a lower new Armor Class is allowed (SRD 5.2.1, "Shield", pp. 161-162; "Rolling 20 or 1", p. 7).
// Path: per case.
func TestReevaluateAttack(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, row, path   string
		natural, bonus    int
		was, now          d20.ArmorClass
		wantHit, wantCrit bool
	}{
		{"total 14 hits AC 12, misses AC 17", "A10", "unhappy (rule interaction)", 10, 4, 12, 17, false, false},
		{"total 17 still hits AC 17", "FR-044", "unhappy (boundary)", 13, 4, 12, 17, true, false},
		{"total 16 misses AC 17", "FR-044", "unhappy (boundary)", 12, 4, 12, 17, false, false},
		{"natural 20 against AC 12 then 17", "A11", "unhappy (boundary)", 20, 0, 12, 17, true, true},
		{"natural 20 against a very high AC", "A11", "unhappy (boundary)", 20, 0, 12, 40, true, true},
		{"natural 1 stays a miss against any AC", "FR-033", "unhappy (boundary)", 1, 30, 12, 5, false, false},
		{"a lower new Armor Class turns a miss into a hit", "FR-044", "unhappy (rule interaction)", 5, 2, 17, 7, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			was := smAttack(t, c.natural, c.bonus, c.was)
			got, err := combat.ReevaluateAttack(was, c.now)
			if err != nil || got.Hit() != c.wantHit || got.IsCriticalHit() != c.wantCrit || got.Target() != c.now || got.Total() != was.Total() {
				t.Errorf("ReevaluateAttack = %+v, %v, want hit=%v crit=%v against %d [%s%s]", got, err, c.wantHit, c.wantCrit, c.now, spellModifiersSpec, c.row)
			}
			if !got.Modifiers().Equal(was.Modifiers()) || got.Roll().Natural() != was.Roll().Natural() || got.CriticalRange() != was.CriticalRange() {
				t.Error("the roll, modifiers and critical range must be the same")
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-044 - the error of d20.ResolveAttack is returned unchanged, for the zero attack result.
// Path: unhappy (invalid input).
func TestReevaluateAttackOfZeroResult(t *testing.T) {
	t.Parallel()
	_, want := d20.ResolveAttack(d20.Roll{}, d20.RollModifiers{}, 12, 0)
	_, err := combat.ReevaluateAttack(d20.AttackRollResult{}, 12)
	if err == nil || want == nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ReevaluateAttack(zero) = %v, want %v", err, want)
	}
}
