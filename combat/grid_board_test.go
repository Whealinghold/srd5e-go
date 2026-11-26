package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/grid-positions-and-movement.md FR-008 (SRD 5.2.1, "Corners", "Difficult Terrain", p. 13, p. 14; "Occupied Space", p. 185) - filled beats difficult, difficult beats open, and a square in neither set is open.
// Path: happy.
func TestGridTerrainPrecedence(t *testing.T) {
	t.Parallel()
	terrain := combat.MustGridTerrain(squares(t, 1, 1, 2, 2), squares(t, 2, 2, 3, 3))
	for _, c := range []struct {
		s    combat.GridSquare
		want combat.SquareTerrain
	}{
		{sq(t, 1, 1), combat.SquareTerrainDifficult},
		{sq(t, 2, 2), combat.SquareTerrainFilled},
		{sq(t, 3, 3), combat.SquareTerrainFilled},
		{sq(t, 0, 0), combat.SquareTerrainOpen},
		{sq(t, 1_000_000, -1_000_000), combat.SquareTerrainOpen},
	} {
		if got := terrain.TerrainOf(c.s); got != c.want {
			t.Errorf("TerrainOf(%v) = %v, want %v", c.s, got, c.want)
		}
	}
	if (combat.GridTerrain{}).TerrainOf(sq(t, 1, 1)) != combat.SquareTerrainOpen {
		t.Error("the zero terrain is not all open")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-008 - the two sets are copied into the terrain: later changes to the arguments do not show.
// Path: unhappy (immutability).
func TestGridTerrainCopiesItsArguments(t *testing.T) {
	t.Parallel()
	difficult, filled := squares(t, 1, 1), squares(t, 2, 2)
	terrain := combat.MustGridTerrain(difficult, filled)
	difficult[0], filled[0] = sq(t, 7, 7), sq(t, 8, 8)
	if terrain.TerrainOf(sq(t, 1, 1)) != combat.SquareTerrainDifficult || terrain.TerrainOf(sq(t, 2, 2)) != combat.SquareTerrainFilled ||
		terrain.TerrainOf(sq(t, 7, 7)) != combat.SquareTerrainOpen {
		t.Error("the terrain follows its argument slices")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-008 - SquareTerrain names its three values.
// Path: happy.
func TestSquareTerrainString(t *testing.T) {
	t.Parallel()
	for v, want := range map[combat.SquareTerrain]string{combat.SquareTerrainOpen: "open", combat.SquareTerrainDifficult: "difficult", combat.SquareTerrainFilled: "filled", -1: "unknown", 3: "unknown"} {
		if v.String() != want {
			t.Errorf("SquareTerrain(%d).String() = %q, want %q", int(v), v.String(), want)
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-006 (SRD 5.2.1, "Creature Size", p. 14) - NewGridCreature accepts exactly the square of the size and keeps what it was given.
// Path: happy.
func TestNewGridCreature(t *testing.T) {
	t.Parallel()
	f := fp(t, 3, 4, 2, 2)
	c, err := combat.NewGridCreature(combat.MustCombatantID("ogre"), content.SizeLarge, f, combat.ConditionActionAvailabilityNone)
	if err != nil || c.IsZero() || !c.IsFor("ogre") || c.IsFor("") || c.IsFor("goblin") || c.Size() != content.SizeLarge || c.Footprint() != f || c.Availability() != combat.ConditionActionAvailabilityNone {
		t.Errorf("NewGridCreature = %#v, %v", c, err)
	}
	if !(combat.GridCreature{}).IsZero() {
		t.Error("the zero creature is valid")
	}
}

// Requirement: per case
// Path: per case.
func TestNewGridCreatureRejects(t *testing.T) {
	t.Parallel()
	const fr = "doc/specs/grid-positions-and-movement.md FR-006"
	id := combat.MustCombatantID("c")
	one, two := fp(t, 0, 0, 1, 1), fp(t, 0, 0, 2, 2)
	avail := combat.ConditionActionAvailabilityAvailable
	cases := []struct {
		name, requirement, path string
		id                      combat.CombatantID
		size                    content.CreatureSize
		footprint               combat.GridFootprint
		availability            combat.ConditionActionAvailability
	}{
		{"Large on one square", fr, "unhappy (invalid input)", id, content.SizeLarge, one, avail},
		{"Medium on four squares", fr, "unhappy (invalid input)", id, content.SizeMedium, two, avail},
		{"Large on a 2 by 1 rectangle", fr, "unhappy (invalid input)", id, content.SizeLarge, fp(t, 0, 0, 2, 1), avail},
		{"the zero footprint", fr, "unhappy (invalid input)", id, content.SizeMedium, combat.GridFootprint{}, avail},
		{"an invalid size", fr, "unhappy (invalid input)", id, 0, one, avail},
		{"a blank id", fr, "unhappy (invalid input)", "", content.SizeMedium, one, avail},
		{"a whitespace id", fr, "unhappy (invalid input)", " ", content.SizeMedium, one, avail},
		{"the zero availability", fr, "unhappy (invalid input)", id, content.SizeMedium, one, 0},
		{"an availability past the last", fr, "unhappy (invalid input)", id, content.SizeMedium, one, combat.ConditionActionAvailabilityNone + 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewGridCreature(c.id, c.size, c.footprint, c.availability)
			if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
				t.Errorf("NewGridCreature = %v, %v, want ErrInvalidInput [%s]", got, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-006, FR-009 - WithFootprint moves a creature to another footprint of its size and refuses any other.
// Path: unhappy (invalid input).
func TestGridCreatureWithFootprint(t *testing.T) {
	t.Parallel()
	c := gridCreature(t, "ogre", content.SizeLarge, 0, 0)
	moved, err := c.WithFootprint(fp(t, 5, 5, 2, 2))
	if err != nil || moved.Footprint() != fp(t, 5, 5, 2, 2) || !moved.IsFor("ogre") || c.Footprint() != fp(t, 0, 0, 2, 2) {
		t.Errorf("WithFootprint = %v, %v (original %v)", moved, err, c.Footprint())
	}
	if _, err := c.WithFootprint(fp(t, 5, 5, 1, 1)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("WithFootprint(wrong size) = %v", err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-007 (SRD 5.2.1, Creature Size and Space table, "Moving around Other Creatures", p. 14) - a board accepts overlapping footprints: Tiny creatures share a square and a creature can end a turn in another's space.
// Path: unhappy (rule interaction).
func TestGridBoardAcceptsOverlappingFootprints(t *testing.T) {
	t.Parallel()
	a, b := gridCreature(t, "a", content.SizeTiny, 0, 0), gridCreature(t, "b", content.SizeTiny, 0, 0)
	big := gridCreature(t, "big", content.SizeLarge, 0, 0)
	got, err := combat.NewGridBoard(combat.GridTerrain{}, []combat.GridCreature{a, b, big})
	if err != nil || len(got.Creatures()) != 3 {
		t.Errorf("NewGridBoard(overlapping) = %v, %v", got, err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-009 - a board refuses a duplicate id or a zero creature, copies its creatures in and out, keeps insertion order, and an empty or nil list gives an empty board.
// Path: unhappy (state error).
func TestNewGridBoard(t *testing.T) {
	t.Parallel()
	a, b := gridCreature(t, "a", content.SizeMedium, 0, 0), gridCreature(t, "b", content.SizeMedium, 1, 0)
	if _, err := combat.NewGridBoard(combat.GridTerrain{}, []combat.GridCreature{a, b, gridCreature(t, "a", content.SizeMedium, 5, 5)}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a duplicate id: %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewGridBoard(combat.GridTerrain{}, []combat.GridCreature{a, {}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero creature: %v, want ErrInvalidInput", err)
	}
	for _, in := range [][]combat.GridCreature{nil, {}} {
		if got, err := combat.NewGridBoard(combat.GridTerrain{}, in); err != nil || len(got.Creatures()) != 0 {
			t.Errorf("NewGridBoard(%v) = %v, %v, want the empty board", in, got, err)
		}
	}
	in := []combat.GridCreature{a, b}
	board := gridBoard(t, combat.GridTerrain{}, in...)
	in[0] = b
	out := board.Creatures()
	if !slices.Equal(out, []combat.GridCreature{a, b}) {
		t.Fatalf("Creatures() = %v, want insertion order a, b and a copy of the argument", out)
	}
	out[1] = a
	if !slices.Equal(board.Creatures(), []combat.GridCreature{a, b}) {
		t.Error("Creatures() returned the stored slice")
	}
	if board.Terrain().TerrainOf(sq(t, 0, 0)) != combat.SquareTerrainOpen {
		t.Error("Terrain() did not return the terrain")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-009, FR-032 - Place adds a creature at the end or replaces the one with its id in place; the receiver is unchanged; the zero creature is refused.
// Path: unhappy (state error).
func TestGridBoardPlace(t *testing.T) {
	t.Parallel()
	a, b := gridCreature(t, "a", content.SizeMedium, 0, 0), gridCreature(t, "b", content.SizeMedium, 1, 0)
	board := gridBoard(t, combat.GridTerrain{}, a, b)
	c := gridCreature(t, "c", content.SizeMedium, 2, 0)
	added, err := board.Place(c)
	if err != nil || !slices.Equal(added.Creatures(), []combat.GridCreature{a, b, c}) {
		t.Errorf("Place(new) = %v, %v", added.Creatures(), err)
	}
	moved := gridCreature(t, "a", content.SizeMedium, 9, 9)
	replaced, err := board.Place(moved)
	if err != nil || !slices.Equal(replaced.Creatures(), []combat.GridCreature{moved, b}) {
		t.Errorf("Place(existing) = %v, %v, want a replaced in its position", replaced.Creatures(), err)
	}
	if !slices.Equal(board.Creatures(), []combat.GridCreature{a, b}) {
		t.Error("Place changed the receiver")
	}
	if _, err := board.Place(combat.GridCreature{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("Place(zero) = %v, want ErrInvalidInput", err)
	}
	empty, err := (combat.GridBoard{}).Place(a)
	if err != nil || len(empty.Creatures()) != 1 {
		t.Errorf("Place on the zero board = %v, %v", empty.Creatures(), err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-009, D-9 - Remove returns a board without the creature, refuses an id that is not on the board, and leaves the receiver unchanged.
// Path: unhappy (state error).
func TestGridBoardRemove(t *testing.T) {
	t.Parallel()
	a, b, c := gridCreature(t, "a", content.SizeMedium, 0, 0), gridCreature(t, "b", content.SizeMedium, 1, 0), gridCreature(t, "c", content.SizeMedium, 2, 0)
	board := gridBoard(t, combat.GridTerrain{}, a, b, c)
	got, err := board.Remove("b")
	if err != nil || !slices.Equal(got.Creatures(), []combat.GridCreature{a, c}) {
		t.Errorf("Remove(b) = %v, %v", got.Creatures(), err)
	}
	if !slices.Equal(board.Creatures(), []combat.GridCreature{a, b, c}) {
		t.Error("Remove changed the receiver")
	}
	if got, err := board.Remove("z"); !errors.Is(err, srderr.ErrInvalidInput) || len(got.Creatures()) != 0 {
		t.Errorf("Remove(absent) = %v, %v, want ErrInvalidInput", got.Creatures(), err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-009 - Place replaces the creature with its id wherever it stands in the order, and Remove takes out exactly the creature with that id, the first, a later one and one that sorts before others.
// Path: unhappy (boundary).
func TestGridBoardPlaceAndRemoveByIDAtAnyPosition(t *testing.T) {
	t.Parallel()
	c, a, b := gridCreature(t, "c", content.SizeMedium, 0, 0), gridCreature(t, "a", content.SizeMedium, 1, 0), gridCreature(t, "b", content.SizeMedium, 2, 0)
	board := gridBoard(t, combat.GridTerrain{}, c, a, b)
	moved := gridCreature(t, "a", content.SizeMedium, 9, 9)
	if got, err := board.Place(moved); err != nil || !slices.Equal(got.Creatures(), []combat.GridCreature{c, moved, b}) {
		t.Errorf("Place(a) = %v, %v, want a replaced in the middle", got.Creatures(), err)
	}
	last := gridCreature(t, "b", content.SizeMedium, 9, 9)
	if got, err := board.Place(last); err != nil || !slices.Equal(got.Creatures(), []combat.GridCreature{c, a, last}) {
		t.Errorf("Place(b) = %v, %v, want b replaced at the end", got.Creatures(), err)
	}
	for id, want := range map[combat.CombatantID][]combat.GridCreature{"c": {a, b}, "a": {c, b}, "b": {c, a}} {
		if got, err := board.Remove(id); err != nil || !slices.Equal(got.Creatures(), want) {
			t.Errorf("Remove(%s) = %v, %v, want %v", id, got.Creatures(), err, want)
		}
	}
}
