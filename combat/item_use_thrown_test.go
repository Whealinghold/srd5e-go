package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func humanoid() combat.ThrownAtCreature {
	return combat.ThrownAtCreature{ID: gid("target"), Size: content.SizeMedium, Type: content.CreatureTypeHumanoid}
}

// Requirement: per case
// Path: per case.
func TestThrownItemRulesPossibility(t *testing.T) {
	t.Parallel()
	seen, unseen, open, total := combat.ThrownSightSeen, combat.ThrownSightUnseen, combat.ThrownCoverTargetable, combat.ThrownCoverTotal
	obj := combat.ThrownAtObject{}
	cases := []struct {
		name, requirement, path string
		kind                    combat.ThrownItemKind
		target                  combat.ThrownTarget
		distance                int
		sight                   combat.ThrownSight
		cover                   combat.ThrownCover
		want                    combat.ThrownNotPossibleReason
	}{
		{"acid at a creature", itemUseSpec + "FR-006", "happy", combat.ThrownAcid, humanoid(), 20, seen, open, 0},
		{"acid at an object", itemUseSpec + "FR-006 table 4.2a acid object", "happy", combat.ThrownAcid, obj, 20, seen, open, 0},
		{"holy water at an object", itemUseSpec + "FR-006", "unhappy (invalid input)", combat.ThrownHolyWater, obj, 5, seen, open, combat.ThrownReasonTargetNotACreature},
		{"net at an object", itemUseSpec + "FR-006", "unhappy (invalid input)", combat.ThrownNet, obj, 5, seen, open, combat.ThrownReasonTargetNotACreature},
		{"total cover", itemUseSpec + "FR-006 table 4.2a total cover", "unhappy (state error)", combat.ThrownAcid, humanoid(), 5, seen, total, combat.ThrownReasonTotalCover},
		{"acid at 0 feet", itemUseSpec + "FR-006", "unhappy (boundary)", combat.ThrownAcid, humanoid(), 0, seen, open, 0},
		{"acid at 20 feet", itemUseSpec + "FR-006", "unhappy (boundary)", combat.ThrownAcid, humanoid(), 20, seen, open, 0},
		{"acid at 21 feet", itemUseSpec + "FR-006", "unhappy (boundary)", combat.ThrownAcid, humanoid(), 21, seen, open, combat.ThrownReasonBeyondRange},
		{"net at 15 feet", itemUseSpec + "FR-006", "unhappy (boundary)", combat.ThrownNet, humanoid(), 15, seen, open, 0},
		{"net at 16 feet", itemUseSpec + "FR-006", "unhappy (boundary)", combat.ThrownNet, humanoid(), 16, seen, open, combat.ThrownReasonBeyondRange},
		{"unseen acid", itemUseSpec + "FR-006 FR-008", "unhappy (rule interaction)", combat.ThrownAcid, humanoid(), 5, unseen, open, combat.ThrownReasonTargetNotSeen},
		{"unseen oil is valid", itemUseSpec + "FR-008", "unhappy (rule interaction)", combat.ThrownOil, humanoid(), 5, unseen, open, 0},
		{"oil at 21 feet unseen", itemUseSpec + "FR-006 FR-008", "unhappy (boundary)", combat.ThrownOil, humanoid(), 21, unseen, open, combat.ThrownReasonBeyondRange},
		{"object then cover then range then sight", itemUseSpec + "FR-006", "unhappy (rule interaction)", combat.ThrownNet, obj, 99, unseen, total, combat.ThrownReasonTargetNotACreature},
		{"cover before range and sight", itemUseSpec + "FR-006", "unhappy (rule interaction)", combat.ThrownAcid, humanoid(), 99, unseen, total, combat.ThrownReasonTotalCover},
		{"range before sight", itemUseSpec + "FR-006", "unhappy (rule interaction)", combat.ThrownAcid, humanoid(), 99, unseen, open, combat.ThrownReasonBeyondRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.ThrownItemRules{}.Possibility(c.kind, c.target, c.distance, c.sight, c.cover)
			if err != nil || got != c.want {
				t.Errorf("Possibility = %v, %v, want %v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-007 - Possibility returns a typed error carrying the violated field, matching srderr.ErrInvalidInput, and never panics.
// Path: unhappy (invalid input).
func TestThrownItemRulesPossibilityInvalid(t *testing.T) {
	t.Parallel()
	ok, seen, open := humanoid(), combat.ThrownSightSeen, combat.ThrownCoverTargetable
	cases := []struct {
		name  string
		call  func() error
		field combat.ItemUseField
		value int64
	}{
		{"kind", func() error { _, e := combat.ThrownItemRules{}.Possibility(0, ok, 5, seen, open); return e }, combat.ItemUseFieldKind, 0},
		{"nil target", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, nil, 5, seen, open)
			return e
		}, combat.ItemUseFieldTarget, 0},
		{"creature without id", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, combat.ThrownAtCreature{Size: content.SizeMedium, Type: content.CreatureTypeBeast}, 5, seen, open)
			return e
		}, combat.ItemUseFieldTarget, 0},
		{"creature without size", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, combat.ThrownAtCreature{ID: "a", Type: content.CreatureTypeBeast}, 5, seen, open)
			return e
		}, combat.ItemUseFieldTarget, 0},
		{"creature without type", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, combat.ThrownAtCreature{ID: "a", Size: content.SizeMedium}, 5, seen, open)
			return e
		}, combat.ItemUseFieldTarget, 0},
		{"negative distance", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, ok, -1, seen, open)
			return e
		}, combat.ItemUseFieldDistanceFeet, -1},
		{"min distance", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, ok, math.MinInt, seen, open)
			return e
		}, combat.ItemUseFieldDistanceFeet, math.MinInt},
		{"sight", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, ok, 5, 0, open)
			return e
		}, combat.ItemUseFieldSight, 0},
		{"cover", func() error {
			_, e := combat.ThrownItemRules{}.Possibility(combat.ThrownAcid, ok, 5, seen, 9)
			return e
		}, combat.ItemUseFieldCover, 9},
	}
	for _, c := range cases {
		err := c.call()
		var ue *combat.ItemUseError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ue) || ue.Field != c.field || ue.Value != c.value ||
			!errors.Is(err, &combat.ItemUseError{Field: c.field, Value: c.value}) || errors.Is(err, &combat.ItemUseError{Field: c.field + 1, Value: c.value}) {
			t.Errorf("%s: err = %v, want field %v value %d", c.name, err, c.field, c.value)
		}
	}
}

// Requirement: doc/specs/item-use-effects.md FR-012, FR-014, FR-019 - Holy Water at a creature that is neither Fiend nor Undead and a Net at a Huge or Gargantuan creature have no save; a flask is consumed whatever the save, a Net never.
// Path: unhappy (rule interaction, boundary).
func TestThrownItemRulesSaveSkipAndConsumption(t *testing.T) {
	t.Parallel()
	creature := func(size content.CreatureSize, typ content.CreatureType) combat.ThrownAtCreature {
		return combat.ThrownAtCreature{ID: "a", Size: size, Type: typ}
	}
	cases := []struct {
		name   string
		kind   combat.ThrownItemKind
		target combat.ThrownTarget
		want   combat.ThrownNoEffectReason
	}{
		{"holy water at a fiend", combat.ThrownHolyWater, creature(content.SizeMedium, content.CreatureTypeFiend), 0},
		{"holy water at an undead", combat.ThrownHolyWater, creature(content.SizeMedium, content.CreatureTypeUndead), 0},
		{"holy water at a humanoid", combat.ThrownHolyWater, humanoid(), combat.ThrownNoEffectNotFiendOrUndead},
		{"holy water at a celestial", combat.ThrownHolyWater, creature(content.SizeMedium, content.CreatureTypeCelestial), combat.ThrownNoEffectNotFiendOrUndead},
		{"acid at a humanoid", combat.ThrownAcid, humanoid(), 0},
		{"net at a large creature", combat.ThrownNet, creature(content.SizeLarge, content.CreatureTypeBeast), 0},
		{"net at a huge creature", combat.ThrownNet, creature(content.SizeHuge, content.CreatureTypeBeast), combat.ThrownNoEffectTargetTooLarge},
		{"net at a gargantuan creature", combat.ThrownNet, creature(content.SizeGargantuan, content.CreatureTypeBeast), combat.ThrownNoEffectTargetTooLarge},
		{"huge acid target saves", combat.ThrownAcid, creature(content.SizeHuge, content.CreatureTypeBeast), 0},
		{"holy water at an object", combat.ThrownHolyWater, combat.ThrownAtObject{}, 0},
	}
	for _, c := range cases {
		if got := (combat.ThrownItemRules{}).SaveSkip(c.kind, c.target); got != c.want {
			t.Errorf("%s: SaveSkip = %v, want %v [FR-012 FR-014]", c.name, got, c.want)
		}
	}
	for k := combat.ThrownAcid; k <= combat.ThrownNet; k++ {
		got := (combat.ThrownItemRules{}).Consumption(k)
		if k == combat.ThrownNet {
			if got != (combat.ItemNotConsumed{}) {
				t.Errorf("Net consumption = %#v, want not consumed", got)
			}
			continue
		}
		if c, ok := got.(combat.ItemConsumed); !ok || c.Count() != 1 || c.Item() != k.ItemID() {
			t.Errorf("%v consumption = %#v, want one %q", k, got, k.ItemID())
		}
	}
	if got := (combat.ThrownItemRules{}).Consumption(0); got != (combat.ItemNotConsumed{}) {
		t.Errorf("invalid kind consumption = %#v", got)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-011, FR-013, FR-014, FR-016 - the effect of a failed save: damage for Acid and Holy Water, damage and burning for Alchemist's Fire, a fresh wet coat for Oil, the Net's restraint; the damage is unadjusted and an invalid kind or amount is an error.
// Path: happy; unhappy (invalid input).
func TestThrownItemRulesOnFailedSave(t *testing.T) {
	t.Parallel()
	rules := combat.ThrownItemRules{}
	for kind, typ := range map[combat.ThrownItemKind]content.DamageType{combat.ThrownAcid: content.DamageAcid, combat.ThrownHolyWater: content.DamageRadiant} {
		e, err := rules.OnFailedSave(kind, 7)
		d, ok := e.(combat.ThrownDamageDealt)
		if err != nil || !ok || d.Damage() != combat.MustDamage(7, typ) {
			t.Errorf("%v: effect = %#v, %v, want 7 %v damage", kind, e, err, typ)
		}
	}
	e, err := rules.OnFailedSave(combat.ThrownAlchemistsFire, 0)
	if b, ok := e.(combat.ThrownDamageAndBurningStarted); err != nil || !ok || b.Damage() != combat.MustDamage(0, content.DamageFire) {
		t.Errorf("Alchemist's Fire effect = %#v, %v", e, err)
	}
	e, err = rules.OnFailedSave(combat.ThrownOil, 0)
	if o, ok := e.(combat.ThrownOilCovered); err != nil || !ok || o.Coat() != combat.NewOilCoat() || o.Coat().State() != combat.OilCoatWet || o.Coat().Elapsed() != 0 {
		t.Errorf("Oil effect = %#v, %v, want a fresh wet coat", e, err)
	}
	e, err = rules.OnFailedSave(combat.ThrownNet, 0)
	if n, ok := e.(combat.ThrownRestrained); err != nil || !ok || n.Restraint() != (combat.NetRestraint{}) {
		t.Errorf("Net effect = %#v, %v", e, err)
	}
	if _, err := rules.OnFailedSave(0, 1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldKind}) {
		t.Errorf("invalid kind: %v", err)
	}
	if _, err := rules.OnFailedSave(combat.ThrownAcid, -1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldDamage, Value: -1}) {
		t.Errorf("negative damage: %v", err)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-014 - the Net's restraint: escape is Strength (Athletics) at DC 10, the Net is Armor Class 10 with 5 Hit Points and immune to Bludgeoning, Poison and Psychic; a success escapes, a failure holds fast, a destroyed Net frees the creature.
// Path: happy; unhappy (boundary).
func TestNetRestraint(t *testing.T) {
	t.Parallel()
	r := combat.NetRestraint{}
	check := r.EscapeCheck()
	if check.DifficultyClass() != 10 || check.Skill() != content.SkillAthletics || check.Ability().String() != "strength" {
		t.Errorf("escape check = %v %v DC %d, want Strength (Athletics) DC 10", check.Ability(), check.Skill(), check.DifficultyClass())
	}
	st := r.Statistics()
	if st.ArmorClass() != 10 || st.HitPoints().Maximum() != 5 || st.HitPoints().Current() != 5 {
		t.Errorf("statistics = AC %d, HP %v, want 10 and 5", st.ArmorClass(), st.HitPoints())
	}
	if imm := st.Affinities().Immunities(); !slices.Equal(imm, []content.DamageType{content.DamageBludgeoning, content.DamagePoison, content.DamagePsychic}) {
		t.Errorf("immunities = %v, want Bludgeoning, Poison and Psychic", imm)
	}
	if got := r.Escape(d20.Success); got != (combat.NetEscaped{}) {
		t.Errorf("Escape(success) = %#v", got)
	}
	if got := r.Escape(0); got != (combat.NetHoldsFast{}) {
		t.Errorf("Escape(0) = %#v, want the Net to hold fast", got)
	}
	if got := r.Escape(d20.Failure); got != (combat.NetHoldsFast{}) {
		t.Errorf("Escape(failure) = %#v", got)
	}
	if got := r.Destroyed(); got != (combat.NetEscaped{}) {
		t.Errorf("Destroyed = %#v", got)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-073 - the reason enums print the snake_case values the catalogue lists; the zero reason is none and anything else unknown.
// Path: unhappy (invalid input).
func TestThrownEnumStrings(t *testing.T) {
	t.Parallel()
	for _, p := range [][2]string{
		{combat.ThrownReasonTargetNotACreature.String(), "target_not_a_creature"}, {combat.ThrownReasonTotalCover.String(), "total_cover"},
		{combat.ThrownReasonBeyondRange.String(), "beyond_range"}, {combat.ThrownReasonTargetNotSeen.String(), "target_not_seen"},
		{combat.ThrownNotPossibleReason(0).String(), "none"}, {combat.ThrownNotPossibleReason(-1).String(), "unknown"}, {combat.ThrownNotPossibleReason(9).String(), "unknown"},
		{combat.ThrownNoEffectSaveSucceeded.String(), "save_succeeded"}, {combat.ThrownNoEffectNotFiendOrUndead.String(), "not_fiend_or_undead"},
		{combat.ThrownNoEffectTargetTooLarge.String(), "target_too_large"}, {combat.ThrownNoEffectReason(0).String(), "none"}, {combat.ThrownNoEffectReason(-1).String(), "unknown"}, {combat.ThrownNoEffectReason(9).String(), "unknown"},
		{combat.ThrownSightSeen.String(), "seen"}, {combat.ThrownSightUnseen.String(), "unseen"}, {combat.ThrownSight(0).String(), "unknown"},
		{combat.ThrownCoverTargetable.String(), "targetable"}, {combat.ThrownCoverTotal.String(), "total"}, {combat.ThrownCover(0).String(), "unknown"},
		{combat.ThrownAcid.String(), "acid"}, {combat.ThrownNet.String(), "net"},
		{combat.ItemUseFieldKind.String(), "kind"}, {combat.ItemUseFieldDamage.String(), "damage"}, {combat.ItemUseField(0).String(), "unknown"},
	} {
		if p[0] != p[1] {
			t.Errorf("String() = %q, want %q", p[0], p[1])
		}
	}
	for _, v := range []int{-1, 0, 3, 9} {
		if combat.ThrownSight(v).Valid() || combat.ThrownCover(v).Valid() {
			t.Errorf("%d is valid as a sight or a cover", v)
		}
	}
	if !combat.ThrownSightSeen.Valid() || !combat.ThrownSightUnseen.Valid() || !combat.ThrownCoverTargetable.Valid() || !combat.ThrownCoverTotal.Valid() {
		t.Error("a member is not valid")
	}
}
