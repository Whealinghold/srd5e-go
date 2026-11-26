package combat_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func resTakeRun(tb testing.TB, e combat.Encounter, l combat.ActionLedger, who string, req combat.TurnRequest) (combat.TakeCombatantActionResult, error) {
	tb.Helper()
	cmd, err := combat.NewTakeCombatantActionCommand(e, l, cid(who), req)
	if err != nil {
		tb.Fatalf("NewTakeCombatantActionCommand: %v", err)
	}
	return combat.NewTakeCombatantAction().Run(ctxBackground(), cmd)
}

// Requirement: doc/specs/combat-action-resolution.md FR-051, FR-052, FR-053, FR-091 - each request spends its resource through the tracker, commits it through the encounter and returns the tracker's events: the action (Dodge, Dash, Disengage, Attack, which resolves nothing), the Bonus Action a rule grants and the free interaction.
// Path: per case.
func TestTakeCombatantActionSpendsTheResource(t *testing.T) {
	t.Parallel()
	grant, err := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "cunning_action"}, combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDisengage})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, requirement, path string
		request                 combat.TurnRequest
		events                  []string
	}{
		{"Dodge", resSpec + "FR-053", "happy", combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}}, []string{"action_taken"}},
		{"Disengage", resSpec + "FR-053", "happy", combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDisengage}}, []string{"action_taken"}},
		{"Dash", resSpec + "FR-053", "happy", combat.SpendAction{Choice: combat.DashChoice{Speed: combat.SpeedNameWalk}}, []string{"action_taken"}},
		{"Attack resolves nothing", resSpec + "FR-052", "unhappy (rule interaction)", combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionAttack}}, []string{"action_taken"}},
		{"Bonus Action with a grant", resSpec + "FR-051", "happy", combat.SpendBonusAction{Grant: grant}, []string{"bonus_action_used"}},
		{"free interaction", resSpec + "FR-051", "happy", combat.SpendInteraction{}, []string{"object_interacted"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := encABC(t)
			got := resTake(t, e, combat.ActionLedger{}, "a", c.request)
			if !slices.Equal(resKinds(got.Events()), c.events) {
				t.Errorf("events = %v, want %v [%s]", resKinds(got.Events()), c.events, c.requirement)
			}
			if got.Encounter().Tracker().Equal(e.Tracker()) || !got.Ledger().Equal(combat.ActionLedger{}) {
				t.Errorf("the tracker must change and the ledger must not [FR-052]")
			}
			for _, id := range []string{"a", "b", "c"} {
				if !encMember(t, e, id).Equal(encMember(t, got.Encounter(), id)) {
					t.Errorf("combatant %s changed: an action resolves no effect [FR-052]", id)
				}
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-016, FR-055, FR-054, 4.2 - an Incapacitated combatant, whether by the condition or by Unconscious, Paralyzed, Petrified or Stunned, is refused any action, Bonus Action or Reaction through TrackerRejection(incapacitated), read from its own conditions; a Reaction may be taken by a combatant that is not the active one and changes nothing of the active turn.
// Path: per case.
func TestTakeCombatantActionIncapacitatedAndReactions(t *testing.T) {
	t.Parallel()
	grant, _ := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "cunning_action"}, combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDisengage})
	choice := combat.FeatureReaction{Feature: content.MustFeatureID("uncanny_dodge"), Granted: combat.NoStandardAction{}}
	for _, cond := range []combat.Condition{combat.Incapacitated{}, combat.Unconscious{}, combat.Paralyzed{}, combat.Petrified{}, combat.Stunned{}} {
		e, _ := encStart(t, resPC(t, "a", combat.SideA, cond), resPC(t, "b", combat.SideB))
		for name, req := range map[string]combat.TurnRequest{
			"action":       combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}},
			"Bonus Action": combat.SpendBonusAction{Grant: grant},
			"Reaction":     combat.SpendReaction{Choice: choice},
		} {
			res, err := resTakeRun(t, e, combat.ActionLedger{}, "a", req)
			if err != nil {
				t.Fatal(err)
			}
			resWant(t, resAs[combat.TakeCombatantActionRejected](t, res), e, combat.ActionLedger{}, combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}, nil)
			_ = name
		}
	}
	e := encABC(t)
	got := resTake(t, e, combat.ActionLedger{}, "b", combat.SpendReaction{Choice: choice})
	if want := []string{"reaction_taken"}; !slices.Equal(resKinds(got.Events()), want) {
		t.Errorf("events = %v, want %v", resKinds(got.Events()), want)
	}
	if g, w := got.Encounter().Tracker(), e.Tracker(); g.Active() != w.Active() || g.Round() != w.Round() || g.Resources() != w.Resources() || !g.ReactionSpent(cid("b")) {
		t.Errorf("the active combatant, round and resources must be unchanged and the Reaction spent [FR-054]")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-051, FR-010, FR-054 - the tracker's refusals end the call with TrackerRejection and nothing changes: an action spent, a Bonus Action with no grant, a second interaction, a combatant that is not the active one acting, a dead combatant, an unknown one.
// Path: per case.
func TestTakeCombatantActionRefusals(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	dodged := resDodge(t, e, combat.ActionLedger{}, "a")
	touched := resTake(t, e, combat.ActionLedger{}, "a", combat.SpendInteraction{})
	deadB, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	dodge := combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}}
	reaction := combat.SpendReaction{Choice: combat.FeatureReaction{Feature: content.MustFeatureID("uncanny_dodge"), Granted: combat.NoStandardAction{}}}
	cases := []struct {
		name, requirement string
		e                 combat.Encounter
		who               string
		req               combat.TurnRequest
		want              combat.ResolutionRejection
		problem           combat.ResolutionProblem
	}{
		{"action already spent", resSpec + "FR-051", dodged.Encounter(), "a", dodge, combat.TrackerRejection{Rejection: combat.TurnActionAlreadyTakenRejection{}}, nil},
		{"Bonus Action with no grant", resSpec + "FR-051", e, "a", combat.SpendBonusAction{}, combat.TrackerRejection{Rejection: combat.TurnNoBonusActionGrantedRejection{}}, nil},
		{"second free interaction", resSpec + "FR-051", touched.Encounter(), "a", combat.SpendInteraction{}, combat.TrackerRejection{Rejection: combat.TurnNeedsUtilizeActionRejection{}}, nil},
		{"action by a combatant that is not active", resSpec + "FR-051", e, "b", dodge, combat.TrackerRejection{Rejection: combat.TurnNotActiveCombatantRejection{}}, nil},
		{"Reaction by a dead combatant", resSpec + "FR-054, FR-012", deadB, "b", reaction, combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil},
		{"ended", resSpec + "FR-010", resEnded(t, e), "a", dodge, combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil},
		{"unknown combatant", resSpec + "FR-011", e, "ghost", dodge, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := resTakeRun(t, c.e, combat.ActionLedger{}, c.who, c.req)
			if err != nil {
				t.Fatal(err)
			}
			if c.problem != nil {
				if inv := resAs[combat.TakeCombatantActionInvalid](t, res); inv.Problem() != c.problem || !inv.Encounter().Equal(c.e) || len(inv.Events()) != 0 {
					t.Errorf("problem %#v, want %#v with nothing changed [%s]", inv.Problem(), c.problem, c.requirement)
				}
				return
			}
			resWant(t, resAs[combat.TakeCombatantActionRejected](t, res), c.e, combat.ActionLedger{}, c.want, nil)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-117, FR-119, 4.18.1 Help - Help gives the ledger a benefit and spends the action; a refusal (the helper lacks the proficiency, helps itself, the task allows no help, the enemy is beyond 5 feet or is not an enemy) stops the call before the action is spent, with the facts the host stated as the only input.
// Path: per case.
func TestTakeCombatantActionHelp(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	medicine := combat.AssistedSkill(content.SkillMedicine)
	with, _ := combat.NewHelpGiverProficiencies([]content.Skill{content.SkillMedicine}, nil)
	without, _ := combat.NewHelpGiverProficiencies(nil, nil)
	check := func(ally string, p combat.HelpGiverProficiencies, j combat.AssistanceJudgement) combat.TurnRequest {
		return combat.TakeHelpAction{Request: combat.AbilityCheckHelpRequest{Ally: cid(ally), Proficiency: medicine, HelperProficiencies: p, Judgement: j}}
	}
	attackHelp := func(target combat.AttackAssistTarget) combat.TurnRequest {
		return combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("b"), Target: target}}
	}
	cases := []struct {
		name, requirement, path string
		request                 combat.TurnRequest
		refusal                 combat.LedgerRefusal
	}{
		{"ability check help", resSpec + "FR-117", "happy", check("c", with, combat.AssistanceJudgementPossible), nil},
		{"attack roll help, enemy within 5 feet", resSpec + "FR-117, 4.18.1", "happy", attackHelp(combat.AttackAssistTargetEnemyWithinFiveFeet), nil},
		{"helper lacks proficiency", resSpec + "FR-117", "unhappy (rule interaction)", check("c", without, combat.AssistanceJudgementPossible),
			combat.LedgerAbilityCheckHelpRefused{Reason: combat.AbilityCheckHelpRefusalHelperLacksProficiency}},
		{"helps itself", resSpec + "FR-117", "unhappy (rule interaction)", check("a", with, combat.AssistanceJudgementPossible), combat.LedgerAbilityCheckHelpRefused{Reason: combat.AbilityCheckHelpRefusalCannotHelpSelf}},
		{"assistance not possible", resSpec + "FR-117", "unhappy (rule interaction)", check("c", with, combat.AssistanceJudgementNotPossible),
			combat.LedgerAbilityCheckHelpRefused{Reason: combat.AbilityCheckHelpRefusalAssistanceNotPossible}},
		{"enemy beyond 5 feet", resSpec + "FR-117, 4.18.1", "unhappy (boundary)", attackHelp(combat.AttackAssistTargetEnemyBeyondFiveFeet),
			combat.LedgerAttackRollHelpRefused{Reason: combat.AttackRollHelpRefusalEnemyNotWithinFiveFeet}},
		{"not an enemy", resSpec + "FR-117", "unhappy (rule interaction)", attackHelp(combat.AttackAssistTargetNotAnEnemy), combat.LedgerAttackRollHelpRefused{Reason: combat.AttackRollHelpRefusalTargetNotAnEnemy}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := resTakeRun(t, e, combat.ActionLedger{}, "a", c.request)
			if err != nil {
				t.Fatal(err)
			}
			if c.refusal != nil {
				resWant(t, resAs[combat.TakeCombatantActionRejected](t, res), e, combat.ActionLedger{}, combat.LedgerRejection{Refusal: c.refusal}, nil)
				return
			}
			done := resAs[combat.TakeCombatantActionCompleted](t, res)
			if want := []string{"action_taken", "help_given"}; !slices.Equal(resKinds(done.Events()), want) || done.Ledger().HelpBenefits().Len() != 1 {
				t.Errorf("events %v, benefits %d, want %v and one benefit [%s]", resKinds(done.Events()), done.Ledger().HelpBenefits().Len(), want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-118, 4.18.1 weapon handling - handling a weapon records the change in the Attack action's slot without spending a tracker resource or the free interaction; with no Attack action, or no attack made for an "after" change, it is refused by the ledger.
// Path: per case.
func TestTakeCombatantActionHandlesAWeapon(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	begun, _ := resAttack(t, resParams(t, e, "a", "b", resBegin()), 14, 4)
	drop := combat.HandleWeapon{Change: combat.WeaponChangeDrop, Timing: combat.WeaponChangeAfterAttack}
	stow := combat.HandleWeapon{Change: combat.WeaponChangeStow, Timing: combat.WeaponChangeAfterAttack}
	got := resTake(t, begun.Encounter(), begun.Ledger(), "a", drop)
	if want := []string{"weapon_changed_in_attack_action"}; !slices.Equal(resKinds(got.Events()), want) || !got.Encounter().Equal(begun.Encounter()) {
		t.Errorf("events %v, encounter equal %v, want %v and the encounter untouched: no tracker resource and no interaction [FR-118]", resKinds(got.Events()), got.Encounter().Equal(begun.Encounter()), want)
	}
	cases := []struct {
		name   string
		ledger combat.ActionLedger
		req    combat.TurnRequest
		want   combat.LedgerRefusal
	}{
		{"no Attack action in progress", combat.ActionLedger{}, drop, combat.LedgerNoAttackActionInProgress{}},
		{"the slot is used", got.Ledger(), stow, combat.LedgerWeaponChangeRefused{Reason: combat.WeaponChangeRefusalAlreadyUsedForThisAttack}},
		{"before an attack when none remains", begun.Ledger(), combat.HandleWeapon{Change: combat.WeaponChangeDrawFromSheath, Timing: combat.WeaponChangeBeforeAttack},
			combat.LedgerWeaponChangeRefused{Reason: combat.WeaponChangeRefusalNoAttackRemaining}},
	}
	for _, c := range cases {
		res, err := resTakeRun(t, begun.Encounter(), c.ledger, "a", c.req)
		if err != nil {
			t.Fatal(err)
		}
		resWant(t, resAs[combat.TakeCombatantActionRejected](t, res), begun.Encounter(), c.ledger, combat.LedgerRejection{Refusal: c.want}, nil)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-122, FR-114, 4.18.1 attack replaced - a replaced attack spends the action when it begins the Attack action, counts as an attack made and rolls no attack; a replacement that continues the action makes no tracker call; the limit, no progress and a spent action refuse it.
// Path: per case.
func TestTakeCombatantActionReplacesAnAttack(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	one, two := allowanceOf(t, 1, 1), allowanceOf(t, 2, 2)
	rep := replacementOf(t)
	begin := func(a combat.AttackAllowance) combat.TurnRequest {
		return combat.ReplaceAttack{Cost: combat.BeginsAttackActionByReplacing{Allowance: a}, Replacement: rep}
	}
	cont := combat.ReplaceAttack{Cost: combat.ContinuesAttackActionByReplacing{}, Replacement: rep}
	first := resTake(t, e, combat.ActionLedger{}, "a", begin(two))
	if want := []string{"action_taken", "attack_replaced_in_attack_action"}; !slices.Equal(resKinds(first.Events()), want) || len(resAttacks(t, first.Ledger())) != 1 {
		t.Errorf("events %v, want %v with one attack counted", resKinds(first.Events()), want)
	}
	second := resTake(t, first.Encounter(), first.Ledger(), "a", cont)
	if want := []string{"attack_replaced_in_attack_action"}; !slices.Equal(resKinds(second.Events()), want) || !second.Encounter().Equal(first.Encounter()) || len(resAttacks(t, second.Ledger())) != 2 {
		t.Errorf("events %v, want %v with no tracker call and two attacks counted [FR-122]", resKinds(second.Events()), want)
	}
	full := resTake(t, e, combat.ActionLedger{}, "a", begin(one))
	dodged := resDodge(t, e, combat.ActionLedger{}, "a")
	cases := []struct {
		name   string
		e      combat.Encounter
		ledger combat.ActionLedger
		req    combat.TurnRequest
		want   combat.ResolutionRejection
	}{
		{"limit reached", full.Encounter(), full.Ledger(), cont, combat.LedgerRejection{Refusal: combat.LedgerAttackRefused{Reason: combat.AttackRefusalLimitReached}}},
		{"no Attack action in progress", e, combat.ActionLedger{}, cont, combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}},
		{"action already taken", dodged.Encounter(), combat.ActionLedger{}, begin(two), combat.TrackerRejection{Rejection: combat.TurnActionAlreadyTakenRejection{}}},
	}
	for _, c := range cases {
		res, err := resTakeRun(t, c.e, c.ledger, "a", c.req)
		if err != nil {
			t.Fatal(err)
		}
		resWant(t, resAs[combat.TakeCombatantActionRejected](t, res), c.e, c.ledger, c.want, nil)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-098, plan-phase-4.md P10 - TakeHelpAction with a ledger already holding 256 pending Help benefits is TakeCombatantActionInvalid with LimitExceededProblem, the encounter and ledger unchanged and no event; with 255 it completes.
// Path: unhappy (boundary).
func TestTakeCombatantActionHelpBenefitCap(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	request := combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("b"), Target: combat.AttackAssistTargetEnemyWithinFiveFeet}}
	ledgerOf := func(n int) combat.ActionLedger {
		list := make([]combat.HelpBenefit, n)
		for i := range list {
			list[i] = rollHelp(t, fmt.Sprintf("h%d", i), "e")
		}
		l, err := combat.NewActionLedger(nil, benefitsOf(t, list...))
		if err != nil {
			t.Fatal(err)
		}
		return l
	}
	t.Run("255 pending benefits complete", func(t *testing.T) {
		t.Parallel()
		res, err := resTakeRun(t, e, ledgerOf(255), "a", request)
		if err != nil {
			t.Fatal(err)
		}
		if done := resAs[combat.TakeCombatantActionCompleted](t, res); done.Ledger().HelpBenefits().Len() != 256 {
			t.Errorf("benefits = %d, want 256", done.Ledger().HelpBenefits().Len())
		}
	})
	t.Run("256 pending benefits are invalid", func(t *testing.T) {
		t.Parallel()
		ledger := ledgerOf(256)
		res, err := resTakeRun(t, e, ledger, "a", request)
		if err != nil {
			t.Fatal(err)
		}
		inv := resAs[combat.TakeCombatantActionInvalid](t, res)
		if inv.Problem() != (combat.LimitExceededProblem{}) || !inv.Encounter().Equal(e) || !inv.Ledger().Equal(ledger) || len(inv.Events()) != 0 {
			t.Errorf("= %#v, want limit_exceeded with the encounter and ledger unchanged and no event", inv.Problem())
		}
	})
}
