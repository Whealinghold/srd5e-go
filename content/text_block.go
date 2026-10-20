package content

import "slices"

// TextBlock is the closed family of printed-text blocks that keep verbatim
// text in a neutral shape, such as a magic item description. It names no kind
// of content, so any kind may use it. Variants: TextParagraph, TextEntry,
// TextBulletList, TextTable, TextEmbeddedStatBlock. Consumers switch over it
// exhaustively. Every variant is immutable and safe for concurrent use, and
// copies its slices on construction and on read; text is stored verbatim.
type TextBlock interface{ isTextBlock() }

// TextParagraph is one paragraph of text. The zero value is invalid; build it
// with [NewTextParagraph]. Comparable.
type TextParagraph struct{ text string }

// NewTextParagraph returns a paragraph of text, which must contain a
// non-white-space character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText otherwise. It never panics.
func NewTextParagraph(text string) (TextParagraph, error) {
	if err := checkText("TextParagraph", text); err != nil {
		return TextParagraph{}, err
	}
	return TextParagraph{text: text}, nil
}

// MustTextParagraph is like [NewTextParagraph] but panics with its error when
// text is invalid.
func MustTextParagraph(text string) TextParagraph {
	p, err := NewTextParagraph(text)
	if err != nil {
		panic(err)
	}
	return p
}

// Text returns the text.
func (p TextParagraph) Text() string { return p.text }

func (TextParagraph) isTextBlock() {}

// TextEntry is a run-in heading and its text. The zero value is invalid;
// build it with [NewTextEntry]. Comparable.
type TextEntry struct{ name, text string }

// NewTextEntry returns an entry. The name must be non-empty with no leading
// or trailing white space, and the text must contain a non-white-space
// character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName or BlockBlankText, checked in that order. It never panics.
func NewTextEntry(name, text string) (TextEntry, error) {
	if err := checkName("TextEntry", name); err != nil {
		return TextEntry{}, err
	}
	if err := checkText("TextEntry", text); err != nil {
		return TextEntry{}, err
	}
	return TextEntry{name: name, text: text}, nil
}

// MustTextEntry is like [NewTextEntry] but panics with its error when the
// arguments are invalid.
func MustTextEntry(name, text string) TextEntry {
	e, err := NewTextEntry(name, text)
	if err != nil {
		panic(err)
	}
	return e
}

// Name returns the run-in heading.
func (e TextEntry) Name() string { return e.name }

// Text returns the text.
func (e TextEntry) Text() string { return e.text }

func (TextEntry) isTextBlock() {}

// checkTexts requires at least one string and every one non-blank.
func checkTexts(block string, ss []string) error {
	if len(ss) == 0 { // equivalent mutant: == and <= agree for a length
		return blockError(block, BlockEmpty)
	}
	for _, s := range ss {
		if err := checkText(block, s); err != nil {
			return err
		}
	}
	return nil
}

// TextBulletList is a bulleted list. The zero value is invalid; build it with
// [NewTextBulletList]. It holds a slice, so compare it with
// [TextBulletList.Equal].
type TextBulletList struct{ items []string }

// NewTextBulletList returns a list of a copy of items. There must be at least
// one item, and every item must contain a non-white-space character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// or BlockBlankText otherwise. It never panics.
func NewTextBulletList(items []string) (TextBulletList, error) {
	if err := checkTexts("TextBulletList", items); err != nil {
		return TextBulletList{}, err
	}
	return TextBulletList{items: slices.Clone(items)}, nil
}

// MustTextBulletList is like [NewTextBulletList] but panics with its error when the
// arguments are invalid.
func MustTextBulletList(items []string) TextBulletList {
	v, err := NewTextBulletList(items)
	if err != nil {
		panic(err)
	}
	return v
}

// Items returns the items as a new slice.
func (l TextBulletList) Items() []string { return slices.Clone(l.items) }

// Equal reports whether l and m have the same items in the same order.
func (l TextBulletList) Equal(m TextBulletList) bool { return slices.Equal(l.items, m.items) }

func (TextBulletList) isTextBlock() {}

// TextTable is a table with a header and rows, and optionally a title. The
// zero value is invalid; build it with [NewTextTable] or
// [NewTitledTextTable]. It holds slices, so compare it with
// [TextTable.Equal].
type TextTable struct {
	title    string
	hasTitle bool
	header   []string
	rows     [][]string
}

// NewTextTable returns an untitled table of copies of header and rows. The
// header must have at least one cell and every header cell must contain a
// non-white-space character. There must be at least one row, every row must be
// exactly as wide as the header, and a row cell may be empty but not
// white space only.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockEmpty for
// an empty header or no rows, BlockBlankText for a blank cell, BlockRaggedRow
// for a row of another width. Checks run header, then rows in order. It never
// panics.
func NewTextTable(header []string, rows [][]string) (TextTable, error) {
	const block = "TextTable"
	if err := checkTexts(block, header); err != nil {
		return TextTable{}, err
	}
	if len(rows) == 0 { // equivalent mutant: == and <= agree for a length
		return TextTable{}, blockError(block, BlockEmpty)
	}
	cp := make([][]string, len(rows))
	for i, row := range rows {
		if len(row) != len(header) {
			return TextTable{}, blockError(block, BlockRaggedRow)
		}
		for _, cell := range row {
			if cell != "" { // equivalent mutant: != and > agree for strings against ""
				if err := checkText(block, cell); err != nil {
					return TextTable{}, err
				}
			}
		}
		cp[i] = slices.Clone(row)
	}
	return TextTable{header: slices.Clone(header), rows: cp}, nil
}

// NewTitledTextTable is like [NewTextTable] and also sets the title, which
// must contain a non-white-space character (so an empty title is rejected,
// not treated as absent). The title is checked first.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText for a blank title, or the errors of NewTextTable. It never
// panics.
func NewTitledTextTable(title string, header []string, rows [][]string) (TextTable, error) {
	if err := checkText("TextTable", title); err != nil {
		return TextTable{}, err
	}
	t, err := NewTextTable(header, rows)
	if err != nil {
		return TextTable{}, err
	}
	t.title, t.hasTitle = title, true
	return t, nil
}

// MustTextTable is like [NewTextTable] but panics with its error when the
// arguments are invalid.
func MustTextTable(header []string, rows [][]string) TextTable {
	v, err := NewTextTable(header, rows)
	if err != nil {
		panic(err)
	}
	return v
}

// MustTitledTextTable is like [NewTitledTextTable] but panics with its error when the
// arguments are invalid.
func MustTitledTextTable(title string, header []string, rows [][]string) TextTable {
	v, err := NewTitledTextTable(title, header, rows)
	if err != nil {
		panic(err)
	}
	return v
}

// Title returns the title and whether the table has one.
func (t TextTable) Title() (string, bool) { return t.title, t.hasTitle }

// Header returns the header cells as a new slice.
func (t TextTable) Header() []string { return slices.Clone(t.header) }

// Rows returns the rows as a new slice of new slices.
func (t TextTable) Rows() [][]string {
	rows := make([][]string, len(t.rows))
	for i, row := range t.rows {
		rows[i] = slices.Clone(row)
	}
	return rows
}

// Equal reports whether t and u have the same title (or both none), header
// and rows.
func (t TextTable) Equal(u TextTable) bool {
	return t.title == u.title && t.hasTitle == u.hasTitle &&
		slices.Equal(t.header, u.header) && slices.EqualFunc(t.rows, u.rows, slices.Equal)
}

func (TextTable) isTextBlock() {}

// TextEmbeddedStatBlock is a stat block printed inside other text, kept as
// its printed lines. The zero value is invalid; build it with
// [NewTextEmbeddedStatBlock]. It holds a slice, so compare it with
// [TextEmbeddedStatBlock.Equal].
type TextEmbeddedStatBlock struct{ lines []string }

// NewTextEmbeddedStatBlock returns a stat block of a copy of lines. There
// must be at least one line, and every line must contain a non-white-space
// character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// or BlockBlankText otherwise. It never panics.
func NewTextEmbeddedStatBlock(lines []string) (TextEmbeddedStatBlock, error) {
	if err := checkTexts("TextEmbeddedStatBlock", lines); err != nil {
		return TextEmbeddedStatBlock{}, err
	}
	return TextEmbeddedStatBlock{lines: slices.Clone(lines)}, nil
}

// MustTextEmbeddedStatBlock is like [NewTextEmbeddedStatBlock] but panics with its error when the
// arguments are invalid.
func MustTextEmbeddedStatBlock(lines []string) TextEmbeddedStatBlock {
	v, err := NewTextEmbeddedStatBlock(lines)
	if err != nil {
		panic(err)
	}
	return v
}

// Lines returns the lines as a new slice.
func (s TextEmbeddedStatBlock) Lines() []string { return slices.Clone(s.lines) }

// Equal reports whether s and o have the same lines in the same order.
func (s TextEmbeddedStatBlock) Equal(o TextEmbeddedStatBlock) bool {
	return slices.Equal(s.lines, o.lines)
}

func (TextEmbeddedStatBlock) isTextBlock() {}
