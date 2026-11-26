package combat

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const outcomeSpec = "doc/specs/spell-outcome-tables.md "

// Requirement: doc/specs/spell-outcome-tables.md FR-001 - validate reports a gap, an overlap, a first band that does not start at 1, a last band that does not end at the last face and an empty table, each as an error that matches srderr.ErrInvalidInput; a table that covers every face once is valid.
// Path: per case.
func TestOutcomeTableValidate(t *testing.T) {
	t.Parallel()
	b := func(low, high int) outcomeBand[int] { return outcomeBand[int]{low, high, 0} }
	cases := []struct {
		name, requirement, path string
		bands                   []outcomeBand[int]
		wantErr                 bool
	}{
		{"every face once", "FR-001", "happy", []outcomeBand[int]{b(1, 3), b(4, 6)}, false},
		{"one band over the whole die", "FR-001", "happy", []outcomeBand[int]{b(1, 6)}, false},
		{"a gap", "FR-001", "unhappy (invalid input)", []outcomeBand[int]{b(1, 2), b(4, 6)}, true},
		{"an overlap", "FR-001", "unhappy (invalid input)", []outcomeBand[int]{b(1, 4), b(4, 6)}, true},
		{"the first band starts at 2", "FR-001", "unhappy (invalid input)", []outcomeBand[int]{b(2, 6)}, true},
		{"the last band ends one short", "FR-001", "unhappy (boundary)", []outcomeBand[int]{b(1, 3), b(4, 5)}, true},
		{"the last band ends one past", "FR-001", "unhappy (boundary)", []outcomeBand[int]{b(1, 3), b(4, 7)}, true},
		{"a band that ends before it starts", "FR-001", "unhappy (invalid input)", []outcomeBand[int]{b(1, 3), b(4, 3), b(4, 6)}, true},
		{"no band", "FR-001", "unhappy (empty and minimal)", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := newOutcomeTable(dice.D6(), c.bands).validate()
			if (err != nil) != c.wantErr || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("validate() = %v, want error %v [%s%s]", err, c.wantErr, outcomeSpec, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-001, FR-003 - every printed table, built fresh by its function, covers each face of its die exactly once (SRD 5.2.1, "Teleport", p. 168; "Confusion", p. 117; "Prismatic Rays", pp. 154-155; "Reincarnate", p. 158; "Blink", p. 113).
// Path: happy.
func TestEveryPrintedTableCoversItsDie(t *testing.T) {
	t.Parallel()
	for f := PermanentCircle; f <= FalseDestination; f++ {
		table, ok := teleportTable(f)
		if !ok || table.validate() != nil {
			t.Errorf("teleport table of %v is not valid", f)
		}
	}
	checks := map[string]error{
		"confusion":   confusionTable().validate(),
		"reincarnate": reincarnateTable().validate(),
		"prismatic":   prismaticTable().validate(),
		"blink":       blinkTable().validate(),
	}
	for name, err := range checks {
		if err != nil {
			t.Errorf("%s table: %v", name, err)
		}
	}
	if _, ok := teleportTable(0); ok {
		t.Error("a familiarity that is not a row must have no table")
	}
	if a, b := confusionTable(), confusionTable(); &a.bands[0] == &b.bands[0] {
		t.Error("each call must build a fresh table (FR-003)")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-002 - lookup returns the value at both ends of every band, and ErrFaceOutOfRange (which is an invalid input) for 0, a negative face and sides plus one; a face no band holds in a broken table is an invalid input.
// Path: per case.
func TestOutcomeTableLookup(t *testing.T) {
	t.Parallel()
	table := newOutcomeTable(dice.D6(), []outcomeBand[string]{{1, 3, "low"}, {4, 6, "high"}})
	cases := []struct {
		name, requirement, path string
		face                    int
		want                    string
		wantErr                 error
	}{
		{"first face", "FR-002", "unhappy (boundary)", 1, "low", nil},
		{"top of the first band", "FR-002", "unhappy (boundary)", 3, "low", nil},
		{"bottom of the second band", "FR-002", "unhappy (boundary)", 4, "high", nil},
		{"last face", "FR-002", "unhappy (boundary)", 6, "high", nil},
		{"zero", "FR-002", "unhappy (boundary)", 0, "", ErrFaceOutOfRange},
		{"negative", "FR-002", "unhappy (invalid input)", -1, "", ErrFaceOutOfRange},
		{"sides plus one", "FR-002", "unhappy (boundary)", 7, "", ErrFaceOutOfRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := table.lookup(c.face)
			if got != c.want || !errors.Is(err, c.wantErr) {
				t.Errorf("lookup(%d) = %q, %v, want %q, %v [%s%s]", c.face, got, err, c.want, c.wantErr, outcomeSpec, c.requirement)
			}
			if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("lookup(%d) error does not match ErrInvalidInput", c.face)
			}
		})
	}
	broken := newOutcomeTable(dice.D6(), []outcomeBand[string]{{1, 2, "low"}})
	if _, err := broken.lookup(5); !errors.Is(err, srderr.ErrInvalidInput) || errors.Is(err, ErrFaceOutOfRange) {
		t.Errorf("a face no band holds = %v, want an invalid input that is not ErrFaceOutOfRange", err)
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-077 - a layer that is not a colour is never the outermost one, and an unknown destroyer destroys nothing: the unexported fit check reports the wrong destroyer for a nil destroyer on every colour.
// Path: unhappy (invalid input).
func TestDestroyerFitsRejectsNil(t *testing.T) {
	t.Parallel()
	for c := PrismaticRed; c <= PrismaticViolet; c++ {
		if got := destroyerFits(c, nil); got != LayerWrongDestroyer {
			t.Errorf("destroyerFits(%v, nil) = %v, want LayerWrongDestroyer", c, got)
		}
	}
}
