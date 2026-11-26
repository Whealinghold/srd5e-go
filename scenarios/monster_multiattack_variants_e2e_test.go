package scenarios_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// The refs of the monsters of these scenarios: the Multiattack is Action 0 and the attacks follow.
var (
	grpFClaws    = grpCRef(combat.MonsterPartActions, 1)
	grpFTailRef  = grpCRef(combat.MonsterPartActions, 2)
	grpFFlame    = grpCRef(combat.MonsterPartActions, 3)
	grpFSlam     = grpCRef(combat.MonsterPartActions, 1)
	grpFBite     = grpCRef(combat.MonsterPartActions, 2)
	grpFScreech  = grpCRef(combat.MonsterPartActions, 3)
	grpFSprayRef = grpCRef(combat.MonsterPartActions, 4)
)

func grpFGroup(n int, parts ...string) content.AttackSlotGroup {
	return grpCMust(content.NewAttackSlotGroup(n, parts))
}

func grpFOption(part string) content.ReplacementOption {
	return grpCMust(content.NewReplacementOption(part, "", 0))
}

// grpFChimera is a monster with two alternative Multiattack variants, as printed for several creatures: "one Claws attack and one Tail attack, or two Hurl Flame attacks".
func grpFChimera() content.Monster {
	plan := grpCMust(content.NewMultiattackPlan([]content.MultiattackVariant{
		grpCMust(content.NewMultiattackVariant("", []content.AttackSlotGroup{grpFGroup(1, "Claws"), grpFGroup(1, "Tail")}, nil, nil)),
		grpCMust(content.NewMultiattackVariant("", []content.AttackSlotGroup{grpFGroup(2, "Hurl Flame")}, nil, nil)),
	}))
	return grpCMonster(func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{
			grpCPart(content.MultiattackName, content.StatBlockPartFields{Plan: plan}),
			grpCUsage("Claws", content.PartUnlimited{}), grpCUsage("Tail", content.PartUnlimited{}), grpCUsage("Hurl Flame", content.PartUnlimited{}),
		}
	})
}

// grpFSlammer has one variant of three Slam attacks and a Bite, with two replacement rules: Screech may replace one attack of any kind any number of times, and Spray (a Recharge 5-6 action) may replace two Slam attacks once.
func grpFSlammer() content.Monster {
	anyAttack := grpCMust(content.NewReplacementRule(1, content.ReplacementScopeAnyAttack{}, []content.ReplacementOption{grpFOption("Screech")}, content.ReplaceAnyNumber))
	onlySlam := grpCMust(content.NewReplacementRule(2, grpCMust(content.NewReplacementScopeOneAttackPart("Slam")), []content.ReplacementOption{grpFOption("Spray")}, content.ReplaceOnce))
	plan := grpCMust(content.NewMultiattackPlan([]content.MultiattackVariant{
		grpCMust(content.NewMultiattackVariant("", []content.AttackSlotGroup{grpFGroup(3, "Slam"), grpFGroup(1, "Bite")}, nil, []content.ReplacementRule{anyAttack, onlySlam})),
	}))
	return grpCMonster(func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{
			grpCPart(content.MultiattackName, content.StatBlockPartFields{Plan: plan}),
			grpCUsage("Slam", content.PartUnlimited{}), grpCUsage("Bite", content.PartUnlimited{}), grpCUsage("Screech", content.PartUnlimited{}),
			grpCUsage("Spray", grpCRecharge(5, 6)),
		}
	})
}

// grpFTurn follows one turn of one monster's Multiattack.
type grpFTurn struct {
	t        *testing.T
	progress combat.MultiattackProgress
	uses     combat.LimitedUseState
}

func grpFBegin(t *testing.T, m content.Monster) grpFTurn {
	t.Helper()
	id := grpCID("beast")
	uses := grpCMust(combat.StartLimitedUse(m, id, combat.OutsideLair))
	out := grpCMust(combat.StartMultiattack(m, id, uses))
	started, ok := out.(combat.MultiattackStarted)
	if !ok {
		t.Fatalf("StartMultiattack = %#v, want started", out)
	}
	return grpFTurn{t, started.Progress(), uses}
}

// recorded applies a step that must be recorded and returns the turn after it.
func (g grpFTurn) recorded(o combat.MultiattackStepOutcome, err error) grpFTurn {
	g.t.Helper()
	if err != nil {
		g.t.Fatal(err)
	}
	rec, ok := o.(combat.MultiattackStepRecorded)
	if !ok {
		g.t.Fatalf("step = %#v, want recorded", o)
	}
	g.progress, g.uses = rec.Transition.Progress(), rec.Transition.LimitedUse()
	return g
}

// refused returns the allowance refusal of a step that must be stopped by the plan.
func (g grpFTurn) refused(o combat.MultiattackStepOutcome, err error) combat.MultiattackAllowanceRefusal {
	g.t.Helper()
	if err != nil {
		g.t.Fatal(err)
	}
	n, ok := o.(combat.MultiattackNotInAllowance)
	if !ok {
		g.t.Fatalf("step = %#v, want not in allowance", o)
	}
	return n.Reason
}

func (g grpFTurn) attack(part combat.LimitedUsePartRef) (combat.MultiattackStepOutcome, error) {
	return g.progress.RecordAttack(part, g.uses)
}

func (g grpFTurn) replace(option string, replaced combat.LimitedUsePartRef) (combat.MultiattackStepOutcome, error) {
	return g.progress.RecordReplacement(grpFOption(option), replaced, g.uses)
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-015, FR-013, FR-016 - the variants of a Multiattack are alternatives and the candidates narrow with each step: a Claws attack closes the Hurl Flame variant, so mixing the two is refused as attack_not_allowed, and once one variant is left the other never comes back, even after the Claws and Tail attacks are all made; the same in the other direction, where one Hurl Flame closes the first variant and the second Hurl Flame still fits.
// Path: unhappy (rule interaction).
func TestGrpFMultiattackVariantsNarrowWithEachStep(t *testing.T) {
	t.Parallel()
	m := grpFChimera()

	// Claws first: the first variant is the only one left.
	turn := grpFBegin(t, m)
	if n := turn.progress.AttacksRemaining(grpFFlame); n == 0 {
		t.Fatalf("Hurl Flame attacks at the start = 0, want the second variant open")
	}
	turn = turn.recorded(turn.attack(grpFClaws))
	if got := turn.refused(turn.attack(grpFFlame)); got != combat.MultiattackAttackNotAllowed {
		t.Errorf("Hurl Flame after Claws = %v, want attack_not_allowed", got)
	}
	if n := turn.progress.AttacksRemaining(grpFFlame); n != 0 {
		t.Errorf("Hurl Flame attacks left after Claws = %d, want 0", n)
	}
	if got := turn.refused(turn.attack(grpFClaws)); got != combat.MultiattackAllowanceSpent {
		t.Errorf("a second Claws = %v, want allowance_spent", got)
	}
	turn = turn.recorded(turn.attack(grpFTailRef))
	if got := turn.refused(turn.attack(grpFFlame)); got != combat.MultiattackAttackNotAllowed {
		t.Errorf("Hurl Flame once Claws and Tail are made = %v, want attack_not_allowed (the closed variant never comes back)", got)
	}
	if !slices.Equal(grpFAttackKinds(turn.progress), []combat.LimitedUsePartRef{grpFClaws, grpFTailRef}) {
		t.Errorf("attacks recorded = %v, want Claws then Tail", grpFAttackKinds(turn.progress))
	}

	// Hurl Flame first: the second variant is the only one left, for two attacks.
	other := grpFBegin(t, m)
	other = other.recorded(other.attack(grpFFlame))
	if got := other.refused(other.attack(grpFClaws)); got != combat.MultiattackAttackNotAllowed {
		t.Errorf("Claws after Hurl Flame = %v, want attack_not_allowed", got)
	}
	if got := other.refused(other.attack(grpFTailRef)); got != combat.MultiattackAttackNotAllowed {
		t.Errorf("Tail after Hurl Flame = %v, want attack_not_allowed", got)
	}
	other = other.recorded(other.attack(grpFFlame))
	if got := other.refused(other.attack(grpFFlame)); got != combat.MultiattackAllowanceSpent {
		t.Errorf("a third Hurl Flame = %v, want allowance_spent", got)
	}
	// A part that no variant lists is never in the allowance, at the start or after narrowing.
	start := grpFBegin(t, m)
	if got := start.refused(start.attack(grpCRef(combat.MonsterPartActions, 0))); got != combat.MultiattackAttackNotAllowed {
		t.Errorf("the Multiattack as its own attack = %v, want attack_not_allowed", got)
	}
}

func grpFAttackKinds(p combat.MultiattackProgress) []combat.LimitedUsePartRef { return p.Attacks() }

// Requirement: doc/specs/monster-stat-block-actions.md FR-018 - a replacement succeeds when a rule lists the option for the attack and its scope allows it: Screech (any attack, any number of times) replaces one Slam at a time without ever being limited until the group has no attack left to give, Spray (Slam only, two attacks, once) replaces two Slam attacks and is then refused as replacement_limit_reached, an option or a replaced attack that no rule lists is replacement_not_listed, a group with too few attacks left is not_enough_attacks_to_replace, and a replacement takes the slots it names and counts as no attack made.
// Path: per case.
func TestGrpFMultiattackReplacementsOnceAndAnyNumber(t *testing.T) {
	t.Parallel()
	m := grpFSlammer()

	// Any number: three Screech replacements use up the three Slam slots one at a time; a fourth has nothing left to replace.
	turn := grpFBegin(t, m)
	for i := range 3 {
		turn = turn.recorded(turn.replace("Screech", grpFSlam))
		if got := turn.progress.AttacksRemaining(grpFSlam); got != 2-i {
			t.Fatalf("Slam attacks left after %d Screech replacements = %d, want %d", i+1, got, 2-i)
		}
	}
	if len(turn.progress.Attacks()) != 0 {
		t.Errorf("replacements counted as attacks made: %v", turn.progress.Attacks())
	}
	if got := turn.refused(turn.replace("Screech", grpFSlam)); got != combat.MultiattackNotEnoughAttacksToReplace {
		t.Errorf("a fourth Screech over Slam = %v, want not_enough_attacks_to_replace", got)
	}
	// Screech may also replace the Bite, the other group, and is again never limited.
	turn = turn.recorded(turn.replace("Screech", grpFBite))
	if got := turn.refused(turn.replace("Screech", grpFBite)); got != combat.MultiattackNotEnoughAttacksToReplace {
		t.Errorf("a second Screech over the one Bite = %v, want not_enough_attacks_to_replace", got)
	}

	// Once, two attacks: Spray replaces two Slam attacks at once; the same rule a second time is the limit.
	spray := grpFBegin(t, m)
	spray = spray.recorded(spray.replace("Spray", grpFSlam))
	if got := spray.progress.AttacksRemaining(grpFSlam); got != 1 {
		t.Errorf("Slam attacks left after Spray = %d, want 1 (it took two of three)", got)
	}
	if spray.uses.AvailabilityOf(grpFSprayRef) != combat.UseExhausted {
		t.Errorf("Spray after the replacement = %v, want spent", spray.uses.AvailabilityOf(grpFSprayRef))
	}
	if got := spray.refused(spray.replace("Spray", grpFSlam)); got != combat.MultiattackReplacementLimitReached {
		t.Errorf("a second Spray = %v, want replacement_limit_reached", got)
	}

	// Not listed: Spray cannot replace the Bite (its scope is Slam only), Bite is no listed option, and an option nobody lists.
	fresh := grpFBegin(t, m)
	if got := fresh.refused(fresh.replace("Spray", grpFBite)); got != combat.MultiattackReplacementNotListed {
		t.Errorf("Spray over the Bite = %v, want replacement_not_listed", got)
	}
	if got := fresh.refused(fresh.replace("Bite", grpFSlam)); got != combat.MultiattackReplacementNotListed {
		t.Errorf("Bite as an option = %v, want replacement_not_listed", got)
	}
	if got := fresh.refused(fresh.replace("Screech", grpCRef(combat.MonsterPartActions, 0))); got != combat.MultiattackReplacementNotListed {
		t.Errorf("Screech over a part that is no attack = %v, want replacement_not_listed", got)
	}

	// Too few attacks: with two of the three Slam attacks made, Spray (two attacks) does not fit; Screech (one) still does.
	spent := grpFBegin(t, m)
	spent = spent.recorded(spent.attack(grpFSlam))
	spent = spent.recorded(spent.attack(grpFSlam))
	if got := spent.refused(spent.replace("Spray", grpFSlam)); got != combat.MultiattackNotEnoughAttacksToReplace {
		t.Errorf("Spray with one Slam left = %v, want not_enough_attacks_to_replace", got)
	}
	spent = spent.recorded(spent.replace("Screech", grpFSlam))
	if len(spent.progress.Attacks()) != 2 {
		t.Errorf("attacks made = %v, want the two Slam attacks and no replacement", spent.progress.Attacks())
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-020, FR-018, FR-007 - the allowance is checked before the usage of the part: a replacement refused by its rule (the once limit) is reported as not in the allowance even though the option's part is spent too; a replacement the plan allows with the part spent is stopped by the part gate with the usage refusal; and a refused step leaves the progress and the limited-use state exactly as the caller held them.
// Path: unhappy (rule interaction).
func TestGrpFMultiattackAllowanceIsCheckedBeforeUsage(t *testing.T) {
	t.Parallel()
	m := grpFSlammer()

	// Spray is spent before the turn: the plan allows the replacement, the part gate stops it.
	turn := grpFBegin(t, m)
	turn.uses = grpCSpent(t, turn.uses, grpFSprayRef)
	before, usesBefore := turn.progress, turn.uses
	out, err := turn.replace("Spray", grpFSlam)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(combat.MultiattackStepUsageRefused); !ok {
		t.Fatalf("Spray with the part spent = %#v, want the usage refusal", out)
	}
	if !turn.progress.Equal(before) || turn.uses.AvailabilityOf(grpFSprayRef) != usesBefore.AvailabilityOf(grpFSprayRef) {
		t.Errorf("a usage refusal changed the progress or the limited-use state")
	}

	// Spray was used this turn (the once limit is reached) and the part is spent: the allowance refusal wins.
	used := grpFBegin(t, m)
	used = used.recorded(used.replace("Spray", grpFSlam))
	if got := used.refused(used.replace("Spray", grpFSlam)); got != combat.MultiattackReplacementLimitReached {
		t.Errorf("a second Spray with the part spent = %v, want replacement_limit_reached before any usage refusal", got)
	}
	again, usesNow := used.progress, used.uses
	used.refused(used.replace("Spray", grpFSlam))
	used.refused(used.attack(grpFScreech)) // Screech is no attack of the plan
	if !used.progress.Equal(again) || used.uses.AvailabilityOf(grpFSprayRef) != usesNow.AvailabilityOf(grpFSprayRef) {
		t.Errorf("an allowance refusal changed the progress or the limited-use state")
	}
}
