package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

type contribution struct {
	kind content.ConditionKind
	circ modifier.RollCircumstance
}

func contributionsOf(r combat.RollContributions) []contribution {
	var out []contribution
	for _, c := range r.Items() {
		out = append(out, contribution{c.Kind(), c.Circumstance()})
	}
	return out
}

// Requirement: doc/specs/conditions.md FR-105, FR-108 (SRD 5.2.1, "Advantage", p. 176; "Disadvantage", p. 181) - the
// contributions of both creatures combine with modifier.Resolve: any number of Advantages gives Advantage, any number of
// Disadvantages Disadvantage, one of each cancels to normal; the breakdown is ordered by kind.
// Path: per case.
func TestAttackRollContributionsCombine(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	cases := []struct {
		name, requirement string
		attacker, target  []combat.Condition
		distance          int
		mode              modifier.RollMode
		want              []contribution
	}{
		{"attacker Poisoned and Blinded: two Disadvantages", "pp. 177, 186", condConds(combat.Poisoned{}, combat.Blinded{}), nil, 30, modifier.WithDisadvantage,
			[]contribution{{cB, dis}, {cPo, dis}}},
		{"target Restrained and Paralyzed: two Advantages", "pp. 186, 187", nil, condConds(combat.Restrained{}, combat.Paralyzed{}), 30, modifier.WithAdvantage,
			[]contribution{{cPa, adv}, {cR, adv}}},
		{"attacker Blinded, target Paralyzed: one each", "pp. 177, 186", condConds(combat.Blinded{}), condConds(combat.Paralyzed{}), 30, modifier.Normal,
			[]contribution{{cB, dis}, {cPa, adv}}},
		{"attacker Poisoned, target Blinded and Restrained", "pp. 177, 186, 187", condConds(combat.Poisoned{}), condConds(combat.Blinded{}, combat.Restrained{}), 30, modifier.Normal,
			[]contribution{{cB, adv}, {cPo, dis}, {cR, adv}}},
		{"attacker Invisible, target Prone at 6 feet", "pp. 184, 186", condConds(combat.Invisible{}), condConds(combat.Prone{}), 6, modifier.Normal,
			[]contribution{{cIv, adv}, {cPr, dis}}},
		{"no conditions", "p. 179 (FR-109)", nil, nil, 5, modifier.Normal, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			facts := combat.ConditionAttackFacts{AttackerID: idA(), TargetID: idB(), DistanceFeet: c.distance,
				AttackerSeesTarget: combat.SightRelationCannotSee, TargetSeesAttacker: combat.SightRelationCannotSee}
			got, err := combat.ConditionEffects{}.AttackRoll(condSet(t, c.attacker), condSet(t, c.target), facts)
			if err != nil {
				t.Fatal(err)
			}
			if got.Mode() != c.mode || !slices.Equal(contributionsOf(got), c.want) {
				t.Errorf("AttackRoll = %v (%v), want %v (%v) [conditions.md FR-105; SRD 5.2.1 %s]", contributionsOf(got), got.Mode(), c.want, c.mode, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-108 - a set built in two orders gives the same mode and the same contribution
// order; the contribution slice is a copy.
// Path: unhappy (determinism).
func TestRollContributionsAreOrderedAndCopied(t *testing.T) {
	t.Parallel()
	fwd := condSet(t, condConds(combat.Poisoned{}, combat.Blinded{}, combat.Prone{}))
	rev := condSet(t, condConds(combat.Prone{}, combat.Blinded{}, combat.Poisoned{}))
	facts := combat.ConditionAttackFacts{AttackerID: idA(), TargetID: idB()}
	a, _ := combat.ConditionEffects{}.AttackRoll(fwd, combat.ConditionSet{}, facts)
	b, _ := combat.ConditionEffects{}.AttackRoll(rev, combat.ConditionSet{}, facts)
	if !slices.Equal(contributionsOf(a), contributionsOf(b)) || a.Mode() != b.Mode() {
		t.Errorf("two build orders differ: %v vs %v", contributionsOf(a), contributionsOf(b))
	}
	a.Items()[0] = combat.RollContribution{}
	if a.Items()[0].Kind() != cB {
		t.Errorf("Items aliases its storage")
	}
	if (combat.RollContributions{}).Mode() != modifier.Normal || len((combat.RollContributions{}).Items()) != 0 {
		t.Errorf("the zero RollContributions is not neutral")
	}
}
