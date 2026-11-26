package combat_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/combat-action-resolution.md FR-018, FR-019, FR-036, FR-040, FR-089, FR-091, 4.15.1 - a hit by the first attack of an Attack action spends the action, records the attack, rolls the d20 and the damage, hands the damage to the encounter and returns the events in order: the tracker's, the ledger's, AttackPerformed, then the damage.
// Path: happy.
func TestPerformAttackHit(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	got, dice := resAttack(t, resParams(t, e, "a", "b", resBegin()), 14, 4)
	if want := []string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken"}; !slices.Equal(resKinds(got.Events()), want) {
		t.Errorf("events = %v, want %v", resKinds(got.Events()), want)
	}
	if !slices.Equal(dice, []string{"1d20", "1d8"}) {
		t.Errorf("dice = %v, want the d20 then the damage", dice)
	}
	if resHP(t, got.Encounter(), "b") != 13 || resHP(t, e, "b") != 20 {
		t.Errorf("target has %d Hit Points and the encounter that came in %d, want 13 and 20 [FR-004]", resHP(t, got.Encounter(), "b"), resHP(t, e, "b"))
	}
	res := got.Resolutions()
	if len(res) != 1 || res[0].Dealt() != 7 {
		t.Errorf("resolutions = %v, want one of 7 dealt (4 + 3)", res)
	}
	if at := got.Attack().(combat.AttackResolved); !at.Hit() || at.Attack().Roll().Natural() != 14 {
		t.Errorf("attack = %+v, want the hit on the natural 14", at)
	}
	if n := len(resAttacks(t, got.Ledger())); n != 1 {
		t.Errorf("the ledger holds %d attacks, want 1", n)
	}
	ev := got.Events()
	ap := ev[2].(combat.AttackPerformed)
	if hit, ok := ap.Outcome().(combat.AttackHitOutcome); !ok || hit.HitKind() != combat.NormalHit || ap.Round() != 1 || ap.EncounterID() != eid() ||
		!slices.Equal(ap.Combatants(), []combat.CombatantID{cid("a"), cid("b")}) {
		t.Errorf("AttackPerformed = %+v, want a normal hit by a on b in round 1", ap)
	}
	if !encMember(t, e, "a").Equal(encMember(t, got.Encounter(), "a")) || !encMember(t, e, "c").Equal(encMember(t, got.Encounter(), "c")) {
		t.Errorf("a combatant the attack did not name changed [FR-004]")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-023, FR-039 - a miss completes the call: the resource is spent, the attack counted and AttackPerformed produced, with no damage roll and no damage event; a hit with no DamageSpec applies nothing.
// Path: unhappy (boundary).
func TestPerformAttackMissAndHitWithoutDamage(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	miss, dice := resAttack(t, resParams(t, e, "a", "b", resBegin()), 9)
	if want := []string{"action_taken", "attack_made_in_attack_action", "attack_performed"}; !slices.Equal(resKinds(miss.Events()), want) || !slices.Equal(dice, []string{"1d20"}) {
		t.Errorf("miss: events %v, dice %v, want %v and one d20 [FR-023]", resKinds(miss.Events()), dice, want)
	}
	if _, ok := miss.Events()[2].(combat.AttackPerformed).Outcome().(combat.AttackMissOutcome); !ok || resHP(t, miss.Encounter(), "b") != 20 || len(miss.Resolutions()) != 0 {
		t.Errorf("a miss must be an AttackMissOutcome with no damage")
	}
	p := resParams(t, e, "a", "b", resBegin())
	spell := meleeParams()
	p.Attack = attackCommand(t, spell)
	noDamage, dice := resAttack(t, p, 15)
	if want := []string{"action_taken", "attack_made_in_attack_action", "attack_performed"}; !slices.Equal(resKinds(noDamage.Events()), want) || !slices.Equal(dice, []string{"1d20"}) ||
		resHP(t, noDamage.Encounter(), "b") != 20 {
		t.Errorf("hit without a DamageSpec: events %v, dice %v, want %v and no damage [FR-039]", resKinds(noDamage.Events()), dice, want)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-031, FR-036 - a natural 20 is a Critical Hit: the damage dice are doubled and the damage is applied in one command carrying HitKind CriticalHit.
// Path: unhappy (boundary).
func TestPerformAttackCriticalHit(t *testing.T) {
	t.Parallel()
	got, dice := resAttack(t, resParams(t, encABC(t), "a", "b", resBegin()), 20, 3, 4)
	if !slices.Equal(dice, []string{"1d20", "2d8"}) {
		t.Errorf("dice = %v, want 1d20 then 2d8", dice)
	}
	out := got.Events()[2].(combat.AttackPerformed).Outcome().(combat.AttackHitOutcome)
	if out.HitKind() != combat.CriticalHit || got.Resolutions()[0].Dealt() != 10 || resHP(t, got.Encounter(), "b") != 10 {
		t.Errorf("kind %v, dealt %d, target %d, want a Critical Hit of 3 + 4 + 3 leaving 10", out.HitKind(), got.Resolutions()[0].Dealt(), resHP(t, got.Encounter(), "b"))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-013 - the attacker may be its own target: the rules apply once in each role.
// Path: unhappy (rule interaction).
func TestPerformAttackSelfTarget(t *testing.T) {
	t.Parallel()
	got, _ := resAttack(t, resParams(t, encABC(t), "a", "a", resBegin()), 14, 4)
	if resHP(t, got.Encounter(), "a") != 13 {
		t.Errorf("the attacker has %d Hit Points, want 13", resHP(t, got.Encounter(), "a"))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-019, FR-113, FR-116, 4.4.1 and 4.18.1 - what each cost spends: the Attack action begins and its first attack is counted; Extra Attack continues without asking the tracker again; a Bonus Action, a Reaction and an attack already paid leave the Attack action in progress as it was.
// Path: per case.
func TestPerformAttackCosts(t *testing.T) {
	t.Parallel()
	two := allowanceOf(t, 2, 2)
	grant, err := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "cunning_action"}, combat.GrantsSimpleAction{Action: combat.SimpleStandardActionAttack})
	if err != nil {
		t.Fatal(err)
	}
	e := encABC(t)
	begun, _ := resAttack(t, resParams(t, e, "a", "b", combat.BeginsAttackAction{Allowance: two, Means: combat.AttackMeansOtherWeapon}), 14, 4)
	cases := []struct {
		name, requirement, path string
		attacker, target        string
		cost                    combat.AttackCost
		ledger                  combat.ActionLedger
		events                  []string
		attacks                 int
	}{
		{"Attack action begins", resSpec + "FR-019, 4.4.1", "happy", "a", "b", combat.BeginsAttackAction{Allowance: two, Means: combat.AttackMeansOtherWeapon}, combat.ActionLedger{},
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken"}, 1},
		{"Extra Attack continues, no second tracker call", resSpec + "FR-113, 4.18.1", "happy", "a", "b", combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, begun.Ledger(),
			[]string{"attack_made_in_attack_action", "attack_performed", "damage_taken"}, 2},
		{"Bonus Action attack leaves the Attack action as it was", resSpec + "FR-116, FR-019", "unhappy (rule interaction)", "a", "b", combat.BonusActionCost{Grant: grant}, begun.Ledger(),
			[]string{"bonus_action_used", "attack_performed", "damage_taken"}, 1},
		{"attack already paid", resSpec + "FR-020, 4.4.1", "unhappy (rule interaction)", "a", "b", combat.AttackAlreadyPaid{}, begun.Ledger(),
			[]string{"attack_performed", "damage_taken"}, 1},
		{"Opportunity Attack by Reaction, on another turn", resSpec + "FR-116, FR-024, 4.4.1", "unhappy (rule interaction)", "b", "a",
			combat.ReactionCost{Choice: combat.OpportunityAttackReaction{Provoker: cid("a"), Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}}, begun.Ledger(),
			[]string{"reaction_taken", "attack_performed", "damage_taken"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, e, c.attacker, c.target, c.cost)
			p.Ledger = c.ledger
			got, _ := resAttack(t, p, 14, 4)
			if !slices.Equal(resKinds(got.Events()), c.events) {
				t.Errorf("events = %v, want %v [%s]", resKinds(got.Events()), c.events, c.requirement)
			}
			if n := len(resAttacks(t, got.Ledger())); c.attacks != 0 && n != c.attacks {
				t.Errorf("the ledger counts %d attacks, want %d [%s]", n, c.attacks, c.requirement)
			}
		})
	}
	// A Reaction or Bonus Action attack with no Attack action in progress records none (FR-116).
	for _, cost := range []combat.AttackCost{combat.BonusActionCost{Grant: grant}, combat.AttackAlreadyPaid{}} {
		got, _ := resAttack(t, resParams(t, e, "a", "b", cost), 14, 4)
		if _, in := got.Ledger().AttackAction().(combat.AttackActionInProgress); in {
			t.Errorf("%T started an Attack action, want none [FR-116]", cost)
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-113, FR-017, FR-010 steps 4 and 5, 4.4.1 and 4.18.1 - the ledger refuses before the tracker, and the tracker before any die: a limit reached, no Attack action in progress, a spent action, an attacker that is not active, a Bonus Action with no grant and a Reaction already spent each end the call with the rule, the encounter and ledger unchanged and no die rolled.
// Path: per case.
func TestPerformAttackRefusals(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	one := resBegin()
	begun, _ := resAttack(t, resParams(t, e, "a", "b", one), 14, 4)
	dodged := resDodge(t, e, combat.ActionLedger{}, "a")
	reacted, _ := resAttack(t, resParams(t, e, "b", "a", combat.ReactionCost{Choice: combat.OpportunityAttackReaction{Provoker: cid("a"), Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}}), 14, 4)
	incapacitated, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Incapacitated{}), resPC(t, "b", combat.SideB))
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		attacker, target        string
		cost                    combat.AttackCost
		ledger                  combat.ActionLedger
		want                    combat.ResolutionRejection
	}{
		{"limit reached", resSpec + "FR-113, 4.18.1", "unhappy (boundary)", e, "a", "b", combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, begun.Ledger(),
			combat.LedgerRejection{Refusal: combat.LedgerAttackRefused{Reason: combat.AttackRefusalLimitReached}}},
		{"continues with no Attack action", resSpec + "FR-113", "unhappy (state)", e, "a", "b", combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, combat.ActionLedger{},
			combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}},
		{"continues another combatant's Attack action", resSpec + "FR-113", "unhappy (state)", e, "c", "b", combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, begun.Ledger(),
			combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}},
		{"ledger before tracker: Incapacitated and no Attack action", resSpec + "4.2.1 ledger before tracker", "unhappy (rule interaction)", incapacitated, "a", "b",
			combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, combat.ActionLedger{}, combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}},
		{"action already spent", resSpec + "FR-019, 4.4.1", "unhappy (state)", dodged.Encounter(), "a", "b", one, combat.ActionLedger{},
			combat.TrackerRejection{Rejection: combat.TurnActionAlreadyTakenRejection{}}},
		{"not the active combatant", resSpec + "FR-019, 4.4.1", "unhappy (state)", e, "b", "a", one, combat.ActionLedger{}, combat.TrackerRejection{Rejection: combat.TurnNotActiveCombatantRejection{}}},
		{"Incapacitated attacker", resSpec + "FR-016, FR-055", "unhappy (rule interaction)", incapacitated, "a", "b", one, combat.ActionLedger{}, combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}},
		{"Bonus Action with no grant", resSpec + "FR-019, 4.4.1", "unhappy (state)", e, "a", "b", combat.BonusActionCost{}, combat.ActionLedger{}, combat.TrackerRejection{Rejection: combat.TurnNoBonusActionGrantedRejection{}}},
		{"Reaction already spent", resSpec + "FR-019, 4.4.1", "unhappy (state)", reacted.Encounter(), "b", "a",
			combat.ReactionCost{Choice: combat.OpportunityAttackReaction{Provoker: cid("a"), Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}}, combat.ActionLedger{},
			combat.TrackerRejection{Rejection: combat.TurnReactionAlreadyTakenRejection{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, c.e, c.attacker, c.target, c.cost)
			p.Ledger = c.ledger
			res, roller, err := resPerform(t, p, 14, 4)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			resWant(t, resAs[combat.PerformAttackRejected](t, res), c.e, c.ledger, c.want, roller)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md 4.2.1, FR-010 to FR-012, FR-128, FR-129 - the stop order of an attack: the encounter ended before an unknown id, an unknown id (the attacker first) before a dead one, a dead creature before the tracker, the tracker before the charmer restriction, that before a cover that makes the attack impossible; every stop ends the call before any die.
// Path: per case.
func TestPerformAttackStopOrder(t *testing.T) {
	t.Parallel()
	healthy := encABC(t)
	deadA, _ := encStart(t, resDead(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	deadB, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Incapacitated{}), resDead(t, "b", combat.SideB))
	charmed, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: cid("b")}}), resPC(t, "b", combat.SideB))
	charmedIncap, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: cid("b")}}, combat.Incapacitated{}), resPC(t, "b", combat.SideB))
	cover := func(p combat.PerformAttackParams) combat.PerformAttackParams {
		q := meleeParams()
		q.Damage = slashing(1, 8)
		q.Cover = []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
		p.Attack = attackCommand(t, q)
		return p
	}
	type row struct {
		name, requirement, path string
		e                       combat.Encounter
		attacker, target        string
		edit                    func(combat.PerformAttackParams) combat.PerformAttackParams
		rejection               combat.ResolutionRejection
		problem                 combat.ResolutionProblem
	}
	endedUnknown := resEnded(t, healthy)
	cases := []row{
		{"ended and unknown attacker", resSpec + "4.2.1 ended and unknown", "unhappy (rule interaction)", endedUnknown, "ghost", "b", nil,
			combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil},
		{"unknown attacker", resSpec + "4.2.1 unknown ids", "unhappy (invalid input)", healthy, "ghost", "b", nil, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"unknown attacker and target name the attacker", resSpec + "4.2.1 unknown ids", "unhappy (invalid input)", healthy, "ghost", "phantom", nil, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"unknown target, dead attacker", resSpec + "4.2.1 unknown ids", "unhappy (invalid input)", deadA, "a", "ghost", nil, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"dead attacker and target name the attacker", resSpec + "4.2.1 dead", "unhappy (rule interaction)", deadA, "a", "b", nil,
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("a")}}, nil},
		{"dead target before Incapacitated attacker", resSpec + "4.2.1 dead", "unhappy (rule interaction)", deadB, "a", "b", nil,
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil},
		{"tracker before charmer: Incapacitated", resSpec + "4.2.1 tracker before charmer", "unhappy (rule interaction)", charmedIncap, "a", "b", nil,
			combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}, nil},
		{"charmer before Total Cover", resSpec + "4.2.1 charmer and Total Cover", "unhappy (rule interaction)", charmed, "a", "b", cover, combat.TargetingForbiddenRejection{}, nil},
		{"Total Cover alone", resSpec + "FR-021", "unhappy (state)", healthy, "a", "b", cover, combat.AttackNotPossibleRejection{Reason: combat.AttackNotPossibleReasonTotalCover}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, c.e, c.attacker, c.target, resBegin())
			if c.edit != nil {
				p = c.edit(p)
			}
			res, roller, err := resPerform(t, p, 14, 4)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if c.problem != nil {
				inv := resAs[combat.PerformAttackInvalid](t, res)
				if inv.Problem() != c.problem || !inv.Encounter().Equal(c.e) || len(inv.Events()) != 0 || len(roller.Requests()) != 0 {
					t.Errorf("problem %#v, want %#v with the encounter unchanged and no die [%s]", inv.Problem(), c.problem, c.requirement)
				}
				return
			}
			resWant(t, resAs[combat.PerformAttackRejected](t, res), c.e, combat.ActionLedger{}, c.rejection, roller)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-021 - an impossible attack spends nothing and rolls nothing, so the same inputs then succeed with a possible attack: the ledger and the tracker came back as they were.
// Path: unhappy (state).
func TestPerformAttackNotPossibleSpendsNothing(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	q := meleeParams()
	q.Damage = slashing(1, 8)
	q.Cover = []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	p := resParams(t, e, "a", "b", resBegin())
	p.Attack = attackCommand(t, q)
	res, roller, err := resPerform(t, p, 14, 4)
	if err != nil {
		t.Fatal(err)
	}
	rej := resAs[combat.PerformAttackRejected](t, res)
	if rej.Rejection() != (combat.AttackNotPossibleRejection{Reason: combat.AttackNotPossibleReasonTotalCover}) || len(roller.Requests()) != 0 {
		t.Fatalf("rejection %#v, dice %v, want Total Cover and no die", rej.Rejection(), notations(roller))
	}
	if _, in := rej.Ledger().AttackAction().(combat.AttackActionInProgress); in || !rej.Encounter().Equal(e) {
		t.Errorf("the ledger or the tracker changed on a refusal [FR-021]")
	}
	if got, _ := resAttack(t, resParams(t, rej.Encounter(), "a", "b", resBegin()), 14, 4); resHP(t, got.Encounter(), "b") != 13 {
		t.Errorf("the action was spent by an impossible attack")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-022 - an attack that misses automatically (a ranged weapon attack underwater beyond normal range) completes: the resource is spent, the attack recorded and AttackPerformed produced, with no roll and no damage.
// Path: unhappy (rule interaction).
func TestPerformAttackAutomaticMiss(t *testing.T) {
	t.Parallel()
	q := rangedParams()
	q.Distance = 100
	q.Situation = combat.AttackSituation{Underwater: true}
	q.Damage = slashing(1, 8)
	p := resParams(t, encABC(t), "a", "b", resBegin())
	p.Attack, p.Scene = attackCommand(t, q), resSeen(t, 100)
	got, dice := resAttack(t, p)
	out, ok := got.Events()[2].(combat.AttackPerformed).Outcome().(combat.AttackAutomaticallyMissedOutcome)
	if want := []string{"action_taken", "attack_made_in_attack_action", "attack_performed"}; !slices.Equal(resKinds(got.Events()), want) || len(dice) != 0 || !ok ||
		out.Reason() != combat.AutomaticMissReasonUnderwaterBeyondNormalRange || resHP(t, got.Encounter(), "b") != 20 {
		t.Errorf("events %v, dice %v, outcome %#v, want an automatic miss with no die", resKinds(got.Events()), dice, got.Events()[2])
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-027 to FR-032, FR-125, 4.5.1 - the conditions, Dodge and Exhaustion reach the roll: the circumstances reported in AttackResolved are the base command's then the condition ones then Dodge, the number of d20s follows the combined mode, an Exhaustion reduction lowers the total at the boundary and an automatic Critical Hit doubles the dice.
// Path: per case.
func TestPerformAttackConditionInputsReachTheRoll(t *testing.T) {
	t.Parallel()
	pair := func(a, b []combat.Condition) combat.Encounter {
		e, _ := encStart(t, resPC(t, "a", combat.SideA, a...), resPC(t, "b", combat.SideB, b...))
		return e
	}
	dodging := func() combat.Encounter {
		e, _ := encStart(t, resPC(t, "b", combat.SideB), resPC(t, "a", combat.SideA))
		d := resDodge(t, e, combat.ActionLedger{}, "b")
		return resAdvance(t, d.Encounter(), d.Ledger()).Encounter()
	}()
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		faces                   []int
		dice                    []string
		circumstances           int
		hp                      int
	}{
		{"no conditions", resSpec + "4.5.1 none", "happy", pair(nil, nil), []int{14, 4}, []string{"1d20", "1d8"}, 0, 13},
		{"attacker Poisoned: Disadvantage, the lower die", resSpec + "FR-027, 4.5.1", "happy", pair([]combat.Condition{combat.Poisoned{}}, nil), []int{20, 15, 4}, []string{"2d20", "1d8"}, 1, 13},
		{"target Restrained: Advantage, the higher die", resSpec + "FR-027, 4.5.1", "happy", pair(nil, []combat.Condition{combat.Restrained{}}), []int{3, 14, 4}, []string{"2d20", "1d8"}, 1, 13},
		{"Advantage and Disadvantage cancel to one die", resSpec + "FR-027, FR-028, 4.5.1", "unhappy (rule interaction)", pair([]combat.Condition{combat.Poisoned{}},
			[]combat.Condition{combat.Blinded{}}), []int{14, 4}, []string{"1d20", "1d8"}, 2, 13},
		{"target Paralyzed: Advantage and an automatic Critical Hit", resSpec + "FR-031, 4.5.1 critical", "unhappy (boundary)", pair(nil, []combat.Condition{combat.Paralyzed{}}),
			[]int{3, 14, 4, 4}, []string{"2d20", "2d8"}, 1, 20 - 11},
		{"target dodging sees the attacker: Disadvantage", resSpec + "FR-029, 4.5.1 dodging", "unhappy (rule interaction)", dodging, []int{20, 2}, []string{"2d20"}, 1, 20},
		{"Exhaustion 2: natural 13 totals 14 and misses", resSpec + "FR-030, 4.5.1 Exhaustion", "unhappy (boundary)", pair([]combat.Condition{exhaustion(2)}, nil), []int{13}, []string{"1d20"}, 0, 20},
		{"Exhaustion 2: natural 14 totals 15 and hits", resSpec + "FR-030, 4.5.1 Exhaustion", "unhappy (boundary)", pair([]combat.Condition{exhaustion(2)}, nil), []int{14, 4}, []string{"1d20", "1d8"}, 0, 13},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, dice := resAttack(t, resParams(t, c.e, "a", "b", resBegin()), c.faces...)
			at := got.Attack().(combat.AttackResolved)
			if !slices.Equal(dice, c.dice) || len(at.Circumstances()) != c.circumstances || resHP(t, got.Encounter(), "b") != c.hp {
				t.Errorf("dice %v, circumstances %v, target %d; want %v, %d circumstances, %d [%s]", dice, at.Circumstances(), resHP(t, got.Encounter(), "b"), c.dice, c.circumstances, c.hp, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-032, FR-004 - the derived inputs are added to the base command and never replace it: the command's own circumstance stays first, and the command the host built is not altered.
// Path: unhappy (immutability).
func TestPerformAttackKeepsTheBaseCommand(t *testing.T) {
	t.Parallel()
	q := meleeParams()
	q.Damage = slashing(1, 8)
	q.Circumstances = []modifier.RollCircumstance{advantage}
	e, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Poisoned{}), resPC(t, "b", combat.SideB))
	p := resParams(t, e, "a", "b", resBegin())
	p.Attack = attackCommand(t, q)
	got, _ := resAttack(t, p, 14, 4)
	if cs := got.Attack().(combat.AttackResolved).Circumstances(); !slices.Equal(cs, []modifier.RollCircumstance{advantage, disadvantage}) {
		t.Errorf("circumstances = %v, want the base command's Advantage and then the condition's Disadvantage", cs)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-037, FR-136, 4.6.1 - a melee attack by a creature that chose KnockOut leaves the target at 1 Hit Point and Unconscious instead of dropping it; without the choice the same damage drops the creature to 0.
// Path: unhappy (rule interaction).
func TestPerformAttackKnockOut(t *testing.T) {
	t.Parallel()
	for _, knock := range []combat.KnockOutChoice{combat.KnockOut, combat.NoKnockOut} {
		e, _ := encStart(t, resPC(t, "a", combat.SideA), encPC(t, "b", combat.SideB, 5, 20))
		p := resParams(t, e, "a", "b", resBegin())
		p.KnockOut = knock
		got, _ := resAttack(t, p, 14, 4)
		if knock == combat.KnockOut && (resHP(t, got.Encounter(), "b") != 1 || !resHas(t, got.Encounter(), "b", content.ConditionUnconscious) || !encMember(t, got.Encounter(), "b").KnockedOut()) {
			t.Errorf("knocked out: %d Hit Points, Unconscious %v", resHP(t, got.Encounter(), "b"), resHas(t, got.Encounter(), "b", content.ConditionUnconscious))
		}
		if knock == combat.NoKnockOut && (resHP(t, got.Encounter(), "b") != 0 || encMember(t, got.Encounter(), "b").KnockedOut()) {
			t.Errorf("not knocked out: %d Hit Points", resHP(t, got.Encounter(), "b"))
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-036, FR-040, 4.6.1 - the damage of every component is applied in one command: a Dying target takes one Death Saving Throw failure from a hit with two Damage Types and two from a Critical Hit, not four; the resolutions and events equal what the root returns for the same inputs.
// Path: unhappy (rule interaction).
func TestPerformAttackAppliesOneDamageCommand(t *testing.T) {
	t.Parallel()
	two := combat.MustDamageSpec(
		combat.MustDiceDamage(1, dice.MustDie(8), content.DamageSlashing, 0, true),
		combat.MustDiceDamage(1, dice.MustDie(4), content.DamageFire, 0, false))
	cases := []struct {
		name  string
		faces []int
		fails int
	}{
		{"normal hit", []int{14, 4, 2}, 1},
		{"Critical Hit", []int{20, 4, 3, 2, 1}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB))
			q := meleeParams()
			q.Damage = two
			p := resParams(t, e, "a", "b", resBegin())
			p.Attack = attackCommand(t, q)
			got, _ := resAttack(t, p, c.faces...)
			dying, ok := encMember(t, got.Encounter(), "b").HitPoints().LifeState().(combat.Dying)
			if !ok || dying.Failures() != c.fails {
				t.Errorf("life state %#v, want Dying with %d failures [FR-036]", encMember(t, got.Encounter(), "b").HitPoints().LifeState(), c.fails)
			}
			direct, err := e.ApplyDamage(cid("b"), got.Attack().(combat.AttackResolved).Damage(), nil, combat.MeleeAttackWithoutKnockOut, got.Attack().(combat.AttackResolved).HitKind())
			if err != nil || !slices.Equal(direct.Resolutions(), got.Resolutions()) {
				t.Errorf("resolutions %v, want what the root returns %v [FR-040] (%v)", got.Resolutions(), direct.Resolutions(), err)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-038, FR-040, 4.6.1 adjustments - the host's adjustments reach the damage unchanged, in the order given, and what the encounter's rules make of them is what the call returns: the resolutions equal the root's for the same inputs, and the adjustments do change the damage dealt.
// Path: unhappy (rule interaction).
func TestPerformAttackPassesAdjustmentsUnchanged(t *testing.T) {
	t.Parallel()
	flat := combat.NewFlatAdjustment(-5, combat.AllDamage{})
	for _, c := range []struct {
		name string
		adj  []combat.DamageAdjustment
	}{
		{"flat then half", []combat.DamageAdjustment{flat, combat.HalfDamage{}}},
		{"half then flat", []combat.DamageAdjustment{combat.HalfDamage{}, flat}},
		{"none", nil},
	} {
		e := encABC(t)
		p := resParams(t, e, "a", "b", resBegin())
		p.DamageAdjustments = c.adj
		got, _ := resAttack(t, p, 14, 4)
		direct, err := e.ApplyDamage(cid("b"), got.Attack().(combat.AttackResolved).Damage(), c.adj, combat.MeleeAttackWithoutKnockOut, combat.NormalHit)
		if err != nil || !slices.Equal(got.Resolutions(), direct.Resolutions()) || !slices.Equal(encKinds(got.Events())[3:], encKinds(direct.Events())) {
			t.Errorf("%s: resolutions %v and events %v, want what the root returns %v and %v (%v)", c.name, got.Resolutions(), encKinds(got.Events()), direct.Resolutions(), encKinds(direct.Events()), err)
		}
		if c.adj != nil && got.Resolutions()[0].Dealt() >= 7 {
			t.Errorf("%s: dealt %d, want the adjustments to lower the 7", c.name, got.Resolutions()[0].Dealt())
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-117, FR-123, 4.18.1 Help - a Help benefit the attacker's ally gave against the target gives the attack Advantage, is spent by the completed attack (HelpBenefitUsed follows AttackPerformed) and is not given by an attack of a combatant the host did not state as allied; the ledger a refused call returns still holds it.
// Path: unhappy (rule interaction).
func TestPerformAttackConsumesHelp(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "c", combat.SideA), resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	helped := resTake(t, e, combat.ActionLedger{}, "c", combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("b"), Target: combat.AttackAssistTargetEnemyWithinFiveFeet}})
	e = resAdvance(t, helped.Encounter(), helped.Ledger()).Encounter()
	allied, err := combat.NewAlliedHelpers(cid("c"))
	if err != nil {
		t.Fatal(err)
	}
	p := resParams(t, e, "a", "b", resBegin())
	p.Ledger = helped.Ledger()
	p.AlliedHelpers = allied
	got, dice := resAttack(t, p, 3, 14, 4)
	if want := []string{"action_taken", "attack_made_in_attack_action", "attack_performed", "help_benefit_used", "damage_taken"}; !slices.Equal(resKinds(got.Events()), want) || !slices.Equal(dice, []string{"2d20", "1d8"}) {
		t.Errorf("events %v, dice %v, want %v with Advantage", resKinds(got.Events()), dice, want)
	}
	if got.Ledger().HelpBenefits().Len() != 0 {
		t.Errorf("the ledger still holds %d benefits, want the spent one gone", got.Ledger().HelpBenefits().Len())
	}
	// A helper the host did not state as allied gives nothing, and the benefit stays pending.
	q := resParams(t, e, "a", "b", resBegin())
	q.Ledger = helped.Ledger()
	again, dice := resAttack(t, q, 14, 4)
	if !slices.Equal(dice, []string{"1d20", "1d8"}) || again.Ledger().HelpBenefits().Len() != 1 {
		t.Errorf("dice %v, benefits %d, want one d20 and the benefit still pending [FR-119]", dice, again.Ledger().HelpBenefits().Len())
	}
	// The benefit is spent only when the call completes: an impossible attack leaves it pending.
	cover := meleeParams()
	cover.Cover = []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	r := resParams(t, e, "a", "b", resBegin())
	r.Ledger, r.AlliedHelpers, r.Attack = helped.Ledger(), allied, attackCommand(t, cover)
	res, _, err := resPerform(t, r)
	if err != nil || resAs[combat.PerformAttackRejected](t, res).Ledger().HelpBenefits().Len() != 1 {
		t.Errorf("a refusal must leave the benefit pending [FR-117]: %v", err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-123, 4.18.1 Help benefit not dropped - a hit that kills a helper with a pending benefit does not drop it: the ledger still holds the benefit and no HelpBenefitDropped is produced.
// Path: unhappy (rule interaction).
func TestPerformAttackDoesNotDropHelpBenefits(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encC(t, "b", combat.SideB, status(t, 3, 20, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())), resPC(t, "a", combat.SideA))
	helped := resTake(t, e, combat.ActionLedger{}, "b", combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("a"), Target: combat.AttackAssistTargetEnemyWithinFiveFeet}})
	next := resAdvance(t, helped.Encounter(), helped.Ledger())
	p := resParams(t, next.Encounter(), "a", "b", resBegin())
	p.Ledger = next.Ledger()
	got, _ := resAttack(t, p, 14, 4)
	if got.Ledger().HelpBenefits().Len() != 1 || slices.Contains(resKinds(got.Events()), "help_benefit_dropped") || !slices.Contains(resKinds(got.Events()), "died") {
		t.Errorf("benefits %d, events %v, want the benefit kept, the helper dead and no HelpBenefitDropped", got.Ledger().HelpBenefits().Len(), resKinds(got.Events()))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-129 - a stop and a result depend only on the encounter's state and the command's values, not on the order of an unordered collection: commands that differ only in the order of the allied helpers give equal results.
// Path: unhappy (rule interaction).
func TestPerformAttackAlliedHelpersOrderIsIrrelevant(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "d", combat.SideA), resPC(t, "c", combat.SideA), resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	helped := resTake(t, e, combat.ActionLedger{}, "d", combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{Enemy: cid("b"), Target: combat.AttackAssistTargetEnemyWithinFiveFeet}})
	one := resAdvance(t, helped.Encounter(), helped.Ledger())
	next := resAdvance(t, one.Encounter(), one.Ledger())
	var results []combat.PerformAttackCompleted
	for _, order := range [][]string{{"c", "d"}, {"d", "c"}} {
		ids := []combat.CombatantID{cid(order[0]), cid(order[1])}
		allied, err := combat.NewAlliedHelpers(ids...)
		if err != nil {
			t.Fatal(err)
		}
		p := resParams(t, next.Encounter(), "a", "b", resBegin())
		p.Ledger, p.AlliedHelpers = helped.Ledger(), allied
		done, _ := resAttack(t, p, 3, 14, 4)
		results = append(results, done)
	}
	if !reflect.DeepEqual(results[0], results[1]) {
		t.Errorf("the two orders of the allied helpers gave different results [FR-129]")
	}
}
