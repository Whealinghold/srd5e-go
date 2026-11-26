package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewPerformUnarmedStrikeCommand(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	opportunity := func(provoker string, v combat.Visibility) combat.AttackCost {
		return combat.ReactionCost{Choice: combat.OpportunityAttackReaction{Provoker: cid(provoker), Means: combat.DepartureBySpeed, Visibility: v}}
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*combat.PerformUnarmedStrikeParams)
		ok                      bool
	}{
		{"valid", resSpec + "FR-042", "happy", func(*combat.PerformUnarmedStrikeParams) {}, true},
		{"zero encounter", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.Encounter = combat.Encounter{} }, false},
		{"blank attacker", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.AttackerID = "" }, false},
		{"blank target", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.TargetID = "" }, false},
		{"zero Strike command", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.Strike = combat.ResolveUnarmedStrikeCommand{} }, false},
		{"scene distance differs from the strike's", resSpec + "FR-135", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.Scene = resSeen(t, 10) }, false},
		{"undefined knock out choice", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.KnockOut = combat.KnockOutChoice(4) }, false},
		{"nil adjustment", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.DamageAdjustments = []combat.DamageAdjustment{nil} }, false},
		{"nil cost", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.Cost = nil }, false},
		{"weapon means with an Unarmed Strike", resSpec + "FR-142", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) {
			p.Cost = combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansOtherWeapon}
		}, false},
		{"Extra Attack with the unarmed means", resSpec + "FR-142", "happy", func(p *combat.PerformUnarmedStrikeParams) {
			p.Cost = combat.ContinuesAttackAction{Means: combat.AttackMeansUnarmedStrike}
		}, true},
		{"Opportunity Attack by Unarmed Strike", resSpec + "FR-042, FR-024", "happy", func(p *combat.PerformUnarmedStrikeParams) { p.Cost = opportunity("b", combat.VisibilitySeen) }, true},
		{"Opportunity Attack against another creature than its provoker", resSpec + "FR-138", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.Cost = opportunity("c", combat.VisibilitySeen) }, false},
		{"Opportunity Attack with the wrong Visibility", resSpec + "FR-140", "unhappy (invalid input)", func(p *combat.PerformUnarmedStrikeParams) { p.Cost = opportunity("b", combat.VisibilityUnseen) }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resStrikeParams(t, e, "a", "b", grappleOption(t, ability.Strength, strengthSave(t)))
			c.edit(&p)
			_, err := combat.NewPerformUnarmedStrikeCommand(p)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("error = %v, want ok=%v with ErrInvalidInput on refusal [%s]", err, c.ok, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-042, FR-024, FR-116, FR-010 - an Unarmed Strike takes the stops of an attack: an Opportunity Attack by Reaction completes without touching the Attack action, a Reaction already spent, a limit reached, an attacker Charmed by the target, a target behind Total Cover and a hit's Help benefit each end or shape the call as for PerformAttack.
// Path: per case.
func TestPerformUnarmedStrikeSharesTheStopsOfAnAttack(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	oa := combat.ReactionCost{Choice: combat.OpportunityAttackReaction{Provoker: cid("a"), Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}}
	p := resStrikeParams(t, e, "b", "a", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	p.Cost = oa
	got, _ := resStrike(t, p, 14)
	if want := []string{"reaction_taken", "unarmed_strike_performed", "damage_taken"}; !slices.Equal(resKinds(got.Events()), want) || !got.Encounter().Tracker().ReactionSpent(cid("b")) ||
		len(got.Resolutions()) != 1 || got.Resolutions()[0].Dealt() != 4 {
		t.Errorf("events %v, resolutions %v, want %v and 4 dealt by the Opportunity Attack", resKinds(got.Events()), got.Resolutions(), want)
	}
	again := resStrikeParams(t, got.Encounter(), "b", "a", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	again.Cost = oa
	cmd, _ := combat.NewPerformUnarmedStrikeCommand(again)
	uc, _ := combat.NewPerformUnarmedStrike(newFixed(14))
	res, err := uc.Run(ctxBackground(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.PerformUnarmedStrikeRejected](t, res), got.Encounter(), combat.ActionLedger{}, combat.TrackerRejection{Rejection: combat.TurnReactionAlreadyTakenRejection{}}, nil)

	begun, _ := resStrike(t, resStrikeParams(t, e, "a", "b", combat.UnarmedStrikeDamageOption{ArmorClass: 15}), 14)
	more := resStrikeParams(t, begun.Encounter(), "a", "b", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	more.Ledger = begun.Ledger()
	more.Cost = combat.ContinuesAttackAction{Means: combat.AttackMeansUnarmedStrike}
	cmd, _ = combat.NewPerformUnarmedStrikeCommand(more)
	res, err = uc.Run(ctxBackground(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.PerformUnarmedStrikeRejected](t, res), begun.Encounter(), begun.Ledger(),
		combat.LedgerRejection{Refusal: combat.LedgerAttackRefused{Reason: combat.AttackRefusalLimitReached}}, nil)

	charmed, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: cid("b")}}), resPC(t, "b", combat.SideB))
	cmd, _ = combat.NewPerformUnarmedStrikeCommand(resStrikeParams(t, charmed, "a", "b", grappleOption(t, ability.Strength, strengthSave(t))))
	res, err = uc.Run(ctxBackground(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.PerformUnarmedStrikeRejected](t, res), charmed, combat.ActionLedger{}, combat.TargetingForbiddenRejection{}, nil)

	// Help: the Damage option consumes a pending benefit and rolls with Advantage.
	three, _ := encStart(t, resPC(t, "c", combat.SideA), resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	helped := resTake(t, three, combat.ActionLedger{}, "c", combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("b"), Target: combat.AttackAssistTargetEnemyWithinFiveFeet}})
	next := resAdvance(t, helped.Encounter(), helped.Ledger())
	allied, _ := combat.NewAlliedHelpers(cid("c"))
	hp := resStrikeParams(t, next.Encounter(), "a", "b", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	hp.Ledger, hp.AlliedHelpers = next.Ledger(), allied
	done, dice := resStrike(t, hp, 3, 14)
	if !slices.Equal(dice, []string{"2d20"}) || done.Ledger().HelpBenefits().Len() != 0 || !slices.Contains(resKinds(done.Events()), "help_benefit_used") {
		t.Errorf("dice %v, benefits %d, events %v, want Advantage and the benefit spent", dice, done.Ledger().HelpBenefits().Len(), resKinds(done.Events()))
	}
}
