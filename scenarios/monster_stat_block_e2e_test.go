package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The refs of grpCBrute, the monster of these scenarios.
var (
	grpCMultiattack = grpCRef(combat.MonsterPartActions, 0)
	grpCClaw        = grpCRef(combat.MonsterPartActions, 1)
	grpCTail        = grpCRef(combat.MonsterPartActions, 2)
	grpCSpray       = grpCRef(combat.MonsterPartActions, 3)
	grpCMind        = grpCRef(combat.MonsterPartActions, 4)
	grpCResistance  = grpCRef(combat.MonsterPartTraits, 0)
	grpCRush        = grpCRef(combat.MonsterPartBonusActions, 0)
	grpCParry       = grpCRef(combat.MonsterPartReactions, 0)
	grpCRiposte     = grpCRef(combat.MonsterPartReactions, 1)
	grpCLegendaryMv = grpCRef(combat.MonsterPartLegendaryActions, 0)
)

const grpCBruteID = "brute"

// grpCBrute is a synthetic monster with every kind of limited part and a Multiattack: Resistance 3/Day (4/Day in
// Lair), Actions Multiattack, Claw, Tail, Spray (Recharge 5-6) and Mind (2/Day), a Bonus Action, a Reaction
// without a limit and a 1/Day Reaction, and Legendary Actions 3 (4 in Lair).
func grpCBrute() content.Monster {
	group := func(n int, parts ...string) content.AttackSlotGroup {
		return grpCMust(content.NewAttackSlotGroup(n, parts))
	}
	rule := grpCMust(content.NewReplacementRule(1, grpCMust(content.NewReplacementScopeOneAttackPart("Claw")),
		[]content.ReplacementOption{grpCMust(content.NewReplacementOption("Spray", "", 0))}, content.ReplaceOnce))
	plan := grpCMust(content.NewMultiattackPlan([]content.MultiattackVariant{
		grpCMust(content.NewMultiattackVariant("", []content.AttackSlotGroup{group(1, "Claw"), group(1, "Tail")},
			[]content.AdditionalUse{grpCMust(content.NewAdditionalUse([]string{"Mind"}))}, []content.ReplacementRule{rule})),
		grpCMust(content.NewMultiattackVariant("", []content.AttackSlotGroup{group(2, "Claw")}, nil, nil)),
	}))
	return grpCMonster(func(f *content.MonsterFields) {
		f.Traits = []content.StatBlockPart{grpCUsage("Resistance", grpCPerDay(3, 4))}
		f.Actions = []content.StatBlockPart{
			grpCPart(content.MultiattackName, content.StatBlockPartFields{Plan: plan}),
			grpCUsage("Claw", content.PartUnlimited{}), grpCUsage("Tail", content.PartUnlimited{}),
			grpCUsage("Spray", grpCRecharge(5, 6)), grpCUsage("Mind", grpCPerDay(2, 0)),
		}
		f.BonusActions = []content.StatBlockPart{grpCUsage("Rush", content.PartUnlimited{})}
		f.Reactions = []content.StatBlockPart{grpCUsage("Parry", content.PartUnlimited{}), grpCUsage("Riposte", grpCPerDay(1, 0))}
		f.LegendaryActions = grpCMust(content.NewLegendaryActions(3, 4,
			[]content.StatBlockPart{grpCUsage("Legendary Move", content.PartUnlimited{})}))
	})
}

func grpCStart(t *testing.T, lair combat.LairStatus) combat.LimitedUseState {
	t.Helper()
	return grpCMust(combat.StartLimitedUse(grpCBrute(), grpCID(grpCBruteID), lair))
}

func grpCSpent(t *testing.T, s combat.LimitedUseState, part combat.LimitedUsePartRef) combat.LimitedUseState {
	t.Helper()
	out, ok := s.Spend(part).(combat.LimitedUseSpent)
	if !ok {
		t.Fatalf("Spend(%v) was refused", part)
	}
	return out.Transition.State()
}

func grpCTurnStart(t *testing.T, roller combat.Roller, s combat.LimitedUseState) combat.MonsterTurnStartApplied {
	t.Helper()
	uc, err := combat.NewApplyMonsterTurnStart(roller)
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), combat.ApplyMonsterTurnStartCommand{State: s})
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.MonsterTurnStartApplied)
}

func grpCWindow(who string, round int) combat.LegendaryActionWindow {
	return grpCMust(combat.NewLegendaryActionWindow(grpCID(who), round))
}

func grpCLegendaryLeft(t *testing.T, s combat.LimitedUseState) int {
	t.Helper()
	l, ok := s.Legendary().(combat.LegendaryTracked)
	if !ok {
		t.Fatalf("legendary track = %#v, want tracked", s.Legendary())
	}
	return l.Remaining()
}

func grpCLegendary(t *testing.T, s combat.LimitedUseState, w combat.LegendaryActionWindow, a combat.ConditionActionAvailability) (combat.LimitedUseState, combat.LegendaryActionRefusal) {
	t.Helper()
	out, err := s.TakeLegendaryAction(w, a)
	if err != nil {
		t.Fatal(err)
	}
	switch v := out.(type) {
	case combat.LegendaryActionAllowed:
		return v.Transition.State(), 0
	case combat.LegendaryActionRefused:
		return s, v.Reason
	}
	t.Fatalf("outcome = %#v", out)
	return s, 0
}

// Requirement: doc/specs/monster-limited-use.md FR-009, FR-010, FR-011, FR-012, FR-024, FR-035, FR-036 - a Recharge 5-6 action is spent, then rolled for with one d6 at the start of each of the monster's turns: a 4 leaves it spent, a 5 readies it; a turn with nothing to roll asks the roller for nothing.
// Path: happy.
func TestGrpCMonsterRechargeAcrossTurns(t *testing.T) {
	t.Parallel()
	s := grpCStart(t, combat.OutsideLair)
	if got := s.AvailabilityOf(grpCSpray); got != combat.UseAvailable {
		t.Fatalf("Spray availability at start = %v, want available", got)
	}

	idle := dicetest.NewMinimumRoller()
	if got := grpCTurnStart(t, idle, s); len(idle.Requests()) != 0 || len(got.Rolls()) != 0 || len(got.Events()) != 0 {
		t.Errorf("a turn with nothing to recharge: %d requests, %d rolls, events %v, want none", len(idle.Requests()), len(got.Rolls()), grpCKindsOf(got.Events()))
	}

	s = grpCSpent(t, s, grpCSpray)
	if out, ok := s.Spend(grpCSpray).(combat.LimitedUseSpendRefused); !ok || out.Reason != combat.LimitedUseRefusedExhausted {
		t.Fatalf("a second Spray = %#v, want refused as exhausted", out)
	}

	miss := dicetest.NewFixedRoller(4)
	turn := grpCTurnStart(t, miss, s)
	if reqs := miss.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D6(), 0) {
		t.Errorf("requests = %v, want one 1d6", reqs)
	}
	if r := turn.Rolls(); len(r) != 1 || r[0].Face() != 4 || r[0].Result() != combat.RechargeNotRegained || r[0].Part() != grpCSpray {
		t.Fatalf("rolls = %v, want a 4 on Spray that did not recharge", r)
	}
	s = turn.State()
	if s.AvailabilityOf(grpCSpray) != combat.UseExhausted {
		t.Errorf("Spray after a 4 = %v, want still spent", s.AvailabilityOf(grpCSpray))
	}

	hit := grpCTurnStart(t, dicetest.NewFixedRoller(5), s)
	if r := hit.Rolls(); len(r) != 1 || r[0].Result() != combat.RechargeRegained {
		t.Fatalf("rolls = %v, want a 5 that recharged", r)
	}
	if hit.State().AvailabilityOf(grpCSpray) != combat.UseAvailable {
		t.Errorf("Spray after a 5 = %v, want ready", hit.State().AvailabilityOf(grpCSpray))
	}
	if !slices.Equal(grpCKindsOf(hit.Events()), []string{"monster_recharge_rolled"}) {
		t.Errorf("events = %v, want one monster_recharge_rolled", grpCKindsOf(hit.Events()))
	}
}

func grpCKindsOf(events []combat.MonsterLimitedUseEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// Requirement: doc/specs/monster-limited-use.md FR-035, FR-036, FR-037, FR-012 - the turn start use case returns a roller failure wrapped as a dice port failure with no state, refuses a nil roller and the zero state as invalid input, and does not roll for a state that needs no roll.
// Path: unhappy (failure).
func TestGrpCMonsterTurnStartFailures(t *testing.T) {
	t.Parallel()
	s := grpCSpent(t, grpCStart(t, combat.OutsideLair), grpCSpray)
	cause := errors.New("dice service down")
	uc, err := combat.NewApplyMonsterTurnStart(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }))
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), combat.ApplyMonsterTurnStartCommand{State: s})
	var port *srderr.PortError
	if res != nil || !errors.As(err, &port) || port.Port != dice.Port || !errors.Is(err, cause) {
		t.Errorf("roller failure = %v, %v, want no result and a dice PortError wrapping the cause", res, err)
	}
	if _, err := combat.NewApplyMonsterTurnStart(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Run(context.Background(), combat.ApplyMonsterTurnStartCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero state error = %v, want ErrInvalidInput", err)
	}
	// A roll the host builds by hand must match what is needed: a state with a spent Recharge part and no roll is invalid.
	if _, err := s.StartOfOwnTurn(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("StartOfOwnTurn with no roll for a spent part = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-015 to FR-023, FR-022, FR-024 - Legendary Actions are taken after another creature's turn, one per window, three in all; the refusals come in the printed order; the uses return in full at the start of the monster's own turn, which says so in an event.
// Path: happy and unhappy (rule interaction).
func TestGrpCMonsterLegendaryActionsAcrossARound(t *testing.T) {
	t.Parallel()
	avail := combat.ConditionActionAvailabilityAvailable
	s := grpCStart(t, combat.OutsideLair)
	if grpCLegendaryLeft(t, s) != 3 {
		t.Fatalf("uses at start = %d, want 3", grpCLegendaryLeft(t, s))
	}

	// The monster's own turn is no window; an Incapacitated monster takes none.
	if _, why := grpCLegendary(t, s, grpCWindow(grpCBruteID, 1), avail); why != combat.LegendaryRefusedOwnTurn {
		t.Errorf("own turn = %v, want own_turn", why)
	}
	if _, why := grpCLegendary(t, s, grpCWindow("hero", 1), combat.ConditionActionAvailabilityNone); why != combat.LegendaryRefusedIncapacitated {
		t.Errorf("incapacitated = %v, want incapacitated", why)
	}
	// The own-turn refusal outranks the incapacitated one (the order of FR-021).
	if _, why := grpCLegendary(t, s, grpCWindow(grpCBruteID, 1), combat.ConditionActionAvailabilityNone); why != combat.LegendaryRefusedOwnTurn {
		t.Errorf("own turn while incapacitated = %v, want own_turn first", why)
	}

	s, why := grpCLegendary(t, s, grpCWindow("hero", 1), avail)
	if why != 0 || grpCLegendaryLeft(t, s) != 2 {
		t.Fatalf("first action: refusal %v, %d left, want taken with 2 left", why, grpCLegendaryLeft(t, s))
	}
	if _, why := grpCLegendary(t, s, grpCWindow("hero", 1), avail); why != combat.LegendaryRefusedWindowUsed {
		t.Errorf("a second action in one window = %v, want window_already_used", why)
	}
	s, _ = grpCLegendary(t, s, grpCWindow("ally", 1), avail)
	s, _ = grpCLegendary(t, s, grpCWindow("hero", 2), avail)
	if grpCLegendaryLeft(t, s) != 0 {
		t.Fatalf("uses after three = %d, want 0", grpCLegendaryLeft(t, s))
	}
	if _, why := grpCLegendary(t, s, grpCWindow("ally", 2), avail); why != combat.LegendaryRefusedNoUses {
		t.Errorf("a fourth action = %v, want no_uses_remaining", why)
	}
	// A rest does not return Legendary Action uses; the start of the monster's own turn does.
	if got := grpCLegendaryLeft(t, s.AfterRest(character.LongRest).State()); got != 0 {
		t.Errorf("uses after a Long Rest = %d, want 0 (FR-023)", got)
	}
	turn := grpCTurnStart(t, dicetest.NewMinimumRoller(), s)
	if grpCLegendaryLeft(t, turn.State()) != 3 {
		t.Errorf("uses at the monster's turn = %d, want 3", grpCLegendaryLeft(t, turn.State()))
	}
	if !slices.Equal(grpCKindsOf(turn.Events()), []string{"legendary_action_uses_regained"}) {
		t.Errorf("turn events = %v, want legendary_action_uses_regained", grpCKindsOf(turn.Events()))
	}

	// A monster with no Legendary Actions refuses first; a zero window and an undefined availability are invalid input.
	plain := grpCMust(combat.StartLimitedUse(grpCMonster(nil), grpCID("plain"), combat.OutsideLair))
	if _, why := grpCLegendary(t, plain, grpCWindow("hero", 1), avail); why != combat.LegendaryRefusedNoActions {
		t.Errorf("no Legendary Actions = %v, want no_legendary_actions", why)
	}
	if _, err := s.TakeLegendaryAction(combat.LegendaryActionWindow{}, avail); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero window error = %v, want ErrInvalidInput", err)
	}
	if _, err := s.TakeLegendaryAction(grpCWindow("hero", 1), 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("undefined availability error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-006, FR-007, FR-008, FR-025, FR-026, FR-027, FR-028, FR-002, FR-004 - a 2/Day action is spent twice and then refused; a Short Rest readies a Recharge action but not the X/Day action; a Long Rest restores both and says what it restored; moving into the lair raises the maxima and keeps the uses already spent, and reporting the same lair status changes nothing.
// Path: happy and unhappy (rule interaction).
func TestGrpCMonsterRestsAndLairStatus(t *testing.T) {
	t.Parallel()
	s := grpCStart(t, combat.OutsideLair)
	s = grpCSpent(t, grpCSpent(t, s, grpCMind), grpCMind)
	s = grpCSpent(t, s, grpCSpray)
	if out, ok := s.Spend(grpCMind).(combat.LimitedUseSpendRefused); !ok || out.Reason != combat.LimitedUseRefusedExhausted {
		t.Fatalf("a third Mind = %#v, want refused as exhausted", out)
	}
	if out, ok := s.Spend(grpCClaw).(combat.LimitedUseSpendRefused); !ok || out.Reason != combat.LimitedUseRefusedNotLimited {
		t.Fatalf("Claw has no limit: %#v, want refused as not limited", out)
	}

	short := s.AfterRest(character.ShortRest)
	if short.State().AvailabilityOf(grpCMind) != combat.UseExhausted || short.State().AvailabilityOf(grpCSpray) != combat.UseAvailable {
		t.Errorf("after a Short Rest: Mind %v Spray %v, want Mind still exhausted and Spray ready", short.State().AvailabilityOf(grpCMind), short.State().AvailabilityOf(grpCSpray))
	}
	long := s.AfterRest(character.LongRest)
	if long.State().AvailabilityOf(grpCMind) != combat.UseAvailable || long.State().AvailabilityOf(grpCSpray) != combat.UseAvailable {
		t.Errorf("after a Long Rest: Mind %v Spray %v, want both available", long.State().AvailabilityOf(grpCMind), long.State().AvailabilityOf(grpCSpray))
	}
	ev := long.Events()
	if len(ev) != 1 {
		t.Fatalf("Long Rest events = %v, want one", grpCKindsOf(ev))
	}
	restored, ok := ev[0].(combat.LimitedUsesRestored)
	if !ok || len(restored.Subjects()) != 2 || restored.Subjects()[0] != (combat.PartSubject{Part: grpCSpray}) || restored.Subjects()[1] != (combat.PartSubject{Part: grpCMind}) {
		t.Errorf("restored = %#v, want Spray then Mind in printed order", ev[0])
	}
	if again := long.State().AfterRest(character.LongRest); len(again.Events()) != 0 {
		t.Errorf("a second Long Rest produced %v, want no event", grpCKindsOf(again.Events()))
	}

	// Resistance 3/Day, 4/Day in Lair: one use spent outside the lair leaves 2; the lair keeps the one spent (4 - 1 = 3).
	res := grpCSpent(t, grpCStart(t, combat.OutsideLair), grpCResistance)
	perDay := func(st combat.LimitedUseState) combat.PerDayEntry {
		for _, e := range st.Entries() {
			if p, ok := e.(combat.PerDayEntry); ok && p.Part() == grpCResistance {
				return p
			}
		}
		t.Fatal("no Resistance entry")
		return combat.PerDayEntry{}
	}
	if perDay(res).Remaining() != 2 || perDay(res).Maximum() != 3 {
		t.Fatalf("Resistance outside the lair = %d of %d, want 2 of 3", perDay(res).Remaining(), perDay(res).Maximum())
	}
	in := res.ChangeLair(combat.InLair)
	if perDay(in.State()).Remaining() != 3 || grpCLegendaryLeft(t, in.State()) != 4 {
		t.Errorf("in the lair: Resistance %d left, Legendary %d, want 3 and 4", perDay(in.State()).Remaining(), grpCLegendaryLeft(t, in.State()))
	}
	if !slices.Equal(grpCKindsOf(in.Events()), []string{"monster_lair_status_changed"}) {
		t.Errorf("lair events = %v, want monster_lair_status_changed", grpCKindsOf(in.Events()))
	}
	if same := in.State().ChangeLair(combat.InLair); len(same.Events()) != 0 || !same.State().Equal(in.State()) {
		t.Errorf("reporting the same lair status changed something: %v", grpCKindsOf(same.Events()))
	}
	if bad := in.State().ChangeLair(0); len(bad.Events()) != 0 || !bad.State().Equal(in.State()) {
		t.Errorf("an undefined lair status changed something")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-001, FR-002, FR-003, FR-004, FR-006, FR-007, FR-008, FR-009, FR-005, FR-048 - the use of a part costs what its section says: an Action carries the action choice, a Bonus Action the grant, a Reaction needs its trigger reported and a Legendary Action first passes the legendary rules; a 1/Day Reaction is usable once; the wrong section, a missing part and a Trait are refused and leave the limited-use state as it was.
// Path: happy and unhappy (state).
func TestGrpCMonsterUsesPartsAtTheirCost(t *testing.T) {
	t.Parallel()
	m := grpCBrute()
	self := grpCID(grpCBruteID)
	s := grpCStart(t, combat.OutsideLair)
	use := func(st combat.LimitedUseState, req combat.StatBlockUseRequest) combat.StatBlockUseOutcome {
		t.Helper()
		out, err := combat.UseStatBlock(m, self, req, st)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	act := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: grpCClaw}).(combat.StatBlockUseAllowed).Transition
	if _, ok := act.ActionChoice(); !ok || act.Cost() != combat.StatBlockCostAction {
		t.Errorf("Action use: cost %v choice present %v, want action with a choice", act.Cost(), ok)
	}
	if !slices.Contains(grpCKinds(act.Events()), "stat_block_part_used") {
		t.Errorf("events = %v, want stat_block_part_used", grpCKinds(act.Events()))
	}
	bonus := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostBonusAction, Part: grpCRush}).(combat.StatBlockUseAllowed).Transition
	if _, ok := bonus.BonusActionGrant(); !ok {
		t.Errorf("a Bonus Action use carries no grant")
	}

	// A Reaction needs the trigger: not reported is refused; reported is allowed; a 1/Day Reaction goes once.
	if out, ok := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostReaction, Part: grpCParry, Trigger: combat.ReactionTriggerNotOccurred}).(combat.StatBlockUseRefused); !ok || out.Refusal != (combat.StatBlockReactionTriggerNotReported{}) {
		t.Errorf("a Reaction with no trigger = %#v, want refused for the trigger", out)
	}
	first := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostReaction, Part: grpCRiposte, Trigger: combat.ReactionTriggerOccurred}).(combat.StatBlockUseAllowed).Transition
	if _, ok := first.Reaction(); !ok {
		t.Errorf("a Reaction use carries no reaction")
	}
	if out, ok := use(first.LimitedUse(), combat.StatBlockUseRequest{Cost: combat.StatBlockCostReaction, Part: grpCRiposte, Trigger: combat.ReactionTriggerOccurred}).(combat.StatBlockUseRefused); !ok || out.Refusal != (combat.StatBlockPartUsageExhausted{}) {
		t.Errorf("a second 1/Day Reaction = %#v, want refused as exhausted", out)
	}

	// Wrong section, missing part and a passive Trait.
	if out, ok := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostBonusAction, Part: grpCClaw}).(combat.StatBlockUseRefused); !ok || out.Refusal != (combat.StatBlockPartInWrongSection{}) {
		t.Errorf("an Action used as a Bonus Action = %#v, want wrong section", out)
	}
	if out, ok := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: grpCRef(combat.MonsterPartActions, 40)}).(combat.StatBlockUseRefused); !ok || out.Refusal != (combat.StatBlockPartNotFound{}) {
		t.Errorf("a part past the list = %#v, want not found", out)
	}
	if out, ok := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: grpCResistance}).(combat.StatBlockUseRefused); !ok || out.Refusal != (combat.StatBlockPartInWrongSection{}) {
		t.Errorf("a Trait used as an action = %#v, want wrong section", out)
	}

	// A Legendary Action first passes the legendary rules: own turn is refused with that reason; another creature's turn is allowed and spends a use.
	if out, ok := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: grpCLegendaryMv, Window: grpCWindow(grpCBruteID, 1),
		Availability: combat.ConditionActionAvailabilityAvailable}).(combat.StatBlockUseRefused); !ok || out.Refusal != (combat.StatBlockLegendaryUseRefused{Reason: combat.LegendaryRefusedOwnTurn}) {
		t.Errorf("a Legendary Action in the own turn = %#v, want the legendary refusal own_turn", out)
	}
	leg := use(s, combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: grpCLegendaryMv, Window: grpCWindow("hero", 1),
		Availability: combat.ConditionActionAvailabilityAvailable}).(combat.StatBlockUseAllowed).Transition
	if grpCLegendaryLeft(t, leg.LimitedUse()) != 2 {
		t.Errorf("uses after a Legendary Action = %d, want 2", grpCLegendaryLeft(t, leg.LimitedUse()))
	}

	// Invalid input is an error, not a refusal: the zero monster and a state of another combatant.
	if _, err := combat.UseStatBlock(content.Monster{}, self, combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: grpCClaw}, s); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero monster error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.UseStatBlock(m, grpCID("someone_else"), combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: grpCClaw}, s); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("state of another combatant error = %v, want ErrInvalidInput", err)
	}

	// The feature id of a part is derived from the monster and the part names.
	id, err := combat.StatBlockFeatureID(m, grpCClaw)
	if err != nil || id == "" {
		t.Errorf("StatBlockFeatureID = %q, %v, want a derived id", id, err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-010, FR-012, FR-013, FR-014, FR-015, FR-016, FR-018, FR-019, FR-020, FR-021 - one turn of Multiattack: the plan is a ceiling across alternative variants, each attack, the additional use (whose own limit is spent) and the replacement are recorded in any order, a step the plan does not allow is refused with its reason and a part whose limit is spent stops the step; a plan with no unresolved names is reported clean.
// Path: happy and unhappy (rule interaction).
func TestGrpCMonsterMultiattackTurn(t *testing.T) {
	t.Parallel()
	m := grpCBrute()
	self := grpCID(grpCBruteID)
	if problems := combat.MultiattackPlanProblemsFor(m); len(problems) != 0 {
		t.Fatalf("plan problems = %v, want none", problems)
	}
	start := func(lu combat.LimitedUseState) combat.MultiattackProgress {
		t.Helper()
		out, err := combat.StartMultiattack(m, self, lu)
		if err != nil {
			t.Fatal(err)
		}
		return out.(combat.MultiattackStarted).Progress()
	}
	step := func(o combat.MultiattackStepOutcome, err error) combat.MultiattackTransition {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		rec, ok := o.(combat.MultiattackStepRecorded)
		if !ok {
			t.Fatalf("step = %#v, want recorded", o)
		}
		return rec.Transition
	}
	lu := grpCStart(t, combat.OutsideLair)
	p := start(lu)
	if p.AttacksRemaining(grpCClaw) == 0 {
		t.Fatalf("Claw attacks at the start = 0, want some")
	}

	// Claw, Tail and one Mind: the first variant. Order is free (Mind first).
	tr := step(p.RecordAdditionalUse(grpCMind, lu))
	p, lu = tr.Progress(), tr.LimitedUse()
	if lu.AvailabilityOf(grpCMind) != combat.UseAvailable { // 2/Day, one spent
		t.Errorf("Mind after one use = %v, want one use left", lu.AvailabilityOf(grpCMind))
	}
	if !slices.Equal(grpCKinds(tr.Events()), []string{"limited_use_part_spent", "multiattack_additional_use_recorded"}) {
		t.Errorf("events = %v, want the spend of Mind then multiattack_additional_use_recorded", grpCKinds(tr.Events()))
	}
	tr = step(p.RecordAttack(grpCTail, lu))
	p, lu = tr.Progress(), tr.LimitedUse()
	tr = step(p.RecordAttack(grpCClaw, lu))
	p, lu = tr.Progress(), tr.LimitedUse()
	if !slices.Equal(grpCKinds(tr.Events()), []string{"multiattack_attack_recorded"}) {
		t.Errorf("events = %v, want multiattack_attack_recorded", grpCKinds(tr.Events()))
	}

	// The ceiling: a second Claw, a second Tail and a second additional use are all refused with their reasons.
	refusal := func(o combat.MultiattackStepOutcome, err error) combat.MultiattackAllowanceRefusal {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		n, ok := o.(combat.MultiattackNotInAllowance)
		if !ok {
			t.Fatalf("step = %#v, want not in allowance", o)
		}
		return n.Reason
	}
	if got := refusal(p.RecordAttack(grpCClaw, lu)); got != combat.MultiattackAllowanceSpent {
		t.Errorf("a second Claw = %v, want allowance_spent", got)
	}
	if got := refusal(p.RecordAdditionalUse(grpCMind, lu)); got != combat.MultiattackAdditionalUseAlreadyMade {
		t.Errorf("a second additional use = %v, want additional_use_already_made", got)
	}
	if got := refusal(p.RecordAttack(grpCSpray, lu)); got != combat.MultiattackAttackNotAllowed {
		t.Errorf("an attack the plan does not list = %v, want attack_not_allowed", got)
	}

	// A replacement: a Spray in place of the Claw spends the Recharge, once; with the Spray already spent the step is stopped by the part gate.
	fresh := grpCStart(t, combat.OutsideLair)
	rp := start(fresh)
	tr = step(rp.RecordReplacement(grpCMust(content.NewReplacementOption("Spray", "", 0)), grpCClaw, fresh))
	if tr.LimitedUse().AvailabilityOf(grpCSpray) != combat.UseExhausted {
		t.Errorf("Spray after the replacement = %v, want spent", tr.LimitedUse().AvailabilityOf(grpCSpray))
	}
	if !slices.Equal(grpCKinds(tr.Events()), []string{"limited_use_part_spent", "multiattack_attack_replaced"}) {
		t.Errorf("events = %v, want the spend of Spray then multiattack_attack_replaced", grpCKinds(tr.Events()))
	}
	spent := grpCSpent(t, fresh, grpCSpray)
	out, err := start(spent).RecordReplacement(grpCMust(content.NewReplacementOption("Spray", "", 0)), grpCClaw, spent)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(combat.MultiattackStepUsageRefused); !ok {
		t.Errorf("a replacement with a spent Spray = %#v, want a usage refusal", out)
	}

	// No Multiattack part: not available. The zero limited-use state is invalid input.
	plain := grpCMonster(nil)
	na, err := combat.StartMultiattack(plain, grpCID("plain"), grpCMust(combat.StartLimitedUse(plain, grpCID("plain"), combat.OutsideLair)))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := na.(combat.MultiattackNotAvailable); !ok {
		t.Errorf("a monster with no Multiattack = %#v, want not available", na)
	}
	if _, err := combat.StartMultiattack(m, self, combat.LimitedUseState{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero limited-use state error = %v, want ErrInvalidInput", err)
	}
	_ = grpCMultiattack
}

// grpCDamage builds a stat-block damage instance of the printed number, the dice and the type.
func grpCDamage(number int, e dice.Expression, typ content.DamageType) content.StatBlockDamage {
	return grpCMust(content.NewStatBlockDamage(number, e, grpCMust(content.NewSpecificDamageType(typ)), false))
}

// grpCBreath is a monster whose one Action, Steam Breath, is a DC 13 Dexterity saving throw for 5 (1d8 + 1) Fire damage and half as much on a success.
func grpCBreath() (content.Monster, combat.LimitedUsePartRef) {
	failure := grpCMust(content.NewEffectClause(content.ClauseFailure, "Failure.", nil, []content.StatBlockDamage{grpCDamage(5, dice.MustExpression(1, dice.D8(), 1), content.DamageFire)}))
	success := grpCMust(content.NewEffectClause(content.ClauseSuccess, "Success.", content.HalfDamageOnly{}, nil))
	st := grpCMust(content.NewStatBlockSavingThrow(ability.Dexterity, 13, "", ""))
	part := grpCPart("Steam Breath", content.StatBlockPartFields{SavingThrow: st, Clauses: []content.EffectClause{failure, success}})
	m := grpCMonster(func(f *content.MonsterFields) { f.Actions = []content.StatBlockPart{part} })
	return m, grpCRef(combat.MonsterPartActions, 0)
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-044, FR-045, FR-033, FR-026, FR-027, FR-032, FR-034, FR-035, FR-037, FR-038 - the damage of one use is rolled once (the dice notation, with the printed flat modifier) and shared by every target that saves: the target that fails takes all of it, the target that succeeds takes half, and neither the number notation nor the saves roll the damage again.
// Path: happy.
func TestGrpCMonsterBreathRolledOnceAndSharedBySaves(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, part := grpCBreath()

	// Rolled by the dice notation: 1d8 shows 6, plus the printed 1 is 7. The number notation would be the printed 5.
	damageOf := grpCMust(combat.NewDamageNotationChoices(combat.DamageNotationDice))
	roller := dicetest.NewFixedRoller(6)
	rollUse, err := combat.NewRollStatBlockDamage(roller)
	if err != nil {
		t.Fatal(err)
	}
	instances := []content.StatBlockDamage{grpCDamage(5, dice.MustExpression(1, dice.D8(), 1), content.DamageFire)}
	res, err := rollUse.Run(ctx, combat.RollStatBlockDamageCommand{Damage: instances, Choices: damageOf})
	if err != nil {
		t.Fatal(err)
	}
	rolled := res.(combat.StatBlockDamageRolled).Damage()
	if len(rolled) != 1 || rolled[0].Amount() != 7 || rolled[0].Type() != content.DamageFire {
		t.Fatalf("rolled = %v, want 7 Fire", rolled)
	}
	if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D8(), 1) {
		t.Errorf("requests = %v, want one request for the printed 1d8+1", reqs)
	}
	numberOnly := grpCMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber))
	idle := dicetest.NewMinimumRoller()
	rollNumber := grpCMust(combat.NewRollStatBlockDamage(idle))
	res, err = rollNumber.Run(ctx, combat.RollStatBlockDamageCommand{Damage: instances, Choices: numberOnly})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.(combat.StatBlockDamageRolled).Damage(); len(got) != 1 || got[0].Amount() != 5 || len(idle.Requests()) != 0 {
		t.Errorf("number notation = %v with %d requests, want the printed 5 and no die", got, len(idle.Requests()))
	}

	// Two heroes save against the same rolled damage: the first fails (a 5), the second succeeds (a 15).
	e := grpCEncounter(t, grpCPlayer(t, "dragon_stand_in", combat.SideB, 50), grpCPlayer(t, "a", combat.SideA, 30), grpCPlayer(t, "b", combat.SideA, 30))
	save := combat.NewResolveStatBlockSave
	for _, c := range []struct {
		target string
		face   int
		lost   int
		want   combat.SaveEffectDamageAdjustment
	}{
		{"a", 5, 7, combat.SaveEffectDamageUnchanged},
		{"b", 15, 3, combat.SaveEffectDamageHalved},
	} {
		cmd, err := combat.NewResolveStatBlockSaveCommand(combat.ResolveStatBlockSaveParams{
			Encounter: e, SourceID: grpCID("dragon_stand_in"), TargetID: grpCID(c.target), Monster: m, Part: part,
			Save: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}, Damage: rolled, Delivery: combat.OtherSource,
		})
		if err != nil {
			t.Fatal(err)
		}
		uc, err := save(dicetest.NewFixedRoller(c.face))
		if err != nil {
			t.Fatal(err)
		}
		out, err := uc.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		done := out.(combat.ResolveStatBlockSaveCompleted)
		applies, ok := done.Effect().(combat.StatBlockSaveEffectApplies)
		if !ok || applies.DamageAdjustment() != c.want {
			t.Errorf("%s: effect = %#v, want applies with %v", c.target, done.Effect(), c.want)
		}
		after, _ := done.Encounter().Find(grpCID(c.target))
		if lost := 30 - after.HitPoints().HitPoints().Current(); lost != c.lost {
			t.Errorf("%s lost %d Hit Points, want %d", c.target, lost, c.lost)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-044, FR-046, FR-027 - a roller failure while rolling the damage is returned as a dice port failure with no result, a command of more than 64 instances is an Invalid result with the limit error and rolls nothing, a command with no instance or a mismatched choice list is invalid input, and a nil roller is invalid input.
// Path: unhappy (failure and boundary).
func TestGrpCMonsterDamageRollFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	one := []content.StatBlockDamage{grpCDamage(5, dice.MustExpression(1, dice.D8(), 1), content.DamageFire)}
	dice1 := grpCMust(combat.NewDamageNotationChoices(combat.DamageNotationDice))

	cause := errors.New("dice service down")
	failing := grpCMust(combat.NewRollStatBlockDamage(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })))
	res, err := failing.Run(ctx, combat.RollStatBlockDamageCommand{Damage: one, Choices: dice1})
	var port *srderr.PortError
	if res != nil || !errors.As(err, &port) || port.Port != dice.Port || !errors.Is(err, cause) {
		t.Errorf("roller failure = %v, %v, want no result and a dice PortError wrapping the cause", res, err)
	}

	roller := dicetest.NewMinimumRoller()
	uc := grpCMust(combat.NewRollStatBlockDamage(roller))
	many := slices.Repeat(one, 65)
	res, err = uc.Run(ctx, combat.RollStatBlockDamageCommand{Damage: many, Choices: grpCMust(combat.NewDamageNotationChoices(slices.Repeat([]combat.DamageNotationChoice{combat.DamageNotationDice}, 65)...))})
	if err != nil {
		t.Fatal(err)
	}
	inv, ok := res.(combat.StatBlockDamageInvalid)
	var limit combat.LimitExceededError
	if !ok || !errors.As(inv.Err(), &limit) || limit.Max != 64 || len(roller.Requests()) != 0 {
		t.Errorf("65 instances = %#v with %d requests, want Invalid with a LimitExceededError of 64 and no roll", res, len(roller.Requests()))
	}

	if _, err := uc.Run(ctx, combat.RollStatBlockDamageCommand{Choices: dice1}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no instance error = %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Run(ctx, combat.RollStatBlockDamageCommand{Damage: slices.Repeat(one, 2), Choices: dice1}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("one choice for two instances error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewRollStatBlockDamage(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewResolveStatBlockSave(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller for the save error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-032, FR-054 - a saving throw command that does not name a part with a Saving Throw notation, or that holds a save that is no attempt, is refused at construction as invalid input, and an unknown target is an Invalid result, not an error.
// Path: unhappy (invalid input).
func TestGrpCMonsterSaveCommandIsValidated(t *testing.T) {
	t.Parallel()
	m, part := grpCBreath()
	e := grpCEncounter(t, grpCPlayer(t, "source", combat.SideB, 50), grpCPlayer(t, "a", combat.SideA, 30))
	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	ok := combat.ResolveStatBlockSaveParams{Encounter: e, SourceID: grpCID("source"), TargetID: grpCID("a"), Monster: m, Part: part, Save: attempt, Delivery: combat.OtherSource}

	noSave := ok
	noSave.Save = nil
	if _, err := combat.NewResolveStatBlockSaveCommand(noSave); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no save error = %v, want ErrInvalidInput", err)
	}
	pastList := ok
	pastList.Part = grpCRef(combat.MonsterPartActions, 9)
	if _, err := combat.NewResolveStatBlockSaveCommand(pastList); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("part past the list error = %v, want ErrInvalidInput", err)
	}
	noNotation := ok
	noNotation.Monster = grpCBrute()
	noNotation.Part = grpCClaw
	if _, err := combat.NewResolveStatBlockSaveCommand(noNotation); err == nil {
		t.Errorf("a part with no Saving Throw notation was accepted")
	}

	ghost := ok
	ghost.TargetID = grpCID("ghost")
	cmd, err := combat.NewResolveStatBlockSaveCommand(ghost)
	if err != nil {
		t.Fatal(err)
	}
	roller := dicetest.NewMinimumRoller()
	res, err := grpCMust(combat.NewResolveStatBlockSave(roller)).Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, isInvalid := res.(combat.ResolveStatBlockSaveInvalid); !isInvalid || len(roller.Requests()) != 0 {
		t.Errorf("unknown target = %#v with %d requests, want an Invalid result and no roll", res, len(roller.Requests()))
	}
}
