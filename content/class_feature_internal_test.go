package content

import (
	"errors"
	"slices"
	"testing"
)

// Requirement: content-classes.md FR-058, FR-060 - a FeatureOption holding a
// cost or a fact that the constructor would reject (only reachable inside the
// package) is invalid when a feature entry checks it: the entry is refused with
// BlockInvalidMember.
// Path: per case.
func TestFeatureEntryRejectsInvalidOptionMembers(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		o    FeatureOption
	}{
		{"negative cost", FeatureOption{id: "a", name: "A", text: "t", pointCost: -1}},
		{"invalid fact", FeatureOption{id: "a", name: "A", text: "t", facts: []FeatureFact{GrantsFeat{}}}},
		{"fact that is not a spell grant", FeatureOption{id: "a", name: "A", text: "t", facts: []FeatureFact{GrantsSubclass{}}}},
	} {
		_, err := NewFeatureEntry("c_f", "F", FeatureEntryFields{Level: 1, Text: "t", Options: []FeatureOption{c.o}})
		var be *BlockError
		if !errors.As(err, &be) || be.Problem != BlockInvalidMember {
			t.Errorf("%s: err = %v, want BlockInvalidMember [unhappy (invalid input)]", c.name, err)
		}
	}
}

// Requirement: content-classes.md FR-058 - the constructors copy their input
// slices, so changing the caller's slice afterwards does not change the grant.
// Path: NewSpellGrant, NewAlwaysPreparedSpells.
func TestSpellGrantCopiesInput(t *testing.T) {
	t.Parallel()
	ids := []SpellID{"a", "b"}
	g, err := NewSpellGrant(1, ids)
	if err != nil {
		t.Fatal(err)
	}
	ids[0] = "z"
	if !slices.Equal(g.spellIDs, []SpellID{"a", "b"}) {
		t.Errorf("spell ids = %v, want the copy [unhappy (immutability)]", g.spellIDs)
	}
	grants := []SpellGrant{g}
	a, err := NewAlwaysPreparedSpells(grants)
	if err != nil {
		t.Fatal(err)
	}
	grants[0] = SpellGrant{}
	if a.grants[0].level != 1 {
		t.Errorf("grants[0] = %v, want the copy [unhappy (immutability)]", a.grants[0])
	}
}

// Requirement: content-classes.md FR-051 - a table row whose cell types differ
// from the first row's is reported and left out of the rows; a header of width 0
// admits no row of any length, while an unknown header (-1) admits any.
// Path: per case.
func TestDecodeTableRowsDropsDeviatingRows(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		json  string
		width int
		rows  int
		probs int
	}{
		{"deviating row dropped", `[["a","b"],["c",1],["d","e"]]`, 2, 2, 1},
		{"width zero rejects a cell", `[["x"]]`, 0, 1, 1},
		{"unknown width admits any", `[["x"],["y","z"]]`, -1, 2, 0},
	} {
		n, ok := parseDocument([]byte(c.json))
		if !ok {
			t.Fatalf("%s: bad fixture", c.name)
		}
		d := &decoder{}
		rows := decodeTableRows(d, n, "/t", c.width)
		if len(rows) != c.rows || len(d.probs.items) != c.probs {
			t.Errorf("%s: rows = %d, problems = %d, want %d and %d [unhappy (boundary)]", c.name, len(rows), len(d.probs.items), c.rows, c.probs)
		}
	}
}

// Requirement: content-classes.md FR-040 - a ColumnNone fits a column of every
// valid kind and of no invalid kind.
// Path: per case.
func TestColumnNoneFitsOnlyValidKinds(t *testing.T) {
	t.Parallel()
	for _, k := range []ClassColumnKind{ColumnKindCount, ColumnKindDice, ColumnKindBonus, ColumnKindFeetBonus} {
		if !(ColumnNone{}).fits(k) {
			t.Errorf("ColumnNone does not fit %v [happy]", k)
		}
	}
	for _, k := range []ClassColumnKind{0, -1, ColumnKindFeetBonus + 1} {
		if (ColumnNone{}).fits(k) {
			t.Errorf("ColumnNone fits invalid kind %d [unhappy (invalid input)]", int(k))
		}
	}
}
