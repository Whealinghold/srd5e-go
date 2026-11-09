package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// SpeciesTrait names a species trait that gives Advantage. The zero value is
// invalid. Comparable and safe for concurrent use.
type SpeciesTrait int

// The traits that give Advantage.
const (
	TraitDwarvenResilience SpeciesTrait = iota + 1
	TraitFeyAncestry
	TraitBrave
	TraitGnomishCunning
	TraitPowerfulBuild
	TraitLargeForm
)

// String returns the snake_case name, or "unknown".
func (s SpeciesTrait) String() string {
	return featNameOf(int(s), "dwarven_resilience", "fey_ancestry", "brave", "gnomish_cunning", "powerful_build", "large_form")
}

// SpeciesAdvantage pairs a kernel circumstance with the trait that gives it, for
// explanation. Pass the Circumstance of every match to modifier.Resolve: several
// matches are one Advantage ("They Don't Stack", SRD 5.2.1, p. 8). Comparable.
type SpeciesAdvantage struct {
	Source       SpeciesTrait
	Circumstance modifier.RollCircumstance
}

func advantageFrom(s SpeciesTrait) SpeciesAdvantage {
	return SpeciesAdvantage{Source: s, Circumstance: modifier.Advantage{}}
}

// SavingThrowSituation describes a saving throw: its ability and, when the save is
// made to avoid or end a condition, that condition (the zero ConditionKind names
// none). Construct it with a keyed literal. Comparable.
type SavingThrowSituation struct {
	Ability   ability.Ability
	Condition content.ConditionKind
}

// AbilityCheckSituation describes an ability check: its ability, whether it is made
// to end the Grappled condition, and whether Large Form is active. Construct it with
// a keyed literal. Comparable.
type AbilityCheckSituation struct {
	Ability         ability.Ability
	EndsGrappled    bool
	LargeFormActive bool
}

// SpeciesPassives declares the passive traits of a species (SRD 5.2.1, pp. 84-86):
// Resistances, Advantage sources, the Hit Point maximum bonus, senses, Speed, size
// rules and skills. The zero value is ready to use. It is stateless and safe for
// concurrent use. Every method gives nothing for a species without the trait and for
// the zero SpeciesTraits.
type SpeciesPassives struct{}

// Resistances returns a new slice of the damage types the species resists: a
// Dragonborn the ancestry's type, a Dwarf Poison, a Tiefling the legacy's type.
func (SpeciesPassives) Resistances(t SpeciesTraits) []content.DamageType {
	switch v := t.choices.(type) {
	case DragonbornChoices:
		return []content.DamageType{v.Ancestry.DamageType()}
	case TieflingChoices:
		return []content.DamageType{v.Legacy.Resistance()}
	}
	if t.species == speciesDwarf {
		return []content.DamageType{content.DamagePoison}
	}
	return []content.DamageType{}
}

// SaveCircumstances returns an Advantage per matching trait for a saving throw:
// Dwarven Resilience when it avoids or ends Poisoned, Fey Ancestry Charmed, Brave
// Frightened, and Gnomish Cunning any Intelligence, Wisdom or Charisma save. It
// returns a new slice, empty for no match.
func (SpeciesPassives) SaveCircumstances(t SpeciesTraits, s SavingThrowSituation) []SpeciesAdvantage {
	out := []SpeciesAdvantage{}
	add := func(species content.SpeciesID, cond content.ConditionKind, trait SpeciesTrait) {
		if t.species == species && s.Condition == cond {
			out = append(out, advantageFrom(trait))
		}
	}
	add(speciesDwarf, content.ConditionPoisoned, TraitDwarvenResilience)
	add(speciesElf, content.ConditionCharmed, TraitFeyAncestry)
	add(speciesHalfling, content.ConditionFrightened, TraitBrave)
	if t.species == speciesGnome && spellcastingAbilityAllowed(s.Ability) {
		out = append(out, advantageFrom(TraitGnomishCunning))
	}
	return out
}

// CheckCircumstances returns an Advantage per matching trait for an ability check:
// Powerful Build for a check that ends Grappled, and Large Form for a Strength
// check while it is active. It returns a new slice, empty for no match.
func (SpeciesPassives) CheckCircumstances(t SpeciesTraits, s AbilityCheckSituation) []SpeciesAdvantage {
	out := []SpeciesAdvantage{}
	if t.species != speciesGoliath {
		return out
	}
	if s.EndsGrappled {
		out = append(out, advantageFrom(TraitPowerfulBuild))
	}
	if s.LargeFormActive && s.Ability == ability.Strength {
		out = append(out, advantageFrom(TraitLargeForm))
	}
	return out
}

// CarryingCapacitySize returns the size to use for carrying capacity: the next larger
// CreatureSize for a Goliath (Powerful Build), Gargantuan staying Gargantuan, and
// size itself for every other species.
func (SpeciesPassives) CarryingCapacitySize(t SpeciesTraits, size content.CreatureSize) content.CreatureSize {
	if t.species == speciesGoliath && size < content.SizeGargantuan {
		return size + 1
	}
	return size
}

// HitPointMaximumBonus returns the Dwarven Toughness bonus (1, plus 1 per level
// gained, from level 1, counted in the total level) and true, or false for any
// other species.
func (SpeciesPassives) HitPointMaximumBonus(t SpeciesTraits) (HitPointMaximumBonus, bool) {
	if t.species != speciesDwarf {
		return HitPointMaximumBonus{}, false
	}
	return MustHitPointMaximumBonus(1, 1, 1, ""), true
}

// DarkvisionFeet returns the range of Darkvision in feet: 60 for a Dragonborn, Gnome,
// Tiefling and the High Elf and Wood Elf lineages, 120 for a Dwarf, an Orc and the
// Drow lineage, and 0 for a Goliath, a Halfling and a Human.
func (SpeciesPassives) DarkvisionFeet(t SpeciesTraits) int {
	switch v := t.choices.(type) {
	case ElfChoices:
		if v.Lineage == ElvenDrow {
			return 120
		}
		return 60
	case DragonbornChoices, GnomeChoices, TieflingChoices:
		return 60
	}
	if t.species == speciesDwarf || t.species == speciesOrc {
		return 120
	}
	return 0
}

// SpeedFeet returns the Speed given the base Speed of the species content: the Wood
// Elf lineage raises it to 35 feet, and an active Large Form adds 10 feet.
func (SpeciesPassives) SpeedFeet(t SpeciesTraits, baseSpeed int, largeFormActive bool) int {
	speed := baseSpeed
	if e, ok := t.choices.(ElfChoices); ok && e.Lineage == ElvenWoodElf {
		speed = max(speed, 35)
	}
	if largeFormActive && t.species == speciesGoliath {
		speed += 10
	}
	return speed
}

// Trance reports whether the species has Trance: magic cannot put the character to
// sleep, and the Long Rest may be taken as TranceLongRest.
func (SpeciesPassives) Trance(t SpeciesTraits) bool { return t.species == speciesElf }

// MayMoveThrough reports whether a Halfling of size own may move through the space
// of a creature of size other: exactly when other is one size larger (Halfling
// Nimbleness). Stopping in the same space stays a movement rule.
func (SpeciesPassives) MayMoveThrough(t SpeciesTraits, own, other content.CreatureSize) bool {
	return t.species == speciesHalfling && own.Valid() && other == own+1
}

// ObscuresForHide reports whether a Halfling of size own may Hide obscured only by a
// creature of size obscurer: when obscurer is at least one size larger (Naturally
// Stealthy). It widens only that test; the other Hide requirements stay.
func (SpeciesPassives) ObscuresForHide(t SpeciesTraits, own, obscurer content.CreatureSize) bool {
	return t.species == speciesHalfling && own.Valid() && obscurer.Valid() && obscurer >= own+1
}

// SkillProficiencies returns a new slice of the skill the species gives: the Keen
// Senses skill of an Elf and the Skillful skill of a Human.
func (SpeciesPassives) SkillProficiencies(t SpeciesTraits) []content.Skill {
	switch v := t.choices.(type) {
	case ElfChoices:
		return []content.Skill{v.KeenSensesSkill}
	case HumanChoices:
		return []content.Skill{v.Skill}
	}
	return []content.Skill{}
}
