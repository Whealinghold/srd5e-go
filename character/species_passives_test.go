package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

var spPassives = character.SpeciesPassives{}

// Requirement: doc/specs/species-trait-behaviour.md FR-010, Table R (SRD 5.2.1, "Damage Resistance" p. 84, "Dwarven Resilience" p. 84, "Fiendish Legacies" p. 86) - Dragonborn resist the ancestry's type, Dwarves Poison, Tieflings the legacy's type, every other species nothing; the slice is a copy.
// Path: per case.
func TestSpeciesPassivesResistances(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		want   []content.DamageType
	}{
		{"Dragonborn Gold", spDragonborn(t, character.DraconicGold), []content.DamageType{content.DamageFire}},
		{"Dwarf", spDwarf(t), []content.DamageType{content.DamagePoison}},
		{"Tiefling Chthonic", spTiefling(t, character.LegacyChthonic), []content.DamageType{content.DamageNecrotic}},
		{"Tiefling Abyssal", spTiefling(t, character.LegacyAbyssal), []content.DamageType{content.DamagePoison}},
		{"Tiefling Infernal", spTiefling(t, character.LegacyInfernal), []content.DamageType{content.DamageFire}},
		{"Human", spHuman(t), nil},
		{"Orc", spOrc(t), nil},
		{"zero", character.SpeciesTraits{}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := spPassives.Resistances(c.traits)
			if !slices.Equal(got, c.want) || got == nil {
				t.Errorf("Resistances = %v, want %v", got, c.want)
			}
			if len(got) > 0 {
				got[0] = 0
				if spPassives.Resistances(c.traits)[0] == 0 {
					t.Error("the returned set is not a copy")
				}
			}
		})
	}
}

// resolved is the RollMode the kernel makes of the circumstances.
func spResolved(as []character.SpeciesAdvantage) modifier.RollMode {
	var cs []modifier.RollCircumstance
	for _, a := range as {
		cs = append(cs, a.Circumstance)
	}
	return modifier.Resolve(cs...)
}

func spSources(as []character.SpeciesAdvantage) []character.SpeciesTrait {
	out := []character.SpeciesTrait{}
	for _, a := range as {
		out = append(out, a.Source)
	}
	return out
}

// Requirement: doc/specs/species-trait-behaviour.md FR-011, FR-012, Table A "save" rows (SRD 5.2.1, "Dwarf", "Elf", "Gnome", "Halfling", pp. 84-86; "They Don't Stack", p. 8) - a save to avoid or end a condition gets Advantage from exactly the trait that names it, Gnomish Cunning covers only Intelligence, Wisdom and Charisma saves, a trait never applies to another condition or to a species that lacks it, and several matches make one Advantage.
// Path: per case.
func TestSpeciesPassivesSaves(t *testing.T) {
	t.Parallel()
	save := func(a ability.Ability, c content.ConditionKind) character.SavingThrowSituation {
		return character.SavingThrowSituation{Ability: a, Condition: c}
	}
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		s      character.SavingThrowSituation
		want   []character.SpeciesTrait
	}{
		{"Dwarf avoiding Poisoned", spDwarf(t), save(ability.Constitution, content.ConditionPoisoned), []character.SpeciesTrait{character.TraitDwarvenResilience}},
		{"Elf ending Charmed", spElf(t, character.ElvenDrow), save(ability.Wisdom, content.ConditionCharmed), []character.SpeciesTrait{character.TraitFeyAncestry}},
		{"Halfling avoiding Frightened", spHalfling(t), save(ability.Wisdom, content.ConditionFrightened), []character.SpeciesTrait{character.TraitBrave}},
		{"Gnome Intelligence", spGnome(t, character.GnomishRock), save(ability.Intelligence, 0), []character.SpeciesTrait{character.TraitGnomishCunning}},
		{"Gnome Wisdom", spGnome(t, character.GnomishRock), save(ability.Wisdom, 0), []character.SpeciesTrait{character.TraitGnomishCunning}},
		{"Gnome Charisma", spGnome(t, character.GnomishForest), save(ability.Charisma, 0), []character.SpeciesTrait{character.TraitGnomishCunning}},
		{"Gnome Strength", spGnome(t, character.GnomishRock), save(ability.Strength, 0), []character.SpeciesTrait{}},
		{"Gnome Dexterity", spGnome(t, character.GnomishRock), save(ability.Dexterity, 0), []character.SpeciesTrait{}},
		{"Gnome Constitution", spGnome(t, character.GnomishRock), save(ability.Constitution, 0), []character.SpeciesTrait{}},
		{"Gnome Wisdom avoiding Charmed: Cunning only", spGnome(t, character.GnomishRock), save(ability.Wisdom, content.ConditionCharmed), []character.SpeciesTrait{character.TraitGnomishCunning}},
		{"Elf Wisdom avoiding Charmed: Fey Ancestry only", spElf(t, character.ElvenWoodElf), save(ability.Wisdom, content.ConditionCharmed), []character.SpeciesTrait{character.TraitFeyAncestry}},
		{"Dwarf avoiding Frightened", spDwarf(t), save(ability.Wisdom, content.ConditionFrightened), []character.SpeciesTrait{}},
		{"Halfling avoiding Poisoned", spHalfling(t), save(ability.Constitution, content.ConditionPoisoned), []character.SpeciesTrait{}},
		{"Human any", spHuman(t), save(ability.Wisdom, content.ConditionCharmed), []character.SpeciesTrait{}},
		{"zero traits", character.SpeciesTraits{}, save(ability.Wisdom, content.ConditionCharmed), []character.SpeciesTrait{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := spPassives.SaveCircumstances(c.traits, c.s)
			if !slices.Equal(spSources(got), c.want) {
				t.Fatalf("sources = %v, want %v", spSources(got), c.want)
			}
			if wantMode := map[bool]modifier.RollMode{true: modifier.WithAdvantage, false: modifier.Normal}[len(c.want) > 0]; spResolved(got) != wantMode {
				t.Errorf("mode = %v, want %v", spResolved(got), wantMode)
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-013, FR-039, Table A "check" rows, "Powerful Build" (SRD 5.2.1, "Powerful Build", "Large Form", p. 86; "Grappled", p. 182; "They Don't Stack", p. 8) - a Goliath gets Advantage on a check that ends Grappled and, with Large Form active, on a Strength check; both together are two sources and one Advantage; any other species and any other check gets none.
// Path: per case.
func TestSpeciesPassivesChecks(t *testing.T) {
	t.Parallel()
	goliath := spGoliath(t, character.AncestryStone)
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		s      character.AbilityCheckSituation
		want   []character.SpeciesTrait
	}{
		{"Strength ending Grappled with Large Form", goliath, character.AbilityCheckSituation{Ability: ability.Strength, EndsGrappled: true, LargeFormActive: true}, []character.SpeciesTrait{character.TraitPowerfulBuild, character.TraitLargeForm}},
		{"Strength, Large Form only", goliath, character.AbilityCheckSituation{Ability: ability.Strength, LargeFormActive: true}, []character.SpeciesTrait{character.TraitLargeForm}},
		{"Dexterity ending Grappled, no Large Form", goliath, character.AbilityCheckSituation{Ability: ability.Dexterity, EndsGrappled: true}, []character.SpeciesTrait{character.TraitPowerfulBuild}},
		{"Dexterity with Large Form", goliath, character.AbilityCheckSituation{Ability: ability.Dexterity, LargeFormActive: true}, []character.SpeciesTrait{}},
		{"Strength without Large Form", goliath, character.AbilityCheckSituation{Ability: ability.Strength}, []character.SpeciesTrait{}},
		{"Human ending Grappled", spHuman(t), character.AbilityCheckSituation{Ability: ability.Strength, EndsGrappled: true, LargeFormActive: true}, []character.SpeciesTrait{}},
		{"zero traits", character.SpeciesTraits{}, character.AbilityCheckSituation{EndsGrappled: true}, []character.SpeciesTrait{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := spPassives.CheckCircumstances(c.traits, c.s)
			if !slices.Equal(spSources(got), c.want) {
				t.Fatalf("sources = %v, want %v", spSources(got), c.want)
			}
			if len(c.want) > 0 && spResolved(got) != modifier.WithAdvantage {
				t.Errorf("mode = %v", spResolved(got))
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-013, Table A "capacity size" (SRD 5.2.1, "Powerful Build" p. 86; "Carrying Capacity" p. 178) - a Goliath counts as one size larger, Gargantuan stays Gargantuan, other species keep their size.
// Path: per case.
func TestSpeciesPassivesCarryingCapacitySize(t *testing.T) {
	t.Parallel()
	goliath := spGoliath(t, character.AncestryHill)
	for _, c := range []struct {
		name         string
		traits       character.SpeciesTraits
		size, wanted content.CreatureSize
	}{
		{"Goliath Medium", goliath, content.SizeMedium, content.SizeLarge},
		{"Goliath Large", goliath, content.SizeLarge, content.SizeHuge},
		{"Goliath Gargantuan", goliath, content.SizeGargantuan, content.SizeGargantuan},
		{"Dwarf Medium", spDwarf(t), content.SizeMedium, content.SizeMedium},
	} {
		if got := spPassives.CarryingCapacitySize(c.traits, c.size); got != c.wanted {
			t.Errorf("%s = %v, want %v", c.name, got, c.wanted)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-014, Table A "Dwarven Toughness" (SRD 5.2.1, p. 84) - only a Dwarf has the Hit Point maximum bonus: 1 at level 1 and 1 more per level gained, counted in the total level.
// Path: happy.
func TestSpeciesPassivesHitPointMaximumBonus(t *testing.T) {
	t.Parallel()
	b, ok := spPassives.HitPointMaximumBonus(spDwarf(t))
	levels := func(n int) character.ClassLevels {
		return character.MustClassLevels([]character.ClassLevel{{Class: "fighter", Level: n}})
	}
	if !ok || b != character.MustHitPointMaximumBonus(1, 1, 1, "") || b.ContributionFor(levels(1)) != 1 || b.ContributionFor(levels(5)) != 5 {
		t.Errorf("Dwarf bonus = %v, %v", b, ok)
	}
	for name, tr := range map[string]character.SpeciesTraits{"Human": spHuman(t), "Orc": spOrc(t), "zero": {}} {
		if got, ok := spPassives.HitPointMaximumBonus(tr); ok || !got.IsZero() {
			t.Errorf("%s: %v, %v", name, got, ok)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-015, Table S (SRD 5.2.1, "Darkvision" and the lineage tables, pp. 84-86; "Large Form" p. 86) - Darkvision is 60 feet for Dragonborn, Gnomes, Tieflings and the High Elf and Wood Elf lineages, 120 for Dwarves, Orcs and Drow, none for Goliaths, Halflings and Humans; the Wood Elf Speed is 35 and an active Large Form adds 10, the base Speed of the content staying the input.
// Path: per case.
func TestSpeciesPassivesSenses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name       string
		traits     character.SpeciesTraits
		darkvision int
		speed      int
		large      bool
		wantSpeed  int
	}{
		{"Dragonborn", spDragonborn(t, character.DraconicRed), 60, 30, false, 30},
		{"High Elf", spElf(t, character.ElvenHighElf), 60, 30, false, 30},
		{"Wood Elf", spElf(t, character.ElvenWoodElf), 60, 30, false, 35},
		{"Wood Elf with a faster base", spElf(t, character.ElvenWoodElf), 60, 40, false, 40},
		{"Drow", spElf(t, character.ElvenDrow), 120, 30, false, 30},
		{"Gnome", spGnome(t, character.GnomishRock), 60, 30, false, 30},
		{"Tiefling", spTiefling(t, character.LegacyAbyssal), 60, 30, false, 30},
		{"Dwarf", spDwarf(t), 120, 30, false, 30},
		{"Orc", spOrc(t), 120, 30, false, 30},
		{"Goliath", spGoliath(t, character.AncestryHill), 0, 35, false, 35},
		{"Goliath with Large Form", spGoliath(t, character.AncestryHill), 0, 35, true, 45},
		{"Halfling with Large Form flag", spHalfling(t), 0, 30, true, 30},
		{"Human", spHuman(t), 0, 30, false, 30},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := spPassives.DarkvisionFeet(c.traits); got != c.darkvision {
				t.Errorf("Darkvision = %d, want %d", got, c.darkvision)
			}
			if got := spPassives.SpeedFeet(c.traits, c.speed, c.large); got != c.wantSpeed {
				t.Errorf("Speed = %d, want %d", got, c.wantSpeed)
			}
		})
	}
	if spPassives.DarkvisionFeet(character.SpeciesTraits{}) != 0 {
		t.Error("the zero traits see in the dark")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-016, Table A "Trance" (SRD 5.2.1, "Trance", p. 85) - only an Elf has Trance.
// Path: happy.
func TestSpeciesPassivesTrance(t *testing.T) {
	t.Parallel()
	if !spPassives.Trance(spElf(t, character.ElvenDrow)) || spPassives.Trance(spDwarf(t)) || spPassives.Trance(character.SpeciesTraits{}) {
		t.Error("Trance is not the Elf's alone")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-017, Table A "size checks" (SRD 5.2.1, "Halfling Nimbleness", "Naturally Stealthy", p. 86; "Hide", p. 183) - a Small Halfling may move through a Medium creature's space and not a Large, Small or Tiny one; it may Hide behind a Medium or Large creature and not a Small one; no other species may.
// Path: per case.
func TestSpeciesPassivesHalflingSizes(t *testing.T) {
	t.Parallel()
	h := spHalfling(t)
	for _, c := range []struct {
		name       string
		other      content.CreatureSize
		move, hide bool
	}{
		{"Tiny", content.SizeTiny, false, false}, {"Small", content.SizeSmall, false, false},
		{"Medium", content.SizeMedium, true, true}, {"Large", content.SizeLarge, false, true}, {"Huge", content.SizeHuge, false, true},
	} {
		if got := spPassives.MayMoveThrough(h, content.SizeSmall, c.other); got != c.move {
			t.Errorf("MayMoveThrough(Small, %s) = %v, want %v", c.name, got, c.move)
		}
		if got := spPassives.ObscuresForHide(h, content.SizeSmall, c.other); got != c.hide {
			t.Errorf("ObscuresForHide(Small, %s) = %v, want %v", c.name, got, c.hide)
		}
	}
	if spPassives.MayMoveThrough(spHuman(t), content.SizeSmall, content.SizeMedium) || spPassives.ObscuresForHide(spHuman(t), content.SizeSmall, content.SizeMedium) {
		t.Error("a Human gets the Halfling traits")
	}
	if spPassives.MayMoveThrough(h, 0, content.SizeSmall) || spPassives.ObscuresForHide(h, content.SizeSmall, 0) || spPassives.ObscuresForHide(h, 0, content.SizeLarge) {
		t.Error("an invalid size passes")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-018, Table A "skills" (SRD 5.2.1, "Keen Senses" p. 85, "Skillful" p. 86) - an Elf gives its Keen Senses skill and a Human its Skillful skill, other species none; the slice is a copy.
// Path: per case.
func TestSpeciesPassivesSkills(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		traits character.SpeciesTraits
		want   []content.Skill
	}{"Elf": {spElf(t, character.ElvenDrow), []content.Skill{content.SkillInsight}}, "Human": {spHuman(t), []content.Skill{content.SkillStealth}}, "Dwarf": {spDwarf(t), nil}} {
		got := spPassives.SkillProficiencies(c.traits)
		if !slices.Equal(got, c.want) || got == nil {
			t.Errorf("%s = %v, want %v", name, got, c.want)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-011, FR-013 - each SpeciesTrait has a snake_case name for explanation; an undefined value is "unknown".
// Path: happy.
func TestSpeciesTraitNames(t *testing.T) {
	t.Parallel()
	for want, tr := range map[string]character.SpeciesTrait{"dwarven_resilience": character.TraitDwarvenResilience, "fey_ancestry": character.TraitFeyAncestry, "brave": character.TraitBrave,
		"gnomish_cunning": character.TraitGnomishCunning, "powerful_build": character.TraitPowerfulBuild, "large_form": character.TraitLargeForm, "unknown": 0} {
		if tr.String() != want {
			t.Errorf("%d = %q, want %q", tr, tr.String(), want)
		}
	}
}
