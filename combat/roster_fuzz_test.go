package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rosFuzzID returns the id of s, or false for an input NewCombatantID refuses.
func rosFuzzID(s string) (combat.CombatantID, bool) {
	id, err := combat.NewCombatantID(s)
	return id, err == nil
}

// rosFuzzPosition builds a position from fuzzed values: kind 0 is after the anchor, anything else by count; tie is passed as it is.
func rosFuzzPosition(kind uint8, anchor string, count int, tie int) combat.RosterPosition {
	if kind%3 == 0 {
		id, _ := rosFuzzID(anchor)
		return combat.RosterAfterCombatant{Anchor: id}
	}
	return combat.RosterAtInitiative{Count: combat.InitiativeCount(count), Tie: combat.TiePlacement(tie)}
}

// rosFuzzCheck asserts the properties of a roster call: an error is invalid input, an applied encounter is valid, ranked and unique, a
// rejection leaves the encounter equal.
func rosFuzzCheck(t *testing.T, before combat.Encounter, enc combat.Encounter, rejected bool, err error) {
	t.Helper()
	switch {
	case err != nil:
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("error %v does not match ErrInvalidInput", err)
		}
	case rejected:
		if !enc.Equal(before) {
			t.Fatalf("a rejection changed the encounter")
		}
	default:
		rosHolds(t, enc)
	}
}

// FuzzNewAddCombatantToEncounterCommand: the constructor and Run never panic for any anchor, count, tie placement, reason or id; an error is
// invalid input; an applied encounter is valid with the counts not increasing and each id once; a rejection leaves it equal.
func FuzzNewAddCombatantToEncounterCommand(f *testing.F) {
	f.Add("a", 7, 1, 1, uint8(0), "x")
	f.Add("", -3, 2, 2, uint8(1), "y")
	f.Add("z", 15, 9, 0, uint8(2), "a")
	f.Add("b", math.MaxInt, 1, 1, uint8(1), "  ")
	f.Add("b", 3, 1, 2, uint8(0), "w")
	f.Fuzz(func(t *testing.T, anchor string, count, tie, reason int, kind uint8, id string) {
		e := rosEnc(t)
		x, ok := rosFuzzID(id)
		if !ok {
			return
		}
		c, err := combat.NewCombatant(combat.CombatantRef{ID: x, Kind: combat.MonsterCombatant{}}, combat.SideA, pcStatus(t, 20, 20, 0), combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		profile := combat.CombatantProfile{ID: x, Size: rosProfile(t, "a").Size, Speeds: rosProfile(t, "a").Speeds}
		cmd, err := combat.NewAddCombatantToEncounterCommand(e, c, profile, rosFuzzPosition(kind, anchor, count, tie), combat.JoinReason(reason))
		if err != nil {
			rosFuzzCheck(t, e, e, false, err)
			return
		}
		res, err := combat.NewAddCombatantToEncounter().Run(context.Background(), cmd)
		if err != nil {
			rosFuzzCheck(t, e, e, false, err)
			return
		}
		switch r := res.(type) {
		case combat.AddCombatantToEncounterApplied:
			rosFuzzCheck(t, e, r.Encounter(), false, nil)
		case combat.AddCombatantToEncounterRejected:
			rosFuzzCheck(t, e, r.Encounter(), true, nil)
		}
	})
}

// FuzzNewRemoveCombatantFromEncounterCommand: the constructor and Run never panic for any id and cause; the properties of the add hold.
func FuzzNewRemoveCombatantFromEncounterCommand(f *testing.F) {
	f.Add("c", 1)
	f.Add("b", 2)
	f.Add("", 0)
	f.Add("z", 9)
	f.Fuzz(func(t *testing.T, id string, cause int) {
		e := rosEnc(t)
		target, _ := rosFuzzID(id)
		cmd, err := combat.NewRemoveCombatantFromEncounterCommand(e, combat.ActionLedger{}, target, combat.RemovalCause(cause))
		if err != nil {
			rosFuzzCheck(t, e, e, false, err)
			return
		}
		res, err := combat.NewRemoveCombatantFromEncounter().Run(context.Background(), cmd)
		if err != nil {
			rosFuzzCheck(t, e, e, false, err)
			return
		}
		switch r := res.(type) {
		case combat.RemoveCombatantFromEncounterApplied:
			rosFuzzCheck(t, e, r.Encounter(), false, nil)
		case combat.RemoveCombatantFromEncounterRejected:
			rosFuzzCheck(t, e, r.Encounter(), true, nil)
		}
	})
}

// FuzzNewRepositionCombatantInEncounterCommand: the constructor and Run never panic for any id, anchor, count and tie placement; the
// properties of the add hold.
func FuzzNewRepositionCombatantInEncounterCommand(f *testing.F) {
	f.Add("c", "a", 10, 1, uint8(0))
	f.Add("d", "b", -5, 2, uint8(1))
	f.Add("c", "c", 0, 0, uint8(0))
	f.Add("", "", math.MaxInt, 7, uint8(2))
	f.Fuzz(func(t *testing.T, id, anchor string, count, tie int, kind uint8) {
		e := rosEnc(t)
		target, _ := rosFuzzID(id)
		cmd, err := combat.NewRepositionCombatantInEncounterCommand(e, target, rosFuzzPosition(kind, anchor, count, tie))
		if err != nil {
			rosFuzzCheck(t, e, e, false, err)
			return
		}
		res, err := combat.NewRepositionCombatantInEncounter().Run(context.Background(), cmd)
		if err != nil {
			rosFuzzCheck(t, e, e, false, err)
			return
		}
		switch r := res.(type) {
		case combat.RepositionCombatantInEncounterApplied:
			rosFuzzCheck(t, e, r.Encounter(), false, nil)
		case combat.RepositionCombatantInEncounterRejected:
			rosFuzzCheck(t, e, r.Encounter(), true, nil)
		}
	})
}
