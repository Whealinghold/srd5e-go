package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-spellcasting.md FR-030, FR-011 - the report carries the spell id, the levels, the ability and its Modifier, the target limit and the note unchanged, comparable with ==, and its attack modifiers are a fresh value each call.
// Path: happy.
func TestReportAccessors(t *testing.T) {
	t.Parallel()
	rig := mscNewRig(t, mscMonster(t, nil))
	rep := mscPerformed(t, rig.cast(mscList(t, mscShapechange), mscSpells()["shapechange"])).Report
	again := mscPerformed(t, rig.cast(mscList(t, mscShapechange), mscSpells()["shapechange"])).Report
	if rep != again {
		t.Errorf("two reports of one cast differ")
	}
	if at, _ := rep.CastAtLevel(); rep.Spell() != "shapechange" || at != 9 || rep.ScalingLevel() != 9 || rep.Ability() != ability.Intelligence ||
		rep.TargetLimit() != combat.TargetLimitUnrestricted ||
		rep.RestrictionNote() != "Beast or Humanoid form only, and no Concentration or Temporary Hit Points required" {
		t.Errorf("report = %v level %d scaling %d ability %v limit %v note %q", rep.Spell(), at, rep.ScalingLevel(), rep.Ability(), rep.TargetLimit(), rep.RestrictionNote())
	}
	if _, has := (combat.MonsterCastReport{}).AttackModifiers(); has {
		t.Errorf("the zero report has attack modifiers")
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-031, FR-035 - each refusal reason, target limit and cast-context value has a snake_case name, and an undefined value reads unknown.
// Path: unhappy (boundary).
func TestEnumNames(t *testing.T) {
	t.Parallel()
	refusals := []string{"not_in_spell_list", "casting_ref_mismatch", "casting_already_in_progress", "cannot_take_actions", "components_statement_unrecognized",
		"verbal_component_missing", "somatic_component_missing", "self_only_violated", "missing_save_dc", "missing_spell_attack_bonus", "uses_exhausted", "casting_not_in_progress"}
	for i, want := range refusals {
		if got := combat.MonsterCastRefusal(i + 1).String(); got != want {
			t.Errorf("MonsterCastRefusal(%d) = %q, want %q", i+1, got, want)
		}
	}
	if combat.MonsterCastRefusalNotInSpellList != 1 || combat.MonsterCastRefusalNotInProgress != 12 || combat.MonsterCastRefusalUsesExhausted != 11 {
		t.Errorf("the refusal order moved")
	}
	for _, v := range []combat.MonsterCastRefusal{0, 13, -1} {
		if v.String() != "unknown" {
			t.Errorf("MonsterCastRefusal(%d) = %q, want unknown", v, v.String())
		}
	}
	if combat.TargetLimitSelfOnly.String() != "self_only" || combat.TargetLimitUnrestricted.String() != "unrestricted" || combat.TargetLimit(0).String() != "unknown" {
		t.Errorf("TargetLimit names = %q %q %q", combat.TargetLimitSelfOnly, combat.TargetLimitUnrestricted, combat.TargetLimit(0))
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-020 - a cast context needs a defined action availability and reads back what it was given.
// Path: unhappy (invalid input).
func TestMonsterCastContext(t *testing.T) {
	t.Parallel()
	c := mscContext(true, false, combat.ConditionActionAvailabilityNone)
	if !c.CanSpeak() || c.CanMakeGestures() || c.Actions() != combat.ConditionActionAvailabilityNone || c.IsZero() {
		t.Errorf("context = %v %v %v", c.CanSpeak(), c.CanMakeGestures(), c.Actions())
	}
	for _, bad := range []combat.ConditionActionAvailability{0, 3, -1} {
		if _, err := combat.NewMonsterCastContext(true, true, bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewMonsterCastContext(%d) error = %v, want ErrInvalidInput", bad, err)
		}
	}
	if !(combat.MonsterCastContext{}).IsZero() {
		t.Errorf("the zero context is not zero")
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-002 - a ListCast reads back its reference and mode.
// Path: happy.
func TestMonsterListCastAccessors(t *testing.T) {
	t.Parallel()
	mode := luMust(content.NewCastingTimeMode("Action", content.NewCastingTimeMagicAction()))
	c := combat.NewMonsterListCast(mscRef(t, mscGeas), mode)
	if c.Spell() != mscRef(t, mscGeas) || c.CastingTimeMode() != mode {
		t.Errorf("ListCast = %v %v", c.Spell(), c.CastingTimeMode())
	}
}
