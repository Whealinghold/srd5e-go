package content

// The wire discriminators of a [TextBlock] (schema FR-152).
const (
	blockKindParagraph  = "paragraph"
	blockKindEntry      = "entry"
	blockKindBulletList = "bulletList"
	blockKindTable      = "table"
	blockKindStatBlock  = "embeddedStatBlock"
)

// decodeTextBlocks reads the array of blocks at n (schema FR-152) with at
// least one element: an empty array is outOfRange at the array. It returns
// the blocks that decoded; a block that did not has reported a problem, which
// drops the entry that holds the list.
func decodeTextBlocks(d *decoder, n node, path string) []TextBlock {
	var out []TextBlock
	d.list(n, path, 1, noUpperBound, func(_ int, el node, p string) {
		if b := decodeTextBlock(d, el, p); b != nil {
			out = append(out, b)
		}
	})
	return out
}

// decodeTextBlock reads one block object: the string "kind" first (missing
// is missingField, a non-string wrongType, another value invalidValue, and
// then no other key is examined), then the keys of that kind in the order of
// FR-152.
func decodeTextBlock(d *decoder, n node, path string) TextBlock {
	o, ok := d.object(n, path)
	if !ok {
		return nil
	}
	defer o.finish()
	kn, kp, present := o.field("kind", true)
	if !present {
		o.skipAll()
		return nil
	}
	kind, ok := d.str(kn, kp)
	if !ok {
		o.skipAll()
		return nil
	}
	var b TextBlock
	var err error
	switch kind {
	case blockKindParagraph:
		var text string
		if n, p, ok := o.field("text", true); ok {
			text, _ = d.text(n, p)
		}
		b, err = NewTextParagraph(text)
	case blockKindEntry:
		var name, text string
		if n, p, ok := o.field("name", true); ok {
			name, _ = d.name(n, p)
		}
		if n, p, ok := o.field("text", true); ok {
			text, _ = d.text(n, p)
		}
		b, err = NewTextEntry(name, text)
	case blockKindBulletList:
		var items []string
		if n, p, ok := o.field("items", true); ok {
			items = decodeTexts(d, n, p)
		}
		b, err = NewTextBulletList(items)
	case blockKindTable:
		b, err = decodeTable(d, o)
	case blockKindStatBlock:
		var lines []string
		if n, p, ok := o.field("lines", true); ok {
			lines = decodeTexts(d, n, p)
		}
		b, err = NewTextEmbeddedStatBlock(lines)
	default:
		d.probs.add(kp, ProblemInvalidValue)
		o.skipAll()
		return nil
	}
	if err != nil { // a problem was reported for any value that made the constructor fail
		return nil
	}
	return b
}

// decodeTexts reads a non-empty array of strings that each pass the text
// rule. It returns the strings that did.
func decodeTexts(d *decoder, n node, path string) []string {
	var out []string
	d.list(n, path, 1, noUpperBound, func(_ int, el node, p string) {
		if s, ok := d.text(el, p); ok {
			out = append(out, s)
		}
	})
	return out
}

// decodeTable reads the keys title?, header and rows of a table block. A row
// that is not as wide as the header is constraintViolation at the row, checked
// only when the header decoded cleanly; a cell may be empty.
func decodeTable(d *decoder, o *object) (TextBlock, error) {
	var title string
	hasTitle := false
	if n, p, ok := o.field("title", false); ok {
		title, hasTitle = d.text(n, p)
	}
	var header []string
	headerClean := false
	if n, p, ok := o.field("header", true); ok {
		before := d.probs.count
		header = decodeTexts(d, n, p)
		headerClean = d.probs.count == before
	}
	var rows [][]string
	if n, p, ok := o.field("rows", true); ok {
		d.list(n, p, 1, noUpperBound, func(_ int, el node, rp string) {
			if row := decodeTableRow(d, el, rp); row != nil {
				if headerClean && len(row) != len(header) {
					d.probs.add(rp, ProblemConstraintViolation)
					return
				}
				rows = append(rows, row)
			}
		})
	}
	if hasTitle {
		return NewTitledTextTable(title, header, rows)
	}
	return NewTextTable(header, rows)
}

// decodeTableRow reads one row of strings, each empty or passing the text
// rule. It returns nil for a row that is not an array or has a bad cell.
func decodeTableRow(d *decoder, n node, path string) []string {
	if n.kind() != nodeArray {
		d.probs.add(path, ProblemWrongType)
		return nil
	}
	before := d.probs.count
	row := []string{} // not nil, so an empty row is told from a failed one
	d.each(n, path, func(_ int, el node, p string) {
		if s, ok := d.checked(el, p, validCell); ok {
			row = append(row, s)
		}
	})
	if d.probs.count != before {
		return nil
	}
	return row
}

// validCell reports whether s is a table cell: empty, or a valid text.
func validCell(s string) bool {
	return s == "" || validText(s) // equivalent mutant: a string is never below "", so == and <= agree
}
