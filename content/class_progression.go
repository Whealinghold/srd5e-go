package content

import (
	"maps"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// ClassColumnKind is the kind of value a class-specific column of the
// Features table holds: count, dice, bonus or feetBonus. The wire spelling is
// the lower camel case name. The zero value is invalid. Comparable and safe
// for concurrent use.
type ClassColumnKind int

// The column kinds.
const (
	ColumnKindCount ClassColumnKind = iota + 1
	ColumnKindDice
	ColumnKindBonus
	ColumnKindFeetBonus
)

func columnKindNames() []string { return []string{"count", "dice", "bonus", "feetBonus"} }

// Valid reports whether k is one of the four kinds.
func (k ClassColumnKind) Valid() bool { return enumValid(columnKindNames(), int(k)) }

// String returns the wire name of k, or "unknown" for an undefined value.
func (k ClassColumnKind) String() string { return enumString(columnKindNames(), int(k)) }

// ParseClassColumnKind returns the kind spelled exactly s ("count", "dice",
// "bonus" or "feetBonus"); matching is case-sensitive.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseClassColumnKind(s string) (ClassColumnKind, error) {
	v, err := parseEnum("ClassColumnKind", columnKindNames(), s)
	return ClassColumnKind(v), err
}

// ClassColumn declares one class-specific column of the Features table: its
// id, its printed heading and the kind of its values. The zero value is
// invalid. Comparable.
type ClassColumn struct {
	id     ClassColumnID
	header string
	kind   ClassColumnKind
}

// NewClassColumn returns the column id headed header whose values are of
// kind.
//
// It returns the *[IDError] of [ParseID] for an invalid id, then a
// *[BlockError] wrapping srderr.ErrInvalidInput: BlockBlankText or
// BlockInvalidCharacter for a header that is blank, multi-line or has a
// rejected character, BlockInvalidMember for a kind that is not a member. It
// never panics.
func NewClassColumn(id ClassColumnID, header string, kind ClassColumnKind) (ClassColumn, error) {
	const block = "ClassColumn"
	if _, err := ParseID(string(id)); err != nil {
		return ClassColumn{}, err
	}
	if err := checkLine(block, header); err != nil {
		return ClassColumn{}, err
	}
	if !kind.Valid() {
		return ClassColumn{}, blockError(block, BlockInvalidMember)
	}
	return ClassColumn{id: id, header: header, kind: kind}, nil
}

// ID returns the column id.
func (c ClassColumn) ID() ClassColumnID { return c.id }

// Header returns the printed column heading.
func (c ClassColumn) Header() string { return c.header }

// Kind returns the kind of the column's values.
func (c ClassColumn) Kind() ClassColumnKind { return c.kind }

func (c ClassColumn) valid() bool {
	_, err := NewClassColumn(c.id, c.header, c.kind)
	return err == nil
}

// checkLine is checkText for a single-line string (no line feed).
func checkLine(block, s string) error {
	switch {
	case blankText(s):
		return blockError(block, BlockBlankText)
	case !cleanCharacters(s, false):
		return blockError(block, BlockInvalidCharacter)
	}
	return nil
}

// ColumnValue is the value of a class-specific column in one row: a closed
// set with five variants, [ColumnCount], [ColumnDice], [ColumnBonus],
// [ColumnFeetBonus] and [ColumnNone] (a printed "—", allowed for every
// kind). Consumers switch over it exhaustively. The variants are immutable,
// comparable and safe for concurrent use.
type ColumnValue interface {
	// fits reports whether the value may stand in a column of kind k.
	fits(k ClassColumnKind) bool
}

func nonNegative(block string, n int) error {
	if n < 0 || n > ClassMaxCount {
		return blockError(block, BlockOutOfRange)
	}
	return nil
}

// ColumnCount is a whole number of at least 0. The zero value is 0.
type ColumnCount struct{ n int }

// NewColumnCount returns the count n, from 0 to [ClassMaxCount].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange otherwise. It never panics.
func NewColumnCount(n int) (ColumnCount, error) {
	if err := nonNegative("ColumnCount", n); err != nil {
		return ColumnCount{}, err
	}
	return ColumnCount{n: n}, nil
}

// Value returns the number.
func (c ColumnCount) Value() int { return c.n }

func (ColumnCount) fits(k ClassColumnKind) bool { return k == ColumnKindCount } // equivalent mutant when == becomes <=: a kind is never below the first

// ColumnDice is a number of dice of one size, such as 1d6. The zero value is
// invalid; build it with [NewColumnDice].
type ColumnDice struct{ count, sides int }

// NewColumnDice returns count dice of sides faces each.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a count outside 1 to [ClassMaxCount] or a number of
// sides the kernel dice.Die rejects. It never panics.
func NewColumnDice(count, sides int) (ColumnDice, error) {
	if _, err := dice.NewDie(sides); err != nil || count < 1 || count > ClassMaxCount {
		return ColumnDice{}, blockError("ColumnDice", BlockOutOfRange)
	}
	return ColumnDice{count: count, sides: sides}, nil
}

// Count returns the number of dice.
func (c ColumnDice) Count() int { return c.count }

// Sides returns the number of faces of each die.
func (c ColumnDice) Sides() int { return c.sides }

func (ColumnDice) fits(k ClassColumnKind) bool { return k == ColumnKindDice }

// ColumnBonus is a bonus of at least 0 (printed "+2"). The zero value is 0.
type ColumnBonus struct{ n int }

// NewColumnBonus returns the bonus n, from 0 to [ClassMaxCount].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange otherwise. It never panics.
func NewColumnBonus(n int) (ColumnBonus, error) {
	if err := nonNegative("ColumnBonus", n); err != nil {
		return ColumnBonus{}, err
	}
	return ColumnBonus{n: n}, nil
}

// Value returns the bonus.
func (c ColumnBonus) Value() int { return c.n }

func (ColumnBonus) fits(k ClassColumnKind) bool { return k == ColumnKindBonus }

// ColumnFeetBonus is a number of feet of at least 0 (printed "+10 ft."). The
// zero value is 0 feet.
type ColumnFeetBonus struct{ n int }

// NewColumnFeetBonus returns the bonus of n feet, from 0 to [ClassMaxCount].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange otherwise. It never panics.
func NewColumnFeetBonus(n int) (ColumnFeetBonus, error) {
	if err := nonNegative("ColumnFeetBonus", n); err != nil {
		return ColumnFeetBonus{}, err
	}
	return ColumnFeetBonus{n: n}, nil
}

// Feet returns the number of feet.
func (c ColumnFeetBonus) Feet() int { return c.n }

func (ColumnFeetBonus) fits(k ClassColumnKind) bool { return k == ColumnKindFeetBonus } // equivalent mutant when == becomes >=: no kind is above the last

// ColumnNone is a printed "—": the column has no value at that level. It is
// allowed in a column of every kind.
type ColumnNone struct{}

func (ColumnNone) fits(k ClassColumnKind) bool { return k.Valid() }

// ColumnNoneText is the wire text of [ColumnNone]: an em dash, U+2014,
// exactly as the SRD prints it.
const ColumnNoneText = "—"

func validColumnValue(v ColumnValue) bool {
	switch x := v.(type) {
	case ColumnCount:
		_, err := NewColumnCount(x.n)
		return err == nil
	case ColumnDice:
		_, err := NewColumnDice(x.count, x.sides)
		return err == nil
	case ColumnBonus:
		_, err := NewColumnBonus(x.n)
		return err == nil
	case ColumnFeetBonus:
		_, err := NewColumnFeetBonus(x.n)
		return err == nil
	case ColumnNone:
		return true
	}
	return false
}

// RowFeature is one element of the Class Features cell of a row: a closed set
// with two variants, [RowClassFeature] (a feature of the class) and
// [RowSubclassFeature] (the "Subclass feature" marker). Consumers switch over
// it exhaustively. The variants are immutable, comparable and safe for
// concurrent use.
type RowFeature interface{ isRowFeature() }

// RowClassFeature names a feature entry of the class, with the printed
// parenthetical annotation ("one use") when the cell has one. The zero value
// is invalid; build it with [NewRowClassFeature].
type RowClassFeature struct {
	featureID  FeatureID
	annotation string
}

// NewRowClassFeature returns a reference to the feature featureID with an
// optional annotation ("" for none).
//
// It returns the *[IDError] of [ParseID] for an invalid id, and a
// *[BlockError] wrapping srderr.ErrInvalidInput with BlockBlankText or
// BlockInvalidCharacter for a non-empty annotation that is blank, multi-line
// or has a rejected character. It never panics.
func NewRowClassFeature(featureID FeatureID, annotation string) (RowClassFeature, error) {
	if _, err := ParseID(string(featureID)); err != nil {
		return RowClassFeature{}, err
	}
	if annotation != "" { // equivalent mutant: no string is below "", so != and > agree
		if err := checkLine("RowClassFeature", annotation); err != nil {
			return RowClassFeature{}, err
		}
	}
	return RowClassFeature{featureID: featureID, annotation: annotation}, nil
}

// FeatureID returns the id of the feature entry.
func (f RowClassFeature) FeatureID() FeatureID { return f.featureID }

// Annotation returns the printed annotation, or "" when there is none.
func (f RowClassFeature) Annotation() string { return f.annotation }

func (RowClassFeature) isRowFeature() {}

// RowSubclassFeature marks the level at which the subclass supplies a
// feature. The zero value is the marker.
type RowSubclassFeature struct{}

func (RowSubclassFeature) isRowFeature() {}

func validRowFeature(f RowFeature) bool {
	switch v := f.(type) {
	case RowClassFeature:
		_, err := NewRowClassFeature(v.featureID, v.annotation)
		return err == nil
	case RowSubclassFeature:
		return true
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}

// ProgressionRow is one row of the Features table of a class: the level, the
// Proficiency Bonus, the class features gained (by reference), the
// spellcasting part and the values of the class-specific columns. The zero
// value is invalid; build it with [NewProgressionRow]. It holds slices and a
// map, so it is not comparable with ==.
type ProgressionRow struct {
	level            int
	proficiencyBonus int
	features         []RowFeature
	spellcasting     SpellcastingRow
	values           map[ClassColumnID]ColumnValue
}

// NewProgressionRow returns the row of level with the given parts, copied.
// spellcasting is nil for a class without spellcasting; features may be empty
// (a printed "—").
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for a level outside 1 to [ClassLevels] or a bonus below 1,
// BlockInvalidMember for a feature, spellcasting part or value that is not
// valid or a value under an invalid column id, and BlockDuplicate for more
// than one [RowSubclassFeature]. It never panics.
func NewProgressionRow(level, proficiencyBonus int, features []RowFeature, spellcasting SpellcastingRow, values map[ClassColumnID]ColumnValue) (ProgressionRow, error) {
	const block = "ProgressionRow"
	if level < 1 || level > ClassLevels || proficiencyBonus < 1 || proficiencyBonus > ClassMaxCount {
		return ProgressionRow{}, blockError(block, BlockOutOfRange)
	}
	markers := 0
	for _, f := range features {
		if !validRowFeature(f) {
			return ProgressionRow{}, blockError(block, BlockInvalidMember)
		}
		if _, ok := f.(RowSubclassFeature); ok {
			markers++
		}
	}
	if markers > 1 {
		return ProgressionRow{}, blockError(block, BlockDuplicate)
	}
	if spellcasting != nil && !validSpellcastingRow(spellcasting) {
		return ProgressionRow{}, blockError(block, BlockInvalidMember)
	}
	for id, v := range values {
		if !id.Valid() || !validColumnValue(v) {
			return ProgressionRow{}, blockError(block, BlockInvalidMember)
		}
	}
	return ProgressionRow{
		level: level, proficiencyBonus: proficiencyBonus, features: cloneOrNil(features),
		spellcasting: spellcasting, values: maps.Clone(values),
	}, nil
}

// Level returns the class level of the row, 1 to 20.
func (r ProgressionRow) Level() int { return r.level }

// ProficiencyBonus returns the Proficiency Bonus of the row.
func (r ProgressionRow) ProficiencyBonus() int { return r.proficiencyBonus }

// Features returns the Class Features cell in printed order, as a new slice.
func (r ProgressionRow) Features() []RowFeature { return slices.Clone(r.features) }

// Spellcasting returns the spellcasting part, or nil for a class without
// spellcasting.
func (r ProgressionRow) Spellcasting() SpellcastingRow { return r.spellcasting }

// Value returns the value of the class-specific column id.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember when the row has no such column.
func (r ProgressionRow) Value(id ClassColumnID) (ColumnValue, error) {
	v, ok := r.values[id]
	if !ok {
		return nil, blockError("ProgressionRow", BlockInvalidMember)
	}
	return v, nil
}

func (r ProgressionRow) valid() bool {
	_, err := NewProgressionRow(r.level, r.proficiencyBonus, r.features, r.spellcasting, r.values)
	return err == nil
}
