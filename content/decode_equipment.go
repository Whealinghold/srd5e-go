package content

import (
	"cmp"
	"math"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// This file holds the field helpers of the equipment rows and the rows of the
// weapons, armor and the simple tables (content-equipment FR-008, FR-012 to
// FR-014, FR-057, FR-064, FR-080, FR-091, FR-098 to FR-119). Tools and
// adventuring gear are in decode_equipment_gear.go.

// has reports whether the object has a member named key, repeated or not.
func (o *object) has(key string) bool {
	_, ok := o.index[key]
	return ok
}

// The helpers below return the value only: a failure leaves the zero value and
// has reported its problem, and a reported problem drops the entry (the
// framework), so no row tracks a flag. Where a rule needs to know whether
// what it reads failed, it compares the problem count.

// decimal reads a number of at least lo and at most 2^31 - 1 (FR-080, FR-091).
// A string, boolean and null are wrongType; a number outside the range, or too
// large for a float64, is outOfRange.
func (d *decoder) decimal(n node, path string, lo float64) float64 {
	if n.kind() != nodeNumber {
		d.probs.add(path, ProblemWrongType)
		return 0
	}
	v, err := n.number().Float64()
	if err != nil || v < lo || v > equipmentMaxInt {
		d.probs.add(path, ProblemOutOfRange)
		return 0
	}
	return v
}

// halfStep reads a number of at least 0.5 that is a multiple of 0.5 (FR-091).
func (d *decoder) halfStep(n node, path string) float64 {
	v := d.decimal(n, path, 0.5)
	if v*2 != math.Trunc(v*2) {
		d.probs.add(path, ProblemOutOfRange)
		return 0
	}
	return v
}

// strictDiceField reads a dice expression with an explicit count and no
// modifier ("1d8"): stricter than dice.ParseExpression (FR-110). Anything else
// is invalidValue.
func (d *decoder) strictDiceField(n node, path string) dice.Expression {
	s, ok := d.str(n, path)
	if !ok {
		return dice.Expression{}
	}
	e, ok := parseStrictDice(s)
	if !ok {
		d.probs.add(path, ProblemInvalidValue)
	}
	return e
}

// fieldEnum reads the required field key as a member of a vocabulary.
func fieldEnum[T any](o *object, key string, parse func(string) (T, error)) T {
	var zero T
	n, p, has := o.field(key, true)
	if !has {
		return zero
	}
	v, _ := enumField(o.d, n, p, parse)
	return v
}

// reqInt reads the required integer field key in [lo, hi].
func (o *object) reqInt(key string, lo, hi int64) int {
	n, p, has := o.field(key, true)
	if !has {
		return 0
	}
	v, _ := o.d.integer(n, p, lo, hi)
	return int(v)
}

// optInt reads the optional integer field key in [lo, hi]; absent is 0.
func (o *object) optInt(key string, lo, hi int64) int {
	n, p, has := o.field(key, false)
	if !has {
		return 0
	}
	v, _ := o.d.integer(n, p, lo, hi)
	return int(v)
}

// reqText reads the required text field key (FR-028).
func (o *object) reqText(key string) string {
	n, p, has := o.field(key, true)
	if !has {
		return ""
	}
	s, _ := o.d.text(n, p)
	return s
}

// optText reads the optional text field key; absent is "".
func (o *object) optText(key string) string {
	n, p, has := o.field(key, false)
	if !has {
		return ""
	}
	s, _ := o.d.text(n, p)
	return s
}

// reqID reads the required id field key.
func (o *object) reqID(key string) ItemID {
	n, p, has := o.field(key, true)
	if !has {
		return ""
	}
	id, _ := o.d.id(n, p)
	return ItemID(id.String())
}

// decodeItemIDs reads a list of lo or more ids that is a set: a repeat is
// duplicate at the index of the later element. An element that failed is left
// out.
func decodeItemIDs(d *decoder, n node, path string, lo int) []ItemID {
	var out []ItemID
	seen := map[ItemID]struct{}{}
	d.arrayLen(n, path, lo, noUpperBound)
	d.each(n, path, func(_ int, el node, p string) {
		id, ok := d.id(el, p)
		if !ok {
			return
		}
		item := ItemID(id.String())
		if _, repeated := seen[item]; repeated {
			d.probs.add(p, ProblemDuplicate)
			return
		}
		seen[item] = struct{}{}
		out = append(out, item)
	})
	return out
}

// decodeCost reads {amount, coin, per?} (FR-012, FR-080).
func decodeCost(d *decoder, n node, p string) Cost {
	o, ok := d.object(n, p)
	if !ok {
		return Cost{}
	}
	amount := o.reqInt("amount", 0, MaxCostAmount)
	coin := fieldEnum(o, "coin", parseCoin)
	var per CostRate
	if pn, pp, has := o.field("per", false); has {
		per, _ = enumField(d, pn, pp, parseCostRate)
	}
	o.finish()
	c, _ := NewCost(amount, coin, per)
	return c
}

// price reads cost and costVaries (FR-013, FR-014): a fixed cost, or, where
// the entry may be a family, costVaries true in its place. A costVaries on an
// entry that may not hold one is constraintViolation at it.
func (o *object) price(allowVaries bool) Price {
	d := o.d
	cn, cp, hasCost := o.field("cost", false)
	vn, vp, hasVaries := o.field("costVaries", false)
	switch {
	case hasVaries && !allowVaries:
		d.probs.add(vp, ProblemConstraintViolation)
		if hasCost {
			decodeCost(d, cn, cp) // for its own problems
		}
		return nil
	case hasVaries:
		varies, ok := d.boolean(vn, vp)
		switch {
		case !ok:
		case !varies:
			d.probs.add(vp, ProblemInvalidValue)
		case hasCost:
			d.probs.add(vp, ProblemConstraintViolation)
		default:
			return CostVaries{}
		}
		return nil
	case !hasCost:
		d.probs.add(cp, ProblemMissingField)
		return nil
	}
	f, err := NewFixedCost(decodeCost(d, cn, cp))
	if err != nil {
		return nil
	}
	return f
}

// fixedPrice reads cost for an entry that has no costVaries.
func (o *object) fixedPrice() FixedCost {
	f, _ := o.price(false).(FixedCost)
	return f
}

// weight reads weightPounds and, where the entry may print "Varies" or a
// parenthetical (vary), weightVaries and weightNote (FR-013, FR-015, FR-080,
// FR-115). A weight with none of them is NoWeight, unless required.
func (o *object) weight(vary, required bool) WeightSpec {
	d := o.d
	pn, pp, hasPounds := o.field("weightPounds", required)
	var pounds float64
	if hasPounds {
		pounds = d.decimal(pn, pp, 0)
	}
	varies := false
	note := ""
	if vary {
		if vn, vp, has := o.field("weightVaries", false); has {
			switch v, ok := d.boolean(vn, vp); {
			case !ok:
			case !v:
				d.probs.add(vp, ProblemInvalidValue)
			case hasPounds:
				d.probs.add(vp, ProblemConstraintViolation)
			default:
				varies = true
			}
		}
		if nn, np, has := o.field("weightNote", false); has {
			var ok bool
			if note, ok = d.text(nn, np); ok && !hasPounds {
				d.probs.add(np, ProblemConstraintViolation)
			}
		}
	}
	switch {
	case varies:
		return WeightVaries{}
	case !hasPounds:
		return NoWeight{}
	}
	w, _ := NewWeight(pounds, note)
	return w
}

// decodeWeapon is the row of the weapons array (FR-098, FR-101 to FR-103,
// FR-106, FR-109 to FR-111): id, name, source?, category, weaponType, damage,
// damageType, properties, mastery, weightPounds?, cost.
func decodeWeapon(o *object) decodedEntry {
	d := o.d
	h := o.header()
	category := fieldEnum(o, "category", parseWeaponCategory)
	weaponType := fieldEnum(o, "weaponType", parseWeaponType)
	var parts damageParts
	if n, p, has := o.field("damage", true); has {
		parts = decodeDamageParts(d, n, p)
	}
	damageType := fieldEnum(o, "damageType", ParseDamageType)
	var props []WeaponPropertyUse
	if n, p, has := o.field("properties", true); has {
		props = decodeProperties(d, n, p)
	}
	mastery := fieldEnum(o, "mastery", ParseMasteryProperty)
	weight := o.weight(false, false)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	damage, _ := parts.build(damageType) // a failed part leaves the zero damage, which NewWeapon rejects
	if w, err := NewWeapon(ItemID(h.id.String()), h.name, h.source, category, weaponType, damage, props, mastery, weight, price); err == nil {
		res.entry = w
	}
	return res
}

// damageParts is the damage of a weapon as read, before the type is known.
type damageParts struct {
	dice   dice.Expression
	fixed  int
	isDice bool
}

func (p damageParts) build(t DamageType) (WeaponDamage, error) {
	if p.isDice {
		return NewWeaponDiceDamage(p.dice, t)
	}
	return NewWeaponFixedDamage(p.fixed, t)
}

// decodeDamageParts reads {dice} or {fixed} (FR-110): neither is missingField
// and both constraintViolation, at the object.
func decodeDamageParts(d *decoder, n node, p string) damageParts {
	o, ok := d.object(n, p)
	if !ok {
		return damageParts{}
	}
	dn, dp, hasDice := o.field("dice", false)
	fn, fp, hasFixed := o.field("fixed", false)
	o.finish()
	switch {
	case !hasDice && !hasFixed:
		d.probs.add(p, ProblemMissingField)
	case hasDice && hasFixed:
		d.probs.add(p, ProblemConstraintViolation)
	case hasDice:
		return damageParts{dice: d.strictDiceField(dn, dp), isDice: true}
	default:
		v, _ := d.integer(fn, fp, 1, equipmentMaxInt)
		return damageParts{fixed: int(v)}
	}
	return damageParts{}
}

// decodeProperties reads the property list of a weapon, in printed order. A
// property that failed is left out.
func decodeProperties(d *decoder, n node, p string) []WeaponPropertyUse {
	d.arrayLen(n, p, 0, noUpperBound)
	var out []WeaponPropertyUse
	seen := map[WeaponProperty]struct{}{}
	d.each(n, p, func(_ int, el node, ep string) {
		if use := decodePropertyUse(d, el, ep, seen); use != nil {
			out = append(out, use)
		}
	})
	return out
}

// decodePropertyUse reads one property of a weapon (FR-111): the property,
// then range, ammunition, damage and condition, each required or forbidden by
// the property. Nil when it failed.
func decodePropertyUse(d *decoder, el node, ep string, seen map[WeaponProperty]struct{}) WeaponPropertyUse {
	o, ok := d.object(el, ep)
	if !ok {
		return nil
	}
	before := d.probs.count
	property := fieldEnum(o, "property", ParseWeaponProperty)
	valid := d.probs.count == before
	if valid {
		if _, repeated := seen[property]; repeated {
			d.probs.add(ep, ProblemDuplicate) // the fields of the repeat are still read, for their own problems
		}
		seen[property] = struct{}{}
	}
	rn, rp, hasRange := o.field("range", false)
	an, ap, hasAmmo := o.field("ammunition", false)
	dn, dp, hasDamage := o.field("damage", false)
	cn, cp, hasCondition := o.field("condition", false)
	defer o.finish()
	if !valid {
		return nil
	}
	// Which of the four fields each property takes; a field it does not take
	// is constraintViolation, a field it needs and lacks is missingField.
	wantRange := property == WeaponPropertyAmmunition || property == WeaponPropertyThrown
	wantAmmo := property == WeaponPropertyAmmunition
	wantDamage := property == WeaponPropertyVersatile
	wantCondition := property == WeaponPropertyTwoHanded
	var rng RangePair
	var ammo AmmunitionType
	var damage dice.Expression
	condition := TwoHandedAlways
	gate := func(has, want, require bool, path string) bool {
		switch {
		case has && !want:
			d.probs.add(path, ProblemConstraintViolation)
		case !has && want && require:
			d.probs.add(path, ProblemMissingField)
		default:
			return has
		}
		return false
	}
	if gate(hasRange, wantRange, true, rp) {
		rng = decodeRange(d, rn, rp)
	}
	if gate(hasAmmo, wantAmmo, true, ap) {
		ammo, _ = enumField(d, an, ap, parseAmmunitionType)
	}
	if gate(hasDamage, wantDamage, true, dp) {
		damage = d.strictDiceField(dn, dp)
	}
	if gate(hasCondition, wantCondition, false, cp) {
		switch s, k := d.str(cn, cp); {
		case !k:
		case s == "unless mounted":
			condition = TwoHandedUnlessMounted
		default:
			d.probs.add(cp, ProblemInvalidValue)
		}
	}
	var use WeaponPropertyUse
	var err error
	switch property { //nolint:exhaustive // the five plain properties take the default
	case WeaponPropertyAmmunition:
		use, err = buildUse(NewAmmunitionUse(rng, ammo))
	case WeaponPropertyThrown:
		use, err = buildUse(NewThrownUse(rng))
	case WeaponPropertyVersatile:
		use, err = buildUse(NewVersatileUse(damage))
	case WeaponPropertyTwoHanded:
		use, err = buildUse(NewTwoHandedUse(condition))
	default:
		use, err = buildUse(NewPlainUse(property))
	}
	if err != nil {
		return nil
	}
	return use
}

// buildUse widens a constructor result to the interface.
func buildUse[T WeaponPropertyUse](u T, err error) (WeaponPropertyUse, error) { return u, err }

// decodeRange reads {normal, long} (FR-111): normal at least 1, long greater
// than normal (constraintViolation at long).
func decodeRange(d *decoder, n node, p string) RangePair {
	o, ok := d.object(n, p)
	if !ok {
		return RangePair{}
	}
	before := d.probs.count
	normal := o.reqInt("normal", 1, equipmentMaxInt)
	long := 0
	ln, lp, has := o.field("long", true)
	if has {
		v, _ := d.integer(ln, lp, 1, equipmentMaxInt)
		long = int(v)
	}
	clean := d.probs.count == before
	o.finish()
	if clean && long <= normal {
		d.probs.add(lp, ProblemConstraintViolation)
		return RangePair{}
	}
	r, _ := NewRangePair(normal, long)
	return r
}

// decodeArmor is the row of the armor array (FR-105, FR-109, FR-112): id,
// name, source?, category, armorClass, strengthRequirement?,
// stealthDisadvantage, weightPounds, cost.
func decodeArmor(o *object) decodedEntry {
	d := o.d
	h := o.header()
	before := d.probs.count
	category := fieldEnum(o, "category", parseArmorCategory)
	var rule ArmorClassRule
	rn, rp, hasRule := o.field("armorClass", true)
	if hasRule {
		rule = decodeArmorClass(d, rn, rp)
	}
	if d.probs.count == before { // the category and the rule both decoded
		if _, bonus := rule.(BonusRule); bonus != (category == ArmorShield) {
			d.probs.add(rp, ProblemConstraintViolation)
		}
	}
	strength := o.optInt("strengthRequirement", armorMinStrength, armorMaxStrength)
	stealth := false
	if n, p, has := o.field("stealthDisadvantage", true); has {
		stealth, _ = d.boolean(n, p)
	}
	weight := o.weight(false, true)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	w, _ := weight.(Weight)
	if a, err := NewArmor(ItemID(h.id.String()), h.name, h.source, category, rule, strength, stealth, w, price); err == nil {
		res.entry = a
	}
	return res
}

// decodeArmorClass reads {base, dexterity, maximum?} or {bonus} (FR-112); nil
// when it failed.
func decodeArmorClass(d *decoder, n node, p string) ArmorClassRule {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	bn, bp, hasBase := o.field("base", false)
	xn, xp, hasDex := o.field("dexterity", false)
	mn, mp, hasMax := o.field("maximum", false)
	un, up, hasBonus := o.field("bonus", false)
	switch {
	case !hasBase && !hasBonus:
		d.probs.add(p, ProblemMissingField)
		return nil
	case hasBase && hasBonus:
		d.probs.add(p, ProblemConstraintViolation)
		return nil
	case hasBonus:
		v, _ := d.integer(un, up, 1, armorBonusMax)
		for _, extra := range []struct {
			path string
			has  bool
		}{{xp, hasDex}, {mp, hasMax}} { // a bonus has no Dexterity treatment
			if extra.has {
				d.probs.add(extra.path, ProblemConstraintViolation)
			}
		}
		r, err := NewBonusRule(int(v))
		if err != nil {
			return nil
		}
		return r
	}
	base, _ := d.integer(bn, bp, 1, armorBaseMax)
	var dex DexterityBonus
	if !hasDex {
		d.probs.add(xp, ProblemMissingField)
	} else if s, sok := d.str(xn, xp); sok {
		switch s {
		case "full":
			dex = FullDexterity{}
		case "capped":
			dex = decodeCap(d, mn, mp, hasMax)
		case "none":
			dex = NoDexterity{}
		default:
			d.probs.add(xp, ProblemInvalidValue)
		}
		if s != "capped" && hasMax {
			d.probs.add(mp, ProblemConstraintViolation)
		}
	}
	r, err := NewBaseRule(int(base), dex)
	if err != nil {
		return nil
	}
	return r
}

// decodeCap reads the maximum of a capped Dexterity bonus, required there; nil
// when it failed.
func decodeCap(d *decoder, n node, p string, has bool) DexterityBonus {
	if !has {
		d.probs.add(p, ProblemMissingField)
		return nil
	}
	v, _ := d.integer(n, p, 0, equipmentMaxInt)
	c, err := NewCappedDexterity(int(v))
	if err != nil {
		return nil
	}
	return c
}

// decodeMount is the row of the mounts array (FR-091): id, name, source?,
// carryingCapacityPounds, cost.
func decodeMount(o *object) decodedEntry {
	h := o.header()
	capacity := o.reqInt("carryingCapacityPounds", 1, equipmentMaxInt)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if m, err := NewMount(ItemID(h.id.String()), h.name, h.source, capacity, price); err == nil {
		res.entry = m
	}
	return res
}

// decodeTack is the row of the tackHarnessAndDrawnVehicles array: id, name,
// source?, weightPounds?, cost.
func decodeTack(o *object) decodedEntry {
	h := o.header()
	weight := o.weight(false, false)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if t, err := NewTackHarnessOrDrawnVehicle(ItemID(h.id.String()), h.name, h.source, weight, price); err == nil {
		res.entry = t
	}
	return res
}

// decodeVehicle is the row of the airborneAndWaterborneVehicles array
// (FR-048, FR-091): id, name, source?, speedMph, crew, passengers?,
// cargoTons?, armorClass, hitPoints, damageThreshold?, weightPounds?, cost.
func decodeVehicle(o *object) decodedEntry {
	d := o.d
	h := o.header()
	var speed, cargo float64
	if n, p, has := o.field("speedMph", true); has {
		speed = d.halfStep(n, p)
	}
	crew := o.reqInt("crew", 1, equipmentMaxInt)
	passengers := o.optInt("passengers", 1, equipmentMaxInt)
	if n, p, has := o.field("cargoTons", false); has {
		cargo = d.halfStep(n, p)
	}
	armorClass := o.reqInt("armorClass", 1, equipmentMaxInt)
	hitPoints := o.reqInt("hitPoints", 1, equipmentMaxInt)
	threshold := o.optInt("damageThreshold", 1, equipmentMaxInt)
	weight := o.weight(false, false)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if v, err := NewAirborneOrWaterborneVehicle(ItemID(h.id.String()), h.name, h.source, speed, crew, passengers, cargo, armorClass, hitPoints, threshold, weight, price); err == nil {
		res.entry = v
	}
	return res
}

// decodeLifestyle is the row of the lifestyleExpenses array (FR-116): id,
// name, source?, cost, text.
func decodeLifestyle(o *object) decodedEntry {
	h := o.header()
	price := o.fixedPrice()
	text := o.reqText("text")
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if e, err := NewLifestyleExpense(ItemID(h.id.String()), h.name, h.source, price, text); err == nil {
		res.entry = e
	}
	return res
}

// decodeFood is the row of the foodDrinkAndLodging array: id, name, source?,
// cost.
func decodeFood(o *object) decodedEntry {
	h := o.header()
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if e, err := NewFoodDrinkOrLodging(ItemID(h.id.String()), h.name, h.source, price); err == nil {
		res.entry = e
	}
	return res
}

// decodeHireling is the row of the hirelings array: id, name, source?, cost.
func decodeHireling(o *object) decodedEntry {
	h := o.header()
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if e, err := NewHireling(ItemID(h.id.String()), h.name, h.source, price); err == nil {
		res.entry = e
	}
	return res
}

// decodeService is the row of the spellcastingServices array (FR-057): id,
// name, source?, spellLevelMin, spellLevelMax, availability, cost. A minimum
// above the maximum is constraintViolation at the maximum.
func decodeService(o *object) decodedEntry {
	d := o.d
	h := o.header()
	before := d.probs.count
	low := o.reqInt("spellLevelMin", equipmentMinSpellLevel, equipmentMaxSpellLevel)
	high := 0
	if n, p, has := o.field("spellLevelMax", true); has {
		v, _ := d.integer(n, p, equipmentMinSpellLevel, equipmentMaxSpellLevel)
		high = int(v)
		if d.probs.count == before && low > high {
			d.probs.add(p, ProblemConstraintViolation)
		}
	}
	availability := fieldEnum(o, "availability", parseAvailability)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if s, err := NewSpellcastingService(ItemID(h.id.String()), h.name, h.source, low, high, availability, price); err == nil {
		res.entry = s
	}
	return res
}

// checkServiceRows reports a gap or overlap between the spell-level ranges of
// the rows: together they must cover 0 to 9 exactly once (FR-057),
// constraintViolation at the array.
func checkServiceRows(d *decoder, path string, entries []Entry) {
	rows := make([]SpellcastingService, 0, len(entries))
	for _, e := range entries {
		s, _ := e.(SpellcastingService) // only decodeService builds this array
		rows = append(rows, s)
	}
	slices.SortStableFunc(rows, func(a, b SpellcastingService) int { return cmp.Compare(a.min, b.min) })
	next := equipmentMinSpellLevel
	for _, r := range rows {
		if r.min != next {
			d.probs.add(path, ProblemConstraintViolation)
			return
		}
		next = r.max + 1
	}
	if next != equipmentMaxSpellLevel+1 {
		d.probs.add(path, ProblemConstraintViolation)
	}
}

// decodeScrollCost is the row of the spellScrollCosts array (FR-064): id,
// name, source?, spellLevel, days, cost.
func decodeScrollCost(o *object) decodedEntry {
	h := o.header()
	level := o.reqInt("spellLevel", equipmentMinSpellLevel, equipmentMaxSpellLevel)
	days := o.reqInt("days", 1, equipmentMaxInt)
	price := o.fixedPrice()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if s, err := NewSpellScrollCost(ItemID(h.id.String()), h.name, h.source, level, days, price); err == nil {
		res.entry = s
	}
	return res
}

// checkScrollCostRows reports a repeated spell level as duplicate at the later
// row's spellLevel and, when none repeats, a level from 0 to 9 that no row
// covers as constraintViolation at the array (FR-064).
func checkScrollCostRows(d *decoder, path string, entries []Entry) {
	seen := map[int]struct{}{}
	repeated := false
	for i, e := range entries {
		s, _ := e.(SpellScrollCost) // only decodeScrollCost builds this array
		if _, dup := seen[s.spellLevel]; dup {
			d.probs.add(indexPath(path, i)+"/spellLevel", ProblemDuplicate)
			repeated = true
		}
		seen[s.spellLevel] = struct{}{}
	}
	if !repeated && len(seen) != equipmentMaxSpellLevel-equipmentMinSpellLevel+1 {
		d.probs.add(path, ProblemConstraintViolation)
	}
}
