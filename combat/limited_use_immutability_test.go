package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/monster-limited-use.md FR-031 - every operation returns a new state and leaves the receiver and its inputs unchanged; the entry list, RechargeRollsNeeded and the events are copies.
// Path: unhappy (immutability).
func TestLimitedUseStateIsImmutable(t *testing.T) {
	t.Parallel()
	s := luSpent(t, luSpent(t, luStart(t, luDrake(t), combat.OutsideLair), luSpray(t)), luMind(t))
	snapshot := luStart(t, luDrake(t), combat.OutsideLair)
	snapshot = luSpent(t, luSpent(t, snapshot, luSpray(t)), luMind(t))
	rolls := map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(6)}
	_ = luSpent(t, s, luRes(t))
	s.Spend(luMind(t))
	s.AfterRest(character.LongRest)
	s.ChangeLair(combat.InLair)
	tr, _ := s.StartOfOwnTurn(rolls)
	_, _ = s.TakeLegendaryAction(luWindow(t, "b", 1), combat.ConditionActionAvailabilityAvailable)
	if !s.Equal(snapshot) {
		t.Error("an operation changed the receiver")
	}
	if len(rolls) != 1 || rolls[luSpray(t)] != luRoll(6) {
		t.Error("StartOfOwnTurn changed its roll map")
	}
	entries := s.Entries()
	entries[0] = combat.PerDayEntry{}
	needed := s.RechargeRollsNeeded()
	needed[0] = combat.LimitedUsePartRef{}
	events := tr.Events()
	events[0] = nil
	if !s.Equal(snapshot) || s.RechargeRollsNeeded()[0] != luSpray(t) || tr.Events()[0] == nil {
		t.Error("a returned slice is not a copy [FR-031]")
	}
	long := s.AfterRest(character.LongRest)
	restored := long.Events()[0].(combat.LimitedUsesRestored).Subjects()
	restored[0] = nil
	if long.Events()[0].(combat.LimitedUsesRestored).Subjects()[0] == nil {
		t.Error("Subjects is not a copy")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-032 - a state and its entries are value objects: equal fields give equal values and any differing field makes them unequal.
// Path: per case.
func TestLimitedUseStateEquality(t *testing.T) {
	t.Parallel()
	base := luStart(t, luDrake(t), combat.OutsideLair)
	if !base.Equal(luStart(t, luDrake(t), combat.OutsideLair)) {
		t.Error("two states started the same way must be equal")
	}
	other := luStart(t, luDrake(t), combat.OutsideLair)
	cases := map[string]combat.LimitedUseState{
		"a part spent":      luSpent(t, other, luMind(t)),
		"another lair":      luStart(t, luDrake(t), combat.InLair),
		"another combatant": luMust(combat.StartLimitedUse(luDrake(t), combat.MustCombatantID("other"), combat.OutsideLair)),
		"a legendary use":   luTake(t, other, luWindow(t, "b", 1)),
		"another monster":   luStart(t, luMonster(t, nil), combat.OutsideLair),
		"the zero state":    {},
	}
	for name, c := range cases {
		if base.Equal(c) {
			t.Errorf("%s: base.Equal(state) is true, want the states to differ", name)
		}
		if c.Equal(base) {
			t.Errorf("%s: state.Equal(base) is true, want the states to differ", name)
		}
	}
	// States that differ in one field only: the stat block id, the lair status.
	plain := luStart(t, luMonster(t, nil), combat.OutsideLair)
	otherID := luMust(combat.StartLimitedUse(luMust(content.NewMonster("other_beast", "Other Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), luFields(t))),
		combat.MustCombatantID(luSelf), combat.OutsideLair))
	onlyLair := luStart(t, luMonster(t, nil), combat.InLair)
	for name, c := range map[string]combat.LimitedUseState{"the stat block id": otherID, "the lair status": onlyLair} {
		if plain.Equal(c) || c.Equal(plain) {
			t.Errorf("states differing in %s only must not be equal", name)
		}
	}
	if !(combat.LimitedUseState{}).Equal(combat.LimitedUseState{}) {
		t.Error("two zero states must be equal")
	}
	e := slices.Clone(base.Entries())
	if e[0] != base.Entries()[0] {
		t.Error("equal entries must compare equal")
	}
}
