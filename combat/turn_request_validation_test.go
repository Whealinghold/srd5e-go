package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-action-resolution.md FR-164, FR-165 - NewTakeCombatantActionCommand refuses a stat block use or a Multiattack request that lacks its monster, part or limited-use state, and accepts the same request when complete.
// Path: per case.
func TestNewTakeCombatantActionCommandStatBlockRequests(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil))
	m := sbMonster(t, &plan, sbPlain("Slam"), nil)
	state := luStart(t, m, combat.OutsideLair)
	slam := sbAction(t, m, "Slam")
	e := sbEncounter(t)
	cases := []struct {
		name, requirement, path string
		request                 combat.TurnRequest
		valid                   bool
	}{
		{"use with no monster", resSpec + "FR-164", "unhappy (invalid input)", combat.UseStatBlockPart{LimitedUse: state}, false},
		{"use with no part", resSpec + "FR-164", "unhappy (invalid input)", combat.UseStatBlockPart{Monster: m, LimitedUse: state}, false},
		{"use with no limited-use state", resSpec + "FR-164", "unhappy (invalid input)", combat.UseStatBlockPart{Monster: m, Request: combat.StatBlockUseRequest{Part: slam}}, false},
		{"Multiattack with no monster", resSpec + "FR-165", "unhappy (invalid input)", combat.BeginMultiattack{LimitedUse: state}, false},
		{"Multiattack with no limited-use state", resSpec + "FR-165", "unhappy (invalid input)", combat.BeginMultiattack{Monster: m}, false},
		{"Multiattack complete", resSpec + "FR-165", "happy", combat.BeginMultiattack{Monster: m, LimitedUse: state}, true},
		{"additional use with no part", resSpec + "FR-165", "unhappy (invalid input)", combat.RecordMultiattackUse{LimitedUse: state}, false},
		{"additional use with no state", resSpec + "FR-165", "unhappy (invalid input)", combat.RecordMultiattackUse{Part: slam}, false},
		{"additional use complete", resSpec + "FR-165", "happy", combat.RecordMultiattackUse{Part: slam, LimitedUse: state}, true},
		{"replacement with no replaced part", resSpec + "FR-165", "unhappy (invalid input)", combat.RecordMultiattackReplacement{LimitedUse: state}, false},
		{"replacement with no state", resSpec + "FR-165", "unhappy (invalid input)", combat.RecordMultiattackReplacement{Replaced: slam}, false},
		{"replacement complete", resSpec + "FR-165", "happy", combat.RecordMultiattackReplacement{Replaced: slam, LimitedUse: state}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewTakeCombatantActionCommand(e, combat.ActionLedger{}, cid(luSelf), c.request)
			if c.valid != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("err = %v, want valid = %v and ErrInvalidInput otherwise [%s]", err, c.valid, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-165, FR-137 - the cost of an attack is checked when the command is built: a Multiattack attack needs its part and limited-use state, and an attack by a Reaction that is not an Opportunity Attack needs nothing more than a valid choice.
// Path: per case.
func TestNewPerformAttackCommandCosts(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil))
	m := sbMonster(t, &plan, sbPlain("Slam"), nil)
	slam := sbAction(t, m, "Slam")
	e := sbEncounter(t)
	feature := combat.ReactionCost{Choice: combat.FeatureReaction{Feature: "riposte", Granted: combat.NoStandardAction{}}}
	for _, c := range []struct {
		name, requirement, path string
		cost                    combat.AttackCost
		valid                   bool
	}{
		{"Multiattack with no part", resSpec + "FR-165", "unhappy (invalid input)", combat.RecordsMultiattack{LimitedUse: luStart(t, m, combat.OutsideLair)}, false},
		{"Multiattack with no state", resSpec + "FR-165", "unhappy (invalid input)", combat.RecordsMultiattack{Part: slam}, false},
		{"Multiattack complete", resSpec + "FR-165", "happy", combat.RecordsMultiattack{Part: slam, LimitedUse: luStart(t, m, combat.OutsideLair)}, true},
		{"Reaction by a feature", resSpec + "FR-137", "happy", feature, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewPerformAttackCommand(resParams(t, e, luSelf, "b", c.cost))
			if c.valid != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("err = %v, want valid = %v and ErrInvalidInput otherwise [%s]", err, c.valid, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-165, FR-164 - a Multiattack that names a granted standard action spends the action with that grant (the ActionTaken event carries it), and an attack that records a Multiattack with none in progress is refused with no die and nothing changed.
// Path: unhappy (state).
func TestBeginMultiattackGrantedActionAndNoneInProgress(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil))
	m := sbMonster(t, &plan, sbPlain("Slam"), nil)
	state := luStart(t, m, combat.OutsideLair)
	e := sbEncounter(t)
	grant := combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDodge}
	begun := resAs[combat.TakeCombatantActionCompleted](t, sbTake(t, e, combat.ActionLedger{}, combat.BeginMultiattack{Monster: m, LimitedUse: state, Granted: grant}))
	var granted combat.GrantedAction
	for _, ev := range begun.Events() {
		if turn, ok := ev.(combat.TurnEventOccurred); ok {
			if taken, ok := turn.Event().(combat.ActionTaken); ok {
				granted = taken.Choice().(combat.FeatureActionChoice).Granted
			}
		}
	}
	if granted != combat.GrantedAction(grant) {
		t.Errorf("granted action = %#v, want %#v [FR-165]", granted, grant)
	}
	p := resParams(t, e, luSelf, "b", combat.RecordsMultiattack{Part: sbAction(t, m, "Slam"), LimitedUse: state})
	res, roller, err := resPerform(t, p, 14, 4)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, res, e, combat.ActionLedger{}, combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}, roller)
}
