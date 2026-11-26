package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func resAdvanceRun(tb testing.TB, e combat.Encounter, l combat.ActionLedger, updates []combat.SpeedUpdate, occupants []content.CreatureSize) (combat.AdvanceCombatTurnResult, error) {
	tb.Helper()
	cmd, err := combat.NewAdvanceCombatTurnCommand(e, l, updates, occupants)
	if err != nil {
		tb.Fatalf("NewAdvanceCombatTurnCommand: %v", err)
	}
	return combat.NewAdvanceCombatTurn().Run(ctxBackground(), cmd)
}

// Requirement: doc/specs/combat-action-resolution.md FR-064, FR-067, FR-115, 4.15.1 advance - the turn advances through the root: the active combatant and the round change, a dead combatant's turn is not skipped, and the Attack action in progress ends with the turn.
// Path: happy.
func TestAdvanceCombatTurnAdvances(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB), resPC(t, "c", combat.SideA))
	begun, _ := resAttack(t, resParams(t, e, "a", "c", resBegin()), 14, 4)
	got := resAdvance(t, begun.Encounter(), begun.Ledger())
	if want := []string{"tracker_turn_ended", "tracker_turn_started"}; !slices.Equal(resKinds(got.Events()), want) ||
		got.Encounter().Tracker().Active().Combatant != cid("b") || got.Encounter().Tracker().Round() != 1 {
		t.Errorf("events %v, active %v: want %v with the dead b active, its turn not skipped [FR-067]", resKinds(got.Events()), got.Encounter().Tracker().Active().Combatant, want)
	}
	if _, in := got.Ledger().AttackAction().(combat.AttackActionInProgress); in {
		t.Errorf("the Attack action must end with the turn [FR-115]")
	}
	again := resAdvance(t, got.Encounter(), got.Ledger())
	wrap := resAdvance(t, again.Encounter(), again.Ledger())
	if want := []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started"}; !slices.Equal(resKinds(wrap.Events()), want) || wrap.Encounter().Tracker().Round() != 2 {
		t.Errorf("events %v, round %d, want %v and round 2", resKinds(wrap.Events()), wrap.Encounter().Tracker().Round(), want)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-066, 4.15.1 advance - a Dying combatant's turn start gives DyingCombatantTurnStarted, passed through as the encounter produced it and once; the use case rolls no Death Saving Throw (it has no roller).
// Path: unhappy (state).
func TestAdvanceCombatTurnPromptsADyingCombatant(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB))
	got := resAdvance(t, e, combat.ActionLedger{})
	if want := []string{"tracker_turn_ended", "tracker_turn_started", "dying_combatant_turn_started"}; !slices.Equal(resKinds(got.Events()), want) {
		t.Errorf("events = %v, want %v", resKinds(got.Events()), want)
	}
	if encMember(t, got.Encounter(), "b").DeathSavingThrowMade() {
		t.Errorf("a Death Saving Throw was made, want the host's call to roll it")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-118, FR-123, 4.18.1 Help - a Help benefit its giver did not use before its next turn expires with one HelpBenefitExpired, after the encounter's events; a benefit of another creature stays.
// Path: unhappy (state).
func TestAdvanceCombatTurnExpiresHelp(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB), resPC(t, "c", combat.SideA))
	helped := resTake(t, e, combat.ActionLedger{}, "a", combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("b"), Target: combat.AttackAssistTargetEnemyWithinFiveFeet}})
	toB := resAdvance(t, helped.Encounter(), helped.Ledger())
	toC := resAdvance(t, toB.Encounter(), toB.Ledger())
	if toB.Ledger().HelpBenefits().Len() != 1 || toC.Ledger().HelpBenefits().Len() != 1 {
		t.Fatalf("benefits %d and %d, want the benefit kept on the turns of b and c", toB.Ledger().HelpBenefits().Len(), toC.Ledger().HelpBenefits().Len())
	}
	toA := resAdvance(t, toC.Encounter(), toC.Ledger())
	if want := []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started", "help_benefit_expired"}; !slices.Equal(resKinds(toA.Events()), want) || toA.Ledger().HelpBenefits().Len() != 0 {
		t.Errorf("events %v, benefits %d, want %v and none left", resKinds(toA.Events()), toA.Ledger().HelpBenefits().Len(), want)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-065, 4.15.1 advance - a creature that ends its turn in a space shared with another falls Prone through the encounter, after the advance's events; a creature that already has Prone, is Immune to it or is dead does not stop the call and gives no condition event.
// Path: per case.
func TestAdvanceCombatTurnAppliesProneInASharedSpace(t *testing.T) {
	t.Parallel()
	occupants := []content.CreatureSize{content.SizeMedium}
	for _, c := range []struct {
		name, requirement string
		active            combat.Combatant
		events            []string
		prone             bool
	}{
		{"falls Prone", resSpec + "FR-065", resPC(t, "a", combat.SideA), []string{"tracker_turn_ended", "fell_prone_in_shared_space", "tracker_turn_started", "condition_applied"}, true},
		{"already Prone", resSpec + "FR-065", resPC(t, "a", combat.SideA, combat.Prone{}), []string{"tracker_turn_ended", "fell_prone_in_shared_space", "tracker_turn_started"}, true},
		{"Immune to Prone", resSpec + "FR-065", resImmune(t, "a", combat.SideA, content.ConditionProne), []string{"tracker_turn_ended", "fell_prone_in_shared_space", "tracker_turn_started"}, false},
		{"dead", resSpec + "FR-065", resDead(t, "a", combat.SideA), []string{"tracker_turn_ended", "fell_prone_in_shared_space", "tracker_turn_started"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, _ := encStart(t, c.active, resPC(t, "b", combat.SideB))
			res, err := resAdvanceRun(t, e, combat.ActionLedger{}, nil, occupants)
			if err != nil {
				t.Fatal(err)
			}
			got := resAs[combat.AdvanceCombatTurnCompleted](t, res)
			if !slices.Equal(resKinds(got.Events()), c.events) || resHas(t, got.Encounter(), "a", content.ConditionProne) != c.prone {
				t.Errorf("events %v, Prone %v; want %v and %v [%s]", resKinds(got.Events()), resHas(t, got.Encounter(), "a", content.ConditionProne), c.events, c.prone, c.requirement)
			}
		})
	}
	// Alone in its space: nothing falls.
	got := resAdvance(t, encABC(t), combat.ActionLedger{})
	if slices.Contains(resKinds(got.Events()), "fell_prone_in_shared_space") {
		t.Errorf("a creature alone in its space fell Prone")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-064, FR-011, FR-128, FR-009 - an update naming a combatant the encounter does not hold is an invalid result naming the smallest such id, with no event and the encounter and ledger unchanged; known updates pass through as MovementSpeedsUpdated; an ended encounter is refused first.
// Path: per case.
func TestAdvanceCombatTurnUpdatesAndStops(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	ledger := combat.ActionLedger{}
	fast := trkSpeeds(t, 40)
	res, err := resAdvanceRun(t, e, ledger, []combat.SpeedUpdate{{Combatant: cid("zz"), Speeds: fast}, {Combatant: cid("ghost"), Speeds: fast}, {Combatant: cid("b"), Speeds: fast}}, nil)
	inv := resAs[combat.AdvanceCombatTurnInvalid](t, res)
	if err != nil || inv.Problem() != (combat.UnknownCombatantProblem{ID: cid("ghost")}) || !inv.Encounter().Equal(e) || !inv.Ledger().Equal(ledger) || len(inv.Events()) != 0 {
		t.Errorf("= %#v, %v, want an invalid result naming ghost, the smallest id, with nothing changed", res, err)
	}
	res, err = resAdvanceRun(t, e, ledger, []combat.SpeedUpdate{{Combatant: cid("b"), Speeds: fast}}, nil)
	got := resAs[combat.AdvanceCombatTurnCompleted](t, res)
	if err != nil || !slices.Contains(resKinds(got.Events()), "movement_speeds_updated") {
		t.Errorf("events = %v, %v, want MovementSpeedsUpdated passed through", resKinds(got.Events()), err)
	}
	ended := resEnded(t, e)
	res, err = resAdvanceRun(t, ended, ledger, []combat.SpeedUpdate{{Combatant: cid("ghost"), Speeds: fast}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.AdvanceCombatTurnRejected](t, res), ended, ledger, combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil)
}

// Requirement: doc/specs/combat-action-resolution.md FR-096, FR-004 - the command refuses a zero encounter, a blank id in an update and a size that is not defined, and takes its own copy of the lists.
// Path: unhappy (invalid input).
func TestNewAdvanceCombatTurnCommand(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	fast := trkSpeeds(t, 40)
	for name, c := range map[string]struct {
		enc       combat.Encounter
		updates   []combat.SpeedUpdate
		occupants []content.CreatureSize
	}{
		"zero encounter": {combat.Encounter{}, nil, nil},
		"blank id":       {e, []combat.SpeedUpdate{{Speeds: fast}}, nil},
		"undefined size": {e, nil, []content.CreatureSize{content.CreatureSize(99)}},
	} {
		if _, err := combat.NewAdvanceCombatTurnCommand(c.enc, combat.ActionLedger{}, c.updates, c.occupants); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	updates := []combat.SpeedUpdate{{Combatant: cid("b"), Speeds: fast}}
	cmd, err := combat.NewAdvanceCombatTurnCommand(e, combat.ActionLedger{}, updates, nil)
	if err != nil {
		t.Fatal(err)
	}
	updates[0].Combatant = cid("ghost")
	res, err := combat.NewAdvanceCombatTurn().Run(ctxBackground(), cmd)
	if _, ok := res.(combat.AdvanceCombatTurnCompleted); !ok || err != nil {
		t.Errorf("= %T, %v, want the copy taken at construction to be used", res, err)
	}
}
