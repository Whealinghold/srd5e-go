package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SquareTerrain is what lies on one square (SRD 5.2.1, "Corners", p. 13; "Difficult Terrain", p. 14). The zero
// value is SquareTerrainOpen.
type SquareTerrain int

// The three kinds of square.
const (
	// SquareTerrainOpen is ordinary ground.
	SquareTerrainOpen SquareTerrain = iota
	// SquareTerrainDifficult is Difficult Terrain.
	SquareTerrainDifficult
	// SquareTerrainFilled is a square a wall, a large tree or another terrain feature fills (SRD 5.2.1,
	// "Occupied Space", p. 185).
	SquareTerrainFilled
)

// String returns "open", "difficult", "filled" or "unknown".
func (t SquareTerrain) String() string {
	if t < SquareTerrainOpen || t > SquareTerrainFilled {
		return "unknown"
	}
	return []string{"open", "difficult", "filled"}[t]
}

// GridTerrain is the terrain of a board: the Difficult Terrain squares and the filled squares. Every other square
// is open, because the grid has no edge. The zero GridTerrain is all open. Immutable; safe for concurrent use.
type GridTerrain struct{ squares map[GridSquare]SquareTerrain }

// NewGridTerrain returns the terrain with the given Difficult Terrain and filled squares, copied. A square in
// both lists is filled. It returns a LimitExceededError (matching srderr.ErrInvalidInput) when the two lists hold
// more than 65536 squares together (plan P10). It never panics.
func NewGridTerrain(difficult, filled []GridSquare) (GridTerrain, error) {
	if err := overCap("terrain squares", len(difficult)+len(filled), maxTerrainSquares); err != nil {
		return GridTerrain{}, err
	}
	m := make(map[GridSquare]SquareTerrain, len(difficult)+len(filled))
	for _, s := range difficult {
		m[s] = SquareTerrainDifficult
	}
	for _, s := range filled {
		m[s] = SquareTerrainFilled
	}
	return GridTerrain{m}, nil
}

// MustGridTerrain is NewGridTerrain for lists known to be within the cap. It panics with the error NewGridTerrain
// returns, and only then.
func MustGridTerrain(difficult, filled []GridSquare) GridTerrain {
	t, err := NewGridTerrain(difficult, filled)
	if err != nil {
		panic(err)
	}
	return t
}

// TerrainOf returns the terrain of square: filled when it is filled, else difficult when it is Difficult Terrain,
// else open.
func (t GridTerrain) TerrainOf(square GridSquare) SquareTerrain { return t.squares[square] }

// GridCreature is a creature on the grid: its id, its Creature Size, the space it occupies and whether it can act
// (SRD 5.2.1, "Creature Size", "Moving around Other Creatures", p. 14). The zero GridCreature is invalid
// (IsZero). Immutable; comparable with ==. Safe for concurrent use.
type GridCreature struct {
	id           CombatantID
	size         content.CreatureSize
	footprint    GridFootprint
	availability ConditionActionAvailability
}

// NewGridCreature returns the creature. It returns an error matching srderr.ErrInvalidInput for a blank id, an
// invalid size or availability, or a footprint that is not the square GridFootprintForCreature gives for size
// (its width and height both the side of the size). It never panics.
func NewGridCreature(id CombatantID, size content.CreatureSize, footprint GridFootprint, availability ConditionActionAvailability) (GridCreature, error) {
	_, idErr := NewCombatantID(string(id))
	want, sizeErr := GridFootprintForCreature(size, footprint.Origin())
	if idErr != nil || sizeErr != nil || footprint != want ||
		(availability != ConditionActionAvailabilityAvailable && availability != ConditionActionAvailabilityNone) {
		return GridCreature{}, invalidInput("a grid creature needs an id, an availability and the footprint of its size")
	}
	return GridCreature{id, size, footprint, availability}, nil
}

// IsZero reports whether c is the invalid zero creature.
func (c GridCreature) IsZero() bool { return c.id.IsZero() }

// IsFor reports whether the creature has the id. A host that needs the id keeps the one it built the creature
// with; combat never returns a CombatantID (combat-foundations.md FR-009). The zero id is never matched.
func (c GridCreature) IsFor(id CombatantID) bool { return !id.IsZero() && c.id == id }

// Size returns the Creature Size.
func (c GridCreature) Size() content.CreatureSize { return c.size }

// Footprint returns the space the creature occupies.
func (c GridCreature) Footprint() GridFootprint { return c.footprint }

// Availability returns whether the creature can act.
func (c GridCreature) Availability() ConditionActionAvailability { return c.availability }

// WithFootprint returns the creature moved to footprint, which must be the square of its size (as in
// NewGridCreature, whose error it returns).
func (c GridCreature) WithFootprint(footprint GridFootprint) (GridCreature, error) {
	return NewGridCreature(c.id, c.size, footprint, c.availability)
}

// GridBoard is the map of one fight: its terrain and the creatures on it, in insertion order. It does not reject
// overlapping footprints, because Tiny creatures share a square and a creature can end a turn in another's space
// (SRD 5.2.1, "Moving around Other Creatures", p. 14). The zero GridBoard is the empty board of open terrain.
// Immutable: every operation returns a new board. Safe for concurrent use.
type GridBoard struct {
	terrain   GridTerrain
	creatures []GridCreature
}

// NewGridBoard returns the board with terrain and creatures (copied). It returns an error matching
// srderr.ErrInvalidInput for a zero creature or two creatures with the same id. More than 256
// creatures return a LimitExceededError (plan P10). It never panics.
func NewGridBoard(terrain GridTerrain, creatures []GridCreature) (GridBoard, error) {
	if err := overCap("grid creatures", len(creatures), maxGridCreatures); err != nil {
		return GridBoard{}, err
	}
	seen := make(map[CombatantID]bool, len(creatures))
	for _, c := range creatures {
		if c.IsZero() || seen[c.id] {
			return GridBoard{}, invalidInput("a board needs distinct valid creatures")
		}
		seen[c.id] = true
	}
	return GridBoard{terrain, slices.Clone(creatures)}, nil
}

// Terrain returns the terrain of the board.
func (b GridBoard) Terrain() GridTerrain { return b.terrain }

// Creatures returns a copy of the creatures in insertion order.
func (b GridBoard) Creatures() []GridCreature { return slices.Clone(b.creatures) }

// Place returns a board with creature added, or replacing the creature with its id in the same position. It
// returns an error matching srderr.ErrInvalidInput for the zero creature, and a LimitExceededError for a new creature on a board
// already holding 256 (plan P10); replacing one by id at the cap is accepted.
func (b GridBoard) Place(creature GridCreature) (GridBoard, error) {
	if creature.IsZero() {
		return GridBoard{}, invalidInput("a board places a valid creature")
	}
	out := slices.Clone(b.creatures)
	if i := slices.IndexFunc(out, func(c GridCreature) bool { return c.id == creature.id }); i >= 0 {
		out[i] = creature
	} else {
		if len(out) >= maxGridCreatures {
			return GridBoard{}, LimitExceededError{Collection: "grid creatures", Max: maxGridCreatures}
		}
		out = append(out, creature)
	}
	return GridBoard{b.terrain, out}, nil
}

// Remove returns a board without the creature with the given id. It returns an error matching
// srderr.ErrInvalidInput when no such creature is on the board.
func (b GridBoard) Remove(id CombatantID) (GridBoard, error) {
	i := slices.IndexFunc(b.creatures, func(c GridCreature) bool { return c.id == id })
	if i < 0 {
		return GridBoard{}, invalidInput("the creature is not on the board")
	}
	return GridBoard{b.terrain, slices.Delete(slices.Clone(b.creatures), i, i+1)}, nil
}
