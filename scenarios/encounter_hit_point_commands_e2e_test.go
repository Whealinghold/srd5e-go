package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpBMember is one combatant of a grpBStartEncounter call.
type grpBMember struct {
	id         string
	side       combat.Side
	status     combat.HitPointStatus
	conditions combat.ConditionSet
}

// grpBStartEncounter establishes initiative for the members in the order given and starts the encounter, through the public use cases only.
func grpBStartEncounter(t *testing.T, members ...grpBMember) combat.Encounter {
	t.Helper()
	ctx := context.Background()
	speed, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	var combatants []combat.Combatant
	for i, m := range members {
		id := combat.MustCombatantID(m.id)
		standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(20 - 2*i)})
		profiles = append(profiles, combat.CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: speed})
		c, err := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: content.MonsterID(m.id)}}, m.side, m.status, m.conditions)
		if err != nil {
			t.Fatal(err)
		}
		combatants = append(combatants, c)
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	est, err := combat.NewEstablishInitiativeOrder().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("grpb"), est.(combat.InitiativeOrderEstablished), combatants))
	if err != nil {
		t.Fatal(err)
	}
	return started.(combat.StartEncounterApplied).Encounter()
}

func grpBFind(t *testing.T, e combat.Encounter, id string) combat.Combatant {
	t.Helper()
	c, ok := e.Find(combat.MustCombatantID(id))
	if !ok {
		t.Fatalf("no combatant %q in the encounter", id)
	}
	return c
}

// Requirement: doc/specs/encounter-aggregate.md FR-074, FR-075 (SRD 5.2.1, "Stabilizing a Character", p. 18; hit-points-and-damage.md FR-067, FR-073) - a Dying combatant is stabilized by the host's Help check, becomes
// Stable (no more Death Saving Throws), and once the host's rolled delay is over is recovered to 1 Hit Point, Alive and awake.
// Path: happy.
func TestGrpBEncounterStabilizesThenRecoversACombatant(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dying, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 20, 0), combat.Dying{}, combat.FallsUnconscious, combat.NoAffinities())
	e := grpBStartEncounter(t,
		grpBMember{"hero", combat.SideA, dying, combat.MustConditionSet([]combat.Condition{combat.Unconscious{}}, nil)},
		grpBMember{"orc", combat.SideB, combat.MustHitPointStatus(15, combat.DiesInstantly, combat.NoAffinities()), combat.ConditionSet{}})

	stab, err := combat.NewStabilizeCombatantCommand(e, "hero")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := combat.NewStabilizeCombatant().Run(ctx, stab)
	if err != nil {
		t.Fatal(err)
	}
	e = sr.(combat.StabilizeCombatantApplied).Encounter()
	if m := grpBFind(t, e, "hero"); m.HitPoints().LifeState() != (combat.Stable{}) || m.HitPoints().MakesDeathSavingThrows() {
		t.Fatalf("after stabilizing = %v, want Stable with no Death Saving Throws", m.HitPoints().LifeState())
	}

	rec, err := combat.NewRecoverStableCombatantCommand(e, "hero")
	if err != nil {
		t.Fatal(err)
	}
	rr, err := combat.NewRecoverStableCombatant().Run(ctx, rec)
	if err != nil {
		t.Fatal(err)
	}
	e = rr.(combat.RecoverStableCombatantApplied).Encounter()
	m := grpBFind(t, e, "hero")
	if hp := m.HitPoints().HitPoints(); hp.Current() != 1 || m.HitPoints().LifeState() != (combat.Alive{}) {
		t.Errorf("after recovery = %+v %v, want 1 Hit Point and Alive", hp, m.HitPoints().LifeState())
	}
	for _, k := range m.Conditions().Effective() {
		if k == content.ConditionUnconscious {
			t.Errorf("the recovered hero is still Unconscious")
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-074, FR-075, FR-079, FR-045 - stabilizing a combatant that is not Dying and recovering one that is not Stable come back as HitPointRuleRejection (NotDying,
// NotStable), a Dead combatant is CombatantIsDeadRejection, an id outside the encounter is UnknownCombatantRejection, an ended encounter refuses, and a blank id or a zero encounter is invalid input; the
// encounter is returned unchanged by every refusal.
// Path: unhappy (state error, invalid input).
func TestGrpBEncounterRefusesStabilizingAndRecoveringTheWrongCombatant(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dead, _ := combat.NewDead(combat.MassiveDamage)
	deadStatus, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 20, 0), dead, combat.FallsUnconscious, combat.NoAffinities())
	e := grpBStartEncounter(t,
		grpBMember{"hero", combat.SideA, combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()), combat.ConditionSet{}},
		grpBMember{"fallen", combat.SideA, deadStatus, combat.ConditionSet{}},
		grpBMember{"orc", combat.SideB, combat.MustHitPointStatus(15, combat.DiesInstantly, combat.NoAffinities()), combat.ConditionSet{}})
	hero, fallen := combat.MustCombatantID("hero"), combat.MustCombatantID("fallen")

	stabilize := func(enc combat.Encounter, id combat.CombatantID) combat.EncounterRejection {
		t.Helper()
		cmd, err := combat.NewStabilizeCombatantCommand(enc, id)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewStabilizeCombatant().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		r := res.(combat.StabilizeCombatantRejected)
		if !r.Encounter().Equal(enc) {
			t.Errorf("the refusal of %s changed the encounter", id)
		}
		return r.Rejection
	}
	recover := func(enc combat.Encounter, id combat.CombatantID) combat.EncounterRejection {
		t.Helper()
		cmd, err := combat.NewRecoverStableCombatantCommand(enc, id)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRecoverStableCombatant().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.RecoverStableCombatantRejected).Rejection
	}

	if got := stabilize(e, hero); got != (combat.HitPointRuleRejection{ID: hero, Reason: combat.NotDying}) {
		t.Errorf("stabilizing an Alive combatant = %#v, want NotDying", got)
	}
	if got := recover(e, hero); got != (combat.HitPointRuleRejection{ID: hero, Reason: combat.NotStable}) {
		t.Errorf("recovering an Alive combatant = %#v, want NotStable", got)
	}
	if got := stabilize(e, fallen); got != (combat.CombatantIsDeadRejection{ID: fallen}) {
		t.Errorf("stabilizing a Dead combatant = %#v, want CombatantIsDead", got)
	}
	if got := recover(e, fallen); got != (combat.CombatantIsDeadRejection{ID: fallen}) {
		t.Errorf("recovering a Dead combatant = %#v, want CombatantIsDead", got)
	}
	ghost := combat.MustCombatantID("ghost")
	if got := stabilize(e, ghost); got != (combat.UnknownCombatantRejection{ID: ghost}) {
		t.Errorf("stabilizing an unknown id = %#v, want UnknownCombatant", got)
	}

	endCmd, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		t.Fatal(err)
	}
	er, err := combat.NewEndEncounter().Run(ctx, endCmd)
	if err != nil {
		t.Fatal(err)
	}
	over := er.(combat.EndEncounterApplied).Encounter()
	if got := stabilize(over, hero); got != (combat.EncounterAlreadyEndedRejection{}) {
		t.Errorf("stabilizing after the end = %#v, want EncounterAlreadyEnded", got)
	}

	if _, err := combat.NewStabilizeCombatantCommand(e, combat.CombatantID("")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank id err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewRecoverStableCombatantCommand(combat.Encounter{}, hero); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero encounter err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewStabilizeCombatant().Run(ctx, combat.StabilizeCombatantCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero command err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-076, FR-077 (hit-points-and-damage.md FR-029, FR-030, FR-046, FR-047) - a combatant gains Temporary Hit Points (the higher of old and new when the host
// picks AcceptNew over KeepCurrent), loses maximum Hit Points to a drain, dies when the maximum reaches 0 (HitPointMaximumZero), and the commands reject an amount below 1 and a negative reduction as invalid input.
// Path: happy and unhappy (boundary, invalid input, state error).
func TestGrpBEncounterTemporaryHitPointsAndMaximumReduction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := grpBStartEncounter(t,
		grpBMember{"hero", combat.SideA, combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()), combat.ConditionSet{}},
		grpBMember{"orc", combat.SideB, combat.MustHitPointStatus(15, combat.DiesInstantly, combat.NoAffinities()), combat.ConditionSet{}})
	hero := combat.MustCombatantID("hero")

	temp := func(enc combat.Encounter, amount int, choice combat.TemporaryHitPointsChoice) combat.Encounter {
		t.Helper()
		cmd, err := combat.NewGiveCombatantTemporaryHitPointsCommand(enc, hero, amount, choice)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewGiveCombatantTemporaryHitPoints().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.GiveCombatantTemporaryHitPointsApplied).Encounter()
	}
	e = temp(e, 5, combat.AcceptNew)
	if got := grpBFind(t, e, "hero").HitPoints().HitPoints().Temporary(); got != 5 {
		t.Fatalf("temporary = %d, want 5", got)
	}
	if got := grpBFind(t, temp(e, 3, combat.AcceptNew), "hero").HitPoints().HitPoints().Temporary(); got != 3 {
		t.Errorf("AcceptNew with 3 = %d temporary, want 3 (they do not stack)", got)
	}
	if got := grpBFind(t, temp(e, 3, combat.KeepCurrent), "hero").HitPoints().HitPoints().Temporary(); got != 5 {
		t.Errorf("KeepCurrent against 3 = %d temporary, want 5", got)
	}

	drain := func(enc combat.Encounter, amount int) combat.Encounter {
		t.Helper()
		cmd, err := combat.NewReduceCombatantHitPointMaximumCommand(enc, hero, amount)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewReduceCombatantHitPointMaximum().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.ReduceCombatantHitPointMaximumApplied).Encounter()
	}
	e = drain(e, 8)
	if hp := grpBFind(t, e, "hero").HitPoints().HitPoints(); hp.Maximum() != 12 || hp.Current() != 12 {
		t.Errorf("after the drain = %+v, want 12 of 12", hp)
	}
	e = drain(e, 12)
	if cause, dead := grpBFind(t, e, "hero").Death(); !dead || cause != combat.HitPointMaximumZero {
		t.Errorf("a maximum of 0 left the hero %v dead=%v, want Dead by HitPointMaximumZero", cause, dead)
	}
	again, err := combat.NewReduceCombatantHitPointMaximumCommand(e, hero, 1)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewReduceCombatantHitPointMaximum().Run(ctx, again)
	if err != nil {
		t.Fatal(err)
	}
	if r := res.(combat.ReduceCombatantHitPointMaximumRejected); r.Rejection != (combat.CombatantIsDeadRejection{ID: hero}) {
		t.Errorf("draining a Dead combatant = %#v, want CombatantIsDead", r.Rejection)
	}

	if _, err := combat.NewGiveCombatantTemporaryHitPointsCommand(e, hero, 0, combat.AcceptNew); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("0 temporary Hit Points err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewReduceCombatantHitPointMaximumCommand(e, hero, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative reduction err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-063, FR-064, FR-037 and conditions.md FR-036, FR-040 (SRD 5.2.1, "Exhaustion", p. 181) - a combatant at Exhaustion level 3 loses two levels to a Long Rest
// (level 1), then the last level ends the condition; reducing a combatant without Exhaustion is refused as DoesNotHaveCondition; a Dead combatant is refused; levels below 1 are invalid input.
// Path: happy and unhappy (boundary, state error, invalid input).
func TestGrpBEncounterReducesExhaustion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dead, _ := combat.NewDead(combat.Exhaustion)
	deadStatus, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 20, 0), dead, combat.FallsUnconscious, combat.NoAffinities())
	e := grpBStartEncounter(t,
		grpBMember{"hero", combat.SideA, combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()),
			combat.MustConditionSet([]combat.Condition{combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(3)}}, nil)},
		grpBMember{"rested", combat.SideA, combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()), combat.ConditionSet{}},
		grpBMember{"fallen", combat.SideA, deadStatus, combat.ConditionSet{}},
		grpBMember{"orc", combat.SideB, combat.MustHitPointStatus(15, combat.DiesInstantly, combat.NoAffinities()), combat.ConditionSet{}})
	hero := combat.MustCombatantID("hero")
	reduce := func(enc combat.Encounter, id combat.CombatantID, levels int) combat.ReduceCombatantExhaustionResult {
		t.Helper()
		cmd, err := combat.NewReduceCombatantExhaustionCommand(enc, id, levels)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewReduceCombatantExhaustion().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	step := reduce(e, hero, 2).(combat.ReduceCombatantExhaustionApplied)
	if got := grpBFind(t, step.Encounter(), "hero").Conditions().Explicit(); len(got) != 1 || got[0] != (combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)}) {
		t.Errorf("after two levels off = %v, want Exhaustion level 1", got)
	}
	if got := kinds(step.Events()); len(got) != 1 || got[0] != "condition_level_reduced" {
		t.Errorf("events = %v, want one condition_level_reduced", got)
	}
	last := reduce(step.Encounter(), hero, 1).(combat.ReduceCombatantExhaustionApplied)
	if got := grpBFind(t, last.Encounter(), "hero").Conditions().Explicit(); len(got) != 0 {
		t.Errorf("after the last level = %v, want no condition", got)
	}

	if r := reduce(e, combat.MustCombatantID("rested"), 1).(combat.ReduceCombatantExhaustionRejected); r.Rejection != (combat.DoesNotHaveConditionRejection{ID: combat.MustCombatantID("rested"), Kind: content.ConditionExhaustion}) {
		t.Errorf("reducing no Exhaustion = %#v, want DoesNotHaveCondition(Exhaustion)", r.Rejection)
	}
	fallen := combat.MustCombatantID("fallen")
	if r := reduce(e, fallen, 1).(combat.ReduceCombatantExhaustionRejected); r.Rejection != (combat.CombatantIsDeadRejection{ID: fallen}) {
		t.Errorf("reducing a Dead combatant = %#v, want CombatantIsDead", r.Rejection)
	}
	if _, err := combat.NewReduceCombatantExhaustionCommand(e, hero, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("0 levels err = %v, want ErrInvalidInput", err)
	}
}
