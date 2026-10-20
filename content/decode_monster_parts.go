package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/dice"
)

// This file holds the field readers of the parts of a stat block: the five
// part lists, the usage, attack, saving throw, clauses, damage and
// spellcasting of a part, and the Multiattack plan (content-monsters FR-033 to
// FR-047, FR-098 to FR-101). A reader returns the zero value, or a nil
// variant, when a member had a problem; the callers compare the problem count
// to tell. A list keeps a zero value at the index of an element that had a
// problem, so the paths of the list rules still name the right index.

// The wire names of the "kind" of a usage, a Success rider, a spell frequency
// and a replacement scope.
const (
	usageKindUnlimited      = "unlimited"
	usageKindPerDay         = "per_day"
	usageKindRecharge       = "recharge"
	usageKindRechargeAfter  = "recharge_after_rest"
	usageKindOther          = "other"
	riderKindHalfDamageOnly = "half_damage_only"
	riderKindHalfDamage     = "half_damage"
	riderKindImmune         = "immune_to_part"
	frequencyKindAtWill     = "at_will"
	frequencyKindPerDay     = "per_day"
	frequencyKindPerDayEach = "per_day_each"
	scopeKindAnyAttack      = "any_attack"
	scopeKindOneAttackPart  = "one_attack_part"
	fieldKind               = "kind"
)

// kindOf reads the required "kind" member of o as one of allowed. A missing,
// non-string or unknown kind is the only problem of the object: its other
// keys are not examined. It returns "" then.
func (o *object) kindOf(allowed ...string) string {
	kn, kp, present := o.field(fieldKind, true)
	if !present {
		o.skipAll()
		return ""
	}
	s, ok := o.d.str(kn, kp)
	if !ok {
		o.skipAll()
		return ""
	}
	if !slices.Contains(allowed, s) {
		o.d.probs.add(kp, ProblemInvalidValue)
		o.skipAll()
		return ""
	}
	return s
}

// decodeMonsterParts reads traits, actions, bonusActions, reactions and
// legendaryActions, then applies the rules between parts: a second Multiattack
// and the plan names that resolve (FR-036, FR-099).
func decodeMonsterParts(o *object, f *MonsterFields) {
	d := o.d
	clean := true
	for _, l := range []struct {
		key string
		dst *[]StatBlockPart
	}{{"traits", &f.Traits}, {"actions", &f.Actions}, {"bonusActions", &f.BonusActions}, {"reactions", &f.Reactions}} {
		if n, p, ok := o.field(l.key, false); ok {
			before := d.probs.count
			*l.dst = decodePartList(d, n, p)
			clean = clean && d.clean(before)
		}
	}
	d.addIssues(o.path, multiattackIssues(f.Actions))
	if n, p, ok := o.field("legendaryActions", false); ok {
		before := d.probs.count
		f.LegendaryActions = decodeLegendaryActions(d, n, p)
		clean = clean && d.clean(before)
	}
	if clean {
		d.addIssues(o.path, planIssues(*f))
	}
}

// decodePartList reads an array of parts.
func decodePartList(d *decoder, n node, p string) []StatBlockPart {
	var out []StatBlockPart
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		out = append(out, decodePart(d, el, ep))
	})
	return out
}

// decodeLegendaryActions reads {uses, usesInLair?, actions} (FR-040). No
// action is constraintViolation at actions.
func decodeLegendaryActions(d *decoder, n node, p string) LegendaryActions {
	o, ok := d.object(n, p)
	if !ok {
		return LegendaryActions{}
	}
	defer o.finish()
	before := d.probs.count
	uses, _ := o.intAt("uses", MonsterMinUses, MonsterMaxNumber)
	inLair, _, _ := o.optIntAt("usesInLair", MonsterMinUses, MonsterMaxNumber)
	var actions []StatBlockPart
	if an, ap, present := o.field("actions", true); present {
		abefore := d.probs.count
		actions = decodePartList(d, an, ap)
		if d.clean(abefore) && len(actions) == 0 {
			d.probs.add(ap, ProblemConstraintViolation)
		}
	}
	if !d.clean(before) {
		return LegendaryActions{}
	}
	l, _ := NewLegendaryActions(uses, inLair, actions)
	return l
}

// decodePart reads {name, usage, text, attack?, savingThrow?, clauses?,
// spellcasting?, plan?} (FR-033, FR-034). It returns the zero part when a
// member had a problem.
func decodePart(d *decoder, n node, p string) StatBlockPart {
	o, ok := d.object(n, p)
	if !ok {
		return StatBlockPart{}
	}
	defer o.finish()
	before := d.probs.count
	var name string
	if nn, np, present := o.field("name", true); present {
		name, _ = d.name(nn, np)
	}
	var f StatBlockPartFields
	if un, up, present := o.field("usage", true); present {
		f.Usage = decodeUsage(d, un, up)
	}
	if tn, tp, present := o.field("text", true); present {
		f.Text, _ = d.text(tn, tp)
	}
	if an, ap, present := o.field("attack", false); present {
		f.Attack = decodeAttack(d, an, ap)
	}
	if sn, sp, present := o.field("savingThrow", false); present {
		f.SavingThrow = decodeSavingThrow(d, sn, sp)
	}
	if cn, cp, present := o.field("clauses", false); present {
		f.Clauses = decodeClauses(d, cn, cp)
	}
	if sn, sp, present := o.field("spellcasting", false); present {
		f.Spellcasting = decodeSpellcasting(d, sn, sp)
	}
	if pn, pp, present := o.field("plan", false); present {
		f.Plan = decodePlan(d, pn, pp)
	}
	if !d.clean(before) {
		return StatBlockPart{}
	}
	if is := partProblems(name, f); len(is) > 0 {
		d.addIssues(p, is)
		return StatBlockPart{}
	}
	part, _ := NewStatBlockPart(name, f)
	return part
}

// decodeUsage reads {kind, count?, inLairCount?, low?, high?, text?} (FR-035).
// It returns nil when a member had a problem.
func decodeUsage(d *decoder, n node, p string) PartUsage {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	var out PartUsage
	switch o.kindOf(usageKindUnlimited, usageKindPerDay, usageKindRecharge, usageKindRechargeAfter, usageKindOther) {
	case usageKindUnlimited:
		out = PartUnlimited{}
	case usageKindRechargeAfter:
		out = PartRechargeAfterRest{}
	case usageKindPerDay:
		count, _ := o.intAt("count", MonsterMinUses, MonsterMaxNumber)
		inLair, _, _ := o.optIntAt("inLairCount", MonsterMinUses, MonsterMaxNumber)
		if d.clean(before) {
			out, _ = nilUsage(NewPartPerDay(count, inLair))
		}
	case usageKindRecharge:
		low, _ := o.intAt("low", MonsterMinRecharge, MonsterMaxRecharge)
		high, _ := o.intAt("high", MonsterMinRecharge, MonsterMaxRecharge)
		if d.clean(before) {
			if low > high {
				d.probs.add(p, ProblemConstraintViolation)
			} else {
				out, _ = nilUsage(NewPartRecharge(low, high))
			}
		}
	case usageKindOther:
		if tn, tp, present := o.field("text", true); present {
			if s, ok := d.text(tn, tp); ok {
				out, _ = nilUsage(NewPartUsageOther(s))
			}
		}
	}
	if !d.clean(before) {
		return nil
	}
	return out
}

// nilUsage returns v as a PartUsage, or nil when err is not nil.
func nilUsage[V PartUsage](v V, err error) (PartUsage, error) {
	if err != nil {
		return nil, err
	}
	return v, nil
}

// decodeAttack reads {kind, bonus, qualifier?, reach?, range?} (FR-037). It
// returns the zero attack when a member had a problem.
func decodeAttack(d *decoder, n node, p string) StatBlockAttack {
	o, ok := d.object(n, p)
	if !ok {
		return StatBlockAttack{}
	}
	defer o.finish()
	before := d.probs.count
	var f StatBlockAttackFields
	if kn, kp, present := o.field(fieldKind, true); present {
		f.Kind, _ = enumField(d, kn, kp, ParseStatBlockAttackKind)
	}
	f.Bonus, _ = o.intAt("bonus", -MonsterMaxNumber, MonsterMaxNumber)
	f.Qualifier = o.optText("qualifier")
	f.Reach, f.HasReach, _ = o.optIntAt("reach", 0, MonsterMaxNumber)
	if rn, rp, present := o.field("range", false); present {
		f.Range = decodeAttackRange(d, rn, rp)
	}
	if !d.clean(before) {
		return StatBlockAttack{}
	}
	a, _ := NewStatBlockAttack(f)
	return a
}

// decodeAttackRange reads {normal, long?} (FR-037). A long range that is not
// above the normal is constraintViolation at long.
func decodeAttackRange(d *decoder, n node, p string) StatBlockAttackRange {
	o, ok := d.object(n, p)
	if !ok {
		return StatBlockAttackRange{}
	}
	defer o.finish()
	before := d.probs.count
	normal, normalOK := o.intAt("normal", MonsterMinRange, MonsterMaxNumber)
	long, hasLong, longOK := o.optIntAt("long", MonsterMinRange, MonsterMaxNumber)
	if hasLong && longOK && normalOK && long <= normal {
		d.probs.add(keyPath(p, "long"), ProblemConstraintViolation)
	}
	if !d.clean(before) {
		return StatBlockAttackRange{}
	}
	r, _ := NewStatBlockAttackRange(normal, long)
	return r
}

// decodeSavingThrow reads {ability, dc, target?, prefix?} (FR-039). It returns
// the zero saving throw when a member had a problem.
func decodeSavingThrow(d *decoder, n node, p string) StatBlockSavingThrow {
	o, ok := d.object(n, p)
	if !ok {
		return StatBlockSavingThrow{}
	}
	defer o.finish()
	before := d.probs.count
	var ab ability.Ability
	if an, ap, present := o.field("ability", true); present {
		ab, _ = enumField(d, an, ap, parseAbilityExact)
	}
	dc, _ := o.intAt("dc", MonsterMinSaveDC, MonsterMaxNumber)
	target := o.optText("target")
	prefix := o.optText("prefix")
	if !d.clean(before) {
		return StatBlockSavingThrow{}
	}
	s, _ := NewStatBlockSavingThrow(ab, dc, target, prefix)
	return s
}

// decodeClauses reads an array of effect clauses; a clause with a problem is
// the zero clause at its index.
func decodeClauses(d *decoder, n node, p string) []EffectClause {
	var out []EffectClause
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		out = append(out, decodeClause(d, el, ep))
	})
	return out
}

// decodeClause reads {label, text, successRider?, damage?} (FR-038, FR-100). A
// rider on a label other than success is constraintViolation at the rider.
func decodeClause(d *decoder, n node, p string) EffectClause {
	o, ok := d.object(n, p)
	if !ok {
		return EffectClause{}
	}
	defer o.finish()
	before := d.probs.count
	var label EffectClauseLabel
	if ln, lp, present := o.field("label", true); present {
		label, _ = enumField(d, ln, lp, ParseEffectClauseLabel)
	}
	var text string
	if tn, tp, present := o.field("text", true); present {
		text, _ = d.text(tn, tp)
	}
	var rider SaveSuccessRider
	if rn, rp, present := o.field("successRider", false); present {
		rbefore := d.probs.count
		rider = decodeRider(d, rn, rp)
		if d.clean(rbefore) && label.Valid() && label != ClauseSuccess {
			d.probs.add(rp, ProblemConstraintViolation)
		}
	}
	var damage []StatBlockDamage
	if dn, dp, present := o.field("damage", false); present {
		d.list(dn, dp, 0, noUpperBound, func(_ int, el node, ep string) {
			damage = append(damage, decodeDamage(d, el, ep))
		})
	}
	if !d.clean(before) {
		return EffectClause{}
	}
	c, _ := NewEffectClause(label, text, rider, damage)
	return c
}

// decodeRider reads {kind, hours?} (FR-100). It returns nil when a member had
// a problem.
func decodeRider(d *decoder, n node, p string) SaveSuccessRider {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf(riderKindHalfDamageOnly, riderKindHalfDamage, riderKindImmune) {
	case riderKindHalfDamageOnly:
		return HalfDamageOnly{}
	case riderKindHalfDamage:
		return HalfDamage{}
	case riderKindImmune:
		if hours, ok := o.intAt("hours", 1, MonsterMaxNumber); ok {
			r, _ := NewImmuneToPartForHours(hours)
			return r
		}
	}
	return nil
}

// decodeDamage reads {number, dice?, damageType | text,
// underwaterResistanceDenied?} (FR-042, FR-043, FR-101). None of damageType and
// text is missingField at damageType, both is constraintViolation at the
// instance, and the flag on anything but Fire is constraintViolation at the
// flag. It returns the zero damage when a member had a problem.
func decodeDamage(d *decoder, n node, p string) StatBlockDamage {
	o, ok := d.object(n, p)
	if !ok {
		return StatBlockDamage{}
	}
	defer o.finish()
	before := d.probs.count
	number, _ := o.intAt("number", 0, MonsterMaxNumber)
	var e dice.Expression
	if dn, dp, present := o.field("dice", false); present {
		if do, ok := d.object(dn, dp); ok {
			e, _ = decodeDiceMembers(do)
			do.finish()
		}
	}
	tn, tp, hasType := o.field("damageType", false)
	xn, xp, hasText := o.field("text", false)
	switch {
	case !hasType && !hasText:
		d.probs.add(tp, ProblemMissingField)
	case hasType && hasText:
		d.probs.add(p, ProblemConstraintViolation)
	}
	var ref DamageTypeRef
	switch {
	case hasType:
		if t, ok := enumField(d, tn, tp, ParseDamageType); ok {
			ref = SpecificDamageType{t: t}
		}
	case hasText:
		if s, ok := d.text(xn, xp); ok {
			ref = DescribedDamageType{text: s}
		}
	}
	flag := false
	if fn, fp, present := o.field("underwaterResistanceDenied", false); present {
		var ok bool
		flag, ok = d.boolean(fn, fp)
		if s, isSpecific := ref.(SpecificDamageType); ok && flag && ref != nil && (!isSpecific || s.t != DamageFire) {
			d.probs.add(fp, ProblemConstraintViolation)
		}
	}
	if !d.clean(before) {
		return StatBlockDamage{}
	}
	dm, _ := NewStatBlockDamage(number, e, ref, flag)
	return dm
}

// decodeSpellcasting reads {ability, dc?, attackBonus?, components?, groups}
// (FR-044 to FR-046). No group is constraintViolation at groups. It returns
// the zero value when a member had a problem.
func decodeSpellcasting(d *decoder, n node, p string) MonsterSpellcasting {
	o, ok := d.object(n, p)
	if !ok {
		return MonsterSpellcasting{}
	}
	defer o.finish()
	before := d.probs.count
	var f MonsterSpellcastingFields
	if an, ap, present := o.field("ability", true); present {
		f.Ability, _ = enumField(d, an, ap, parseAbilityExact)
	}
	f.SaveDC, f.HasSaveDC, _ = o.optIntAt("dc", MonsterMinSaveDC, MonsterMaxNumber)
	f.AttackBonus, f.HasAttackBonus, _ = o.optIntAt("attackBonus", -MonsterMaxNumber, MonsterMaxNumber)
	f.Components = o.optText("components")
	if gn, gp, present := o.field("groups", true); present {
		gbefore := d.probs.count
		d.list(gn, gp, 0, noUpperBound, func(_ int, el node, ep string) {
			f.Groups = append(f.Groups, decodeSpellGroup(d, el, ep))
		})
		if d.clean(gbefore) && len(f.Groups) == 0 {
			d.probs.add(gp, ProblemConstraintViolation)
		}
	}
	if !d.clean(before) {
		return MonsterSpellcasting{}
	}
	s, _ := NewMonsterSpellcasting(f)
	return s
}

// decodeSpellGroup reads {frequency, spells} (FR-045). No spell is
// constraintViolation at spells. It returns the zero group when a member had a
// problem.
func decodeSpellGroup(d *decoder, n node, p string) MonsterSpellGroup {
	o, ok := d.object(n, p)
	if !ok {
		return MonsterSpellGroup{}
	}
	defer o.finish()
	before := d.probs.count
	var freq SpellFrequency
	if fn, fp, present := o.field("frequency", true); present {
		freq = decodeFrequency(d, fn, fp)
	}
	var spells []MonsterSpellEntry
	if sn, sp, present := o.field("spells", true); present {
		sbefore := d.probs.count
		d.list(sn, sp, 0, noUpperBound, func(_ int, el node, ep string) {
			spells = append(spells, decodeSpellEntry(d, el, ep))
		})
		if d.clean(sbefore) && len(spells) == 0 {
			d.probs.add(sp, ProblemConstraintViolation)
		}
	}
	if !d.clean(before) {
		return MonsterSpellGroup{}
	}
	g, _ := NewMonsterSpellGroup(freq, spells)
	return g
}

// decodeFrequency reads {kind, count?} (FR-045). It returns nil when a member
// had a problem.
func decodeFrequency(d *decoder, n node, p string) SpellFrequency {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf(frequencyKindAtWill, frequencyKindPerDay, frequencyKindPerDayEach) {
	case frequencyKindAtWill:
		return SpellFrequencyAtWill{}
	case frequencyKindPerDay:
		if count, ok := o.intAt("count", MonsterMinUses, MonsterMaxNumber); ok {
			f, _ := NewSpellFrequencyPerDay(count)
			return f
		}
	case frequencyKindPerDayEach:
		if count, ok := o.intAt("count", MonsterMinUses, MonsterMaxNumber); ok {
			f, _ := NewSpellFrequencyPerDayEach(count)
			return f
		}
	}
	return nil
}

// decodeSpellEntry reads {name, note?, castAtLevel?} (FR-046). It returns the
// zero entry when a member had a problem.
func decodeSpellEntry(d *decoder, n node, p string) MonsterSpellEntry {
	o, ok := d.object(n, p)
	if !ok {
		return MonsterSpellEntry{}
	}
	defer o.finish()
	before := d.probs.count
	var name string
	if nn, np, present := o.field("name", true); present {
		name, _ = d.name(nn, np)
	}
	note := o.optText("note")
	level, _, _ := o.optIntAt("castAtLevel", MonsterMinSpellLevel, MonsterMaxSpellLevel)
	if !d.clean(before) {
		return MonsterSpellEntry{}
	}
	e, _ := NewMonsterSpellEntry(name, note, level)
	return e
}

// decodePlan reads {variants} (FR-098, FR-099). The rules of the plan run when
// every variant decoded; the name resolution is the monster's (planIssues). It
// returns the zero plan when a member had a problem.
func decodePlan(d *decoder, n node, p string) MultiattackPlan {
	o, ok := d.object(n, p)
	if !ok {
		return MultiattackPlan{}
	}
	defer o.finish()
	before := d.probs.count
	var plan MultiattackPlan
	if vn, vp, present := o.field("variants", true); present {
		d.list(vn, vp, 0, noUpperBound, func(_ int, el node, ep string) {
			plan.variants = append(plan.variants, decodeVariant(d, el, ep))
		})
	}
	if !d.clean(before) {
		return MultiattackPlan{}
	}
	if is := plan.problems(); len(is) > 0 {
		d.addIssues(p, is)
		return MultiattackPlan{}
	}
	return plan
}

// decodeVariant reads {requiresUsed?, groups, additionalUses?, replacements?}
// (FR-098). It returns the zero variant when a member had a problem.
func decodeVariant(d *decoder, n node, p string) MultiattackVariant {
	o, ok := d.object(n, p)
	if !ok {
		return MultiattackVariant{}
	}
	defer o.finish()
	before := d.probs.count
	var v MultiattackVariant
	if rn, rp, present := o.field("requiresUsed", false); present {
		v.requiresUsed, _ = d.name(rn, rp)
	}
	if gn, gp, present := o.field("groups", true); present {
		if _, ok := d.arrayLen(gn, gp, 1, noUpperBound); ok {
			d.each(gn, gp, func(_ int, el node, ep string) {
				v.groups = append(v.groups, decodeGroup(d, el, ep))
			})
		}
	}
	if an, ap, present := o.field("additionalUses", false); present {
		d.list(an, ap, 0, noUpperBound, func(_ int, el node, ep string) {
			v.additionalUse = append(v.additionalUse, decodeAdditionalUse(d, el, ep))
		})
	}
	if rn, rp, present := o.field("replacements", false); present {
		d.list(rn, rp, 0, noUpperBound, func(_ int, el node, ep string) {
			v.replacements = append(v.replacements, decodeReplacement(d, el, ep))
		})
	}
	if !d.clean(before) {
		return MultiattackVariant{}
	}
	if is := v.problems(); len(is) > 0 {
		d.addIssues(p, is)
		return MultiattackVariant{}
	}
	return v
}

// decodeGroup reads {count, parts} (FR-098): a repeated part is duplicate at
// the later one.
func decodeGroup(d *decoder, n node, p string) AttackSlotGroup {
	o, ok := d.object(n, p)
	if !ok {
		return AttackSlotGroup{}
	}
	defer o.finish()
	before := d.probs.count
	count, _ := o.intAt("count", MonsterMinAttackCount, MonsterMaxNumber)
	var parts []string
	if pn, pp, present := o.field("parts", true); present {
		parts = d.strings(pn, pp, 1, true, d.name)
	}
	if !d.clean(before) {
		return AttackSlotGroup{}
	}
	g, _ := NewAttackSlotGroup(count, parts)
	return g
}

// decodeAdditionalUse reads {options} (FR-098).
func decodeAdditionalUse(d *decoder, n node, p string) AdditionalUse {
	o, ok := d.object(n, p)
	if !ok {
		return AdditionalUse{}
	}
	defer o.finish()
	before := d.probs.count
	var options []string
	if on, op, present := o.field("options", true); present {
		options = d.strings(on, op, 1, true, d.name)
	}
	if !d.clean(before) {
		return AdditionalUse{}
	}
	u, _ := NewAdditionalUse(options)
	return u
}

// decodeReplacement reads {count, scope, options, limit} (FR-098).
func decodeReplacement(d *decoder, n node, p string) ReplacementRule {
	o, ok := d.object(n, p)
	if !ok {
		return ReplacementRule{}
	}
	defer o.finish()
	before := d.probs.count
	count, _ := o.intAt("count", MonsterMinAttackCount, MonsterMaxNumber)
	var scope ReplacementScope
	if sn, sp, present := o.field("scope", true); present {
		scope = decodeScope(d, sn, sp)
	}
	var options []ReplacementOption
	if on, op, present := o.field("options", true); present {
		seen := map[string]struct{}{}
		d.list(on, op, 1, noUpperBound, func(_ int, el node, ep string) {
			opt := decodeReplacementOption(d, el, ep)
			if _, dup := seen[opt.part]; dup && opt.part != "" {
				d.probs.add(keyPath(ep, "part"), ProblemDuplicate)
			}
			seen[opt.part] = struct{}{}
			options = append(options, opt)
		})
	}
	var limit ReplacementLimit
	if ln, lp, present := o.field("limit", true); present {
		limit, _ = enumField(d, ln, lp, ParseReplacementLimit)
	}
	if !d.clean(before) {
		return ReplacementRule{}
	}
	r, _ := NewReplacementRule(count, scope, options, limit)
	return r
}

// decodeScope reads {kind, part?} (FR-098). It returns nil when a member had a
// problem.
func decodeScope(d *decoder, n node, p string) ReplacementScope {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	switch o.kindOf(scopeKindAnyAttack, scopeKindOneAttackPart) {
	case scopeKindAnyAttack:
		return ReplacementScopeAnyAttack{}
	case scopeKindOneAttackPart:
		if pn, pp, present := o.field("part", true); present {
			if name, ok := d.name(pn, pp); ok {
				s, _ := NewReplacementScopeOneAttackPart(name)
				return s
			}
		}
	}
	return nil
}

// decodeReplacementOption reads {part, spell?{name, castAtLevel?}} (FR-098).
// It returns the zero option when a member had a problem.
func decodeReplacementOption(d *decoder, n node, p string) ReplacementOption {
	o, ok := d.object(n, p)
	if !ok {
		return ReplacementOption{}
	}
	defer o.finish()
	before := d.probs.count
	var part, spell string
	var level int
	if pn, pp, present := o.field("part", true); present {
		part, _ = d.name(pn, pp)
	}
	if so, ok := o.sub("spell", false); ok {
		if sn, sp, present := so.field("name", true); present {
			spell, _ = d.name(sn, sp)
		}
		level, _, _ = so.optIntAt("castAtLevel", MonsterMinSpellLevel, MonsterMaxSpellLevel)
		so.finish()
	}
	if !d.clean(before) {
		return ReplacementOption{}
	}
	opt, _ := NewReplacementOption(part, spell, level)
	return opt
}
