package content

import "math"

// spellMaxInt bounds every integer of the spell shape the decoder reads, so a
// decoded value is the same on every platform integer width (schema FR-021).
// A larger number is outOfRange.
const spellMaxInt = math.MaxInt32

// The wire discriminators of the spell parts (content-spells.md FR-108).
const (
	castingKindMagicAction = "magicAction"
	castingKindBonusAction = "bonusAction"
	castingKindReaction    = "reaction"
	castingKindMinutes     = "minutes"
	castingKindHours       = "hours"
	rangeKindSelf          = "self"
	rangeKindTouch         = "touch"
	rangeKindDistance      = "distance"
	rangeKindSight         = "sight"
	rangeKindUnlimited     = "unlimited"
	rangeKindSpecial       = "special"
	durationKindInstant    = "instantaneous"
	durationKindTimed      = "timed"
	durationKindDispelled  = "untilDispelled"
	durationKindTriggered  = "untilDispelledOrTriggered"
	durationKindSpecial    = "special"
	fieldAmount            = "amount"
	fieldTrigger           = "trigger"
	fieldLabel             = "label"
	fieldUnit              = "unit"
	fieldEffects           = "effects"
	fieldDescription       = "description"
	fieldCostsGP           = "costsGp"
	fieldConsumesMaterial  = "consumesMaterial"
	fieldHigherLevel       = "higherLevel"
	fieldCantripUpgrade    = "cantripUpgrade"
	fieldCastingTimes      = "castingTimes"
	fieldMaterial          = "material"
)

// optBool reads the optional boolean field key; absent is false.
func (o *object) optBool(key string) bool {
	n, p, has := o.field(key, false)
	if !has {
		return false
	}
	v, _ := o.d.boolean(n, p)
	return v
}

// reqBool reads the required boolean field key.
func (o *object) reqBool(key string) bool {
	n, p, has := o.field(key, true)
	if !has {
		return false
	}
	v, _ := o.d.boolean(n, p)
	return v
}

// decodeSpell is the row of the spells array (content-spells.md FR-107 to
// FR-111): id, name, source?, level, school, classes, castingTimes, ritual,
// range, components, duration, description, higherLevel?, cantripUpgrade?,
// effects?. The entry holds no reference to another kind, so the row registers
// no site: the class ids are not reference sites (FR-110).
func decodeSpell(o *object) decodedEntry {
	d := o.d
	before := d.probs.count
	h := o.header()
	res := decodedEntry{id: h.id, idOK: h.idOK}
	var f SpellFields
	levelOK := false
	if n, p, ok := o.field("level", true); ok {
		var v int64
		if v, levelOK = d.integer(n, p, MinSpellLevel, MaxSpellLevel); levelOK {
			f.Level = SpellLevel(v)
		}
	}
	if n, p, ok := o.field("school", true); ok {
		f.School, _ = enumField(d, n, p, ParseSpellSchool)
	}
	if n, p, ok := o.field("classes", true); ok {
		f.Classes = decodeClassIDs(d, n, p)
	}
	if n, p, ok := o.field(fieldCastingTimes, true); ok {
		f.CastingTimes = decodeCastingTimes(d, n, p)
	}
	f.Ritual = o.reqBool("ritual")
	if n, p, ok := o.field("range", true); ok {
		f.Range = decodeSpellRange(d, n, p)
	}
	if n, p, ok := o.field("components", true); ok {
		f.Components = decodeSpellComponents(d, n, p)
	}
	if n, p, ok := o.field("duration", true); ok {
		f.Duration = decodeSpellDuration(d, n, p)
	}
	if n, p, ok := o.field(fieldDescription, true); ok {
		f.Description = decodeTextBlocks(d, n, p)
	}
	if n, p, ok := o.field(fieldHigherLevel, false); ok {
		f.HigherLevel = decodeTextBlocks(d, n, p)
	}
	if n, p, ok := o.field(fieldCantripUpgrade, false); ok {
		f.CantripUpgrade = decodeTextBlocks(d, n, p)
	}
	if n, p, ok := o.field(fieldEffects, false); ok {
		f.Effects = decodeSpellEffects(d, n, p)
		// A scaling that contradicts the level is a rule between two fields
		// (spell-effect-data.md FR-009, FR-031): constraintViolation at effects.
		if f.Effects != nil && levelOK && f.Effects.CheckSpellLevel(int(f.Level)) != nil {
			d.probs.add(p, ProblemConstraintViolation)
		}
	}
	if !d.clean(before) {
		return res
	}
	if s, err := NewSpell(SpellID(h.id.String()), h.name, h.source, f); err == nil {
		res.entry = s
	}
	return res
}

// decodeClassIDs reads the classes of a spell: an array of any length of ids,
// a repeat duplicate at the later index (FR-110). A bad id has reported its
// problem.
func decodeClassIDs(d *decoder, n node, p string) []ClassID {
	var out []ClassID
	seen := map[ID]struct{}{}
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		id, ok := d.id(el, ep)
		if !ok {
			return
		}
		if _, repeated := seen[id]; repeated {
			d.probs.add(ep, ProblemDuplicate)
			return
		}
		seen[id] = struct{}{}
		out = append(out, ClassID(id.String()))
	})
	return out
}

// decodeCastingTimes reads the casting times (FR-030): one element is a single
// time that may carry no label; two or more are modes, each with a label.
func decodeCastingTimes(d *decoder, n node, p string) SpellCastingTimes {
	count, ok := d.arrayLen(n, p, 1, noUpperBound)
	if !ok {
		return nil
	}
	modes := count >= MinCastingTimeModes
	before := d.probs.count
	var times []CastingTime
	var labels []string
	d.each(n, p, func(_ int, el node, ep string) {
		t, label := decodeCastingTime(d, el, ep, modes)
		times, labels = append(times, t), append(labels, label)
	})
	if !d.clean(before) {
		return nil
	}
	if !modes {
		s, _ := NewSpellCastingTimesSingle(times[0])
		return s
	}
	list := make([]CastingTimeMode, len(times))
	for i, t := range times {
		list[i], _ = NewCastingTimeMode(labels[i], t)
	}
	m, _ := NewSpellCastingTimesModes(list)
	return m
}

// decodeCastingTime reads one casting time object: kind, amount?, trigger?,
// label?. The label is required when there are modes and not allowed on a lone
// time (constraintViolation at the label).
func decodeCastingTime(d *decoder, n node, p string, modes bool) (CastingTime, string) {
	o, ok := d.object(n, p)
	if !ok {
		return nil, ""
	}
	defer o.finish()
	kind := o.kindOf(castingKindMagicAction, castingKindBonusAction, castingKindReaction, castingKindMinutes, castingKindHours)
	if kind == "" {
		return nil, ""
	}
	var t CastingTime
	switch kind {
	case castingKindMagicAction:
		t = NewCastingTimeMagicAction()
	case castingKindBonusAction:
		t = NewCastingTimeBonusAction()
		if n, tp, ok := o.field(fieldTrigger, false); ok {
			if s, ok := d.text(n, tp); ok {
				t, _ = NewCastingTimeBonusActionTriggered(s)
			}
		}
	case castingKindReaction:
		if n, tp, ok := o.field(fieldTrigger, true); ok {
			if s, ok := d.text(n, tp); ok {
				t, _ = NewCastingTimeReaction(s)
			}
		}
	case castingKindMinutes:
		if a, ok := o.intAt(fieldAmount, MinCastingTimeAmount, spellMaxInt); ok {
			t, _ = NewCastingTimeMinutes(a)
		}
	case castingKindHours:
		if a, ok := o.intAt(fieldAmount, MinCastingTimeAmount, spellMaxInt); ok {
			t, _ = NewCastingTimeHours(a)
		}
	}
	var label string
	switch ln, lp, present := o.field(fieldLabel, modes); {
	case present && !modes:
		d.probs.add(lp, ProblemConstraintViolation)
	case present:
		label, _ = d.name(ln, lp)
	}
	return t, label
}

// decodeSpellRange reads the range object: kind, amount?, unit? (FR-108).
func decodeSpellRange(d *decoder, n node, p string) SpellRange {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf(rangeKindSelf, rangeKindTouch, rangeKindDistance, rangeKindSight, rangeKindUnlimited, rangeKindSpecial) {
	case rangeKindSelf:
		return NewSpellRangeSelf()
	case rangeKindTouch:
		return NewSpellRangeTouch()
	case rangeKindSight:
		return NewSpellRangeSight()
	case rangeKindUnlimited:
		return NewSpellRangeUnlimited()
	case rangeKindSpecial:
		return NewSpellRangeSpecial()
	case rangeKindDistance:
		before := d.probs.count
		amount, _ := o.intAt(fieldAmount, MinSpellRangeDistance, spellMaxInt)
		var unit DistanceUnit
		if n, up, ok := o.field(fieldUnit, true); ok {
			unit, _ = enumField(d, n, up, ParseDistanceUnit)
		}
		if r, err := NewSpellRangeDistance(amount, unit); err == nil && d.clean(before) {
			return r
		}
	}
	return nil
}

// decodeSpellDuration reads the duration object: kind, amount?, unit?, limit?
// (FR-108).
func decodeSpellDuration(d *decoder, n node, p string) SpellDuration {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf(durationKindInstant, durationKindTimed, durationKindDispelled, durationKindTriggered, durationKindSpecial) {
	case durationKindInstant:
		return NewSpellDurationInstantaneous()
	case durationKindDispelled:
		return NewSpellDurationUntilDispelled()
	case durationKindTriggered:
		return NewSpellDurationUntilDispelledOrTriggered()
	case durationKindSpecial:
		return NewSpellDurationSpecial()
	case durationKindTimed:
		before := d.probs.count
		amount, _ := o.intAt(fieldAmount, MinSpellDurationAmount, spellMaxInt)
		unit := reqEnum(o, fieldUnit, ParseDurationUnit)
		limit := reqEnum(o, "limit", ParseDurationLimit)
		if v, err := NewSpellDurationTimed(amount, unit, limit); err == nil && d.clean(before) {
			return v
		}
	}
	return nil
}

// reqEnum reads the required vocabulary field key; a failure leaves the zero
// value and has reported its problem.
func reqEnum[T any](o *object, key string, parse func(string) (T, error)) T {
	var v T
	if n, p, ok := o.field(key, true); ok {
		v, _ = enumField(o.d, n, p, parse)
	}
	return v
}

// optEnum reads the optional vocabulary field key; absent is the zero value.
func optEnum[T any](o *object, key string, parse func(string) (T, error)) T {
	var v T
	if n, p, ok := o.field(key, false); ok {
		v, _ = enumField(o.d, n, p, parse)
	}
	return v
}

// decodeSpellComponents reads the components object: verbal, somatic,
// material? (FR-049). It returns the zero value when a part had a problem.
func decodeSpellComponents(d *decoder, n node, p string) SpellComponents {
	o, ok := d.object(n, p)
	if !ok {
		return SpellComponents{}
	}
	defer o.finish()
	before := d.probs.count
	c := NewSpellComponents(o.reqBool("verbal"), o.reqBool("somatic"))
	if mn, mp, present := o.field(fieldMaterial, false); present {
		if m, ok := decodeMaterial(d, mn, mp); ok {
			c, _ = c.WithMaterial(m)
		}
	}
	if !d.clean(before) {
		return SpellComponents{}
	}
	return c
}

// decodeMaterial reads the material object: description, costsGp,
// consumesMaterial, all required.
func decodeMaterial(d *decoder, n node, p string) (MaterialComponent, bool) {
	o, ok := d.object(n, p)
	if !ok {
		return MaterialComponent{}, false
	}
	defer o.finish()
	before := d.probs.count
	description := o.reqText(fieldDescription)
	var costs []int
	if cn, cp, present := o.field(fieldCostsGP, true); present {
		d.list(cn, cp, 0, noUpperBound, func(_ int, el node, ep string) {
			if v, ok := d.integer(el, ep, MinMaterialCostGP, spellMaxInt); ok {
				costs = append(costs, int(v))
			}
		})
	}
	consumes := o.reqBool(fieldConsumesMaterial)
	m, err := NewMaterialComponent(description, costs, consumes)
	return m, err == nil && d.clean(before)
}
