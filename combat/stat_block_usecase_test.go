package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sbEncounter starts an encounter in which the fixture monster (id luSelf) is the active combatant against "b".
func sbEncounter(tb testing.TB, conds ...combat.Condition) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encC(tb, luSelf, combat.SideB, pcStatus(tb, 20, 20, 0), conds...), encPC(tb, "b", combat.SideA, 20, 20))
	return e
}

func sbTake(tb testing.TB, e combat.Encounter, l combat.ActionLedger, req combat.TurnRequest) combat.TakeCombatantActionResult {
	tb.Helper()
	cmd, err := combat.NewTakeCombatantActionCommand(e, l, cid(luSelf), req)
	if err != nil {
		tb.Fatalf("NewTakeCombatantActionCommand: %v", err)
	}
	res, err := combat.NewTakeCombatantAction().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res
}

// Requirement: doc/specs/combat-action-resolution.md FR-164, FR-091 - using a stat block part spends the economy it names through the tracker (the action for an Actions part, the Bonus Action, the Reaction; nothing for a Legendary Action) and StatBlockPartUsed follows the tracker's events; the new LimitedUseState is carried by the result.
// Path: per case.
func TestTakeCombatantActionUsesStatBlockParts(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	req := func(cost combat.StatBlockActionCost, ref combat.LimitedUsePartRef) combat.UseStatBlockPart {
		return combat.UseStatBlockPart{Monster: m, Request: sbRequest(cost, ref), LimitedUse: state}
	}
	legendary := combat.UseStatBlockPart{Monster: m, Request: sbLegendaryRequest(t, sbRef(t, m, combat.MonsterPartLegendaryActions, "Tail Swipe"), "b"), LimitedUse: state}
	cases := []struct {
		name, requirement, path string
		request                 combat.TurnRequest
		kinds                   []string
	}{
		{"an action", resSpec + "FR-164; SRD 5.2.1, \"Actions\", p. 256", "happy", req(combat.StatBlockCostAction, sbAction(t, m, "Slam")), []string{"action_taken", "stat_block_part_used"}},
		{"a Bonus Action", resSpec + "FR-164; SRD 5.2.1, \"Bonus Action\", p. 257", "happy", req(combat.StatBlockCostBonusAction, sbRef(t, m, combat.MonsterPartBonusActions, "Boar Rush")), []string{"bonus_action_used", "stat_block_part_used"}},
		{"a Reaction", resSpec + "FR-164; SRD 5.2.1, \"Reactions\", p. 257", "happy", req(combat.StatBlockCostReaction, sbRef(t, m, combat.MonsterPartReactions, "Unnerving Gaze")), []string{"reaction_taken", "stat_block_part_used"}},
		{"a Legendary Action spends no tracker resource", resSpec + "FR-164; SRD 5.2.1, \"Legendary Actions\", p. 257", "unhappy (rule interaction)", legendary, []string{"legendary_action_taken", "stat_block_part_used"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := resAs[combat.TakeCombatantActionCompleted](t, sbTake(t, sbEncounter(t), combat.ActionLedger{}, c.request))
			if kinds := resKinds(got.Events()); !slices.Equal(kinds, c.kinds) {
				t.Errorf("events = %v, want %v [%s]", kinds, c.kinds, c.requirement)
			}
			if lu, ok := got.LimitedUse(); !ok || lu.IsZero() {
				t.Errorf("LimitedUse() = %v, %v, want the new state [%s]", lu, ok, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-164, FR-010 step 4a, FR-009 - a refused use (a part not in the section, a Reaction whose trigger was not reported, a part whose usage is spent) is StatBlockRejection before the tracker, so no resource is spent; a tracker rejection (Incapacitated) is a TrackerRejection that leaves the limited-use state to the host; both leave encounter and ledger as they came.
// Path: per case.
func TestTakeCombatantActionStatBlockRefusals(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	dominate := sbAction(t, m, "Dominate Mind")
	spent := luSpent(t, luSpent(t, state, dominate), dominate)
	noTrigger := sbRequest(combat.StatBlockCostReaction, sbRef(t, m, combat.MonsterPartReactions, "Unnerving Gaze"))
	noTrigger.Trigger = combat.ReactionTriggerNotOccurred
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		req                     combat.UseStatBlockPart
		want                    combat.ResolutionRejection
	}{
		{"usage spent", resSpec + "FR-164", "unhappy (state)", sbEncounter(t),
			combat.UseStatBlockPart{Monster: m, Request: sbRequest(combat.StatBlockCostAction, dominate), LimitedUse: spent}, combat.StatBlockRejection{Refusal: combat.StatBlockPartUsageExhausted{}}},
		{"trigger not reported", resSpec + "FR-164", "unhappy (state)", sbEncounter(t),
			combat.UseStatBlockPart{Monster: m, Request: noTrigger, LimitedUse: state}, combat.StatBlockRejection{Refusal: combat.StatBlockReactionTriggerNotReported{}}},
		{"part in another section", resSpec + "FR-164", "unhappy (invalid input)", sbEncounter(t),
			combat.UseStatBlockPart{Monster: m, Request: sbRequest(combat.StatBlockCostBonusAction, sbAction(t, m, "Slam")), LimitedUse: state}, combat.StatBlockRejection{Refusal: combat.StatBlockPartInWrongSection{}}},
		{"Incapacitated monster", resSpec + "FR-164, FR-055", "unhappy (rule interaction)", sbEncounter(t, combat.Stunned{}),
			combat.UseStatBlockPart{Monster: m, Request: sbRequest(combat.StatBlockCostAction, sbAction(t, m, "Slam")), LimitedUse: state}, combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := sbTake(t, c.e, combat.ActionLedger{}, c.req)
			resWant(t, res, c.e, combat.ActionLedger{}, c.want, nil)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-165, FR-113, 4.18.1 Multiattack - a Multiattack is the monster's Attack action: BeginMultiattack spends the action once and keeps the progress in the ledger; each attack is a PerformAttack with RecordsMultiattack, which asks the tracker for nothing and records the attack; the attack past the plan's allowance is LedgerMultiattackRefused with no die; a monster with no plan is StatBlockRejection(MultiattackNotAvailable); a use or replacement with no Multiattack is refused; AdvanceCombatTurn drops the progress.
// Path: unhappy (state).
func TestMultiattackThroughTheLedger(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil))
	m := sbMonster(t, &plan, sbPlain("Slam"), nil)
	state := luStart(t, m, combat.OutsideLair)
	e := sbEncounter(t)

	begun := resAs[combat.TakeCombatantActionCompleted](t, sbTake(t, e, combat.ActionLedger{}, combat.BeginMultiattack{Monster: m, LimitedUse: state}))
	if kinds := resKinds(begun.Events()); !slices.Equal(kinds, []string{"action_taken", "stat_block_part_used"}) {
		t.Errorf("events = %v, want the action and the use of the Multiattack [FR-165]", kinds)
	}
	if _, in := begun.Ledger().Multiattack().(combat.MultiattackInProgress); !in {
		t.Fatalf("Multiattack = %#v, want one in progress", begun.Ledger().Multiattack())
	}
	e, ledger, lu := begun.Encounter(), begun.Ledger(), state
	if got, ok := begun.LimitedUse(); ok {
		lu = got
	}
	slam := sbAction(t, m, "Slam")
	attack := func(faces ...int) (combat.PerformAttackResult, []string) {
		p := resParams(t, e, luSelf, "b", combat.RecordsMultiattack{Part: slam, LimitedUse: lu})
		p.Ledger = ledger
		res, roller, err := resPerform(t, p, faces...)
		if err != nil {
			t.Fatal(err)
		}
		return res, notations(roller)
	}
	for i := range 2 {
		res, dice := attack(2)
		done := resAs[combat.PerformAttackCompleted](t, res)
		if kinds := resKinds(done.Events()); kinds[0] != "multiattack_attack_recorded" || slices.Contains(kinds, "action_taken") || len(dice) != 1 {
			t.Errorf("attack %d: events %v, dice %v, want a recorded Multiattack attack with no tracker event [FR-165]", i+1, kinds, dice)
		}
		e, ledger = done.Encounter(), done.Ledger()
		if got, ok := done.LimitedUse(); ok {
			lu = got
		}
	}
	res, dice := attack(2)
	resWant(t, res, e, ledger, combat.LedgerRejection{Refusal: combat.LedgerMultiattackRefused{Outcome: combat.MultiattackNotInAllowance{Reason: combat.MultiattackAllowanceSpent}}}, nil)
	if len(dice) != 0 {
		t.Errorf("dice = %v, want none for a refused attack [FR-113]", dice)
	}

	use := resAs[combat.TakeCombatantActionRejected](t, sbTake(t, e, ledger, combat.RecordMultiattackUse{Part: slam, LimitedUse: lu}))
	if _, ok := use.Rejection().(combat.LedgerRejection).Refusal.(combat.LedgerMultiattackRefused); !ok {
		t.Errorf("an additional use the plan does not list gives %#v, want LedgerMultiattackRefused", use.Rejection())
	}
	next := resAdvance(t, e, ledger)
	if _, none := next.Ledger().Multiattack().(combat.NoMultiattack); !none {
		t.Errorf("Multiattack after the turn = %#v, want none [FR-165]", next.Ledger().Multiattack())
	}
	resWant(t, sbTake(t, e, combat.ActionLedger{}, combat.RecordMultiattackUse{Part: slam, LimitedUse: lu}), e, combat.ActionLedger{}, combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}, nil)
	noPlan := sbBeast(t)
	resWant(t, sbTake(t, sbEncounter(t), combat.ActionLedger{}, combat.BeginMultiattack{Monster: noPlan, LimitedUse: luStart(t, noPlan, combat.OutsideLair)}),
		sbEncounter(t), combat.ActionLedger{}, combat.StatBlockRejection{MultiattackNotAvailable: true}, nil)
}

// Requirement: doc/specs/combat-action-resolution.md FR-165, FR-112 - the Multiattack state of the ledger: the zero ledger holds none, WithMultiattack accepts none and a progress and rejects a zero one or another variant, Equal compares it, and the ledger it returns is a new value.
// Path: unhappy (invalid input).
func TestActionLedgerMultiattackState(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil))
	m := sbMonster(t, &plan, sbPlain("Slam"), nil)
	started := luMust(combat.StartMultiattack(m, sbSelf(), luStart(t, m, combat.OutsideLair))).(combat.MultiattackStarted)
	l := combat.ActionLedger{}
	if _, none := l.Multiattack().(combat.NoMultiattack); !none {
		t.Errorf("zero ledger Multiattack = %#v, want none", l.Multiattack())
	}
	with, err := l.WithMultiattack(combat.MultiattackInProgress{Progress: started.Progress()})
	if err != nil || with.Equal(l) || !l.Equal(combat.ActionLedger{}) {
		t.Errorf("WithMultiattack = %v, equal to the old %v, the old changed %v, want a new ledger", err, with.Equal(l), !l.Equal(combat.ActionLedger{}))
	}
	for name, s := range map[string]combat.MultiattackState{"zero progress": combat.MultiattackInProgress{}, "unknown": struct{ combat.MultiattackState }{}} {
		if _, err := l.WithMultiattack(s); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("WithMultiattack(%s) = %v, want ErrInvalidInput", name, err)
		}
	}
	if back, err := with.WithMultiattack(nil); err != nil || !back.Equal(l) {
		t.Errorf("WithMultiattack(nil) = %v, %v, want the ledger with none", back, err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-165, monster-stat-block-actions.md FR-018 - a Multiattack in progress takes a replacement of an attack (RecordMultiattackReplacement) through the ledger, spending no tracker resource and carrying the new limited-use state; a replacement the plan does not list is LedgerMultiattackRefused.
// Path: per case.
func TestMultiattackReplacementThroughTheLedger(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(3, "Spear")), nil, sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbOption("Shield Bash"))))
	m := sbMonster(t, &plan, []content.StatBlockPart{sbPart("Spear", content.StatBlockPartFields{}), sbPart("Shield Bash", content.StatBlockPartFields{})}, nil)
	state := luStart(t, m, combat.OutsideLair)
	begun := resAs[combat.TakeCombatantActionCompleted](t, sbTake(t, sbEncounter(t), combat.ActionLedger{}, combat.BeginMultiattack{Monster: m, LimitedUse: state}))
	spear := sbAction(t, m, "Spear")
	replace := combat.RecordMultiattackReplacement{Option: sbOption("Shield Bash"), Replaced: spear, LimitedUse: state}
	done := resAs[combat.TakeCombatantActionCompleted](t, sbTake(t, begun.Encounter(), begun.Ledger(), replace))
	if kinds := resKinds(done.Events()); !slices.Equal(kinds, []string{"multiattack_attack_replaced"}) {
		t.Errorf("events = %v, want the replacement and no tracker event [FR-165]", kinds)
	}
	if _, ok := done.LimitedUse(); !ok {
		t.Errorf("LimitedUse() reports no new state [FR-165]")
	}
	again := resAs[combat.TakeCombatantActionRejected](t, sbTake(t, done.Encounter(), done.Ledger(), replace))
	if _, ok := again.Rejection().(combat.LedgerRejection).Refusal.(combat.LedgerMultiattackRefused); !ok {
		t.Errorf("a second replacement of a once rule gives %#v, want LedgerMultiattackRefused [FR-165]", again.Rejection())
	}
	if (combat.LedgerMultiattackRefused{Outcome: combat.MultiattackStepUsageRefused{}}).String() != "multiattack_step_usage_refused" {
		t.Errorf("the name of a usage refusal is wrong")
	}
}
