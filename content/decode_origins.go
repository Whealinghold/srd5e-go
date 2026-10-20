package content

import "github.com/Whealinghold/srd5e-go/ability"

// parseAbilityExact returns the ability spelled exactly s, as
// ability.Ability.String does ("strength"). Unlike ability.ParseAbility it is
// case sensitive, as every wire vocabulary is (schema FR-025).
func parseAbilityExact(s string) (ability.Ability, error) {
	for _, a := range ability.All() {
		if a.String() == s {
			return a, nil
		}
	}
	return 0, &ParseError{Vocabulary: "Ability", Value: s} // equivalent mutant: replaces the literal 0 with 0
}

// The wire names of the "kind" of a prerequisite and of a tool proficiency
// (schema FR-118, FR-119).
const (
	prerequisiteKindLevel        = "level"
	prerequisiteKindAbilityScore = "abilityScore"
	prerequisiteKindFeature      = "feature"
	toolKindSpecific             = "specific"
	toolKindChoice               = "choice"
)

// decodeBackground is the row of the backgrounds array
// (content-species-backgrounds-feats FR-024 to FR-032): id, name, source?,
// abilityScores, feat, skillProficiencies, toolProficiency, equipmentOptions.
// Whether the feat exists and is an Origin feat is ValidateSet's (FR-025). The
// row builds what it read and leaves problems to the framework, as the species
// row does.
func decodeBackground(o *object) decodedEntry {
	d := o.d
	h := o.header()
	var f BackgroundFields
	if n, p, ok := o.field("abilityScores", true); ok {
		f.AbilityScores, _ = enumSet(d, n, p, BackgroundAbilityScores, BackgroundAbilityScores, parseAbilityExact)
	}
	if n, p, ok := o.field("feat", true); ok {
		f.Feat = decodeBackgroundFeat(d, n, p)
	}
	if n, p, ok := o.field("skillProficiencies", true); ok {
		f.SkillProficiencies, _ = enumSet(d, n, p, BackgroundSkillProficiencies, BackgroundSkillProficiencies, ParseSkill)
	}
	if n, p, ok := o.field("toolProficiency", true); ok {
		f.ToolProficiency = decodeToolProficiency(d, n, p)
	}
	if n, p, ok := o.field("equipmentOptions", true); ok {
		f.EquipmentOptions = decodeEquipmentOptions(d, n, p)
	}
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if b, err := NewBackground(BackgroundID(h.id.String()), h.name, h.source, f); err == nil {
		res.entry = b
	}
	return res
}

func decodeBackgroundFeat(d *decoder, n node, p string) BackgroundFeatReference {
	o, ok := d.object(n, p)
	if !ok {
		return BackgroundFeatReference{}
	}
	var featID FeatID
	var choice string
	if n, fp, ok := o.field("featId", true); ok {
		if id, ok := d.id(n, fp); ok {
			featID = FeatID(id.String())
		}
	}
	if n, cp, ok := o.field("choice", false); ok {
		choice, _ = d.line(n, cp)
	}
	o.finish()
	r, _ := NewBackgroundFeatReference(featID, choice) // the zero reference on a problem, which NewBackground rejects
	return r
}

// decodeToolProficiency reads {kind, name} or {kind, category} (schema
// FR-119). An unknown kind is invalidValue at the kind. It returns nil when
// the proficiency could not be built.
func decodeToolProficiency(d *decoder, n node, p string) ToolProficiency {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			if s == toolKindSpecific || s == toolKindChoice {
				kind = s
			} else {
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var res ToolProficiency
	switch kind {
	case toolKindSpecific:
		var name string
		if n, np, ok := o.field("name", true); ok {
			name, _ = d.name(n, np)
		}
		if v, err := NewSpecificToolProficiency(name); err == nil {
			res = v
		}
	case toolKindChoice:
		var category string
		if n, cp, ok := o.field("category", true); ok {
			category, _ = d.text(n, cp)
		}
		if v, err := NewChoiceToolProficiency(category); err == nil {
			res = v
		}
	default:
		o.skip("name", "category") // the kind is unknown: its fields are not unknown keys
	}
	o.finish()
	return res
}

// decodeEquipmentOptions reads {"A": option, "B": option}, both required
// (FR-029); any other key is unknownField.
func decodeEquipmentOptions(d *decoder, n node, p string) EquipmentOptions {
	o, ok := d.object(n, p)
	if !ok {
		return EquipmentOptions{}
	}
	var a, b EquipmentOption
	if n, ap, ok := o.field("A", true); ok {
		a = decodeEquipmentOption(d, n, ap)
	}
	if n, bp, ok := o.field("B", true); ok {
		b = decodeEquipmentOption(d, n, bp)
	}
	o.finish()
	return NewEquipmentOptions(a, b)
}

// decodeEquipmentOption reads {items, gold} (FR-029, FR-030); the item
// quantity is outOfRange below 1.
func decodeEquipmentOption(d *decoder, n node, p string) EquipmentOption {
	o, ok := d.object(n, p)
	if !ok {
		return EquipmentOption{}
	}
	var items []EquipmentItem
	var gold int64
	if n, ip, ok := o.field("items", true); ok {
		d.list(n, ip, 0, noUpperBound, func(_ int, el node, ep string) {
			if item, ok := decodeEquipmentItem(d, el, ep); ok {
				items = append(items, item)
			}
		})
	}
	if n, gp, ok := o.field("gold", true); ok {
		gold, _ = d.integer(n, gp, 0, MaxEquipmentGold)
	}
	o.finish()
	opt, _ := NewEquipmentOption(items, int(gold)) // the zero option on a problem, dropped with the entry
	return opt
}

func decodeEquipmentItem(d *decoder, el node, p string) (EquipmentItem, bool) {
	o, ok := d.object(el, p)
	if !ok {
		return EquipmentItem{}, false // equivalent mutant: a zero item only makes NewEquipmentOption reject, and its problem is already reported
	}
	var name string
	var quantity int64
	if n, np, ok := o.field("name", true); ok {
		name, _ = d.name(n, np)
	}
	if n, qp, ok := o.field("quantity", true); ok {
		quantity, _ = d.integer(n, qp, 1, MaxEquipmentQuantity)
	}
	o.finish()
	item, err := NewEquipmentItem(name, int(quantity))
	return item, err == nil
}

// decodeFeat is the row of the feats array (FR-033 to FR-041, FR-063; SX-2,
// SX-4): id, name, source?, category, prerequisites?, benefits, repeatable,
// repeatableText?.
func decodeFeat(o *object) decodedEntry {
	d := o.d
	h := o.header()
	var f FeatFields
	if n, p, ok := o.field("category", true); ok {
		f.Category, _ = enumField(d, n, p, ParseFeatCategory)
	}
	if n, p, ok := o.field("prerequisites", false); ok {
		f.Prerequisites = decodePrerequisites(d, n, p)
	}
	if n, p, ok := o.field("benefits", true); ok {
		f.Benefits = decodeBenefits(d, n, p)
	}
	repeatableOK := false
	if n, p, ok := o.field("repeatable", true); ok {
		f.Repeatable, repeatableOK = d.boolean(n, p)
	}
	// repeatableText is required when repeatable is true (missingField) and
	// forbidden when it is false (constraintViolation); a repeatable that did
	// not decode leaves the rule unchecked.
	if n, p, ok := o.field("repeatableText", repeatableOK && f.Repeatable); ok {
		f.RepeatableText, _ = d.text(n, p)
		if repeatableOK && !f.Repeatable {
			d.probs.add(p, ProblemConstraintViolation)
		}
	}
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if v, err := NewFeat(FeatID(h.id.String()), h.name, h.source, f); err == nil {
		res.entry = v
	}
	return res
}

// decodeBenefits reads the non-empty benefits of a feat: {name?, text}.
func decodeBenefits(d *decoder, n node, path string) []FeatBenefit {
	var out []FeatBenefit
	d.list(n, path, 1, noUpperBound, func(_ int, el node, p string) {
		o, ok := d.object(el, p)
		if !ok {
			return
		}
		var name, text string
		if n, np, ok := o.field("name", false); ok {
			name, _ = d.name(n, np)
		}
		if n, tp, ok := o.field("text", true); ok {
			text, _ = d.text(n, tp)
		}
		o.finish()
		if b, err := NewFeatBenefit(name, text); err == nil {
			out = append(out, b)
		}
	})
	return out
}

// decodePrerequisites reads the optional prerequisites of a feat: a non-empty
// list (SX-4), each {kind: level | abilityScore | feature, ...} (schema
// FR-118).
func decodePrerequisites(d *decoder, n node, path string) []Prerequisite {
	var out []Prerequisite
	d.list(n, path, 1, noUpperBound, func(_ int, el node, p string) {
		if pre := decodePrerequisite(d, el, p); pre != nil { // equivalent mutant (to true): a nil prerequisite only makes NewFeat reject, and its problem is already reported
			out = append(out, pre)
		}
	})
	return out
}

// decodePrerequisite reads one prerequisite object. It returns nil when the
// prerequisite could not be built.
func decodePrerequisite(d *decoder, el node, p string) Prerequisite {
	o, ok := d.object(el, p)
	if !ok {
		return nil
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			switch s {
			case prerequisiteKindLevel, prerequisiteKindAbilityScore, prerequisiteKindFeature:
				kind = s
			default:
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var res Prerequisite
	switch kind {
	case prerequisiteKindLevel:
		var minimum int64
		if n, mp, ok := o.field("minimum", true); ok {
			minimum, _ = d.integer(n, mp, prerequisiteMinLevel, prerequisiteMaxLevel)
		}
		if v, err := NewLevelPrerequisite(int(minimum)); err == nil {
			res = v
		}
	case prerequisiteKindAbilityScore:
		var abilities []ability.Ability
		var minimum int64
		if n, ap, ok := o.field("abilities", true); ok {
			abilities, _ = enumSet(d, n, ap, 1, noUpperBound, parseAbilityExact)
		}
		if n, mp, ok := o.field("minimum", true); ok {
			minimum, _ = d.integer(n, mp, ability.MinScore, ability.MaxScore)
		}
		if v, err := NewAbilityScorePrerequisite(abilities, int(minimum)); err == nil {
			res = v
		}
	case prerequisiteKindFeature:
		var name string
		if n, np, ok := o.field("name", true); ok {
			name, _ = d.name(n, np)
		}
		if v, err := NewFeaturePrerequisite(name); err == nil {
			res = v
		}
	default:
		o.skip("minimum", "abilities", "name") // the kind is unknown: its fields are not unknown keys
	}
	o.finish()
	return res
}
