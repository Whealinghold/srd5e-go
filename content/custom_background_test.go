package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// priced returns a valuation from a table of item name to value; a name absent
// from it has no price.
func priced(m map[string]content.EquipmentItemValue) content.EquipmentValuation {
	return func(i content.EquipmentItem) (content.EquipmentItemValue, bool) {
		v, ok := m[i.Name()]
		return v, ok
	}
}

func item(name string, qty int) content.EquipmentItem { return content.MustEquipmentItem(name, qty) }

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-073, "Cases: custom background equipment" - the
// five-step equipment check: exactly 5000 Copper Pieces including gold, no Martial item, an unpriced item skips
// the total; problems come in item order, then the total.
// Path: per case.
func TestCustomBackgroundEquipmentProblems(t *testing.T) {
	t.Parallel()
	thousand := content.MustEquipmentItemValue(1000, false)
	martial := content.MustEquipmentItemValue(1000, true)
	free := content.MustEquipmentItemValue(0, false)
	table := map[string]content.EquipmentItemValue{"a": thousand, "b": thousand, "m": martial, "free": free, "m50": content.MustEquipmentItemValue(5000, true)}
	cases := []struct {
		name, path string
		option     content.EquipmentOption
		valuation  content.EquipmentValuation
		want       []content.CustomBackgroundEquipmentProblem
	}{
		{"50 GP by gold alone", "happy", content.MustEquipmentOption(nil, 50), priced(table), nil},
		{"items plus gold, twice", "happy", content.MustEquipmentOption([]content.EquipmentItem{item("a", 2)}, 30), priced(table), nil},
		{"items plus gold, three of one", "happy", content.MustEquipmentOption([]content.EquipmentItem{item("b", 3)}, 20), priced(table), nil},
		{"49 GP", "unhappy (boundary)", content.MustEquipmentOption(nil, 49), priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentValueNotFiftyGold{CopperPieces: 4900}}},
		{"51 GP", "unhappy (boundary)", content.MustEquipmentOption(nil, 51), priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentValueNotFiftyGold{CopperPieces: 5100}}},
		{"empty option", "unhappy (empty case)", content.EquipmentOption{}, priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentValueNotFiftyGold{CopperPieces: 0}}},
		{"unpriced item skips the total", "unhappy (missing content)", content.MustEquipmentOption([]content.EquipmentItem{item("x", 1)}, 50), priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentItemUnpriced{ItemName: "x"}}},
		{"unpriced and wrong total", "unhappy (rule interaction)", content.MustEquipmentOption([]content.EquipmentItem{item("x", 1), item("a", 1)}, 10), priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentItemUnpriced{ItemName: "x"}}},
		{"nil valuation prices nothing", "unhappy (missing content)", content.MustEquipmentOption([]content.EquipmentItem{item("a", 1)}, 40), nil, []content.CustomBackgroundEquipmentProblem{content.EquipmentItemUnpriced{ItemName: "a"}}},
		{"Martial item at 50 GP", "unhappy (rule interaction)", content.MustEquipmentOption([]content.EquipmentItem{item("m50", 1)}, 0), priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentItemExcluded{ItemName: "m50"}}},
		{"Martial item and wrong total", "unhappy (rule interaction)", content.MustEquipmentOption([]content.EquipmentItem{item("m", 1)}, 10), priced(table),
			[]content.CustomBackgroundEquipmentProblem{content.EquipmentItemExcluded{ItemName: "m"}, content.EquipmentValueNotFiftyGold{CopperPieces: 2000}}},
		{"problems in item order", "unhappy (rule interaction)", content.MustEquipmentOption([]content.EquipmentItem{item("m", 1), item("x", 1), item("m", 1)}, 0), priced(table),
			[]content.CustomBackgroundEquipmentProblem{content.EquipmentItemExcluded{ItemName: "m"}, content.EquipmentItemUnpriced{ItemName: "x"}, content.EquipmentItemExcluded{ItemName: "m"}}},
		{"zero-cost item", "unhappy (boundary)", content.MustEquipmentOption([]content.EquipmentItem{item("free", 1)}, 50), priced(table), nil},
		{"quantity multiplies the price", "unhappy (boundary)", content.MustEquipmentOption([]content.EquipmentItem{item("a", 4)}, 10), priced(table), nil},
		{"quantity multiplies the price, wrong total", "unhappy (boundary)", content.MustEquipmentOption([]content.EquipmentItem{item("a", 4)}, 11), priced(table), []content.CustomBackgroundEquipmentProblem{content.EquipmentValueNotFiftyGold{CopperPieces: 5100}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := content.CustomBackgroundEquipmentProblems(c.option, c.valuation)
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %#v, want %#v [FR-073, %s]", got, c.want, c.path)
			}
		})
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-073 - a huge quantity and price saturate the
// total instead of overflowing it, and the total is still reported wrong.
// Path: unhappy (boundary).
func TestCustomBackgroundEquipmentTotalSaturates(t *testing.T) {
	t.Parallel()
	v := content.MustEquipmentItemValue(content.MaxEquipmentQuantity, false)
	opt := content.MustEquipmentOption([]content.EquipmentItem{
		item("a", content.MaxEquipmentQuantity), item("a", content.MaxEquipmentQuantity), item("a", content.MaxEquipmentQuantity),
		item("a", content.MaxEquipmentQuantity), item("a", content.MaxEquipmentQuantity),
	}, content.MaxEquipmentGold)
	got := content.CustomBackgroundEquipmentProblems(opt, priced(map[string]content.EquipmentItemValue{"a": v}))
	if len(got) != 1 {
		t.Fatalf("problems = %v, want the one wrong total", got)
	}
	if p, ok := got[0].(content.EquipmentValueNotFiftyGold); !ok || p.CopperPieces != 1<<61 {
		t.Errorf("problem = %#v, want the total saturated at 2^61", got[0])
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-073, "negative cost" row - an item value is
// built from a price of 0 to math.MaxInt32; a negative price is an error.
// Path: unhappy (invalid input).
func TestNewEquipmentItemValue(t *testing.T) {
	t.Parallel()
	if v, err := content.NewEquipmentItemValue(0, true); err != nil || v.CopperPieces() != 0 || !v.IsMartialWeaponOrArmor() {
		t.Errorf("NewEquipmentItemValue(0, true) = (%+v, %v) [boundary]", v, err)
	}
	if v, err := content.NewEquipmentItemValue(1<<31-1, false); err != nil || v.CopperPieces() != 1<<31-1 || v.IsMartialWeaponOrArmor() {
		t.Errorf("NewEquipmentItemValue(max) = (%+v, %v) [boundary]", v, err)
	}
	for _, cp := range []int{-1, plusOne(1<<31 - 1)} {
		_, err := content.NewEquipmentItemValue(cp, false)
		wantBlockError(t, err, "EquipmentItemValue", content.BlockOutOfRange)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("price %d: error does not wrap ErrInvalidInput", cp)
		}
		assertMustPanics(t, func() { content.MustEquipmentItemValue(cp, false) })
	}
}
