package content

// This file holds the rows of the tools and adventuringGear arrays
// (content-equipment FR-029 to FR-031, FR-033 to FR-042, FR-113 to FR-117).

// The wire names of the Craft reference kinds (FR-031).
const (
	craftKindItem = "item"
	craftKindSet  = "set"
)

// decodeTool is the row of the tools array: id, name, source?, then by what
// the entry is. A variant has variantOf, price and weight, and none of the
// fields of its family; a family has variants; any other tool is standalone.
// The rest: toolGroup, ability, utilize, craft?, variants?, weightPounds?,
// cost (FR-029, FR-113).
func decodeTool(o *object) decodedEntry {
	d := o.d
	h := o.header()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if o.has("variantOf") {
		return decodeToolVariant(o, h, res)
	}
	// A field that failed leaves its zero value, which the constructor
	// rejects, and its problem makes the framework drop the entry: no flag
	// tracks the failures.
	group := fieldEnum(o, "toolGroup", parseToolGroup)
	ab := fieldEnum(o, "ability", parseAbilityExact)
	var utilize []ToolUtilizeAction
	if n, p, has := o.field("utilize", true); has {
		utilize = decodeUtilize(d, n, p)
	}
	var craft []CraftReference
	if n, p, has := o.field("craft", false); has {
		craft = decodeCraft(d, n, p)
	}
	var variants []ItemID
	if n, p, has := o.field("variants", false); has {
		variants = decodeItemIDs(d, n, p, 1)
	}
	weight := o.weight(true, false)
	price := o.price(o.has("variants"))
	id := ItemID(h.id.String())
	if o.has("variants") {
		if t, err := NewToolFamily(id, h.name, h.source, group, ab, utilize, craft, variants, price, weight); err == nil {
			res.entry = t
		}
		return res
	}
	fixed, _ := price.(FixedCost)
	if t, err := NewStandaloneTool(id, h.name, h.source, group, ab, utilize, craft, fixed, weight); err == nil {
		res.entry = t
	}
	return res
}

// decodeToolVariant reads the rest of a tool with variantOf: a field of a
// family or standalone tool on it is constraintViolation (FR-029, FR-113).
func decodeToolVariant(o *object, h entryHeader, res decodedEntry) decodedEntry {
	for _, key := range []string{"toolGroup", "ability", "utilize", "craft", "variants"} {
		if _, p, has := o.field(key, false); has {
			o.d.probs.add(p, ProblemConstraintViolation)
		}
	}
	family := o.reqID("variantOf")
	weight := o.weight(true, false)
	price := o.fixedPrice()
	if t, err := NewToolVariant(ItemID(h.id.String()), h.name, h.source, family, price, weight); err == nil {
		res.entry = t
	}
	return res
}

// decodeUtilize reads the non-empty Utilize actions {description, dc}. An
// action that failed is left out (its problem is reported).
func decodeUtilize(d *decoder, n node, path string) []ToolUtilizeAction {
	d.arrayLen(n, path, 1, noUpperBound)
	var out []ToolUtilizeAction
	d.each(n, path, func(_ int, el node, p string) {
		o, ok := d.object(el, p)
		if !ok {
			return
		}
		description := o.reqText("description")
		dc := o.reqInt("dc", 1, equipmentMaxInt)
		o.finish()
		if a, err := NewToolUtilizeAction(description, dc); err == nil {
			out = append(out, a)
		}
	})
	return out
}

// decodeCraft reads a Craft list of {kind: item, id} and {kind: set, set,
// except?} (FR-031). A reference that failed is left out.
func decodeCraft(d *decoder, n node, path string) []CraftReference {
	d.arrayLen(n, path, 0, noUpperBound)
	var out []CraftReference
	d.each(n, path, func(_ int, el node, p string) {
		if r := decodeCraftReference(d, el, p); r != nil {
			out = append(out, r)
		}
	})
	return out
}

// decodeCraftReference reads one Craft reference; nil when it failed.
func decodeCraftReference(d *decoder, el node, p string) CraftReference {
	o, ok := d.object(el, p)
	if !ok {
		return nil
	}
	defer o.finish()
	kind := ""
	if n, kp, has := o.field("kind", true); has {
		if s, sok := d.str(n, kp); sok {
			if s == craftKindItem || s == craftKindSet {
				kind = s
			} else {
				d.probs.add(kp, ProblemInvalidValue)
			}
		}
	}
	switch kind {
	case craftKindItem:
		id := o.reqID("id")
		if r, err := NewCraftItem(id); err == nil {
			return r
		}
	case craftKindSet:
		set := fieldEnum(o, "set", parseCraftSetName)
		var except []ItemID
		if n, ep, has := o.field("except", false); has {
			except = decodeItemIDs(d, n, ep, 0)
		}
		if r, err := NewCraftSet(set, except); err == nil {
			return r
		}
	default:
		o.skip("id", "set", "except") // the kind is unknown: its fields are not unknown keys
	}
	return nil
}

// The wire names of gearKind (FR-039).
const (
	gearKindPlain             = "plain"
	gearKindMagic             = "magic"
	gearKindFamily            = "family"
	gearKindScrollFamily      = "scrollFamily"
	gearKindAmmunitionVariant = "ammunitionVariant"
	gearKindFocusVariant      = "focusVariant"
	gearKindScrollVariant     = "scrollVariant"
	gearKindPack              = "pack"
)

func validGearKind(s string) bool {
	switch s {
	case gearKindPlain, gearKindMagic, gearKindFamily, gearKindScrollFamily, gearKindAmmunitionVariant,
		gearKindFocusVariant, gearKindScrollVariant, gearKindPack:
		return true
	}
	return false
}

// gearField reads a field only some gear kinds have: absent is not an
// error unless the kind needs it (missingField), and present on a kind that
// lacks it is constraintViolation (FR-039). has is false when the field is
// absent or was reported, so the caller reads it only when has.
func (o *object) gearField(key string, applies, needed bool) (n node, p string, has bool) {
	n, p, has = o.field(key, false)
	switch {
	case has && !applies:
		o.d.probs.add(p, ProblemConstraintViolation)
		return n, p, false
	case !has && applies && needed:
		o.d.probs.add(p, ProblemMissingField)
	}
	return n, p, has
}

// decodeGear is the row of the adventuringGear array (FR-039): id, name,
// source?, gearKind, then the price, weight, note?, text and the fields the
// kind has: variants, variantOf, alsoCountsAs, ammunition, contents,
// spellLevel, craftingOverride. A field that failed leaves its zero value,
// which the constructor rejects, and its problem makes the framework drop the
// entry: no flag tracks the failures.
func decodeGear(o *object) decodedEntry {
	d := o.d
	h := o.header()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	kind := ""
	if n, p, has := o.field("gearKind", true); has {
		if s, ok := d.str(n, p); ok {
			if validGearKind(s) {
				kind = s
			} else {
				d.probs.add(p, ProblemInvalidValue)
			}
		}
	}
	if kind == "" {
		o.skipAll() // without its kind, the fields are not unknown keys
		return res
	}
	isFamily := kind == gearKindFamily || kind == gearKindScrollFamily
	isVariant := kind == gearKindAmmunitionVariant || kind == gearKindFocusVariant || kind == gearKindScrollVariant
	// A price or weight that failed is nil, so FR-081's rule below reports
	// only what was read.
	before := d.probs.count
	price := o.price(isFamily)
	if d.probs.count != before {
		price = nil
	}
	before = d.probs.count
	weight := o.weight(true, false)
	if d.probs.count != before {
		weight = nil
	}
	note := o.optText("note")
	var text string
	if isVariant {
		text = o.optText("text")
	} else {
		text = o.reqText("text")
	}
	if kind == gearKindScrollFamily || kind == gearKindScrollVariant {
		scrollMustBeUnpriced(o, kind, price, weight)
	}

	var variants []ItemID
	if n, p, has := o.gearField("variants", isFamily, true); has {
		variants = decodeItemIDs(d, n, p, 1)
	}
	var variantOf ItemID
	if n, p, has := o.gearField("variantOf", isVariant, true); has {
		id, _ := d.id(n, p)
		variantOf = ItemID(id.String())
	}
	var alsoCountsAs ItemID
	if n, p, has := o.gearField("alsoCountsAs", kind == gearKindFocusVariant, false); has {
		id, _ := d.id(n, p)
		alsoCountsAs = ItemID(id.String())
	}
	var ammunition AmmunitionData
	if n, p, has := o.gearField("ammunition", kind == gearKindAmmunitionVariant, true); has {
		ammunition = decodeAmmunition(d, n, p)
	}
	var contents []PackItem
	if n, p, has := o.gearField("contents", kind == gearKindPack, true); has {
		contents = decodeContents(d, n, p)
	}
	spellLevel := 0
	if n, p, has := o.gearField("spellLevel", kind == gearKindScrollVariant, true); has {
		v, _ := d.integer(n, p, equipmentMinSpellLevel, equipmentMaxSpellLevel)
		spellLevel = int(v)
	}
	var override CraftingOverride
	if n, p, has := o.gearField("craftingOverride", kind == gearKindMagic, false); has {
		override = decodeCraftingOverride(d, n, p)
	}
	res.entry = buildGear(kind, ItemID(h.id.String()), h.name, h.source, price, weight, note, text, variants, variantOf, alsoCountsAs, ammunition, contents, spellLevel, override)
	return res
}

// scrollMustBeUnpriced reports, as constraintViolation at the field, a Spell
// Scroll family with a fixed cost and a family or row with a weight (FR-081).
func scrollMustBeUnpriced(o *object, kind string, price Price, weight WeightSpec) {
	if _, fixed := price.(FixedCost); kind == gearKindScrollFamily && fixed {
		o.d.probs.add(keyPath(o.path, "cost"), ProblemConstraintViolation)
	}
	if _, none := weight.(NoWeight); weight != nil && !none {
		key := "weightPounds"
		if !o.has(key) {
			key = "weightVaries"
		}
		o.d.probs.add(keyPath(o.path, key), ProblemConstraintViolation)
	}
}

// buildGear builds the entry of the gear kind from what was read; nil when a
// constructor rejects it (the fields were each valid, so only a rule between
// them can).
func buildGear(kind string, id ItemID, name string, source SourceCitation, price Price, weight WeightSpec, note, text string, variants []ItemID, variantOf, alsoCountsAs ItemID, ammunition AmmunitionData, contents []PackItem, spellLevel int, override CraftingOverride) Entry {
	fixed, _ := price.(FixedCost)
	switch kind {
	case gearKindPlain:
		if g, err := NewPlainGear(id, name, source, fixed, weight, note, text); err == nil {
			return g
		}
	case gearKindMagic:
		if g, err := NewMagicGear(id, name, source, fixed, weight, note, text, override); err == nil {
			return g
		}
	case gearKindFamily:
		if g, err := NewGearFamily(id, name, source, price, weight, note, text, variants); err == nil {
			return g
		}
	case gearKindScrollFamily:
		if g, err := NewScrollFamily(id, name, source, note, text, variants); err == nil {
			return g
		}
	case gearKindAmmunitionVariant:
		if g, err := NewAmmunitionVariant(id, name, source, variantOf, fixed, weight, note, text, ammunition); err == nil {
			return g
		}
	case gearKindFocusVariant:
		if g, err := NewFocusVariant(id, name, source, variantOf, fixed, weight, note, text, alsoCountsAs); err == nil {
			return g
		}
	case gearKindScrollVariant:
		if g, err := NewScrollVariant(id, name, source, variantOf, fixed, note, text, spellLevel); err == nil {
			return g
		}
	case gearKindPack:
		if g, err := NewPack(id, name, source, fixed, weight, note, text, contents); err == nil {
			return g
		}
	}
	return nil
}

// decodeAmmunition reads {amount, storage} (FR-041); the zero value when it
// failed.
func decodeAmmunition(d *decoder, n node, p string) AmmunitionData {
	o, ok := d.object(n, p)
	if !ok {
		return AmmunitionData{}
	}
	amount := o.reqInt("amount", 1, equipmentMaxInt)
	storage := o.reqText("storage")
	o.finish()
	a, _ := NewAmmunitionData(amount, storage)
	return a
}

// decodeContents reads the non-empty contents of a pack: {id, quantity,
// unit?} (FR-042). An item that failed is left out.
func decodeContents(d *decoder, n node, path string) []PackItem {
	d.arrayLen(n, path, 1, noUpperBound)
	var out []PackItem
	d.each(n, path, func(_ int, el node, p string) {
		o, ok := d.object(el, p)
		if !ok {
			return
		}
		id := o.reqID("id")
		quantity := o.reqInt("quantity", 1, equipmentMaxInt)
		unit := o.optText("unit")
		o.finish()
		if item, err := NewPackItem(id, quantity, unit); err == nil {
			out = append(out, item)
		}
	})
	return out
}

// decodeCraftingOverride reads {rawMaterials, days, toolIds} (FR-117); every
// part is required and the tool list may be empty. The zero value when it
// failed.
func decodeCraftingOverride(d *decoder, n node, p string) CraftingOverride {
	o, ok := d.object(n, p)
	if !ok {
		return CraftingOverride{}
	}
	var raw Cost
	if rn, rp, has := o.field("rawMaterials", true); has {
		raw = decodeCost(d, rn, rp)
	}
	days := o.reqInt("days", 1, equipmentMaxInt)
	var tools []ItemID
	if tn, tp, has := o.field("toolIds", true); has {
		tools = decodeItemIDs(d, tn, tp, 0)
	}
	o.finish()
	c, _ := NewCraftingOverride(raw, days, tools)
	return c
}

// familyLinks returns what one tools or adventuringGear entry says about the
// family links: its id, the variants it lists (nil for a non-family) and the
// family it names in variantOf (empty for a non-variant).
func familyLinks(e Entry) (id ItemID, variants []ItemID, variantOf ItemID) {
	switch v := e.(type) {
	case ToolFamily:
		id, variants = v.ID(), v.variants
	case ToolVariant:
		id, variantOf = v.ID(), v.variantOf
	case GearFamily:
		id, variants = v.ID(), v.variants
	case ScrollFamily:
		id, variants = v.ID(), v.variants
	case GearVariant:
		id, variantOf = v.ID(), v.VariantOf()
	}
	return id, variants, variantOf
}

// checkFamilyLinks reports the cross-entry rules of FR-030 and FR-039 as
// constraintViolation at the later variants/<j>: a variant listed by two
// families, and a listed variant whose variantOf names another family. A
// target that is no entry of this array is ValidateSet's (unresolvedReference).
func checkFamilyLinks(d *decoder, path string, entries []Entry) {
	parent := map[ItemID]ItemID{} // variant id to its variantOf
	for _, e := range entries {
		if id, _, of := familyLinks(e); of != "" {
			parent[id] = of
		}
	}
	listed := map[ItemID]struct{}{} // variants some earlier family lists
	for i, e := range entries {
		id, variants, _ := familyLinks(e)
		for j, v := range variants {
			_, twice := listed[v]
			listed[v] = struct{}{}
			if of, ok := parent[v]; twice || (ok && of != id) {
				d.probs.add(indexPath(keyPath(indexPath(path, i), "variants"), j), ProblemConstraintViolation)
			}
		}
	}
}
