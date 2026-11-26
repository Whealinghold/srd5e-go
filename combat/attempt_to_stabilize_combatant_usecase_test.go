package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestAttemptToStabilizeCombatantChecks(t *testing.T) {
	t.Parallel()
	duo := func(helper []combat.Condition) combat.Encounter {
		e, _ := encStart(t, resPC(t, "a", combat.SideA, helper...), resDying(t, "b", combat.SideB, combat.Unconscious{}))
		return e
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		requires                combat.CheckRequirement
		faces                   []int
		dice                    []string
		events                  []string
		stabilized              bool
		forced                  bool
	}{
		{"12 against DC 10", resSpec + "FR-072, 4.11.1 DC 10", "happy", duo(nil), 0, []int{12}, []string{"1d20"}, []string{"action_taken", "became_stable"}, true, false},
		{"10 equals the DC", resSpec + "FR-072, 4.11.1 DC 10", "unhappy (boundary)", duo(nil), 0, []int{10}, []string{"1d20"}, []string{"action_taken", "became_stable"}, true, false},
		{"9 fails and leaves the action spent", resSpec + "FR-073, 4.11.1 DC 10", "unhappy (boundary)", duo(nil), 0, []int{9}, []string{"1d20"}, []string{"action_taken"}, false, false},
		{"Poisoned: Disadvantage, the lower die 6 fails", resSpec + "FR-073, 4.11.1 helper conditions", "unhappy (rule interaction)", duo([]combat.Condition{combat.Poisoned{}}), 0,
			[]int{15, 6}, []string{"2d20"}, []string{"action_taken"}, false, false},
		{"Exhaustion 2: 14 less 4 meets the DC", resSpec + "FR-073, FR-030, 4.11.1 helper conditions", "unhappy (boundary)", duo([]combat.Condition{exhaustion(2)}), 0,
			[]int{14}, []string{"1d20"}, []string{"action_taken", "became_stable"}, true, false},
		{"Exhaustion 2: 13 less 4 fails", resSpec + "FR-073, FR-030, 4.11.1 helper conditions", "unhappy (boundary)", duo([]combat.Condition{exhaustion(2)}), 0,
			[]int{13}, []string{"1d20"}, []string{"action_taken"}, false, false},
		{"Blinded and sight required: automatic failure", resSpec + "FR-073, 4.11.1 requires", "unhappy (rule interaction)", duo([]combat.Condition{combat.Blinded{}}), combat.CheckRequirementSight,
			nil, nil, []string{"action_taken"}, false, true},
		{"Deafened and hearing required: automatic failure", resSpec + "FR-073, 4.11.1 requires", "unhappy (rule interaction)", duo([]combat.Condition{combat.Deafened{}}), combat.CheckRequirementHearing,
			nil, nil, []string{"action_taken"}, false, true},
		{"Blinded and nothing required: rolled", resSpec + "FR-073, Q-50, 4.11.1 requires", "unhappy (rule interaction)", duo([]combat.Condition{combat.Blinded{}}), 0,
			[]int{12}, []string{"1d20"}, []string{"action_taken", "became_stable"}, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resStabilizeParams(c.e)
			p.Requires = c.requires
			res, dice, err := resStabilize(t, p, c.faces...)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			got := resAs[combat.AttemptToStabilizeCombatantCompleted](t, res)
			_, isForced := got.Check().(combat.TestForcedFailure)
			if !slices.Equal(dice, c.dice) || !slices.Equal(resKinds(got.Events()), c.events) || got.Stabilized() != c.stabilized || isForced != c.forced {
				t.Errorf("dice %v, events %v, stabilized %v, forced %v; want %v, %v, %v, %v [%s]", dice, resKinds(got.Events()), got.Stabilized(), isForced, c.dice, c.events, c.stabilized, c.forced, c.requirement)
			}
			life := encLife(encMember(t, got.Encounter(), "b"))
			if (life == "stable") != c.stabilized || (!c.stabilized && life != "dying") {
				t.Errorf("patient is %s, want stable=%v and otherwise unchanged [FR-073]", life, c.stabilized)
			}
			if !encMember(t, got.Encounter(), "b").Conditions().Equal(encMember(t, c.e, "b").Conditions()) {
				t.Errorf("the patient stays Unconscious [FR-070]")
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-010 steps 2, 3, 3a, 5, FR-072, FR-074, 4.11.1 stabilize stops - the stops come before the action is spent and any die: a patient that is Stable, dead, unknown or knocked out at 1 Hit Point, a helper that is unknown or dead or Incapacitated, a helper that is its own patient; with the helper before the patient within each step and the patient's NotDying before the helper's tracker refusal.
// Path: per case.
func TestAttemptToStabilizeCombatantStops(t *testing.T) {
	t.Parallel()
	stable := encC(t, "b", combat.SideB, status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), combat.Unconscious{})
	knockedOut := encC(t, "b", combat.SideB, pcStatus(t, 1, 20, 0), combat.Unconscious{})
	dyingB := resDying(t, "b", combat.SideB, combat.Unconscious{})
	cases := []struct {
		name, requirement string
		e                 combat.Encounter
		edit              func(*combat.AttemptToStabilizeCombatantParams)
		rejection         combat.ResolutionRejection
		problem           combat.ResolutionProblem
	}{
		{"patient Stable", resSpec + "FR-010 3a, 4.11.1 stabilize stops", resDuoOf(t, resPC(t, "a", combat.SideA), stable), nil,
			combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid("b"), Reason: combat.NotDying}}, nil},
		{"patient knocked out at 1 Hit Point", resSpec + "FR-072, Q-41", resDuoOf(t, resPC(t, "a", combat.SideA), knockedOut), nil,
			combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid("b"), Reason: combat.NotDying}}, nil},
		{"patient dead", resSpec + "FR-010 3, 4.11.1 stabilize stops", resDuoOf(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB)), nil,
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil},
		{"unknown patient", resSpec + "FR-010 2, 4.11.1 stabilize stops", resDuoOf(t, resPC(t, "a", combat.SideA), dyingB), func(p *combat.AttemptToStabilizeCombatantParams) { p.PatientID = cid("ghost") },
			nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"unknown helper", resSpec + "FR-010 2, 4.11.1 stabilize stops", resDuoOf(t, resPC(t, "a", combat.SideA), dyingB), func(p *combat.AttemptToStabilizeCombatantParams) { p.HelperID = cid("ghost") },
			nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"dead helper and unknown patient: unknown first", resSpec + "FR-010 2 before 3, 4.11.1 stabilize stops", resDuoOf(t, resDead(t, "a", combat.SideA), dyingB),
			func(p *combat.AttemptToStabilizeCombatantParams) { p.PatientID = cid("ghost") }, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"dead helper", resSpec + "FR-010 3", resDuoOf(t, resDead(t, "a", combat.SideA), dyingB), nil, combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("a")}}, nil},
		{"Incapacitated helper and Stable patient: NotDying first", resSpec + "FR-010 3a before 5, 4.11.1 stabilize stops", resDuoOf(t, resPC(t, "a", combat.SideA, combat.Incapacitated{}), stable), nil,
			combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid("b"), Reason: combat.NotDying}}, nil},
		{"Incapacitated helper", resSpec + "FR-055, 4.11.1 helper conditions", resDuoOf(t, resPC(t, "a", combat.SideA, combat.Incapacitated{}), dyingB), nil,
			combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}, nil},
		{"the helper is the patient and not Dying", resSpec + "FR-074", encABC(t), func(p *combat.AttemptToStabilizeCombatantParams) { p.PatientID = cid("a") },
			combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid("a"), Reason: combat.NotDying}}, nil},
		{"the helper is the patient and Dying: Unconscious cannot help itself", resSpec + "FR-074, FR-055", resDuoOf(t, resDying(t, "a", combat.SideA, combat.Unconscious{}), resPC(t, "b", combat.SideB)),
			func(p *combat.AttemptToStabilizeCombatantParams) { p.PatientID = cid("a") }, combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}, nil},
		{"the helper is not the active combatant", resSpec + "FR-072", resThree(t, resPC(t, "c", combat.SideA), resPC(t, "a", combat.SideA), dyingB), nil, nil, nil},
		{"ended", resSpec + "FR-010 1", resEnded(t, resDuoOf(t, resPC(t, "a", combat.SideA), dyingB)), nil, combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resStabilizeParams(c.e)
			if c.edit != nil {
				c.edit(&p)
			}
			res, dice, err := resStabilize(t, p, 15)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if c.problem != nil {
				inv := resAs[combat.AttemptToStabilizeCombatantInvalid](t, res)
				if inv.Problem() != c.problem || !inv.Encounter().Equal(c.e) || len(dice) != 0 {
					t.Errorf("problem %#v, dice %v, want %#v and no die [%s]", inv.Problem(), dice, c.problem, c.requirement)
				}
				return
			}
			if c.rejection == nil { // the helper is not the active combatant: the tracker names it
				rej := resAs[combat.AttemptToStabilizeCombatantRejected](t, res)
				if _, ok := rej.Rejection().(combat.TrackerRejection); !ok || len(dice) != 0 {
					t.Errorf("rejection %#v, dice %v, want the tracker's [%s]", rej.Rejection(), dice, c.requirement)
				}
				return
			}
			rej := resAs[combat.AttemptToStabilizeCombatantRejected](t, res)
			if rej.Rejection() != c.rejection || !rej.Encounter().Equal(c.e) || len(dice) != 0 || len(rej.Events()) != 0 {
				t.Errorf("rejection %#v, dice %v, want %#v with nothing changed and no die [%s]", rej.Rejection(), dice, c.rejection, c.requirement)
			}
		})
	}
}

// resDuoOf starts an encounter of the two combatants, the first active.
func resDuoOf(tb testing.TB, a, b combat.Combatant) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, a, b)
	return e
}

// Requirement: doc/specs/combat-action-resolution.md FR-072, FR-117, 4.18.1 Help - a pending Help benefit of the helper for a Wisdom (Medicine) check gives Advantage, is spent by the attempt (HelpBenefitUsed follows the tracker's events) and is not spent by a forced failure, which rolls no die.
// Path: unhappy (rule interaction).
func TestAttemptToStabilizeCombatantConsumesHelp(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "c", combat.SideA), resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB, combat.Unconscious{}))
	with, _ := combat.NewHelpGiverProficiencies([]content.Skill{content.SkillMedicine}, nil)
	helped := resTake(t, e, combat.ActionLedger{}, "c", combat.TakeHelpAction{Request: combat.AbilityCheckHelpRequest{Ally: cid("a"), Proficiency: combat.AssistedSkill(content.SkillMedicine),
		HelperProficiencies: with, Judgement: combat.AssistanceJudgementPossible}})
	next := resAdvance(t, helped.Encounter(), helped.Ledger())
	p := resStabilizeParams(next.Encounter())
	p.Ledger = next.Ledger()
	res, dice, err := resStabilize(t, p, 3, 12)
	got := resAs[combat.AttemptToStabilizeCombatantCompleted](t, res)
	if want := []string{"action_taken", "help_benefit_used", "became_stable"}; err != nil || !slices.Equal(dice, []string{"2d20"}) || !slices.Equal(resKinds(got.Events()), want) || got.Ledger().HelpBenefits().Len() != 0 {
		t.Errorf("dice %v, events %v, benefits %d, want 2d20, %v and none left: %v", dice, resKinds(got.Events()), got.Ledger().HelpBenefits().Len(), want, err)
	}
	forced := resStabilizeParams(next.Encounter())
	forced.Ledger, forced.Check = next.Ledger(), d20.AutomaticFailure{}
	res, dice, err = resStabilize(t, forced)
	if kept := resAs[combat.AttemptToStabilizeCombatantCompleted](t, res); err != nil || len(dice) != 0 || kept.Ledger().HelpBenefits().Len() != 1 {
		t.Errorf("a forced failure: dice %v, benefits %d, want no die and the benefit kept: %v", dice, kept.Ledger().HelpBenefits().Len(), err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-096 - the command refuses a zero encounter, a blank id, a check that is neither a RollAttempt nor an AutomaticFailure and a requirement that is not defined.
// Path: unhappy (invalid input).
func TestNewAttemptToStabilizeCombatantCommand(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	for name, edit := range map[string]func(*combat.AttemptToStabilizeCombatantParams){
		"zero encounter":        func(p *combat.AttemptToStabilizeCombatantParams) { p.Encounter = combat.Encounter{} },
		"blank helper":          func(p *combat.AttemptToStabilizeCombatantParams) { p.HelperID = "" },
		"blank patient":         func(p *combat.AttemptToStabilizeCombatantParams) { p.PatientID = "" },
		"nil check":             func(p *combat.AttemptToStabilizeCombatantParams) { p.Check = nil },
		"undefined requirement": func(p *combat.AttemptToStabilizeCombatantParams) { p.Requires = combat.CheckRequirement(9) },
	} {
		p := resStabilizeParams(e)
		edit(&p)
		if _, err := combat.NewAttemptToStabilizeCombatantCommand(p); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	p := resStabilizeParams(e)
	p.Check = d20.AutomaticFailure{}
	if _, err := combat.NewAttemptToStabilizeCombatantCommand(p); err != nil {
		t.Errorf("an automatic failure is a valid check: %v", err)
	}
}

// resThree starts an encounter of the three combatants, the first active.
func resThree(tb testing.TB, a, b, c combat.Combatant) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, a, b, c)
	return e
}
