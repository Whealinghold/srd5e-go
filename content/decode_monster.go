package content

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/dice"
)

// This file holds the row of the monsters array (content-monsters FR-048 to
// FR-054) and the field readers of the stat block header, the combat
// highlights and the abilities and details. The parts, notation, plan and
// spellcasting are in decode_monster_parts.go. A reader returns its value and
// whether it decoded with no problem; a problem drops the entry (the
// framework), so no reader needs to recover. The rules between members of one
// list are the same helpers the constructor uses (monster.go), reported at
// the path they name.

// clean reports whether no problem was added since the count was before.
func (d *decoder) clean(before int) bool { return d.probs.count == before }

// addIssues reports issues, whose paths are relative to base.
func (d *decoder) addIssues(base string, is []issue) {
	for _, i := range is {
		d.probs.add(base+"/"+i.path, i.kind)
	}
}

// sub reads the field key as an object. ok is false when it is absent (a
// missingField problem when required) or not an object.
func (o *object) sub(key string, required bool) (*object, bool) {
	n, p, present := o.field(key, required)
	if !present {
		return nil, false
	}
	return o.d.object(n, p)
}

// intAt reads the required integer field key in [lo, hi] and says whether it
// decoded.
func (o *object) intAt(key string, lo, hi int64) (int, bool) {
	n, p, present := o.field(key, true)
	if !present {
		return 0, false
	}
	v, ok := o.d.integer(n, p, lo, hi)
	return int(v), ok
}

// optIntAt reads the optional integer field key in [lo, hi]. present is true
// when the key is there, ok when it decoded.
func (o *object) optIntAt(key string, lo, hi int64) (v int, present, ok bool) {
	n, p, present := o.field(key, false)
	if !present {
		return 0, false, true
	}
	x, ok := o.d.integer(n, p, lo, hi)
	return int(x), true, ok
}

// strings reads an array of lo or more strings, each by read, and a repeat is
// duplicate at the later element when unique. The caller tells a problem from
// the count. The strings that decoded come back.
func (d *decoder) strings(n node, path string, lo int, unique bool, read func(node, string) (string, bool)) []string {
	if _, ok := d.arrayLen(n, path, lo, noUpperBound); !ok {
		return nil
	}
	var out []string
	seen := map[string]struct{}{}
	d.each(n, path, func(_ int, el node, p string) {
		s, ok := read(el, p)
		if !ok {
			return
		}
		if _, dup := seen[s]; unique && dup {
			d.probs.add(p, ProblemDuplicate)
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	})
	return out
}

// decodeMonster is the row of the monsters array: id, name, source, sizes,
// swarmOf?, type, tags?, alignment, armorClass, initiative, hitPoints, speeds,
// abilities, skills?, vulnerabilities?, resistances?, immunities?, gear?,
// senses, languages, challengeRating, experiencePoints,
// experiencePointsInLair?, traits?, actions?, bonusActions?, reactions?,
// legendaryActions?. The entry holds no reference to another kind, so the row
// registers no site.
func decodeMonster(o *object) decodedEntry {
	d := o.d
	before := d.probs.count
	var f MonsterFields
	var id ID
	var idOK bool
	if n, p, ok := o.field("id", true); ok {
		id, idOK = d.id(n, p)
	}
	res := decodedEntry{id: id, idOK: idOK}
	var name string
	nameOK := false
	if n, p, ok := o.field("name", true); ok {
		name, nameOK = d.name(n, p)
	}
	if idOK && nameOK {
		if derived, err := DeriveMonsterID(name); err != nil || derived.String() != id.String() {
			d.probs.add(keyPath(o.path, "id"), ProblemConstraintViolation)
		}
	}
	source := decodeMonsterSource(o)
	decodeMonsterHeader(o, &f)
	decodeMonsterHighlights(o, &f)
	decodeMonsterAbilities(o, &f)
	decodeMonsterDetails(o, &f)
	decodeMonsterParts(o, &f)
	if !d.clean(before) {
		return res
	}
	if e, err := NewMonster(MonsterID(id.String()), name, source, f); err == nil {
		res.entry = e
	}
	return res
}

// decodeMonsterSource reads the required source {document, section, page}
// (FR-004): the page of 258 to 364 is outOfRange, a section other than the two
// of the SRD invalidValue. It returns the zero citation when a part had a
// problem.
func decodeMonsterSource(o *object) SourceCitation {
	d := o.d
	so, ok := o.sub("source", true)
	if !ok {
		return SourceCitation{}
	}
	defer so.finish()
	before := d.probs.count
	var document, section string
	var page int
	if n, p, ok := so.field("document", true); ok {
		document, _ = d.line(n, p)
	}
	if n, p, ok := so.field("section", true); ok {
		if s, ok := d.line(n, p); ok {
			if monsterSectionOK(s) {
				section = s
			} else {
				d.probs.add(p, ProblemInvalidValue)
			}
		}
	}
	page, _ = so.intAt("page", MonsterMinPage, MonsterMaxPage)
	if !d.clean(before) {
		return SourceCitation{}
	}
	c, _ := NewSourceCitation(document, section, page)
	return c
}

// decodeMonsterHeader reads sizes, swarmOf, type, tags and alignment (FR-008,
// FR-009, FR-011, FR-050).
func decodeMonsterHeader(o *object, f *MonsterFields) {
	d := o.d
	if n, p, ok := o.field("sizes", true); ok {
		before := d.probs.count
		if _, ok := d.arrayLen(n, p, 0, noUpperBound); ok {
			d.each(n, p, func(_ int, el node, ep string) {
				if z, ok := enumField(d, el, ep, ParseCreatureSize); ok {
					f.Sizes = append(f.Sizes, z)
				}
			})
		}
		if d.clean(before) {
			d.addIssues(o.path, sizeIssues(f.Sizes))
		}
	}
	if n, p, ok := o.field("swarmOf", false); ok {
		if z, ok := enumField(d, n, p, ParseCreatureSize); ok {
			f.Swarm, _ = NewSwarmComposition(z)
		}
	}
	if n, p, ok := o.field("type", true); ok {
		f.Type, _ = enumField(d, n, p, ParseCreatureType)
	}
	if n, p, ok := o.field("tags", false); ok {
		f.Tags = d.strings(n, p, 0, false, d.text)
	}
	if n, p, ok := o.field("alignment", true); ok {
		f.Alignment, _ = enumField(d, n, p, ParseAlignment)
	}
}

// decodeMonsterHighlights reads armorClass, initiative, hitPoints and speeds
// (FR-013 to FR-017).
func decodeMonsterHighlights(o *object, f *MonsterFields) {
	f.ArmorClass, _ = o.intAt("armorClass", 0, MonsterMaxNumber)
	if io, ok := o.sub("initiative", true); ok {
		mod, modOK := io.intAt("modifier", -MonsterMaxNumber, MonsterMaxNumber)
		score, scoreOK := io.intAt("score", -MonsterMaxNumber, MonsterMaxNumber)
		io.finish()
		if modOK && scoreOK {
			f.Initiative, _ = NewStatBlockInitiative(mod, score)
		}
	}
	if ho, ok := o.sub("hitPoints", true); ok {
		average, avgOK := ho.intAt("average", MonsterMinHitPointsAverage, MonsterMaxNumber)
		e, diceOK := decodeDiceMembers(ho) // the dice members sit beside the average
		ho.finish()
		if avgOK && diceOK {
			f.HitPoints, _ = NewHitPointsEntry(average, e)
		}
	}
	if n, p, ok := o.field("speeds", true); ok {
		f.Speeds = decodeSpeeds(o.d, n, p, o.path)
	}
}

// decodeSpeeds reads the speeds array and applies the list rules of FR-016.
func decodeSpeeds(d *decoder, n node, p, entryPath string) []SpeedEntry {
	before := d.probs.count
	var out []SpeedEntry
	if _, ok := d.arrayLen(n, p, 0, noUpperBound); !ok {
		return nil
	}
	d.each(n, p, func(_ int, el node, ep string) {
		out = append(out, decodeSpeed(d, el, ep)) // a zero entry stands for one with a problem
	})
	if d.clean(before) {
		d.addIssues(entryPath, speedIssues(out))
	}
	return out
}

// decodeSpeed reads {modes, feet, hover?, qualifier?} (FR-016). It returns the
// zero entry when a member had a problem.
func decodeSpeed(d *decoder, n node, p string) SpeedEntry {
	o, ok := d.object(n, p)
	if !ok {
		return SpeedEntry{}
	}
	defer o.finish()
	before := d.probs.count
	var modes []SpeedMode
	if mn, mp, present := o.field("modes", true); present {
		modes, _ = enumSet(d, mn, mp, 1, noUpperBound, ParseSpeedMode)
	}
	feet, _ := o.intAt("feet", 0, MonsterMaxNumber)
	hover := false
	if hn, hp, present := o.field("hover", false); present {
		var ok bool
		hover, ok = d.boolean(hn, hp)
		if ok && hover && d.clean(before) && !containsMode(modes, SpeedFly) {
			d.probs.add(hp, ProblemConstraintViolation)
		}
	}
	qualifier := o.optText("qualifier")
	if !d.clean(before) {
		return SpeedEntry{}
	}
	e, _ := NewSpeedEntry(modes, feet, hover, qualifier)
	return e
}

func containsMode(modes []SpeedMode, m SpeedMode) bool {
	for _, x := range modes {
		if x == m {
			return true
		}
	}
	return false
}

// abilityKey returns the JSON key of an ability in the abilities object.
func abilityKey(a ability.Ability) string {
	switch a {
	case ability.Strength:
		return "str"
	case ability.Dexterity:
		return "dex"
	case ability.Constitution:
		return "con"
	case ability.Intelligence:
		return "int"
	case ability.Wisdom:
		return "wis"
	case ability.Charisma:
		return "cha"
	}
	return ""
}

// decodeMonsterAbilities reads the required abilities object: one
// {score, save} member per ability, keyed str, dex, con, int, wis and cha
// (FR-018). A missing ability or save is missingField, a score outside 1 to 30
// outOfRange.
func decodeMonsterAbilities(o *object, f *MonsterFields) {
	d := o.d
	ao, ok := o.sub("abilities", true)
	if !ok {
		return
	}
	defer ao.finish()
	before := d.probs.count
	scores := map[ability.Ability]ability.Score{}
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		eo, ok := ao.sub(abilityKey(a), true)
		if !ok {
			continue
		}
		if v, ok := eo.intAt("score", ability.MinScore, ability.MaxScore); ok {
			scores[a], _ = ability.NewScore(v)
		}
		if v, ok := eo.intAt("save", -MonsterMaxNumber, MonsterMaxNumber); ok {
			saves[a] = v
		}
		eo.finish()
	}
	if !d.clean(before) {
		return
	}
	if s, err := ability.NewScores(scores); err == nil {
		f.Abilities, _ = NewAbilityBlock(s, saves)
	}
}

// decodeMonsterDetails reads skills, the three defence lists, gear, senses,
// languages, challengeRating and the experience points (FR-019 to FR-032).
func decodeMonsterDetails(o *object, f *MonsterFields) {
	d := o.d
	if n, p, ok := o.field("skills", false); ok {
		f.Skills = decodeSkills(d, n, p)
	}
	for _, list := range []struct {
		key        string
		dst        *[]DefenseEntry
		conditions bool
	}{{"vulnerabilities", &f.Vulnerabilities, false}, {"resistances", &f.Resistances, false}, {"immunities", &f.Immunities, true}} {
		if n, p, ok := o.field(list.key, false); ok {
			*list.dst = decodeDefenses(d, n, p, o.path, list.key, list.conditions)
		}
	}
	if n, p, ok := o.field("gear", false); ok {
		f.Gear = d.strings(n, p, 0, false, d.text)
	}
	if so, ok := o.sub("senses", true); ok {
		if n, p, present := so.field("special", false); present {
			f.Senses = decodeSenses(d, n, p)
		}
		f.PassivePerception, _ = so.intAt("passivePerception", -MonsterMaxNumber, MonsterMaxNumber)
		so.finish()
	}
	if lo, ok := o.sub("languages", true); ok {
		f.Languages = decodeLanguages(d, lo)
		lo.finish()
	}
	if n, p, ok := o.field("challengeRating", true); ok {
		f.ChallengeRating, _ = enumField(d, n, p, ParseChallengeRating)
	}
	xp, xpOK := o.intAt("experiencePoints", 0, MonsterMaxNumber)
	if n, p, present := o.field("experiencePointsInLair", false); present {
		if lair, ok := d.integer(n, p, 0, MonsterMaxNumber); ok {
			f.ExperiencePointsInLair = int(lair)
			if xpOK {
				d.addIssues(o.path, xpIssues(xp, int(lair), true))
			}
		}
	}
	f.ExperiencePoints = xp
}

// decodeSkills reads the skills object: a skill name to its bonus. A name
// outside the vocabulary is invalidValue at its key.
func decodeSkills(d *decoder, n node, p string) map[Skill]int {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	out := map[Skill]int{}
	seen := map[string]struct{}{}
	for _, m := range o.members {
		if _, done := seen[m.key]; done {
			continue
		}
		seen[m.key] = struct{}{}
		vn, vp, _ := o.field(m.key, false)
		s, err := ParseSkill(m.key)
		if err != nil || !m.keyOK {
			d.probs.add(vp, ProblemInvalidValue)
			continue
		}
		if v, ok := d.integer(vn, vp, -MonsterMaxNumber, MonsterMaxNumber); ok {
			out[s] = int(v)
		}
	}
	return out
}

// decodeDefenses reads one defence list. An entry that had a problem is nil,
// and the list rules of FR-024 and FR-025 run when every entry decoded.
func decodeDefenses(d *decoder, n node, p, entryPath, list string, conditionsAllowed bool) []DefenseEntry {
	before := d.probs.count
	if _, ok := d.arrayLen(n, p, 0, noUpperBound); !ok {
		return nil
	}
	var out []DefenseEntry
	d.each(n, p, func(_ int, el node, ep string) {
		out = append(out, decodeDefense(d, el, ep))
	})
	if d.clean(before) {
		d.addIssues(entryPath, defenseIssues(list, out, conditionsAllowed))
	}
	return out
}

// decodeDefense reads {damageType | condition | text, qualifier?}. None of
// the three is missingField at damageType, more than one is
// constraintViolation at the entry. It returns nil when a member had a
// problem.
func decodeDefense(d *decoder, n node, p string) DefenseEntry {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	before := d.probs.count
	dn, dp, hasDamage := o.field("damageType", false)
	cn, cp, hasCondition := o.field("condition", false)
	tn, tp, hasText := o.field("text", false)
	qualifier := o.optText("qualifier")
	switch count := boolCount(hasDamage, hasCondition, hasText); {
	case count == 0:
		d.probs.add(dp, ProblemMissingField)
	case count > 1:
		d.probs.add(p, ProblemConstraintViolation)
	}
	var out DefenseEntry
	switch {
	case hasDamage:
		if t, ok := enumField(d, dn, dp, ParseDamageType); ok {
			out, _ = nilOnErr(NewDamageTypeDefense(t, qualifier))
		}
	case hasCondition:
		if c, ok := enumField(d, cn, cp, ParseConditionKind); ok {
			out, _ = nilOnErr(NewConditionDefense(c, qualifier))
		}
	case hasText:
		if s, ok := d.text(tn, tp); ok {
			out, _ = nilOnErr(NewDescribedDefense(s))
		}
	}
	if !d.clean(before) {
		return nil
	}
	return out
}

// boolCount returns how many of its arguments are true.
func boolCount(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}

// nilOnErr returns v as T, or the zero T when err is not nil.
func nilOnErr[V DefenseEntry](v V, err error) (DefenseEntry, error) {
	if err != nil {
		return nil, err
	}
	return v, nil
}

// decodeSenses reads the special senses (FR-027, FR-028).
func decodeSenses(d *decoder, n node, p string) []SenseEntry {
	var out []SenseEntry
	d.list(n, p, 0, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		defer o.finish()
		before := d.probs.count
		var sense SpecialSense
		if kn, kp, present := o.field("kind", true); present {
			sense, _ = enumField(d, kn, kp, ParseSpecialSense)
		}
		feet, _ := o.intAt("feet", 0, MonsterMaxNumber)
		qualifier := o.optText("qualifier")
		if d.clean(before) {
			e, _ := NewSenseEntry(sense, feet, qualifier)
			out = append(out, e)
		}
	})
	return out
}

// decodeLanguages reads {phrases?, telepathyFeet?, telepathyQualifier?}
// (FR-029). A qualifier with no range is constraintViolation at the
// qualifier.
func decodeLanguages(d *decoder, o *object) LanguagesEntry {
	before := d.probs.count
	var phrases []string
	if n, p, ok := o.field("phrases", false); ok {
		phrases = d.strings(n, p, 0, false, d.text)
	}
	feet, hasFeet, _ := o.optIntAt("telepathyFeet", 0, MonsterMaxNumber)
	qn, qp, hasQualifier := o.field("telepathyQualifier", false)
	qualifier := ""
	if hasQualifier {
		qualifier, _ = d.text(qn, qp)
		if !hasFeet {
			d.probs.add(qp, ProblemConstraintViolation)
		}
	}
	if !d.clean(before) {
		return LanguagesEntry{}
	}
	l, _ := NewLanguagesEntry(phrases)
	if hasFeet {
		l, _ = l.WithTelepathy(feet, qualifier)
	}
	return l
}

// decodeDiceMembers reads the count, faces and modifier members of the object
// o as dice (FR-015, FR-043): a count outside 1 to dice.MaxCount, a die that
// is not one of the allowed faces and a modifier beyond dice.MaxModifier are
// outOfRange. ok is false when a member had a problem.
func decodeDiceMembers(o *object) (dice.Expression, bool) {
	d := o.d
	before := d.probs.count
	count, _ := o.intAt("count", MonsterMinHitDiceCount, dice.MaxCount)
	faces := 0
	if fn, fp, present := o.field("faces", true); present {
		v, ok := d.integer(fn, fp, 1, MonsterMaxNumber)
		faces = int(v)
		if ok && !monsterDieFacesOK(faces) {
			d.probs.add(fp, ProblemOutOfRange)
		}
	}
	modifier, _ := o.intAt("modifier", -dice.MaxModifier, dice.MaxModifier)
	if !d.clean(before) {
		return dice.Expression{}, false
	}
	die, _ := dice.NewDie(faces)
	e, err := dice.NewExpression(count, die, modifier)
	return e, err == nil
}
