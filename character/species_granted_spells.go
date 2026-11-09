package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// speciesThaumaturgy is the Thaumaturgy cantrip of Otherworldly Presence. Content
// keeps that trait as text, so the id is a constant here (SRD 5.2.1, p. 86).
const speciesThaumaturgy content.SpellID = "thaumaturgy"

// SpeciesGrantedSpell is a spell a species grants (SRD 5.2.1, pp. 84-86): the spell,
// the spellcasting ability, whether it is always prepared (and so not counted
// against a prepared-spell limit) and the limited use of its free cast, if any. It
// is comparable. Only SpeciesGrantedSpells builds one.
type SpeciesGrantedSpell struct {
	spell          content.SpellID
	ability        ability.Ability
	alwaysPrepared bool
	use            SpeciesUse
}

// Spell returns the id of the spell.
func (s SpeciesGrantedSpell) Spell() content.SpellID { return s.spell }

// Ability returns the spellcasting ability of the trait, chosen with the lineage.
func (s SpeciesGrantedSpell) Ability() ability.Ability { return s.ability }

// AlwaysPrepared reports whether the spell is always prepared.
func (s SpeciesGrantedSpell) AlwaysPrepared() bool { return s.alwaysPrepared }

// Use returns the SpeciesUse of the free cast without a spell slot and true, or
// false for a cantrip. The spell can also be cast with a spell slot of its level.
func (s SpeciesGrantedSpell) Use() (SpeciesUse, bool) { return s.use, s.use != 0 }

// ClockworkDevice holds the facts of the Tiny clockwork device a Rock Gnome creates
// by casting Prestidigitation for 10 minutes (SRD 5.2.1, "Gnomish Lineage", p. 85).
// Comparable.
type ClockworkDevice struct {
	MinutesToCreate int
	Size            content.CreatureSize
	ArmorClass      int
	HitPoints       int
	MaxAtOnce       int
	HoursLasting    int
}

// SpeciesGrantedSpells lists the spells a species grants by character level (SRD
// 5.2.1, "Elf" p. 85, "Gnome" p. 85, "Tiefling" p. 86). The ids are code constants
// pinned against the bundled content by a test; no content lookup is made. The zero
// value is ready to use. It is stateless and safe for concurrent use.
type SpeciesGrantedSpells struct{}

// Of returns a new slice of the spells held at a total level, cantrips first: the level
// 1 benefit from level 1, the level 3 spell from level 3 and the level 5 spell from
// level 5, none below its level. A High Elf's cantrip is the one it holds now. The
// slice is empty for a species that grants no spell and for the zero values.
func (SpeciesGrantedSpells) Of(t SpeciesTraits, level Level) []SpeciesGrantedSpell {
	out := []SpeciesGrantedSpell{}
	n := level.Value()
	cantrip := func(id content.SpellID, a ability.Ability) {
		out = append(out, SpeciesGrantedSpell{spell: id, ability: a})
	}
	lineage := func(a ability.Ability, three, five content.SpellID) {
		if n >= 3 {
			out = append(out, SpeciesGrantedSpell{spell: three, ability: a, alwaysPrepared: true, use: LineageSpellThreeUse})
		}
		if n >= 5 {
			out = append(out, SpeciesGrantedSpell{spell: five, ability: a, alwaysPrepared: true, use: LineageSpellFiveUse})
		}
	}
	if n < 1 {
		return out
	}
	switch v := t.choices.(type) {
	case ElfChoices:
		switch v.Lineage {
		case ElvenDrow:
			cantrip("dancing_lights", v.Ability)
			lineage(v.Ability, "faerie_fire", "darkness")
		case ElvenHighElf:
			current, _ := t.HighElfCantrip()
			cantrip(current, v.Ability)
			lineage(v.Ability, "detect_magic", "misty_step")
		case ElvenWoodElf:
			cantrip("druidcraft", v.Ability)
			lineage(v.Ability, "longstrider", "pass_without_trace")
		}
	case TieflingChoices:
		first := [...]content.SpellID{"poison_spray", "chill_touch", "fire_bolt"}
		three := [...]content.SpellID{"ray_of_sickness", "false_life", "hellish_rebuke"}
		five := [...]content.SpellID{"hold_person", "ray_of_enfeeblement", "darkness"}
		if v.Legacy.Valid() {
			i := v.Legacy - 1
			cantrip(first[i], v.Ability)
			cantrip(speciesThaumaturgy, v.Ability)
			lineage(v.Ability, three[i], five[i])
		}
	case GnomeChoices:
		switch v.Lineage {
		case GnomishForest:
			cantrip("minor_illusion", v.Ability)
			out = append(out, SpeciesGrantedSpell{spell: "speak_with_animals", ability: v.Ability, alwaysPrepared: true, use: GnomeSpellUse})
		case GnomishRock:
			cantrip("mending", v.Ability)
			cantrip("prestidigitation", v.Ability)
		}
	}
	return out
}

// ClockworkDevice returns the facts of the device and true for a Rock Gnome, or
// false for any other character.
func (SpeciesGrantedSpells) ClockworkDevice(t SpeciesTraits) (ClockworkDevice, bool) {
	g, ok := t.choices.(GnomeChoices)
	if !ok || g.Lineage != GnomishRock {
		return ClockworkDevice{}, false
	}
	return ClockworkDevice{MinutesToCreate: 10, Size: content.SizeTiny, ArmorClass: 5, HitPoints: 1, MaxAtOnce: 3, HoursLasting: 8}, true
}
