package character

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	// MonkClassID is the id of the Monk class.
	MonkClassID content.ClassID = "monk"
	// OpenHandSubclassID is the id of the Warrior of the Open Hand, the only Monk subclass in SRD 5.2.1.
	OpenHandSubclassID content.SubclassID = "warrior_of_the_open_hand"

	monkMartialArtsColumn       content.ClassColumnID = "martial_arts"
	monkFocusPointsColumn       content.ClassColumnID = "focus_points"
	monkUnarmoredMovementColumn content.ClassColumnID = "unarmored_movement"
)

// The Monk levels that give each feature (SRD 5.2.1, "Monk Features", p. 50) and each Warrior of the Open Hand feature (p. 52).
const (
	monkFocusLevel           = 2
	monkDeflectLevel         = 3
	monkSlowFallLevel        = 4
	monkExtraAttackLevel     = 5
	monkEmpoweredLevel       = 6
	monkEvasionLevel         = 7
	monkAcrobaticLevel       = 9
	monkHeightenedLevel      = 10
	monkDeflectEnergyLevel   = 13
	monkSurvivorLevel        = 14
	monkPerfectFocusLevel    = 15
	monkSuperiorDefenseLevel = 18
	monkBodyAndMindLevel     = 20
	openHandTechniqueLevel   = 3
	openHandWholenessLevel   = 6
	openHandFleetStepLevel   = 11
	openHandPalmLevel        = 17
)

// MonkProgressionReason is why NewMonkProgression refused a definition. The zero
// value is not a reason.
type MonkProgressionReason int

const (
	// NotTheMonkDefinition: the definition is not the Monk's.
	NotTheMonkDefinition MonkProgressionReason = iota + 1
	// MonkColumnMissing: the martial_arts, focus_points or unarmored_movement column is missing or of the wrong kind.
	MonkColumnMissing
	// MonkRowsIncomplete: the Martial Arts cell of a level is a dash or not a single die.
	MonkRowsIncomplete
)

// String returns "not_the_monk_definition", "column_missing", "rows_incomplete" or "unknown".
func (r MonkProgressionReason) String() string {
	switch r {
	case NotTheMonkDefinition:
		return "not_the_monk_definition"
	case MonkColumnMissing:
		return "column_missing"
	case MonkRowsIncomplete:
		return "rows_incomplete"
	}
	return "unknown"
}

// MonkProgressionError reports one reason NewMonkProgression refused a definition.
// It matches srderr.ErrInvalidInput with errors.Is and carries no display text.
// Build it with a keyed literal.
type MonkProgressionError struct{ Reason MonkProgressionReason }

// Error renders the error. The text is not a contract.
func (e *MonkProgressionError) Error() string {
	return "character: invalid Monk progression: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*MonkProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *MonkProgressionError of the same Reason.
func (e *MonkProgressionError) Is(target error) bool {
	t, ok := target.(*MonkProgressionError)
	return ok && t.Reason == e.Reason
}

// MonkLevelStats are the numbers a Monk level gives (SRD 5.2.1, "Monk Features",
// p. 50): the Martial Arts die, the Focus Points maximum and the Unarmored Movement
// bonus. It carries no Proficiency Bonus, which follows the total level (SRD 5.2.1,
// "Multiclassing", p. 25). Only MonkProgression.StatsAt builds one; the zero value
// is invalid. Immutable, comparable with ==, safe for concurrent use.
type MonkLevelStats struct {
	level                           MonkLevel
	martialArts                     dice.Die
	focusPoints, unarmoredMovementF int
}

// Level returns the Monk level the numbers belong to.
func (s MonkLevelStats) Level() MonkLevel { return s.level }

// MartialArtsDie returns the Martial Arts die: d6, d8, d10 or d12.
func (s MonkLevelStats) MartialArtsDie() dice.Die { return s.martialArts }

// FocusPoints returns the Focus Points maximum; 0 at level 1.
func (s MonkLevelStats) FocusPoints() int { return s.focusPoints }

// UnarmoredMovementFeet returns the Unarmored Movement bonus in feet; 0 at level 1.
func (s MonkLevelStats) UnarmoredMovementFeet() int { return s.unarmoredMovementF }

// IsZero reports whether s is the invalid zero value.
func (s MonkLevelStats) IsZero() bool { return s.level.IsZero() }

// MonkProgression turns a Monk level into MonkLevelStats. The Martial Arts die,
// Focus Points and Unarmored Movement are read from the columns of the Monk
// ClassDefinition when it is built; it holds no port and no clock. It is immutable
// and comparable with ==; the zero value is invalid (see IsZero). Safe for
// concurrent use.
type MonkProgression struct {
	sides              [MaxLevel]int
	focusPoints, speed [MaxLevel]int
	set                bool
}

// NewMonkProgression reads the three columns of the levels 1 to 20 from def. A
// dash (content.ColumnNone) reads as 0. It checks structure only, never a value
// against the SRD, so house-rule content is accepted.
//
// It returns an error joining a *MonkProgressionError per reason found, matching
// srderr.ErrInvalidInput: NotTheMonkDefinition for a definition of another class or
// the zero definition, MonkColumnMissing for a missing or mistyped column and
// MonkRowsIncomplete for a level whose cell is of the wrong kind or a die that is
// not a single die. It never panics.
func NewMonkProgression(def content.ClassDefinition) (MonkProgression, error) {
	if def.ID() != MonkClassID {
		return MonkProgression{}, errors.Join(&MonkProgressionError{Reason: NotTheMonkDefinition})
	}
	kinds := map[content.ClassColumnID]content.ClassColumnKind{
		monkMartialArtsColumn: content.ColumnKindDice, monkFocusPointsColumn: content.ColumnKindCount, monkUnarmoredMovementColumn: content.ColumnKindFeetBonus,
	}
	for id, kind := range kinds {
		if col, err := def.Column(id); err != nil || col.Kind() != kind {
			return MonkProgression{}, errors.Join(&MonkProgressionError{Reason: MonkColumnMissing})
		}
	}
	var p MonkProgression
	for level := 1; level <= MaxLevel; level++ {
		row, _ := def.RowAt(level)               // a definition holds rows 1 to 20, so the level is in range
		v, _ := row.Value(monkMartialArtsColumn) // the decoder checked that every column has a cell of its kind or a dash
		die, isDie := v.(content.ColumnDice)
		if !isDie || die.Count() != 1 {
			return MonkProgression{}, errors.Join(&MonkProgressionError{Reason: MonkRowsIncomplete})
		}
		p.sides[level-1], p.focusPoints[level-1], p.speed[level-1] = die.Sides(), monkAmountAt(row, monkFocusPointsColumn), monkAmountAt(row, monkUnarmoredMovementColumn)
	}
	p.set = true
	return p, nil
}

// monkAmountAt returns the count or the feet of column id in row, and 0 for a dash. The column kind was checked by
// NewMonkProgression, so no other cell can be there.
func monkAmountAt(row content.ProgressionRow, id content.ClassColumnID) int {
	v, _ := row.Value(id)
	switch c := v.(type) {
	case content.ColumnCount:
		return c.Value()
	case content.ColumnFeetBonus:
		return c.Feet()
	}
	return 0
}

// MustMonkProgression is NewMonkProgression for a definition known to be valid. It
// panics with the error NewMonkProgression returns, and only then.
func MustMonkProgression(def content.ClassDefinition) MonkProgression {
	p, err := NewMonkProgression(def)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p MonkProgression) IsZero() bool { return !p.set }

// StatsAt returns the numbers of the Monk level; the zero MonkLevelStats for the
// zero progression or level.
func (p MonkProgression) StatsAt(level MonkLevel) MonkLevelStats {
	if !p.set || level.IsZero() {
		return MonkLevelStats{}
	}
	n := level.n
	die, _ := dice.NewDie(p.sides[n-1]) // the sides were read from a ColumnDice, which the decoder bounds to a die
	return MonkLevelStats{level: level, martialArts: die, focusPoints: p.focusPoints[n-1], unarmoredMovementF: p.speed[n-1]}
}
