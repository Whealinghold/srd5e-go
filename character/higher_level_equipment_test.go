package character_test

import (
	"errors"
	"maps"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func allowanceAt(t *testing.T, level int) character.HigherLevelEquipmentAllowance {
	t.Helper()
	a, ok := character.HigherLevelEquipmentRules{}.ForStartingLevel(character.MustLevel(level)).(character.HigherLevelEquipmentAllowance)
	if !ok {
		t.Fatalf("level %d gave no allowance", level)
	}
	return a
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-081 to FR-084, Table SE
// (SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24) - level 1 gives
// NoHigherLevelEquipment; every level 2 to 20 gives an allowance whose fixed gold, rolled gold
// (1d10 x 25 or x 250 GP, absent for 2 to 4) and magic item counts by rarity follow the tiers,
// at both edges of every tier (2, 4, 5, 10, 11, 16, 17, 20); Legendary and Artifact are absent.
// Path: per case.
func TestHigherLevelEquipmentRulesTableSE(t *testing.T) {
	t.Parallel()
	type items = map[content.ItemRarity]int
	common, uncommon, rare, veryRare := content.RarityCommon, content.RarityUncommon, content.RarityRare, content.RarityVeryRare
	tiers := []struct {
		levels      []int
		fixed       int
		multiplier  int // 0 for no rolled gold
		magic       items
		requirement string
		path        string
	}{
		{[]int{2, 3, 4}, 0, 0, items{common: 1}, "FR-081, FR-082, FR-083, FR-084", "unhappy (boundary)"},
		{[]int{5, 7, 10}, 500, 25, items{common: 1, uncommon: 1}, "FR-082, FR-083, FR-084", "unhappy (boundary)"},
		{[]int{11, 13, 16}, 5000, 250, items{common: 2, uncommon: 3, rare: 1}, "FR-082, FR-083, FR-084", "unhappy (boundary)"},
		{[]int{17, 18, 20}, 20000, 250, items{common: 2, uncommon: 4, rare: 3, veryRare: 1}, "FR-082, FR-083, FR-084", "unhappy (boundary)"},
	}
	for _, tier := range tiers {
		for _, level := range tier.levels {
			a := allowanceAt(t, level)
			if got := a.FixedGold(); got != content.MustCost(tier.fixed, content.CoinGP, 0) {
				t.Errorf("level %d fixed gold = %v, want %d GP [%s]", level, got, tier.fixed, tier.requirement)
			}
			rolled := a.RolledGold()
			switch {
			case tier.multiplier == 0 && len(rolled) != 0:
				t.Errorf("level %d rolled gold = %v, want none", level, rolled)
			case tier.multiplier != 0 && (len(rolled) != 1 || rolled[0].Expression().Notation() != "1d10" ||
				rolled[0].Multiplier() != content.MustCost(tier.multiplier, content.CoinGP, 0)):
				t.Errorf("level %d rolled gold = %v, want 1d10 x %d GP", level, rolled, tier.multiplier)
			}
			if got := a.MagicItems(); !maps.Equal(got, tier.magic) {
				t.Errorf("level %d magic items = %v, want %v", level, got, tier.magic)
			}
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-081 - starting level 1
// (and the zero Level, which is not a level) give NoHigherLevelEquipment, and every level 2 to 20
// an allowance.
// Path: unhappy (boundary).
func TestHigherLevelEquipmentRulesLevelOneHasNone(t *testing.T) {
	t.Parallel()
	rules := character.HigherLevelEquipmentRules{}
	if got := rules.ForStartingLevel(character.MustLevel(1)); got != (character.NoHigherLevelEquipment{}) {
		t.Errorf("level 1 = %#v, want NoHigherLevelEquipment", got)
	}
	if got := rules.ForStartingLevel(character.Level{}); got != (character.NoHigherLevelEquipment{}) {
		t.Errorf("zero level = %#v, want NoHigherLevelEquipment", got)
	}
	for n := 2; n <= 20; n++ {
		if _, ok := rules.ForStartingLevel(character.MustLevel(n)).(character.HigherLevelEquipmentAllowance); !ok {
			t.Errorf("level %d gave no allowance", n)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-116, Table SM (SRD 5.2.1,
// p. 24) - TotalGold gives the fixed gold plus the roll times the multiplier at the minimum and
// maximum of every tier, and the fixed gold alone, with no roll, for a tier without rolled gold.
// Path: per case.
func TestTotalGoldTableSM(t *testing.T) {
	t.Parallel()
	one, ten, five := 1, 10, 5
	for _, c := range []struct {
		name  string
		level int
		roll  *int
		want  int
	}{
		{"level 3, none rolled", 3, nil, 0},
		{"level 5, minimum", 5, &one, 525},
		{"level 10, maximum", 10, &ten, 750},
		{"level 11, minimum", 11, &one, 5250},
		{"level 16, maximum", 16, &ten, 7500},
		{"level 17, minimum", 17, &one, 20250},
		{"level 20, maximum", 20, &ten, 22500},
		{"level 18, middle", 18, &five, 21250},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := allowanceAt(t, c.level).TotalGold(c.roll)
			if err != nil || got != c.want {
				t.Errorf("TotalGold = %d, %v, want %d GP [FR-116, Table SM]", got, err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-116 - TotalGold returns an
// error matching srderr.ErrInvalidInput carrying the argument and value for a nil roll on a tier
// with rolled gold, rolls 0 and 11, and any roll on a tier with no rolled gold.
// Path: per case.
func TestTotalGoldInvalidInput(t *testing.T) {
	t.Parallel()
	zero, eleven, one := 0, 11, 1
	for _, c := range []struct {
		name  string
		level int
		roll  *int
		want  *character.HigherLevelEquipmentError
	}{
		{"nil roll for a rolled tier", 5, nil, &character.HigherLevelEquipmentError{Argument: "roll"}},
		{"roll 0", 5, &zero, &character.HigherLevelEquipmentError{Argument: "roll", InvalidValue: 0}},
		{"roll 11", 5, &eleven, &character.HigherLevelEquipmentError{Argument: "roll", InvalidValue: 11}},
		{"a roll for a tier with no rolled gold", 3, &one, &character.HigherLevelEquipmentError{Argument: "roll", InvalidValue: 1}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := allowanceAt(t, c.level).TotalGold(c.roll)
			if got != 0 || !errors.Is(err, c.want) || !isInvalid(err) || err.Error() == "" {
				t.Errorf("TotalGold = %d, %v, want an invalid-input error %v", got, err, c.want)
			}
		})
	}
	if errors.Is(&character.HigherLevelEquipmentError{Argument: "roll"}, &character.HigherLevelEquipmentError{Argument: "x"}) {
		t.Error("errors with another argument must not match")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-089 - the allowance and
// RolledGold compare by value, and the slice and map they expose are copies.
// Path: unhappy (immutability).
func TestAllowanceIsImmutableAndComparesByValue(t *testing.T) {
	t.Parallel()
	a := allowanceAt(t, 12)
	if a != allowanceAt(t, 11) || a == allowanceAt(t, 17) || a == allowanceAt(t, 5) {
		t.Error("allowances must compare by value")
	}
	a.MagicItems()[content.RarityCommon] = 99
	rolled := a.RolledGold()
	rolled[0] = character.RolledGold{}
	if a.MagicItems()[content.RarityCommon] != 2 || len(a.RolledGold()) != 1 || a.RolledGold()[0].IsZero() {
		t.Error("the exposed map and slice must be copies")
	}
	if !(character.RolledGold{}).IsZero() {
		t.Error("the zero RolledGold must be zero")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-116 - HigherLevelEquipmentError matches only an
// equal *HigherLevelEquipmentError (and srderr.ErrInvalidInput): another error, a typed nil target and an error
// of another value do not match.
// Path: unhappy (invalid input).
func TestHigherLevelEquipmentErrorMatchesOnlyAnEqualError(t *testing.T) {
	t.Parallel()
	e := &character.HigherLevelEquipmentError{Argument: "roll", InvalidValue: 11}
	var nilTarget *character.HigherLevelEquipmentError
	if errors.Is(e, errors.New("other")) || errors.Is(e, nilTarget) || errors.Is(e, &character.HigherLevelEquipmentError{Argument: "roll", InvalidValue: 12}) {
		t.Error("a different target must not match")
	}
	if !e.Is(e) || e.Is(nil) || nilTarget.Is(e) {
		t.Error("Is must match an equal error and reject nil on either side")
	}
}
