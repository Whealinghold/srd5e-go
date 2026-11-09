package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func worn(t *testing.T, instance string, slot character.WornSlot, pair string) character.WornItem {
	t.Helper()
	p := character.PairID("")
	if pair != "" {
		p = must(character.NewPairID(pair))
	}
	return must(character.NewWornItem(inst(instance), slot, p))
}

// wearAll wears items in order, failing the test on a refusal.
func wearAll(t *testing.T, items ...character.WornItem) character.WornItems {
	t.Helper()
	w := character.WornItems{}
	for _, it := range items {
		r := must(w.Wear(it, content.ItemFitAdjusts, 0, 0))
		ok, isWorn := r.(character.WornItemsWorn)
		if !isWorn {
			t.Fatalf("wear %v refused: %#v", it, r)
		}
		w = ok.Items
	}
	return w
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-064 to FR-066, FR-068 and table "Wearing").
// Path: per case.
func TestWornItemsWear(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Multiple Items of the Same Kind", p. 103`
	cl, ft, gl := character.WornSlotCloak, character.WornSlotFootwear, character.WornSlotGloves
	cases := []struct {
		name, requirement, path string
		already                 []character.WornItem
		item                    character.WornItem
		exception               character.SlotException
		wantOccupied            character.WornSlot
	}{
		{"first cloak", p, "happy", nil, worn(t, "c1", cl, ""), 0, 0},
		{"second cloak", p, "unhappy (boundary)", []character.WornItem{worn(t, "c1", cl, "")}, worn(t, "c2", cl, ""), 0, cl},
		{"second cloak, GM allows", p, "unhappy (rule interaction)", []character.WornItem{worn(t, "c1", cl, "")}, worn(t, "c2", cl, ""), character.SlotExceptionGMAllows, 0},
		{"second armor", p, "unhappy (boundary)", []character.WornItem{worn(t, "a1", character.WornSlotArmor, "")}, worn(t, "a2", character.WornSlotArmor, ""), 0, character.WornSlotArmor},
		{"second headwear", p, "unhappy (boundary)", []character.WornItem{worn(t, "h1", character.WornSlotHeadwear, "")}, worn(t, "h2", character.WornSlotHeadwear, ""), 0, character.WornSlotHeadwear},
		{"rings have no slot", p, "unhappy (empty case)", []character.WornItem{worn(t, "r1", character.WornSlotNone, ""), worn(t, "r2", character.WornSlotNone, ""), worn(t, "r3", character.WornSlotNone, "")}, worn(t, "r4", character.WornSlotNone, ""), 0, 0},
		{"first half of a pair", p, "happy", nil, worn(t, "b1", ft, "pair1"), 0, 0},
		{"second half of the same pair", p, "happy", []character.WornItem{worn(t, "b1", ft, "pair1")}, worn(t, "b2", ft, "pair1"), 0, 0},
		{"second pair of footwear", p, "unhappy (boundary)", []character.WornItem{worn(t, "b1", ft, "pair1"), worn(t, "b2", ft, "pair1")}, worn(t, "b3", ft, "pair2"), 0, ft},
		{"third half, different pairs", p, "unhappy (boundary)", []character.WornItem{worn(t, "b1", ft, "pair1"), worn(t, "b2", ft, "pair2")}, worn(t, "b3", ft, "pair3"), 0, ft},
		{"mismatched boots are an ordinary case", p, "unhappy (rule interaction)", []character.WornItem{worn(t, "b1", ft, "pair1")}, worn(t, "b2", ft, "pair2"), 0, 0},
		{"a pair-slot item with no pair fills the slot", p, "unhappy (boundary)", []character.WornItem{worn(t, "g1", gl, "")}, worn(t, "g2", gl, "pair1"), 0, gl},
		{"a pair-slot item with no pair after a half", p, "unhappy (boundary)", []character.WornItem{worn(t, "g1", gl, "pair1")}, worn(t, "g2", gl, ""), 0, gl},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			w := wearAll(t, c.already...)
			before := w.Items()
			got, err := w.Wear(c.item, content.ItemFitAdjusts, 0, c.exception)
			if err != nil {
				t.Fatal(err)
			}
			if !w.Equal(wearAllLoose(before)) {
				t.Error("input changed")
			}
			if c.wantOccupied != 0 {
				if got != (character.WornItemsSlotOccupied{Slot: c.wantOccupied}) {
					t.Errorf("= %#v, want occupied %v [%s]", got, c.wantOccupied, c.requirement)
				}
				return
			}
			r, ok := got.(character.WornItemsWorn)
			if !ok || len(r.Items.Items()) != len(c.already)+1 {
				t.Errorf("= %#v [%s]", got, c.requirement)
			}
		})
	}
}

// wearAllLoose rebuilds WornItems from a copy of its items (no slot rules), for comparison.
func wearAllLoose(items []character.WornItem) character.WornItems {
	w := character.WornItems{}
	for _, it := range items {
		r, _ := w.Wear(it, content.ItemFitAdjusts, 0, character.SlotExceptionGMAllows)
		w = r.(character.WornItemsWorn).Items
	}
	return w
}

// Requirement: doc/specs/magic-item-rules.md FR-066 - armor, cloak, Shield and rings are all worn together (a Shield and rings take no slot).
// Path: unhappy (rule interaction).
func TestWornItemsArmorCloakShieldRings(t *testing.T) {
	t.Parallel()
	w := wearAll(t, worn(t, "a", character.WornSlotArmor, ""), worn(t, "c", character.WornSlotCloak, ""), worn(t, "s", character.WornSlotNone, ""), worn(t, "r1", character.WornSlotNone, ""))
	if len(w.Items()) != 4 {
		t.Errorf("items = %v", w.Items())
	}
	items := w.Items()
	items[0] = character.WornItem{}
	if w.Items()[0].InstanceID() != "a" || w.Items()[0].Slot() != character.WornSlotArmor || w.Items()[2].PairID() != "" {
		t.Error("Items is not a copy, or accessors")
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-070 - a fixed-size item needs a host judgement that it fits; an adjusting item fits any wearer).
// Path: per case.
func TestWornItemsFit(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Wearing and Wielding Items", p. 103`
	cases := []struct {
		name      string
		fit       content.ItemFit
		judgement character.FitJudgement
		fits      bool
	}{
		{"adjusts", content.ItemFitAdjusts, character.FitNotSupplied, true},
		{"adjusts, judged not fitting", content.ItemFitAdjusts, character.FitDoesNotFit, true},
		{"fixed, fits", content.ItemFitFixed, character.FitFits, true},
		{"fixed, does not fit", content.ItemFitFixed, character.FitDoesNotFit, false},
		{"fixed, no judgement", content.ItemFitFixed, character.FitNotSupplied, false},
	}
	for _, c := range cases {
		got, err := character.WornItems{}.Wear(worn(t, "x", character.WornSlotCloak, ""), c.fit, c.judgement, 0)
		_, ok := got.(character.WornItemsWorn)
		if err != nil || ok != c.fits {
			t.Errorf("%s: %#v, %v [%s]", c.name, got, err, p)
		}
		if !ok && got != (character.WornItemsDoesNotFit{}) {
			t.Errorf("%s: %#v", c.name, got)
		}
	}
	for name, run := range map[string]func() error{
		"zero item": func() error {
			_, e := character.WornItems{}.Wear(character.WornItem{}, content.ItemFitAdjusts, 0, 0)
			return e
		},
		"unknown fit": func() error { _, e := character.WornItems{}.Wear(worn(t, "x", 0, ""), 0, 0, 0); return e },
		"unknown exception": func() error {
			_, e := character.WornItems{}.Wear(worn(t, "x", 0, ""), content.ItemFitAdjusts, 0, 9)
			return e
		},
		"blank instance": func() error { _, e := character.NewWornItem("", 0, ""); return e },
		"unknown slot":   func() error { _, e := character.NewWornItem(inst("x"), 99, ""); return e },
		"blank pair":     func() error { _, e := character.NewPairID(" "); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-067 to FR-069 - a paired item gives its benefits only with both halves, unless the GM allows).
// Path: per case.
func TestPairedItemBenefit(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Paired Items", p. 103; "Magic Item Rules", p. 205`
	ft := character.WornSlotFootwear
	both := wearAll(t, worn(t, "b1", ft, "pair1"), worn(t, "b2", ft, "pair1"))
	one := wearAll(t, worn(t, "g1", character.WornSlotGloves, "pair1"))
	mismatched := wearAll(t, worn(t, "b1", ft, "pair1"), worn(t, "b2", ft, "pair2"))
	cases := []struct {
		name      string
		pair      string
		worn      character.WornItems
		exception character.PairException
		want      bool
	}{
		{"both halves", "pair1", both, 0, true},
		{"one half", "pair1", one, 0, false},
		{"one half, GM allows", "pair1", one, character.PairExceptionGMAllows, true},
		{"mismatched boots, first pair", "pair1", mismatched, 0, false},
		{"mismatched boots, second pair", "pair2", mismatched, 0, false},
		{"none worn, GM allows", "pair1", character.WornItems{}, character.PairExceptionGMAllows, false},
		{"no pair id", "", wearAll(t, worn(t, "x", ft, "")), character.PairExceptionGMAllows, false},
	}
	for _, c := range cases {
		id := character.PairID(c.pair)
		if got := (character.PairedItemBenefit{}).Of(id, c.worn, c.exception); got != c.want {
			t.Errorf("%s: %v [%s]", c.name, got, p)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-064 - slot capacities and names.
// Path: unhappy (boundary).
func TestWornSlotCapacity(t *testing.T) {
	t.Parallel()
	want := map[character.WornSlot]int{character.WornSlotNone: 0, character.WornSlotFootwear: 2, character.WornSlotGloves: 2, character.WornSlotBracers: 2, character.WornSlotArmor: 1, character.WornSlotHeadwear: 1, character.WornSlotCloak: 1}
	for slot, n := range want {
		if slot.Capacity() != n || slot.String() == "unknown" {
			t.Errorf("%v: capacity %d", slot, slot.Capacity())
		}
	}
	if character.WornSlot(99).Capacity() != 0 || character.WornSlot(99).String() != "unknown" || character.WornSlot(-1).String() != "unknown" {
		t.Error("unknown slot")
	}
}
