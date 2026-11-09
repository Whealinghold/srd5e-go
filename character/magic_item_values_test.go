package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func priced(gold int64) content.MagicItemValue {
	return must(content.PricedMagicItemValueFromGold(gold))
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-017, FR-019, FR-020 and table "Magic item value cases").
// Path: per case.
func TestMagicItemValuesOf(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Rarities and Values", p. 206; "Magic Item Values by Rarity", p. 205`
	cases := []struct {
		name, requirement, path string
		rarity                  content.ItemRarity
		consumable              bool
		incorporatedGold        int64
		want                    content.MagicItemValue
	}{
		{"Common consumable", p, "unhappy (rule interaction)", content.RarityCommon, true, 0, priced(50)},
		{"Uncommon not consumable", p, "happy", content.RarityUncommon, false, 0, priced(400)},
		{"Rare consumable", p, "unhappy (rule interaction)", content.RarityRare, true, 0, priced(2000)},
		{"Legendary consumable", p, "unhappy (rule interaction)", content.RarityLegendary, true, 0, priced(100000)},
		{"+1 Armor (Plate Armor)", p, "happy", content.RarityRare, false, 1500, priced(5500)},
		{"incorporated cost 0", p, "unhappy (boundary)", content.RarityRare, false, 0, priced(4000)},
		{"Artifact plain", p, "unhappy (rule interaction)", content.RarityArtifact, false, 0, content.PricelessMagicItemValue{}},
		{"Artifact with an incorporated cost", p, "unhappy (rule interaction)", content.RarityArtifact, false, 1500, content.PricelessMagicItemValue{}},
		{"Artifact consumable", p, "unhappy (rule interaction)", content.RarityArtifact, true, 0, content.PricelessMagicItemValue{}},
		{"consumable with incorporated cost", p, "unhappy (rule interaction)", content.RarityRare, true, 1500, priced(3500)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.MagicItemValues{}.Of(c.rarity, c.consumable, c.incorporatedGold*100)
			if err != nil || got != c.want {
				t.Errorf("Of = %#v, %v, want %#v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	for name, run := range map[string]func() error{
		"negative cost":  func() error { _, e := character.MagicItemValues{}.Of(content.RarityRare, false, -1); return e },
		"unknown rarity": func() error { _, e := character.MagicItemValues{}.Of(0, false, 0); return e },
		"sum overflows": func() error {
			_, e := character.MagicItemValues{}.Of(content.RarityRare, false, content.MaxMagicItemCopper)
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-018 - a Spell Scroll is worth double its scribing cost, never halved).
// Path: per case.
func TestMagicItemValuesOfSpellScroll(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Rarities and Values", p. 206; "Scribing Spell Scrolls", p. 103`
	for _, c := range []struct {
		level int
		gold  int
		want  int64
	}{{0, 15, 30}, {1, 25, 50}, {9, 50000, 100000}} {
		row := must(content.NewSpellScrollCost(content.ItemID("row"), "row", noSource, c.level, 1, price(gp(c.gold))))
		got, err := character.MagicItemValues{}.OfSpellScroll(row)
		if err != nil || got != priced(c.want) {
			t.Errorf("level %d = %#v, %v, want %d GP [%s]", c.level, got, err, c.want, p)
		}
	}
	rated := must(content.NewSpellScrollCost(content.ItemID("row"), "row", noSource, 1, 1, price(must(content.NewCost(2, content.CoinSP, content.CostPerDay)))))
	for name, row := range map[string]content.SpellScrollCost{"zero row": {}, "rated cost": rated} {
		if _, err := (character.MagicItemValues{}).OfSpellScroll(row); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-021 - the incorporated cost is the cost in Copper Pieces; a rated cost is invalid).
// Path: per case.
func TestIncorporatedItemCostFromCost(t *testing.T) {
	t.Parallel()
	got, err := character.IncorporatedItemCost{}.FromCost("plate_x", gp(1500))
	if err != nil || got.Item() != "plate_x" || got.CostCopperPieces() != 150000 {
		t.Errorf("coins = %+v, %v", got, err)
	}
	if zero, err := (character.IncorporatedItemCost{}).FromCost("plate_x", gp(0)); err != nil || zero.CostCopperPieces() != 0 {
		t.Errorf("zero = %+v, %v", zero, err)
	}
	rated := must(content.NewCost(2, content.CoinSP, content.CostPerDay))
	if _, err := (character.IncorporatedItemCost{}).FromCost("plate_x", rated); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("rated: %v", err)
	}
	if _, err := (character.IncorporatedItemCost{}).FromCost("plate_x", content.Cost{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero cost: %v", err)
	}
}
