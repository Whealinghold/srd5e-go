package content

// decodeFeatureEntries reads a list of feature entries; the valid id of each
// joins ids for the document-wide duplicate check.
func decodeFeatureEntries(d *decoder, n node, p string, ids *[]nestedID) []FeatureEntry {
	var out []FeatureEntry
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		if e, ok := decodeFeatureEntry(d, el, ep, ids); ok {
			out = append(out, e)
		}
	})
	return out
}

// decodeFeatureEntry reads {id, name, level, text, facts?, options?, tables?}
// (FR-051). The rules between its parts are reported right after its fields,
// and only when they decoded cleanly.
func decodeFeatureEntry(d *decoder, el node, p string, ids *[]nestedID) (FeatureEntry, bool) {
	o, ok := d.object(el, p)
	if !ok {
		return FeatureEntry{}, false // equivalent mutant: the non-object was reported, and the entry is dropped with it
	}
	before := d.probs.count
	var id FeatureID
	var name string
	var f FeatureEntryFields
	if n, ip, ok := o.field("id", true); ok {
		if raw, ok := d.id(n, ip); ok {
			id = FeatureID(raw.String())
			*ids = append(*ids, nestedID{group: nestedFeature, path: ip, id: id.String()})
		}
	}
	if n, np, ok := o.field("name", true); ok {
		name, _ = d.name(n, np)
	}
	if n, lp, ok := o.field("level", true); ok {
		v, _ := d.integer(n, lp, 1, ClassLevels)
		f.Level = int(v)
	}
	if n, tp, ok := o.field("text", true); ok {
		f.Text, _ = d.text(n, tp)
	}
	if n, fp, ok := o.field("facts", false); ok {
		f.Facts = decodeFacts(d, n, fp, false)
	}
	if n, op, ok := o.field("options", false); ok {
		f.Options = decodeFeatureOptions(d, n, op)
	}
	if n, tp, ok := o.field("tables", false); ok {
		f.Tables = decodeTables(d, n, tp)
	}
	o.finish()
	if d.probs.count != before { // equivalent mutant when != becomes >: the count never falls below before
		return FeatureEntry{}, false // equivalent mutant: a problem was reported, so no entry is built
	}
	e := FeatureEntry{id: id, name: name, level: f.Level, text: f.Text, facts: f.Facts, options: f.Options, tables: f.Tables}
	for _, is := range e.problems() {
		d.probs.add(p+"/"+is.path, is.kind)
	}
	if d.probs.count != before { // equivalent mutants: NewFeatureEntry rejects exactly what problems reports, so skipping this gate leaves the same result
		return FeatureEntry{}, false // equivalent mutant: a problem was reported, so no entry is built
	}
	entry, err := NewFeatureEntry(id, name, f)
	return entry, err == nil
}

// decodeFacts reads a list of facts (FR-055): {kind: grantsSubclass},
// {kind: grantsFeat, category}, {kind: alwaysPreparedSpells, grants}. A fact
// equal to an earlier one is duplicate at its element. With onlySpells (an
// option's facts, FR-064) any other kind is constraintViolation.
func decodeFacts(d *decoder, n node, p string, onlySpells bool) []FeatureFact {
	var out []FeatureFact
	seen := map[string]struct{}{}
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		fact, ok := decodeFact(d, el, ep)
		if !ok {
			return
		}
		if _, spells := fact.(AlwaysPreparedSpells); onlySpells && !spells {
			d.probs.add(ep, ProblemConstraintViolation)
			return
		}
		k := factKey(fact)
		if _, dup := seen[k]; dup {
			d.probs.add(ep, ProblemDuplicate)
			return
		}
		seen[k] = struct{}{}
		out = append(out, fact)
	})
	return out
}

func decodeFact(d *decoder, el node, p string) (FeatureFact, bool) {
	o, ok := d.object(el, p)
	if !ok {
		return nil, false // equivalent mutant: the non-object was reported, so no fact is built
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			switch s {
			case factKindGrantsSubclass, factKindGrantsFeat, factKindAlwaysPrepared:
				kind = s
			default:
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var res FeatureFact
	switch kind {
	case factKindGrantsSubclass:
		res = GrantsSubclass{}
	case factKindGrantsFeat:
		var category FeatCategory
		if n, cp, ok := o.field("category", true); ok {
			category, _ = enumField(d, n, cp, ParseFeatCategory)
		}
		if v, err := NewGrantsFeat(category); err == nil {
			res = v
		}
	case factKindAlwaysPrepared:
		if n, gp, ok := o.field("grants", true); ok {
			if v, err := NewAlwaysPreparedSpells(decodeSpellGrants(d, n, gp)); err == nil {
				res = v
			}
		}
	default:
		o.skip("category", "grants") // the kind is unknown: its fields are not unknown keys
	}
	o.finish()
	return res, res != nil
}

// decodeSpellGrants reads the non-empty grants of an always-prepared fact
// (FR-058): the level is 1 to 20 and strictly above the grant before it
// (invalidValue otherwise); the spell ids are non-empty and distinct.
func decodeSpellGrants(d *decoder, n node, p string) []SpellGrant {
	var out []SpellGrant
	last := 0
	d.list(n, p, 1, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		var level int64
		if n, lp, ok := o.field("level", true); ok {
			if level, ok = d.integer(n, lp, 1, ClassLevels); ok {
				if int(level) <= last {
					d.probs.add(lp, ProblemInvalidValue)
				}
				last = int(level)
			}
		}
		var ids []SpellID
		inGrant := map[SpellID]struct{}{}
		if n, sp, ok := o.field("spellIds", true); ok {
			d.list(n, sp, 1, noUpperBound, func(_ int, sel node, sep string) {
				if id, ok := decodeSpellID(d, sel, sep); ok {
					if _, dup := inGrant[id]; dup {
						d.probs.add(sep, ProblemDuplicate)
					}
					inGrant[id] = struct{}{}
					ids = append(ids, id)
				}
			})
		}
		o.finish()
		if g, err := NewSpellGrant(int(level), ids); err == nil {
			out = append(out, g)
		}
	})
	return out
}

// decodeFeatureOptions reads the options of a feature (FR-061 to FR-064).
func decodeFeatureOptions(d *decoder, n node, p string) []FeatureOption {
	var out []FeatureOption
	seen := map[FeatureOptionID]struct{}{}
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		if opt, ok := decodeFeatureOption(d, el, ep, seen); ok {
			out = append(out, opt)
		}
	})
	return out
}

func decodeFeatureOption(d *decoder, el node, p string, seen map[FeatureOptionID]struct{}) (FeatureOption, bool) {
	o, ok := d.object(el, p)
	if !ok {
		return FeatureOption{}, false // equivalent mutant: the non-object was reported, so no option is built
	}
	var id FeatureOptionID
	var name string
	var f FeatureOptionFields
	if n, ip, ok := o.field("id", true); ok {
		if raw, ok := d.id(n, ip); ok {
			id = FeatureOptionID(raw.String())
			if _, dup := seen[id]; dup {
				d.probs.add(ip, ProblemDuplicate)
			}
			seen[id] = struct{}{}
		}
	}
	if n, np, ok := o.field("name", true); ok {
		name, _ = d.name(n, np)
	}
	if n, tp, ok := o.field("text", true); ok {
		f.Text, _ = d.text(n, tp)
	}
	if n, pp, ok := o.field("prerequisite", false); ok {
		f.Prerequisite = decodeOptionPrerequisite(d, n, pp)
	}
	if n, cp, ok := o.field("pointCost", false); ok {
		v, _ := d.integer(n, cp, 1, ClassMaxCount)
		f.PointCost = int(v)
	}
	if n, rp, ok := o.field("repeatable", false); ok {
		if v, ok := d.boolean(n, rp); ok {
			if !v {
				d.probs.add(rp, ProblemInvalidValue)
			}
			f.Repeatable = v
		}
	}
	if n, fp, ok := o.field("facts", false); ok {
		f.Facts = decodeFacts(d, n, fp, true)
	}
	o.finish()
	opt, err := NewFeatureOption(id, name, f)
	return opt, err == nil
}

// decodeOptionPrerequisite reads {minClassLevel?, requiresOptionIds?, note?},
// at least one part (FR-062); {} is constraintViolation. That a required id
// names another option of the feature, and no cycle, is a rule between parts of
// the feature.
func decodeOptionPrerequisite(d *decoder, n node, p string) OptionPrerequisite {
	o, ok := d.object(n, p)
	if !ok {
		return OptionPrerequisite{}
	}
	var level int64
	var requires []FeatureOptionID
	var note string
	present := false
	if n, lp, ok := o.field("minClassLevel", false); ok {
		present = true
		level, _ = d.integer(n, lp, 1, ClassLevels)
	}
	if n, rp, ok := o.field("requiresOptionIds", false); ok {
		present = true
		requires = []FeatureOptionID{}
		inReq := map[FeatureOptionID]struct{}{}
		d.list(n, rp, 1, noUpperBound, func(_ int, el node, ep string) {
			if raw, ok := d.id(el, ep); ok {
				id := FeatureOptionID(raw.String())
				if _, dup := inReq[id]; dup {
					d.probs.add(ep, ProblemDuplicate)
				}
				inReq[id] = struct{}{}
				requires = append(requires, id)
			}
		})
	}
	if n, np, ok := o.field("note", false); ok {
		present = true
		note, _ = d.text(n, np)
	}
	o.finish()
	if !present {
		d.probs.add(p, ProblemConstraintViolation)
		return OptionPrerequisite{}
	}
	v, _ := NewOptionPrerequisite(int(level), requires, note)
	return v
}

// decodeTables reads the embedded tables of a feature (FR-066).
func decodeTables(d *decoder, n node, p string) []FeatureTable {
	var out []FeatureTable
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		var name string
		var columns []string
		var rows [][]TableCell
		if n, np, ok := o.field("name", true); ok {
			name, _ = d.name(n, np)
		}
		columnsBefore := d.probs.count
		if n, cp, ok := o.field("columns", true); ok {
			d.list(n, cp, 1, noUpperBound, func(_ int, hel node, hep string) {
				if h, ok := d.line(hel, hep); ok {
					columns = append(columns, h)
				}
			})
		}
		width := len(columns)
		if d.probs.count != columnsBefore { // equivalent mutant: the count never falls below before, so != and > agree
			width = -1 // the header is unknown: the rows are not measured against it
		}
		if n, rp, ok := o.field("rows", true); ok {
			rows = decodeTableRows(d, n, rp, width)
		}
		o.finish()
		if t, err := NewFeatureTable(name, columns, rows); err == nil {
			out = append(out, t)
		}
	})
	return out
}

// decodeTableRows reads the rows of a table. Each is as long as the header
// (width, or any length when it is -1): another length is outOfRange at the row
// and its cells are not read. A cell is a JSON string or integer, any other
// type wrongType, and a cell of a different type from the first of its column
// is wrongType at that cell (the first deviation of each column).
func decodeTableRows(d *decoder, n node, p string, width int) [][]TableCell {
	var rows [][]TableCell
	var text []bool // per column: the first cell was text
	var broken []bool
	d.list(n, p, 1, noUpperBound, func(_ int, el node, rp string) {
		lo, hi := width, width
		if width < 0 {
			lo, hi = 0, noUpperBound
		}
		var row []TableCell
		clean := true
		d.list(el, rp, lo, hi, func(j int, cel node, cp string) {
			cell, isText, ok := decodeTableCell(d, cel, cp)
			if !ok {
				clean = false
				return
			}
			fresh := j >= len(text) // equivalent mutants: a column whose first cell is j == len(text) or beyond is padded with its own kind, so no deviation is seen either way
			for len(text) <= j {    // a column whose earlier cells all failed has no first cell yet
				text, broken = append(text, isText), append(broken, false)
			}
			if !fresh && text[j] != isText && !broken[j] {
				broken[j] = true
				d.probs.add(cp, ProblemWrongType)
			}
			if text[j] != isText {
				clean = false
				return
			}
			row = append(row, cell)
		})
		if clean {
			rows = append(rows, row)
		}
	})
	return rows
}

// decodeTableCell reads one cell: a string (a text cell) or an integer.
func decodeTableCell(d *decoder, n node, p string) (TableCell, bool, bool) {
	switch n.kind() { //nolint:exhaustive // every other JSON type is wrongType.
	case nodeString:
		s, ok := d.text(n, p)
		if c, err := NewTableCellText(s); ok && err == nil { // equivalent mutant: a text that fails the cell rule failed the field rule too, so ok && and || agree
			return c, true, true
		}
		return nil, false, false // equivalent mutant: a problem was reported, so no table is built
	case nodeNumber:
		v, ok := d.integer(n, p, -ClassMaxCount, ClassMaxCount)
		return TableCellInt{n: int(v)}, false, ok
	}
	d.probs.add(p, ProblemWrongType)
	return nil, false, false // equivalent mutant: a problem was reported, so no table is built
}

// decodeSubclasses reads the subclasses of a class (FR-070); each valid
// subclass id joins the document-wide duplicate check, then its features'.
func decodeSubclasses(d *decoder, n node, p string, nested *classNested) []SubclassDefinition {
	var out []SubclassDefinition
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		h := o.header()
		if h.idOK {
			nested.subclasses = append(nested.subclasses, nestedID{group: nestedSubclass, path: ep + "/id", id: h.id.String()})
		}
		var features []FeatureEntry
		if n, fp, ok := o.field("features", true); ok {
			d.list(n, fp, 1, noUpperBound, func(_ int, fel node, fep string) {
				if e, ok := decodeFeatureEntry(d, fel, fep, &nested.features); ok {
					features = append(features, e)
				}
			})
		}
		o.finish()
		if s, err := NewSubclassDefinition(SubclassID(h.id.String()), h.name, h.source, features); err == nil {
			out = append(out, s)
		}
	})
	return out
}
