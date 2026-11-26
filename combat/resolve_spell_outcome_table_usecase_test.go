package combat_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func otUse(tb testing.TB, r combat.Roller, opts ...combat.Option) combat.ResolveSpellOutcomeTable {
	tb.Helper()
	return luMust(combat.NewResolveSpellOutcomeTable(r, opts...))
}

// otRequests lists the expressions the roller was asked for, in order.
func otRequests(r *dicetest.FixedRoller) []string {
	out := []string{}
	for _, e := range r.Requests() {
		out = append(out, e.Notation())
	}
	return out
}

// otResolve runs cmd against a fixed roller of faces and fails the test on an error.
func otResolve(t *testing.T, faces []int, cmd combat.ResolveSpellOutcomeTableCommand) (combat.ResolveSpellOutcomeTableResult, []string) {
	t.Helper()
	roller := dicetest.NewFixedRoller(faces...)
	res, err := otUse(t, roller).Resolve(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Resolve(%#v) = %v", cmd, err)
	}
	return res, otRequests(roller)
}

func otAttempt() d20.RollAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
}

func otTargets(names ...string) []combat.PrismaticSaveTarget {
	out := make([]combat.PrismaticSaveTarget, len(names))
	for i, n := range names {
		out[i] = combat.PrismaticSaveTarget{ID: combat.MustCombatantID(n), Attempt: otAttempt()}
	}
	return out
}

// otRepeat returns n copies of v.
func otRepeat(n, v int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}
	return out
}

// Requirement: doc/specs/spell-outcome-tables.md FR-013 to FR-017, FR-006 - Teleport rolls the d100 of the familiarity row; a Mishap deals 3d10 Force to each teleporting creature (one roll for an object), then the d100 is rolled again, with every Mishap kept in order; Off Target rolls 2d12 then 1d8; Similar Area and On Target roll nothing more; the certain rows (a permanent circle, a linked object) request no die; a False destination ends as Similar Area (SRD 5.2.1, "Teleport", "Mishap", "Off Target", p. 168).
// Path: per case.
func TestResolveTeleport(t *testing.T) {
	t.Parallel()
	one := luMust(combat.NewTeleportingCreatures(otIDs("caster")))
	three := luMust(combat.NewTeleportingCreatures(otIDs("caster", "ally", "mule")))
	on, similar := combat.ArrivesOnTarget{}, combat.ArrivesInSimilarArea{}
	off := func(miles int, d combat.CompassDirection) combat.ArrivesOffTarget {
		return luMust(combat.NewArrivesOffTarget(miles, d))
	}
	cases := []struct {
		name, requirement, path string
		familiarity             combat.TeleportFamiliarity
		subject                 combat.TeleportSubject
		faces                   []int
		wantRequests            []string
		wantMishaps             [][]int
		wantArrival             combat.TeleportTableArrival
	}{
		{"very familiar arrives on target", "FR-017", "happy", combat.VeryFamiliar, one, []int{25}, []string{"1d100"}, nil, on},
		{"Similar Area rolls nothing more", "FR-017", "happy", combat.VeryFamiliar, one, []int{6}, []string{"1d100"}, nil, similar},
		{"Off Target 2 miles east", "FR-016", "unhappy (boundary)", combat.VeryFamiliar, one, []int{14, 1, 1, 1}, []string{"1d100", "2d12", "1d8"}, nil, off(2, combat.East)},
		{"Off Target 24 miles northeast", "FR-016", "unhappy (boundary)", combat.VeryFamiliar, one, []int{24, 12, 12, 8}, []string{"1d100", "2d12", "1d8"}, nil, off(24, combat.Northeast)},
		{"a Mishap damages each of three creatures, then rerolls", "FR-013", "happy", combat.VeryFamiliar, three, []int{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 25},
			[]string{"1d100", "3d10", "3d10", "3d10", "1d100"}, [][]int{{6, 15, 24}}, on},
		{"an object takes one Mishap roll", "FR-013", "unhappy (boundary)", combat.VeryFamiliar, combat.TeleportingObject{}, []int{5, 2, 3, 4, 100},
			[]string{"1d100", "3d10", "1d100"}, [][]int{{9}}, on},
		{"two Mishaps in a row", "FR-014", "unhappy (rule interaction)", combat.SeenCasually, one, []int{1, 1, 1, 1, 33, 2, 2, 2, 54},
			[]string{"1d100", "3d10", "1d100", "3d10", "1d100"}, [][]int{{3}, {6}}, on},
		{"a Mishap then Off Target", "FR-015", "unhappy (determinism)", combat.ViewedOnceOrDescribed, one, []int{43, 10, 10, 10, 54, 12, 11, 5},
			[]string{"1d100", "3d10", "1d100", "2d12", "1d8"}, [][]int{{30}}, off(23, combat.West)},
		{"a False destination ends as Similar Area", "FR-014", "unhappy (rule interaction)", combat.FalseDestination, one, []int{50, 5, 5, 5, 51},
			[]string{"1d100", "3d10", "1d100"}, [][]int{{15}}, similar},
		{"a False destination at face 100", "FR-012", "unhappy (boundary)", combat.FalseDestination, one, []int{100}, []string{"1d100"}, nil, similar},
		{"a permanent circle requests no die", "FR-011", "unhappy (rule interaction)", combat.PermanentCircle, three, nil, []string{}, nil, on},
		{"a linked object requests no die", "FR-011", "unhappy (rule interaction)", combat.LinkedObject, combat.TeleportingObject{}, nil, []string{}, nil, on},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, requests := otResolve(t, c.faces, combat.RollTeleportOutcomeCommand{Subject: c.subject, Familiarity: c.familiarity})
			got := res.(combat.SpellTableTeleportRolled).Outcome()
			var mishaps [][]int
			for _, m := range got.Mishaps() {
				var amounts []int
				for _, d := range m.Damage() {
					if d.Type() != content.DamageForce {
						t.Errorf("Mishap damage type = %v, want Force", d.Type())
					}
					amounts = append(amounts, d.Amount())
				}
				mishaps = append(mishaps, amounts)
			}
			if !slices.Equal(requests, c.wantRequests) || !reflect.DeepEqual(mishaps, c.wantMishaps) || got.Arrival() != c.wantArrival ||
				got.Familiarity() != c.familiarity {
				t.Errorf("requests = %v, mishaps = %v, arrival = %#v, want %v, %v, %#v [%s%s]", requests, mishaps, got.Arrival(), c.wantRequests, c.wantMishaps, c.wantArrival, otSpec, c.requirement)
			}
		})
	}
	for i, want := range []combat.CompassDirection{combat.East, combat.Southeast, combat.South, combat.Southwest, combat.West, combat.Northwest, combat.North, combat.Northeast} {
		res, _ := otResolve(t, []int{14, 3, 4, i + 1}, combat.RollTeleportOutcomeCommand{Subject: one, Familiarity: combat.VeryFamiliar})
		if got := res.(combat.SpellTableTeleportRolled).Outcome().Arrival(); got != combat.TeleportTableArrival(off(7, want)) {
			t.Errorf("d8 face %d = %#v, want 7 miles %v (FR-016)", i+1, got, want)
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-019, FR-092 - a Teleport outcome applies nothing, is equal to another outcome of the same rolls and unequal to a different one, and its slices are copies: changing what Mishaps, Damage or IDs returned changes nothing.
// Path: unhappy (immutability).
func TestTeleportOutcomeImmutabilityAndEquality(t *testing.T) {
	t.Parallel()
	subject := luMust(combat.NewTeleportingCreatures(otIDs("a", "b")))
	cmd := combat.RollTeleportOutcomeCommand{Subject: subject, Familiarity: combat.VeryFamiliar}
	faces := []int{1, 1, 1, 1, 2, 2, 2, 25}
	first, _ := otResolve(t, faces, cmd)
	again, _ := otResolve(t, faces, cmd)
	a, b := first.(combat.SpellTableTeleportRolled).Outcome(), again.(combat.SpellTableTeleportRolled).Outcome()
	if !a.Equal(b) || !reflect.DeepEqual(first, again) {
		t.Error("equal commands and roller sequences must give equal outcomes (FR-092)")
	}
	a.Mishaps()[0].Damage()[0] = combat.MustDamage(1000, content.DamageForce)
	a.Mishaps()[0] = combat.TeleportMishap{}
	if !a.Equal(b) || a.Mishaps()[0].Damage()[0].Amount() != 3 {
		t.Error("a copy of Mishaps must not change the outcome")
	}
	other, _ := otResolve(t, []int{1, 1, 1, 1, 2, 2, 2, 6}, cmd)
	obj, _ := otResolve(t, []int{1, 1, 1, 1, 25}, combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.VeryFamiliar})
	if a.Equal(other.(combat.SpellTableTeleportRolled).Outcome()) || a.Equal(obj.(combat.SpellTableTeleportRolled).Outcome()) {
		t.Error("a different arrival or subject must be unequal")
	}
	if !reflect.DeepEqual(a.Subject(), subject) {
		t.Errorf("Subject() = %#v", a.Subject())
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-030 to FR-036, table C - Confusion rolls 1d10 and then the extra die of its row, in that order: 1d4 for a random move (1 north, 2 east, 3 south, 4 west), 1d<n> to pick among n of 2 or more candidates, none for one candidate, none candidate or the other rows; the attack is one melee attack and the target does not move (SRD 5.2.1, "Confusion", p. 117).
// Path: per case.
func TestResolveConfusion(t *testing.T) {
	t.Parallel()
	ids := func(n int) []combat.CombatantID {
		out := make([]combat.CombatantID, n)
		for i := range out {
			out[i] = combat.CombatantID(fmt.Sprintf("c%04d", i+1))
		}
		return out
	}
	cases := []struct {
		name, requirement, path string
		faces                   []int
		candidates              []combat.CombatantID
		wantRequests            []string
		want                    string
		wantRow                 combat.ConfusionRow
	}{
		{"C1 moves north", "FR-031", "happy", []int{1, 1}, nil, []string{"1d10", "1d4"}, "moves north", combat.ConfusionRowMovesRandomly},
		{"C1 moves east", "FR-031", "happy", []int{1, 2}, nil, []string{"1d10", "1d4"}, "moves east", combat.ConfusionRowMovesRandomly},
		{"C1 moves south", "FR-031", "happy", []int{1, 3}, nil, []string{"1d10", "1d4"}, "moves south", combat.ConfusionRowMovesRandomly},
		{"C1 moves west", "FR-031", "unhappy (boundary)", []int{1, 4}, nil, []string{"1d10", "1d4"}, "moves west", combat.ConfusionRowMovesRandomly},
		{"C2 does nothing at 2", "FR-032", "unhappy (boundary)", []int{2}, ids(3), []string{"1d10"}, "does nothing", combat.ConfusionRowDoesNothing},
		{"C3 does nothing at 6", "FR-032", "unhappy (boundary)", []int{6}, nil, []string{"1d10"}, "does nothing", combat.ConfusionRowDoesNothing},
		{"C4 picks one of three at random", "FR-033", "unhappy (boundary)", []int{7, 2}, ids(3), []string{"1d10", "1d3"}, "attacks c0002", combat.ConfusionRowAttacksRandomCreature},
		{"C4 picks the first of two", "FR-033", "unhappy (boundary)", []int{7, 1}, ids(2), []string{"1d10", "1d2"}, "attacks c0001", combat.ConfusionRowAttacksRandomCreature},
		{"C4 picks the last of 1000", "FR-034", "unhappy (boundary)", []int{7, 1000}, ids(1000), []string{"1d10", "1d1000"}, "attacks c1000", combat.ConfusionRowAttacksRandomCreature},
		{"C5 one candidate needs no die", "FR-033", "unhappy (boundary)", []int{8}, ids(1), []string{"1d10"}, "attacks c0001", combat.ConfusionRowAttacksRandomCreature},
		{"C6 no creature in reach", "FR-033", "unhappy (empty and minimal)", []int{8}, nil, []string{"1d10"}, "no creature in reach", combat.ConfusionRowAttacksRandomCreature},
		{"C6 an empty list", "FR-033", "unhappy (empty and minimal)", []int{7}, []combat.CombatantID{}, []string{"1d10"}, "no creature in reach", combat.ConfusionRowAttacksRandomCreature},
		{"C7 the target chooses at 9", "FR-032", "unhappy (boundary)", []int{9}, ids(2), []string{"1d10"}, "chooses", combat.ConfusionRowChoosesBehavior},
		{"C8 the target chooses at 10", "FR-032", "happy", []int{10}, nil, []string{"1d10"}, "chooses", combat.ConfusionRowChoosesBehavior},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, requests := otResolve(t, c.faces, combat.RollConfusionBehaviorCommand{CreaturesWithinReach: c.candidates})
			turn := res.(combat.SpellTableConfusionRolled).Turn()
			if !slices.Equal(requests, c.wantRequests) || otConfusion(turn.Behavior()) != c.want || turn.Row() != c.wantRow || !turn.CannotTakeBonusActionsOrReactions() {
				t.Errorf("requests = %v, behaviour = %v, row = %v, want %v, %v, %v [%s%s]", requests, otConfusion(turn.Behavior()), turn.Row(), c.wantRequests, c.want, c.wantRow, otSpec, c.requirement)
			}
			if a, ok := turn.Behavior().(combat.ConfusionAttacksRandomCreature); ok && (a.AttackCount() != 1 || a.TargetMoves()) {
				t.Error("the attack is one attack and the target does not move (FR-035)")
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-034, plan P10 - a list of 1001 candidates is refused before any die with an error that matches srderr.ErrInvalidInput and ErrSpellTableLimit while 1000 is accepted; a repeated or zero id is an invalid input; the list is not changed.
// Path: unhappy (invalid input).
func TestResolveConfusionCandidateLimits(t *testing.T) {
	t.Parallel()
	ids := make([]combat.CombatantID, 1001)
	for i := range ids {
		ids[i] = combat.CombatantID(fmt.Sprintf("c%04d", i))
	}
	cases := []struct {
		name  string
		list  []combat.CombatantID
		limit bool
	}{
		{"1001 candidates", ids, true},
		{"a repeated id", []combat.CombatantID{"a", "b", "a"}, false},
		{"a zero id", []combat.CombatantID{"a", ""}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(7, 1)
			before := slices.Clone(c.list)
			res, err := otUse(t, roller).Resolve(context.Background(), combat.RollConfusionBehaviorCommand{CreaturesWithinReach: c.list})
			if res != nil || !errors.Is(err, srderr.ErrInvalidInput) || errors.Is(err, combat.ErrSpellTableLimit) != c.limit || len(roller.Requests()) != 0 || !slices.Equal(before, c.list) {
				t.Errorf("Resolve = %v, %v with %d requests, want an invalid input (limit %v) and no die [%sFR-034]", res, err, len(roller.Requests()), c.limit, otSpec)
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-040 to FR-043 - Blink rolls one 1d6: 1 to 3 stay and keep the location whichever it is; 4 to 6 on the other plane vanish to the Ethereal Plane; 4 to 6 already on the Ethereal Plane end the spell and carry the return as data (SRD 5.2.1, "Blink", pp. 113-114).
// Path: per case.
func TestResolveBlink(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		face                    int
		location                combat.BlinkLocation
		want                    string
	}{
		{"1 stays on the other plane", "FR-042", "unhappy (boundary)", 1, combat.BlinkOnOtherPlane, "stays other_plane"},
		{"3 stays on the other plane", "FR-042", "unhappy (boundary)", 3, combat.BlinkOnOtherPlane, "stays other_plane"},
		{"1 stays on the Ethereal Plane", "FR-042", "unhappy (boundary)", 1, combat.BlinkOnEtherealPlane, "stays ethereal_plane"},
		{"3 stays on the Ethereal Plane", "FR-042", "unhappy (boundary)", 3, combat.BlinkOnEtherealPlane, "stays ethereal_plane"},
		{"4 vanishes", "FR-040", "unhappy (boundary)", 4, combat.BlinkOnOtherPlane, "vanishes"},
		{"6 vanishes", "FR-040", "unhappy (boundary)", 6, combat.BlinkOnOtherPlane, "vanishes"},
		{"4 already ethereal ends the spell", "FR-041", "unhappy (rule interaction)", 4, combat.BlinkOnEtherealPlane, "ends and returns"},
		{"6 already ethereal ends the spell", "FR-041", "unhappy (rule interaction)", 6, combat.BlinkOnEtherealPlane, "ends and returns"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, requests := otResolve(t, []int{c.face}, combat.RollBlinkTurnEndCommand{Location: c.location})
			if got := res.(combat.SpellTableBlinkRolled).Outcome(); otBlink(got) != c.want || !slices.Equal(requests, []string{"1d6"}) {
				t.Errorf("outcome = %v, requests = %v, want %v and one 1d6 [%s%s]", otBlink(got), requests, c.want, otSpec, c.requirement)
			}
		})
	}
}

// otConfusion names a Confusion behaviour for an assertion.
func otConfusion(b combat.ConfusionBehavior) string {
	switch v := b.(type) {
	case combat.ConfusionMovesRandomly:
		return "moves " + v.Direction().String()
	case combat.ConfusionDoesNothing:
		return "does nothing"
	case combat.ConfusionAttacksRandomCreature:
		return "attacks " + string(v.Target)
	case combat.ConfusionNoCreatureInReach:
		return "no creature in reach"
	case combat.ConfusionTargetChoosesBehavior:
		return "chooses"
	}
	return fmt.Sprintf("unknown %#v", b)
}

// otBlink names a Blink outcome for an assertion.
func otBlink(o combat.BlinkTurnEndOutcome) string {
	switch v := o.(type) {
	case combat.BlinkStaysOnPlane:
		return "stays " + v.Location().String()
	case combat.BlinkVanishesToEtherealPlane:
		return "vanishes"
	case combat.BlinkEndsAlreadyEthereal:
		if _, ok := v.Return().(combat.BlinkReturnsToOtherPlane); ok {
			return "ends and returns"
		}
	}
	return fmt.Sprintf("unknown %#v", o)
}

// Requirement: doc/specs/spell-outcome-tables.md FR-051, FR-052, FR-053, FR-004 - Reincarnate rolls 1d10 and rolls again on each 1 (rerolls counts the 1s); a chosen species rolls no die; a creature that is not a Humanoid is refused with NotHumanoid, one dead more than 10 days with DeadTooLong, both in that order, and exactly 10 days is accepted; the check applies to a chosen species too (SRD 5.2.1, "Reincarnate", p. 158).
// Path: per case.
func TestResolveReincarnation(t *testing.T) {
	t.Parallel()
	const tenDays = 10 * 24 * time.Hour
	ones := func(n int, then int) []int { return append(otRepeat(n, 1), then) }
	cases := []struct {
		name, requirement, path string
		cmd                     combat.RollReincarnationSpeciesCommand
		faces                   []int
		wantRequests            int
		wantSpecies             content.SpeciesID
		wantRerolls             int
		chosen                  bool
		wantRefusals            []combat.SpellTableRefusal
	}{
		{"face 2 is dragonborn", "FR-050", "happy", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}, []int{2}, 1, "dragonborn", 0, false, nil},
		{"face 10 is tiefling", "FR-050", "unhappy (boundary)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}, []int{10}, 1, "tiefling", 0, false, nil},
		{"1, 1, 4 is an elf after two rerolls", "FR-051", "unhappy (rule interaction)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}, []int{1, 1, 4}, 3, "elf", 2, false, nil},
		{"100 ones then a species", "FR-004", "unhappy (boundary)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}, ones(100, 3), 101, "dwarf", 100, false, nil},
		{"a chosen species rolls nothing", "FR-052", "happy", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid, ChosenSpecies: "goliath"}, nil, 0, "goliath", 0, true, nil},
		{"dead exactly 10 days is accepted", "FR-053", "unhappy (boundary)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid, DeadFor: tenDays}, []int{8}, 1, "human", 0, false, nil},
		{"dead 10 days and a millisecond is refused", "FR-053", "unhappy (boundary)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid, DeadFor: tenDays + time.Millisecond}, nil, 0, "", 0, false, []combat.SpellTableRefusal{combat.DeadTooLong}},
		{"a Beast is refused", "FR-053", "unhappy (rule interaction)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeBeast, DeadFor: time.Hour}, nil, 0, "", 0, false, []combat.SpellTableRefusal{combat.NotHumanoid}},
		{"a Beast dead too long is refused twice, in order", "FR-053", "unhappy (rule interaction)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeBeast, DeadFor: tenDays + 1}, nil, 0, "", 0, false, []combat.SpellTableRefusal{combat.NotHumanoid, combat.DeadTooLong}},
		{"the check applies to a chosen species", "FR-053", "unhappy (rule interaction)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeUndead, ChosenSpecies: "elf"}, nil, 0, "", 0, false, []combat.SpellTableRefusal{combat.NotHumanoid}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := otUse(t, roller).Resolve(context.Background(), c.cmd)
			if err != nil || len(roller.Requests()) != c.wantRequests {
				t.Fatalf("Resolve = %v, %v with %d requests, want %d [%sFR-051, FR-052, FR-053]", res, err, len(roller.Requests()), c.wantRequests, otSpec)
			}
			for _, e := range roller.Requests() {
				if e.Notation() != "1d10" {
					t.Errorf("request %v, want only 1d10", e)
				}
			}
			switch v := res.(type) {
			case combat.SpellTableRefused:
				if !slices.Equal(v.Refusals(), c.wantRefusals) {
					t.Errorf("refusals = %v, want %v", v.Refusals(), c.wantRefusals)
				}
			case combat.SpellTableReincarnationRolled:
				switch o := v.Outcome().(type) {
				case combat.ReincarnationSpeciesRolled:
					if c.chosen || o.Species() != c.wantSpecies || o.Rerolls() != c.wantRerolls {
						t.Errorf("rolled %q after %d rerolls, want %q after %d", o.Species(), o.Rerolls(), c.wantSpecies, c.wantRerolls)
					}
				case combat.ReincarnationSpeciesChosen:
					if !c.chosen || o.Species() != c.wantSpecies {
						t.Errorf("chosen %q, want %q", o.Species(), c.wantSpecies)
					}
				}
			default:
				t.Errorf("result = %#v", res)
			}
		})
	}
	for face := 2; face <= 10; face++ {
		species, _, _ := combat.ReincarnationSpeciesFor(face)
		res, _ := otResolve(t, []int{face}, combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid})
		if got := res.(combat.SpellTableReincarnationRolled).Outcome().(combat.ReincarnationSpeciesRolled).Species(); got != species {
			t.Errorf("face %d = %q, want %q (R%d)", face, got, species, face)
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-053, FR-052, FR-090 - a negative dead time, a creature type that is not valid and a chosen species that is not a species id are invalid input before any die, with an error that matches srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestResolveReincarnationInvalidInput(t *testing.T) {
	t.Parallel()
	humanoid := content.CreatureTypeHumanoid
	cmds := map[string]combat.RollReincarnationSpeciesCommand{
		"a negative dead time": {CreatureType: humanoid, DeadFor: -time.Second},
		"no creature type":     {DeadFor: time.Hour},
		"a bad species id":     {CreatureType: humanoid, ChosenSpecies: "Not A Species"},
	}
	for name, cmd := range cmds {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := otUse(t, roller).Resolve(context.Background(), cmd)
			if res != nil || !errors.Is(err, srderr.ErrInvalidInput) || len(roller.Requests()) != 0 {
				t.Errorf("Resolve = %v, %v with %d requests, want an invalid input and no die", res, err, len(roller.Requests()))
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-004, FR-091 - a roll-again chain that keeps asking is stopped: 101 consecutive 1s on Reincarnate, a Teleport that is a Mishap 101 times in a row and a Prismatic Spray 8 that keeps rerolling 8 return a *srderr.PortError matching srderr.ErrContractViolation with no result; exactly 100 rerolls are allowed (the 101st reroll request is not made).
// Path: unhappy (boundary).
func TestRollAgainChainsStop(t *testing.T) {
	t.Parallel()
	humanoid := content.CreatureTypeHumanoid
	cases := []struct {
		name         string
		cmd          combat.ResolveSpellOutcomeTableCommand
		roller       *dicetest.FixedRoller
		wantRequests int
	}{
		{"Reincarnate 101 ones", combat.RollReincarnationSpeciesCommand{CreatureType: humanoid}, dicetest.NewFixedRoller(otRepeat(101, 1)...), 101},
		{"Teleport 101 Mishaps", combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.FalseDestination},
			dicetest.NewMinimumRoller(), 202},
		{"Prismatic Spray an 8 that keeps asking", combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: otTargets("a")}, dicetest.NewMaximumRoller(), 102},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := otUse(t, c.roller).Resolve(context.Background(), c.cmd)
			envPortFailure(t, err)
			if res != nil || !errors.Is(err, srderr.ErrContractViolation) || len(c.roller.Requests()) != c.wantRequests {
				t.Errorf("Resolve = %v, %v with %d requests, want a contract violation after %d [%sFR-004]", res, err, len(c.roller.Requests()), c.wantRequests, otSpec)
			}
		})
	}
	// Exactly 100 rerolls are allowed: a first 8, then a ray that shows 100 eights before a 2, then a 3 (orange and yellow).
	faces := slices.Concat([]int{8}, otRepeat(100, 8), []int{2, 3}, otRepeat(12, 1))
	roller := dicetest.NewFixedRoller(faces...)
	res, err := otUse(t, roller).Resolve(context.Background(), combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: []combat.PrismaticSaveTarget{{ID: "a", Attempt: d20.AutomaticFailure{}}}})
	if err != nil {
		t.Fatalf("Resolve with 100 rerolls = %v, %v, want a result", res, err)
	}
	if rays := res.(combat.SpellTablePrismaticSprayRolled).Targets()[0].Rays(); !slices.Equal(rays, []combat.PrismaticColor{combat.PrismaticOrange, combat.PrismaticYellow}) {
		t.Errorf("rays = %v, want orange and yellow", rays)
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-061, FR-062, FR-063, FR-064, FR-065, FR-066, FR-067, FR-068 - Prismatic Spray rolls every target's ray first (an 8 gives two more rays, each rerolled while it shows 8), then one 12d6 for the whole cast if any ray is a damage ray, then each target's Dexterity save; a damage ray gives the full roll, halved on a success; indigo and violet strike only on a failed save; ChooseToFail and AutomaticFailure roll no d20; no target is valid and rolls nothing (SRD 5.2.1, "Prismatic Spray", "Prismatic Rays", pp. 154-155; "Damage against Multiple Targets", p. 16).
// Path: per case.
func TestResolvePrismaticSpray(t *testing.T) {
	t.Parallel()
	twelve := otRepeat(12, 3) // 36 damage
	fail, pass := 5, 15       // against DC 15
	type strike struct {
		amount int
		typ    content.DamageType
		halved bool
	}
	cases := []struct {
		name, requirement, path string
		targets                 []combat.PrismaticSaveTarget
		faces                   []int
		wantRequests            []string
		wantRays                [][]combat.PrismaticColor
		wantStrikes             [][]strike
		wantConditions          [][]content.ConditionKind
	}{
		{"red, a failed save", "FR-065", "happy", otTargets("a"), slices.Concat([]int{1}, twelve, []int{fail}), []string{"1d8", "12d6", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticRed}}, [][]strike{{{36, content.DamageFire, false}}}, nil},
		{"red, a successful save halves", "FR-065", "happy", otTargets("a"), slices.Concat([]int{1}, twelve, []int{pass}), []string{"1d8", "12d6", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticRed}}, [][]strike{{{36, content.DamageFire, true}}}, nil},
		{"orange Acid", "FR-060", "happy", otTargets("a"), slices.Concat([]int{2}, twelve, []int{fail}), []string{"1d8", "12d6", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticOrange}}, [][]strike{{{36, content.DamageAcid, false}}}, nil},
		{"blue Cold", "FR-060", "happy", otTargets("a"), slices.Concat([]int{5}, twelve, []int{fail}), []string{"1d8", "12d6", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticBlue}}, [][]strike{{{36, content.DamageCold, false}}}, nil},
		{"indigo, a failed save", "FR-066", "happy", otTargets("a"), []int{6, fail}, []string{"1d8", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticIndigo}}, [][]strike{nil}, [][]content.ConditionKind{{content.ConditionRestrained}}},
		{"indigo, a successful save gives nothing", "FR-066", "unhappy (rule interaction)", otTargets("a"), []int{6, pass}, []string{"1d8", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticIndigo}}, [][]strike{nil}, [][]content.ConditionKind{nil}},
		{"violet, a failed save", "FR-066", "happy", otTargets("a"), []int{7, fail}, []string{"1d8", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticViolet}}, [][]strike{nil}, [][]content.ConditionKind{{content.ConditionBlinded}}},
		{"violet, a successful save gives nothing", "FR-066", "unhappy (rule interaction)", otTargets("a"), []int{7, pass}, []string{"1d8", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticViolet}}, [][]strike{nil}, [][]content.ConditionKind{nil}},
		{"an 8 gives two rays and one damage roll", "FR-061", "unhappy (rule interaction)", otTargets("a"), slices.Concat([]int{8, 3, 5}, twelve, []int{fail}),
			[]string{"1d8", "1d8", "1d8", "12d6", "1d20"}, [][]combat.PrismaticColor{{combat.PrismaticYellow, combat.PrismaticBlue}},
			[][]strike{{{36, content.DamageLightning, false}, {36, content.DamageCold, false}}}, nil},
		{"an 8 inside the rerolls is rolled again", "FR-061", "unhappy (rule interaction)", otTargets("a"), slices.Concat([]int{8, 8, 2, 4}, twelve, []int{fail}),
			[]string{"1d8", "1d8", "1d8", "1d8", "12d6", "1d20"}, [][]combat.PrismaticColor{{combat.PrismaticOrange, combat.PrismaticGreen}},
			[][]strike{{{36, content.DamageAcid, false}, {36, content.DamagePoison, false}}}, nil},
		{"two equal rays are applied twice", "FR-063", "unhappy (rule interaction)", otTargets("a"), slices.Concat([]int{8, 1, 1}, twelve, []int{pass}),
			[]string{"1d8", "1d8", "1d8", "12d6", "1d20"}, [][]combat.PrismaticColor{{combat.PrismaticRed, combat.PrismaticRed}},
			[][]strike{{{36, content.DamageFire, true}, {36, content.DamageFire, true}}}, nil},
		{"an 8 with a damage ray and indigo", "FR-066", "unhappy (rule interaction)", otTargets("a"), slices.Concat([]int{8, 1, 6}, twelve, []int{fail}),
			[]string{"1d8", "1d8", "1d8", "12d6", "1d20"}, [][]combat.PrismaticColor{{combat.PrismaticRed, combat.PrismaticIndigo}},
			[][]strike{{{36, content.DamageFire, false}}}, [][]content.ConditionKind{{content.ConditionRestrained}}},
		{"two targets share one damage roll and roll their saves in order", "FR-062", "unhappy (determinism)", otTargets("a", "b"),
			slices.Concat([]int{1, 2}, twelve, []int{fail, pass}), []string{"1d8", "1d8", "12d6", "1d20", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticRed}, {combat.PrismaticOrange}},
			[][]strike{{{36, content.DamageFire, false}}, {{36, content.DamageAcid, true}}}, nil},
		{"no damage ray rolls no damage", "FR-062", "unhappy (determinism)", otTargets("a", "b"), []int{6, 7, fail, fail}, []string{"1d8", "1d8", "1d20", "1d20"},
			[][]combat.PrismaticColor{{combat.PrismaticIndigo}, {combat.PrismaticViolet}}, [][]strike{nil, nil},
			[][]content.ConditionKind{{content.ConditionRestrained}, {content.ConditionBlinded}}},
		{"ChooseToFail rolls no d20", "FR-064", "unhappy (rule interaction)", []combat.PrismaticSaveTarget{{ID: "a", Attempt: d20.ChooseToFail{}}},
			slices.Concat([]int{1}, twelve), []string{"1d8", "12d6"}, [][]combat.PrismaticColor{{combat.PrismaticRed}}, [][]strike{{{36, content.DamageFire, false}}}, nil},
		{"AutomaticFailure rolls no d20", "FR-064", "unhappy (rule interaction)", []combat.PrismaticSaveTarget{{ID: "a", Attempt: d20.AutomaticFailure{}}},
			[]int{6}, []string{"1d8"}, [][]combat.PrismaticColor{{combat.PrismaticIndigo}}, [][]strike{nil}, [][]content.ConditionKind{{content.ConditionRestrained}}},
		{"no target", "FR-067", "unhappy (empty and minimal)", nil, nil, []string{}, nil, nil, nil},
		{"an empty list", "FR-067", "unhappy (empty and minimal)", []combat.PrismaticSaveTarget{}, nil, []string{}, nil, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, requests := otResolve(t, c.faces, combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: c.targets})
			got := res.(combat.SpellTablePrismaticSprayRolled).Targets()
			if !slices.Equal(requests, c.wantRequests) || len(got) != len(c.targets) {
				t.Fatalf("requests = %v for %d targets, want %v for %d [%s%s]", requests, len(got), c.wantRequests, len(c.targets), otSpec, c.requirement)
			}
			for i, tr := range got {
				var strikes []strike
				var conditions []content.ConditionKind
				for _, s := range tr.Strikes() {
					switch v := s.(type) {
					case combat.PrismaticDamageStrike:
						strikes = append(strikes, strike{v.Damage().Amount(), v.Damage().Type(), v.Halved()})
					case combat.PrismaticConditionStrike:
						conditions = append(conditions, v.Condition())
					}
				}
				var wantConditions []content.ConditionKind
				if i < len(c.wantConditions) {
					wantConditions = c.wantConditions[i]
				}
				if tr.ID != c.targets[i].ID || !slices.Equal(tr.Rays(), c.wantRays[i]) || !slices.Equal(strikes, c.wantStrikes[i]) || !slices.Equal(conditions, wantConditions) {
					t.Errorf("target %d: rays %v, strikes %v, conditions %v; want %v, %v, %v [%s%s]", i, tr.Rays(), strikes, conditions, c.wantRays[i], c.wantStrikes[i], wantConditions, otSpec, c.requirement)
				}
				if tr.Save() == nil {
					t.Errorf("target %d has no save evidence", i)
				}
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-078, FR-079, FR-065, FR-066 - a creature crossing a Prismatic Wall makes a Dexterity save against each layer still present, in order from red; each damage layer rolls its own 12d6 after its save whatever the save (half on a success), indigo and violet strike only on a failed save; a wall with no layer left gives an empty crossing and rolls nothing (SRD 5.2.1, "Prismatic Wall", p. 155).
// Path: per case.
func TestResolvePrismaticWallCrossing(t *testing.T) {
	t.Parallel()
	fail, pass := 5, 15
	all := slices.Concat([]int{fail}, otRepeat(12, 1), []int{pass}, otRepeat(12, 2), []int{fail}, otRepeat(12, 3), []int{pass}, otRepeat(12, 4), []int{fail}, otRepeat(12, 5), []int{fail, pass})
	cases := []struct {
		name, requirement, path string
		destroyed               int
		attempt                 d20.SaveAttempt
		faces                   []int
		wantRequests            []string
		want                    []string
	}{
		{"all seven layers", "FR-078", "happy", 0, otAttempt(), all,
			[]string{"1d20", "12d6", "1d20", "12d6", "1d20", "12d6", "1d20", "12d6", "1d20", "12d6", "1d20", "1d20"},
			[]string{"red fire 12 full", "orange acid 24 half", "yellow lightning 36 full", "green poison 48 half", "blue cold 60 full", "indigo restrained", "violet none"}},
		{"three layers left", "FR-078", "unhappy (boundary)", 4, otAttempt(), slices.Concat([]int{fail}, otRepeat(12, 1), []int{fail, fail}),
			[]string{"1d20", "12d6", "1d20", "1d20"}, []string{"blue cold 12 full", "indigo restrained", "violet blinded"}},
		{"only violet left", "FR-078", "unhappy (boundary)", 6, otAttempt(), []int{pass}, []string{"1d20"}, []string{"violet none"}},
		{"no layer left", "FR-079", "unhappy (empty and minimal)", 7, otAttempt(), nil, []string{}, []string{}},
		{"ChooseToFail rolls no d20 but each damage", "FR-064", "unhappy (rule interaction)", 4, d20.ChooseToFail{}, slices.Concat(otRepeat(12, 1)),
			[]string{"12d6"}, []string{"blue cold 12 full", "indigo restrained", "violet blinded"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wall := luMust(combat.NewPrismaticWallStateAfter(c.destroyed))
			cmd := combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Wall: wall, Creature: combat.PrismaticSaveTarget{ID: "walker", Attempt: c.attempt}}
			res, requests := otResolve(t, c.faces, cmd)
			crossing := res.(combat.SpellTablePrismaticWallCrossed).Crossing()
			got := []string{}
			for _, l := range crossing.Layers() {
				s := l.Color().String()
				if l.Save() == nil {
					t.Errorf("layer %s has no save evidence", s)
				}
				switch v, ok := l.Strike(); x := v.(type) {
				case combat.PrismaticDamageStrike:
					half := "full"
					if x.Halved() {
						half = "half"
					}
					s += fmt.Sprintf(" %s %d %s", strings.ToLower(x.Damage().Type().String()), x.Damage().Amount(), half)
				case combat.PrismaticConditionStrike:
					s += " " + strings.ToLower(x.Condition().String())
				default:
					if ok {
						t.Errorf("strike = %#v", v)
					}
					s += " none"
				}
				got = append(got, s)
			}
			if !slices.Equal(requests, c.wantRequests) || !reflect.DeepEqual(got, c.want) || crossing.Creature != "walker" {
				t.Errorf("requests = %v, layers = %v, want %v and %v [%s%s]", requests, got, c.wantRequests, c.want, otSpec, c.requirement)
			}
			if got := len(c.want); got != len(crossing.Layers()) {
				t.Errorf("layers = %d, want %d", len(crossing.Layers()), got)
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-067, FR-068, FR-090, plan P10 - a Prismatic Spray target list with a repeated id, a zero id or no save attempt, and a wall creature with a zero id or no attempt, are invalid input before any die; 256 targets are accepted and 257 are refused with ErrSpellTableLimit; the command is not changed.
// Path: per case.
func TestResolvePrismaticInvalidCommands(t *testing.T) {
	t.Parallel()
	many := func(n int) []combat.PrismaticSaveTarget {
		out := make([]combat.PrismaticSaveTarget, n)
		for i := range out {
			out[i] = combat.PrismaticSaveTarget{ID: combat.CombatantID(fmt.Sprintf("t%03d", i)), Attempt: d20.AutomaticFailure{}}
		}
		return out
	}
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ResolveSpellOutcomeTableCommand
		limit                   bool
	}{
		{"a repeated target id", "FR-067", "unhappy (invalid input)", combat.RollPrismaticSprayCommand{SaveDC: otDC(), Targets: otTargets("a", "a")}, false},
		{"a zero target id", "FR-067", "unhappy (invalid input)", combat.RollPrismaticSprayCommand{SaveDC: otDC(), Targets: []combat.PrismaticSaveTarget{{Attempt: otAttempt()}}}, false},
		{"a target without an attempt", "FR-068", "unhappy (invalid input)", combat.RollPrismaticSprayCommand{SaveDC: otDC(), Targets: []combat.PrismaticSaveTarget{{ID: "a"}}}, false},
		{"257 targets", "FR-067", "unhappy (boundary)", combat.RollPrismaticSprayCommand{SaveDC: otDC(), Targets: many(257)}, true},
		{"a wall creature without an id", "FR-068", "unhappy (invalid input)", combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Creature: combat.PrismaticSaveTarget{Attempt: otAttempt()}}, false},
		{"a wall creature without an attempt", "FR-068", "unhappy (invalid input)", combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Creature: combat.PrismaticSaveTarget{ID: "a"}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewMinimumRoller()
			res, err := otUse(t, roller).Resolve(context.Background(), c.cmd)
			if res != nil || !errors.Is(err, srderr.ErrInvalidInput) || errors.Is(err, combat.ErrSpellTableLimit) != c.limit || len(roller.Requests()) != 0 {
				t.Errorf("Resolve = %v, %v with %d requests, want an invalid input (limit %v) and no die [%s%s]", res, err, len(roller.Requests()), c.limit, otSpec, c.requirement)
			}
		})
	}
	targets := many(256)
	res, err := otUse(t, dicetest.NewMinimumRoller()).Resolve(context.Background(), combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: targets})
	if err != nil || len(res.(combat.SpellTablePrismaticSprayRolled).Targets()) != 256 || targets[255].ID != "t255" {
		t.Errorf("256 targets = %v, %v, want 256 results (the cap, FR-067)", res, err)
	}
}

// otDC is the spell save DC of the Prismatic tests.
func otDC() d20.DifficultyClass { return 15 }

// Requirement: doc/specs/spell-outcome-tables.md FR-092, FR-064 - the Prismatic results own their slices: changing what Targets, Rays, Strikes or Layers returned changes nothing, and equal commands with equal roller sequences give equal results.
// Path: unhappy (immutability).
func TestPrismaticResultsAreCopies(t *testing.T) {
	t.Parallel()
	faces := slices.Concat([]int{8, 1, 6}, otRepeat(12, 3), []int{5})
	cmd := combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: otTargets("a")}
	first, _ := otResolve(t, faces, cmd)
	second, _ := otResolve(t, faces, cmd)
	if !reflect.DeepEqual(first, second) {
		t.Error("equal commands and roller sequences must give equal results")
	}
	got := first.(combat.SpellTablePrismaticSprayRolled)
	targets := got.Targets()
	targets[0].Rays()[0] = combat.PrismaticViolet
	targets[0].Strikes()[0] = nil
	targets[0] = combat.PrismaticTargetResult{}
	if !reflect.DeepEqual(first, second) || len(got.Targets()[0].Rays()) != 2 || got.Targets()[0].Strikes()[0] == nil {
		t.Error("a copy of the results must not change the result")
	}
	wall, _ := otResolve(t, slices.Concat([]int{15}, otRepeat(12, 1), []int{15, 15}), combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Wall: luMust(combat.NewPrismaticWallStateAfter(4)), Creature: combat.PrismaticSaveTarget{ID: "w", Attempt: otAttempt()}})
	crossing := wall.(combat.SpellTablePrismaticWallCrossed).Crossing()
	crossing.Layers()[0] = combat.PrismaticLayerCrossing{}
	if len(crossing.Layers()) != 3 || crossing.Layers()[0].Color() != combat.PrismaticBlue {
		t.Error("a copy of the layers must not change the crossing")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-004, FR-091 - exactly 100 Mishaps in a row are allowed: a 101st d100 that is not a Mishap gives a result holding 100 Mishaps, in contrast to the 101st Mishap, which stops the chain.
// Path: unhappy (boundary).
func TestResolveTeleportAtTheMishapCap(t *testing.T) {
	t.Parallel()
	var faces []int
	for range 100 {
		faces = append(faces, 1, 1, 1, 1) // a d100 Mishap, then the 3d10 of one object
	}
	faces = append(faces, 51) // False destination, 51 or more: not a Mishap
	res, requests := otResolve(t, faces, combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.FalseDestination})
	got := res.(combat.SpellTableTeleportRolled).Outcome()
	if len(got.Mishaps()) != 100 || got.Arrival() != combat.TeleportTableArrival(combat.ArrivesInSimilarArea{}) || len(requests) != 201 {
		t.Errorf("mishaps = %d, arrival = %#v, requests = %d, want 100, Similar Area, 201 [%sFR-004]", len(got.Mishaps()), got.Arrival(), len(requests), otSpec)
	}
}
