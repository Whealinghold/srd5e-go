package combat_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sbBeast is a monster with one part of each kind of use: a Trait, Actions (an unlimited Slam, a 2/Day Dominate
// Mind and a Recharge 5-6 Spray), a Bonus Action, Reactions (an unlimited Gaze and a 1/Day Parry) and Legendary
// Actions (3 uses; an unlimited Tail Swipe and a 1/Day Limited Swipe).
func sbBeast(tb testing.TB) content.Monster {
	tb.Helper()
	return luMonster(tb, func(f *content.MonsterFields) {
		f.Traits = sbPlain("Amphibious")
		f.Actions = []content.StatBlockPart{sbPart("Slam", content.StatBlockPartFields{}),
			sbPart("Dominate Mind", content.StatBlockPartFields{Usage: luPerDay(2, 0)}),
			sbPart("Spray", content.StatBlockPartFields{Usage: luRecharge(5, 6)})}
		f.BonusActions = sbPlain("Boar Rush")
		f.Reactions = []content.StatBlockPart{sbPart("Unnerving Gaze", content.StatBlockPartFields{}),
			sbPart("Parry", content.StatBlockPartFields{Usage: luPerDay(1, 0)})}
		f.LegendaryActions = luMust(content.NewLegendaryActions(3, 0, []content.StatBlockPart{
			sbPart("Tail Swipe", content.StatBlockPartFields{}), sbPart("Limited Swipe", content.StatBlockPartFields{Usage: luPerDay(1, 0)})}))
	})
}

func sbUse(tb testing.TB, m content.Monster, req combat.StatBlockUseRequest, s combat.LimitedUseState) combat.StatBlockUseOutcome {
	tb.Helper()
	out, err := combat.UseStatBlock(m, sbSelf(), req, s)
	if err != nil {
		tb.Fatalf("UseStatBlock(%+v) error = %v", req, err)
	}
	return out
}

func sbAllowed(tb testing.TB, out combat.StatBlockUseOutcome) combat.StatBlockUseTransition {
	tb.Helper()
	a, ok := out.(combat.StatBlockUseAllowed)
	if !ok {
		tb.Fatalf("outcome = %#v, want StatBlockUseAllowed", out)
	}
	return a.Transition
}

func sbRefused(tb testing.TB, out combat.StatBlockUseOutcome) combat.StatBlockUseRefusal {
	tb.Helper()
	r, ok := out.(combat.StatBlockUseRefused)
	if !ok {
		tb.Fatalf("outcome = %#v, want StatBlockUseRefused", out)
	}
	return r.Refusal
}

func sbRequest(cost combat.StatBlockActionCost, part combat.LimitedUsePartRef) combat.StatBlockUseRequest {
	return combat.StatBlockUseRequest{Cost: cost, Part: part, Trigger: combat.ReactionTriggerOccurred}
}

func sbLegendaryRequest(tb testing.TB, part combat.LimitedUsePartRef, who string) combat.StatBlockUseRequest {
	tb.Helper()
	return combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: part,
		Window: luWindow(tb, who, 1), Availability: combat.ConditionActionAvailabilityAvailable}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-001, table EC-1 "cost per section" - the cost of a
// section is passive for Traits, action for Actions, bonus action, reaction and legendary action for the others
// (SRD 5.2.1, "Traits", "Actions", p. 256; "Bonus Action", "Reactions", "Legendary Actions", p. 257).
// Path: per case.
func TestStatBlockActionCostOfSection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		section                 combat.MonsterPartSection
		want                    combat.StatBlockActionCost
		wantName                string
	}{
		{"traits", `FR-001; "Traits", p. 256`, "happy", combat.MonsterPartTraits, combat.StatBlockCostPassive, "passive"},
		{"actions", `FR-001; "Actions", p. 256`, "happy", combat.MonsterPartActions, combat.StatBlockCostAction, "action"},
		{"bonus actions", `FR-001; "Bonus Action", p. 257`, "happy", combat.MonsterPartBonusActions, combat.StatBlockCostBonusAction, "bonus_action"},
		{"reactions", `FR-001; "Reactions", p. 257`, "happy", combat.MonsterPartReactions, combat.StatBlockCostReaction, "reaction"},
		{"legendary actions", `FR-001; "Legendary Actions", p. 257`, "happy", combat.MonsterPartLegendaryActions, combat.StatBlockCostLegendaryAction, "legendary_action"},
		{"undefined section", `FR-001`, "unhappy (invalid input)", 0, 0, "unknown"},
		{"section past the last", `FR-001`, "unhappy (boundary)", combat.MonsterPartLegendaryActions + 1, 0, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.StatBlockActionCostOf(c.section)
			if got != c.want || got.String() != c.wantName || got.Valid() != (c.want != 0) {
				t.Errorf("StatBlockActionCostOf(%v) = %v (%q, valid %v), want %v (%q) [%s]", c.section, got, got.String(), got.Valid(), c.want, c.wantName, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-002, table EC-1 "action entry" - using an Actions part
// consumes the action: the outcome carries the FeatureActionChoice naming the part's FeatureID with NoStandardAction
// by default, or the granted action given (SRD 5.2.1, "Actions", p. 256).
// Path: happy.
func TestUseStatBlockActionConsumesTheAction(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	slam := sbAction(t, m, "Slam")
	state := luStart(t, m, combat.OutsideLair)
	tr := sbAllowed(t, sbUse(t, m, sbRequest(combat.StatBlockCostAction, slam), state))
	choice, ok := tr.ActionChoice()
	if !ok || choice.Feature != content.MustFeatureID("fixture_beast__slam") || choice.Granted != (combat.NoStandardAction{}) {
		t.Errorf("ActionChoice() = %+v, %v, want the Slam feature with NoStandardAction", choice, ok)
	}
	if _, ok := tr.BonusActionGrant(); ok {
		t.Error("an action also reports a Bonus Action grant")
	}
	if _, ok := tr.Reaction(); ok {
		t.Error("an action also reports a Reaction")
	}
	if tr.Cost() != combat.StatBlockCostAction || !tr.LimitedUse().Equal(state) {
		t.Errorf("cost = %v, limited use changed = %v, want an action and an unchanged state", tr.Cost(), !tr.LimitedUse().Equal(state))
	}
	dash := combat.GrantsDash{Speed: combat.SpeedNameWalk}
	req := sbRequest(combat.StatBlockCostAction, slam)
	req.Granted = dash
	got, _ := sbAllowed(t, sbUse(t, m, req, state)).ActionChoice()
	if got.Granted != combat.GrantedAction(dash) {
		t.Errorf("granted = %#v, want the Dash the request gave", got.Granted)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-003, table EC-1 "bonus action" and "no bonus section" - using a
// Bonus Actions part carries a BonusActionGrant naming the part's FeatureID; a monster with no Bonus Actions has no part
// to request (SRD 5.2.1, "Bonus Action", p. 257).
// Path: per case.
func TestUseStatBlockBonusAction(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	tr := sbAllowed(t, sbUse(t, m, sbRequest(combat.StatBlockCostBonusAction, sbRef(t, m, combat.MonsterPartBonusActions, "Boar Rush")), state))
	grant, ok := tr.BonusActionGrant()
	if !ok || grant.Source() != (combat.BonusActionFromFeature{Feature: content.MustFeatureID("fixture_beast__boar_rush")}) || grant.Granted() != (combat.NoStandardAction{}) {
		t.Errorf("BonusActionGrant() = %+v, %v, want the Boar Rush feature", grant, ok)
	}
	if _, ok := tr.ActionChoice(); ok {
		t.Error("a Bonus Action also reports an action")
	}
	none := luMonster(t, nil)
	if got := sbRefused(t, sbUse(t, none, sbRequest(combat.StatBlockCostBonusAction, luRef(t, combat.MonsterPartBonusActions, 0)), luStart(t, none, combat.OutsideLair))); got != (combat.StatBlockPartNotFound{}) {
		t.Errorf("refusal for a monster with no Bonus Actions = %#v, want StatBlockPartNotFound", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-004, table EC-1 "reaction, trigger" and "reaction, no trigger" -
// a Reaction needs the host's report that the trigger occurred; otherwise it is refused and nothing changes (SRD 5.2.1,
// "Reactions", p. 257).
// Path: per case.
func TestUseStatBlockReactionNeedsItsTrigger(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	gaze := sbRef(t, m, combat.MonsterPartReactions, "Unnerving Gaze")
	state := luStart(t, m, combat.OutsideLair)
	tr := sbAllowed(t, sbUse(t, m, sbRequest(combat.StatBlockCostReaction, gaze), state))
	r, ok := tr.Reaction()
	if !ok || r.Feature != content.MustFeatureID("fixture_beast__unnerving_gaze") || r.Granted != (combat.NoStandardAction{}) {
		t.Errorf("Reaction() = %+v, %v, want the Gaze feature", r, ok)
	}
	req := sbRequest(combat.StatBlockCostReaction, gaze)
	req.Trigger = combat.ReactionTriggerNotOccurred
	if got := sbRefused(t, sbUse(t, m, req, state)); got != (combat.StatBlockReactionTriggerNotReported{}) {
		t.Errorf("refusal = %#v, want StatBlockReactionTriggerNotReported", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-006, table EC-1 "wrong section" - a part in the wrong section for
// the kind of use, a Trait among them, is refused; a ref with no part is refused (SRD 5.2.1, "Traits", p. 256).
// Path: per case.
func TestUseStatBlockWrongSectionAndMissingPart(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	cases := []struct {
		name, requirement, path string
		cost                    combat.StatBlockActionCost
		part                    combat.LimitedUsePartRef
		want                    combat.StatBlockUseRefusal
	}{
		{"trait used as an action", `FR-006; "Traits", p. 256`, "unhappy (invalid input)", combat.StatBlockCostAction, luRef(t, combat.MonsterPartTraits, 0), combat.StatBlockPartInWrongSection{}},
		{"reaction requested as a Bonus Action", `FR-006`, "unhappy (invalid input)", combat.StatBlockCostBonusAction, luRef(t, combat.MonsterPartReactions, 0), combat.StatBlockPartInWrongSection{}},
		{"index past the section", `FR-006`, "unhappy (boundary)", combat.StatBlockCostAction, luRef(t, combat.MonsterPartActions, 3), combat.StatBlockPartNotFound{}},
		{"last action is found", `FR-006`, "unhappy (boundary)", combat.StatBlockCostAction, luRef(t, combat.MonsterPartActions, 2), nil},
		{"section with no part", `FR-006`, "unhappy (missing content)", combat.StatBlockCostReaction, luRef(t, combat.MonsterPartReactions, 2), combat.StatBlockPartNotFound{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out := sbUse(t, m, sbRequest(c.cost, c.part), state)
			if c.want == nil {
				sbAllowed(t, out)
				return
			}
			if got := sbRefused(t, out); got != c.want {
				t.Errorf("refusal = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
	if got := sbRefused(t, sbUse(t, m, sbLegendaryRequest(t, luRef(t, combat.MonsterPartActions, 0), "hero"), state)); got != (combat.StatBlockPartInWrongSection{}) {
		t.Errorf("action requested as a Legendary Action = %#v, want StatBlockPartInWrongSection", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-007, table EC-1 "X/Day spend", "Recharge spent" and "unlimited part" -
// a part with a limit is spent exactly once, down to its last use and no further; a spent Recharge part is refused; an unlimited
// part leaves the state equal (SRD 5.2.1, "Limited Usage", p. 257).
// Path: unhappy (boundary).
func TestUseStatBlockSpendsALimitedPart(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	mind := sbAction(t, m, "Dominate Mind")
	state := luStart(t, m, combat.OutsideLair)
	req := sbRequest(combat.StatBlockCostAction, mind)
	first := sbAllowed(t, sbUse(t, m, req, state))
	if got := luPerDayOf(t, first.LimitedUse(), mind).Remaining(); got != 1 {
		t.Errorf("uses after the first = %d, want 1", got)
	}
	second := sbAllowed(t, sbUse(t, m, req, first.LimitedUse()))
	if got := luPerDayOf(t, second.LimitedUse(), mind).Remaining(); got != 0 {
		t.Errorf("uses after the second = %d, want 0 (the last use)", got)
	}
	if got := sbRefused(t, sbUse(t, m, req, second.LimitedUse())); got != (combat.StatBlockPartUsageExhausted{}) {
		t.Errorf("third use = %#v, want StatBlockPartUsageExhausted (one past the last)", got)
	}
	if luPerDayOf(t, state, mind).Remaining() != 2 {
		t.Error("the input state changed")
	}
	spray := sbAction(t, m, "Spray")
	spent := luSpent(t, state, spray)
	if got := sbRefused(t, sbUse(t, m, sbRequest(combat.StatBlockCostAction, spray), spent)); got != (combat.StatBlockPartUsageExhausted{}) {
		t.Errorf("spent Recharge part = %#v, want StatBlockPartUsageExhausted", got)
	}
	if !spent.Equal(luSpent(t, state, spray)) {
		t.Error("a refusal changed the state")
	}
	if got := sbAllowed(t, sbUse(t, m, sbRequest(combat.StatBlockCostAction, sbAction(t, m, "Slam")), state)).LimitedUse(); !got.Equal(state) {
		t.Error("an unlimited part changed the state")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-008, table EC-1 "legendary use", "legendary refused" and "legendary,
// incapacitated" - a Legendary Action first passes the Legendary check and costs none of the action, Bonus Action and Reaction
// (no FeatureActionChoice); every refusal of the check maps one to one and spends nothing; the part's own limit is checked
// after (SRD 5.2.1, "Legendary Actions", p. 257).
// Path: per case.
func TestUseStatBlockLegendaryAction(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	tail := sbRef(t, m, combat.MonsterPartLegendaryActions, "Tail Swipe")
	limited := sbRef(t, m, combat.MonsterPartLegendaryActions, "Limited Swipe")
	state := luStart(t, m, combat.OutsideLair)
	tr := sbAllowed(t, sbUse(t, m, sbLegendaryRequest(t, tail, "hero"), state))
	if _, ok := tr.ActionChoice(); ok || tr.Cost() != combat.StatBlockCostLegendaryAction {
		t.Errorf("a Legendary Action cost %v and reported an action", tr.Cost())
	}
	if got := luLegendaryOf(t, tr.LimitedUse()).Remaining(); got != 2 {
		t.Errorf("Legendary uses after = %d, want 2", got)
	}
	events := tr.Events()
	if len(events) != 2 || events[0].Kind() != "legendary_action_taken" || events[1].Kind() != "stat_block_part_used" {
		t.Errorf("events = %v, want the Legendary Action taken then the part used", eventKinds(events))
	}
	refusals := []struct {
		name, path string
		req        combat.StatBlockUseRequest
		want       combat.LegendaryActionRefusal
	}{
		{"own turn window", "unhappy (rule interaction)", sbLegendaryRequest(t, tail, luSelf), combat.LegendaryRefusedOwnTurn},
		{"incapacitated", "unhappy (rule interaction)", combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: tail,
			Window: luWindow(t, "hero", 1), Availability: combat.ConditionActionAvailabilityNone}, combat.LegendaryRefusedIncapacitated},
	}
	for _, c := range refusals {
		got := sbRefused(t, sbUse(t, m, c.req, state))
		if got != (combat.StatBlockLegendaryUseRefused{Reason: c.want}) {
			t.Errorf("%s: refusal = %#v, want StatBlockLegendaryUseRefused(%v)", c.name, got, c.want)
		}
	}
	taken := luTake(t, state, luWindow(t, "hero", 1))
	if got := sbRefused(t, sbUse(t, m, sbLegendaryRequest(t, tail, "hero"), taken)); got != (combat.StatBlockLegendaryUseRefused{Reason: combat.LegendaryRefusedWindowUsed}) {
		t.Errorf("second use in one window = %#v, want window used", got)
	}
	// The part's own limit is checked after the Legendary check, and a refusal spends no Legendary use.
	spent := luSpent(t, state, limited)
	if got := sbRefused(t, sbUse(t, m, sbLegendaryRequest(t, limited, "hero"), spent)); got != (combat.StatBlockPartUsageExhausted{}) {
		t.Errorf("spent legendary part = %#v, want StatBlockPartUsageExhausted", got)
	}
	if got := luLegendaryOf(t, spent).Remaining(); got != 3 {
		t.Errorf("the refused use spent a Legendary use: %d left, want 3", got)
	}
	none := luMonster(t, nil)
	if got := sbRefused(t, sbUse(t, none, sbLegendaryRequest(t, luRef(t, combat.MonsterPartLegendaryActions, 0), "hero"), luStart(t, none, combat.OutsideLair))); got != (combat.StatBlockPartNotFound{}) {
		t.Errorf("monster with no Legendary Actions = %#v, want StatBlockPartNotFound", got)
	}
}

func eventKinds(events []combat.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-009, table EC-1 "precedence" - when several refusals apply the order is
// part not found, wrong section, trigger not reported, Legendary refused, usage exhausted; the first is returned.
// Path: per case.
func TestUseStatBlockRefusalPrecedence(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	parry := sbRef(t, m, combat.MonsterPartReactions, "Parry")
	limited := sbRef(t, m, combat.MonsterPartLegendaryActions, "Limited Swipe")
	mind := sbAction(t, m, "Dominate Mind")
	state := luSpent(t, luSpent(t, luSpent(t, luStart(t, m, combat.OutsideLair), parry), limited), mind)
	state = luSpent(t, state, mind)
	notReported := func(part combat.LimitedUsePartRef) combat.StatBlockUseRequest {
		r := sbRequest(combat.StatBlockCostReaction, part)
		r.Trigger = combat.ReactionTriggerNotOccurred
		return r
	}
	own := sbLegendaryRequest(t, limited, luSelf)
	cases := []struct {
		name, path string
		req        combat.StatBlockUseRequest
		want       combat.StatBlockUseRefusal
	}{
		{"not found beats the trigger", "unhappy (rule interaction)", notReported(luRef(t, combat.MonsterPartReactions, 9)), combat.StatBlockPartNotFound{}},
		{"wrong section beats the trigger", "unhappy (rule interaction)", notReported(mind), combat.StatBlockPartInWrongSection{}},
		{"wrong section beats exhausted", "unhappy (rule interaction)", sbRequest(combat.StatBlockCostReaction, mind), combat.StatBlockPartInWrongSection{}},
		{"trigger beats exhausted", "unhappy (rule interaction)", notReported(parry), combat.StatBlockReactionTriggerNotReported{}},
		{"Legendary check beats exhausted", "unhappy (rule interaction)", own, combat.StatBlockLegendaryUseRefused{Reason: combat.LegendaryRefusedOwnTurn}},
		{"exhausted last", "unhappy (rule interaction)", sbRequest(combat.StatBlockCostReaction, parry), combat.StatBlockPartUsageExhausted{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := state
			if got := sbRefused(t, sbUse(t, m, c.req, state)); got != c.want {
				t.Errorf("refusal = %#v, want %#v", got, c.want)
			}
			if !state.Equal(before) {
				t.Error("a refusal changed the state")
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 and srderr.ErrInvalidInput - untrusted input is an error, never a
// refusal and never a panic.
// Path: per case.
func TestUseStatBlockRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	slam := sbAction(t, m, "Slam")
	other := luMust(content.NewMonster("other_beast", "Other Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), luFields(t)))
	valid := sbRequest(combat.StatBlockCostAction, slam)
	with := func(f func(*combat.StatBlockUseRequest)) combat.StatBlockUseRequest {
		r := valid
		f(&r)
		return r
	}
	cases := []struct {
		name  string
		m     content.Monster
		who   combat.CombatantID
		req   combat.StatBlockUseRequest
		state combat.LimitedUseState
	}{
		{"zero monster", content.Monster{}, sbSelf(), valid, state},
		{"zero combatant", m, combat.CombatantID(""), valid, state},
		{"zero limited-use state", m, sbSelf(), valid, combat.LimitedUseState{}},
		{"state of another combatant", m, combat.MustCombatantID("other"), valid, state},
		{"state of another monster", other, sbSelf(), valid, state},
		{"passive cost", m, sbSelf(), with(func(r *combat.StatBlockUseRequest) { r.Cost = combat.StatBlockCostPassive }), state},
		{"zero cost", m, sbSelf(), with(func(r *combat.StatBlockUseRequest) { r.Cost = 0 }), state},
		{"cost past the last", m, sbSelf(), with(func(r *combat.StatBlockUseRequest) { r.Cost = combat.StatBlockCostLegendaryAction + 1 }), state},
		{"zero part", m, sbSelf(), with(func(r *combat.StatBlockUseRequest) { r.Part = combat.LimitedUsePartRef{} }), state},
		{"invalid granted action", m, sbSelf(), with(func(r *combat.StatBlockUseRequest) { r.Granted = combat.GrantsDash{} }), state},
		{"reaction without a report", m, sbSelf(), combat.StatBlockUseRequest{Cost: combat.StatBlockCostReaction, Part: luRef(t, combat.MonsterPartReactions, 0)}, state},
		{"legendary without a window", m, sbSelf(), combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: luRef(t, combat.MonsterPartLegendaryActions, 0),
			Availability: combat.ConditionActionAvailabilityAvailable}, state},
		{"legendary without availability", m, sbSelf(), combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: luRef(t, combat.MonsterPartLegendaryActions, 0),
			Window: luWindow(t, "hero", 1)}, state},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := combat.UseStatBlock(c.m, c.who, c.req, c.state)
			if out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("UseStatBlock = %#v, %v, want a nil outcome and an invalid-input error", out, err)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - an operation leaves its inputs unchanged and returns copies.
// Path: unhappy (immutability).
func TestUseStatBlockLeavesInputsUnchangedAndReturnsCopies(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	before := luStart(t, m, combat.OutsideLair)
	req := sbRequest(combat.StatBlockCostAction, sbAction(t, m, "Dominate Mind"))
	tr := sbAllowed(t, sbUse(t, m, req, state))
	if !state.Equal(before) {
		t.Error("UseStatBlock changed the input state")
	}
	combattest.CheckEventsCopy(t, tr.Events)
	if events := tr.Events(); len(events) != 2 || events[0].Kind() != "limited_use_part_spent" || events[1].Kind() != "stat_block_part_used" {
		t.Errorf("events = %v, want the usage spent then the part used", eventKinds(events))
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-005, table EC-1 "feature id" - a part's FeatureID is
// "<monster id>__<part id>", the part id derived as a MonsterID is: lower-cased, apostrophes dropped, each other run of
// characters one underscore, the ends trimmed.
// Path: per case.
func TestStatBlockFeatureIDDerivation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path, part, want string
	}{
		{"plain name", `FR-005; "Aboleth", p. 258`, "happy", "Consume Memories", "fixture_beast__consume_memories"},
		{"apostrophe dropped", `FR-005`, "unhappy (rule interaction)", "Devil's Sight", "fixture_beast__devils_sight"},
		{"punctuation runs collapse", `FR-005`, "unhappy (rule interaction)", "Fire -- Breath (Recharge 5-6)", "fixture_beast__fire_breath_recharge_5_6"},
		{"leading digit is a valid part id", `FR-005`, "unhappy (boundary)", "2 Claws", "fixture_beast__2_claws"},
		{"typographic apostrophe dropped", `FR-005`, "unhappy (rule interaction)", "Dragon’s Roar", "fixture_beast__dragons_roar"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m := sbMonster(t, nil, sbPlain(c.part), nil)
			got, err := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 0))
			if err != nil || got != content.FeatureID(c.want) || !got.Valid() {
				t.Errorf("StatBlockFeatureID(%q) = %q, %v, want %q [%s]", c.part, got, err, c.want, c.requirement)
			}
			if p := combat.StatBlockFeatureIDProblems(m); len(p) != 0 {
				t.Errorf("StatBlockFeatureIDProblems = %v, want none", p)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-005, table EC-1 "id rejected" and "id duplicate" - a name with no letter or digit,
// an id over 64 characters and a second part deriving the same id are each a *StatBlockFeatureIDRejected matching
// srderr.ErrInvalidInput; both duplicates are listed by StatBlockFeatureIDProblems.
// Path: per case.
func TestStatBlockFeatureIDRejections(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 60)
	m := sbMonster(t, nil, sbPlain("' - '", long, "Spellcasting", "Spell casting", "Spell-casting", "Fine"), func(f *content.MonsterFields) {
		f.Traits = sbPlain("Spellcasting")
	})
	cases := []struct {
		name    string
		section combat.MonsterPartSection
		index   int
		want    combat.StatBlockFeatureIDReason
	}{
		{"name with no letter or digit", combat.MonsterPartActions, 0, combat.FeatureIDInvalidName},
		{"id over the limit", combat.MonsterPartActions, 1, combat.FeatureIDTooLong},
		{"name repeated across sections", combat.MonsterPartTraits, 0, combat.FeatureIDDuplicate},
		{"same name in one section", combat.MonsterPartActions, 2, combat.FeatureIDDuplicate},
		{"names differing only in punctuation", combat.MonsterPartActions, 3, combat.FeatureIDDuplicate},
		{"the other name differing only in punctuation", combat.MonsterPartActions, 4, combat.FeatureIDDuplicate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ref := luRef(t, c.section, c.index)
			id, err := combat.StatBlockFeatureID(m, ref)
			var rejected *combat.StatBlockFeatureIDRejected
			if !errors.As(err, &rejected) || !errors.Is(err, srderr.ErrInvalidInput) || id != "" {
				t.Fatalf("StatBlockFeatureID = %q, %v, want a *StatBlockFeatureIDRejected", id, err)
			}
			if rejected.Part() != ref || rejected.Reason() != c.want {
				t.Errorf("rejection = part %v reason %v, want part %v reason %v", rejected.Part(), rejected.Reason(), ref, c.want)
			}
		})
	}
	_, dup1 := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartTraits, 0))
	_, dup2 := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 2))
	_, bad := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 0))
	var dup2Rejected *combat.StatBlockFeatureIDRejected
	if !errors.As(dup2, &dup2Rejected) || !errors.Is(dup1, dup2Rejected) || errors.Is(bad, dup2Rejected) {
		t.Error("rejections match each other by reason: two duplicates match, a duplicate and an invalid name do not")
	}
	if _, err := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 5)); err != nil {
		t.Errorf("a part with a free id = %v, want an id", err)
	}
	problems := combat.StatBlockFeatureIDProblems(m)
	var got []combat.LimitedUsePartRef
	for _, p := range problems {
		got = append(got, p.Part())
	}
	want := []combat.LimitedUsePartRef{luRef(t, combat.MonsterPartTraits, 0), luRef(t, combat.MonsterPartActions, 0), luRef(t, combat.MonsterPartActions, 1),
		luRef(t, combat.MonsterPartActions, 2), luRef(t, combat.MonsterPartActions, 3), luRef(t, combat.MonsterPartActions, 4)}
	if !slices.Equal(got, want) {
		t.Errorf("StatBlockFeatureIDProblems parts = %v, want %v (printed order, both duplicates)", got, want)
	}
	if p := combat.StatBlockFeatureIDProblems(content.Monster{}); p != nil {
		t.Errorf("StatBlockFeatureIDProblems(zero monster) = %v, want nil", p)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-005 and FR-049 - StatBlockFeatureID rejects a zero monster and a part
// the monster does not have with an error matching srderr.ErrInvalidInput that is not a feature id rejection; UseStatBlock
// returns the rejection for a part whose id cannot be derived instead of a FeatureID.
// Path: unhappy (invalid input).
func TestStatBlockFeatureIDInvalidInputAndUseError(t *testing.T) {
	t.Parallel()
	m := sbMonster(t, nil, sbPlain("Slam", "Slam"), nil)
	for name, call := range map[string]func() error{
		"zero monster": func() error {
			_, err := combat.StatBlockFeatureID(content.Monster{}, luRef(t, combat.MonsterPartActions, 0))
			return err
		},
		"missing part": func() error {
			_, err := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 5))
			return err
		},
		"zero part": func() error { _, err := combat.StatBlockFeatureID(m, combat.LimitedUsePartRef{}); return err },
	} {
		var rejected *combat.StatBlockFeatureIDRejected
		if err := call(); !errors.Is(err, srderr.ErrInvalidInput) || errors.As(err, &rejected) {
			t.Errorf("%s: error = %v, want a plain invalid-input error", name, err)
		}
	}
	out, err := combat.UseStatBlock(m, sbSelf(), sbRequest(combat.StatBlockCostAction, luRef(t, combat.MonsterPartActions, 0)), luStart(t, m, combat.OutsideLair))
	var rejected *combat.StatBlockFeatureIDRejected
	if out != nil || !errors.As(err, &rejected) || rejected.Reason() != combat.FeatureIDDuplicate {
		t.Errorf("UseStatBlock for a duplicate id = %#v, %v, want the duplicate rejection", out, err)
	}
	// A Legendary Action is not recorded by FeatureID, so a defective id does not stop it.
	dup := luMonster(t, func(f *content.MonsterFields) {
		f.LegendaryActions = luMust(content.NewLegendaryActions(1, 0, sbPlain("Same", "Same")))
	})
	if _, err := combat.UseStatBlock(dup, sbSelf(), sbLegendaryRequest(t, luRef(t, combat.MonsterPartLegendaryActions, 0), "hero"), luStart(t, dup, combat.OutsideLair)); err != nil {
		t.Errorf("Legendary Action of a duplicate id = %v, want it used", err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-005 - the rejection reasons and the reaction report print their names.
// Path: happy.
func TestStatBlockUseEnumNames(t *testing.T) {
	t.Parallel()
	if got := []string{combat.FeatureIDInvalidName.String(), combat.FeatureIDTooLong.String(), combat.FeatureIDDuplicate.String(), combat.StatBlockFeatureIDReason(0).String()}; !slices.Equal(got, []string{"invalid_name", "too_long", "duplicate", "unknown"}) {
		t.Errorf("reason names = %v", got)
	}
	if got := []string{combat.ReactionTriggerOccurred.String(), combat.ReactionTriggerNotOccurred.String(), combat.ReactionTriggerReport(0).String()}; !slices.Equal(got, []string{"occurred", "not_occurred", "unknown"}) {
		t.Errorf("trigger names = %v", got)
	}
}

// FuzzStatBlockFeatureID checks that no part name makes StatBlockFeatureID panic, and that an id it returns is valid and
// begins with the monster id and the joint.
func FuzzStatBlockFeatureID(f *testing.F) {
	for _, s := range []string{"Slam", "", "' - '", "Devil's Sight", "2 Claws", strings.Repeat("x", 80), "ÄÖ", "a--b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		part, err := content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T."})
		if err != nil {
			return
		}
		m := luMonster(t, func(f *content.MonsterFields) { f.Actions = []content.StatBlockPart{part} })
		id, err := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 0))
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if !id.Valid() || !strings.HasPrefix(id.String(), "fixture_beast__") {
			t.Fatalf("id %q is not a valid id under the monster", id)
		}
	})
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-002, FR-003, FR-004, FR-008 - the transition reports the economy of the cost it was made at and no other: an action only
// the FeatureActionChoice, a Bonus Action only the BonusActionGrant, a Reaction only the FeatureReaction and a Legendary Action none; and the cost it carries.
// Path: per case.
func TestUseStatBlockTransitionReportsOnlyItsOwnEconomy(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	cases := []struct {
		name string
		req  combat.StatBlockUseRequest
		want [3]bool // action, Bonus Action, Reaction
	}{
		{"action", sbRequest(combat.StatBlockCostAction, sbAction(t, m, "Slam")), [3]bool{true, false, false}},
		{"Bonus Action", sbRequest(combat.StatBlockCostBonusAction, sbRef(t, m, combat.MonsterPartBonusActions, "Boar Rush")), [3]bool{false, true, false}},
		{"Reaction", sbRequest(combat.StatBlockCostReaction, sbRef(t, m, combat.MonsterPartReactions, "Unnerving Gaze")), [3]bool{false, false, true}},
		{"Legendary Action", sbLegendaryRequest(t, sbRef(t, m, combat.MonsterPartLegendaryActions, "Tail Swipe"), "hero"), [3]bool{false, false, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := sbAllowed(t, sbUse(t, m, c.req, state))
			_, a := tr.ActionChoice()
			_, b := tr.BonusActionGrant()
			_, r := tr.Reaction()
			if got := [3]bool{a, b, r}; got != c.want || tr.Cost() != c.req.Cost {
				t.Errorf("economy %v at cost %v, want %v at cost %v", got, tr.Cost(), c.want, c.req.Cost)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-004, FR-008 - the trigger report matters only to a Reaction and the window and availability only to a Legendary Action: an
// action, a Bonus Action or a Legendary Action ignores a trigger report, with none or "not occurred".
// Path: per case.
func TestUseStatBlockIgnoresFieldsOfOtherCosts(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	slam := sbAction(t, m, "Slam")
	for _, trigger := range []combat.ReactionTriggerReport{0, combat.ReactionTriggerOccurred, combat.ReactionTriggerNotOccurred, 9} {
		req := combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: slam, Trigger: trigger}
		sbAllowed(t, sbUse(t, m, req, state))
		req = combat.StatBlockUseRequest{Cost: combat.StatBlockCostBonusAction, Part: sbRef(t, m, combat.MonsterPartBonusActions, "Boar Rush"), Trigger: trigger}
		sbAllowed(t, sbUse(t, m, req, state))
		req = sbLegendaryRequest(t, sbRef(t, m, combat.MonsterPartLegendaryActions, "Tail Swipe"), "hero")
		req.Trigger = trigger
		sbAllowed(t, sbUse(t, m, req, state))
	}
	// A window and an availability given to an action are not read.
	req := sbLegendaryRequest(t, slam, "hero")
	req.Cost = combat.StatBlockCostAction
	req.Availability = combat.ConditionActionAvailabilityNone
	sbAllowed(t, sbUse(t, m, req, state))
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-009, FR-049 - a malformed request is an error even when its part is missing or in the wrong section: the error is not one of the
// refusals, so it comes before them.
// Path: per case.
func TestUseStatBlockErrorComesBeforeRefusals(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	missing := luRef(t, combat.MonsterPartReactions, 9)
	cases := []struct {
		name string
		req  combat.StatBlockUseRequest
	}{
		{"reaction without a report, part missing", combat.StatBlockUseRequest{Cost: combat.StatBlockCostReaction, Part: missing}},
		{"reaction with an undefined report, part missing", combat.StatBlockUseRequest{Cost: combat.StatBlockCostReaction, Part: missing, Trigger: 3}},
		{"legendary without a window, part missing", combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: missing, Availability: combat.ConditionActionAvailabilityAvailable}},
		{"legendary without availability, part missing", combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: missing, Window: luWindow(t, "hero", 1)}},
		{"legendary with an undefined availability, part missing", combat.StatBlockUseRequest{Cost: combat.StatBlockCostLegendaryAction, Part: missing, Window: luWindow(t, "hero", 1), Availability: 7}},
		{"passive cost, part missing", combat.StatBlockUseRequest{Cost: combat.StatBlockCostPassive, Part: missing}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if out, err := combat.UseStatBlock(m, sbSelf(), c.req, state); out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("UseStatBlock = %#v, %v, want an invalid-input error", out, err)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - a limited-use state of another combatant or of another monster is an error whichever way the ids compare.
// Path: per case.
func TestUseStatBlockRejectsAStateOfAnotherCreatureWhicheverWay(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	state := luStart(t, m, combat.OutsideLair)
	earlier := luMust(content.NewMonster("aaa_beast", "Aaa Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), luFields(t)))
	req := sbRequest(combat.StatBlockCostAction, luRef(t, combat.MonsterPartActions, 0))
	for name, who := range map[string]combat.CombatantID{"a combatant before": "aaa", "a combatant after": "zzz"} {
		if _, err := combat.UseStatBlock(m, who, req, state); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := combat.UseStatBlock(earlier, sbSelf(), req, state); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a monster before: error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-005 - the part id keeps every letter and digit at the edges of their ranges (A, Z, a, z, 0, 9) and turns the characters just
// outside them into separators, drops no leading separator, and an id of exactly 64 characters is valid while 65 is too long.
// Path: per case.
func TestStatBlockFeatureIDCharacterRanges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path, part, want string
	}{
		{"a character above the typographic apostrophe is a separator", `FR-005`, "unhappy (boundary)", "a…b", "fixture_beast__a_b"},
		{"every edge letter and digit", `FR-005`, "unhappy (boundary)", "AZaz09", "fixture_beast__azaz09"},
		{"characters just outside the ranges", `FR-005`, "unhappy (boundary)", "a@b[c`d{e/f:g", "fixture_beast__a_b_c_d_e_f_g"},
		{"leading and trailing separators", `FR-005`, "unhappy (boundary)", "--Slam--", "fixture_beast__slam"},
		{"apostrophes inside a word", `FR-005`, "unhappy (rule interaction)", "D'e’v", "fixture_beast__dev"},
		{"an id of exactly 64 characters", `FR-005`, "unhappy (boundary)", strings.Repeat("a", 49), "fixture_beast__" + strings.Repeat("a", 49)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m := sbMonster(t, nil, sbPlain(c.part), nil)
			got, err := combat.StatBlockFeatureID(m, luRef(t, combat.MonsterPartActions, 0))
			if err != nil || got != content.FeatureID(c.want) {
				t.Errorf("StatBlockFeatureID(%q) = %q, %v, want %q [%s]", c.part, got, err, c.want, c.requirement)
			}
		})
	}
	over := sbMonster(t, nil, sbPlain(strings.Repeat("a", 50)), nil)
	if _, err := combat.StatBlockFeatureID(over, luRef(t, combat.MonsterPartActions, 0)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an id of 65 characters = %v, want a rejection", err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-005 - a Legendary Action part has a FeatureID like any part, a name it shares with an Action is a duplicate for both, and
// StatBlockFeatureIDProblems names the reason of each.
// Path: per case.
func TestStatBlockFeatureIDLegendaryAndProblemReasons(t *testing.T) {
	t.Parallel()
	m := sbBeast(t)
	tail := sbRef(t, m, combat.MonsterPartLegendaryActions, "Tail Swipe")
	if id, err := combat.StatBlockFeatureID(m, tail); err != nil || id != content.MustFeatureID("fixture_beast__tail_swipe") {
		t.Errorf("StatBlockFeatureID(legendary) = %q, %v", id, err)
	}
	dup := luMonster(t, func(f *content.MonsterFields) {
		f.Actions = sbPlain("Tail Swipe", "' - '")
		f.LegendaryActions = luMust(content.NewLegendaryActions(1, 0, sbPlain("Tail Swipe")))
	})
	problems := combat.StatBlockFeatureIDProblems(dup)
	got := map[combat.LimitedUsePartRef]combat.StatBlockFeatureIDReason{}
	for _, p := range problems {
		got[p.Part()] = p.Reason()
	}
	want := map[combat.LimitedUsePartRef]combat.StatBlockFeatureIDReason{
		luRef(t, combat.MonsterPartActions, 0): combat.FeatureIDDuplicate, luRef(t, combat.MonsterPartActions, 1): combat.FeatureIDInvalidName,
		luRef(t, combat.MonsterPartLegendaryActions, 0): combat.FeatureIDDuplicate}
	if len(problems) != 3 || got[luRef(t, combat.MonsterPartActions, 0)] != want[luRef(t, combat.MonsterPartActions, 0)] || got[luRef(t, combat.MonsterPartActions, 1)] != combat.FeatureIDInvalidName ||
		got[luRef(t, combat.MonsterPartLegendaryActions, 0)] != combat.FeatureIDDuplicate {
		t.Errorf("problems = %v, want %v", got, want)
	}
	// A duplicate does not match an invalid name, whichever side is the target.
	_, e1 := combat.StatBlockFeatureID(dup, luRef(t, combat.MonsterPartActions, 0))
	_, e2 := combat.StatBlockFeatureID(dup, luRef(t, combat.MonsterPartActions, 1))
	var r1, r2 *combat.StatBlockFeatureIDRejected
	if !errors.As(e1, &r1) || !errors.As(e2, &r2) || errors.Is(e1, r2) || errors.Is(e2, r1) || !errors.Is(e1, r1) {
		t.Error("rejections of different reasons match each other, or a rejection does not match itself")
	}
}
