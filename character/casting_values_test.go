package character_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/casting-and-concentration.md FR-080, Section 5 - the enums have fixed snake_case names, "unknown"
// outside their range, and the rejection kinds are ordered as the fixed reporting order.
// Path: happy.
func TestCastingEnumNames(t *testing.T) {
	t.Parallel()
	kinds := []string{"cannot_cast_spells", "spell_unavailable", "untrained_armor", "casting_already_in_progress", "not_known", "not_prepared", "not_in_spellbook",
		"spellbook_not_readable", "not_a_ritual", "trigger_not_met", "missing_component_verbal", "missing_component_somatic", "missing_component_material",
		"slot_already_spent_this_turn", "slot_level_too_low", "no_slot_available", "casting_not_in_progress"}
	for i, want := range kinds {
		if got := character.CastRejectionKind(i + 1).String(); got != want {
			t.Errorf("CastRejectionKind(%d) = %q, want %q", i+1, got, want)
		}
	}
	if character.CastRejectionKind(0).String() != "unknown" || character.CastRejectionKind(len(kinds)+1).String() != "unknown" {
		t.Error("an out-of-range kind has a name")
	}
	for i, want := range []string{"normal", "ritual", "feature_granted", "spell_scroll", "duplicated"} {
		m := character.CastMode(i + 1)
		if m.String() != want || !m.Valid() {
			t.Errorf("CastMode(%d) = %q, want %q", i+1, m.String(), want)
		}
	}
	if character.CastMode(0).Valid() || character.CastMode(6).Valid() || character.CastMode(0).String() != "unknown" || character.CastMode(6).String() != "unknown" {
		t.Error("an out-of-range mode is valid or named")
	}
	for k, want := range map[character.ComponentKind]string{character.ComponentVerbal: "verbal", character.ComponentSomatic: "somatic", character.ComponentMaterial: "material", 0: "unknown"} {
		if k.String() != want {
			t.Errorf("ComponentKind %d = %q, want %q", k, k.String(), want)
		}
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-080, Section 5 - a CastRejection lists its reasons sorted into the
// fixed order, once each, as a copy.
// Path: happy.
func TestCastRejectionIsOrderedAndCopied(t *testing.T) {
	t.Parallel()
	got, err := castCase{spell: castSpell(t, preparedL3, 3), slot: choice(2), mods: mods(func(f *character.CastingFacts) { f.UntrainedArmorWorn = true; f.CanSpeak = false })}.run(t)
	rej := got.(character.CastSpellRejected).Rejection()
	want := kinds(character.CastRejectionUntrainedArmor, character.CastRejectionMissingVerbal, character.CastRejectionSlotLevelTooLow)
	if err != nil || !slices.Equal(rej.Reasons(), want) || rej.First() != character.CastRejectionUntrainedArmor || !rej.Has(character.CastRejectionSlotLevelTooLow) || rej.Has(character.CastRejectionNoSlotAvailable) {
		t.Fatalf("rejection = %v, %v", rej.Reasons(), err)
	}
	rej.Reasons()[0] = character.CastRejectionNoSlotAvailable
	if !slices.Equal(rej.Reasons(), want) {
		t.Error("the reasons can be changed through the accessor")
	}
	if (character.CastRejection{}).First() != 0 {
		t.Error("an empty rejection has a first reason")
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-143, FR-150 - SpellCastTerms validates its figures.
// Path: unhappy (invalid input) per case.
func TestSpellCastTerms(t *testing.T) {
	t.Parallel()
	good, err := character.NewSpellCastTerms(13, -1, 20)
	if err != nil || good.SaveDC() != 13 || good.AttackBonus() != -1 || good.CantripScalingLevel() != 20 || good.IsZero() || !(character.SpellCastTerms{}).IsZero() {
		t.Fatalf("terms = %+v, %v", good, err)
	}
	for name, c := range map[string][3]int{"DC 0": {0, 0, 0}, "negative scaling": {10, 0, -1}, "scaling 21": {10, 0, 21}} {
		if _, err := character.NewSpellCastTerms(c[0], c[1], c[2]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := character.NewSpellCastTerms(1, 0, 0); err != nil {
		t.Errorf("DC 1: %v", err)
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-129, FR-146, Section 5 - NewCastingContext rejects a class focus kind
// in FeatureFocusKinds, an invalid held focus and an invalid spell id, and copies its slices on the way in and out.
// Path: per case.
func TestNewCastingContext(t *testing.T) {
	t.Parallel()
	bad := map[string]character.CastingFacts{
		"L54 class kind in FeatureFocusKinds": {FeatureFocusKinds: []character.SpellcastingFocusKind{character.FocusHolySymbol}},
		"invalid feature kind":                {FeatureFocusKinds: []character.SpellcastingFocusKind{0}},
		"invalid held focus":                  {HeldFocus: 99},
		"invalid unavailable spell":           {UnavailableSpells: []content.SpellID{""}},
	}
	for name, f := range bad {
		if _, err := character.NewCastingContext(f); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	in := character.CastingFacts{CanCastSpells: true, HeldFocus: character.FocusPactWeapon, FeatureFocusKinds: []character.SpellcastingFocusKind{character.FocusPactWeapon}, UnavailableSpells: []content.SpellID{"wish"}}
	c, err := character.NewCastingContext(in)
	if err != nil {
		t.Fatal(err)
	}
	in.FeatureFocusKinds[0], in.UnavailableSpells[0] = character.FocusBookOfShadows, "other"
	out := c.Facts()
	out.UnavailableSpells[0] = "changed"
	if got := c.Facts(); got.FeatureFocusKinds[0] != character.FocusPactWeapon || got.UnavailableSpells[0] != "wish" || !got.CanCastSpells {
		t.Errorf("facts = %+v, want the original copied [FR-146]", got)
	}
	if zero := (character.CastingContext{}).Facts(); zero.CanCastSpells {
		t.Error("the zero context can cast")
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-147, Table U (U18) - a CastReport can be built with no class, slot,
// DC or attack bonus, and rejects an invalid spell, source, class or slot.
// Path: per case.
func TestNewCastReport(t *testing.T) {
	t.Parallel()
	r, err := character.NewCastReport(character.CastReportParts{Spell: "fireball", Source: character.CastSourceScroll})
	if err != nil {
		t.Fatal(err)
	}
	_, hasClass := r.Class()
	_, _, hasSlot := r.Slot()
	_, hasAt := r.CastAtLevel()
	_, hasSc := r.ScalingLevel()
	_, hasDC := r.SaveDC()
	_, hasBonus := r.AttackBonus()
	if hasClass || hasSlot || hasAt || hasSc || hasDC || hasBonus || r.CountsAsSlotSpell() || r.Spell() != "fireball" || r.Source() != character.CastSourceScroll {
		t.Errorf("report = %+v, want every optional field absent", r)
	}
	full, err := character.NewCastReport(character.CastReportParts{Spell: "fireball", Class: character.CasterWizard, Source: character.CastSourceSlot, HasSlotSpent: true, SlotPool: character.PoolPactMagic,
		SlotLevel: slotLevel(3), CastAtLevel: 3, ScalingLevel: 3, CountsAsSlot: true})
	pool, lvl, ok := full.Slot()
	if err != nil || !ok || pool != character.PoolPactMagic || lvl != slotLevel(3) || !full.CountsAsSlotSpell() {
		t.Errorf("full report = %+v, %v", full, err)
	}
	for name, p := range map[string]character.CastReportParts{
		"zero spell":      {Source: character.CastSourceSlot},
		"zero source":     {Spell: "a"},
		"source too high": {Spell: "a", Source: character.CastSourceMonster + 1},
		"invalid class":   {Spell: "a", Source: character.CastSourceSlot, Class: 99},
		"slot no pool":    {Spell: "a", Source: character.CastSourceSlot, HasSlotSpent: true, SlotLevel: slotLevel(1)},
		"slot no level":   {Spell: "a", Source: character.CastSourceSlot, HasSlotSpent: true, SlotPool: character.PoolPactMagic},
		"level 21":        {Spell: "a", Source: character.CastSourceSlot, CastAtLevel: 21},
		"scaling -1":      {Spell: "a", Source: character.CastSourceSlot, ScalingLevel: -1},
	} {
		if _, err := character.NewCastReport(p); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	dc, bonus := (character.SpellcastingStats{}).SaveDC(), (character.SpellcastingStats{}).AttackBonus()
	withFigures, err := character.NewCastReport(character.CastReportParts{Spell: "a", Source: character.CastSourceSlot, SaveDC: &dc, AttackBonus: &bonus})
	_, d := withFigures.SaveDC()
	_, b := withFigures.AttackBonus()
	if err != nil || !d || !b {
		t.Errorf("figures: %v %v %v", err, d, b)
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-147 (amendment A1 of monster-spellcasting.md) - a report with CastSourceMonster is accepted with no class and no slot and carries a printed DC and bonus exactly; the zero source and the value after it are still rejected; a casting in progress never holds the source.
// Path: per case.
func TestMonsterCastReportFigures(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		dc    int
		bonus int
	}{{"a DC the player formula cannot give", 25, 15}, {"the lowest DC and a negative bonus", 1, -3}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dc, err := character.NewSpellSaveDC(c.dc)
			if err != nil {
				t.Fatal(err)
			}
			bonus := character.NewSpellAttackBonus(c.bonus)
			r, err := character.NewCastReport(character.CastReportParts{Spell: "fireball", Source: character.CastSourceMonster, SaveDC: &dc, AttackBonus: &bonus})
			if err != nil {
				t.Fatal(err)
			}
			gotDC, hasDC := r.SaveDC()
			gotBonus, hasBonus := r.AttackBonus()
			_, hasClass := r.Class()
			_, _, hasSlot := r.Slot()
			if r.Source() != character.CastSourceMonster || !hasDC || gotDC.Value() != c.dc || !hasBonus || gotBonus.Value() != c.bonus || hasClass || hasSlot || r.Source().String() != "monster" {
				t.Errorf("report = %+v, want a monster source with DC %d and bonus %d only", r, c.dc, c.bonus)
			}
		})
	}
	t.Run("a monster report with a class or a slot is invalid", func(t *testing.T) {
		t.Parallel()
		for name, p := range map[string]character.CastReportParts{
			"class":         {Spell: "fireball", Source: character.CastSourceMonster, Class: character.CasterWizard},
			"slot spent":    {Spell: "fireball", Source: character.CastSourceMonster, HasSlotSpent: true},
			"counts a slot": {Spell: "fireball", Source: character.CastSourceMonster, CountsAsSlot: true},
		} {
			if _, err := character.NewCastReport(p); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
			}
		}
	})
	t.Run("a DC below 1 is invalid", func(t *testing.T) {
		t.Parallel()
		for _, n := range []int{0, -1, math.MinInt} {
			if _, err := character.NewSpellSaveDC(n); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewSpellSaveDC(%d) err = %v, want ErrInvalidInput", n, err)
			}
		}
	})
	t.Run("a casting in progress never holds the monster source", func(t *testing.T) {
		t.Parallel()
		if _, err := character.NewCastingInProgress("fireball", character.CastSourceMonster, character.CasterWizard, content.CastingTimeMode{}, nil, 1); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}

// Requirement: doc/specs/casting-and-concentration.md FR-147 - a cast report accepts a cast level and a scaling level up to MaxLevel and refuses the next one; a spent-slot pool and level are kept only when HasSlotSpent is set; a monster report carries no slot pool or slot level even when no slot is marked spent.
// Path: unhappy (boundary).
func TestCastReportBoundsAndSlotParts(t *testing.T) {
	t.Parallel()
	slot3 := character.MustSlotLevel(3)
	base := func(mod func(*character.CastReportParts)) character.CastReportParts {
		p := character.CastReportParts{Spell: "fireball", Source: character.CastSourceSlot, Class: character.CasterWizard}
		mod(&p)
		return p
	}
	cases := []struct {
		name   string
		parts  character.CastReportParts
		wantOK bool
	}{
		{"cast level at the maximum", base(func(p *character.CastReportParts) { p.CastAtLevel = character.MaxLevel }), true},
		{"cast level one past the maximum", base(func(p *character.CastReportParts) { p.CastAtLevel = character.MaxLevel + 1 }), false},
		{"scaling level at the maximum", base(func(p *character.CastReportParts) { p.ScalingLevel = character.MaxLevel }), true},
		{"scaling level one past the maximum", base(func(p *character.CastReportParts) { p.ScalingLevel = character.MaxLevel + 1 }), false},
		{"monster with only a slot pool", character.CastReportParts{Spell: "fireball", Source: character.CastSourceMonster, SlotPool: character.PoolSpellcasting}, false},
		{"monster with an undefined negative slot pool", character.CastReportParts{Spell: "fireball", Source: character.CastSourceMonster, SlotPool: character.SpellSlotPool(-1)}, false},
		{"monster with only a slot level", character.CastReportParts{Spell: "fireball", Source: character.CastSourceMonster, SlotLevel: slot3}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := character.NewCastReport(c.parts)
			if c.wantOK != (err == nil) || (!c.wantOK && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("NewCastReport = %v, want ok=%v (ErrInvalidInput when refused)", err, c.wantOK)
			}
		})
	}
	t.Run("a pool and level without HasSlotSpent are not kept", func(t *testing.T) {
		t.Parallel()
		r, err := character.NewCastReport(character.CastReportParts{Spell: "fireball", Source: character.CastSourceSlot, Class: character.CasterWizard, SlotPool: character.PoolSpellcasting, SlotLevel: slot3})
		if err != nil {
			t.Fatal(err)
		}
		if pool, level, ok := r.Slot(); ok || pool != 0 || !level.IsZero() {
			t.Errorf("Slot() = %v, %v, %v, want no slot", pool, level, ok)
		}
	})
	t.Run("a negative held focus is invalid", func(t *testing.T) {
		t.Parallel()
		if _, err := character.NewCastingContext(character.CastingFacts{HeldFocus: character.SpellcastingFocusKind(-1)}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewCastingContext(HeldFocus -1) err = %v, want ErrInvalidInput", err)
		}
	})
}
