package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var rules combat.GrappleRules

// Requirement: per case
// Path: per case.
func TestGrappleDC(t *testing.T) {
	t.Parallel()
	const p190 = `"Unarmed Strike", Grapple, p. 190`
	cases := []struct {
		name, requirement, path string
		modifier, proficiency   int
		want                    d20.DifficultyClass
	}{
		{"ordinary", "FR-094, table 4.10.6, " + p190, "happy", 3, 2, 13},
		{"negative modifier", "FR-094, table 4.10.6", "unhappy (boundary)", -1, 2, 9},
		{"no cap or floor", "FR-094", "unhappy (boundary)", -50, 0, -42},
		{"saturates at the top", "plan-phase-4.md section 7 (overflow)", "unhappy (boundary)", math.MaxInt, 2, d20.DifficultyClass(math.MaxInt)},
		{"saturates at the bottom", "plan-phase-4.md section 7 (overflow)", "unhappy (boundary)", math.MinInt, math.MinInt, d20.DifficultyClass(math.MinInt)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := rules.GrappleDC(c.modifier, c.proficiency); got != c.want {
				t.Errorf("GrappleDC(%d, %d) = %d, want %d [%s]", c.modifier, c.proficiency, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-095, FR-035 ("Unarmed Strike", p. 190; "Grappling", p. 182) - the target chooses a
// Strength or Dexterity save, and the escape options are Athletics then Acrobatics, in order, as fresh slices.
// Path: happy.
func TestGrappleRulesLists(t *testing.T) {
	t.Parallel()
	if got := rules.OptionSaveAbilities(); !slices.Equal(got, []ability.Ability{ability.Strength, ability.Dexterity}) {
		t.Errorf("OptionSaveAbilities = %v", got)
	}
	if got := rules.EscapeOptions(); !slices.Equal(got, []combat.GrappleEscapeOption{combat.GrappleEscapeOptionStrengthAthletics, combat.GrappleEscapeOptionDexterityAcrobatics}) {
		t.Errorf("EscapeOptions = %v", got)
	}
	rules.EscapeOptions()[0] = 0
	if rules.EscapeOptions()[0] != combat.GrappleEscapeOptionStrengthAthletics {
		t.Error("EscapeOptions is not a fresh slice")
	}
}

// Requirement: per case
// Path: per case.
func TestGrapplePossibility(t *testing.T) {
	t.Parallel()
	const p190 = `"Unarmed Strike", Grapple, p. 190; "Creature Size", p. 14`
	free, none := combat.HandAvailabilityHandFree, combat.HandAvailabilityNoHandFree
	tooLarge := combat.GrappleNotPossible{Reason: combat.GrappleNotPossibleReasonTargetTooLarge}
	noHand := combat.GrappleNotPossible{Reason: combat.GrappleNotPossibleReasonNoFreeHand}
	cases := []struct {
		name, requirement, path string
		attacker, target        content.CreatureSize
		hand                    combat.HandAvailability
		want                    combat.GrapplePossibility
	}{
		{"same size", "FR-096, table 4.10.6, " + p190, "happy", content.SizeMedium, content.SizeMedium, free, combat.GrapplePossible{}},
		{"one larger", "FR-096", "unhappy (boundary)", content.SizeMedium, content.SizeLarge, free, combat.GrapplePossible{}},
		{"two larger", "FR-096", "unhappy (boundary)", content.SizeMedium, content.SizeHuge, free, tooLarge},
		{"smaller", "FR-096", "unhappy (boundary)", content.SizeLarge, content.SizeTiny, free, combat.GrapplePossible{}},
		{"Tiny against Small", "FR-096", "unhappy (boundary)", content.SizeTiny, content.SizeSmall, free, combat.GrapplePossible{}},
		{"Tiny against Medium", "FR-096", "unhappy (boundary)", content.SizeTiny, content.SizeMedium, free, tooLarge},
		{"Gargantuan against Gargantuan", "FR-096", "unhappy (boundary)", content.SizeGargantuan, content.SizeGargantuan, free, combat.GrapplePossible{}},
		{"no hand free", "FR-096", "unhappy (state error)", content.SizeMedium, content.SizeMedium, none, noHand},
		{"both fail", "FR-096", "unhappy (rule interaction)", content.SizeMedium, content.SizeHuge, none, tooLarge},
		{"an unstated hand is not free", "package design", "unhappy (invalid input)", content.SizeMedium, content.SizeMedium, 0, noHand},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := rules.Possibility(c.attacker, c.target, c.hand); got != c.want {
				t.Errorf("Possibility = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestGrappleHold(t *testing.T) {
	t.Parallel()
	incap := condSet(t, condConds(combat.Incapacitated{}))
	cases := []struct {
		name, requirement, path string
		conds                   combat.ConditionSet
		distance, rng           int
		opts                    []combat.GrappleOption
		want                    combat.GrappleHold
		bad                     bool
	}{
		{"holds", "FR-087", "happy", combat.ConditionSet{}, 0, 5, nil, combat.GrappleHoldHolds, false},
		{"distance equal to the range holds", "FR-087", "unhappy (boundary)", combat.ConditionSet{}, 5, 5, nil, combat.GrappleHoldHolds, false},
		{"range 0, distance 0 holds", "FR-087, table 4.10.2", "unhappy (boundary)", combat.ConditionSet{}, 0, 0, nil, combat.GrappleHoldHolds, false},
		{"range 0, distance 1", "FR-087, table 4.10.2", "unhappy (boundary)", combat.ConditionSet{}, 1, 0, nil, combat.GrappleHoldOutOfRange, false},
		{"one past the range", "FR-087", "unhappy (boundary)", combat.ConditionSet{}, 6, 5, nil, combat.GrappleHoldOutOfRange, false},
		{"Incapacitated", "FR-087", "happy", incap, 0, 5, nil, combat.GrappleHoldGrapplerIncapacitated, false},
		{"Incapacitated and out of range", "FR-088", "unhappy (rule interaction)", incap, 60, 5, nil, combat.GrappleHoldGrapplerIncapacitated, false},
		{"dead, can act, in range", "FR-104", "unhappy (rule interaction)", combat.ConditionSet{}, 0, 5, []combat.GrappleOption{combat.WithGrapplerDead(true)}, combat.GrappleHoldGrapplerIncapacitated, false},
		{"dead and out of range", "FR-104, FR-088", "unhappy (rule interaction)", combat.ConditionSet{}, 60, 5, []combat.GrappleOption{combat.WithGrapplerDead(true)}, combat.GrappleHoldGrapplerIncapacitated, false},
		{"not dead by option", "FR-104", "unhappy (boundary)", combat.ConditionSet{}, 0, 5, []combat.GrappleOption{combat.WithGrapplerDead(true), combat.WithGrapplerDead(false)}, combat.GrappleHoldHolds, false},
		{"negative distance", "FR-087", "unhappy (invalid input)", combat.ConditionSet{}, -1, 5, nil, 0, true},
		{"negative range only", "FR-087, table 4.10.7", "unhappy (invalid input)", combat.ConditionSet{}, 0, -1, nil, 0, true},
		{"both negative", "FR-101", "unhappy (rule interaction)", combat.ConditionSet{}, -1, -1, nil, 0, true},
		{"negative while dead", "FR-087", "unhappy (invalid input)", combat.ConditionSet{}, -1, 5, []combat.GrappleOption{combat.WithGrapplerDead(true)}, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := rules.Hold(c.conds, c.distance, c.rng, c.opts...)
			if (err != nil) != c.bad || c.bad && !errors.Is(err, srderr.ErrInvalidInput) || got != c.want {
				t.Errorf("Hold = %v, %v, want %v (invalid %v) [%s]", got, err, c.want, c.bad, c.requirement)
			}
		})
	}
	// FR-101: the distance is checked before the range, so the two errors differ.
	_, dist := rules.Hold(combat.ConditionSet{}, -1, -1)
	_, rng := rules.Hold(combat.ConditionSet{}, 0, -1)
	if dist.Error() == rng.Error() {
		t.Errorf("both negative and range-only give the same error %q; FR-101 wants the distance named", dist)
	}
}

// Requirement: per case
// Path: per case.
func TestGrappleMovementCost(t *testing.T) {
	t.Parallel()
	const p182 = `"Movable", p. 182; "Creature Size", p. 14`
	cases := []struct {
		name, requirement, path string
		grappler, grappled      content.CreatureSize
		want                    int
	}{
		{"same size", "FR-093, " + p182, "happy", content.SizeMedium, content.SizeMedium, 1},
		{"one size smaller", "FR-093", "unhappy (boundary)", content.SizeLarge, content.SizeMedium, 1},
		{"two sizes smaller", "FR-093", "unhappy (boundary)", content.SizeLarge, content.SizeSmall, 0},
		{"Tiny", "FR-093", "unhappy (boundary)", content.SizeMedium, content.SizeTiny, 0},
		{"Tiny by Tiny", "FR-093", "unhappy (boundary)", content.SizeTiny, content.SizeTiny, 0},
		{"Small by Tiny", "FR-093", "unhappy (boundary)", content.SizeSmall, content.SizeTiny, 0},
		{"larger", "FR-093", "unhappy (boundary)", content.SizeMedium, content.SizeLarge, 1},
		{"Huge by Medium", "FR-093", "unhappy (boundary)", content.SizeHuge, content.SizeMedium, 0},
		{"Gargantuan by Large", "FR-093", "unhappy (boundary)", content.SizeGargantuan, content.SizeLarge, 0},
		{"Gargantuan by Huge", "FR-093", "unhappy (boundary)", content.SizeGargantuan, content.SizeHuge, 1},
		{"Gargantuan by Medium", "FR-093", "unhappy (boundary)", content.SizeGargantuan, content.SizeMedium, 0},
		{"an unset grappled size costs", "package design", "unhappy (invalid input)", content.SizeLarge, 0, 1},
		{"an unset grappler size costs", "package design", "unhappy (invalid input)", 0, content.SizeSmall, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := rules.MovementCost(c.grappler, c.grappled); got != c.want {
				t.Errorf("MovementCost(%v, %v) = %d, want %d [%s]", c.grappler, c.grappled, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-089, FR-090, FR-091, table 4.10.6 - EndOnSet removes Grappled and emits
// ConditionRemoved then GrappleEnded with the grappler read from the condition and the cause unchanged, for each cause;
// a set without Grappled gives NotGrappled with no event; a plain ConditionSet.Remove emits no GrappleEnded; the input
// set is unchanged.
// Path: per case.
func TestGrappleEndOnSet(t *testing.T) {
	t.Parallel()
	set := condSet(t, condConds(combat.Grappled{Grappler: gid("a")}, combat.Poisoned{}))
	for _, cause := range []combat.GrappleEndCause{combat.GrappleEndCauseEscaped, combat.GrappleEndCauseGrapplerIncapacitated, combat.GrappleEndCauseOutOfRange, combat.GrappleEndCauseReleased} {
		a, ok := rules.EndOnSet(set, gid("b"), cause).(combat.GrappleEndOnSetApplied)
		if !ok {
			t.Fatalf("EndOnSet(%v) was not applied", cause)
		}
		removed, _ := combat.NewConditionRemoved(gid("b"), content.ConditionGrappled)
		ended, _ := combat.NewGrappleEnded(gid("b"), gid("a"), cause)
		if got := a.Events(); !slices.Equal(got, []combat.Event{removed, ended}) {
			t.Errorf("events = %v, want ConditionRemoved then %v [FR-089]", got, ended)
		}
		if !a.Conditions().Equal(condSet(t, condConds(combat.Poisoned{}))) {
			t.Errorf("conditions = %v, want only Poisoned [FR-089]", a.Conditions().Explicit())
		}
		combattest.CheckEventsCopy(t, a.Events)
	}
	if !set.Equal(condSet(t, condConds(combat.Grappled{Grappler: gid("a")}, combat.Poisoned{}))) {
		t.Error("EndOnSet changed its input [FR-090]")
	}
	for name, r := range map[string]combat.GrappleEndOnSetResult{
		"empty set":      rules.EndOnSet(combat.ConditionSet{}, gid("b"), combat.GrappleEndCauseReleased),
		"other kinds":    rules.EndOnSet(condSet(t, condConds(combat.Restrained{})), gid("b"), combat.GrappleEndCauseReleased),
		"zero target":    rules.EndOnSet(set, "", combat.GrappleEndCauseReleased),
		"zero cause":     rules.EndOnSet(set, gid("b"), 0),
		"cause past end": rules.EndOnSet(set, gid("b"), 5),
	} {
		if r != (combat.GrappleEndOnSetNotGrappled{}) {
			t.Errorf("%s: EndOnSet = %#v, want NotGrappled [FR-090]", name, r)
		}
	}
	// FR-091: the plain removal reports only the condition.
	plain := set.Remove(gid("b"), content.ConditionGrappled)
	if ev := condEvents(plain); len(ev) != 1 || ev[0].Kind() != "condition_removed" {
		t.Errorf("ConditionSet.Remove events = %v, want only ConditionRemoved [FR-091]", ev)
	}
	// An immune creature still has its Grappled record removed (the immunity suppresses, it does not delete).
	immune := condSet(t, condConds(combat.Grappled{Grappler: gid("a")}), content.ConditionGrappled)
	if _, ok := rules.EndOnSet(immune, gid("b"), combat.GrappleEndCauseReleased).(combat.GrappleEndOnSetApplied); !ok {
		t.Error("EndOnSet did not remove an explicit Grappled that an immunity suppresses")
	}
}
