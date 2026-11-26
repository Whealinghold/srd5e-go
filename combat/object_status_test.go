package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// envObject builds a Medium object with hp Hit Points, the standard affinities and no scores.
func envObject(tb testing.TB, hp int, threshold int, aff combat.DamageAffinities) combat.ObjectStatus {
	tb.Helper()
	var th combat.DamageThreshold
	if threshold > 0 {
		var err error
		if th, err = combat.NewDamageThreshold(threshold); err != nil {
			tb.Fatal(err)
		}
	}
	s, err := combat.NewObjectStatus(content.SizeMedium, 15, combat.MustHitPoints(hp, 30, 0), th, aff, nil)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-072 - an object holds Hit Points with no Temporary Hit Points; an invalid size and an unknown ability scores value are ErrInvalidInput errors; the zero status is invalid and the accessors read back.
// Path: unhappy (invalid input).
func TestNewObjectStatus(t *testing.T) {
	t.Parallel()
	aff := (combat.ObjectAffinities{}).Standard()
	s, err := combat.NewObjectStatus(content.SizeLarge, 17, combat.MustHitPoints(5, 27, 0), combat.DamageThreshold{}, aff, nil)
	if err != nil || s.Size() != content.SizeLarge || s.ArmorClass() != 17 || s.HitPoints() != combat.MustHitPoints(5, 27, 0) || s.Threshold().Amount() != 0 ||
		s.Affinities() != aff || s.AbilityScores() != (combat.NoAbilityScores{}) || s.IsZero() || s.IsDestroyed() {
		t.Errorf("= %+v, %v", s, err)
	}
	cases := map[string]struct {
		size   content.CreatureSize
		hp     combat.HitPoints
		scores combat.ObjectAbilityScores
	}{
		"temporary hit points": {content.SizeLarge, combat.MustHitPoints(5, 27, 1), nil}, "zero size": {0, combat.MustHitPoints(5, 27, 0), nil},
		"unknown scores": {content.SizeLarge, combat.MustHitPoints(5, 27, 0), badScores{}},
	}
	for name, c := range cases {
		if _, err := combat.NewObjectStatus(c.size, 10, c.hp, combat.DamageThreshold{}, aff, c.scores); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	if !(combat.ObjectStatus{}).IsZero() {
		t.Error("the zero status is not zero")
	}
	if dead := envObject(t, 0, 0, aff); !dead.IsDestroyed() {
		t.Error("an object at 0 Hit Points is not destroyed")
	}
}

type badScores struct{ combat.ObjectAbilityScores }

// Requirement: doc/specs/environment-hazards-and-objects.md FR-073, FR-074 - the standard affinities are Immunity to Poison and Psychic damage only, and a GM ruling adds to them without replacing them.
// Path: per case.
func TestObjectAffinities(t *testing.T) {
	t.Parallel()
	std := (combat.ObjectAffinities{}).Standard()
	if got := std.Immunities(); len(got) != 2 || got[0] != content.DamagePoison || got[1] != content.DamagePsychic || len(std.Resistances()) != 0 || len(std.Vulnerabilities()) != 0 {
		t.Errorf("Standard = immunities %v resistances %v vulnerabilities %v", std.Immunities(), std.Resistances(), std.Vulnerabilities())
	}
	ruling, _ := combat.NewDamageAffinities([]combat.DamageScope{combat.DamageOfType{Type: content.DamageCold}}, []content.DamageType{content.DamageFire}, []content.DamageType{content.DamageAcid})
	poisonToo, _ := combat.NewDamageAffinities(nil, nil, []content.DamageType{content.DamagePoison})
	if i := (combat.ObjectAffinities{}).StandardWith(poisonToo).Immunities(); len(i) != 2 || i[0] != content.DamagePoison || i[1] != content.DamagePsychic {
		t.Errorf("a ruling that repeats a standard Immunity gives %v, want Poison and Psychic kept", i)
	}
	got := (combat.ObjectAffinities{}).StandardWith(ruling)
	if i := got.Immunities(); len(i) != 3 || i[0] != content.DamageAcid || i[1] != content.DamagePoison || i[2] != content.DamagePsychic ||
		len(got.Vulnerabilities()) != 1 || got.Vulnerabilities()[0] != content.DamageFire || len(got.Resistances()) != 1 {
		t.Errorf("StandardWith = %+v", got)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-087, FR-088 (table 4.9d) - an object with no ability scores cannot make ability checks and fails every saving throw; one with a score for an ability can do both for it only.
// Path: per case.
func TestObjectAbilityChecksAndSaves(t *testing.T) {
	t.Parallel()
	none := envObject(t, 10, 0, combat.NoAffinities())
	for _, a := range ability.All() {
		if none.AbilityCheck(a) != combat.ConditionPermissionForbidden || none.SavingThrow(a) != combat.SavingThrowEffectAutomaticFailure {
			t.Errorf("NoAbilityScores %v: check %v save %v, want forbidden and automatic failure", a, none.AbilityCheck(a), none.SavingThrow(a))
		}
	}
	scores, err := combat.NewAssignedAbilityScores(map[ability.Ability]ability.Score{ability.Strength: ability.MustScore(20)})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := combat.NewObjectStatus(content.SizeMedium, 15, combat.MustHitPoints(10, 10, 0), combat.DamageThreshold{}, combat.NoAffinities(), scores)
	if s.AbilityCheck(ability.Strength) != combat.ConditionPermissionAllowed || s.SavingThrow(ability.Strength) != combat.SavingThrowEffectUnaffected ||
		s.AbilityCheck(ability.Dexterity) != combat.ConditionPermissionForbidden || s.SavingThrow(ability.Dexterity) != combat.SavingThrowEffectAutomaticFailure {
		t.Error("an assigned Strength did not allow Strength only")
	}
	if scores.Score(ability.Strength).Value() != 20 || !scores.Score(ability.Dexterity).IsZero() || !scores.Score(0).IsZero() || !scores.Score(7).IsZero() {
		t.Error("Score accessor is wrong")
	}
	if none.SavingThrow(0) != combat.SavingThrowEffectAutomaticFailure || none.AbilityCheck(9) != combat.ConditionPermissionForbidden {
		t.Error("an invalid ability is not forbidden / automatic failure")
	}
	empty, err := combat.NewAssignedAbilityScores(nil)
	if err != nil || empty != (combat.AssignedAbilityScores{}) {
		t.Errorf("no assigned scores = %v, %v", empty, err)
	}
	for name, m := range map[string]map[ability.Ability]ability.Score{"invalid ability": {0: ability.MustScore(10)}, "zero score": {ability.Strength: {}}} {
		if _, err := combat.NewAssignedAbilityScores(m); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	// A failed object saving throw is built by the caller with d20.FailWithoutRoll and d20.Automatic (FR-088).
	if _, err := d20.FailWithoutRoll(d20.SavingThrow, d20.Automatic); err != nil {
		t.Errorf("FailWithoutRoll = %v", err)
	}
}
