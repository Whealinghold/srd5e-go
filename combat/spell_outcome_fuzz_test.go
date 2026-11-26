package combat_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-018 and testing.md section 10 - NewTeleportingCreatures never panics on any list of ids and either accepts 1 to 9 distinct non-empty ids or returns an invalid input.
// Path: unhappy (invalid input).
func FuzzNewTeleportingCreatures(f *testing.F) {
	for _, s := range []string{"", "a", "a,b", "a,a", "a,,b", "a,b,c,d,e,f,g,h,i", "a,b,c,d,e,f,g,h,i,j"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		var ids []combat.CombatantID
		for _, p := range strings.Split(in, ",") {
			ids = append(ids, combat.CombatantID(p))
		}
		got, err := combat.NewTeleportingCreatures(ids)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if n := len(got.IDs()); n < 1 || n > 9 {
			t.Fatalf("accepted %d ids", n)
		}
	})
}

// Requirement: doc/specs/spell-outcome-tables.md FR-071, FR-075, FR-016, FR-031 and testing.md section 10 - the constructors over host numbers (NewIndigoSaveTrack, NewPrismaticWallStateAfter, NewArrivesOffTarget, NewConfusionMovesRandomly) never panic and accept only values in their range, and a track or wall they build can be driven on without a panic.
// Path: unhappy (invalid input).
func FuzzSpellOutcomeNumbers(f *testing.F) {
	for _, n := range []int{0, 1, 2, 3, 7, 8, 24, 25, -1, math.MaxInt} {
		f.Add(n, n)
	}
	f.Fuzz(func(t *testing.T, a, b int) {
		if tr, err := combat.NewIndigoSaveTrack(a, b); err == nil {
			if a < 0 || a > 2 || b < 0 || b > 2 {
				t.Fatalf("accepted track %d %d", a, b)
			}
			if tr.Record(d20.Success) == nil || tr.Record(d20.Failure) == nil {
				t.Fatal("Record gave no step")
			}
		}
		if w, err := combat.NewPrismaticWallStateAfter(a); err == nil {
			if a < 0 || a > 7 || len(w.Layers()) != 7-a {
				t.Fatalf("accepted wall %d", a)
			}
			w.Destroy(combat.PrismaticColor(b), combat.StrongWind{})
		} else if !errors.Is(err, combat.ErrWallDestroyedCount) {
			t.Fatalf("wall error %v", err)
		}
		if _, err := combat.NewArrivesOffTarget(a, combat.CompassDirection(b)); err == nil && (a < 2 || a > 24 || b < 1 || b > 8) {
			t.Fatalf("accepted arrival %d %d", a, b)
		}
		if _, err := combat.NewConfusionMovesRandomly(combat.CompassDirection(a)); err == nil && !combat.CompassDirection(a).IsCardinal() {
			t.Fatalf("accepted direction %d", a)
		}
	})
}

// Requirement: doc/specs/spell-outcome-tables.md FR-002, FR-010, FR-030, FR-044, FR-050, FR-060 and testing.md section 10 - every table lookup never panics, accepts exactly the faces of its die and returns ErrFaceOutOfRange for the rest.
// Path: unhappy (boundary).
func FuzzSpellOutcomeTableLookups(f *testing.F) {
	for _, n := range []int{0, 1, 6, 7, 8, 10, 11, 100, 101, -5} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, face int) {
		check := func(name string, sides int, err error) {
			if inside := face >= 1 && face <= sides; inside != (err == nil) || (!inside && !errors.Is(err, combat.ErrFaceOutOfRange)) {
				t.Fatalf("%s(%d) error = %v", name, face, err)
			}
		}
		_, err := combat.TeleportKindFor(combat.SeenCasually, face)
		check("TeleportKindFor", 100, err)
		_, err = combat.ConfusionRowFor(face)
		check("ConfusionRowFor", 10, err)
		_, err = combat.BlinkVanishesOn(face)
		check("BlinkVanishesOn", 6, err)
		_, _, err = combat.ReincarnationSpeciesFor(face)
		check("ReincarnationSpeciesFor", 10, err)
		_, _, err = combat.PrismaticRayFor(face)
		check("PrismaticRayFor", 8, err)
	})
}
