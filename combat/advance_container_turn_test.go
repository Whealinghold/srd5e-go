package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ctSwallowed swallows a Small "p" into the fixture's container "m" and returns the set and p's conditions.
func ctSwallowed(tb testing.TB, name string, own bool) (combat.Containments, combat.ConditionSet) {
	tb.Helper()
	in := ctEnterInput(tb, name)
	in.OccupantSize, in.DuringContainerTurn, in.Attempt = content.SizeSmall, own, nil
	if name != "frog" && name != "toad" {
		in.Attempt = rollOf(0)
	}
	c := ctContained(tb, mustEnter(tb, in, 1))
	return c.Containments(), c.Conditions()
}

func ctAdvance(tb testing.TB, in combat.AdvanceContainerTurnInput) combat.AdvanceContainerTurnResult {
	tb.Helper()
	cmd, err := combat.NewAdvanceContainerTurnCommand(in)
	if err != nil {
		tb.Fatalf("NewAdvanceContainerTurnCommand: %v", err)
	}
	res, err := combat.NewAdvanceContainerTurn().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res
}

func ctAdvanced(tb testing.TB, res combat.AdvanceContainerTurnResult) combat.AdvanceContainerTurnAdvanced {
	tb.Helper()
	a, ok := res.(combat.AdvanceContainerTurnAdvanced)
	if !ok {
		tb.Fatalf("result = %#v, want advanced", res)
	}
	return a
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-043, FR-044 - the start boundary reports the damage due of every record whose periodic damage is due at the start of the turn, one per occupant in entry order, each incrementing its record's ticks; the end boundary, records trapped in a dead container and terms with no periodic damage give none.
// Path: per case.
func TestAdvanceContainerTurnPeriodicDamage(t *testing.T) {
	t.Parallel()
	worm := ctHolding(t, "worm", "m", "a", "b")
	start := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: worm, Container: gid("m"), Boundary: combat.ContainerTurnBoundaryStart}))
	due := start.DamageDue()
	if len(due) != 2 || due[0].Target != gid("a") || due[1].Target != gid("b") || due[0].Cause() != combat.ContainmentDamageCausePeriodic ||
		due[0].Terms()[0].Expression().Notation() != "5d6" || due[0].Terms()[0].DamageType() != content.DamageAcid || due[0].Halved() {
		t.Fatalf("start dues = %+v, want 5d6 acid for a then b", due)
	}
	for _, r := range start.Containments().All() {
		if r.DamageTicks() != 1 {
			t.Errorf("%v ticks = %d, want 1", r.Occupant, r.DamageTicks())
		}
	}
	if worm.All()[0].DamageTicks() != 0 {
		t.Error("the input set must not change")
	}
	if end := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: worm, Container: gid("m"), Boundary: combat.ContainerTurnBoundaryEnd})); len(end.DamageDue()) != 0 {
		t.Error("a start-timed damage is not due at the end")
	}
	if other := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: worm, Container: gid("other"), Boundary: combat.ContainerTurnBoundaryStart})); len(other.DamageDue()) != 0 {
		t.Error("another container's turn changes nothing")
	}
	remorhaz := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: ctHolding(t, "remorhaz", "m", "a"), Container: gid("m"), Boundary: combat.ContainerTurnBoundaryStart}))
	if terms := remorhaz.DamageDue()[0].Terms(); len(terms) != 2 || terms[0].DamageType() != content.DamageAcid || terms[1].DamageType() != content.DamageFire {
		t.Errorf("Remorhaz terms = %v, want acid then fire", terms)
	}
	trapped := ctSet(t, ctRecord(t, "m", "a", content.SizeMedium, ctTerms(t, "worm"), 1, func(in *combat.ContainmentInput) { in.Phase = mustTrapped(t, 20) }))
	if got := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: trapped, Container: gid("m"), Boundary: combat.ContainerTurnBoundaryStart})); len(got.DamageDue()) != 0 {
		t.Error("a trapped record takes no damage: the container takes no turns")
	}
	noPeriodic := ctSet(t, ctRecord(t, "m", "a", content.SizeMedium, ctTerms(t, "kraken"), 1))
	if got := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: noPeriodic, Container: gid("m"), Boundary: combat.ContainerTurnBoundaryEnd})); len(got.DamageDue()) != 0 {
		t.Error("end boundary of start-timed terms")
	}
	empty := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Container: gid("m"), Boundary: combat.ContainerTurnBoundaryEnd}))
	if empty.Containments().Len() != 0 || len(empty.Releases()) != 0 || len(empty.Events()) != 0 {
		t.Error("a container holding nobody has nothing due")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-078, FR-044, table 4.7.3 - the Giant Frog's first tick is the end of its next turn: a frog that swallowed on its own turn has no damage at the end of that turn and the flag is cleared; the next end boundary deals 2d4 and disgorges a survivor Prone, or removes one who is not among the survivors; the toad deals damage at the end of the turn it swallowed on and never disgorges.
// Path: per case.
func TestAdvanceContainerTurnFrogAndToad(t *testing.T) {
	t.Parallel()
	end := combat.ContainerTurnBoundaryEnd
	// The frog's own turn.
	set, conds := ctSwallowed(t, "frog", true)
	first := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: set, Container: gid("m"), Boundary: end, Survivors: []combat.CombatantID{gid("p")}}))
	rec := first.Containments().ContainerOf(gid("p")).(combat.InContainer).Record()
	if len(first.DamageDue()) != 0 || rec.TickDeferred() || rec.DamageTicks() != 0 || len(first.Releases()) != 0 {
		t.Fatalf("end of the swallowing turn: due %v deferred %v ticks %d releases %d, want nothing but the flag cleared", first.DamageDue(), rec.TickDeferred(), rec.DamageTicks(), len(first.Releases()))
	}
	second := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: first.Containments(), Container: gid("m"), Boundary: end,
		OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): conds}, Survivors: []combat.CombatantID{gid("p")}}))
	due := second.DamageDue()
	if len(due) != 1 || due[0].Terms()[0].Expression().Notation() != "2d4" || second.Containments().Len() != 0 || len(second.Releases()) != 1 {
		t.Fatalf("end of the next turn: due %v, set %d, releases %d, want 2d4 and a disgorged frog meal", due, second.Containments().Len(), len(second.Releases()))
	}
	rel := second.Releases()[0]
	if rel.Cause() != combat.ContainmentLeaveCauseDisgorged || !slices.Equal(rel.Conditions().Effective(), []content.ConditionKind{content.ConditionProne}) || rel.Occupant != gid("p") {
		t.Errorf("release = %v conditions %v, want disgorged Prone and the added conditions removed", rel.Cause(), rel.Conditions().Effective())
	}
	if !slices.Equal(ctKinds(second.Events()), []string{"condition_removed", "condition_removed", "occupant_released", "condition_applied"}) {
		t.Errorf("events = %v", ctKinds(second.Events()))
	}

	// Swallowed on another creature's turn: the first end boundary deals the damage and disgorges.
	other, otherConds := ctSwallowed(t, "frog", false)
	got := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: other, Container: gid("m"), Boundary: end,
		OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): otherConds}, Survivors: []combat.CombatantID{gid("p")}}))
	if len(got.DamageDue()) != 1 || len(got.Releases()) != 1 {
		t.Errorf("another creature's turn: due %d releases %d, want 1 and 1", len(got.DamageDue()), len(got.Releases()))
	}

	// The frog's meal did not survive the damage: it is removed, with no condition touched.
	dead := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: other, Container: gid("m"), Boundary: end}))
	if len(dead.Releases()) != 1 || dead.Releases()[0].Cause() != combat.ContainmentLeaveCauseOccupantRemoved || dead.Containments().Len() != 0 || len(dead.DamageDue()) != 1 ||
		dead.Releases()[0].Conditions() != (combat.ConditionSet{}) {
		t.Errorf("not among the survivors: %+v", dead)
	}

	// The toad: damage at the end of the turn it swallowed on, every turn, never disgorged.
	toad, _ := ctSwallowed(t, "toad", true)
	for i := 0; i < 2; i++ {
		res := ctAdvanced(t, ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: toad, Container: gid("m"), Boundary: end, Survivors: []combat.CombatantID{gid("p")}}))
		if len(res.DamageDue()) != 1 || res.DamageDue()[0].Terms()[0].Expression().Notation() != "3d6" || len(res.Releases()) != 0 || res.Containments().Len() != 1 {
			t.Fatalf("toad turn %d: %+v", i, res)
		}
		toad = res.Containments()
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-063 - the map of conditions holds exactly the creatures the call disgorges; a missing or extra id is an Invalid result, not an error or a panic.
// Path: per case.
func TestAdvanceContainerTurnConditionMap(t *testing.T) {
	t.Parallel()
	set, conds := ctSwallowed(t, "frog", false)
	end, start := combat.ContainerTurnBoundaryEnd, combat.ContainerTurnBoundaryStart
	survivors := []combat.CombatantID{gid("p")}
	for _, c := range []struct {
		name     string
		boundary combat.ContainerTurnBoundary
		have     map[combat.CombatantID]combat.ConditionSet
		survive  []combat.CombatantID
		want     combat.ContainmentProblem
	}{
		{"a disgorged creature without conditions", end, nil, survivors, combat.ContainmentProblemMissingOccupantConditions},
		{"an empty map", end, map[combat.CombatantID]combat.ConditionSet{}, survivors, combat.ContainmentProblemMissingOccupantConditions},
		{"an extra creature", end, map[combat.CombatantID]combat.ConditionSet{gid("p"): conds, gid("zz"): {}}, survivors, combat.ContainmentProblemUnknownOccupant},
		{"conditions for nobody disgorged", end, map[combat.CombatantID]combat.ConditionSet{gid("p"): conds}, nil, combat.ContainmentProblemUnknownOccupant},
		{"conditions at the start boundary", start, map[combat.CombatantID]combat.ConditionSet{gid("p"): conds}, survivors, combat.ContainmentProblemUnknownOccupant},
	} {
		res := ctAdvance(t, combat.AdvanceContainerTurnInput{Containments: set, Container: gid("m"), Boundary: c.boundary, OccupantConditions: c.have, Survivors: c.survive})
		if res != (combat.AdvanceContainerTurnInvalid{Problem: c.want}) {
			t.Errorf("%s: result = %#v, want invalid %v", c.name, res, c.want)
		}
	}
	for name, in := range map[string]combat.AdvanceContainerTurnInput{
		"blank container": {Boundary: start}, "boundary 0": {Container: gid("m")}, "boundary 3": {Container: gid("m"), Boundary: 3}} {
		if _, err := combat.NewAdvanceContainerTurnCommand(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	surv := []combat.CombatantID{gid("p")}
	cmd, _ := combat.NewAdvanceContainerTurnCommand(combat.AdvanceContainerTurnInput{Containments: set, Container: gid("m"), Boundary: end,
		OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): conds}, Survivors: surv})
	surv[0] = gid("zz")
	res, err := combat.NewAdvanceContainerTurn().Run(context.Background(), cmd)
	if err != nil || reflect.TypeOf(res) != reflect.TypeOf(combat.AdvanceContainerTurnAdvanced{}) || len(res.(combat.AdvanceContainerTurnAdvanced).Releases()) != 1 {
		t.Errorf("the command must copy its survivors: %#v %v", res, err)
	}
}
