package content

import "slices"

// TableCell is one cell of a [FeatureTable]: a closed set with two variants,
// [TableCellText] and [TableCellInt]. Consumers switch over it exhaustively.
// The variants are immutable, comparable and safe for concurrent use.
type TableCell interface{ isTableCell() }

// TableCellText is a text cell. The zero value is invalid; build it with
// [NewTableCellText].
type TableCellText struct{ text string }

// NewTableCellText returns a text cell.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter when text breaks the text rule
// (schema FR-028). It never panics.
func NewTableCellText(text string) (TableCellText, error) {
	if err := checkText("TableCellText", text); err != nil {
		return TableCellText{}, err
	}
	return TableCellText{text: text}, nil
}

// Text returns the cell text.
func (c TableCellText) Text() string { return c.text }

func (TableCellText) isTableCell() {}

// TableCellInt is an integer cell. The zero value is the cell 0.
type TableCellInt struct{ n int }

// NewTableCellInt returns an integer cell.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for n outside minus to plus [ClassMaxCount]. It never
// panics.
func NewTableCellInt(n int) (TableCellInt, error) {
	if n < -ClassMaxCount || n > ClassMaxCount {
		return TableCellInt{}, blockError("TableCellInt", BlockOutOfRange)
	}
	return TableCellInt{n: n}, nil
}

// Value returns the integer.
func (c TableCellInt) Value() int { return c.n }

func (TableCellInt) isTableCell() {}

func validTableCell(c TableCell) bool {
	switch v := c.(type) {
	case TableCellText:
		_, err := NewTableCellText(v.text)
		return err == nil
	case TableCellInt:
		_, err := NewTableCellInt(v.n)
		return err == nil
	}
	return false
}

// FeatureTable is a table the SRD prints inside a feature that no fact
// captures: a name, column headers and rows of cells, every column all text or
// all integers. A table printed as side-by-side column pairs is stored one
// pair per row. The zero value is invalid; build it with [NewFeatureTable]. It
// holds slices, so it is not comparable with ==.
type FeatureTable struct {
	name    string
	columns []string
	rows    [][]TableCell
}

// NewFeatureTable returns the table called name, copied.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName; BlockEmpty for no column or no row; BlockBlankText or
// BlockInvalidCharacter for a header that is blank or has a rejected
// character; BlockRaggedRow for a row not as long as the header list;
// BlockInvalidMember for a cell that is not valid; BlockConstraint for a
// column that mixes text and integers. It never panics.
func NewFeatureTable(name string, columns []string, rows [][]TableCell) (FeatureTable, error) {
	const block = "FeatureTable"
	if err := checkName(block, name); err != nil {
		return FeatureTable{}, err
	}
	if len(columns) == 0 || len(rows) == 0 { // equivalent mutants: a length is never negative, so == and <= agree
		return FeatureTable{}, blockError(block, BlockEmpty)
	}
	for _, h := range columns {
		if err := checkLine(block, h); err != nil {
			return FeatureTable{}, err
		}
	}
	for _, row := range rows {
		if len(row) != len(columns) {
			return FeatureTable{}, blockError(block, BlockRaggedRow)
		}
		for _, c := range row {
			if !validTableCell(c) {
				return FeatureTable{}, blockError(block, BlockInvalidMember)
			}
		}
	}
	for j := range columns {
		_, text := rows[0][j].(TableCellText)
		for _, row := range rows[1:] {
			if _, ok := row[j].(TableCellText); ok != text {
				return FeatureTable{}, blockError(block, BlockConstraint)
			}
		}
	}
	out := make([][]TableCell, len(rows))
	for i, row := range rows {
		out[i] = slices.Clone(row)
	}
	return FeatureTable{name: name, columns: slices.Clone(columns), rows: out}, nil
}

// Name returns the printed table name.
func (t FeatureTable) Name() string { return t.name }

// Columns returns the column headers, as a new slice.
func (t FeatureTable) Columns() []string { return slices.Clone(t.columns) }

// Rows returns the rows, as a new slice of new slices.
func (t FeatureTable) Rows() [][]TableCell {
	out := make([][]TableCell, len(t.rows))
	for i, row := range t.rows {
		out[i] = slices.Clone(row)
	}
	return out
}

func (t FeatureTable) valid() bool {
	_, err := NewFeatureTable(t.name, t.columns, t.rows)
	return err == nil
}
