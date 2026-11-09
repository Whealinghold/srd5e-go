package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestFreeCastingRuleResolve(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		level, roll             int
		want                    character.FreeCastingOutcome
	}{
		{"level 1, roll 1", featsSpec + ` FR-053, Table F; SRD 5.2.1, "Boon of Spell Recall", Free Casting, p. 88`, "unhappy (boundary)", 1, 1, character.FreeCastingSpared},
		{"level 1, roll 2", featsSpec + " FR-053, Table F", "unhappy (boundary)", 1, 2, character.FreeCastingSpent},
		{"level 2, roll 2", featsSpec + " FR-053, Table F", "happy", 2, 2, character.FreeCastingSpared},
		{"level 3, roll 4", featsSpec + " FR-053, Table F", "happy", 3, 4, character.FreeCastingSpent},
		{"level 3, roll 3", featsSpec + " FR-053, Table F", "happy", 3, 3, character.FreeCastingSpared},
		{"level 4, roll 4", featsSpec + " FR-053, Table F", "unhappy (boundary)", 4, 4, character.FreeCastingSpared},
		{"level 4, roll 3", featsSpec + " FR-053, Table F", "unhappy (boundary)", 4, 3, character.FreeCastingSpent},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.FreeCastingRule{}.Resolve(character.MustSlotLevel(c.level), c.roll)
			if err != nil || got != c.want {
				t.Errorf("= %v, %v, want %v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-054, Table F "levels 5 and 9" - the benefit applies to slot levels 1 to 4 only; Resolve refuses an inapplicable level and a roll outside 1 to 4 with ErrInvalidInput.
// Path: unhappy (boundary).
func TestFreeCastingRuleApplies(t *testing.T) {
	t.Parallel()
	rule := character.FreeCastingRule{}
	for n := 1; n <= 9; n++ {
		l := character.MustSlotLevel(n)
		if rule.Applies(l) != (n <= 4) {
			t.Errorf("Applies(%d) = %v", n, rule.Applies(l))
		}
		if _, err := rule.Resolve(l, 1); (err == nil) != (n <= 4) || err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Resolve(%d, 1) error = %v", n, err)
		}
	}
	if rule.Applies(character.SlotLevel{}) {
		t.Error("the zero slot level does not apply")
	}
	for _, roll := range []int{0, 5, -1} {
		if _, err := rule.Resolve(character.MustSlotLevel(2), roll); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("roll %d: %v", roll, err)
		}
	}
	if _, err := rule.Resolve(character.SlotLevel{}, 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero level: %v", err)
	}
}

// Requirement: doc/specs/feats-behaviour.md section 5 - the outcomes print "spared" and "spent" and are valid only within the two.
// Path: per case.
func TestFreeCastingOutcome(t *testing.T) {
	t.Parallel()
	if character.FreeCastingSpared.String() != "spared" || character.FreeCastingSpent.String() != "spent" || character.FreeCastingOutcome(0).String() != "unknown" {
		t.Error("String")
	}
	if !character.FreeCastingSpared.Valid() || !character.FreeCastingSpent.Valid() || character.FreeCastingOutcome(0).Valid() || character.FreeCastingOutcome(3).Valid() {
		t.Error("Valid")
	}
}

// Requirement: per case
// Path: per case.
func TestGreatWeaponFightingTreat(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		rolled, want            int
		invalid                 bool
	}{
		{"1 becomes 3", featsSpec + ` FR-047; SRD 5.2.1, "Great Weapon Fighting", p. 88`, "unhappy (boundary)", 1, 3, false},
		{"2 becomes 3", featsSpec + " FR-047", "unhappy (boundary)", 2, 3, false},
		{"3 stays 3", featsSpec + " FR-047", "unhappy (boundary)", 3, 3, false},
		{"4 stays 4", featsSpec + " FR-047", "unhappy (boundary)", 4, 4, false},
		{"12 stays 12", featsSpec + " FR-047", "unhappy (boundary)", 12, 12, false},
		{"0 is an error", featsSpec + " FR-047", "unhappy (boundary)", 0, 0, true},
		{"-1 is an error", featsSpec + " FR-047", "unhappy (boundary)", -1, 0, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.GreatWeaponFighting{}.Treat(c.rolled)
			if c.invalid {
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("err = %v [%s]", err, c.requirement)
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("= %d, %v, want %d [%s]", got, err, c.want, c.requirement)
			}
		})
	}
}
