package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-050 - Available gives escape for all five kinds, burst for Chain, Manacles and Rope only, and pick the lock and the key for Manacles only; an invalid kind has none (SRD 5.2.1, pp. 96-99).
// Path: per case.
func TestHoldRulesAvailable(t *testing.T) {
	t.Parallel()
	esc, burst, pick, key := combat.HoldReleaseMethodEscape, combat.HoldReleaseMethodBurst, combat.HoldReleaseMethodPickLock, combat.HoldReleaseMethodKey
	want := map[combat.HoldKind][]combat.HoldReleaseMethod{combat.HoldKindNet: {esc}, combat.HoldKindChain: {esc, burst}, combat.HoldKindManacles: {esc, burst, pick, key},
		combat.HoldKindRope: {esc, burst}, combat.HoldKindHuntingTrap: {esc}, 0: {}, 9: {}}
	for kind, methods := range want {
		got := combat.HoldRules{}.Available(kind)
		if !slices.Equal(got, methods) || got == nil {
			t.Errorf("Available(%v) = %v, want %v", kind, got, methods)
		}
	}
}

func relReq(tb testing.TB, kind combat.HoldKind, method combat.HoldReleaseMethod) combat.ReleaseRequest {
	tb.Helper()
	in := holdIn("h", kind)
	return combat.ReleaseRequest{Hold: gearHold(tb, in), Method: method, Actor: combat.ActorIsHeld{}, Actions: combat.ConditionActionAvailabilityAvailable, HasThievesTools: true}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-050, FR-051, FR-052, FR-053, FR-055, FR-058 - Plan gives, per kind and method, the check of table 4.9b (skill, Difficulty Class, cause on success) or a refusal in the order methodNotAvailable, helperNotAllowed, helperOutOfReach, noAction, toolsMissing: a helper may escape a Net or Hunting Trap within reach and nothing else of Chain, Manacles or Rope, may pick a lock or use a key, and an actor that cannot act is refused even with a key (SRD 5.2.1, pp. 96-99).
// Path: per case.
func TestHoldRulesPlan(t *testing.T) {
	t.Parallel()
	esc, burst, pick, key := combat.HoldReleaseMethodEscape, combat.HoldReleaseMethodBurst, combat.HoldReleaseMethodPickLock, combat.HoldReleaseMethodKey
	net, chain, cuffs, rope, trap := combat.HoldKindNet, combat.HoldKindChain, combat.HoldKindManacles, combat.HoldKindRope, combat.HoldKindHuntingTrap
	helper := func(in bool) combat.ReleaseActor { return combat.ActorIsHelper{WithinReach: in} }
	none, tools := combat.ConditionActionAvailabilityNone, false
	cases := []struct {
		name, path string
		kind       combat.HoldKind
		method     combat.HoldReleaseMethod
		change     func(*combat.ReleaseRequest)
		want       combat.GearRejection
		skill      content.Skill
		dc         d20.DifficultyClass
		cause      combat.HoldEndCause
	}{
		{"escape a net", "happy", net, esc, nil, 0, content.SkillAthletics, 10, combat.HoldEndCauseEscaped},
		{"escape a chain", "happy", chain, esc, nil, 0, content.SkillAcrobatics, 18, combat.HoldEndCauseEscaped},
		{"escape manacles", "happy", cuffs, esc, nil, 0, content.SkillSleightOfHand, 20, combat.HoldEndCauseEscaped},
		{"escape a rope", "happy", rope, esc, nil, 0, content.SkillAcrobatics, 15, combat.HoldEndCauseEscaped},
		{"escape a hunting trap", "happy", trap, esc, nil, 0, content.SkillAthletics, 13, combat.HoldEndCauseEscaped},
		{"burst a chain", "happy", chain, burst, nil, 0, content.SkillAthletics, 20, combat.HoldEndCauseBurst},
		{"burst manacles", "happy", cuffs, burst, nil, 0, content.SkillAthletics, 25, combat.HoldEndCauseBurst},
		{"burst a rope", "happy", rope, burst, nil, 0, content.SkillAthletics, 20, combat.HoldEndCauseBurst},
		{"pick the manacles' lock", "happy", cuffs, pick, nil, 0, content.SkillSleightOfHand, 15, combat.HoldEndCauseUnlocked},
		{"burst a net", "unhappy (invalid input)", net, burst, nil, combat.GearRejectionMethodNotAvailable, 0, 0, 0},
		{"burst a hunting trap", "unhappy (invalid input)", trap, burst, nil, combat.GearRejectionMethodNotAvailable, 0, 0, 0},
		{"pick a chain's lock", "unhappy (invalid input)", chain, pick, nil, combat.GearRejectionMethodNotAvailable, 0, 0, 0},
		{"key for a rope", "unhappy (invalid input)", rope, key, nil, combat.GearRejectionMethodNotAvailable, 0, 0, 0},
		{"helper escapes a net within reach", "happy", net, esc, func(r *combat.ReleaseRequest) { r.Actor = helper(true) }, 0, content.SkillAthletics, 10, combat.HoldEndCauseEscaped},
		{"helper beyond reach of a net", "unhappy (boundary)", net, esc, func(r *combat.ReleaseRequest) { r.Actor = helper(false) }, combat.GearRejectionHelperOutOfReach, 0, 0, 0},
		{"helper frees a hunting trap within reach", "happy", trap, esc, func(r *combat.ReleaseRequest) { r.Actor = helper(true) }, 0, content.SkillAthletics, 13, combat.HoldEndCauseEscaped},
		{"helper beyond reach of a hunting trap", "unhappy (boundary)", trap, esc, func(r *combat.ReleaseRequest) { r.Actor = helper(false) }, combat.GearRejectionHelperOutOfReach, 0, 0, 0},
		{"helper escapes a chain", "unhappy (rule interaction)", chain, esc, func(r *combat.ReleaseRequest) { r.Actor = helper(true) }, combat.GearRejectionHelperNotAllowed, 0, 0, 0},
		{"helper bursts manacles", "unhappy (rule interaction)", cuffs, burst, func(r *combat.ReleaseRequest) { r.Actor = helper(true) }, combat.GearRejectionHelperNotAllowed, 0, 0, 0},
		{"helper bursts a rope", "unhappy (rule interaction)", rope, burst, func(r *combat.ReleaseRequest) { r.Actor = helper(false) }, combat.GearRejectionHelperNotAllowed, 0, 0, 0},
		{"helper picks the lock", "unhappy (rule interaction)", cuffs, pick, func(r *combat.ReleaseRequest) { r.Actor = helper(false) }, 0, content.SkillSleightOfHand, 15, combat.HoldEndCauseUnlocked},
		{"helper uses the key", "unhappy (rule interaction)", cuffs, key, func(r *combat.ReleaseRequest) { r.Actor = helper(false) }, 0, 0, 0, combat.HoldEndCauseUnlocked},
		{"the key takes no check", "happy", cuffs, key, nil, 0, 0, 0, combat.HoldEndCauseUnlocked},
		{"no action to escape", "unhappy (state error)", net, esc, func(r *combat.ReleaseRequest) { r.Actions = none }, combat.GearRejectionNoAction, 0, 0, 0},
		{"no action to burst", "unhappy (state error)", chain, burst, func(r *combat.ReleaseRequest) { r.Actions = none }, combat.GearRejectionNoAction, 0, 0, 0},
		{"no action to pick the lock", "unhappy (state error)", cuffs, pick, func(r *combat.ReleaseRequest) { r.Actions = none }, combat.GearRejectionNoAction, 0, 0, 0},
		{"no action to use the key", "unhappy (state error)", cuffs, key, func(r *combat.ReleaseRequest) { r.Actions = none }, combat.GearRejectionNoAction, 0, 0, 0},
		{"no Thieves' Tools", "unhappy (state error)", cuffs, pick, func(r *combat.ReleaseRequest) { r.HasThievesTools = tools }, combat.GearRejectionToolsMissing, 0, 0, 0},
		{"the key needs no tools", "unhappy (rule interaction)", cuffs, key, func(r *combat.ReleaseRequest) { r.HasThievesTools = tools }, 0, 0, 0, combat.HoldEndCauseUnlocked},
		{"no action wins over no tools", "unhappy (rule interaction)", cuffs, pick, func(r *combat.ReleaseRequest) { r.Actions, r.HasThievesTools = none, tools }, combat.GearRejectionNoAction, 0, 0, 0},
		{"helper rule wins over no action", "unhappy (rule interaction)", chain, esc, func(r *combat.ReleaseRequest) { r.Actor, r.Actions = helper(true), none }, combat.GearRejectionHelperNotAllowed, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := relReq(t, c.kind, c.method)
			if c.change != nil {
				c.change(&r)
			}
			plan, got, err := combat.HoldRules{}.Plan(r)
			if err != nil || got != c.want {
				t.Fatalf("Plan = %v, %v; want %v [%s FR-052]", got, err, c.want, gearSpec)
			}
			check, needs := plan.Check()
			if c.want != 0 {
				if plan != (combat.HoldReleasePlan{}) {
					t.Errorf("a refusal must have the zero plan, got %+v", plan)
				}
				return
			}
			if needs != (c.dc != 0) || check.Skill() != c.skill || check.DifficultyClass() != c.dc || plan.Cause() != c.cause {
				t.Errorf("plan = %+v (check %v), want %v DC %d cause %v [%s FR-051]", plan, needs, c.skill, c.dc, c.cause, gearSpec)
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-055, FR-056 - the lock-picking check is Dexterity (Sleight of Hand) DC 15; only a failed attempt to escape a Hunting Trap costs 1 Piercing damage (SRD 5.2.1, "Manacles", p. 98; "Hunting Trap", pp. 97-98).
// Path: happy; unhappy (rule interaction).
func TestHoldRulesPickLockAndFailureDamage(t *testing.T) {
	t.Parallel()
	if c := (combat.HoldRules{}).PickLockCheck(); c.Skill() != content.SkillSleightOfHand || c.DifficultyClass() != 15 {
		t.Errorf("PickLockCheck = %+v, want Sleight of Hand DC 15", c)
	}
	for k := combat.HoldKindNet; k <= combat.HoldKindHuntingTrap; k++ {
		d, ok := combat.HoldRules{}.FailureDamage(k)
		if want := k == combat.HoldKindHuntingTrap; ok != want || (ok && d != combat.MustDamage(1, content.DamagePiercing)) {
			t.Errorf("FailureDamage(%v) = %v, %v", k, d, ok)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-052, FR-071 - Plan returns a GearError (invalid_value) for a zero hold, a method that is not valid, an actor that is nil or another type, and actions that are not valid, never a refusal.
// Path: unhappy (invalid input).
func TestHoldRulesPlanInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*combat.ReleaseRequest){
		"zero hold":       func(r *combat.ReleaseRequest) { r.Hold = combat.GearHold{} },
		"invalid method":  func(r *combat.ReleaseRequest) { r.Method = 0 },
		"nil actor":       func(r *combat.ReleaseRequest) { r.Actor = nil },
		"invalid actions": func(r *combat.ReleaseRequest) { r.Actions = 0 },
	}
	for name, change := range cases {
		r := relReq(t, combat.HoldKindNet, combat.HoldReleaseMethodEscape)
		change(&r)
		if plan, refusal, err := (combat.HoldRules{}).Plan(r); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) || plan != (combat.HoldReleasePlan{}) || refusal != 0 {
			t.Errorf("%s: Plan = %+v, %v, %v; want an invalid_value GearError", name, plan, refusal, err)
		}
	}
}
