package content

import (
	"math"
	"slices"
)

// noUpperBound is the upper bound of a list that has none: the document size
// is the only limit (MaxDocumentBytes).
const noUpperBound = math.MaxInt

// The kind rows below read each field of an entry, report every problem they
// find, and then offer what they read to the constructor. They do not track
// whether a problem occurred: when one did, the framework builds no Set, so an
// entry (or a part) built from partial fields is dropped, and a constructor
// that rejects what the decoder accepted is reported as a bug of the row.

// decodeSpecies is the row of the species array (content-species-backgrounds-feats
// FR-010 to FR-023, FR-057, FR-060, FR-061; SX-1, SX-3): id, name, source?,
// creatureType, sizes, sizeText?, speed, traits, choices?.
func decodeSpecies(o *object) decodedEntry {
	d := o.d
	h := o.header()
	var f SpeciesFields
	if n, p, ok := o.field("creatureType", true); ok {
		f.CreatureType, _ = enumField(d, n, p, ParseCreatureType)
	}
	if n, p, ok := o.field("sizes", true); ok {
		f.Sizes = decodeSizes(d, n, p)
	}
	if n, p, ok := o.field("sizeText", false); ok {
		f.SizeText, _ = d.text(n, p)
	}
	if n, p, ok := o.field("speed", true); ok {
		v, _ := d.integer(n, p, 0, SpeciesMaxSpeed)
		f.Speed = int(v)
	}
	traitsOK := false // the traits are known only when they all decoded
	if n, p, ok := o.field("traits", true); ok {
		before := d.probs.count
		f.Traits = decodeNamedTexts(d, n, p)
		traitsOK = d.probs.count == before // equivalent mutant: the count never falls below before, so == and <= agree
	}
	if n, p, ok := o.field("choices", false); ok {
		f.Choices = decodeChoices(d, n, p, f.Traits, traitsOK)
	}
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if s, err := NewSpecies(SpeciesID(h.id.String()), h.name, h.source, f); err == nil {
		res.entry = s
	}
	return res
}

// decodeSizes reads the sizes of a species: one or two distinct members that
// are Small or Medium (FR-012). A bad length is one outOfRange and the
// elements are not examined; a size that is not allowed is invalidValue, a
// repeat duplicate, at the element.
func decodeSizes(d *decoder, n node, path string) []CreatureSize {
	var out []CreatureSize
	d.list(n, path, 1, SpeciesMaxSizes, func(_ int, el node, p string) {
		z, ok := enumField(d, el, p, ParseCreatureSize)
		switch {
		case !ok:
		case !sizeAllowed(z):
			d.probs.add(p, ProblemInvalidValue)
		case slices.Contains(out, z):
			d.probs.add(p, ProblemDuplicate)
		default:
			out = append(out, z)
		}
	})
	return out
}

// decodeNamedTexts reads a non-empty list of {name, text} objects with
// distinct names (FR-014, schema FR-116). A repeated name is duplicate at the
// later element's name.
func decodeNamedTexts(d *decoder, n node, path string) []NamedText {
	var out []NamedText
	seen := map[string]struct{}{}
	d.list(n, path, 1, noUpperBound, func(_ int, el node, p string) {
		o, ok := d.object(el, p)
		if !ok {
			return
		}
		var name, text string
		if n, np, ok := o.field("name", true); ok {
			if name, ok = d.name(n, np); ok {
				if _, repeated := seen[name]; repeated {
					d.probs.add(np, ProblemDuplicate)
				}
				seen[name] = struct{}{}
			}
		}
		if n, tp, ok := o.field("text", true); ok {
			text, _ = d.text(n, tp)
		}
		o.finish()
		if t, err := NewNamedText(name, text); err == nil {
			out = append(out, t)
		}
	})
	return out
}

// decodeChoices reads the optional choices of a species (SX-1). Each
// traitName must be the name of a trait of the species (FR-060); when the
// traits had a problem the link is not checked, so one bad trait does not
// cascade.
func decodeChoices(d *decoder, n node, path string, traits []NamedText, traitsOK bool) []SpeciesChoice {
	var out []SpeciesChoice
	d.list(n, path, 0, noUpperBound, func(_ int, el node, p string) {
		o, ok := d.object(el, p)
		if !ok {
			return
		}
		var traitName string
		if n, tp, ok := o.field("traitName", true); ok {
			if traitName, ok = d.name(n, tp); ok && traitsOK && !slices.ContainsFunc(traits, func(t NamedText) bool { return t.name == traitName }) {
				d.probs.add(tp, ProblemConstraintViolation)
			}
		}
		var options []ChoiceOption
		if n, op, ok := o.field("options", true); ok {
			options = decodeOptions(d, n, op)
		}
		o.finish()
		if c, err := NewSpeciesChoice(traitName, options); err == nil {
			out = append(out, c)
		}
	})
	return out
}

// decodeOptions reads the options of a choice: at least one, with distinct
// names, all of the variant of the first option that has a valid kind
// (FR-016). A variant that differs is constraintViolation at its kind.
func decodeOptions(d *decoder, n node, path string) []ChoiceOption {
	var out []ChoiceOption
	first := ""
	names := map[string]struct{}{}
	d.list(n, path, 1, noUpperBound, func(_ int, el node, p string) {
		if opt := decodeOption(d, el, p, &first, names); opt != nil { // equivalent mutant: appending a nil option only makes NewSpeciesChoice reject, which drops the choice with its problems already reported
			out = append(out, opt)
		}
	})
	return out
}

// decodeOption reads one option object: kind, name, then the fields of its
// variant. first holds the variant of the first valid kind seen so far. It
// returns nil when the option could not be built.
func decodeOption(d *decoder, el node, p string, first *string, names map[string]struct{}) ChoiceOption {
	o, ok := d.object(el, p)
	if !ok {
		return nil
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			switch s {
			case optionKindDamageType, optionKindLineage, optionKindText:
				kind = s
				if *first == "" { // equivalent mutant: a string is never below "", so == and <= agree
					*first = s
				} else if s != *first {
					d.probs.add(kp, ProblemConstraintViolation)
				}
			default:
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var name string
	if n, np, ok := o.field("name", true); ok {
		if name, ok = d.name(n, np); ok {
			if _, repeated := names[name]; repeated {
				d.probs.add(np, ProblemDuplicate)
			}
			names[name] = struct{}{}
		}
	}
	var opt ChoiceOption
	switch kind {
	case optionKindDamageType:
		opt = decodeDamageTypeOption(o, name)
	case optionKindLineage:
		opt = decodeLineageOption(o, name)
	case optionKindText:
		opt = decodeTextOption(o, name)
	default:
		o.skip("damageType", "levelBenefits", "text", "spellIds") // the variant is unknown: its fields are not unknown keys
	}
	o.finish()
	return opt
}

func decodeDamageTypeOption(o *object, name string) ChoiceOption {
	var dt DamageType
	if n, p, ok := o.field("damageType", true); ok {
		dt, _ = enumField(o.d, n, p, ParseDamageType)
	}
	if opt, err := NewDamageTypeOption(name, dt); err == nil {
		return opt
	}
	return nil
}

func decodeTextOption(o *object, name string) ChoiceOption {
	var text string
	var ids []SpellID
	if n, p, ok := o.field("text", true); ok {
		text, _ = o.d.text(n, p)
	}
	if n, p, ok := o.field("spellIds", false); ok {
		o.d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
			if id, ok := decodeSpellID(o.d, el, ep); ok {
				ids = append(ids, id)
			}
		})
	}
	if opt, err := NewTextOption(name, text, ids); err == nil {
		return opt
	}
	return nil
}

func decodeLineageOption(o *object, name string) ChoiceOption {
	var benefits []LevelBenefit
	if n, p, ok := o.field("levelBenefits", true); ok {
		benefits = decodeLevelBenefits(o.d, n, p)
	}
	if opt, err := NewLineageOption(name, benefits); err == nil {
		return opt
	}
	return nil
}

// decodeLevelBenefits reads the three level benefits of a lineage option
// (FR-061). A length other than three is one outOfRange; when all three
// decoded, a repeated level is duplicate at the later benefit's level and
// levels in another order are constraintViolation at the list.
func decodeLevelBenefits(d *decoder, n node, path string) []LevelBenefit {
	var out []LevelBenefit
	d.list(n, path, len(lineageLevels()), len(lineageLevels()), func(_ int, el node, p string) {
		if b, ok := decodeLevelBenefit(d, el, p); ok {
			out = append(out, b)
		}
	})
	if len(out) != len(lineageLevels()) { // equivalent mutant: the list bounds cap out at three, so != and < agree
		return out
	}
	levels := make([]int, len(out))
	for i, b := range out {
		levels[i] = b.level
	}
	//nolint:exhaustive // lineageLevelProblem returns only no problem, BlockDuplicate or BlockConstraint.
	switch problem, i := lineageLevelProblem(levels); problem {
	case BlockDuplicate:
		d.probs.add(indexPath(path, i)+"/level", ProblemDuplicate)
	case BlockConstraint:
		d.probs.add(path, ProblemConstraintViolation)
	default: // no problem: the levels are 1, 3, 5
	}
	return out
}

func decodeLevelBenefit(d *decoder, el node, p string) (LevelBenefit, bool) {
	o, ok := d.object(el, p)
	if !ok {
		return LevelBenefit{}, false
	}
	var level int64
	var text string
	var spell SpellID
	if n, lp, ok := o.field("level", true); ok {
		if level, ok = d.integer(n, lp, -MaxSafeInteger, MaxSafeInteger); ok && (level < 1 || level > 5 || !validLineageLevel(int(level))) {
			// Tested as int64 first: int(level) would wrap on 32-bit platforms. Equivalent mutants on 64-bit hosts, where
			// validLineageLevel alone rejects the same values: each || to && survives.
			d.probs.add(lp, ProblemInvalidValue)
			level = 0
		}
	}
	if n, tp, ok := o.field("text", true); ok {
		text, _ = d.text(n, tp)
	}
	if n, sp, ok := o.field("spellId", true); ok {
		spell, _ = decodeSpellID(d, n, sp)
	}
	o.finish()
	b, err := NewLevelBenefit(int(level), text, spell)
	return b, err == nil
}

// decodeSpellID reads the id of a spell (a site of schema FR-126): the id rule
// of FR-030. It is resolved only by ValidateSet.
func decodeSpellID(d *decoder, n node, path string) (SpellID, bool) {
	id, ok := d.id(n, path)
	return SpellID(id.String()), ok
}
