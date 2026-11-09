package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Synthetic content for the creation tests: classes, backgrounds, species and tools with
// the shapes SRD 5.2.1 gives them (pp. 31, 47, 49, 83-86), under test-only ids and numbers.

func classItems(pairs ...any) []content.ClassEquipmentItem {
	var out []content.ClassEquipmentItem
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, must(content.NewClassEquipmentItem(pairs[i].(string), pairs[i+1].(int))))
	}
	return out
}

func classOption(gold int, pairs ...any) content.ClassEquipmentOption {
	return must(content.NewClassEquipmentOption(classItems(pairs...), gold))
}

// crClass rebuilds base with the given skill choice, tool proficiencies and starting equipment.
func crClass(t *testing.T, base content.ClassDefinition, id string, skills content.SkillChoice, tools []content.ClassToolProficiency, eq content.ClassStartingEquipment) content.ClassDefinition {
	t.Helper()
	return must(content.NewClassDefinition(content.ClassID(id), id, noSource, content.ClassFields{
		PrimaryAbility: base.PrimaryAbility(), HitPointDie: base.HitPointDie(), SavingThrowProficiencies: base.SavingThrowProficiencies(),
		SkillChoice: skills, WeaponProficiencies: base.WeaponProficiencies(), ToolProficiencies: tools, ArmorTraining: base.ArmorTraining(),
		StartingEquipment: eq, MulticlassGrants: base.MulticlassGrants(), Spellcasting: base.Spellcasting(), Columns: base.Columns(),
		Progression: base.Progression(), Features: base.Features(), Subclasses: base.Subclasses(),
	}))
}

// crFighter is a Fighter: d10, two skills from a list of eight, options A, B and C.
func crFighter(t *testing.T) content.ClassDefinition {
	t.Helper()
	skills := must(content.NewSkillChoiceListed(2, []content.Skill{
		content.SkillAcrobatics, content.SkillAnimalHandling, content.SkillAthletics, content.SkillHistory,
		content.SkillInsight, content.SkillIntimidation, content.SkillPerception, content.SkillSurvival}))
	eq := must(content.NewClassStartingEquipment(classOption(4, "Greatsword", 1, "Flail", 1), classOption(11, "Longsword", 1),
		classOption(155)))
	return crClass(t, hlsFighter(t), "fighter", skills, nil, eq)
}

// crBard is a Bard: d8, any three skills, three Musical Instruments, options A and B only.
func crBard(t *testing.T) content.ClassDefinition {
	t.Helper()
	base := advClass(t, advSpec{id: "bard", die: "D8", kind: "allOf", abilities: []string{"charisma"}})
	tools := []content.ClassToolProficiency{must(content.NewClassToolProficiencyChoice([]string{"Musical Instrument"}, 3))}
	eq := must(content.NewClassStartingEquipment(classOption(19, "Leather Armor", 1, "Musical Instrument of your choice", 1), classOption(90)))
	return crClass(t, base, "bard", must(content.NewSkillChoiceAny(3)), tools, eq)
}

// crMonk is a Monk: d8, two listed skills, one Artisan's Tools or Musical Instrument.
func crMonk(t *testing.T) content.ClassDefinition {
	t.Helper()
	base := advClass(t, advSpec{id: "monk", die: "D8", kind: "allOf", abilities: []string{"dexterity", "wisdom"}})
	skills := must(content.NewSkillChoiceListed(2, []content.Skill{content.SkillAcrobatics, content.SkillAthletics, content.SkillStealth}))
	tools := []content.ClassToolProficiency{must(content.NewClassToolProficiencyChoice([]string{"Artisan’s Tools", "Musical Instrument"}, 1))}
	eq := must(content.NewClassStartingEquipment(
		classOption(11, "Spear", 1, "Artisan's Tools or Musical Instrument chosen for the tool proficiency above", 1), classOption(50)))
	return crClass(t, base, "monk", skills, tools, eq)
}

// crWizard is a Wizard: d6, listed skills, options A and B.
func crWizard(t *testing.T) content.ClassDefinition {
	t.Helper()
	base := advClass(t, advSpec{id: "wizard", die: "D6", kind: "allOf", abilities: []string{"intelligence"}})
	skills := must(content.NewSkillChoiceListed(2, []content.Skill{content.SkillArcana, content.SkillHistory, content.SkillInsight}))
	return crClass(t, base, "wizard", skills, nil, must(content.NewClassStartingEquipment(classOption(5, "Quarterstaff", 1), classOption(55))))
}

func bgItems(pairs ...any) []content.EquipmentItem {
	var out []content.EquipmentItem
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, must(content.NewEquipmentItem(pairs[i].(string), pairs[i+1].(int))))
	}
	return out
}

func bgOption(gold int, pairs ...any) content.EquipmentOption {
	return must(content.NewEquipmentOption(bgItems(pairs...), gold))
}

func crBackground(id, name, feat, qualifier string, abilities []ability.Ability, skills []content.Skill, tool content.ToolProficiency, a, b content.EquipmentOption) content.Background {
	return content.MustBackground(content.BackgroundID(id), name, noSource, content.BackgroundFields{
		AbilityScores: abilities, Feat: content.MustBackgroundFeatReference(content.FeatID(feat), qualifier), SkillProficiencies: skills,
		ToolProficiency: tool, EquipmentOptions: content.NewEquipmentOptions(a, b),
	})
}

// crSoldier lists Strength, Dexterity, Constitution; its tool is a Gaming Set of the player's choice.
func crSoldier() content.Background {
	return crBackground("soldier", "Soldier", "savage_attacker", "", []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution},
		[]content.Skill{content.SkillAthletics, content.SkillIntimidation}, must(content.NewChoiceToolProficiency("Gaming Set")),
		bgOption(14, "Spear", 1, "Gaming Set (same as above)", 1), bgOption(50))
}

// crAcolyte lists Intelligence, Wisdom, Charisma; its feat asks for the Cleric list.
func crAcolyte() content.Background {
	return crBackground("acolyte", "Acolyte", "magic_initiate", "Cleric", []ability.Ability{ability.Intelligence, ability.Wisdom, ability.Charisma},
		[]content.Skill{content.SkillInsight, content.SkillReligion}, must(content.NewSpecificToolProficiency("Calligrapher’s Supplies")),
		bgOption(8, "Holy Symbol", 1), bgOption(50))
}

// crSage lists Constitution, Intelligence, Wisdom.
func crSage() content.Background {
	return crBackground("sage", "Sage", "alert", "", []ability.Ability{ability.Constitution, ability.Intelligence, ability.Wisdom},
		[]content.Skill{content.SkillArcana, content.SkillHistory}, must(content.NewSpecificToolProficiency("Calligrapher’s Supplies")),
		bgOption(8, "Book", 1), bgOption(50))
}

func textChoice(trait string, options ...string) content.SpeciesChoice {
	var os []content.ChoiceOption
	for _, o := range options {
		os = append(os, must(content.NewTextOption(o, "Text.", nil)))
	}
	return content.MustSpeciesChoice(trait, os)
}

func crSpecies(id string, sizes []content.CreatureSize, choices ...content.SpeciesChoice) content.Species {
	traits := []content.NamedText{must(content.NewNamedText("Trait", "Text."))}
	for _, c := range choices {
		traits = append(traits, must(content.NewNamedText(c.TraitName(), "Text.")))
	}
	return content.MustSpecies(content.SpeciesID(id), id, noSource, content.SpeciesFields{
		CreatureType: content.CreatureTypeHumanoid, Sizes: sizes, SizeText: "Size.", Speed: 30, Traits: traits, Choices: choices,
	})
}

func crSpeciesAll() []content.Entry {
	return []content.Entry{
		crSpecies("human", []content.CreatureSize{content.SizeMedium, content.SizeSmall}),
		crSpecies("dwarf", []content.CreatureSize{content.SizeMedium}),
		crSpecies("gnome", []content.CreatureSize{content.SizeSmall}, textChoice("Gnomish Lineage", "Forest Gnome", "Rock Gnome")),
		crSpecies("dragonborn", []content.CreatureSize{content.SizeMedium}, textChoice("Draconic Ancestry", "Gold", "Red")),
		crSpecies("elf", []content.CreatureSize{content.SizeMedium}, textChoice("Elven Lineage", "Drow", "High Elf")),
		crSpecies("tiefling", []content.CreatureSize{content.SizeMedium, content.SizeSmall}, textChoice("Fiendish Legacy", "Infernal", "Abyssal")),
		crSpecies("orc", []content.CreatureSize{content.SizeMedium}),
	}
}

func crTools() []content.Entry {
	calligrapher := must(content.NewStandaloneTool("calligraphers_supplies", "Calligrapher’s Supplies", noSource, content.ToolGroupArtisans, ability.Dexterity,
		[]content.ToolUtilizeAction{action("Write text", 15)}, nil, price(gp(10)), lbs(5)))
	smith := must(content.NewStandaloneTool("smiths_tools", "Smith’s Tools", noSource, content.ToolGroupArtisans, ability.Strength,
		[]content.ToolUtilizeAction{action("Repair", 15)}, nil, price(gp(20)), lbs(8)))
	thieves := must(content.NewStandaloneTool("thieves_tools", "Thieves’ Tools", noSource, content.ToolGroupOther, ability.Dexterity,
		[]content.ToolUtilizeAction{action("Pick a lock", 15)}, nil, price(gp(25)), lbs(1)))
	family := func(id, name string, variants ...string) content.ToolFamily {
		return must(content.NewToolFamily(content.ItemID(id), name, noSource, content.ToolGroupOther, ability.Wisdom,
			[]content.ToolUtilizeAction{action("Play", 10)}, nil, idsOf(variants...), content.CostVaries{}, content.WeightVaries{}))
	}
	variant := func(id, name, fam string) content.ToolVariant {
		return must(content.NewToolVariant(content.ItemID(id), name, noSource, content.ItemID(fam), price(gp(1)), lbs(1)))
	}
	return []content.Entry{calligrapher, smith, thieves,
		family("gaming_set", "Gaming Set", "dice_set", "playing_cards"), variant("dice_set", "Dice Set", "gaming_set"), variant("playing_cards", "Playing Card Set", "gaming_set"),
		family("musical_instrument", "Musical Instrument", "lute", "flute", "drum", "horn"), variant("lute", "Lute", "musical_instrument"),
		variant("flute", "Flute", "musical_instrument"), variant("drum", "Drum", "musical_instrument"),
		variant("horn", "Horn", "musical_instrument"),
	}
}

// crEntries is the content of the creation tests; extra entries are appended.
func crEntries(t *testing.T, extra ...content.Entry) []content.Entry {
	t.Helper()
	entries := []content.Entry{crFighter(t), crBard(t), crMonk(t), crWizard(t), crSoldier(), crAcolyte(), crSage(),
		savageFeat(), initiateFeat(), alertFeat(), skilledFeat(), asiFeat()}
	entries = append(entries, crSpeciesAll()...)
	entries = append(entries, crTools()...)
	for _, s := range miSpells(t) {
		entries = append(entries, s)
	}
	return append(entries, extra...)
}

func crSource(t *testing.T, extra ...content.Entry) *contenttest.FakeSource {
	t.Helper()
	return contenttest.NewFakeSource(crEntries(t, extra...)...)
}

// crScores is the Fighter row of the Standard Array by Class table (p. 21).
func crScores() character.AbilityScoreAssignment {
	return character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
		ability.Strength: 15, ability.Dexterity: 14, ability.Constitution: 13, ability.Intelligence: 8, ability.Wisdom: 10, ability.Charisma: 12})
}

// crCommand is a valid level 1 Fighter command: Soldier, Dwarf, Dice Set, class option A.
func crCommand() character.CreateCharacterCommand {
	return character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier("pc-1"), ClassID: "fighter", BackgroundID: "soldier", SpeciesID: "dwarf",
		AbilityScores: crScores(), BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Strength, ability.Constitution),
		ClassSkills: []content.Skill{content.SkillAthletics, content.SkillPerception}, ClassEquipment: character.OptionA,
		BackgroundEquipment: character.OptionA, BackgroundTool: character.ChosenTool{Tool: "dice_set"},
		Languages: []content.Language{content.LanguageDwarvish, content.LanguageElvish}, Alignment: content.AlignmentLawfulGood,
	}
}

func crUseCase(t *testing.T, src character.ContentSource, roller character.Roller, opts ...character.Option) character.CreateCharacter {
	t.Helper()
	uc, err := character.NewCreateCharacter(append([]character.Option{character.WithSource(src), character.WithRoller(roller)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

// crRun runs cmd against the standard content with a roller that fails the test if it is called.
func crRun(t *testing.T, cmd character.CreateCharacterCommand, rolls ...int) (character.CreateCharacterResult, *dicetest.FixedRoller) {
	t.Helper()
	roller := dicetest.NewFixedRoller(rolls...)
	res, err := crUseCase(t, crSource(t), roller).Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res, roller
}

func crCreated(t *testing.T, res character.CreateCharacterResult) character.CreateCharacterSucceeded {
	t.Helper()
	s, ok := res.(character.CreateCharacterSucceeded)
	if !ok {
		t.Fatalf("result = %#v, want CreateCharacterSucceeded", res)
	}
	return s
}

// crProblems returns the (kind, step) pairs of a rejection, in order.
func crKinds(t *testing.T, res character.CreateCharacterResult) []character.CreateCharacterProblemKind {
	t.Helper()
	r, ok := res.(character.CreateCharacterRejected)
	if !ok {
		t.Fatalf("result = %#v, want CreateCharacterRejected", res)
	}
	var kinds []character.CreateCharacterProblemKind
	for _, p := range r.Problems() {
		kinds = append(kinds, p.Kind())
	}
	return kinds
}
