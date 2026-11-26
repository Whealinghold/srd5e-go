package combat_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const otSpec = "doc/specs/spell-outcome-tables.md "

func otIDs(names ...string) []combat.CombatantID {
	out := make([]combat.CombatantID, len(names))
	for i, n := range names {
		out[i] = combat.MustCombatantID(n)
	}
	return out
}

// Requirement: doc/specs/spell-outcome-tables.md FR-003, FR-010, FR-011, FR-012, table T - every row and boundary face of the Teleportation Outcome table: the first and last face of each band (T1 to T6), the no-roll rows giving On Target on 1 and 100, and the False destination giving only Mishap and Similar Area (SRD 5.2.1, "Teleportation Outcome", p. 168).
// Path: per case.
func TestTeleportKindForEveryBoundary(t *testing.T) {
	t.Parallel()
	const (
		mishap  = combat.TeleportMishapOutcome
		similar = combat.TeleportSimilarArea
		off     = combat.TeleportOffTarget
		on      = combat.TeleportOnTarget
	)
	type face struct {
		face int
		want combat.TeleportOutcomeKind
	}
	cases := []struct {
		name, requirement, path string
		familiarity             combat.TeleportFamiliarity
		faces                   []face
	}{
		{"T1 permanent circle", "FR-011", "unhappy (rule interaction)", combat.PermanentCircle, []face{{1, on}, {100, on}}},
		{"T2 linked object", "FR-011", "unhappy (rule interaction)", combat.LinkedObject, []face{{1, on}, {100, on}}},
		{"T3 very familiar", "FR-010", "unhappy (boundary)", combat.VeryFamiliar,
			[]face{{1, mishap}, {5, mishap}, {6, similar}, {13, similar}, {14, off}, {24, off}, {25, on}, {100, on}}},
		{"T4 seen casually", "FR-010", "unhappy (boundary)", combat.SeenCasually,
			[]face{{1, mishap}, {33, mishap}, {34, similar}, {43, similar}, {44, off}, {53, off}, {54, on}, {100, on}}},
		{"T5 viewed once or described", "FR-010", "unhappy (boundary)", combat.ViewedOnceOrDescribed,
			[]face{{1, mishap}, {43, mishap}, {44, similar}, {53, similar}, {54, off}, {73, off}, {74, on}, {100, on}}},
		{"T6 false destination", "FR-012", "unhappy (boundary)", combat.FalseDestination,
			[]face{{1, mishap}, {50, mishap}, {51, similar}, {100, similar}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, f := range c.faces {
				got, err := combat.TeleportKindFor(c.familiarity, f.face)
				if err != nil || got != f.want {
					t.Errorf("TeleportKindFor(%v, %d) = %v, %v, want %v [%s%s]", c.familiarity, f.face, got, err, f.want, otSpec, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-010, FR-002 - a face outside 1 to 100 is ErrFaceOutOfRange and a familiarity that is not a row is an invalid input; TeleportNeedsRoll is false for the two certain rows and for a value that is not a row.
// Path: unhappy (invalid input).
func TestTeleportKindForInvalidInput(t *testing.T) {
	t.Parallel()
	for _, face := range []int{0, -1, 101} {
		if _, err := combat.TeleportKindFor(combat.VeryFamiliar, face); !errors.Is(err, combat.ErrFaceOutOfRange) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("face %d = %v, want ErrFaceOutOfRange", face, err)
		}
	}
	if _, err := combat.TeleportKindFor(0, 50); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero familiarity = %v, want ErrInvalidInput", err)
	}
	for f, want := range map[combat.TeleportFamiliarity]bool{combat.PermanentCircle: false, combat.LinkedObject: false, combat.VeryFamiliar: true,
		combat.SeenCasually: true, combat.ViewedOnceOrDescribed: true, combat.FalseDestination: true, 0: false, 7: false} {
		if got := combat.TeleportNeedsRoll(f); got != want {
			t.Errorf("TeleportNeedsRoll(%v) = %v, want %v", f, got, want)
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-018 - NewTeleportingCreatures accepts 1 to 9 distinct ids ("you and up to eight") and rejects none, ten, a repeated id and a zero id with an error that matches srderr.ErrInvalidInput; the ids are copied on construction and on read.
// Path: per case.
func TestNewTeleportingCreatures(t *testing.T) {
	t.Parallel()
	nine := make([]string, 9)
	for i := range nine {
		nine[i] = fmt.Sprintf("c%d", i)
	}
	cases := []struct {
		name, requirement, path string
		ids                     []combat.CombatantID
		ok                      bool
	}{
		{"one creature", "FR-018", "unhappy (boundary)", otIDs("a"), true},
		{"nine creatures", "FR-018", "unhappy (boundary)", otIDs(nine...), true},
		{"ten creatures", "FR-018", "unhappy (boundary)", otIDs(append(nine, "c9")...), false},
		{"none", "FR-018", "unhappy (empty and minimal)", nil, false},
		{"an empty slice", "FR-018", "unhappy (empty and minimal)", []combat.CombatantID{}, false},
		{"a repeated id", "FR-018", "unhappy (invalid input)", otIDs("a", "b", "a"), false},
		{"a zero id", "FR-018", "unhappy (invalid input)", []combat.CombatantID{"a", ""}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewTeleportingCreatures(c.ids)
			if c.ok != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Fatalf("NewTeleportingCreatures(%v) = %v, want ok=%v [%s%s]", c.ids, err, c.ok, otSpec, c.requirement)
			}
			if c.ok && !slices.Equal(got.IDs(), c.ids) {
				t.Errorf("IDs() = %v, want %v", got.IDs(), c.ids)
			}
		})
	}
	ids := otIDs("a", "b")
	got := luMust(combat.NewTeleportingCreatures(ids))
	ids[0] = "changed"
	got.IDs()[1] = "changed"
	if !slices.Equal(got.IDs(), otIDs("a", "b")) {
		t.Errorf("IDs() = %v after the input and a read copy were changed, want a, b (FR-019)", got.IDs())
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-016 - NewArrivesOffTarget accepts 2 to 24 miles (the total of 2d12) and the eight directions of the d8 in order, and rejects 1 and 25 miles and a direction that is not one of them with an error that matches srderr.ErrInvalidInput; the d8 faces map to east, southeast, south, southwest, west, northwest, north and northeast (SRD 5.2.1, "Off Target", p. 168).
// Path: per case.
func TestNewArrivesOffTarget(t *testing.T) {
	t.Parallel()
	names := []string{"east", "southeast", "south", "southwest", "west", "northwest", "north", "northeast"}
	for i, want := range names {
		d := combat.CompassDirection(i + 1)
		if d.String() != want || !d.Valid() {
			t.Errorf("face %d = %q, want %q", i+1, d.String(), want)
		}
	}
	cases := []struct {
		name  string
		miles int
		dir   combat.CompassDirection
		ok    bool
	}{
		{"two miles east", 2, combat.East, true}, {"24 miles northeast", 24, combat.Northeast, true},
		{"one mile", 1, combat.East, false}, {"25 miles", 25, combat.East, false},
		{"no direction", 5, 0, false}, {"direction nine", 5, 9, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewArrivesOffTarget(c.miles, c.dir)
			if c.ok != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Fatalf("NewArrivesOffTarget(%d, %v) = %v, want ok=%v [%sFR-016]", c.miles, c.dir, err, c.ok, otSpec)
			}
			if c.ok && (got.Miles() != c.miles || got.Direction() != c.dir) {
				t.Errorf("arrival = %d miles %v", got.Miles(), got.Direction())
			}
		})
	}
	if combat.CompassDirection(0).String() != "unknown" || !combat.North.IsCardinal() || combat.Northeast.IsCardinal() {
		t.Error("an invalid direction is unknown, north is cardinal and northeast is not")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-003 - the names of the familiarity rows, the table outcomes and the directions are the closed values of section 7.1, and a value outside a set is "unknown".
// Path: per case.
func TestSpellOutcomeEnumNames(t *testing.T) {
	t.Parallel()
	got := []string{combat.PermanentCircle.String(), combat.LinkedObject.String(), combat.VeryFamiliar.String(), combat.SeenCasually.String(),
		combat.ViewedOnceOrDescribed.String(), combat.FalseDestination.String(), combat.TeleportMishapOutcome.String(), combat.TeleportSimilarArea.String(),
		combat.TeleportOffTarget.String(), combat.TeleportOnTarget.String(), combat.TeleportFamiliarity(0).String(), combat.TeleportOutcomeKind(9).String()}
	want := []string{"permanent_circle", "linked_object", "very_familiar", "seen_casually", "viewed_once_or_described", "false_destination",
		"mishap", "similar_area", "off_target", "on_target", "unknown", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}
