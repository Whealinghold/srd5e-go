package combat_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-action-resolution.md FR-098, plan-phase-4.md P10 - an ability check Help asked of a ledger that already holds 256 pending benefits is TakeCombatantActionInvalid with LimitExceededProblem and leaves the encounter and ledger as they came, like the attack roll Help; with 255 it completes (mutation gap: only the attack roll form of the cap was tested through the use case).
// Path: unhappy (boundary).
func TestMutgapAbilityCheckHelpCapThroughTheUseCase(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	with, _ := combat.NewHelpGiverProficiencies([]content.Skill{content.SkillMedicine}, nil)
	request := combat.TakeHelpAction{Request: combat.AbilityCheckHelpRequest{Ally: cid("c"), Proficiency: combat.AssistedSkill(content.SkillMedicine),
		HelperProficiencies: with, Judgement: combat.AssistanceJudgementPossible}}
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
	res, err := resTakeRun(t, e, ledgerOf(255), "a", request)
	if err != nil {
		t.Fatal(err)
	}
	if done := resAs[combat.TakeCombatantActionCompleted](t, res); done.Ledger().HelpBenefits().Len() != 256 {
		t.Errorf("255 pending: benefits = %d, want 256", done.Ledger().HelpBenefits().Len())
	}
	ledger := ledgerOf(256)
	res, err = resTakeRun(t, e, ledger, "a", request)
	if err != nil {
		t.Fatal(err)
	}
	inv := resAs[combat.TakeCombatantActionInvalid](t, res)
	if inv.Problem() != (combat.LimitExceededProblem{}) || !inv.Encounter().Equal(e) || !inv.Ledger().Equal(ledger) || len(inv.Events()) != 0 {
		t.Errorf("256 pending: = %#v, want limit_exceeded with the encounter and ledger unchanged and no event", inv.Problem())
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-136 - the scene distance and the attack distance must be equal: a scene nearer than the attack is as invalid as one farther (mutation gap: only the farther scene was tested).
// Path: unhappy (boundary).
func TestMutgapPerformAttackSceneDistanceBothWays(t *testing.T) {
	t.Parallel()
	for _, feet := range []int{4, 6} {
		p := resParams(t, encABC(t), "a", "b", resBegin())
		p.Scene = resSeen(t, feet)
		if _, err := combat.NewPerformAttackCommand(p); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("scene at %d feet against an attack at 5: err = %v, want ErrInvalidInput", feet, err)
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-136, FR-090 - the zero use case and the zero command are each invalid input, whichever of the two is built (mutation gap: the two guards were only tested together).
// Path: unhappy (invalid input).
func TestMutgapPerformAttackZeroUseCaseOrCommand(t *testing.T) {
	t.Parallel()
	cmd, err := combat.NewPerformAttackCommand(resParams(t, encABC(t), "a", "b", resBegin()))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := (combat.PerformAttack{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case = %v, %v, want ErrInvalidInput", res, err)
	}
	use, err := combat.NewPerformAttack(dicetest.NewFixedRoller(14, 4))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := use.Run(context.Background(), combat.PerformAttackCommand{}); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero command = %v, %v, want ErrInvalidInput", res, err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-089, FR-165 - LimitedUse reports a state only for an attack made with RecordsMultiattack; an ordinary attack reports none.
// Path: unhappy (state).
func TestMutgapPerformAttackLimitedUseIsOnlyForMultiattack(t *testing.T) {
	t.Parallel()
	got, _ := resAttack(t, resParams(t, encABC(t), "a", "b", resBegin()), 14, 4)
	if lu, ok := got.LimitedUse(); ok || !lu.IsZero() {
		t.Errorf("LimitedUse() = %v, %v, want the zero state and false for an ordinary attack", lu, ok)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-161, FR-091 - the target's Parry is planned on the tracker as the attack's cost left it: after a parried attack the attacker's action stays spent and a second action is refused; Parry is not offered against a ranged attack; AttackPerformed carries a miss when Parry made one and a hit when the hit stands (mutation gaps: the tracker handed to Parry, the melee test and the outcome of a Parry that does not make a miss).
// Path: unhappy (rule interaction).
func TestMutgapParryUsesTheTrackerAfterTheCost(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	p := resParams(t, e, "a", "b", resBegin())
	p.TargetParry = parryChoice(t, true)
	became, _ := resAttack(t, p, 10, 4)
	if _, ok := became.Events()[3].(combat.AttackPerformed).Outcome().(combat.AttackMissOutcome); !ok {
		t.Errorf("a hit that Parry turned into a miss: outcome = %#v, want AttackMissOutcome", became.Events()[3].(combat.AttackPerformed).Outcome())
	}
	again := combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionHelp}}
	cmd, err := combat.NewTakeCombatantActionCommand(became.Encounter(), combat.ActionLedger{}, cid("a"), again)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewTakeCombatantAction().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if rej := resAs[combat.TakeCombatantActionRejected](t, res); rej.Rejection() != (combat.TrackerRejection{Rejection: combat.TurnActionAlreadyTakenRejection{}}) {
		t.Errorf("a second action after the parried attack = %#v, want the action already taken", rej.Rejection())
	}

	stands, _ := resAttack(t, p, 12, 4)
	if _, ok := stands.Events()[3].(combat.AttackPerformed).Outcome().(combat.AttackHitOutcome); !ok {
		t.Errorf("a hit that Parry did not turn into a miss: outcome = %#v, want AttackHitOutcome", stands.Events()[3].(combat.AttackPerformed).Outcome())
	}

	ranged := resParams(t, e, "a", "b", resBegin())
	ranged.Attack = attackCommand(t, rangedParams())
	ranged.Scene = resSeen(t, 30)
	ranged.TargetParry = parryChoice(t, true)
	rolled, _ := resAttack(t, ranged, 10, 4)
	if _, ok := rolled.Parry().(combat.ParryNotUsed); !ok {
		t.Errorf("Parry against a ranged attack = %#v, want ParryNotUsed", rolled.Parry())
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-014, FR-017, FR-028 - a rule that is skipped for a reason other than "already affected this turn" is reported even when an earlier rule already made its part due: NotAnEnemy (a lower reason) and NotFirstEntryThisTurn (a higher one) are both listed, not swallowed (mutation gap: the comparison on the skip reason only had its equal case tested).
// Path: unhappy (rule interaction).
func TestMutgapSecondRuleOnADuePartKeepsItsSkip(t *testing.T) {
	t.Parallel()
	x := pzCreature("x", false, true, "body") // not an enemy of the caster
	t.Run("NotAnEnemy", func(t *testing.T) {
		t.Parallel()
		facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
			pzRule(content.CreatureEnters, 0, content.AnyCreature), pzRule(content.CreatureEnters, 0, content.EnemiesOfCaster)}})
		got := pzAssessEvent(t, pzCreate(t, pzSpell("two", 1, facts, nil), nil), pzTurn(1, "x"), combat.NewCreatureEntered(x))
		if len(got.Mandatory) != 1 || len(got.Skipped) != 1 || got.Skipped[0].Reason != combat.SkipNotAnEnemy {
			t.Errorf("mandatory %d, skipped %+v, want 1 due and the NotAnEnemy skip", len(got.Mandatory), got.Skipped)
		}
	})
	t.Run("NotFirstEntryThisTurn", func(t *testing.T) {
		t.Parallel()
		facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
			pzRule(content.CreatureEnters, 0, content.AnyCreature), pzRule(content.CreatureFirstEntryOnTurn, 0, content.AnyCreature)}})
		zone := pzCreate(t, pzSpell("two", 1, facts, nil), nil)
		first := pzAssessEvent(t, zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
		second := pzAssessEvent(t, first.Zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
		if len(second.Mandatory) != 1 || len(second.Skipped) != 1 || second.Skipped[0].Reason != combat.SkipNotFirstEntryThisTurn {
			t.Errorf("mandatory %d, skipped %+v, want 1 due and the NotFirstEntryThisTurn skip", len(second.Mandatory), second.Skipped)
		}
	})
}

// Requirement: doc/specs/persistent-spell-zones.md FR-028 - a zone whose only entry rule is FirstEntryOnTurn records the entry (the second entry that turn is skipped), and a zone with later triggers but no FirstEntryOnTurn rule records none (mutation gap: the entry test was only run on a zone that also had a later trigger).
// Path: unhappy (boundary).
func TestMutgapEntryLedgerFollowsTheFirstEntryRule(t *testing.T) {
	t.Parallel()
	x := pzInBody("x")
	only := pzCreate(t, pzSpell("only", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureFirstEntryOnTurn, 0, content.AnyCreature)}}), nil), nil)
	first := pzAssessEvent(t, only, pzTurn(1, "x"), combat.NewCreatureEntered(x))
	second := pzAssessEvent(t, first.Zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
	if len(second.Mandatory) != 0 || len(second.Skipped) != 1 || second.Skipped[0].Reason != combat.SkipNotFirstEntryThisTurn {
		t.Errorf("second entry = %+v, want NotFirstEntryThisTurn", second.Skipped)
	}
	later := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature), pzRule(content.CreatureEndsTurnInside, 1, content.AnyCreature)}})
	got := pzAssessEvent(t, pzCreate(t, pzSpell("later", 2, later, nil), nil), pzTurn(1, "x"), combat.NewCreatureEntered(x))
	if got.Zone.Ledger().Size() != 0 {
		t.Errorf("a zone with no FirstEntryOnTurn rule recorded %d entries, want 0", got.Zone.Ledger().Size())
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-013, FR-014 - each trigger answers one event: ZoneMovesIntoSpace only a ZoneMovedIntoSpace, TravelsWithin only a CreatureTravelled; an event of another kind gives no due effect and no skip, and a stopping zone stops the mover only when the event is a ZoneMovedIntoSpace (mutation gap: the comparisons on the trigger and the && on the stopping rule).
// Path: unhappy (rule interaction).
func TestMutgapTriggersMatchOnlyTheirOwnEvent(t *testing.T) {
	t.Parallel()
	x := pzInBody("x")
	travelled := luMust(combat.NewCreatureTravelled(x, 10))
	travelRule := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzTravel(0, 5)}})
	entersRule := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}})
	cases := []struct {
		name  string
		facts content.ZoneFacts
		event combat.ZoneTriggerEvent
	}{
		{"a TravelsWithin rule ignores ZoneMovedIntoSpace", travelRule, combat.NewZoneMovedIntoSpace(x)},
		{"a CreatureEnters rule ignores CreatureTravelled", entersRule, travelled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := pzAssessEvent(t, pzCreate(t, pzSpell("z", 1, c.facts, nil), nil), pzTurn(1, "x"), c.event)
			if len(got.Mandatory) != 0 || len(got.Skipped) != 0 {
				t.Errorf("mandatory %d, skipped %+v, want none", len(got.Mandatory), got.Skipped)
			}
		})
	}
	stopping := func(on bool) content.ZoneFacts {
		return pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.ZoneMovesIntoSpace, 0, content.AnyCreature)}, StopsMoverOnContact: on})
	}
	for _, c := range []struct {
		name  string
		on    bool
		event combat.ZoneTriggerEvent
		want  bool
	}{
		{"a stopping zone stops a mover moved into it", true, combat.NewZoneMovedIntoSpace(x), true},
		{"a stopping zone does not stop on an entry", true, combat.NewCreatureEntered(x), false},
		{"a zone that does not stop leaves the mover", false, combat.NewZoneMovedIntoSpace(x), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := pzAssessEvent(t, pzCreate(t, pzSpell("z", 1, stopping(c.on), nil), nil), pzTurn(1, "x"), c.event)
			if got.ZoneStoppedForTurn != c.want {
				t.Errorf("ZoneStoppedForTurn = %v, want %v", got.ZoneStoppedForTurn, c.want)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-008, FR-011 - the caster of a moving Emanation is excluded as the origin whatever its id sorts like against another creature's, and a rule limited to creatures the caster can see affects a seen creature that is not an enemy (mutation gaps: the id and scope comparisons).
// Path: unhappy (boundary).
func TestMutgapOriginAndScopeComparisons(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	got := pzAssessEvent(t, zone, pzTurn(2, "ally"), combat.NewCreatureEndedTurn(pzInBody("ally"))) // "ally" sorts before the caster "caster"
	if len(got.Mandatory) != 1 {
		t.Errorf("a creature that is not the origin: dues = %d, want 1", len(got.Mandatory))
	}
	sees := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEndsTurnInside, 0, content.CreaturesCasterCanSee)}})
	seen := pzAssessEvent(t, pzCreate(t, pzSpell("sees", 1, sees, nil), nil), pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzCreature("x", false, true, "body")))
	if len(seen.Mandatory) != 1 || len(seen.Skipped) != 0 {
		t.Errorf("a seen creature that is not an enemy: dues %d, skipped %+v, want 1 and none", len(seen.Mandatory), seen.Skipped)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-023 - ApplyOptionalZoneEffect records the creature only for a once-per-turn zone, applies again on a later turn, and refuses a due effect of another zone, spell or caster whichever way its id sorts (mutation gaps: the once-per-turn guard, the repeat on a later turn and the id comparisons).
// Path: unhappy (rule interaction).
func TestMutgapApplyOptionalZoneEffect(t *testing.T) {
	t.Parallel()
	opt := func(once bool) content.ZoneFacts {
		return pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
			luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, Scope: content.AnyCreature, CasterChoice: true}))}, OncePerTurn: once})
	}
	x := pzInBody("x")
	dueOf := func(zone combat.PersistentZone, round int) combat.ZoneEffectDue {
		return pzAssessEvent(t, zone, pzTurn(round, "x"), combat.NewCreatureEndedTurn(x)).Optional[0]
	}

	free := pzCreate(t, pzSpell("free", 1, opt(false), nil), nil)
	if a, ok := combat.ApplyOptionalZoneEffect(free, pzTurn(2, "x"), dueOf(free, 2)).(combat.OptionalZoneEffectApplied); !ok || a.Zone.Ledger().Size() != 0 {
		t.Errorf("a zone that is not once per turn: %#v, want applied with an empty ledger", a)
	}

	once := pzCreate(t, pzSpell("once", 1, opt(true), nil), nil)
	first, ok := combat.ApplyOptionalZoneEffect(once, pzTurn(2, "x"), dueOf(once, 2)).(combat.OptionalZoneEffectApplied)
	if !ok {
		t.Fatal("the first application was refused")
	}
	if _, ok := combat.ApplyOptionalZoneEffect(first.Zone, pzTurn(3, "x"), dueOf(once, 3)).(combat.OptionalZoneEffectApplied); !ok {
		t.Error("the same creature on a later turn was refused")
	}

	home := pzCreate(t, pzSpell("home", 1, opt(true), nil), nil)
	others := map[string]combat.PersistentZone{
		"zone id before": pzCreate(t, pzSpell("home", 1, opt(true), nil), func(c *combat.CreateZoneCommand) { c.ID = luMust(combat.NewPersistentZoneID("aaa")) }),
		"zone id after":  pzCreate(t, pzSpell("home", 1, opt(true), nil), func(c *combat.CreateZoneCommand) { c.ID = luMust(combat.NewPersistentZoneID("zzz")) }),
		"spell before":   pzCreate(t, pzSpell("aaa", 1, opt(true), nil), nil),
		"spell after":    pzCreate(t, pzSpell("zzz", 1, opt(true), nil), nil),
		"caster before":  pzCreate(t, pzSpell("home", 1, opt(true), nil), func(c *combat.CreateZoneCommand) { c.Caster = combat.MustCombatantID("aaa") }),
		"caster after":   pzCreate(t, pzSpell("home", 1, opt(true), nil), func(c *combat.CreateZoneCommand) { c.Caster = combat.MustCombatantID("zzz") }),
	}
	for name, other := range others {
		got := combat.ApplyOptionalZoneEffect(home, pzTurn(2, "x"), dueOf(other, 2))
		if r, ok := got.(combat.OptionalZoneEffectRefused); !ok || r.Reason != combat.SkipDueNotForThisZone {
			t.Errorf("a due effect of a zone with %s = %#v, want refused DueNotForThisZone", name, got)
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-165 - a Multiattack start, additional use or replacement, and a stat block use, made with the limited-use state of another creature is an error matching srderr.ErrInvalidInput (the call is invalid, not a rule refusal), and a request that fails it changes nothing (mutation gap: the error of the ledger's Multiattack step was swallowed).
// Path: unhappy (invalid input).
func TestMutgapMultiattackStepWithAnotherCreaturesLimitedUse(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil))
	m := sbMonster(t, &plan, sbPlain("Slam"), nil)
	state := luStart(t, m, combat.OutsideLair)
	e := sbEncounter(t)
	begun := resAs[combat.TakeCombatantActionCompleted](t, sbTake(t, e, combat.ActionLedger{}, combat.BeginMultiattack{Monster: m, LimitedUse: state}))
	slam := sbAction(t, m, "Slam")
	foreign := luMust(combat.StartLimitedUse(m, combat.MustCombatantID("someone-else"), combat.OutsideLair))
	for _, req := range []combat.TurnRequest{
		combat.RecordMultiattackUse{Part: slam, LimitedUse: foreign},
		combat.RecordMultiattackReplacement{Replaced: slam, LimitedUse: foreign},
		combat.BeginMultiattack{Monster: m, LimitedUse: foreign},
		combat.UseStatBlockPart{Monster: m, Request: sbRequest(combat.StatBlockCostAction, slam), LimitedUse: foreign},
	} {
		cmd, err := combat.NewTakeCombatantActionCommand(begun.Encounter(), begun.Ledger(), cid(luSelf), req)
		if err != nil {
			t.Fatalf("NewTakeCombatantActionCommand(%T): %v", req, err)
		}
		res, err := combat.NewTakeCombatantAction().Run(context.Background(), cmd)
		if !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%T with another creature's limited-use state = %v, %v, want a nil result and ErrInvalidInput", req, res, err)
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-017, FR-028 - the first-entry skip belongs to a FirstEntryOnTurn rule only: a creature that entered the zone this turn is still affected by an EndsTurnInside rule of the same zone on the same turn (mutation gap: the trigger comparison of the first-entry skip also caught every later trigger).
// Path: unhappy (rule interaction).
func TestMutgapEntryThisTurnDoesNotSkipALaterTrigger(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRule(content.CreatureFirstEntryOnTurn, 0, content.AnyCreature), pzRule(content.CreatureEndsTurnInside, 1, content.AnyCreature)}})
	x := pzInBody("x")
	zone := pzCreate(t, pzSpell("entry", 2, facts, nil), nil)
	entered := pzAssessEvent(t, zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
	ended := pzAssessEvent(t, entered.Zone, pzTurn(1, "x"), combat.NewCreatureEndedTurn(x))
	if len(ended.Mandatory) != 1 || len(ended.Skipped) != 0 {
		t.Errorf("the turn ends after an entry on the same turn: dues %d, skipped %+v, want 1 and none", len(ended.Mandatory), ended.Skipped)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-023 - a due effect names an optional part of the zone only when the rule of that very part has the caster's choice: a zone whose optional rule is on a lower part does not accept the due effect of a higher part (mutation gap: the part comparison of namesOptionalPart only had its equal case tested).
// Path: unhappy (boundary).
func TestMutgapOptionalPartMustBeTheDuesOwnPart(t *testing.T) {
	t.Parallel()
	optional := func(part int) content.ZoneFacts {
		return pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
			luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, PartIndex: part, Scope: content.AnyCreature, CasterChoice: true}))}})
	}
	x := pzInBody("x")
	home := pzCreate(t, pzSpell("home", 2, optional(0), nil), nil)
	other := pzCreate(t, pzSpell("home", 2, optional(1), nil), nil) // the same ids; its optional rule is on part 1
	due := pzAssessEvent(t, other, pzTurn(2, "x"), combat.NewCreatureEndedTurn(x)).Optional[0]
	got := combat.ApplyOptionalZoneEffect(home, pzTurn(2, "x"), due)
	if r, ok := got.(combat.OptionalZoneEffectRefused); !ok || r.Reason != combat.SkipDueNotForThisZone {
		t.Errorf("a due effect of part 1 against a zone whose optional rule is on part 0 = %#v, want refused DueNotForThisZone", got)
	}
}
