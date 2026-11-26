package combat_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// resSpec cites the spec the tests of combat action resolution verify.
const resSpec = "doc/specs/combat-action-resolution.md "

// resAs returns v as a T, failing the test when it is another type.
func resAs[T any](tb testing.TB, v any) T {
	tb.Helper()
	t, ok := v.(T)
	if !ok {
		var want T
		tb.Fatalf("result is a %T, want %T", v, want)
	}
	return t
}

// resScene returns the scene at feet with the sight relations given and no fear source in sight.
func resScene(tb testing.TB, feet int, attackerSees, targetSees combat.SightRelation) combat.AttackScene {
	tb.Helper()
	s, err := combat.NewAttackScene(feet, attackerSees, targetSees, combat.LineOfSightBlocked)
	if err != nil {
		tb.Fatalf("NewAttackScene: %v", err)
	}
	return s
}

// resSeen is the scene at feet in which both creatures see each other.
func resSeen(tb testing.TB, feet int) combat.AttackScene {
	tb.Helper()
	return resScene(tb, feet, combat.SightRelationCanSee, combat.SightRelationCanSee)
}

// resBegin is the cost of an Attack action of one attack with another weapon.
func resBegin() combat.AttackCost {
	return combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansOtherWeapon}
}

// resParams are the params of a melee weapon attack with +5 against AC 15 at 5 feet for 1d8 plus 3 Slashing, by
// attacker against target of the encounter, with the cost.
func resParams(tb testing.TB, e combat.Encounter, attacker, target string, cost combat.AttackCost) combat.PerformAttackParams {
	tb.Helper()
	p := meleeParams()
	p.Damage = slashing(1, 8)
	return combat.PerformAttackParams{Encounter: e, AttackerID: cid(attacker), TargetID: cid(target), Cost: cost,
		Scene: resSeen(tb, 5), Attack: attackCommand(tb, p)}
}

// resPerform runs PerformAttack with a fixed roller of faces; it returns the result, the roller and the error.
func resPerform(tb testing.TB, p combat.PerformAttackParams, faces ...int) (combat.PerformAttackResult, *dicetest.FixedRoller, error) {
	tb.Helper()
	cmd, err := combat.NewPerformAttackCommand(p)
	if err != nil {
		tb.Fatalf("NewPerformAttackCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewPerformAttack(roller)
	if err != nil {
		tb.Fatalf("NewPerformAttack: %v", err)
	}
	res, err := uc.Run(context.Background(), cmd)
	return res, roller, err
}

// resAttack runs the attack and returns the Completed result and the dice requested, failing the test on an error or another variant.
func resAttack(tb testing.TB, p combat.PerformAttackParams, faces ...int) (combat.PerformAttackCompleted, []string) {
	tb.Helper()
	res, roller, err := resPerform(tb, p, faces...)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return resAs[combat.PerformAttackCompleted](tb, res), notations(roller)
}

// resKinds lists the Kind of each event, leaving out BecameBloodied, an optional event of hit-points-and-damage.md table 4.9a.
func resKinds(events []combat.Event) []string {
	return slices.DeleteFunc(encKinds(events), func(k string) bool { return k == "became_bloodied" })
}

// resPC is a healthy 20 Hit Point combatant with the conditions.
func resPC(tb testing.TB, id string, side combat.Side, conds ...combat.Condition) combat.Combatant {
	tb.Helper()
	return encC(tb, id, side, pcStatus(tb, 20, 20, 0), conds...)
}

// resRejected returns the rejection of a Rejected result field, failing the test when the result is another type.
type resRejectedResult interface {
	Rejection() combat.ResolutionRejection
}

// resWant asserts a Rejected result: its rejection, the encounter and ledger unchanged and no event, and no die rolled.
func resWant(tb testing.TB, res any, before combat.Encounter, ledger combat.ActionLedger, want combat.ResolutionRejection, roller *dicetest.FixedRoller) {
	tb.Helper()
	r, ok := res.(resRejectedResult)
	if !ok {
		tb.Fatalf("result = %#v, want a Rejected variant", res)
	}
	if got := r.Rejection(); got != want {
		tb.Errorf("rejection = %#v, want %#v", got, want)
	}
	state := res.(interface {
		Encounter() combat.Encounter
		Events() []combat.Event
	})
	if !state.Encounter().Equal(before) || len(state.Events()) != 0 {
		tb.Errorf("a rejection must carry the unchanged encounter and no event [FR-009]")
	}
	if l, ok := res.(interface{ Ledger() combat.ActionLedger }); ok && !l.Ledger().Equal(ledger) {
		tb.Errorf("a rejection must carry the unchanged ledger [FR-009]")
	}
	if roller != nil && len(roller.Requests()) != 0 {
		tb.Errorf("dice requested %v, want none", notations(roller))
	}
}

// resHas reports whether the combatant of the encounter holds the kind effectively.
func resHas(tb testing.TB, e combat.Encounter, id string, k content.ConditionKind) bool {
	tb.Helper()
	return encHas(encMember(tb, e, id), k)
}

// resEnded returns the encounter after the host ended it.
func resEnded(tb testing.TB, e combat.Encounter) combat.Encounter {
	tb.Helper()
	cmd, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		tb.Fatal(err)
	}
	res, err := combat.NewEndEncounter().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return resAs[combat.EndEncounterApplied](tb, res).Encounter()
}

// resDead is a combatant killed by massive damage.
func resDead(tb testing.TB, id string, side combat.Side) combat.Combatant {
	tb.Helper()
	return encC(tb, id, side, deadStatus(tb, combat.MassiveDamage))
}

// resDying is a Dying combatant with no Death Saving Throw made.
func resDying(tb testing.TB, id string, side combat.Side, conds ...combat.Condition) combat.Combatant {
	tb.Helper()
	return encC(tb, id, side, dyingStatus(tb, 20, 0, 0), conds...)
}

// resTake takes the request for the combatant and returns the Completed result, failing the test on anything else.
func resTake(tb testing.TB, e combat.Encounter, l combat.ActionLedger, who string, request combat.TurnRequest) combat.TakeCombatantActionCompleted {
	tb.Helper()
	cmd, err := combat.NewTakeCombatantActionCommand(e, l, cid(who), request)
	if err != nil {
		tb.Fatalf("NewTakeCombatantActionCommand: %v", err)
	}
	res, err := combat.NewTakeCombatantAction().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return resAs[combat.TakeCombatantActionCompleted](tb, res)
}

// resAdvance advances the turn and returns the Completed result, failing the test on anything else.
func resAdvance(tb testing.TB, e combat.Encounter, l combat.ActionLedger) combat.AdvanceCombatTurnCompleted {
	tb.Helper()
	cmd, err := combat.NewAdvanceCombatTurnCommand(e, l, nil, nil)
	if err != nil {
		tb.Fatalf("NewAdvanceCombatTurnCommand: %v", err)
	}
	res, err := combat.NewAdvanceCombatTurn().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return resAs[combat.AdvanceCombatTurnCompleted](tb, res)
}

// resDodge has the combatant take the Dodge action.
func resDodge(tb testing.TB, e combat.Encounter, l combat.ActionLedger, who string) combat.TakeCombatantActionCompleted {
	tb.Helper()
	return resTake(tb, e, l, who, combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}})
}

// resAttacks returns the attacks the Attack action in progress has counted, failing the test when none is in progress.
func resAttacks(tb testing.TB, l combat.ActionLedger) []combat.AttackEntry {
	tb.Helper()
	p, ok := l.AttackAction().(combat.AttackActionInProgress)
	if !ok {
		tb.Fatalf("no Attack action in progress, want one")
	}
	return p.Progress.Attacks()
}

// resHP returns the current Hit Points of the combatant.
func resHP(tb testing.TB, e combat.Encounter, id string) int {
	tb.Helper()
	return encMember(tb, e, id).HitPoints().HitPoints().Current()
}

// resImmune is a healthy combatant that is Immune to the condition kind.
func resImmune(tb testing.TB, id string, side combat.Side, kind content.ConditionKind) combat.Combatant {
	tb.Helper()
	ref := combat.CombatantRef{ID: cid(id), Kind: combat.MonsterCombatant{ID: content.MonsterID("goblin")}}
	c, err := combat.NewCombatant(ref, side, pcStatus(tb, 20, 20, 0), condSet(tb, nil, kind))
	if err != nil {
		tb.Fatalf("NewCombatant: %v", err)
	}
	return c
}

// resStrikeParams are the params of an Unarmed Strike with the option by attacker against target, at 5 feet, beginning an Attack action.
func resStrikeParams(tb testing.TB, e combat.Encounter, attacker, target string, o combat.UnarmedStrikeOption) combat.PerformUnarmedStrikeParams {
	tb.Helper()
	return combat.PerformUnarmedStrikeParams{Encounter: e, AttackerID: cid(attacker), TargetID: cid(target),
		Cost:   combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansUnarmedStrike},
		Scene:  resSeen(tb, 5),
		Strike: unarmedCommand(tb, unarmedParams(o))}
}

// resStrike runs the Unarmed Strike with a fixed roller of faces and returns the Completed result and the dice requested.
func resStrike(tb testing.TB, p combat.PerformUnarmedStrikeParams, faces ...int) (combat.PerformUnarmedStrikeCompleted, []string) {
	tb.Helper()
	cmd, err := combat.NewPerformUnarmedStrikeCommand(p)
	if err != nil {
		tb.Fatalf("NewPerformUnarmedStrikeCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewPerformUnarmedStrike(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return resAs[combat.PerformUnarmedStrikeCompleted](tb, res), notations(roller)
}

// strengthSave is a Strength saving throw of +0, rolled.
func strengthSave(tb testing.TB) d20.SaveAttempt {
	tb.Helper()
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
}

// ctxBackground is the context of a test that does not cancel.
func ctxBackground() context.Context { return context.Background() }

// newFixed is a fixed roller of the faces.
func newFixed(faces ...int) *dicetest.FixedRoller { return dicetest.NewFixedRoller(faces...) }

// reflectTypeOf is the dynamic type of v, to compare the variant of a closed result.
func reflectTypeOf(v any) reflect.Type { return reflect.TypeOf(v) }

// resStartWith starts an encounter in the order of the combatants, giving the walking Speed in speeds (30 where none is given).
func resStartWith(tb testing.TB, speeds map[string]int, cs ...combat.Combatant) combat.Encounter {
	tb.Helper()
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, c := range cs {
		id := c.Ref().ID
		feet := 30
		if v, ok := speeds[string(id)]; ok {
			feet = v
		}
		standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(100 - 10*i)})
		profiles = append(profiles, trkProfile(tb, id, content.SizeMedium, trkSpeeds(tb, feet)))
	}
	cmd, err := establishCommandAnySize(standings, profiles)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatal(err)
	}
	e, _, err := startAnySize(eid(), resAs[combat.InitiativeOrderEstablished](tb, res), cs)
	if err != nil {
		tb.Fatal(err)
	}
	return e
}

// resMoveRun runs MoveCombatant and returns the result.
func resMoveRun(tb testing.TB, e combat.Encounter, who string, req combat.MovementRequest) (combat.MoveCombatantResult, error) {
	tb.Helper()
	cmd, err := combat.NewMoveCombatantCommand(e, cid(who), req)
	if err != nil {
		tb.Fatalf("NewMoveCombatantCommand: %v", err)
	}
	return combat.NewMoveCombatant().Run(context.Background(), cmd)
}

// resMove runs MoveCombatant and returns the Completed result.
func resMove(tb testing.TB, e combat.Encounter, who string, req combat.MovementRequest) combat.MoveCombatantCompleted {
	tb.Helper()
	res, err := resMoveRun(tb, e, who, req)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return resAs[combat.MoveCombatantCompleted](tb, res)
}

// resAlong is the request to walk feet.
func resAlong(tb testing.TB, feet int) combat.MoveAlong {
	tb.Helper()
	m, err := combat.NewMoveAlong([]combat.MoveSegment{trkWalk(tb, feet)})
	if err != nil {
		tb.Fatal(err)
	}
	return m
}

// resThrow runs RollCombatantDeathSavingThrow with a fixed roller of faces and returns the result, the dice requested and the error.
func resThrow(tb testing.TB, e combat.Encounter, who string, circumstances []modifier.RollCircumstance, modifiers []modifier.Modifier, faces ...int) (combat.RollCombatantDeathSavingThrowResult, []string, error) {
	tb.Helper()
	cmd, err := combat.NewRollCombatantDeathSavingThrowCommand(e, cid(who), circumstances, modifiers)
	if err != nil {
		tb.Fatalf("NewRollCombatantDeathSavingThrowCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewRollCombatantDeathSavingThrow(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	return res, notations(roller), err
}

// resDyingEncounter is an encounter in which "a" is Dying and active with the tallies, and "b" is healthy.
func resDyingEncounter(tb testing.TB, successes, failures int, conds ...combat.Condition) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encC(tb, "a", combat.SideA, dyingStatus(tb, 20, successes, failures), conds...), resPC(tb, "b", combat.SideB))
	return e
}

// resStabilizeParams are the params of the helper "a" making a Wisdom (Medicine) check of +0 to stabilize "b" in the encounter.
func resStabilizeParams(e combat.Encounter) combat.AttemptToStabilizeCombatantParams {
	return combat.AttemptToStabilizeCombatantParams{Encounter: e, HelperID: cid("a"), PatientID: cid("b"),
		Check: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}, FearSource: combat.LineOfSightBlocked}
}

// resStabilize runs AttemptToStabilizeCombatant with a fixed roller of faces and returns the result, the dice requested and the error.
func resStabilize(tb testing.TB, p combat.AttemptToStabilizeCombatantParams, faces ...int) (combat.AttemptToStabilizeCombatantResult, []string, error) {
	tb.Helper()
	cmd, err := combat.NewAttemptToStabilizeCombatantCommand(p)
	if err != nil {
		tb.Fatalf("NewAttemptToStabilizeCombatantCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewAttemptToStabilizeCombatant(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	return res, notations(roller), err
}
