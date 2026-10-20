package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: schema FR-150, FR-151 - paragraph text and entry name and text
// follow FR-027 and FR-028 and are stored verbatim.
// Path: per case.
func TestTextParagraphAndEntry(t *testing.T) {
	t.Parallel()
	p, err := content.NewTextParagraph(" It’s kept.\n ")
	if err != nil || p.Text() != " It’s kept.\n " || p != content.MustTextParagraph(" It’s kept.\n ") {
		t.Errorf("NewTextParagraph valid = (%+v, %v) [happy]", p, err)
	}
	for _, in := range []string{"", " \n\t"} {
		_, err := content.NewTextParagraph(in)
		wantBlockError(t, err, "TextParagraph", content.BlockBlankText)
		assertMustPanics(t, func() { content.MustTextParagraph(in) })
	}

	e, err := content.NewTextEntry("Heading", "Body. ")
	if err != nil || e.Name() != "Heading" || e.Text() != "Body. " || e != content.MustTextEntry("Heading", "Body. ") {
		t.Errorf("NewTextEntry valid = (%+v, %v) [happy]", e, err)
	}
	for _, c := range []struct {
		name, text string
		want       content.BlockProblem
	}{
		{"", "t", content.BlockInvalidName},
		{" n", "t", content.BlockInvalidName},
		{"n ", "t", content.BlockInvalidName},
		{"n", " ", content.BlockBlankText},
		{"", "", content.BlockInvalidName},
	} {
		_, err := content.NewTextEntry(c.name, c.text)
		wantBlockError(t, err, "TextEntry", c.want)
		assertMustPanics(t, func() { content.MustTextEntry(c.name, c.text) })
	}
}

// Requirement: schema FR-151 - a bullet list and an embedded stat block have at
// least one item or line, each non-blank; both copy on construction and on read.
// Path: per case.
func TestTextBulletListAndStatBlock(t *testing.T) {
	t.Parallel()
	in := []string{"a", "b"}
	l, err := content.NewTextBulletList(in)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = "changed"
	l.Items()[1] = "changed"
	if got := l.Items(); got[0] != "a" || got[1] != "b" {
		t.Errorf("Items() = %v after mutating the input and a result, want [a b] [unhappy (immutability)]", got)
	}
	l2, _ := content.NewTextBulletList([]string{"a", "b"})
	l3, _ := content.NewTextBulletList([]string{"a", "c"})
	l4, _ := content.NewTextBulletList([]string{"a"})
	if !l.Equal(l2) || l.Equal(l3) || l.Equal(l4) {
		t.Error("TextBulletList.Equal wrong")
	}
	for _, c := range []struct {
		name  string
		items []string
		want  content.BlockProblem
	}{
		{"nil", nil, content.BlockEmpty},
		{"empty", []string{}, content.BlockEmpty},
		{"blank item", []string{"a", " "}, content.BlockBlankText},
		{"empty item", []string{""}, content.BlockBlankText},
	} {
		_, err := content.NewTextBulletList(c.items)
		wantBlockError(t, err, "TextBulletList", c.want)
		_, err = content.NewTextEmbeddedStatBlock(c.items)
		wantBlockError(t, err, "TextEmbeddedStatBlock", c.want)
		assertMustPanics(t, func() { content.MustTextBulletList(c.items) })
		assertMustPanics(t, func() { content.MustTextEmbeddedStatBlock(c.items) })
	}
	if !content.MustTextBulletList([]string{"a"}).Equal(content.MustTextBulletList([]string{"a"})) ||
		!content.MustTextEmbeddedStatBlock([]string{"a"}).Equal(content.MustTextEmbeddedStatBlock([]string{"a"})) {
		t.Error("Must forms of a valid list and stat block differ from the New forms [happy]")
	}

	lines := []string{"Goblin", "AC 15"}
	s, err := content.NewTextEmbeddedStatBlock(lines)
	if err != nil {
		t.Fatal(err)
	}
	lines[0] = "changed"
	s.Lines()[1] = "changed"
	if got := s.Lines(); got[0] != "Goblin" || got[1] != "AC 15" {
		t.Errorf("Lines() = %v after mutations, want the original [unhappy (immutability)]", got)
	}
	s2, _ := content.NewTextEmbeddedStatBlock([]string{"Goblin", "AC 15"})
	s3, _ := content.NewTextEmbeddedStatBlock([]string{"Goblin"})
	if !s.Equal(s2) || s.Equal(s3) {
		t.Error("TextEmbeddedStatBlock.Equal wrong")
	}
}

// Requirement: schema FR-151 - a table has a non-empty header of non-blank
// cells, at least one row, every row as wide as the header (an empty cell is
// allowed, a white-space-only cell is not), and a title that is absent or non-blank.
// Path: per case.
func TestTextTable(t *testing.T) {
	t.Parallel()
	header := []string{"d6", "Result"}
	rows := [][]string{{"1", "Hit"}, {"2", ""}}
	tbl, err := content.NewTextTable(header, rows)
	if err != nil {
		t.Fatalf("NewTextTable valid = %v [happy]", err)
	}
	if title, has := tbl.Title(); has || title != "" {
		t.Errorf("Title() = (%q, %v), want none", title, has)
	}
	titled, err := content.NewTitledTextTable("Outcomes", header, rows)
	if err != nil {
		t.Fatal(err)
	}
	if title, has := titled.Title(); !has || title != "Outcomes" {
		t.Errorf("Title() = (%q, %v), want (Outcomes, true)", title, has)
	}
	if !tbl.Equal(content.MustTextTable([]string{"d6", "Result"}, [][]string{{"1", "Hit"}, {"2", ""}})) ||
		!titled.Equal(content.MustTitledTextTable("Outcomes", []string{"d6", "Result"}, [][]string{{"1", "Hit"}, {"2", ""}})) {
		t.Error("Must forms of a valid table differ from the New forms [happy]")
	}
	assertMustPanics(t, func() { content.MustTextTable(nil, [][]string{{"x"}}) })
	assertMustPanics(t, func() { content.MustTitledTextTable(" ", []string{"a"}, [][]string{{"x"}}) })
	header[0], rows[0][0] = "changed", "changed"
	tbl.Header()[1] = "changed"
	tbl.Rows()[1][0] = "changed"
	tbl.Rows()[0] = nil
	if h, r := tbl.Header(), tbl.Rows(); h[0] != "d6" || h[1] != "Result" || r[0][0] != "1" || r[1][0] != "2" || len(r) != 2 {
		t.Errorf("table = %v %v after mutating the inputs and results, want the original [unhappy (immutability)]", h, r)
	}

	mk := func(title bool, h []string, r [][]string) (content.TextTable, error) {
		if title {
			return content.NewTitledTextTable("T", h, r)
		}
		return content.NewTextTable(h, r)
	}
	a, _ := mk(false, []string{"a"}, [][]string{{"x"}})
	b, _ := mk(false, []string{"a"}, [][]string{{"x"}})
	for name, other := range map[string]content.TextTable{
		"different header": mustTable(t, []string{"b"}, [][]string{{"x"}}),
		"different cell":   mustTable(t, []string{"a"}, [][]string{{"y"}}),
		"more rows":        mustTable(t, []string{"a"}, [][]string{{"x"}, {"x"}}),
		"titled":           mustTitled(t, "T", []string{"a"}, [][]string{{"x"}}),
	} {
		if a.Equal(other) {
			t.Errorf("Equal true for %s", name)
		}
	}
	if !a.Equal(b) || a.Equal(mustTitled(t, "T", []string{"a"}, [][]string{{"x"}})) || mustTitled(t, "T", []string{"a"}, [][]string{{"x"}}).Equal(mustTitled(t, "U", []string{"a"}, [][]string{{"x"}})) || mustTitled(t, "U", []string{"a"}, [][]string{{"x"}}).Equal(mustTitled(t, "T", []string{"a"}, [][]string{{"x"}})) {
		t.Error("Equal: want equal for same content, unequal across title")
	}

	cases := []struct {
		name   string
		titled bool
		title  string
		header []string
		rows   [][]string
		want   content.BlockProblem
	}{
		{"nil header", false, "", nil, [][]string{{}}, content.BlockEmpty},
		{"empty header", false, "", []string{}, [][]string{{}}, content.BlockEmpty},
		{"blank header cell", false, "", []string{"a", " "}, [][]string{{"x", "y"}}, content.BlockBlankText},
		{"empty header cell", false, "", []string{""}, [][]string{{"x"}}, content.BlockBlankText},
		{"nil rows", false, "", []string{"a"}, nil, content.BlockEmpty},
		{"zero rows", false, "", []string{"a"}, [][]string{}, content.BlockEmpty},
		{"short row", false, "", []string{"a", "b"}, [][]string{{"x"}}, content.BlockRaggedRow},
		{"long row", false, "", []string{"a"}, [][]string{{"x"}, {"x", "y"}}, content.BlockRaggedRow},
		{"blank row cell", false, "", []string{"a"}, [][]string{{" "}}, content.BlockBlankText},
		{"blank title", true, " ", []string{"a"}, [][]string{{"x"}}, content.BlockBlankText},
		{"empty title is not absent", true, "", []string{"a"}, [][]string{{"x"}}, content.BlockBlankText},
		{"title checked first", true, "", nil, nil, content.BlockBlankText},
		{"titled with a bad header", true, "T", []string{" "}, [][]string{{"x"}}, content.BlockBlankText},
		{"titled with no rows", true, "T", []string{"a"}, nil, content.BlockEmpty},
		{"titled with a ragged row", true, "T", []string{"a"}, [][]string{{"x", "y"}}, content.BlockRaggedRow},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if c.titled {
				_, err = content.NewTitledTextTable(c.title, c.header, c.rows)
			} else {
				_, err = content.NewTextTable(c.header, c.rows)
			}
			wantBlockError(t, err, "TextTable", c.want)
		})
	}
}

func mustTable(t *testing.T, header []string, rows [][]string) content.TextTable {
	t.Helper()
	tbl, err := content.NewTextTable(header, rows)
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func mustTitled(t *testing.T, title string, header []string, rows [][]string) content.TextTable {
	t.Helper()
	tbl, err := content.NewTitledTextTable(title, header, rows)
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

// Requirement: schema FR-150 - the five variants form one closed family,
// switched exhaustively.
// Path: happy.
func TestTextBlockVariants(t *testing.T) {
	t.Parallel()
	list, _ := content.NewTextBulletList([]string{"a"})
	stat, _ := content.NewTextEmbeddedStatBlock([]string{"a"})
	blocks := map[string]content.TextBlock{
		"paragraph":         content.MustTextParagraph("a"),
		"entry":             content.MustTextEntry("n", "a"),
		"bulletList":        list,
		"table":             mustTable(t, []string{"a"}, [][]string{{"b"}}),
		"embeddedStatBlock": stat,
	}
	for want, b := range blocks {
		var got string
		switch b.(type) {
		case content.TextParagraph:
			got = "paragraph"
		case content.TextEntry:
			got = "entry"
		case content.TextBulletList:
			got = "bulletList"
		case content.TextTable:
			got = "table"
		case content.TextEmbeddedStatBlock:
			got = "embeddedStatBlock"
		}
		if got != want {
			t.Errorf("%T classified as %q, want %q", b, got, want)
		}
	}
}
