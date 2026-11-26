package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestRollCombatantDeathSavingThrowResults(t *testing.T) {
	t.Parallel()
	type tally struct{ successes, failures int }
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		circumstances           []modifier.RollCircumstance
		modifiers               []modifier.Modifier
		faces                   []int
		dice                    []string
		natural, total          int
		events                  []string
		life                    string
		want                    tally
	}{
		{"12 is a success", resSpec + "FR-069, 4.11.1 faces", "happy", resDyingEncounter(t, 0, 0), nil, nil, []int{12}, []string{"1d20"}, 12, 12,
			[]string{"death_saving_throw_made"}, "dying", tally{1, 0}},
		{"10 is a success", resSpec + "FR-069, 4.11.1 faces", "unhappy (boundary)", resDyingEncounter(t, 0, 0), nil, nil, []int{10}, []string{"1d20"}, 10, 10,
			[]string{"death_saving_throw_made"}, "dying", tally{1, 0}},
		{"9 is a failure", resSpec + "FR-069, 4.11.1 faces", "unhappy (boundary)", resDyingEncounter(t, 0, 0), nil, nil, []int{9}, []string{"1d20"}, 9, 9,
			[]string{"death_saving_throw_made"}, "dying", tally{0, 1}},
		{"a natural 1 is two failures and kills a creature with one", resSpec + "FR-070, 4.11.1 boundary", "unhappy (boundary)", resDyingEncounter(t, 0, 1), nil, nil, []int{1}, []string{"1d20"}, 1, 1,
			[]string{"death_saving_throw_made", "died"}, "dead:death_saving_throw_failures", tally{}},
		{"a natural 20 regains 1 Hit Point and removes Unconscious", resSpec + "FR-070, 4.11.1 boundary", "unhappy (boundary)", resDyingEncounter(t, 0, 0, combat.Unconscious{}), nil, nil, []int{20}, []string{"1d20"}, 20, 20,
			[]string{"death_saving_throw_made", "hit_points_restored", "recovered", "condition_removed", "movement_speeds_review_became_due"}, "alive", tally{}},
		{"the third success makes the creature Stable and keeps Unconscious", resSpec + "FR-070, 4.11.1 boundary", "unhappy (boundary)", resDyingEncounter(t, 2, 0, combat.Unconscious{}), nil, nil, []int{14}, []string{"1d20"}, 14, 14,
			[]string{"death_saving_throw_made", "became_stable"}, "stable", tally{}},
		{"Exhaustion 2 lowers 12 to 8: a failure", resSpec + "FR-069, FR-030, 4.11.1 Exhaustion", "unhappy (boundary)", resDyingEncounter(t, 0, 0, exhaustion(2)), nil, nil, []int{12}, []string{"1d20"}, 12, 8,
			[]string{"death_saving_throw_made"}, "dying", tally{0, 1}},
		{"Exhaustion 2 and 14 is a success at 10", resSpec + "FR-069, 4.11.1 Exhaustion", "unhappy (boundary)", resDyingEncounter(t, 0, 0, exhaustion(2)), nil, nil, []int{14}, []string{"1d20"}, 14, 10,
			[]string{"death_saving_throw_made"}, "dying", tally{1, 0}},
		{"Exhaustion 2 and a natural 20 still regains 1 Hit Point", resSpec + "FR-069, 4.11.1 Exhaustion", "unhappy (boundary)", resDyingEncounter(t, 0, 0, exhaustion(2), combat.Unconscious{}), nil, nil, []int{20}, []string{"1d20"}, 20, 16,
			[]string{"death_saving_throw_made", "hit_points_restored", "recovered", "condition_removed", "movement_speeds_review_became_due"}, "alive", tally{}},
		{"Advantage keeps the higher die", resSpec + "FR-069, 4.11.1 circumstances", "unhappy (rule interaction)", resDyingEncounter(t, 0, 0), []modifier.RollCircumstance{advantage}, nil, []int{4, 12}, []string{"2d20"}, 12, 12,
			[]string{"death_saving_throw_made"}, "dying", tally{1, 0}},
		{"Disadvantage keeps the lower die", resSpec + "FR-069", "unhappy (rule interaction)", resDyingEncounter(t, 0, 0), []modifier.RollCircumstance{disadvantage}, nil, []int{4, 12}, []string{"2d20"}, 4, 4,
			[]string{"death_saving_throw_made"}, "dying", tally{0, 1}},
		{"Advantage and Disadvantage cancel", resSpec + "FR-069", "unhappy (rule interaction)", resDyingEncounter(t, 0, 0), []modifier.RollCircumstance{advantage, disadvantage}, nil, []int{12}, []string{"1d20"}, 12, 12,
			[]string{"death_saving_throw_made"}, "dying", tally{1, 0}},
		{"the host's modifier is in the total and not in the dice request", resSpec + "FR-069, FR-160", "unhappy (rule interaction)", resDyingEncounter(t, 0, 0), nil, []modifier.Modifier{modifier.New(2)}, []int{8}, []string{"1d20"}, 8, 10,
			[]string{"death_saving_throw_made"}, "dying", tally{1, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, dice, err := resThrow(t, c.e, "a", c.circumstances, c.modifiers, c.faces...)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			got := resAs[combat.RollCombatantDeathSavingThrowCompleted](t, res)
			if !slices.Equal(dice, c.dice) || got.Natural() != c.natural || got.Total() != c.total || !slices.Equal(resKinds(got.Events()), c.events) || encLife(encMember(t, got.Encounter(), "a")) != c.life {
				t.Errorf("dice %v, natural %d, total %d, events %v, life %s; want %v, %d, %d, %v, %s [%s]", dice, got.Natural(), got.Total(), resKinds(got.Events()), encLife(encMember(t, got.Encounter(), "a")),
					c.dice, c.natural, c.total, c.events, c.life, c.requirement)
			}
			if d, ok := encMember(t, got.Encounter(), "a").HitPoints().LifeState().(combat.Dying); ok && (d.Successes() != c.want.successes || d.Failures() != c.want.failures) {
				t.Errorf("tallies %d and %d, want %d and %d [%s]", d.Successes(), d.Failures(), c.want.successes, c.want.failures, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-010 steps 3, 3a, 5a, 5b, FR-068, 4.11.1 stops - a throw is refused before any die: not Dying, not the active combatant, dead, a second throw in the same turn; with the order of the spec (dead before not Dying before out of turn).
// Path: per case.
func TestRollCombatantDeathSavingThrowStops(t *testing.T) {
	t.Parallel()
	healthy := encABC(t)
	dyingSecond, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB))
	deadSecond, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	thrown := func() combat.Encounter {
		res, _, err := resThrow(t, resDyingEncounter(t, 0, 0), "a", nil, nil, 12)
		if err != nil {
			t.Fatal(err)
		}
		return resAs[combat.RollCombatantDeathSavingThrowCompleted](t, res).Encounter()
	}()
	cases := []struct {
		name, requirement string
		e                 combat.Encounter
		who               string
		want              combat.ResolutionRejection
	}{
		{"not Dying", resSpec + "FR-010 3a, 4.11.1 stops", healthy, "a", combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid("a"), Reason: combat.NotDying}}},
		{"not active", resSpec + "FR-010 5a, 4.11.1 stops", dyingSecond, "b", combat.DeathSavingThrowOutOfTurnRejection{}},
		{"dead", resSpec + "FR-010 3, 4.11.1 stops", deadSecond, "b", combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}},
		{"not Dying and not active: 3a before 5a", resSpec + "FR-010, 4.11.1 stops", healthy, "b", combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid("b"), Reason: combat.NotDying}}},
		{"a second throw in the same turn", resSpec + "FR-010 5b, FR-068, 4.11.1 once per turn", thrown, "a", combat.EncounterStateRejection{Rejection: combat.DeathSavingThrowAlreadyMadeRejection{ID: cid("a")}}},
		{"ended", resSpec + "FR-010 1", resEnded(t, healthy), "a", combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewRollCombatantDeathSavingThrowCommand(c.e, cid(c.who), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			roller := newFixed(12)
			uc, _ := combat.NewRollCombatantDeathSavingThrow(roller)
			res, err := uc.Run(ctxBackground(), cmd)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			resWant(t, resAs[combat.RollCombatantDeathSavingThrowRejected](t, res), c.e, combat.ActionLedger{}, c.want, roller)
		})
	}
	res, _, err := resThrow(t, healthy, "ghost", nil, nil, 12)
	if inv := resAs[combat.RollCombatantDeathSavingThrowInvalid](t, res); err != nil || inv.Problem() != (combat.UnknownCombatantProblem{ID: cid("ghost")}) {
		t.Errorf("= %#v, %v, want an invalid result naming ghost [FR-011]", res, err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-068, FR-091 - a throw that wakes a Dying creature (a natural 20) prompts a review of its Speed after the encounter's events: Unconscious left.
// Path: unhappy (rule interaction).
func TestRollCombatantDeathSavingThrowPromptsASpeedReview(t *testing.T) {
	t.Parallel()
	res, _, err := resThrow(t, resDyingEncounter(t, 0, 0, combat.Unconscious{}), "a", nil, nil, 20)
	got := resAs[combat.RollCombatantDeathSavingThrowCompleted](t, res)
	last := got.Events()[len(got.Events())-1]
	if _, ok := last.(combat.MovementSpeedsReviewBecameDue); !ok || err != nil || resHas(t, got.Encounter(), "a", content.ConditionUnconscious) {
		t.Errorf("events %v, %v, want the review of Speed last and Unconscious gone", resKinds(got.Events()), err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-096 - the command refuses a zero encounter, a blank id and a nil circumstance, and takes its own copy of the lists.
// Path: unhappy (invalid input).
func TestNewRollCombatantDeathSavingThrowCommand(t *testing.T) {
	t.Parallel()
	e := resDyingEncounter(t, 0, 0)
	for name, c := range map[string]struct {
		e  combat.Encounter
		id combat.CombatantID
		cs []modifier.RollCircumstance
	}{
		"zero encounter":   {combat.Encounter{}, cid("a"), nil},
		"blank id":         {e, "", nil},
		"nil circumstance": {e, cid("a"), []modifier.RollCircumstance{nil}},
	} {
		if _, err := combat.NewRollCombatantDeathSavingThrowCommand(c.e, c.id, c.cs, nil); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	cs := []modifier.RollCircumstance{advantage}
	cmd, err := combat.NewRollCombatantDeathSavingThrowCommand(e, cid("a"), cs, nil)
	if err != nil {
		t.Fatal(err)
	}
	cs[0] = disadvantage
	uc, _ := combat.NewRollCombatantDeathSavingThrow(newFixed(4, 12))
	res, err := uc.Run(ctxBackground(), cmd)
	if got := resAs[combat.RollCombatantDeathSavingThrowCompleted](t, res).Natural(); err != nil || got != 12 {
		t.Errorf("natural %d, want 12 (Advantage taken at construction): %v", got, err)
	}
}
