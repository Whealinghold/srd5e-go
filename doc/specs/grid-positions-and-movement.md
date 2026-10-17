# Grid positions and movement

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md`, `initiative-and-actions.md`, `content-schema-and-decoder.md`, `rule-options.md` |

## 1. Purpose and scope

The optional "Playing on a Grid" rules (SRD 5.2.1 p. 13) as an **opt-in value
layer**: where creatures and objects stand on a square grid, the distance
between any two of them, what a one-square step costs and whether it is allowed
(entering squares, Difficult Terrain, diagonals, corners), and the space each
Creature Size occupies (1, 1, 1, 4, 9, 16 squares). It produces the plain feet
that `attacks.md`, `hiding-searching-and-senses.md` and
`mounted-and-underwater-combat.md` already take, and the `MoveSegment` list that
`initiative-and-actions.md` already judges. It is the first reader of
`CombatRuleOptions.Grid` (`combat-foundations.md` FR-042 to FR-045,
`rule-options.md` FR-029).

Everything is a pure function over immutable values (a board is passed in,
never held by a global or by `Encounter`). Without a grid (`NoGrid`, the
default) nothing here is reachable and the engine stays in feet.

**Out of scope:**
- Charging the movement, passing through creatures, ending a move in an
  occupied space and Speed budgets: `initiative-and-actions.md` FR-077 to FR-089,
  FR-115. This spec only builds the segments those rules judge.
- Area-of-effect shapes (Cone, Cube, Cylinder, Emanation, Line, Sphere, pp. 179 to
  188), Cover, line of sight and light: the SRD gives no
  square-counting rule for them, so `magical-effects.md` (host facts),
  `attacks.md` and `hiding-searching-and-senses.md` keep taking host facts
  (D-6).
- Elevation and three-dimensional distance (flying, falling, jumping), walls
  along square edges, map bounds, pathfinding and "squares I can reach"
  searches: a host concern (D-6).
- Mounted creatures sharing a space (`mounted-and-underwater-combat.md`),
  grappled-creature dragging (`grappling.md`), forced movement and Teleportation
  (the host calls `GridBoard.Place`).
- Holding positions in `encounter-aggregate.md`: it keeps positions out of scope
  (its section 1) and this spec does not change that (D-1).
- Hex grids and any other layout: the SRD describes only a square grid.

## 2. Source

- SRD 5.2.1, "Playing on a Grid" (Squares, Speed, Entering a Square, Corners,
  Ranges), p. 13; "Creature Size" and the Creature Size and Space table,
  "Moving around Other Creatures", "Difficult Terrain", p. 14; "Reach",
  "Opportunity Attacks", "Ranged Attacks in Close Combat", p. 15.
- "Rules Glossary": "Difficult Terrain", p. 181; "Occupied Space", "Opportunity
  Attacks", p. 185; "Reach", p. 186; "Round Down", p. 187; "Size", p. 188;
  "Unoccupied Space", p. 191.
- Package design: `doc/rules/architecture.md` §3 (optional rules, purity,
  immutability), §6; `shared-types-and-ownership.md` FR-005, FR-053, FR-057;
  `content-schema-and-decoder.md` FR-139 (the size table is a fixed fact of
  `CreatureSize`).
- Web research informed the judgement only; no text is used. The SRD states no
  rule for a multi-square mover on Difficult Terrain or corners, and the
  square-counting conventions are package-design choices, not SRD rules; they
  are the interpretations in D-4 and D-5 (see also D-6).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Playing on a Grid, square grid | `SquareGrid` (`combat-foundations.md` FR-043) | Required by every operation (FR-001). |
| square | `GridSquare` (built by `NewGridSquare(column, row int) (GridSquare, error)`) | Whole-number coordinates; row grows southward. Package type. |
| orthogonally / diagonally adjacent | `GridDirection` (eight) | p. 13. |
| space (of a creature or object) | `GridFootprint` | A rectangle of squares: origin (lowest column and row), width, height. |
| Creature Size and Space | `GridFootprint.ForCreature(CreatureSize, origin)` | p. 14; side from `CreatureSize.Space` (content FR-139), not redefined here. |
| creature on the grid | `GridCreature` (built by `NewGridCreature(id, size, footprint, availability)`) | `availability` is `ConditionActionAvailability` (`conditions.md`); "Incapacitated" matters for passing through (p. 14). |
| Difficult Terrain, "terrain feature that fills its space" | `SquareTerrain` (`open`, `difficult`, `filled`), `GridTerrain` | A wall or large tree fills its squares (p. 13); an object completely filling a space makes it occupied (p. 185). |
| the map of one fight | `GridBoard` (built by `NewGridBoard(terrain, creatures)`) | Package term. |
| Ranges (count of squares) | `GridRules.RangeInSquares`, `RangeInFeet`, `IsWithinFeet` | p. 13. |
| Entering a Square | `GridRules.AssessStep` -> closed `GridStepAssessment` (`GridStepAllowed`, `GridStepBlocked` with `GridStepBlockReason`: `enteringFilledSquare`, `cuttingCorner`) | p. 13. |
| a move square by square | `GridRules.PlanPath` -> closed `GridPathPlan` (`GridPathPlanned`, `GridPathBlocked`) | Builds `MoveSegment`s. |
| leaves your reach | `GridRules.LeavesReach` | p. 15, p. 185. |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber. Test references use `doc/specs/grid-positions-and-movement.md FR-nnn`.

### 4.1 Grid, squares, spaces and the board

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | Every `GridRules` operation MUST take a `SquareGrid` argument and have no `NoGrid` form, so code without a grid cannot reach grid rules. A host obtains it by an exhaustive `switch` on `CombatRuleOptions.Grid`. This is the FR that reads `CombatRuleOptions.Grid` (`rule-options.md` FR-029). One square is 5 feet (`SquareGrid.FeetPerSquare`, `combat-foundations.md` FR-044); this spec redefines neither the constant nor `speedInSquares`. | SRD 5.2.1, "Playing on a Grid", "Squares", "Speed", p. 13 | happy |
| FR-002 | `NewGridSquare(column, row)` MUST accept integers from -1,000,000 to 1,000,000 inclusive and returns an `srderr.ErrInvalidInput` error outside that range. `GridSquare` MUST be a comparable value (usable with `==` and as a map key); `NewGridSquare(column, row int) (GridSquare, error)` is the constructor. | architecture.md §3, §6 (the bound keeps every sum within a 32-bit `int` on every platform) | unhappy (boundary) |
| FR-003 | `GridDirection` MUST have exactly eight values with offsets (column, row): north (0,-1), northEast (1,-1), east (1,0), southEast (1,1), south (0,1), southWest (-1,1), west (-1,0), northWest (-1,-1). `IsDiagonal` MUST be true for exactly the four with both offsets non-zero. | SRD 5.2.1, "Entering a Square" ("orthogonally or diagonally adjacent"), p. 13 | happy |
| FR-004 | `NewGridFootprint(origin, width, height)` MUST require width and height from 1 to 1,000 and every square it covers to be a valid `GridSquare`, otherwise an `srderr.ErrInvalidInput` error. It MUST expose `Squares()` (row by row from the origin, a fresh copy on each call), `Contains(square)`, `Overlaps(other)` and `Shifted(direction)` (the same rectangle one square on; an `srderr.ErrInvalidInput` error if a square leaves the FR-002 range), and compare by value. | SRD 5.2.1, "Creature Size", p. 14; architecture.md §3 | unhappy (boundary) |
| FR-005 | `GridFootprint.ForCreature(size, origin)` MUST be square, its side taken from `CreatureSize.Space` (content FR-139): `SharedSquareSpace` (Tiny) side 1; `WholeSquareSpace.SquaresPerSide` otherwise. Cases 4.1a. | SRD 5.2.1, Creature Size and Space table, p. 14 | happy |
| FR-006 | `NewGridCreature(id, size, footprint, availability)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when `footprint` is not the square `ForCreature(size, ...)` gives (width and height both equal to the size's side). | SRD 5.2.1, "Creature Size", p. 14 | unhappy (invalid input) |
| FR-007 | `GridBoard` MUST NOT reject overlapping footprints: Tiny creatures share a square with another creature, and a creature can "somehow end a turn" in another's space. | SRD 5.2.1, Creature Size and Space table, "Moving around Other Creatures", p. 14 | unhappy (rule interaction) |
| FR-008 | `NewGridTerrain(difficult, filled)` MUST hold two sets of `GridSquare` copied into the terrain value (later changes to the arguments do not show) (later changes to the arguments do not show). `TerrainOf(square)` MUST return `filled` when the square is in `filled` (even if also in `difficult`), else `difficult` when in `difficult`, else `open`. A square in neither set is `open`: the grid has no edge. | SRD 5.2.1, "Corners", "Difficult Terrain", p. 13, p. 14; "Occupied Space", p. 185 | happy |
| FR-009 | `NewGridBoard(terrain, creatures)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for two creatures with the same `CombatantID`. `Place(creature)` MUST return a new board with the creature added, or replaced when its id is present (position order kept); `Remove(id)` MUST return a new board without it and returns an `srderr.ErrInvalidInput` error for an id not on the board. `Creatures()` returns a copy in insertion order; no operation changes the receiver. | architecture.md §3 (immutability), §6 | unhappy (state error) |

#### Cases 4.1a: footprint from size (one test per row; origin (0,0))

| Label | Size | Width x height | Squares covered | Source | Path |
|---|---|---|---|---|---|
| Tiny shares a square | Tiny | 1 x 1 | 1 | p. 14 | unhappy (boundary) |
| Small | Small | 1 x 1 | 1 | p. 14 | happy |
| Medium | Medium | 1 x 1 | 1 | p. 14 | happy |
| Large | Large | 2 x 2 | 4 | p. 14 | happy |
| Huge | Huge | 3 x 3 | 9 | p. 14 | happy |
| Gargantuan | Gargantuan | 4 x 4 | 16 | p. 14 | happy |

### 4.2 Ranges

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `RangeInSquares(grid, a, b)` MUST equal the number of squares counted from a square adjacent to `a` to, and including, the first square in `b`'s space, by the shortest route, a diagonal step counting 1. For two rectangles it is `max(dx, dy)` where `dx` is `max(a.minColumn - b.maxColumn, b.minColumn - a.maxColumn, 0)` and `dy` likewise for rows. Cases 4.2a. | SRD 5.2.1, "Playing on a Grid", "Ranges", p. 13 | happy |
| FR-011 | Overlapping or identical spaces MUST give 0; the result MUST be the same with the arguments swapped. | SRD 5.2.1, "Ranges", p. 13; Creature Size and Space table, p. 14 (shared squares) | unhappy (boundary) |
| FR-012 | `RangeInFeet(grid, a, b)` MUST equal `RangeInSquares` times `grid.feetPerSquare`. `IsWithinFeet(grid, a, b, feet)` MUST be true iff `RangeInFeet <= feet` (exactly equal counts), and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for negative `feet`. A `feet` that is not a multiple of 5 behaves as its multiple of 5 rounded down. Cases 4.2b. | SRD 5.2.1, "Ranges", p. 13; "Round Down", p. 187 | unhappy (boundary) |
| FR-013 | Range MUST ignore walls, filled squares, corners and creatures between the two spaces: it measures, it does not move (interpretation: "shortest route" is the count of steps with no obstacle test, because corner blocking is stated only for movement under "Corners"; whether something blocks an attack is Cover). | SRD 5.2.1, "Ranges" and "Corners", p. 13; "Cover", p. 15 | unhappy (rule interaction) |
| FR-014 | Range MUST accept any two footprints: creatures, objects of any rectangle (width and height up to FR-004) and a location (a 1 x 1 footprint). | SRD 5.2.1, "Ranges" ("whether creatures or objects"), p. 13; "Choose a Target" ("a creature, an object, or a location"), p. 15 | happy |
| FR-015 | `DistancesFeetFrom(grid, board, subject)` MUST return a fresh map from the id of every creature on the board other than `subject.id` to `RangeInFeet` from the subject's footprint, in board order; an empty map when nobody else is on the board. It MUST NOT filter by enemy, sight or Incapacitated: the host maps ids to `attacks.md`'s nearby-enemy facts. | SRD 5.2.1, "Ranged Attacks in Close Combat", p. 15 | unhappy (empty case) |
| FR-016 | `LeavesReach(grid, observer, before, after, reachFeet)` MUST be true iff `before` is within `reachFeet` of `observer` and `after` is not (FR-012). `reachFeet` below 5 or negative MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. Whether the observer can see the creature and has its Reaction is the caller's. | SRD 5.2.1, "Reach", "Making an Opportunity Attack", p. 15; "Reach", p. 186 | unhappy (boundary) |

#### Cases 4.2a: range in squares (one test per row; footprints written as origin and size in squares)

| Label | A | B | Squares | Feet | Source | Path |
|---|---|---|---|---|---|---|
| orthogonally adjacent | (0,0) 1x1 | (1,0) 1x1 | 1 | 5 | p. 13 | happy |
| diagonally adjacent | (0,0) 1x1 | (1,1) 1x1 | 1 | 5 | p. 13 | unhappy (boundary) |
| the count of FR-010: two squares between | (0,0) 1x1 | (3,0) 1x1 | 3 | 15 | p. 13 | happy |
| offset in both axes | (0,0) 1x1 | (5,2) 1x1 | 5 | 25 | p. 13 | happy |
| negative coordinates | (-2,-2) 1x1 | (1,1) 1x1 | 3 | 15 | p. 13 | unhappy (boundary) |
| same square | (4,4) 1x1 | (4,4) 1x1 | 0 | 0 | p. 13; p. 14 | unhappy (boundary) |
| Large beside Medium | (0,0) 2x2 | (2,0) 1x1 | 1 | 5 | p. 13; p. 14 | happy |
| Large, Medium one square off | (0,0) 2x2 | (3,0) 1x1 | 2 | 10 | p. 13 | happy |
| Large, Medium touching a corner | (0,0) 2x2 | (2,2) 1x1 | 1 | 5 | p. 13 | unhappy (boundary) |
| Large to Large, two squares between | (0,0) 2x2 | (4,0) 2x2 | 3 | 15 | p. 13 | happy |
| Medium inside a Large space | (0,0) 2x2 | (1,1) 1x1 | 0 | 0 | p. 14 | unhappy (boundary) |
| Gargantuan to a far Medium | (0,0) 4x4 | (10,0) 1x1 | 7 | 35 | p. 13 | happy |
| object 2x1 (a door) to a location | (0,0) 2x1 | (5,0) 1x1 | 4 | 20 | p. 13; p. 15 | happy |

#### Cases 4.2b: `IsWithinFeet` and `LeavesReach` (Medium at (0,0); one test per row)

| Label | Other | Feet or reach | Expected | Source | Path |
|---|---|---|---|---|---|
| at exactly 5 feet | (1,0) | within 5 | true | p. 15 | unhappy (boundary) |
| 10 feet away, within 5 | (2,0) | within 5 | false | p. 15 | unhappy (boundary) |
| 7 feet rounds down to 5 | (1,0) and (2,0) | within 7 | true; false | p. 187 | unhappy (boundary) |
| zero feet, same square | (0,0) | within 0 | true | p. 13 | unhappy (boundary) |
| leaves a 5-foot reach | before (1,0), after (2,0) | reach 5 | true | p. 15 | happy |
| stays inside (circles) | before (1,0), after (0,1) | reach 5 | false | p. 15 | unhappy (boundary) |
| starts outside | before (2,0), after (3,0) | reach 5 | false | p. 15 | unhappy (boundary) |
| 10-foot reach, moves from 10 to 15 | before (2,0), after (3,0) | reach 10 | true | p. 15 | happy |
| reach 4 | any | 4 | an `srderr.ErrInvalidInput` error | FR-016 | unhappy (invalid input) |

### 4.3 Entering squares

A step moves a mover's footprint one square. Its **entered squares** are the
squares of the shifted footprint not in the old one (one row or column for an
orthogonal step; `width + height - 1` squares for a diagonal one).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | A step in any of the eight directions MUST be one square of movement, a segment of exactly 5 feet: a diagonal step costs the same as an orthogonal one. A step shifts the whole footprint, whatever the size. | SRD 5.2.1, "Entering a Square" ("1 square of movement ... orthogonally or diagonally adjacent"), p. 13 | happy |
| FR-021 | A step that enters any `filled` square MUST give `GridStepBlocked(enteringFilledSquare)`. | SRD 5.2.1, "Corners" ("a wall, a large tree, or another terrain feature that fills its space"), p. 13; "Occupied Space", p. 185 | unhappy (rule interaction) |
| FR-022 | A diagonal step MUST give `GridStepBlocked(cuttingCorner)` when no entered square is `filled` but a square entered by either of its two component orthogonal steps (the same footprint shifted by the column offset alone, or by the row offset alone) is `filled`. FR-021 takes precedence. Cases 4.3a. | SRD 5.2.1, "Corners", p. 13 | unhappy (boundary) |
| FR-023 | An orthogonal step MUST NOT be blocked by a filled square it does not enter. Only a terrain feature blocks a corner: a creature beside the corner does not. | SRD 5.2.1, "Corners", p. 13 | unhappy (rule interaction) |
| FR-024 | The declared terrain of a `GridStepAllowed` MUST be `Terrain.Difficult` iff any entered square is `difficult`, else `Terrain.Open`, however many entered squares are difficult. The segment distance stays 5 feet, so `initiative-and-actions.md` FR-077 charges 2 squares (10 feet). A square left behind or merely crossed at a corner does not count. | SRD 5.2.1, "Entering a Square" ("costs 2 squares"), p. 13; "Difficult Terrain", p. 14 ("even if multiple things in a space count as Difficult Terrain"), p. 181 ("isn't cumulative"). Multi-square mover: interpretation, D-4. A creature's space making a square Difficult Terrain is applied by `initiative-and-actions.md` FR-079 from the occupants list (FR-025), not by `terrain` | unhappy (rule interaction) |
| FR-025 | `GridStepAllowed.Occupants` MUST list, once each and in board order, an `Occupant(size, relation, availability)` for every creature on the board other than the mover (by id) whose footprint overlaps an entered square. `relation` is `ally` iff the creature's id is in the supplied `allies` set, else `other`. A creature overlapping only squares the mover keeps is not listed. Passing through or refusing them is `initiative-and-actions.md` FR-079, FR-080. | SRD 5.2.1, "Moving around Other Creatures", p. 14; "Difficult Terrain", p. 181 | happy |
| FR-026 | `GridStepAllowed.End` MUST be `SegmentEnd.OccupiedByCreature` iff the destination footprint overlaps any other creature's footprint (entered or kept squares), else `SegmentEnd.Free`, except that it MUST be `free` when the mover and every overlapping creature are Tiny (interpretation, D-13). The 4-per-square capacity is the host's. | SRD 5.2.1, "Moving around Other Creatures" ("can't willingly end a move in a space occupied"), p. 14; Creature Size and Space table (Tiny: 4 per square), p. 14; "Unoccupied Space", p. 191 | unhappy (state error) |

#### Cases 4.3a: one step (one test per row; board terrain and creatures as stated; Medium mover at (5,5) unless noted)

| Label | Step | Board | Expected | Source | Path |
|---|---|---|---|---|---|
| open, orthogonal | east | empty | allowed; open; no occupants; free | FR-020 | happy |
| open, diagonal | northEast | empty | allowed; open; free; 5 feet | FR-020 | happy |
| difficult entered square | east | (6,5) difficult | allowed; difficult | FR-024 | happy |
| filled entered square | east | (6,5) filled | blocked `enteringFilledSquare` | FR-021 | unhappy (rule interaction) |
| diagonal, destination filled | southEast | (6,6) filled | blocked `enteringFilledSquare` | FR-021 | unhappy (rule interaction) |
| diagonal, east neighbour filled | southEast | (6,5) filled | blocked `cuttingCorner` | FR-022 | unhappy (boundary) |
| diagonal, south neighbour filled | southEast | (5,6) filled | blocked `cuttingCorner` | FR-022 | unhappy (boundary) |
| diagonal, both neighbours filled | southEast | (6,5), (5,6) filled | blocked `cuttingCorner` | FR-022 | unhappy (boundary) |
| diagonal, neighbour only difficult | southEast | (6,5) difficult | allowed; open | FR-024 | unhappy (boundary) |
| orthogonal beside a filled square | east | (6,6) filled | allowed | FR-023 | unhappy (boundary) |
| creature beside the corner | southEast | ally at (6,5) | allowed; occupants none | FR-023 | unhappy (rule interaction) |
| ally in the entered square | east | Medium ally at (6,5) | allowed; occupants [Medium ally]; occupiedByCreature | FR-025, FR-026 | happy |
| Tiny enemy in the square | east | Tiny non-ally at (6,5) | occupants [Tiny other]; occupiedByCreature | FR-025 | unhappy (boundary) |
| Incapacitated creature | east | non-ally, `ConditionActionAvailability.None` at (6,5) | occupants carry that availability | FR-025 | unhappy (rule interaction) |
| the mover's own board entry | east | mover also on the board at (5,5) | occupants none | FR-025 | unhappy (state error) |
| same creature on two entered squares of a Large mover | Large at (5,5), east | Large foe at (7,5) | one occupant | FR-025 | unhappy (boundary) |
| Large mover, difficult in one entered square | Large at (5,5), east | (7,6) difficult | allowed; difficult | FR-024 | unhappy (rule interaction) |
| Large mover, filled square in an entered square | Large at (5,5), southEast | (7,7) filled | blocked `enteringFilledSquare` | FR-021 | unhappy (rule interaction) |
| Large mover, filled square beside the corner (east) | Large at (5,5), southEast | (7,5) filled (entered by the east component step only) | blocked `cuttingCorner` | FR-022 | unhappy (rule interaction) |
| Large mover, filled square beside the corner (south) | Large at (5,5), southEast | (5,7) filled (entered by the south component step only) | blocked `cuttingCorner` | FR-022 | unhappy (rule interaction) |
| Tiny mover into a Tiny creature's square | Tiny at (5,5), east | Tiny non-ally at (6,5) | allowed; occupants [Tiny other]; free | FR-026 | unhappy (boundary) |
| Large mover, creature in a kept square | Large at (5,5), east | Tiny at (6,5) | occupants none; occupiedByCreature | FR-025, FR-026 | unhappy (boundary) |
| Gargantuan, 4 squares entered | Gargantuan at (5,5), east | (9,5) .. (9,8) difficult | allowed; difficult | FR-024 | unhappy (boundary) |

### 4.4 Planning a move

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | `PlanPath(grid, board, mover, directions, allies, mode, speedUse, extraCostPerFoot)` MUST apply FR-020 to FR-026 to each direction in order, the mover's footprint following each allowed step, and return `GridPathPlanned(finalFootprint, segments)` with one `MoveSegment` per step: distance 5, the given `MovementMode`, `SpeedUse` and `extraCostPerFoot`, and the step's terrain, occupants and end. At the first blocked step it MUST return `GridPathBlocked(stepIndex, reason)` and no segments. An empty `directions` list MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | SRD 5.2.1, "Playing on a Grid", "Speed", "Entering a Square", p. 13 | happy |
| FR-029 | `PlanPath` MUST NOT judge creatures that cannot be passed through, ending in an occupied space or whether the Speed suffices; those refusals are `TurnTracker.Move`'s (`initiative-and-actions.md` FR-080, FR-081, FR-086), so a plan with an enemy square is returned as planned. A plan's cost in feet MUST be 5 times its cost in squares, so the tracker accepts it iff its squares fit in `speedInSquares` of the movement left. Cases 4.4a. | SRD 5.2.1, "Speed", p. 13; "Moving around Other Creatures", p. 14 | unhappy (rule interaction) |
| FR-030 | A negative `extraCostPerFoot` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (the `MoveSegment` rule, `initiative-and-actions.md` FR-087); a positive one makes each square cost that many extra squares ("other effects might make a square cost even more"). | SRD 5.2.1, "Entering a Square", p. 13 | unhappy (invalid input) |
| FR-031 | The grid is unbounded: `PlanPath` never refuses a step for leaving a map; a host bounds the play area with `filled` squares. A step whose squares leave the FR-002 range MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | architecture.md §6; D-6 | unhappy (boundary) |
| FR-032 | `AssessStep`, `PlanPath` and every other operation MUST NOT change the board, the terrain or the mover; the host applies a plan with `board.Place(mover with finalFootprint)`. Equal inputs MUST give equal outputs. | architecture.md §3 (purity, immutability) | unhappy (immutability) |

#### Cases 4.4a: planned paths against the tracker (Walk, `walkingSpeed`, `extraCostPerFoot` 0)

| Label | Path from (0,0) Medium | Terrain | Plan | Total cost | Source | Path |
|---|---|---|---|---|---|---|
| six open steps (Speed 30) | east x6 | none | planned, 6 segments | 6 squares, 30 ft | p. 13 | happy |
| seven open steps | east x7 | none | planned, 7 segments (tracker refuses the 7th at Speed 30) | 7 squares | FR-029 | unhappy (boundary) |
| diagonal run | southEast x3 | none | planned; 3 segments | 3 squares | p. 13 | happy |
| one difficult square | east x3 | (2,0) difficult | planned; second segment difficult | 4 squares, 20 ft | p. 13 | unhappy (rule interaction) |
| Speed 32 (rounds down) | east x6 | none | planned; the tracker accepts 6 (30 of 32) and refuses a 7th | 6 squares | p. 13; p. 187 | unhappy (boundary) |
| blocked second step (index 1) | east x4 | (2,0) filled | `GridPathBlocked(1, enteringFilledSquare)`; no segments | none | FR-028 | unhappy (rule interaction) |
| extra cost 1 per foot | east x2 | none | planned; each segment `extraCostPerFoot` 1 | 4 squares, 20 ft | p. 13 | unhappy (rule interaction) |
| enemy in path | east x2 | non-ally Medium at (1,0) | planned; segment 1 lists it; the tracker refuses `blockedByCreature` | n/a | FR-029 | unhappy (rule interaction) |

### 4.5 Failure classification

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | A blocked step or path is an expected outcome and MUST be a returned closed result (`GridStepBlocked`, `GridPathBlocked`), never an error or a panic. Invalid construction (FR-002, FR-004, FR-006, FR-009, FR-012, FR-016, FR-028, FR-030, FR-031) MUST be a returned error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (`MustX` helpers panic with it). No operation reads a port, returns a `*srderr.PortError`, takes a `context.Context` or is asynchronous. | architecture.md §3, §6 | unhappy (invalid input) |

### Withdrawn ids

None withdrawn. Gaps in the numbering (FR-017 to FR-019, FR-027, FR-033 to FR-039) are intentional group spacing: reserved, never used.

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

- **Value objects** (immutable, by value, in package `combat` (files `grid_*.go`)): `GridSquare` (FR-002), `GridDirection` (FR-003), `GridFootprint` (FR-004, FR-005), `GridCreature` (FR-006), `SquareTerrain`, `GridTerrain` (FR-008), `GridBoard` (FR-007, FR-009), closed `GridStepAssessment` (`GridStepAllowed(footprint, terrain, occupants, end)` | `GridStepBlocked(reason)`), `GridStepBlockReason`, closed `GridPathPlan` (`GridPathPlanned(finalFootprint, segments)` | `GridPathBlocked(stepIndex, reason)`). Imported, not defined: `SquareGrid`, `CombatantID`, `CombatRuleOptions` (`combat-foundations.md`), `CreatureSize`, `CreatureSpace` (`content`), `ConditionActionAvailability` (`conditions.md`), `MoveSegment`, `Occupant`, `OccupantRelation`, `Terrain`, `SegmentEnd`, `MovementMode`, `SpeedUse` (`initiative-and-actions.md`).
- **Domain service** (stateless: a zero-size `GridRules` struct with value-receiver methods, or equivalent package functions): `RangeInSquares`, `RangeInFeet`, `IsWithinFeet`, `DistancesFeetFrom`, `LeavesReach`, `AssessStep`, `PlanPath`.
- **No aggregate, no event, no use case**: positions are host state; a plan is applied by the host with `GridBoard.Place`. The tracker's own events (`MovementSpent`) report the move.
- **Invariants:** a square in range, a footprint inside the square range with sides 1 to 1,000, a creature's footprint matching its size, unique ids on a board: the constructors. Blocked rules: `GridRules`.
- **Failures:** returned closed results for blocked steps and paths; a returned `srderr.ErrInvalidInput` error from `NewX` for the invalid values of FR-040.

## 6. Interface

All synchronous and pure; no port, no new `*srderr.PortError`. Package `combat` exports `GridSquare`, `GridDirection`, `GridFootprint`, `GridCreature`, `SquareTerrain`, `GridTerrain`, `GridBoard`, `GridRules`, `GridStepAssessment`, `GridStepAllowed`, `GridStepBlocked`, `GridStepBlockReason`, `GridPathPlan`, `GridPathPlanned`, `GridPathBlocked` (documented; no generic variant names, `architecture.md` §4). Exporting this spec's FR-001 is also the change that makes `CombatRuleOptions` and `PlayingOnAGrid` exported (`combat-foundations.md` section 4.6, `rule-options.md` FR-029). Typical flow: `switch (options.grid) { SquareGrid g => GridRules.PlanPath(g, ...), NoGrid() => /* feet only */ }`, then `TurnTracker.Move(id, plan.segments)`, then `board.Place(...)`. Distances for `attacks.md`, `hiding-searching-and-senses.md` and `mounted-and-underwater-combat.md` come from `RangeInFeet` or `DistancesFeetFrom`.

## 7. Observability

Silent by design: domain only (`doc/rules/observability.md` §3) and no srd5e use case runs these rules, so there is no span, metric or log (the same position as `hiding-searching-and-senses.md` section 7). The seven questions:
1. What can go wrong: a host passes a wrong position, ally set or terrain; a plan that is later refused by the tracker; a coordinate outside the range (a bug).
2. How an operator knows: a programmer error is an `srderr.ErrInvalidInput` error returned to (or a `MustX` panic in) the host's code; a blocked step names its reason in the result; the tracker's refusal and `MovementSpent` event are recorded by `Move` (`initiative-and-actions.md` section 7).
3. To diagnose: the result carries the step index and reason (`GridPathBlocked`), the segments carry terrain, occupants and end.
4. Counted or timed: nothing; the functions are constant or linear in the footprint size.
5. Existing instrumentation: unchanged; `srd5e.combat.*` signals of `Move` are unaffected.
6. Monitor: the `Move` refusal rates already catalogued (a rise in `blockedByCreature` or `endsInOccupiedSpace` after adopting the grid points at wrong ally sets).
7. Noise: none added.

No entry in `doc/observability/signals.md`.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 | unit, architecture | Every `GridRules` method signature takes `SquareGrid`; a scenario shows `NoGrid` takes the feet path; package `combat` exports `CombatRuleOptions`. |
| FR-002, FR-003, FR-004 | unit | Boundaries -1,000,000 / 1,000,000 / one past; eight direction offsets and `IsDiagonal`; footprint sizes 0, 1, 1,000, 1,001; shifting out of range; `Squares` order and unmodifiable; equality. |
| FR-005, FR-006 | unit | Table 4.1a (six sizes, square counts 1, 1, 1, 4, 9, 16); mismatched footprint returns an error. |
| FR-007 to FR-009 | unit | Overlapping footprints accepted; filled beats difficult; unlisted square open; argument sets copied; duplicate id returns an error; `Place` add and replace; `Remove` and absent id; receiver unchanged. |
| FR-010 to FR-014 | unit | Table 4.2a, swapped arguments, `IsWithinFeet` boundaries and negative feet, ignores obstacles (a wall between does not change range), objects and locations. |
| FR-015, FR-016 | unit | Empty board, subject excluded, board order, fresh map; table 4.2b including reach 4 returning an error. |
| FR-020 to FR-026 | unit | Table 4.3a, every row. |
| FR-028 to FR-032 | unit, scenario | Table 4.4a; empty directions returns an error; extra cost copied into segments; out-of-range returns an error; board unchanged; determinism. Scenario (public API): plan with 6 open steps and one difficult square at Speed 30 against `TurnTracker.Move`: accepted at 6 squares of cost, refused `insufficientMovement` beyond; Speed 32 case; enemy in the path refused `blockedByCreature`. |
| FR-040 | unit | Blocked results are returned, not errors; each invalid construction returns an `srderr.ErrInvalidInput` error. |

- Levels: unit (`combat/grid_*_test.go`, package `combat_test`, table-driven), one scenario (`scenarios/`, package `scenarios_test`), architecture (`internal/archtest` import allow-list; exported-name scan on the new names), a fuzz test for `NewGridFootprint` and `GridRules.PlanPath` inputs, and `Example` functions for `PlanPath`. No port, so no double or contract suite.
- Mutation testing (`doc/rules/testing.md` §8): the `combat` domain files are already on the path list (`gomu` runs only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`); the new `grid_*.go` files are in it. The range formula (`max`, the `0` clamp, the off-by-one of "adjacent = 1"), the corner test (either neighbour) and the any-entered-square difficult test are the likely survivors; table rows 4.2a and 4.3a are chosen to kill them.
- Every FR id above appears in the table; none is deliberately uncovered.

## 9. Open questions and decisions

No question is open and none is escalated. Settled, one line each (SRD text cited; "interpretation" marks a reading the SRD does not state; web material informed judgement only and nothing was copied):
- **D-1** Positions are modelled, as an opt-in value layer outside `Encounter`, because the SRD makes the grid optional and `NoGrid` stays the default (`rule-options.md` FR-029; `encounter-aggregate.md` section 1 stays true). Held by the host; applied by `GridBoard.Place`.
- **D-2** A diagonal step costs one square: the text says "1 square of movement to enter an unoccupied square adjacent to your space (orthogonally or diagonally)" (p. 13). No alternating 1-2-1 variant exists in the SRD.
- **D-3** (interpretation) Range counts squares from one adjacent to the first thing to the first square of the other, by the shortest route: for rectangles `max(dx, dy)`. It makes adjacent things 1 square (5 feet), matching the 5-foot reach (p. 15) and the SRD's own "within 5 feet" uses, and it ignores obstacles because only movement is stated to be blocked at corners.
- **D-4** (interpretation) A multi-square mover's step is Difficult Terrain if any square it newly enters is, once: p. 181 asks whether the entered "space" contains the cause and says it "isn't cumulative", and keeping the segment at 5 feet reuses `initiative-and-actions.md` FR-077 unchanged. A wider mover therefore meets Difficult Terrain more often, which is the usual table reading; a host preferring another rule edits the terrain it supplies.
- **D-5** (interpretation) "Cross the corner" is generalised to: no square entered by the diagonal step or by either of its component orthogonal steps may be filled. For a one-square mover that is the classic neighbour test. For a footprint wider than 1 the component-step squares are NOT all entered by the diagonal step (a Large mover at (5,5) stepping southEast enters (7,6), (7,7), (6,7), while the east component also enters (7,5)), so FR-022's separate component test applies to every size. Creatures never block a corner (the text names terrain features).
- **D-6** Area shapes, Cover, line of sight, light, elevation, edge walls, map bounds and reachability search are not modelled: the SRD gives no square-counting rule for them (pp. 15, 179 to 188), and inventing one would put non-SRD content in the package. They stay host facts; a future spec may add a pathfinder over `AssessStep`.
- **D-7** Movement stays in feet in the tracker; the grid converts only the total, so Dash with Speed 33 (66 feet, 13 squares) follows `initiative-and-actions.md` FR-084, not a per-part rounding. All step costs are multiples of 5, so the tracker's feet test equals the SRD's squares test (FR-029).
- **D-8** Coordinates are bounded to +/-1,000,000 and footprints to 1,000 per side so arithmetic cannot overflow a 32-bit `int`; no real map is near this.
- **D-9** `Remove` of an absent id returns an `srderr.ErrInvalidInput` error (programmer error) rather than silently doing nothing, consistent with `Place`'s strictness on duplicates.
- **D-10** Whether Difficult Terrain applies to a flying mover is not decided here: the plan reports the square's terrain and `initiative-and-actions.md` FR-077 governs the cost (interpretation left there).
- **D-11** Phase 4 for the types (the `combat` phase, `shared-types-and-ownership.md` FR-014); the release manifest marks the spec `deferred` for 0.1.0 (low priority; no earlier spec depends on it). Rank 26 in `shared-types-and-ownership.md` section 4.11, above `initiative-and-actions` (25). `attacks`, `hiding-searching-and-senses` and `mounted-and-underwater-combat` (ranks 23 to 24) consume host-passed feet, not a dependency on this spec.
- **D-13** (interpretation) Tiny creatures may share a square: the size table gives 4 Tiny per square (p. 14), so a Tiny mover ending among only Tiny creatures is not "a space occupied by another creature" in the refused sense; any larger mover or occupant keeps `occupiedByCreature`. Without this FR-007's sharing would be unreachable by movement. Capacity (4) is a host concern.
- **D-12** The ally set is passed per call because "ally" is relative to the mover (p. 176 "Ally"); the board stores no allegiance.

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased`, **Added** (not breaking): `combat`: optional square-grid positions and movement (`GridSquare`, `GridDirection`, `GridFootprint`, `GridCreature`, `GridTerrain`, `GridBoard`, `GridRules`, `GridStepAssessment`, `GridPathPlan`), and `CombatRuleOptions` / `PlayingOnAGrid` now exported. No existing behaviour or signal changes.
