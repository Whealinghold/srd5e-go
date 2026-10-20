package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// The wire names of the "kind" tags of the class shapes (content-classes
// FR-011, FR-019, FR-028, FR-043, FR-055).
const (
	abilityKindAllOf          = "allOf"
	abilityKindAnyOf          = "anyOf"
	skillOptionsListed        = "listed"
	skillOptionsAny           = "any"
	spellcastingKindSlots     = "slots"
	spellcastingKindPact      = "pact"
	rowKindSubclassFeature    = "subclassFeature"
	factKindGrantsSubclass    = "grantsSubclass"
	factKindGrantsFeat        = "grantsFeat"
	factKindAlwaysPrepared    = "alwaysPreparedSpells"
	classEquipmentOptionLabel = "C"
)

// classNested collects the valid ids of the subclasses and features of the
// class being decoded, for the document-wide duplicate check (FR-004).
type classNested struct{ subclasses, features []nestedID }

// decodeClass is the row of the classes array (content-classes FR-002 to
// FR-077): id, name, source?, primaryAbility, hitPointDie,
// savingThrowProficiencies, skillChoice, weaponProficiencies,
// toolProficiencies, armorTraining, startingEquipment, multiclassGrants,
// spellcasting?, columns, progression, features, subclasses. The rules between
// parts (ClassDefinition.problems) are reported only when every field decoded
// with no problem, so a part dropped for a problem cannot cascade into bogus
// reference problems.
func decodeClass(o *object) decodedEntry {
	d := o.d
	before := d.probs.count
	h := o.header()
	var f ClassFields
	var nested classNested
	if n, p, ok := o.field("primaryAbility", true); ok {
		f.PrimaryAbility = decodeAbilityRequirement(d, n, p)
	}
	if n, p, ok := o.field("hitPointDie", true); ok {
		f.HitPointDie, _ = enumField(d, n, p, ParseHitPointDie)
	}
	if n, p, ok := o.field("savingThrowProficiencies", true); ok {
		f.SavingThrowProficiencies, _ = enumSet(d, n, p, 1, len(ability.All()), parseAbilityExact)
	}
	if n, p, ok := o.field("skillChoice", true); ok {
		f.SkillChoice = decodeSkillChoice(d, n, p)
	}
	if n, p, ok := o.field("weaponProficiencies", true); ok {
		f.WeaponProficiencies = decodeWeaponProficiencies(d, n, p)
	}
	if n, p, ok := o.field("toolProficiencies", true); ok {
		f.ToolProficiencies = decodeClassTools(d, n, p)
	}
	if n, p, ok := o.field("armorTraining", true); ok {
		f.ArmorTraining, _ = enumSet(d, n, p, 0, len(armorTrainingNames()), ParseArmorTrainingCategory)
	}
	if n, p, ok := o.field("startingEquipment", true); ok {
		f.StartingEquipment = decodeStartingEquipment(d, n, p)
	}
	if n, p, ok := o.field("multiclassGrants", true); ok {
		f.MulticlassGrants = decodeMulticlassGrants(d, n, p)
	}
	if n, p, ok := o.field("spellcasting", false); ok {
		f.Spellcasting = decodeSpellcastingTrait(d, n, p)
	}
	columnsBefore := d.probs.count
	if n, p, ok := o.field("columns", true); ok {
		f.Columns = decodeColumns(d, n, p)
	}
	columnsOK := d.probs.count == columnsBefore // equivalent mutant: the count never falls below before, so == and <= agree
	if n, p, ok := o.field("progression", true); ok {
		f.Progression = decodeProgression(d, n, p, f.Columns, columnsOK)
	}
	if n, p, ok := o.field("features", true); ok {
		f.Features = decodeFeatureEntries(d, n, p, &nested.features)
	}
	if n, p, ok := o.field("subclasses", true); ok {
		f.Subclasses = decodeSubclasses(d, n, p, &nested)
	}
	res := decodedEntry{id: h.id, idOK: h.idOK, nested: append(nested.subclasses, nested.features...)}
	if d.probs.count != before { // equivalent mutant: the count never falls below before, so != and > agree
		return res
	}
	c := ClassDefinition{id: h.id, name: h.name, source: h.source, f: f}
	for _, is := range c.problems() {
		d.probs.add(o.path+"/"+is.path, is.kind)
	}
	if d.probs.count != before { // equivalent mutants: NewClassDefinition rejects exactly what problems reports, so skipping this gate leaves the same result
		return res
	}
	if e, err := NewClassDefinition(ClassID(h.id.String()), h.name, h.source, f); err == nil {
		res.entry = e
	}
	return res
}

// decodeAbilityRequirement reads {kind, abilities} (FR-011 to FR-013). An
// unknown kind is invalidValue at the kind and its abilities are not read. It
// returns nil when the requirement could not be built.
func decodeAbilityRequirement(d *decoder, n node, p string) AbilityRequirement {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			if s == abilityKindAllOf || s == abilityKindAnyOf {
				kind = s
			} else {
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var res AbilityRequirement
	readAbilities := func(lo int) []ability.Ability {
		var out []ability.Ability
		if n, ap, ok := o.field("abilities", true); ok {
			out, _ = enumSet(d, n, ap, lo, len(ability.All()), parseAbilityExact)
		}
		return out
	}
	switch kind {
	case abilityKindAllOf:
		if v, err := NewAllOf(readAbilities(1)...); err == nil {
			res = v
		}
	case abilityKindAnyOf:
		if v, err := NewAnyOf(readAbilities(2)...); err == nil {
			res = v
		}
	default:
		o.skip("abilities") // the kind is unknown: its field is not an unknown key
	}
	o.finish()
	return res
}

// decodeSkillChoice reads {picks, options: {kind, skills?}} (FR-019). A listed
// set needs at least `picks` skills, reported as outOfRange at the skills.
func decodeSkillChoice(d *decoder, n node, p string) SkillChoice {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	var picks int64
	picksOK := false
	if n, pp, ok := o.field("picks", true); ok {
		picks, picksOK = d.integer(n, pp, 1, ClassMaxCount)
	}
	var res SkillChoice
	if n, op, ok := o.field("options", true); ok {
		res = decodeSkillOptions(d, n, op, int(picks), picksOK)
	}
	o.finish()
	return res
}

func decodeSkillOptions(d *decoder, n node, p string, picks int, picksOK bool) SkillChoice {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			if s == skillOptionsListed || s == skillOptionsAny {
				kind = s
			} else {
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var res SkillChoice
	switch kind {
	case skillOptionsAny:
		if v, err := NewSkillChoiceAny(picks); err == nil && picksOK {
			res = v
		}
	case skillOptionsListed:
		var skills []Skill
		clean := false
		if n, sp, ok := o.field("skills", true); ok {
			skills, clean = enumSet(d, n, sp, 1, noUpperBound, ParseSkill)
			if clean && picksOK && len(skills) < picks {
				d.probs.add(sp, ProblemOutOfRange)
			}
		}
		if v, err := NewSkillChoiceListed(picks, skills); err == nil { // a partly read list is never used: a problem was reported
			res = v
		}
	default:
		o.skip("skills") // the kind is unknown: its field is not an unknown key
	}
	o.finish()
	return res
}

// decodeWeaponProficiencies reads {category, requiresAnyProperty?} elements,
// at most one per category (FR-021); a repeat is duplicate at the later
// element.
func decodeWeaponProficiencies(d *decoder, n node, p string) []WeaponProficiency {
	var out []WeaponProficiency
	var seen []WeaponTrainingCategory
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		var cat WeaponTrainingCategory
		if n, cp, ok := o.field("category", true); ok {
			if cat, ok = enumField(d, n, cp, ParseWeaponTrainingCategory); ok {
				if slices.Contains(seen, cat) {
					d.probs.add(ep, ProblemDuplicate)
				}
				seen = append(seen, cat)
			}
		}
		var props []WeaponProperty
		if n, pp, ok := o.field("requiresAnyProperty", false); ok {
			props, _ = enumSet(d, n, pp, 1, noUpperBound, ParseWeaponProperty)
		}
		o.finish()
		if w, err := NewWeaponProficiency(cat, props); err == nil {
			out = append(out, w)
		}
	})
	return out
}

// decodeClassTools reads {kind: specific, name} and {kind: choice, categories,
// picks} elements (FR-023).
func decodeClassTools(d *decoder, n node, p string) []ClassToolProficiency {
	var out []ClassToolProficiency
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		if t := decodeClassTool(d, el, ep); t != nil { // equivalent mutant: a nil proficiency is appended only after a reported problem, which drops the entry
			out = append(out, t)
		}
	})
	return out
}

func decodeClassTool(d *decoder, el node, p string) ClassToolProficiency {
	o, ok := d.object(el, p)
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
	var res ClassToolProficiency
	switch kind {
	case toolKindSpecific:
		var name string
		if n, np, ok := o.field("name", true); ok {
			name, _ = d.name(n, np)
		}
		if v, err := NewClassToolProficiencySpecific(name); err == nil {
			res = v
		}
	case toolKindChoice:
		var categories []string
		inCat := map[string]struct{}{}
		if n, cp, ok := o.field("categories", true); ok {
			d.list(n, cp, 1, noUpperBound, func(_ int, cel node, cep string) {
				if s, ok := d.line(cel, cep); ok {
					if _, dup := inCat[s]; dup {
						d.probs.add(cep, ProblemDuplicate)
					}
					inCat[s] = struct{}{}
					categories = append(categories, s)
				}
			})
		}
		var picks int64
		if n, pp, ok := o.field("picks", true); ok {
			picks, _ = d.integer(n, pp, 1, ClassMaxCount)
		}
		if v, err := NewClassToolProficiencyChoice(categories, int(picks)); err == nil {
			res = v
		}
	default:
		o.skip("name", "categories", "picks") // the kind is unknown: its fields are not unknown keys
	}
	o.finish()
	return res
}

// decodeStartingEquipment reads {"A": option, "B": option, "C"?: option}
// (FR-024); any other key is unknownField.
func decodeStartingEquipment(d *decoder, n node, p string) ClassStartingEquipment {
	o, ok := d.object(n, p)
	if !ok {
		return ClassStartingEquipment{}
	}
	var a, b, c ClassEquipmentOption
	if n, ap, ok := o.field("A", true); ok {
		a = decodeClassEquipmentOption(d, n, ap)
	}
	if n, bp, ok := o.field("B", true); ok {
		b = decodeClassEquipmentOption(d, n, bp)
	}
	var rest []ClassEquipmentOption
	if n, cp, ok := o.field(classEquipmentOptionLabel, false); ok {
		c = decodeClassEquipmentOption(d, n, cp)
		rest = append(rest, c)
	}
	o.finish()
	e, _ := NewClassStartingEquipment(a, b, rest...) // the zero equipment on a problem, which NewClassDefinition rejects
	return e
}

// decodeClassEquipmentOption reads {items: [{name, quantity}], gold}.
func decodeClassEquipmentOption(d *decoder, n node, p string) ClassEquipmentOption {
	o, ok := d.object(n, p)
	if !ok {
		return ClassEquipmentOption{}
	}
	var items []ClassEquipmentItem
	if n, ip, ok := o.field("items", true); ok {
		d.list(n, ip, 0, noUpperBound, func(_ int, el node, ep string) {
			io, ok := d.object(el, ep)
			if !ok {
				return
			}
			var name string
			var quantity int64
			if n, np, ok := io.field("name", true); ok {
				name, _ = d.name(n, np)
			}
			if n, qp, ok := io.field("quantity", true); ok {
				quantity, _ = d.integer(n, qp, 1, ClassMaxCount)
			}
			io.finish()
			if it, err := NewClassEquipmentItem(name, int(quantity)); err == nil {
				items = append(items, it)
			}
		})
	}
	var gold int64
	if n, gp, ok := o.field("gold", true); ok {
		gold, _ = d.integer(n, gp, 0, ClassMaxCount)
	}
	o.finish()
	opt, _ := NewClassEquipmentOption(items, int(gold))
	return opt
}

// decodeMulticlassGrants reads {weaponProficiencies, armorTraining,
// toolProficiencies, skillChoice?} (FR-026). The subset rule against the core
// traits is a rule between parts of the class.
func decodeMulticlassGrants(d *decoder, n node, p string) MulticlassGrants {
	o, ok := d.object(n, p)
	if !ok {
		return MulticlassGrants{}
	}
	var weapons []WeaponProficiency
	var armor []ArmorTrainingCategory
	var tools []ClassToolProficiency
	var skills SkillChoice
	if n, wp, ok := o.field("weaponProficiencies", true); ok {
		weapons = decodeWeaponProficiencies(d, n, wp)
	}
	if n, ap, ok := o.field("armorTraining", true); ok {
		armor, _ = enumSet(d, n, ap, 0, len(armorTrainingNames()), ParseArmorTrainingCategory)
	}
	if n, tp, ok := o.field("toolProficiencies", true); ok {
		tools = decodeClassTools(d, n, tp)
	}
	if n, sp, ok := o.field("skillChoice", false); ok {
		skills = decodeSkillChoice(d, n, sp)
	}
	o.finish()
	g, _ := NewMulticlassGrants(weapons, armor, tools, skills)
	return g
}

// decodeSpellcastingTrait reads {kind: slots, ability, cantrips} or {kind:
// pact, ability} (FR-028). An unknown kind is invalidValue at the kind.
func decodeSpellcastingTrait(d *decoder, n node, p string) ClassSpellcastingTrait {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	kind := ""
	if n, kp, ok := o.field("kind", true); ok {
		if s, ok := d.str(n, kp); ok {
			if s == spellcastingKindSlots || s == spellcastingKindPact {
				kind = s
			} else {
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	var a ability.Ability
	var cantrips bool
	if kind == "" { // equivalent mutant: a string is never below "", so == and <= agree
		o.skip("ability", "cantrips") // the kind is unknown: its fields are not unknown keys
	} else if n, ap, ok := o.field("ability", true); ok {
		a, _ = enumField(d, n, ap, parseAbilityExact)
	}
	var res ClassSpellcastingTrait
	switch kind {
	case spellcastingKindSlots:
		if n, cp, ok := o.field("cantrips", true); ok {
			cantrips, _ = d.boolean(n, cp)
		}
		if v, err := NewSlotSpellcastingTrait(a, cantrips); err == nil {
			res = v
		}
	case spellcastingKindPact:
		if v, err := NewPactSpellcastingTrait(a); err == nil {
			res = v
		}
	}
	o.finish()
	return res
}
