package content

import "github.com/Whealinghold/srd5e-go/dice"

// decodeColumns reads the {id, header, kind} declarations (FR-036); a repeated
// id is duplicate at the later id.
func decodeColumns(d *decoder, n node, p string) []ClassColumn {
	var out []ClassColumn
	seen := map[ClassColumnID]struct{}{}
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		var id ClassColumnID
		if n, ip, ok := o.field("id", true); ok {
			if raw, ok := d.id(n, ip); ok {
				id = ClassColumnID(raw.String())
				if _, dup := seen[id]; dup {
					d.probs.add(ip, ProblemDuplicate)
				}
				seen[id] = struct{}{}
			}
		}
		var header string
		var kind ClassColumnKind
		if n, hp, ok := o.field("header", true); ok {
			header, _ = d.line(n, hp)
		}
		if n, kp, ok := o.field("kind", true); ok {
			kind, _ = enumField(d, n, kp, ParseClassColumnKind)
		}
		o.finish()
		if c, err := NewClassColumn(id, header, kind); err == nil {
			out = append(out, c)
		}
	})
	return out
}

// decodeProgression reads exactly 20 rows (FR-037). columns are the declared
// columns, read only when they decoded cleanly (columnsOK).
func decodeProgression(d *decoder, n node, p string, columns []ClassColumn, columnsOK bool) []ProgressionRow {
	var out []ProgressionRow
	d.list(n, p, ClassLevels, ClassLevels, func(_ int, el node, ep string) {
		if r, ok := decodeRow(d, el, ep, columns, columnsOK); ok {
			out = append(out, r)
		}
	})
	return out
}

// decodeRow reads {level, proficiencyBonus, features, spellcasting?, values}
// (FR-038 to FR-043).
func decodeRow(d *decoder, el node, p string, columns []ClassColumn, columnsOK bool) (ProgressionRow, bool) {
	o, ok := d.object(el, p)
	if !ok {
		return ProgressionRow{}, false // equivalent mutant: a row that is not an object was reported, and the entry is dropped with it
	}
	var level, bonus int64
	var features []RowFeature
	var spell SpellcastingRow
	var values map[ClassColumnID]ColumnValue
	if n, lp, ok := o.field("level", true); ok {
		level, _ = d.integer(n, lp, 1, ClassLevels)
	}
	if n, bp, ok := o.field("proficiencyBonus", true); ok {
		bonus, _ = d.integer(n, bp, 1, ClassMaxCount)
	}
	if n, fp, ok := o.field("features", true); ok {
		features = decodeRowFeatures(d, n, fp)
	}
	if n, sp, ok := o.field("spellcasting", false); ok {
		spell = decodeSpellcastingRow(d, n, sp)
	}
	if n, vp, ok := o.field("values", true); ok {
		values = decodeValues(d, n, vp, columns, columnsOK)
	}
	o.finish()
	r, err := NewProgressionRow(int(level), int(bonus), features, spell, values)
	return r, err == nil
}

// decodeRowFeatures reads the Class Features cell (FR-043): {featureId,
// annotation?} or {kind: subclassFeature}, at most one marker (a second is
// duplicate at its element).
func decodeRowFeatures(d *decoder, n node, p string) []RowFeature {
	var out []RowFeature
	marker := false
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		if _, tagged := o.index["kind"]; tagged {
			kn, kp, _ := o.field("kind", true)
			if s, ok := d.str(kn, kp); !ok || s != rowKindSubclassFeature {
				if ok {
					d.probs.add(kp, ProblemInvalidValue)
				}
				o.skip("featureId", "annotation")
				o.finish()
				return
			}
			if marker {
				d.probs.add(ep, ProblemDuplicate)
			}
			marker = true
			o.finish()
			out = append(out, RowSubclassFeature{})
			return
		}
		var id FeatureID
		var annotation string
		if n, ip, ok := o.field("featureId", true); ok {
			if raw, ok := d.id(n, ip); ok {
				id = FeatureID(raw.String())
			}
		}
		if n, ap, ok := o.field("annotation", false); ok {
			annotation, _ = d.line(n, ap)
		}
		o.finish()
		if f, err := NewRowClassFeature(id, annotation); err == nil {
			out = append(out, f)
		}
	})
	return out
}

// decodeSpellcastingRow reads the spellcasting part of a row (FR-031, FR-032).
// A row with a "slotLevel" is the Pact Magic variant, any other the slot
// variant; whether it suits the class is a rule between parts.
func decodeSpellcastingRow(d *decoder, n node, p string) SpellcastingRow {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	if _, pact := o.index["slotLevel"]; pact {
		return decodePactRow(d, o)
	}
	var cantrips, prepared int64
	hasCantrips := false
	if n, cp, ok := o.field("cantrips", false); ok {
		cantrips, _ = d.integer(n, cp, 0, ClassMaxCount)
		hasCantrips = true
	}
	if n, pp, ok := o.field("preparedSpells", true); ok {
		prepared, _ = d.integer(n, pp, 0, ClassMaxCount)
	}
	var slots []int
	if n, sp, ok := o.field("spellSlots", true); ok {
		d.list(n, sp, ClassSpellLevels, ClassSpellLevels, func(_ int, el node, ep string) {
			v, _ := d.integer(el, ep, 0, ClassMaxCount)
			slots = append(slots, int(v))
		})
	}
	o.finish()
	if r, err := slotRow(int(cantrips), hasCantrips, int(prepared), slots); err == nil {
		return r
	}
	return nil
}

func decodePactRow(d *decoder, o *object) SpellcastingRow {
	var cantrips, prepared, slots, slotLevel int64
	if n, p, ok := o.field("cantrips", true); ok {
		cantrips, _ = d.integer(n, p, 0, ClassMaxCount)
	}
	if n, p, ok := o.field("preparedSpells", true); ok {
		prepared, _ = d.integer(n, p, 0, ClassMaxCount)
	}
	if n, p, ok := o.field("spellSlots", true); ok {
		slots, _ = d.integer(n, p, 1, ClassMaxCount)
	}
	if n, p, ok := o.field("slotLevel", true); ok {
		slotLevel, _ = d.integer(n, p, 1, ClassSpellLevels)
	}
	o.finish()
	if r, err := NewPactSpellcastingRow(int(cantrips), int(prepared), int(slots), int(slotLevel)); err == nil {
		return r
	}
	return nil
}

// decodeValues reads the "values" object of a row against the declared
// columns (FR-039, FR-040): one key per column in declaration order, a missing
// one missingField, an undeclared one unknownField. When the columns could not
// be read the values are skipped.
func decodeValues(d *decoder, n node, p string, columns []ClassColumn, columnsOK bool) map[ClassColumnID]ColumnValue {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	if !columnsOK {
		o.skipAll()
		return nil
	}
	values := make(map[ClassColumnID]ColumnValue, len(columns))
	for _, col := range columns {
		if n, vp, ok := o.field(string(col.id), true); ok {
			if v, ok := decodeColumnValue(d, n, vp, col.kind); ok {
				values[col.id] = v
			}
		}
	}
	o.finish()
	return values
}

// decodeColumnValue reads one column value of kind (FR-040): the string "—"
// for any kind, otherwise a JSON type that suits the kind. Any other string is
// invalidValue.
func decodeColumnValue(d *decoder, n node, p string, kind ClassColumnKind) (ColumnValue, bool) {
	if n.kind() == nodeString {
		s, ok := d.str(n, p)
		if !ok {
			return nil, false
		}
		if s != ColumnNoneText {
			d.probs.add(p, ProblemInvalidValue)
			return nil, false
		}
		return ColumnNone{}, true
	}
	if kind == ColumnKindDice {
		return decodeColumnDice(d, n, p)
	}
	v, ok := d.integer(n, p, 0, ClassMaxCount)
	if !ok {
		return nil, false
	}
	switch kind { //nolint:exhaustive // dice is handled above and kind is valid.
	case ColumnKindCount:
		return ColumnCount{n: int(v)}, true
	case ColumnKindBonus:
		return ColumnBonus{n: int(v)}, true
	}
	return ColumnFeetBonus{n: int(v)}, true
}

func decodeColumnDice(d *decoder, n node, p string) (ColumnValue, bool) {
	o, ok := d.object(n, p)
	if !ok {
		return nil, false
	}
	var count, sides int64
	if n, cp, ok := o.field("count", true); ok {
		count, _ = d.integer(n, cp, 1, ClassMaxCount)
	}
	if n, sp, ok := o.field("sides", true); ok {
		sides, _ = d.integer(n, sp, dice.MinSides, dice.MaxSides)
	}
	o.finish()
	v, err := NewColumnDice(int(count), int(sides))
	return v, err == nil
}
