package combat

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
)

// TraitRejection says why a part with the name of a recognised trait was not recognised.
// The zero value is not a rejection. Comparable and safe for concurrent use.
type TraitRejection int

const (
	// TraitSectionMismatch: the part is in the wrong section (Parry belongs to Reactions,
	// every other trait to Traits).
	TraitSectionMismatch TraitRejection = iota + 1
	// TraitTextMismatch: the verbatim text does not match the template of the trait.
	TraitTextMismatch
	// TraitUsageMismatch: the usage is not the one the trait needs (Legendary Resistance
	// needs an X/Day usage).
	TraitUsageMismatch
	// TraitParameterOutOfRange: a number parsed from the text or the part is outside the
	// range the rules rely on.
	TraitParameterOutOfRange
	// TraitUnknownDamageType: a Damage Type named in the text is not one of the thirteen.
	TraitUnknownDamageType
	// TraitNoSavingThrow: a Death Burst part holds no saving throw.
	TraitNoSavingThrow
)

// String returns the snake_case name of r, or "unknown".
func (r TraitRejection) String() string {
	return enumName([]string{"section_mismatch", "text_mismatch", "usage_mismatch", "parameter_out_of_range", "unknown_damage_type", "no_saving_throw"}, int(r))
}

// UnrecognisedTrait is a part with the name of a trait that failed a check: its position
// and the first check it failed. Comparable. The zero value is invalid. Safe for
// concurrent use.
type UnrecognisedTrait struct {
	part   LimitedUsePartRef
	reason TraitRejection
}

// Part returns the position of the part in the stat block.
func (u UnrecognisedTrait) Part() LimitedUsePartRef { return u.part }

// Reason returns the check the part failed.
func (u UnrecognisedTrait) Reason() TraitRejection { return u.reason }

// RecognisedTraits is what RecogniseTraits found in a stat block: the recognised traits and
// the parts that carry a trait name but failed a check, each in printed order (section
// order, then position). The zero value is empty and valid. It holds slices, so compare it
// with Equal; the accessors return copies. Safe for concurrent use.
type RecognisedTraits struct {
	traits       []MonsterTrait
	unrecognised []UnrecognisedTrait
}

// Traits returns, as a new slice and in printed order, the recognised traits.
func (r RecognisedTraits) Traits() []MonsterTrait { return slices.Clone(r.traits) }

// Unrecognised returns, as a new slice and in printed order, the parts whose name is a trait
// name and that failed a check.
func (r RecognisedTraits) Unrecognised() []UnrecognisedTrait { return slices.Clone(r.unrecognised) }

// Equal reports whether r and o hold the same traits and rejections in the same order.
func (r RecognisedTraits) Equal(o RecognisedTraits) bool {
	return slices.Equal(r.traits, o.traits) && slices.Equal(r.unrecognised, o.unrecognised)
}

// RecogniseTraits recognises the twelve traits of this package in monster (SRD 5.2.1,
// "Traits", p. 256). A part is recognised only when its name equals a trait name (case
// sensitive) and its verbatim text matches the template of that trait: whitespace runs count
// as one space, the typographic and the straight apostrophe are the same, and the noun slot
// matches one or more lower-case words, the same words at every occurrence. A part with a
// trait name that fails a check, in the order section, usage, text, parameters, appears in
// the rejections; a part with any other name appears in neither list. Regeneration in a
// monster that also has Swarm is marked with it (RegenerationTrait.SwarmCannotRegain).
//
// It never returns an error and never panics, for any monster including the zero one. The
// monster is not changed and equal monsters give equal results.
func RecogniseTraits(monster content.Monster) RecognisedTraits {
	legendary, _ := monster.LegendaryActions()
	sections := []struct {
		section MonsterPartSection
		parts   []content.StatBlockPart
	}{
		{MonsterPartTraits, monster.Traits()}, {MonsterPartActions, monster.Actions()}, {MonsterPartBonusActions, monster.BonusActions()},
		{MonsterPartReactions, monster.Reactions()}, {MonsterPartLegendaryActions, legendary.Actions()},
	}
	var out RecognisedTraits
	for _, sec := range sections {
		for i, p := range sec.parts {
			ref := LimitedUsePartRef{sec.section, i}
			trait, rejection, named := recogniseTrait(ref, p)
			switch {
			case !named:
			case trait != nil:
				out.traits = append(out.traits, trait)
			default:
				out.unrecognised = append(out.unrecognised, UnrecognisedTrait{ref, rejection})
			}
		}
	}
	if slices.ContainsFunc(out.traits, func(t MonsterTrait) bool { _, ok := t.(SwarmTrait); return ok }) {
		for j, t := range out.traits {
			if r, ok := t.(RegenerationTrait); ok {
				r.swarm = true
				out.traits[j] = r
			}
		}
	}
	return out
}

// The templates of table T-1 of monster-traits.md. ⟨n⟩ is a noun, ⟨k⟩ an integer and ⟨t⟩ a
// Damage Type word; the text is the text after the name's full stop.
const (
	slotNoun = "⟨n⟩"
	slotInt  = "⟨k⟩"
	slotType = "⟨t⟩"

	templateLegendaryResistance = "If the ⟨n⟩ fails a saving throw, it can choose to succeed instead."
	templateMagicResistance     = "The ⟨n⟩ has Advantage on saving throws against spells and other magical effects."
	templateUndeadFortitude     = "If damage reduces the ⟨n⟩ to 0 Hit Points, it makes a Constitution saving throw (DC 5 plus the damage taken) unless the damage is Radiant or from a Critical Hit. On a successful save, the ⟨n⟩ drops to 1 Hit Point instead."
	templateRegenerationFirst   = "The ⟨n⟩ regains ⟨k⟩ Hit Points at the start of each of its turns if it has at least 1 Hit Point."
	templateRegenerationSecond  = "The ⟨n⟩ regains ⟨k⟩ Hit Points at the start of each of its turns. If the ⟨n⟩ takes ⟨t⟩ or ⟨t⟩ damage, this trait doesn't function on the ⟨n⟩'s next turn. The ⟨n⟩ dies only if it starts its turn with 0 Hit Points and doesn't regenerate."
	templateDeathBurst          = "The ⟨n⟩ explodes when it dies."
	templateSwarm               = "The swarm can occupy another creature's space and vice versa, and the swarm can move through any opening large enough for a Tiny ⟨n⟩. The swarm can't regain Hit Points or gain Temporary Hit Points."
	templatePackTacticsSingular = "The ⟨n⟩ has Advantage on an attack roll against a creature if at least one of the ⟨n⟩'s allies is within 5 feet of the creature and the ally doesn't have the Incapacitated condition."
	templatePackTacticsPlural   = "The ⟨n⟩ has Advantage on attack rolls against a creature if at least one of the ⟨n⟩'s allies is within 5 feet of the creature and the ally doesn't have the Incapacitated condition."
	templateSunlightSensitivity = "While in sunlight, the ⟨n⟩ has Disadvantage on ability checks and attack rolls."
	templateSunlightWeakness    = "While in sunlight, the ⟨n⟩ has Disadvantage on D20 Tests."
	templateFlyby               = "The ⟨n⟩ doesn't provoke an Opportunity Attack when it flies out of an enemy's reach."
	templateAmorphous           = "The ⟨n⟩ can move through a space as narrow as 1 inch without expending extra movement to do so."
	templateSiegeMonster        = "The ⟨n⟩ deals double damage to objects and structures."
	templateParry               = "Trigger: The ⟨n⟩ is hit by a melee attack roll while holding a weapon. Response: The ⟨n⟩ adds ⟨k⟩ to its AC against that attack, possibly causing it to miss."
)

// recogniseTrait checks one part. named is false when the part's name is not a trait name;
// otherwise it returns the trait, or a nil trait and the first check the part failed.
func recogniseTrait(ref LimitedUsePartRef, p content.StatBlockPart) (trait MonsterTrait, rejection TraitRejection, named bool) {
	name := p.Name()
	switch name {
	case "Legendary Resistance", "Magic Resistance", "Undead Fortitude", "Regeneration", "Death Burst", "Swarm",
		"Pack Tactics", "Sunlight Sensitivity", "Sunlight Weakness", "Flyby", "Amorphous", "Siege Monster", "Parry":
	default:
		return nil, 0, false
	}
	section := MonsterPartTraits
	if name == "Parry" {
		section = MonsterPartReactions
	}
	if ref.section != section {
		return nil, TraitSectionMismatch, true
	}
	if _, perDay := p.Usage().(content.PartPerDay); name == "Legendary Resistance" && !perDay {
		return nil, TraitUsageMismatch, true
	}
	if len(p.Text()) > maxTraitTextBytes { // plan P10: no template is near this long, so the text is not scanned
		return nil, TraitTextMismatch, true
	}
	text := normaliseTraitText(p.Text())
	base := traitBase{ref}
	text1 := func(prefix bool, templates ...string) (templateMatch, bool) {
		return matchTemplates(text, prefix, templates...)
	}
	mismatch := func() (MonsterTrait, TraitRejection, bool) { return nil, TraitTextMismatch, true }
	switch name {
	case "Legendary Resistance":
		if _, ok := text1(false, templateLegendaryResistance); ok {
			return LegendaryResistanceTrait{base}, 0, true
		}
	case "Magic Resistance":
		if _, ok := text1(false, templateMagicResistance); ok {
			return MagicResistanceTrait{base}, 0, true
		}
	case "Undead Fortitude":
		if _, ok := text1(false, templateUndeadFortitude); ok {
			return UndeadFortitudeTrait{base}, 0, true
		}
	case "Swarm":
		if _, ok := text1(false, templateSwarm); ok {
			return SwarmTrait{base}, 0, true
		}
	case "Pack Tactics":
		if _, ok := text1(false, templatePackTacticsSingular, templatePackTacticsPlural); ok {
			return PackTacticsTrait{base}, 0, true
		}
	case "Flyby":
		if _, ok := text1(false, templateFlyby); ok {
			return FlybyTrait{base}, 0, true
		}
	case "Amorphous":
		if _, ok := text1(false, templateAmorphous); ok {
			return AmorphousTrait{base}, 0, true
		}
	case "Siege Monster":
		if _, ok := text1(false, templateSiegeMonster); ok {
			return SiegeMonsterTrait{base}, 0, true
		}
	case "Sunlight Sensitivity":
		if _, ok := text1(false, templateSunlightSensitivity); ok {
			return SunlightPenaltyTrait{base, SunlightPenaltyAbilityChecksAndAttackRolls}, 0, true
		}
	case "Sunlight Weakness":
		if _, ok := text1(false, templateSunlightWeakness); ok {
			return SunlightPenaltyTrait{base, SunlightPenaltyAllD20Tests}, 0, true
		}
	case "Regeneration":
		return recogniseRegeneration(ref, text)
	case "Death Burst":
		if _, ok := text1(true, templateDeathBurst); !ok {
			return mismatch()
		}
		save, has := p.SavingThrow()
		if !has {
			return nil, TraitNoSavingThrow, true
		}
		burst, err := NewDeathBurstTrait(ref, save)
		if err != nil {
			return nil, TraitParameterOutOfRange, true
		}
		return burst, 0, true
	case "Parry":
		m, ok := text1(false, templateParry)
		if !ok {
			return mismatch()
		}
		bonus, err := NewParryTrait(ref, parseTraitNumber(m.amount))
		if err != nil {
			return nil, TraitParameterOutOfRange, true
		}
		return bonus, 0, true
	}
	return mismatch()
}

// recogniseRegeneration checks the text of a Regeneration part against its two forms.
func recogniseRegeneration(ref LimitedUsePartRef, text string) (MonsterTrait, TraitRejection, bool) {
	m, ok := matchTemplates(text, false, templateRegenerationFirst)
	if !ok {
		m, ok = matchTemplates(text, false, templateRegenerationSecond)
	}
	if !ok {
		return nil, TraitTextMismatch, true
	}
	var types []content.DamageType
	for _, word := range m.types {
		t, err := content.ParseDamageType(word)
		if err != nil {
			return nil, TraitUnknownDamageType, true
		}
		types = append(types, t)
	}
	t, err := NewRegenerationTrait(ref, parseTraitNumber(m.amount), types)
	if err != nil { // the amount is outside 1 to the content maximum; the types were checked
		return nil, TraitParameterOutOfRange, true
	}
	return t, 0, true
}

// parseTraitNumber parses the digits of an integer slot; a number that does not fit an int is
// 0, which every caller treats as out of range.
func parseTraitNumber(digits string) int {
	n, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return n
}

// normaliseTraitText collapses whitespace runs to one space and turns the typographic
// apostrophe into the straight one.
func normaliseTraitText(s string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(s), " "), "’", "'")
}

// templateMatch is what a template matched: the noun, the integer digits and the Damage Type
// words, in order.
type templateMatch struct {
	noun, amount string
	types        []string
}

// matchTemplates returns the match of the first of templates that text fits. With prefix the
// text may go on after the template.
func matchTemplates(text string, prefix bool, templates ...string) (templateMatch, bool) {
	for _, tmpl := range templates {
		var m templateMatch
		if m.match(tmpl, text, prefix) {
			return m, true
		}
	}
	return templateMatch{}, false
}

// match reports whether text fits tmpl, filling m. A noun slot tries each end of a run of
// lower-case words and backtracks; the later slots of the same noun must repeat it.
func (m *templateMatch) match(tmpl, text string, prefix bool) bool {
	for tmpl != "" {
		switch {
		case strings.HasPrefix(tmpl, slotNoun):
			tmpl = tmpl[len(slotNoun):]
			if m.noun != "" {
				rest, ok := strings.CutPrefix(text, m.noun)
				if !ok {
					return false
				}
				text = rest
				continue
			}
			for _, end := range nounEnds(text) {
				saved := *m
				m.noun = text[:end]
				if m.match(tmpl, text[end:], prefix) {
					return true
				}
				*m = saved
			}
			return false
		case strings.HasPrefix(tmpl, slotInt):
			tmpl = tmpl[len(slotInt):]
			n := runLength(text, func(c byte) bool { return c >= '0' && c <= '9' })
			if n == 0 {
				return false
			}
			m.amount, text = text[:n], text[n:]
		case strings.HasPrefix(tmpl, slotType):
			tmpl = tmpl[len(slotType):]
			n := runLength(text, func(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' })
			if n == 0 {
				return false
			}
			m.types, text = append(m.types, text[:n]), text[n:]
		default:
			literal := tmpl
			if i := strings.Index(tmpl, "⟨"); i >= 0 {
				literal = tmpl[:i]
			}
			rest, ok := strings.CutPrefix(text, literal)
			if !ok {
				return false
			}
			tmpl, text = tmpl[len(literal):], rest
		}
	}
	return prefix || text == ""
}

// runLength is the length of the longest prefix of s whose bytes satisfy in.
func runLength(s string, in func(byte) bool) int {
	n := 0
	for n < len(s) && in(s[n]) {
		n++
	}
	return n
}

// nounEnds returns, shortest first, the ends of the runs of one or more lower-case words
// (a to z) that begin text, the words separated by single spaces.
func nounEnds(text string) []int {
	var ends []int
	i := 0
	for range len(text) { // a word is at least one byte, so this bounds the loop
		n := runLength(text[i:], func(c byte) bool { return c >= 'a' && c <= 'z' })
		if n == 0 {
			break
		}
		i += n
		ends = append(ends, i)
		if i >= len(text) || text[i] != ' ' {
			break
		}
		i++
	}
	return ends
}
