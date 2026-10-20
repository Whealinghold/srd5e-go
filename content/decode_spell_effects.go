package content

import (
	"errors"
	"math"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// This file holds the decoder key "effects" of the spell object
// (spell-effect-data.md FR-030, FR-031, FR-049; revision row
// R2-spell-effects). The value is one object of a ModelledEffects:
// coverage, targeting, parts, zone?. Absence of the key is TextOnlyEffects.
// Every variant is an object with a string "kind", the lower camel case of the
// variant name, and the keys of that variant; an unknown kind is invalidValue
// and no other key of that object is examined. Each rule of a constructor is
// checked field by field with the same predicate; a rule between sibling
// fields is left to the constructor and reported with the kind of its
// BlockProblem at the offending key or, failing that, the object (blockKind).

// blockKind maps the reason of a constructor's *BlockError to the problem kind
// the decoder reports for it (spell-effect-data.md FR-031).
func blockKind(err error) ProblemKind {
	var be *BlockError
	if !errors.As(err, &be) {
		return ProblemInvalidValue
	}
	switch be.Problem {
	case BlockEmpty, BlockOutOfRange:
		return ProblemOutOfRange
	case BlockDuplicate:
		return ProblemDuplicate
	case BlockRaggedRow, BlockConstraint:
		return ProblemConstraintViolation
	case BlockMissing:
		return ProblemMissingField
	case BlockInvalidName, BlockBlankText, BlockInvalidMember, BlockInvalidCharacter:
		return ProblemInvalidValue
	}
	return ProblemInvalidValue
}

// rule reports the rejection err of a constructor at path, with the kind of
// its reason. It is called only when the decoder found no problem of its own in
// what it passed, so it reports a rule between fields (FR-031).
func (d *decoder) rule(path string, err error) { d.probs.add(path, blockKind(err)) }

// decodeSpellEffects reads the effects object and returns the model, or nil
// when it had a problem. A repeated, unknown or mistyped key is a problem as
// everywhere.
func decodeSpellEffects(d *decoder, n node, p string) SpellEffectModel {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	coverage := reqEnum(o, "coverage", ParseEffectCoverage)
	var targeting SpellTargeting
	if tn, tp, ok := o.field("targeting", true); ok {
		targeting = decodeTargeting(d, tn, tp)
	}
	var parts []SpellEffectPart
	partsPath := keyPath(p, "parts")
	partsClean := false
	if pn, pp, ok := o.field("parts", true); ok {
		partsPath = pp
		b := d.probs.count
		d.list(pn, pp, 1, noUpperBound, func(_ int, el node, ep string) {
			if part, ok := decodeEffectPart(d, el, ep); ok {
				parts = append(parts, part)
			}
		})
		partsClean = d.clean(b)
	}
	zn, zp, hasZone := o.field("zone", false)
	var zone ZoneFacts
	zoneOK := false
	if hasZone {
		partCount := -1 // unknown when a part had a problem: the index rule is not checked
		if partsClean {
			partCount = len(parts)
		}
		zone, zoneOK = decodeZone(d, zn, zp, partCount)
	}
	if !d.clean(before) {
		return nil
	}
	m, err := NewModelledEffects(coverage, targeting, parts)
	if err != nil {
		d.rule(partsPath, err) // the rule across parts (FR-045), or a bug reported rather than hidden
		return nil
	}
	if hasZone && zoneOK {
		if m, err = m.WithZoneFacts(zone); err != nil {
			d.rule(zp, err)
			return nil
		}
	}
	return m
}

// decodeTargeting reads a targeting object: kind targetsSelf, targetsChosen
// (subject, count, restriction?, selfPolicy?) or targetsInArea (shape,
// subject, restriction?). selfPolicy absent is MayIncludeCaster, the default
// (FR-040).
func decodeTargeting(d *decoder, n node, p string) SpellTargeting {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	switch o.kindOf("targetsSelf", "targetsChosen", "targetsInArea") {
	case "targetsSelf":
		return TargetsSelf{}
	case "targetsChosen":
		subject := reqEnum(o, "subject", ParseTargetSubject)
		var count TargetCount
		if cn, cp, ok := o.field("count", true); ok {
			count = decodeTargetCount(d, cn, cp)
		}
		restriction := decodeRestriction(o)
		policy := optEnum(o, "selfPolicy", ParseSelfTargetPolicy)
		if !o.has("selfPolicy") {
			policy = MayIncludeCaster
		}
		if !d.clean(before) {
			return nil
		}
		t, err := NewTargetsChosen(subject, count, restriction, policy)
		if err != nil {
			d.rule(keyPath(p, "restriction"), err)
			return nil
		}
		return t
	case "targetsInArea":
		var shape AreaShape
		if sn, sp, ok := o.field("shape", true); ok {
			shape = decodeAreaShape(d, sn, sp)
		}
		subject := reqEnum(o, "subject", ParseTargetSubject)
		restriction := decodeRestriction(o)
		if !d.clean(before) {
			return nil
		}
		t, err := NewTargetsInArea(shape, subject, restriction)
		if err != nil {
			d.rule(keyPath(p, "restriction"), err)
			return nil
		}
		return t
	}
	return nil
}

// decodeRestriction reads the optional "restriction" of o: kind anyCreatureType
// or onlyCreatureTypes (types, a set of at least one creature type). Absent is
// nil, which the constructors read as AnyCreatureType.
func decodeRestriction(o *object) CreatureTypeRestriction {
	n, p, present := o.field("restriction", false)
	if !present {
		return nil
	}
	d := o.d
	ro, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer ro.finish()
	switch ro.kindOf("anyCreatureType", "onlyCreatureTypes") {
	case "anyCreatureType":
		return AnyCreatureType{}
	case "onlyCreatureTypes":
		tn, tp, ok := ro.field("types", true)
		if !ok {
			return nil
		}
		types, clean := enumSet(d, tn, tp, 1, len(creatureTypeNames()), ParseCreatureType)
		if r, err := NewOnlyCreatureTypes(types); clean && err == nil {
			return r
		}
	}
	return nil
}

// decodeAreaShape reads an area object: kind coneArea (lengthFeet), cubeArea
// (sideFeet), cylinderArea (radiusFeet, heightFeet), emanationArea
// (distanceFeet), lineArea (lengthFeet, widthFeet) or sphereArea (radiusFeet);
// each dimension is an integer of at least 1.
func decodeAreaShape(d *decoder, n node, p string) AreaShape {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	dim := func(key string) int {
		v, _ := o.intAt(key, 1, spellMaxInt)
		return v
	}
	before := d.probs.count
	var shape AreaShape
	switch o.kindOf("coneArea", "cubeArea", "cylinderArea", "emanationArea", "lineArea", "sphereArea") {
	case "coneArea":
		shape, _ = NewConeArea(dim("lengthFeet"))
	case "cubeArea":
		shape, _ = NewCubeArea(dim("sideFeet"))
	case "cylinderArea":
		radius := dim("radiusFeet")
		shape, _ = NewCylinderArea(radius, dim("heightFeet"))
	case "emanationArea":
		shape, _ = NewEmanationArea(dim("distanceFeet"))
	case "lineArea":
		length := dim("lengthFeet")
		shape, _ = NewLineArea(length, dim("widthFeet"))
	case "sphereArea":
		shape, _ = NewSphereArea(dim("radiusFeet"))
	}
	if !d.clean(before) {
		return nil
	}
	return shape
}

// decodeTargetCount reads a count object: kind exactlyTargets or upToTargets
// (n, scaling?) or targetsByInstance.
func decodeTargetCount(d *decoder, n node, p string) TargetCount {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	kind := o.kindOf("exactlyTargets", "upToTargets", "targetsByInstance")
	if kind == "targetsByInstance" {
		return TargetsByInstance{}
	}
	if kind == "" {
		return nil
	}
	base, _ := o.intAt("n", 1, spellMaxInt)
	scaling := decodeOptCountScaling(o)
	if !d.clean(before) {
		return nil
	}
	if kind == "exactlyTargets" {
		c, _ := NewExactlyTargets(base, scaling)
		return c
	}
	c, _ := NewUpToTargets(base, scaling)
	return c
}

// decodeOptCountScaling reads the optional "scaling" of o as a count scaling;
// absent is nil (NoCountScaling).
func decodeOptCountScaling(o *object) CountScaling {
	n, p, present := o.field("scaling", false)
	if !present {
		return nil
	}
	return decodeCountScaling(o.d, n, p)
}

// decodeCountScaling reads a count scaling object: kind noCountScaling,
// perSlotLevelCount (aboveLevel, extra) or byCharacterLevelCount (entries of
// level and count).
func decodeCountScaling(d *decoder, n node, p string) CountScaling {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	switch o.kindOf("noCountScaling", "perSlotLevelCount", "byCharacterLevelCount") {
	case "noCountScaling":
		return NoCountScaling{}
	case "perSlotLevelCount":
		above, _ := o.intAt("aboveLevel", math.MinInt32, spellMaxInt)
		extra, _ := o.intAt("extra", 1, spellMaxInt)
		if s, err := NewPerSlotLevelCount(above, extra); err == nil && d.clean(before) {
			return s
		}
	case "byCharacterLevelCount":
		en, ep, ok := o.field("entries", true)
		if !ok {
			return nil
		}
		var entries []CountLevelEntry
		d.list(en, ep, 1, noUpperBound, func(_ int, el node, lp string) {
			if eo, ok := d.object(el, lp); ok {
				level, _ := eo.intAt("level", 1, spellMaxCharacterLevel)
				count, _ := eo.intAt("count", 1, spellMaxInt)
				eo.finish()
				entries = append(entries, CountLevelEntry{Level: level, Count: count})
			}
		})
		if !d.clean(before) {
			return nil
		}
		table, err := NewCountLevelTable(entries)
		if err != nil {
			d.rule(ep, err) // the first level or the order (FR-019)
			return nil
		}
		s, _ := NewByCharacterLevelCount(table)
		return s
	}
	return nil
}

// decodeEffectPart reads a part object: resolution, instances?. Absent instances is
// one instance with no scaling.
func decodeEffectPart(d *decoder, n node, p string) (SpellEffectPart, bool) {
	o, ok := d.object(n, p)
	if !ok {
		return SpellEffectPart{}, false
	}
	defer o.finish()
	before := d.probs.count
	var resolution PartResolution
	if rn, rp, ok := o.field("resolution", true); ok {
		resolution = decodeResolution(d, rn, rp)
	}
	instances, _ := NewPartInstances(1, nil)
	if in, ip, present := o.field("instances", false); present {
		instances = decodeInstances(d, in, ip)
	}
	if !d.clean(before) {
		return SpellEffectPart{}, false
	}
	part, err := NewSpellEffectPart(resolution, instances)
	if err != nil {
		d.rule(p, err)
		return SpellEffectPart{}, false
	}
	return part, true
}

// decodeInstances reads an instances object: base, scaling?.
func decodeInstances(d *decoder, n node, p string) PartInstances {
	o, ok := d.object(n, p)
	if !ok {
		return PartInstances{}
	}
	defer o.finish()
	before := d.probs.count
	base, _ := o.intAt("base", 1, spellMaxInt)
	scaling := decodeOptCountScaling(o)
	if i, err := NewPartInstances(base, scaling); err == nil && d.clean(before) {
		return i
	}
	return PartInstances{}
}

// decodeResolution reads a resolution object: kind attackRollResolution
// (attack, onHit, onMiss?), savingThrowResolution (save, onFailure,
// onSuccess) or automaticResolution (outcomes).
func decodeResolution(d *decoder, n node, p string) PartResolution {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	switch o.kindOf("attackRollResolution", "savingThrowResolution", "automaticResolution") {
	case "attackRollResolution":
		kind := reqEnum(o, "attack", ParseSpellAttackKind)
		onHit := decodeOutcomeList(o, "onHit", 1)
		onMiss := decodeOutcomeList(o, "onMiss", 0)
		if r, err := NewAttackRollResolution(kind, onHit, onMiss); err == nil && d.clean(before) {
			return r
		}
	case "savingThrowResolution":
		save := reqEnum(o, "save", parseAbilityExact)
		onFailure := decodeOutcomeList(o, "onFailure", 1)
		onSuccess := reqEnum(o, "onSuccess", ParseSuccessEffect)
		if !d.clean(before) {
			return nil
		}
		r, err := NewSavingThrowResolution(save, onFailure, onSuccess)
		if err != nil {
			d.rule(keyPath(p, "onSuccess"), err) // half damage with no damage to halve (FR-043)
			return nil
		}
		return r
	case "automaticResolution":
		outcomes := decodeOutcomeList(o, "outcomes", 1)
		if r, err := NewAutomaticResolution(outcomes); err == nil && d.clean(before) {
			return r
		}
	}
	return nil
}

// decodeOutcomeList reads the array field key of o of at least min outcomes
// (required when min is 1, optional when 0). Outcomes that decoded come back.
func decodeOutcomeList(o *object, key string, min int) []SpellOutcome {
	n, p, present := o.field(key, min > 0)
	if !present {
		return nil
	}
	var out []SpellOutcome
	o.d.list(n, p, min, noUpperBound, func(_ int, el node, ep string) {
		if v := decodeOutcome(o.d, el, ep); v != nil {
			out = append(out, v)
		}
	})
	return out
}

// decodeOutcome reads an outcome object: kind spellDealsDamage (amount,
// damageType, addsModifier?), spellRestoresHitPoints and
// spellGrantsTemporaryHitPoints (amount, addsModifier?) or spellAppliesCondition
// (condition, duration). addsModifier absent is none.
func decodeOutcome(d *decoder, n node, p string) SpellOutcome {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	kind := o.kindOf("spellDealsDamage", "spellRestoresHitPoints", "spellGrantsTemporaryHitPoints", "spellAppliesCondition")
	if kind == "spellAppliesCondition" {
		condition := reqEnum(o, "condition", ParseConditionKind)
		var duration ConditionDuration
		if dn, dp, ok := o.field("duration", true); ok {
			duration = decodeConditionDuration(d, dn, dp)
		}
		if v, err := NewSpellAppliesCondition(condition, duration); err == nil && d.clean(before) {
			return v
		}
		return nil
	}
	if kind == "" {
		return nil
	}
	var amount ScalableAmount
	if an, ap, ok := o.field("amount", true); ok {
		amount = decodeScalableAmount(d, an, ap)
	}
	var selection DamageTypeSelection
	if kind == "spellDealsDamage" {
		if sn, sp, ok := o.field("damageType", true); ok {
			selection = decodeDamageType(d, sn, sp)
		}
	}
	adds := optEnum(o, "addsModifier", ParseModifierAddition)
	if !o.has("addsModifier") {
		adds = AddsNothing
	}
	if !d.clean(before) {
		return nil
	}
	var out SpellOutcome
	var err error
	switch kind {
	case "spellDealsDamage":
		out, err = NewSpellDealsDamage(amount, selection, adds)
	case "spellRestoresHitPoints":
		out, err = NewSpellRestoresHitPoints(amount, adds)
	default:
		out, err = NewSpellGrantsTemporaryHitPoints(amount, adds)
	}
	if err != nil {
		d.rule(p, err)
		return nil
	}
	return out
}

// decodeDamageType reads a damage type selection: kind fixedDamageType (type)
// or chosenDamageType (types, a set of at least two).
func decodeDamageType(d *decoder, n node, p string) DamageTypeSelection {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf("fixedDamageType", "chosenDamageType") {
	case "fixedDamageType":
		if t := reqEnum(o, "type", ParseDamageType); t.Valid() {
			s, _ := NewFixedDamageType(t)
			return s
		}
	case "chosenDamageType":
		tn, tp, ok := o.field("types", true)
		if !ok {
			return nil
		}
		types, clean := enumSet(d, tn, tp, 2, len(damageTypeNames()), ParseDamageType)
		if s, err := NewChosenDamageType(types); clean && err == nil {
			return s
		}
	}
	return nil
}

// decodeConditionDuration reads a condition duration: kind forSpellDuration,
// timedConditionDuration (amount, unit) or untilNextTurnBoundary (owner,
// boundary).
func decodeConditionDuration(d *decoder, n node, p string) ConditionDuration {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	switch o.kindOf("forSpellDuration", "timedConditionDuration", "untilNextTurnBoundary") {
	case "forSpellDuration":
		return ForSpellDuration{}
	case "timedConditionDuration":
		amount, _ := o.intAt("amount", 1, spellMaxInt)
		unit := reqEnum(o, "unit", ParseDurationUnit)
		if v, err := NewTimedConditionDuration(amount, unit); err == nil && d.clean(before) {
			return v
		}
	case "untilNextTurnBoundary":
		owner := reqEnum(o, "owner", ParseTurnOwner)
		boundary := reqEnum(o, "boundary", ParseTurnBoundary)
		if v, err := NewUntilNextTurnBoundary(owner, boundary); err == nil && d.clean(before) {
			return v
		}
	}
	return nil
}

// decodeScalableAmount reads an amount object: base, scaling?. The step that
// adds dice to a fixed base is constraintViolation at the scaling (FR-016).
func decodeScalableAmount(d *decoder, n node, p string) ScalableAmount {
	o, ok := d.object(n, p)
	if !ok {
		return ScalableAmount{}
	}
	defer o.finish()
	before := d.probs.count
	var base SpellAmount
	if bn, bp, ok := o.field("base", true); ok {
		base = decodeSpellAmount(d, bn, bp)
	}
	var scaling AmountScaling
	scalingPath := keyPath(p, "scaling")
	if sn, sp, present := o.field("scaling", false); present {
		scalingPath = sp
		scaling = decodeAmountScaling(d, sn, sp)
	}
	if !d.clean(before) {
		return ScalableAmount{}
	}
	a, err := NewScalableAmount(base, scaling)
	if err != nil {
		d.rule(scalingPath, err)
		return ScalableAmount{}
	}
	return a
}

// decodeSpellAmount reads an amount base: kind rolledSpellAmount (roll, dice
// notation as dice.Expression.Notation prints it) or fixedSpellAmount (value,
// at least 1).
func decodeSpellAmount(d *decoder, n node, p string) SpellAmount {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf("rolledSpellAmount", "fixedSpellAmount") {
	case "rolledSpellAmount":
		rn, rp, ok := o.field("roll", true)
		if !ok {
			return nil
		}
		s, ok := d.str(rn, rp)
		if !ok {
			return nil
		}
		e, err := dice.ParseExpression(s)
		if err != nil || e.Notation() != s {
			d.probs.add(rp, ProblemInvalidValue)
			return nil
		}
		a, _ := NewRolledSpellAmount(e)
		return a
	case "fixedSpellAmount":
		if v, ok := o.intAt("value", 1, spellMaxInt); ok {
			a, _ := NewFixedSpellAmount(v)
			return a
		}
	}
	return nil
}

// decodeAmountScaling reads an amount scaling: kind noAmountScaling,
// perSlotLevelAmount (aboveLevel, extraDice?, extraFlat?, each extra absent
// is 0, both 0 is outOfRange at the object) or byCharacterLevelAmount (entries
// of level and value).
func decodeAmountScaling(d *decoder, n node, p string) AmountScaling {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	switch o.kindOf("noAmountScaling", "perSlotLevelAmount", "byCharacterLevelAmount") {
	case "noAmountScaling":
		return NoAmountScaling{}
	case "perSlotLevelAmount":
		above, _ := o.intAt("aboveLevel", math.MinInt32, spellMaxInt)
		extraDice := o.optInt("extraDice", 0, spellMaxInt)
		extraFlat := o.optInt("extraFlat", 0, spellMaxInt)
		if !d.clean(before) {
			return nil
		}
		step, err := NewAmountStep(extraDice, extraFlat)
		if err != nil {
			d.rule(p, err) // both parts 0 (FR-016)
			return nil
		}
		s, _ := NewPerSlotLevelAmount(above, step)
		return s
	case "byCharacterLevelAmount":
		en, ep, ok := o.field("entries", true)
		if !ok {
			return nil
		}
		var entries []AmountLevelEntry
		d.list(en, ep, 1, noUpperBound, func(_ int, el node, lp string) {
			eo, ok := d.object(el, lp)
			if !ok {
				return
			}
			level, _ := eo.intAt("level", 1, spellMaxCharacterLevel)
			var value SpellAmount
			if vn, vp, ok := eo.field("value", true); ok {
				value = decodeSpellAmount(d, vn, vp)
			}
			eo.finish()
			entries = append(entries, AmountLevelEntry{Level: level, Value: value})
		})
		if !d.clean(before) {
			return nil
		}
		table, err := NewAmountLevelTable(entries)
		if err != nil {
			d.rule(ep, err) // the first level or the order (FR-019)
			return nil
		}
		s, _ := NewByCharacterLevelAmount(table)
		return s
	}
	return nil
}

// decodeZone reads the zone object (FR-048, FR-049): rules, mobility?,
// oncePerTurn?, designation?, damageBudget?, stopsMoverOnContact?, movement?.
// damageBudget and a rule's unitFeet are 0 for absent in the model, so the key
// decides: a key that is present holds at least 1 (outOfRange below), an
// absent one is 0. partCount is the number of parts of the model, or -1 when
// unknown; a rule whose partIndex is outside 0 to partCount-1 is
// constraintViolation (the cross-field check is the decoder's, not the
// constructor's).
func decodeZone(d *decoder, n node, p string, partCount int) (ZoneFacts, bool) {
	o, ok := d.object(n, p)
	if !ok {
		return ZoneFacts{}, false
	}
	defer o.finish()
	before := d.probs.count
	var f ZoneFactsFields
	if rn, rp, ok := o.field("rules", true); ok {
		var seen []ZoneRuleFact
		d.list(rn, rp, 1, noUpperBound, func(_ int, el node, ep string) {
			r, ok := decodeZoneRule(d, el, ep, partCount)
			if !ok {
				return
			}
			if slices.Contains(seen, r) {
				d.probs.add(ep, ProblemDuplicate)
				return
			}
			seen = append(seen, r)
		})
		f.Rules = seen
	}
	if mn, mp, present := o.field("mobility", false); present {
		f.Mobility = decodeZoneMobility(d, mn, mp)
	}
	f.OncePerTurn = o.optBool("oncePerTurn")
	f.Designation = optEnum(o, "designation", ParseDesignationPolicy)
	f.DamageBudget = o.optInt("damageBudget", 1, spellMaxInt)
	f.StopsMoverOnContact = o.optBool("stopsMoverOnContact")
	if mn, mp, present := o.field("movement", false); present {
		f.Movement = decodeZoneMovement(d, mn, mp)
	}
	if !d.clean(before) {
		return ZoneFacts{}, false
	}
	z, err := NewZoneFacts(f)
	if err != nil {
		d.rule(p, err)
		return ZoneFacts{}, false
	}
	return z, true
}

// decodeZoneRule reads a rule object: trigger, partIndex, scope, region?,
// unitFeet?, casterChoice?.
func decodeZoneRule(d *decoder, n node, p string, partCount int) (ZoneRuleFact, bool) {
	o, ok := d.object(n, p)
	if !ok {
		return ZoneRuleFact{}, false
	}
	defer o.finish()
	before := d.probs.count
	var f ZoneRuleFields
	f.Trigger = reqEnum(o, "trigger", ParseZoneTrigger)
	if in, ip, present := o.field("partIndex", true); present {
		if v, ok := d.integer(in, ip, math.MinInt32, spellMaxInt); ok {
			f.PartIndex = int(v)
			if partCount >= 0 && (f.PartIndex < 0 || f.PartIndex >= partCount) {
				d.probs.add(ip, ProblemConstraintViolation)
			}
		}
	}
	f.Scope = reqEnum(o, "scope", ParseZoneTargetScope)
	if rn, rp, present := o.field("region", false); present {
		if s, ok := d.name(rn, rp); ok {
			f.Region = ZoneRegionID(s)
		}
	}
	un, up, hasUnit := o.field("unitFeet", false)
	if hasUnit {
		if v, ok := d.integer(un, up, 1, spellMaxInt); ok {
			f.UnitFeet = int(v)
		}
	}
	f.CasterChoice = o.optBool("casterChoice")
	if f.Trigger.Valid() && d.clean(before) && (f.Trigger == TravelsWithin) != hasUnit {
		d.probs.add(up, ProblemConstraintViolation) // unitFeet missing on travelsWithin, or present on another trigger
	}
	if !d.clean(before) {
		return ZoneRuleFact{}, false
	}
	r, err := NewZoneRuleFact(f)
	if err != nil {
		d.rule(p, err)
		return ZoneRuleFact{}, false
	}
	return r, true
}

// decodeZoneMobility reads a mobility object: kind stationaryZone,
// movesWithOrigin, movedByCaster (maxFeet, action, laterTurnsOnly?) or
// driftsAtTurnStart (feet, direction).
func decodeZoneMobility(d *decoder, n node, p string) ZoneMobility {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	switch o.kindOf("stationaryZone", "movesWithOrigin", "movedByCaster", "driftsAtTurnStart") {
	case "stationaryZone":
		return StationaryZone{}
	case "movesWithOrigin":
		return MovesWithOrigin{}
	case "movedByCaster":
		maxFeet, _ := o.intAt("maxFeet", 1, spellMaxInt)
		action := reqEnum(o, "action", ParseZoneActionKind)
		later := o.optBool("laterTurnsOnly")
		if m, err := NewMovedByCaster(maxFeet, action, later); err == nil && d.clean(before) {
			return m
		}
	case "driftsAtTurnStart":
		feet, _ := o.intAt("feet", 1, spellMaxInt)
		direction := reqEnum(o, "direction", ParseDriftDirectionRule)
		if m, err := NewDriftsAtTurnStart(feet, direction); err == nil && d.clean(before) {
			return m
		}
	}
	return nil
}

// decodeZoneMovement reads a movement object: difficultTerrain?, speedHalved?,
// extraFeetPerFoot? (at least 0; absent is 0).
func decodeZoneMovement(d *decoder, n node, p string) ZoneMovementFacts {
	o, ok := d.object(n, p)
	if !ok {
		return ZoneMovementFacts{}
	}
	defer o.finish()
	terrain, halved := o.optBool("difficultTerrain"), o.optBool("speedHalved")
	extra := o.optInt("extraFeetPerFoot", 0, spellMaxInt)
	m, _ := NewZoneMovementFacts(terrain, halved, extra)
	return m
}
