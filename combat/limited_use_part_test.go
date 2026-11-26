package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-limited-use.md FR-005 - LimitedUsePartRef returns an error matching srderr.ErrInvalidInput for a negative index or an undefined section; two references with equal section and index are equal and usable as map keys.
// Path: per case.
func TestNewLimitedUsePartRef(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		section    combat.MonsterPartSection
		index      int
		wantOK     bool
	}{
		{"first part of Actions", "happy", combat.MonsterPartActions, 0, true},
		{"last section", "unhappy (boundary)", combat.MonsterPartLegendaryActions, 7, true},
		{"negative index", "unhappy (invalid input)", combat.MonsterPartActions, -1, false},
		{"zero section", "unhappy (invalid input)", 0, 0, false},
		{"section past the last", "unhappy (invalid input)", combat.MonsterPartLegendaryActions + 1, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewLimitedUsePartRef(c.section, c.index)
			if c.wantOK != (err == nil) || (!c.wantOK && (!errors.Is(err, srderr.ErrInvalidInput) || !r.IsZero())) ||
				(c.wantOK && (r.Section() != c.section || r.Index() != c.index || r.IsZero())) {
				t.Errorf("NewLimitedUsePartRef(%v, %d) = %v, %v, want ok %v [%sFR-005]", c.section, c.index, r, err, c.wantOK, limitedUseSpec)
			}
		})
	}
	m := map[combat.LimitedUsePartRef]int{luBite(t): 1}
	if m[luRef(t, combat.MonsterPartActions, 0)] != 1 || luBite(t) == luRef(t, combat.MonsterPartTraits, 0) || luBite(t) == luRef(t, combat.MonsterPartActions, 1) {
		t.Error("references must compare by section and index [FR-005]")
	}
}

// Requirement: doc/specs/monster-limited-use.md section 3 (vocabulary) - the closed enums answer Valid and a String of the catalogue value, and the zero or an undefined value is invalid and reads "unknown".
// Path: per case.
func TestLimitedUseEnumNames(t *testing.T) {
	t.Parallel()
	pairs := []struct{ got, want string }{
		{combat.InLair.String(), "in_lair"}, {combat.OutsideLair.String(), "outside_lair"}, {combat.LairStatus(0).String(), "unknown"},
		{combat.MonsterPartTraits.String(), "traits"}, {combat.MonsterPartActions.String(), "actions"}, {combat.MonsterPartBonusActions.String(), "bonus_actions"},
		{combat.MonsterPartReactions.String(), "reactions"}, {combat.MonsterPartLegendaryActions.String(), "legendary_actions"}, {combat.MonsterPartSection(9).String(), "unknown"},
		{combat.RechargeReady.String(), "ready"}, {combat.RechargeSpent.String(), "spent"}, {combat.RechargeReadiness(0).String(), "unknown"},
		{combat.RechargeRegained.String(), "regained"}, {combat.RechargeNotRegained.String(), "not_regained"}, {combat.RechargeRollResult(0).String(), "unknown"},
		{combat.UseAvailable.String(), "available"}, {combat.UseExhausted.String(), "exhausted"}, {combat.UseNotLimited.String(), "not_limited"}, {combat.LimitedUseAvailability(0).String(), "unknown"},
		{combat.LimitedUseRefusedExhausted.String(), "exhausted"}, {combat.LimitedUseRefusedNotLimited.String(), "not_limited_use"}, {combat.LimitedUseRefusal(0).String(), "unknown"},
	}
	for _, p := range pairs {
		if p.got != p.want {
			t.Errorf("String() = %q, want %q", p.got, p.want)
		}
	}
	if !combat.InLair.Valid() || !combat.OutsideLair.Valid() || combat.LairStatus(0).Valid() || combat.LairStatus(3).Valid() {
		t.Error("LairStatus.Valid wrong")
	}
	if combat.MonsterPartSection(0).Valid() || !combat.MonsterPartTraits.Valid() || !combat.MonsterPartLegendaryActions.Valid() || combat.MonsterPartSection(6).Valid() {
		t.Error("MonsterPartSection.Valid wrong")
	}
}
