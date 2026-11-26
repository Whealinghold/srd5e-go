package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

func bindScene(tb testing.TB) combat.BindingScene {
	tb.Helper()
	return combat.BindingScene{Kind: combat.BindingKindChain, Willingness: combat.WillingnessUnwilling, DistanceFeet: 5, Target: gid("victim"),
		TargetSize: content.SizeMedium, TargetConditions: gearHeldDown(tb), HoldID: "h"}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-045, FR-047, FR-048, FR-049, table 4.9.2 - Assess refuses in the order targetWilling, outOfReach (Chain and Manacles beyond 5 feet; a Rope at any distance is not), sizeNotAllowed (Manacles take Small and Medium only), targetNotHeldDown, targetAlreadyHeld, idInUse, and otherwise gives the binder's check: Strength (Athletics) 13 for a Chain, Dexterity (Sleight of Hand) 13 for Manacles, 10 for a Rope (SRD 5.2.1, "Chain", "Manacles", "Rope", pp. 96, 98, 99).
// Path: per case.
func TestBindingRulesAssess(t *testing.T) {
	t.Parallel()
	cuffs, rope := combat.BindingKindManacles, combat.BindingKindRope
	existing := gearHolds(t, gearHold(t, holdIn("other", combat.HoldKindChain)))
	cases := []struct {
		name, path string
		change     func(*combat.BindingScene)
		want       combat.GearRejection
		skill      content.Skill
		dc         d20.DifficultyClass
	}{
		{"chain", "happy", func(*combat.BindingScene) {}, 0, content.SkillAthletics, 13},
		{"manacles", "happy", func(s *combat.BindingScene) { s.Kind = cuffs }, 0, content.SkillSleightOfHand, 13},
		{"rope", "happy", func(s *combat.BindingScene) { s.Kind = rope }, 0, content.SkillSleightOfHand, 10},
		{"willing", "unhappy (rule interaction)", func(s *combat.BindingScene) { s.Willingness = combat.WillingnessWilling }, combat.GearRejectionTargetWilling, 0, 0},
		{"willing and unconscious", "unhappy (rule interaction)", func(s *combat.BindingScene) {
			s.Willingness, s.TargetConditions = combat.WillingnessWilling, condSet(t, condConds(combat.Unconscious{}))
		}, combat.GearRejectionTargetWilling, 0, 0},
		{"willing wins over reach", "unhappy (rule interaction)", func(s *combat.BindingScene) { s.Willingness, s.DistanceFeet = combat.WillingnessWilling, 50 }, combat.GearRejectionTargetWilling, 0, 0},
		{"chain at 5 feet", "unhappy (boundary)", func(s *combat.BindingScene) { s.DistanceFeet = 5 }, 0, content.SkillAthletics, 13},
		{"chain at 6 feet", "unhappy (boundary)", func(s *combat.BindingScene) { s.DistanceFeet = 6 }, combat.GearRejectionOutOfReach, 0, 0},
		{"manacles at 6 feet", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.DistanceFeet = cuffs, 6 }, combat.GearRejectionOutOfReach, 0, 0},
		{"rope at the largest distance", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.DistanceFeet = rope, math.MaxInt }, 0, content.SkillSleightOfHand, 10},
		{"reach wins over size", "unhappy (rule interaction)", func(s *combat.BindingScene) { s.Kind, s.DistanceFeet, s.TargetSize = cuffs, 6, content.SizeLarge }, combat.GearRejectionOutOfReach, 0, 0},
		{"manacles on a Tiny creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.TargetSize = cuffs, content.SizeTiny }, combat.GearRejectionSizeNotAllowed, 0, 0},
		{"manacles on a Small creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.TargetSize = cuffs, content.SizeSmall }, 0, content.SkillSleightOfHand, 13},
		{"manacles on a Medium creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.TargetSize = cuffs, content.SizeMedium }, 0, content.SkillSleightOfHand, 13},
		{"manacles on a Large creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.TargetSize = cuffs, content.SizeLarge }, combat.GearRejectionSizeNotAllowed, 0, 0},
		{"manacles on a Gargantuan creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.TargetSize = cuffs, content.SizeGargantuan }, combat.GearRejectionSizeNotAllowed, 0, 0},
		{"chain on a Gargantuan creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.TargetSize = content.SizeGargantuan }, 0, content.SkillAthletics, 13},
		{"rope on a Tiny creature", "unhappy (boundary)", func(s *combat.BindingScene) { s.Kind, s.TargetSize = rope, content.SizeTiny }, 0, content.SkillSleightOfHand, 10},
		{"size wins over held down", "unhappy (rule interaction)", func(s *combat.BindingScene) {
			s.Kind, s.TargetSize, s.TargetConditions = cuffs, content.SizeLarge, combat.ConditionSet{}
		}, combat.GearRejectionSizeNotAllowed, 0, 0},
		{"no conditions", "unhappy (state error)", func(s *combat.BindingScene) { s.TargetConditions = combat.ConditionSet{} }, combat.GearRejectionTargetNotHeldDown, 0, 0},
		{"prone is not held down", "unhappy (state error)", func(s *combat.BindingScene) { s.TargetConditions = condSet(t, condConds(combat.Prone{})) }, combat.GearRejectionTargetNotHeldDown, 0, 0},
		{"already held by a chain", "unhappy (state error)", func(s *combat.BindingScene) { s.Holds = existing }, combat.GearRejectionTargetAlreadyHeld, 0, 0},
		{"a rope beside a chain", "unhappy (rule interaction)", func(s *combat.BindingScene) { s.Holds, s.Kind = existing, rope }, 0, content.SkillSleightOfHand, 10},
		{"hold id in use", "unhappy (state error)", func(s *combat.BindingScene) { s.Holds, s.Kind, s.HoldID = existing, cuffs, "other" }, combat.GearRejectionIDInUse, 0, 0},
		{"another creature's chain", "unhappy (rule interaction)", func(s *combat.BindingScene) { s.Target = "elsewhere"; s.Holds = existing }, 0, content.SkillAthletics, 13},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := bindScene(t)
			c.change(&s)
			check, got, err := combat.BindingRules{}.Assess(s)
			if err != nil || got != c.want {
				t.Fatalf("Assess = %v, %v; want %v [%s FR-045]", got, err, c.want, gearSpec)
			}
			if check.Skill() != c.skill || check.DifficultyClass() != c.dc {
				t.Errorf("check = %+v, want %v DC %d [%s FR-047]", check, c.skill, c.dc, gearSpec)
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-046 - the target is held down when its effective conditions include Grappled, Incapacitated or Restrained, and Unconscious, Paralyzed, Petrified and Stunned qualify through their implied Incapacitated; other conditions do not (SRD 5.2.1, "Chain", p. 96; "Condition", p. 179).
// Path: per case.
func TestBindingRulesHeldDown(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		cond combat.Condition
		want bool
	}{
		"grappled": {combat.Grappled{Grappler: gid("g")}, true}, "incapacitated": {combat.Incapacitated{}, true}, "restrained": {combat.Restrained{}, true},
		"unconscious": {combat.Unconscious{}, true}, "paralyzed": {combat.Paralyzed{}, true}, "petrified": {combat.Petrified{}, true}, "stunned": {combat.Stunned{}, true},
		"prone": {combat.Prone{}, false}, "blinded": {combat.Blinded{}, false}, "poisoned": {combat.Poisoned{}, false}, "frightened": {combat.Frightened{Source: gid("f")}, false},
	}
	for name, c := range cases {
		s := bindScene(t)
		s.TargetConditions = condSet(t, condConds(c.cond))
		_, got, err := combat.BindingRules{}.Assess(s)
		if err != nil || (got == 0) != c.want {
			t.Errorf("%s: Assess refusal = %v, %v; held down should be %v [%s FR-046]", name, got, err, c.want, gearSpec)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-045, FR-071 - Assess returns a GearError for a zero target or blank hold id (blank_id), a negative distance (negative_distance) and a kind, willingness or size that is not valid (invalid_value).
// Path: unhappy (invalid input).
func TestBindingRulesAssessInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		change func(*combat.BindingScene)
		want   combat.GearProblem
	}{
		"zero target":         {func(s *combat.BindingScene) { s.Target = "" }, combat.GearProblemBlankID},
		"blank hold id":       {func(s *combat.BindingScene) { s.HoldID = "" }, combat.GearProblemBlankID},
		"negative distance":   {func(s *combat.BindingScene) { s.DistanceFeet = -1 }, combat.GearProblemNegativeDistance},
		"invalid kind":        {func(s *combat.BindingScene) { s.Kind = 0 }, combat.GearProblemInvalidValue},
		"invalid willingness": {func(s *combat.BindingScene) { s.Willingness = 0 }, combat.GearProblemInvalidValue},
		"invalid size":        {func(s *combat.BindingScene) { s.TargetSize = 0 }, combat.GearProblemInvalidValue},
	}
	for name, c := range cases {
		s := bindScene(t)
		c.change(&s)
		check, refusal, err := combat.BindingRules{}.Assess(s)
		if !errors.Is(err, &combat.GearError{Problem: c.want}) || check != (combat.HoldCheck{}) || refusal != 0 {
			t.Errorf("%s: Assess = %+v, %v, %v; want a %v GearError", name, check, refusal, err, c.want)
		}
	}
}
