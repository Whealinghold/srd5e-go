package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func exhaustion(n int) combat.Condition {
	return combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(n)}
}

// Requirement: per case
// Path: per case.
func TestAttackConditionInputsAttack(t *testing.T) {
	t.Parallel()
	a, tg := cid("a"), cid("t")
	canSee, cannotSee := combat.SightRelationCanSee, combat.SightRelationCannotSee
	scene := func(feet int, aSees, tSees combat.SightRelation, fear combat.LineOfSight) combat.AttackScene {
		s, err := combat.NewAttackScene(feet, aSees, tSees, fear)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	seen := scene(5, canSee, canSee, combat.LineOfSightBlocked)
	cases := []struct {
		name, requirement, path string
		attacker, target        []combat.Condition
		scene                   combat.AttackScene
		dodge                   combat.DodgeBenefits
		targetWalk              int
		circumstances           []modifier.RollCircumstance
		modifiers               []modifier.Modifier
		critical                combat.AutomaticCritical
		forbidden               bool
	}{
		{"none", resSpec + "4.5.1 none", "happy", nil, nil, seen, combat.DodgeBenefitsNone, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"attacker Blinded", resSpec + "4.5.1", "happy", []combat.Condition{combat.Blinded{}}, nil, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"target Blinded", resSpec + "4.5.1", "happy", nil, []combat.Condition{combat.Blinded{}}, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"target Restrained", resSpec + "4.5.1", "happy", nil, []combat.Condition{combat.Restrained{}}, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"attacker Restrained", resSpec + "4.5.1", "happy", []combat.Condition{combat.Restrained{}}, nil, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"attacker Invisible, unseen", resSpec + "4.5.1", "happy", []combat.Condition{combat.Invisible{}}, nil, scene(5, canSee, cannotSee, combat.LineOfSightBlocked),
			combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"target Invisible, unseen", resSpec + "4.5.1", "happy", nil, []combat.Condition{combat.Invisible{}}, scene(5, cannotSee, canSee, combat.LineOfSightBlocked),
			combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"attacker Poisoned", resSpec + "4.5.1", "happy", []combat.Condition{combat.Poisoned{}}, nil, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"attacker Prone", resSpec + "4.5.1", "happy", []combat.Condition{combat.Prone{}}, nil, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"target Prone at 5 feet", resSpec + "4.5.1 target Prone", "unhappy (boundary)", nil, []combat.Condition{combat.Prone{}}, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"target Prone at 6 feet", resSpec + "4.5.1 target Prone", "unhappy (boundary)", nil, []combat.Condition{combat.Prone{}}, scene(6, canSee, canSee, combat.LineOfSightBlocked),
			combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Advantage and Disadvantage both passed, in condition-kind order", resSpec + "4.5.1, FR-027", "unhappy (rule interaction)", []combat.Condition{combat.Poisoned{}},
			[]combat.Condition{combat.Blinded{}}, seen, combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{advantage, disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"dodging, attacker seen", resSpec + "FR-029, 4.5.1 dodging", "unhappy (rule interaction)", nil, nil, seen, combat.DodgeBenefitsActive, 30,
			[]modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"dodging, attacker not seen", resSpec + "FR-029, 4.5.1 dodging", "unhappy (rule interaction)", nil, nil, scene(5, canSee, cannotSee, combat.LineOfSightBlocked),
			combat.DodgeBenefitsActive, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Dodge benefits none", resSpec + "FR-029, 4.5.1 dodging", "unhappy (rule interaction)", nil, nil, seen, combat.DodgeBenefitsNone, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Grappled with a stale profile Speed 30 cancels Dodge", resSpec + "FR-125, 4.5.1 dodging", "unhappy (rule interaction)", nil,
			[]combat.Condition{combat.Grappled{Grappler: cid("g")}}, seen, combat.DodgeBenefitsActive, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Restrained and dodging leaves only Advantage", resSpec + "FR-125, 4.5.1 dodging", "unhappy (rule interaction)", nil, []combat.Condition{combat.Restrained{}}, seen,
			combat.DodgeBenefitsActive, 30, []modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Grappled attacker, target is the grappler", resSpec + "4.5.1 Grappled attacker", "unhappy (boundary)", []combat.Condition{combat.Grappled{Grappler: tg}}, nil, seen,
			combat.DodgeBenefitsNone, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Grappled attacker, another target", resSpec + "4.5.1 Grappled attacker", "unhappy (boundary)", []combat.Condition{combat.Grappled{Grappler: cid("g")}}, nil, seen,
			combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Frightened, source in sight", resSpec + "FR-034, 4.5.1", "unhappy (boundary)", []combat.Condition{combat.Frightened{Source: cid("s")}}, nil,
			scene(5, canSee, canSee, combat.LineOfSightClear), combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{disadvantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Frightened, source out of sight", resSpec + "FR-034, 4.5.1", "unhappy (boundary)", []combat.Condition{combat.Frightened{Source: cid("s")}}, nil,
			scene(5, canSee, canSee, combat.LineOfSightBlocked), combat.DodgeBenefitsNone, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Exhaustion 3", resSpec + "FR-030, 4.5.1 Exhaustion", "unhappy (boundary)", []combat.Condition{exhaustion(3)}, nil, seen, combat.DodgeBenefitsNone, 30,
			nil, []modifier.Modifier{modifier.New(-6)}, combat.AutomaticCriticalNotAutomatic, false},
		{"Paralyzed target at 5 feet", resSpec + "FR-031, 4.5.1 critical", "unhappy (boundary)", nil, []combat.Condition{combat.Paralyzed{}}, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalAutomatic, false},
		{"Paralyzed target at 10 feet", resSpec + "FR-031, 4.5.1 critical", "unhappy (boundary)", nil, []combat.Condition{combat.Paralyzed{}},
			scene(10, canSee, canSee, combat.LineOfSightBlocked), combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Unconscious target at 5 feet (its Prone within 5 feet adds a second Advantage, harmless by FR-028)", resSpec + "FR-031, FR-028, 4.5.1 critical", "unhappy (boundary)", nil, []combat.Condition{combat.Unconscious{}}, seen, combat.DodgeBenefitsNone, 30,
			[]modifier.RollCircumstance{advantage, advantage}, nil, combat.AutomaticCriticalAutomatic, false},
		{"Petrified target at 10 feet", resSpec + "FR-031, 4.5.1 critical", "unhappy (boundary)", nil, []combat.Condition{combat.Petrified{}},
			scene(10, canSee, canSee, combat.LineOfSightBlocked), combat.DodgeBenefitsNone, 30, []modifier.RollCircumstance{advantage}, nil, combat.AutomaticCriticalNotAutomatic, false},
		{"Charmed by the target", resSpec + "FR-033, 4.5.1", "unhappy (rule interaction)", []combat.Condition{combat.Charmed{Charmer: combat.CharmedByCombatant{ID: tg}}}, nil, seen,
			combat.DodgeBenefitsNone, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, true},
		{"Charmed by another", resSpec + "FR-033, 4.5.1", "unhappy (rule interaction)", []combat.Condition{combat.Charmed{Charmer: combat.CharmedByCombatant{ID: cid("x")}}}, nil, seen,
			combat.DodgeBenefitsNone, 30, nil, nil, combat.AutomaticCriticalNotAutomatic, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.AttackConditionInputs{}).Attack(condSet(t, c.attacker), condSet(t, c.target), a, tg, c.scene, c.dodge, c.targetWalk)
			if err != nil {
				t.Fatalf("Attack: %v [%s]", err, c.requirement)
			}
			if !slices.Equal(got.Circumstances(), c.circumstances) || !slices.Equal(got.Modifiers(), c.modifiers) ||
				got.AutomaticCritical() != c.critical || got.TargetingForbidden() != c.forbidden {
				t.Errorf("circumstances %v, modifiers %v, critical %v, forbidden %v; want %v, %v, %v, %v [%s]", got.Circumstances(), got.Modifiers(),
					got.AutomaticCritical(), got.TargetingForbidden(), c.circumstances, c.modifiers, c.critical, c.forbidden, c.requirement)
			}
			add := got.Additions()
			if !slices.Equal(add.Circumstances, c.circumstances) || add.AutomaticCritical != c.critical {
				t.Errorf("Additions = %+v, want the same inputs [FR-032]", add)
			}
		})
	}
	if _, err := (combat.AttackConditionInputs{}).Attack(condSet(t, nil), condSet(t, nil), a, tg, seen, combat.DodgeBenefitsActive, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative Speed gives %v, want ErrInvalidInput [FR-096]", err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-034 - changing only the fear source of the scene from a source in sight to one that is not changes the Frightened
// contribution and nothing else; the results are copies a caller cannot use to alter the inputs (FR-004).
// Path: unhappy (immutability).
func TestAttackConditionInputsResultsAreCopies(t *testing.T) {
	t.Parallel()
	in, err := (combat.AttackConditionInputs{}).Attack(condSet(t, []combat.Condition{combat.Poisoned{}, exhaustion(1)}), condSet(t, nil), cid("a"), cid("t"), resSeen(t, 5), combat.DodgeBenefitsNone, 30)
	if err != nil {
		t.Fatal(err)
	}
	in.Circumstances()[0] = advantage
	in.Modifiers()[0] = modifier.New(9)
	in.Additions().Circumstances[0] = advantage
	if got := in.Circumstances(); !slices.Equal(got, []modifier.RollCircumstance{disadvantage}) || !slices.Equal(in.Modifiers(), []modifier.Modifier{modifier.New(-2)}) {
		t.Errorf("inputs = %v, %v, want them unchanged by a caller's edit of a returned slice", got, in.Modifiers())
	}
}

// Requirement: per case
// Path: per case.
func TestAttackConditionInputsSave(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		target                  []combat.Condition
		save                    ability.Ability
		dodge                   combat.DodgeBenefits
		walk                    int
		forced                  bool
		circumstances           []modifier.RollCircumstance
		modifiers               []modifier.Modifier
	}{
		{"no conditions", resSpec + "FR-045", "happy", nil, ability.Strength, combat.DodgeBenefitsNone, 30, false, nil, nil},
		{"Restrained, Dexterity save", resSpec + "FR-045, 4.7.1", "unhappy (rule interaction)", []combat.Condition{combat.Restrained{}}, ability.Dexterity, combat.DodgeBenefitsNone, 30, false,
			[]modifier.RollCircumstance{disadvantage}, nil},
		{"Restrained, Strength save", resSpec + "FR-045", "unhappy (rule interaction)", []combat.Condition{combat.Restrained{}}, ability.Strength, combat.DodgeBenefitsNone, 30, false, nil, nil},
		{"dodging, Dexterity save", resSpec + "FR-029, 4.7.1", "unhappy (rule interaction)", nil, ability.Dexterity, combat.DodgeBenefitsActive, 30, false,
			[]modifier.RollCircumstance{advantage}, nil},
		{"dodging, Strength save", resSpec + "FR-029, 4.7.1", "unhappy (rule interaction)", nil, ability.Strength, combat.DodgeBenefitsActive, 30, false, nil, nil},
		{"Grappled and dodging, stale profile Speed 30", resSpec + "FR-125, 4.7.1", "unhappy (rule interaction)", []combat.Condition{combat.Grappled{Grappler: cid("g")}}, ability.Dexterity,
			combat.DodgeBenefitsActive, 30, false, nil, nil},
		{"Paralyzed, Strength save", resSpec + "FR-045, 4.7.1 forced", "unhappy (rule interaction)", []combat.Condition{combat.Paralyzed{}}, ability.Strength, combat.DodgeBenefitsNone, 30, true, nil, nil},
		{"Paralyzed, Dexterity save", resSpec + "FR-045, 4.7.1 forced", "unhappy (rule interaction)", []combat.Condition{combat.Paralyzed{}}, ability.Dexterity, combat.DodgeBenefitsNone, 30, true, nil, nil},
		{"Paralyzed, Wisdom save", resSpec + "FR-045", "unhappy (boundary)", []combat.Condition{combat.Paralyzed{}}, ability.Wisdom, combat.DodgeBenefitsNone, 30, false, nil, nil},
		{"Exhaustion 2", resSpec + "FR-030, 4.7.1", "unhappy (boundary)", []combat.Condition{exhaustion(2)}, ability.Strength, combat.DodgeBenefitsNone, 30, false, nil, []modifier.Modifier{modifier.New(-4)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.AttackConditionInputs{}).Save(condSet(t, c.target), c.save, c.dodge, c.walk)
			if err != nil {
				t.Fatalf("Save: %v [%s]", err, c.requirement)
			}
			reason, forced := got.Forced()
			if forced != c.forced || (forced && reason != d20.Automatic) || !slices.Equal(got.Circumstances(), c.circumstances) || !slices.Equal(got.Modifiers(), c.modifiers) {
				t.Errorf("forced %v (%v), circumstances %v, modifiers %v; want %v, %v, %v [%s]", forced, reason, got.Circumstances(), got.Modifiers(), c.forced, c.circumstances, c.modifiers, c.requirement)
			}
		})
	}
	if _, err := (combat.AttackConditionInputs{}).Save(condSet(t, nil), ability.Dexterity, combat.DodgeBenefitsActive, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative Speed gives %v, want ErrInvalidInput [FR-096]", err)
	}
}

// Requirement: per case
// Path: per case.
func TestAttackConditionInputsCheck(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		helper                  []combat.Condition
		fear                    combat.LineOfSight
		requires                combat.CheckRequirement
		forced                  bool
		circumstances           []modifier.RollCircumstance
		modifiers               []modifier.Modifier
	}{
		{"no conditions", resSpec + "FR-073", "happy", nil, combat.LineOfSightBlocked, combat.CheckRequirementNone, false, nil, nil},
		{"Poisoned", resSpec + "FR-073, 4.11.1", "unhappy (rule interaction)", []combat.Condition{combat.Poisoned{}}, combat.LineOfSightBlocked, combat.CheckRequirementNone, false,
			[]modifier.RollCircumstance{disadvantage}, nil},
		{"Exhaustion 2", resSpec + "FR-073, FR-030, 4.11.1", "unhappy (boundary)", []combat.Condition{exhaustion(2)}, combat.LineOfSightBlocked, combat.CheckRequirementNone, false,
			nil, []modifier.Modifier{modifier.New(-4)}},
		{"Frightened, source in sight", resSpec + "FR-073", "unhappy (rule interaction)", []combat.Condition{combat.Frightened{Source: cid("s")}}, combat.LineOfSightClear, combat.CheckRequirementNone, false,
			[]modifier.RollCircumstance{disadvantage}, nil},
		{"Blinded, sight required", resSpec + "FR-073, 4.11.1 requires", "unhappy (rule interaction)", []combat.Condition{combat.Blinded{}}, combat.LineOfSightBlocked, combat.CheckRequirementSight, true, nil, nil},
		{"Deafened, hearing required", resSpec + "FR-073, 4.11.1 requires", "unhappy (rule interaction)", []combat.Condition{combat.Deafened{}}, combat.LineOfSightBlocked, combat.CheckRequirementHearing, true, nil, nil},
		{"Blinded, nothing required", resSpec + "FR-073, Q-50", "unhappy (rule interaction)", []combat.Condition{combat.Blinded{}}, combat.LineOfSightBlocked, combat.CheckRequirementNone, false, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := (combat.AttackConditionInputs{}).Check(condSet(t, c.helper), c.fear, c.requires)
			_, forced := got.Forced()
			if forced != c.forced || !slices.Equal(got.Circumstances(), c.circumstances) || !slices.Equal(got.Modifiers(), c.modifiers) {
				t.Errorf("forced %v, circumstances %v, modifiers %v; want %v, %v, %v [%s]", forced, got.Circumstances(), got.Modifiers(), c.forced, c.circumstances, c.modifiers, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-069, FR-030 - the Exhaustion reduction of a Death Saving Throw is 2 per level, 0 without Exhaustion.
// Path: unhappy (boundary).
func TestAttackConditionInputsExhaustionReduction(t *testing.T) {
	t.Parallel()
	for level, want := range map[int]int{0: 0, 1: 2, 2: 4, 5: 10} {
		var conds []combat.Condition
		if level > 0 {
			conds = append(conds, exhaustion(level))
		}
		if got := (combat.AttackConditionInputs{}).ExhaustionReduction(condSet(t, conds)); got != want {
			t.Errorf("Exhaustion %d reduces by %d, want %d", level, got, want)
		}
	}
}
