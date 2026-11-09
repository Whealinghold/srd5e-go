package character_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/character-creation.md FR-013, FR-028, FR-064 and the table "Level 1 Hit Point cases" - the background increase is
// applied before character.New, so the Constitution modifier of the Hit Point maximum is the increased one.
// Path: per case.
func TestCreateCharacterHitPointsUseIncreasedConstitution(t *testing.T) {
	t.Parallel()
	random := func(con int) character.AbilityScoreAssignment {
		return character.MustAbilityScoreAssignment(character.RandomGeneration, map[ability.Ability]int{
			ability.Strength: 10, ability.Dexterity: 10, ability.Constitution: con, ability.Intelligence: 10, ability.Wisdom: 10, ability.Charisma: 10})
	}
	standard := func(str, dex, con int) character.AbilityScoreAssignment {
		return character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
			ability.Strength: str, ability.Dexterity: dex, ability.Constitution: con, ability.Intelligence: 8, ability.Wisdom: 10, ability.Charisma: 12})
	}
	for _, c := range []struct {
		name, requirement, path string
		class, background       string
		skills                  []content.Skill
		scores                  character.AbilityScoreAssignment
		increase                character.BackgroundAbilityIncrease
		wantCon, wantHP         int
	}{
		{"plus two Strength plus one Constitution lifts the modifier", "p. 21, p. 22", "unhappy (rule interaction)", "fighter", "soldier", nil, crScores(), character.MustPlusTwoPlusOne(ability.Strength, ability.Constitution), 14, 12},
		{"plus one to all lifts the modifier", "p. 21, p. 22", "unhappy (rule interaction)", "fighter", "soldier", nil, crScores(), character.PlusOneToAll{}, 14, 12},
		{"plus two Constitution crosses a band", "p. 21, p. 22", "unhappy (rule interaction)", "fighter", "soldier", nil, standard(13, 14, 15), character.MustPlusTwoPlusOne(ability.Constitution, ability.Strength), 17, 13},
		{"increase lifts a negative modifier", "p. 21, p. 22", "unhappy (rule interaction)", "wizard", "sage", []content.Skill{content.SkillArcana, content.SkillHistory}, random(8), character.MustPlusTwoPlusOne(ability.Constitution, ability.Intelligence), 10, 6},
		{"modifier unchanged elsewhere", "p. 21, p. 22", "happy", "wizard", "sage", []content.Skill{content.SkillArcana, content.SkillHistory}, random(8), character.MustPlusTwoPlusOne(ability.Intelligence, ability.Wisdom), 8, 5},
		{"plus one stays at minus one", "p. 21, p. 22", "unhappy (boundary)", "wizard", "sage", []content.Skill{content.SkillArcana, content.SkillHistory}, random(8), character.MustPlusTwoPlusOne(ability.Intelligence, ability.Constitution), 9, 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			cmd.ClassID, cmd.BackgroundID, cmd.AbilityScores, cmd.BackgroundIncrease = content.ClassID(c.class), content.BackgroundID(c.background), c.scores, c.increase
			if c.skills != nil {
				cmd.ClassSkills, cmd.BackgroundTool, cmd.BackgroundEquipment = c.skills, nil, character.OptionB
				cmd.ClassSkills = []content.Skill{content.SkillArcana, content.SkillHistory}
			}
			got := crCreated(t, crMust(t, cmd))
			if con := got.Character().AbilityScores().Score(ability.Constitution).Value(); con != c.wantCon || got.Character().HitPointMaximum().Value() != c.wantHP {
				t.Errorf("Constitution %d, Hit Point maximum %d, want %d and %d [%s]", con, got.Character().HitPointMaximum().Value(), c.wantCon, c.wantHP, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-038 to FR-042, FR-106 and the table "Species cases" - a species with one size needs no
// size choice, accepts that size, and the selections of a structured choice are recorded and nothing of them is applied.
// Path: per case.
func TestCreateCharacterSpeciesCases(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		species                 string
		size                    character.SizeChoice
		selections              []character.SpeciesTraitSelection
		wantSize                content.CreatureSize
	}{
		{"single size, none chosen", "pp. 84-85", "happy", "dwarf", nil, nil, content.SizeMedium},
		{"single size, same size", "pp. 84-85", "happy", "gnome", character.ChosenSize{Size: content.SizeSmall}, []character.SpeciesTraitSelection{{Trait: "Gnomish Lineage", Option: "Forest Gnome"}}, content.SizeSmall},
		{"two sizes, medium", "p. 86", "happy", "tiefling", character.ChosenSize{Size: content.SizeMedium}, []character.SpeciesTraitSelection{{Trait: "Fiendish Legacy", Option: "Infernal"}}, content.SizeMedium},
		{"ancestry chosen", "p. 84", "happy", "dragonborn", nil, []character.SpeciesTraitSelection{{Trait: "Draconic Ancestry", Option: "Gold"}}, content.SizeMedium},
		{"species with no structured choice", "p. 86", "unhappy (empty and minimal)", "orc", nil, nil, content.SizeMedium},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			cmd.SpeciesID, cmd.Size, cmd.SpeciesSelections = content.SpeciesID(c.species), c.size, c.selections
			rec := crCreated(t, crMust(t, cmd)).Record()
			if rec.Size() != c.wantSize || !reflect.DeepEqual(rec.SpeciesSelections(), c.selections) {
				t.Errorf("size = %v, selections = %v [%s]", rec.Size(), rec.SpeciesSelections(), c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-020, FR-070, FR-109, FR-110, FR-111 and the table "Equipment option cases" - the
// letters a, b and c match the keys A, B and C, the grant lists the class items then the background items, a placeholder is replaced by
// the chosen tool where the SRD ties it to a choice, and the Bard's "of your choice" instrument stays as printed.
// Path: per case.
func TestCreateCharacterEquipmentGrants(t *testing.T) {
	t.Parallel()
	soldier := func(c *character.CreateCharacterCommand) { c.BackgroundTool = character.ChosenTool{Tool: "dice_set"} }
	for _, c := range []struct {
		name, requirement, path string
		mutate                  func(*character.CreateCharacterCommand)
		wantItems               []content.EquipmentItem
		wantGold                int
	}{
		{"Fighter A with Soldier A", "pp. 47, 83", "happy", func(c *character.CreateCharacterCommand) { soldier(c) }, bgItems("Greatsword", 1, "Flail", 1, "Spear", 1, "Dice Set", 1), 18},
		{"Fighter C with Soldier B", "pp. 47, 83", "happy", func(c *character.CreateCharacterCommand) {
			c.ClassEquipment, c.BackgroundEquipment = character.OptionC, character.OptionB
		}, nil, 205},
		{"Fighter B with Soldier A", "pp. 47, 83", "happy", func(c *character.CreateCharacterCommand) {
			c.ClassEquipment = character.OptionB
		}, bgItems("Longsword", 1, "Spear", 1, "Dice Set", 1), 25},
		{"Bard instrument left as printed", "p. 31", "unhappy (rule interaction)", func(c *character.CreateCharacterCommand) {
			c.ClassID, c.BackgroundID = "bard", "acolyte"
			c.ClassSkills = []content.Skill{content.SkillArcana, content.SkillDeception, content.SkillStealth}
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "flute"}, {Tool: "drum"}}
			c.BackgroundTool, c.BackgroundIncrease = nil, character.PlusOneToAll{}
			c.BackgroundFeatChoices = clericInitiate()
			c.AbilityScores = character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
				ability.Strength: 8, ability.Dexterity: 12, ability.Constitution: 13, ability.Intelligence: 10, ability.Wisdom: 14, ability.Charisma: 15})
		}, bgItems("Leather Armor", 1, "Musical Instrument of your choice", 1, "Holy Symbol", 1), 27},
		{"Monk instrument resolved", "p. 49", "unhappy (rule interaction)", func(c *character.CreateCharacterCommand) {
			c.ClassID, c.BackgroundID = "monk", "acolyte"
			c.ClassSkills = []content.Skill{content.SkillAcrobatics, content.SkillAthletics}
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}}
			c.BackgroundTool, c.BackgroundIncrease = nil, character.PlusOneToAll{}
			c.BackgroundFeatChoices = clericInitiate()
			c.AbilityScores = character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
				ability.Strength: 8, ability.Dexterity: 15, ability.Constitution: 13, ability.Intelligence: 10, ability.Wisdom: 14, ability.Charisma: 12})
		}, bgItems("Spear", 1, "Lute", 1, "Holy Symbol", 1), 19},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			c.mutate(&cmd)
			eq := crCreated(t, crMust(t, cmd)).Record().Equipment()
			if len(eq.Items()) != len(c.wantItems) || (len(c.wantItems) > 0 && !reflect.DeepEqual(eq.Items(), c.wantItems)) || eq.Gold() != gp(c.wantGold) {
				t.Errorf("equipment = %+v, want %v and %d GP [%s]", eq, c.wantItems, c.wantGold, c.requirement)
			}
		})
	}
}

type rejectionRow struct {
	name, requirement, path string
	mutate                  func(*character.CreateCharacterCommand)
	want                    []character.CreateCharacterProblemKind
	phase1                  bool
}

func bardCommand(c *character.CreateCharacterCommand) {
	c.ClassID, c.BackgroundID = "bard", "acolyte"
	c.ClassSkills = []content.Skill{content.SkillArcana, content.SkillDeception, content.SkillStealth}
	c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "flute"}, {Tool: "drum"}}
	c.BackgroundTool, c.BackgroundIncrease = nil, character.PlusOneToAll{}
	c.BackgroundFeatChoices = clericInitiate()
}

func monkCommand(c *character.CreateCharacterCommand) {
	bardCommand(c)
	c.ClassID = "monk"
	c.ClassSkills = []content.Skill{content.SkillAcrobatics, content.SkillAthletics}
	c.ClassTools = []character.ChosenTool{{Tool: "lute"}}
}

func rejectionRows() []rejectionRow {
	k := func(kinds ...character.CreateCharacterProblemKind) []character.CreateCharacterProblemKind {
		return kinds
	}
	langs := func(ls ...content.Language) func(*character.CreateCharacterCommand) {
		return func(c *character.CreateCharacterCommand) { c.Languages = ls }
	}
	skills := func(ss ...content.Skill) func(*character.CreateCharacterCommand) {
		return func(c *character.CreateCharacterCommand) { c.ClassSkills = ss }
	}
	increase := func(two, one ability.Ability) func(*character.CreateCharacterCommand) {
		return func(c *character.CreateCharacterCommand) {
			c.BackgroundIncrease = character.MustPlusTwoPlusOne(two, one)
		}
	}
	species := func(id string, size character.SizeChoice, sel ...character.SpeciesTraitSelection) func(*character.CreateCharacterCommand) {
		return func(c *character.CreateCharacterCommand) {
			c.SpeciesID, c.Size, c.SpeciesSelections = content.SpeciesID(id), size, sel
		}
	}
	grants := func(skill content.Skill, feat string, abilities map[string]ability.Ability) func(*character.CreateCharacterCommand) {
		return func(c *character.CreateCharacterCommand) {
			c.SpeciesGrants = must(character.NewSpeciesGrantSelections(skill, content.FeatID(feat), abilities))
		}
	}
	fixed := character.StartingLevelStep{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}
	plan := func(level int, steps ...character.StartingLevelStep) func(*character.CreateCharacterCommand) {
		return func(c *character.CreateCharacterCommand) { c.Options, c.Plan = startingLevel(level), steps }
	}
	acolyte := func(c *character.CreateCharacterCommand) {
		c.BackgroundID, c.BackgroundTool, c.BackgroundIncrease = "acolyte", nil, character.PlusOneToAll{}
		c.BackgroundFeatChoices = clericInitiate()
	}
	const lang, tooling, bg, sp = "p. 20", "pp. 31, 49", "p. 83", "pp. 84-86"
	return []rejectionRow{
		{"method not permitted", "p. 21", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			c.Options = must(character.StandardRuleOptions().WithPermittedMethods([]character.AbilityScoreGenerationMethod{character.PointCost}))
		}, k(character.AbilityScoreMethodNotPermitted), true},
		{"no language", lang, "unhappy (empty and minimal)", langs(), k(character.LanguageCountWrong), true},
		{"one language", lang, "unhappy (boundary)", langs(content.LanguageDwarvish), k(character.LanguageCountWrong), true},
		{"three languages", lang, "unhappy (boundary)", langs(content.LanguageDwarvish, content.LanguageElvish, content.LanguageGiant), k(character.LanguageCountWrong), true},
		{"a Rare language", lang, "unhappy (invalid input)", langs(content.LanguageAbyssal, content.LanguageElvish), k(character.LanguageNotStandard), true},
		{"Thieves' Cant", lang, "unhappy (invalid input)", langs(content.LanguageThievesCant, content.LanguageElvish), k(character.LanguageNotStandard), true},
		{"Common again", lang, "unhappy (invalid input)", langs(content.LanguageCommon, content.LanguageElvish), k(character.LanguageRepeated), true},
		{"the same language twice", lang, "unhappy (invalid input)", langs(content.LanguageOrc, content.LanguageOrc), k(character.LanguageRepeated), true},
		{"Unaligned", "pp. 21-22", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { c.Alignment = content.AlignmentUnaligned }, k(character.AlignmentNotChoosable), true},
		{"plan too short", "p. 24", "unhappy (boundary)", plan(3, fixed), k(character.LevelAdvancementCountWrong), true},
		{"plan too long at level 1", "p. 24", "unhappy (boundary)", plan(1, fixed), k(character.LevelAdvancementCountWrong), true},
		{"class skill repeated", "p. 47", "unhappy (invalid input)", skills(content.SkillAthletics, content.SkillAthletics), k(character.ClassSkillRepeated), true},
		{"species ability not offered", sp, "unhappy (invalid input)", grants(0, "", map[string]ability.Ability{"Elven Lineage": ability.Strength}), k(character.SpeciesAbilityNotOffered), true},
		{"problems of several steps in step order", "p. 19", "unhappy (rule interaction)", func(c *character.CreateCharacterCommand) {
			c.Alignment, c.Languages = content.AlignmentUnaligned, nil
			c.Options = must(character.StandardRuleOptions().WithPermittedMethods([]character.AbilityScoreGenerationMethod{character.PointCost}))
		}, k(character.LanguageCountWrong, character.AbilityScoreMethodNotPermitted, character.AlignmentNotChoosable), true},
		{"class skills: none", "p. 47", "unhappy (empty and minimal)", skills(), k(character.ClassSkillCountWrong), false},
		{"class skills: one", "p. 47", "unhappy (boundary)", skills(content.SkillAthletics), k(character.ClassSkillCountWrong), false},
		{"class skills: three", "p. 47", "unhappy (boundary)", skills(content.SkillAthletics, content.SkillPerception, content.SkillSurvival), k(character.ClassSkillCountWrong), false},
		{"class skill not on the list", "p. 47", "unhappy (invalid input)", skills(content.SkillAthletics, content.SkillArcana), k(character.ClassSkillNotOffered), false},
		{"Bard with two skills", "p. 31", "unhappy (boundary)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassSkills = []content.Skill{content.SkillArcana, content.SkillDeception}
		}, k(character.ClassSkillCountWrong), false},
		{"Bard without tools", tooling, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { bardCommand(c); c.ClassTools = nil }, k(character.ClassToolCountWrong), false},
		{"Bard with two tools", tooling, "unhappy (boundary)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassTools = c.ClassTools[:2]
		}, k(character.ClassToolCountWrong), false},
		{"Bard with a repeated tool", tooling, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "lute"}, {Tool: "drum"}}
		}, k(character.ClassToolCountWrong), false},
		{"Bard tool outside the category", tooling, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "flute"}, {Tool: "smiths_tools"}}
		}, k(character.ClassToolNotInCategory), false},
		{"Bard tool is a family, not a variant", tooling, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "flute"}, {Tool: "musical_instrument"}}
		}, k(character.ClassToolNotInCategory), false},
		{"Bard option C", "p. 31", "unhappy (boundary)", func(c *character.CreateCharacterCommand) { bardCommand(c); c.ClassEquipment = character.OptionC }, k(character.ClassEquipmentOptionUnknown), false},
		{"Monk without a tool", "p. 49", "unhappy (rule interaction)", func(c *character.CreateCharacterCommand) { monkCommand(c); c.ClassTools = nil },
			k(character.ClassToolCountWrong, character.EquipmentPlaceholderUnresolved), false},
		{"Monk with an Artisan's tool", "p. 49", "happy", func(c *character.CreateCharacterCommand) {
			monkCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "smiths_tools"}}
			c.ClassSkills = []content.Skill{content.SkillAcrobatics, content.SkillAthletics}
		}, nil, false},
		{"Soldier without a tool", bg, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { c.BackgroundTool = nil },
			k(character.BackgroundToolMissing, character.EquipmentPlaceholderUnresolved), false},
		{"Soldier tool outside the category", bg, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			c.BackgroundTool = character.ChosenTool{Tool: "smiths_tools"}
		}, k(character.BackgroundToolNotInCategory), false},
		{"Acolyte with a chosen tool", bg, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			acolyte(c)
			c.BackgroundTool = character.ChosenTool{Tool: "dice_set"}
		}, k(character.BackgroundToolNotAllowed), false},
		{"Soldier option C", bg, "unhappy (boundary)", func(c *character.CreateCharacterCommand) { c.BackgroundEquipment = character.OptionC }, k(character.BackgroundEquipmentOptionUnknown), false},
		{"plus two to an unlisted ability", bg, "unhappy (invalid input)", increase(ability.Intelligence, ability.Strength), k(character.BackgroundIncreaseAbilityNotListed), false},
		{"plus one to an unlisted ability", bg, "unhappy (invalid input)", increase(ability.Strength, ability.Charisma), k(character.BackgroundIncreaseAbilityNotListed), false},
		{"Acolyte feat without choices", bg, "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { acolyte(c); c.BackgroundFeatChoices = nil },
			k(character.BackgroundFeatRejected), false},
		{"two sizes, none chosen", sp, "unhappy (invalid input)", species("human", nil), k(character.SizeRequired), false},
		{"one size, another chosen", sp, "unhappy (invalid input)", species("dwarf", character.ChosenSize{Size: content.SizeSmall}), k(character.SizeNotOffered), false},
		{"size the species does not offer", sp, "unhappy (invalid input)", species("tiefling", character.ChosenSize{Size: content.SizeLarge}, character.SpeciesTraitSelection{Trait: "Fiendish Legacy", Option: "Infernal"}), k(character.SizeNotOffered), false},
		{"ancestry missing", sp, "unhappy (invalid input)", species("dragonborn", nil), k(character.SpeciesChoiceMissing), false},
		{"ancestry not an option", sp, "unhappy (invalid input)", species("dragonborn", nil, character.SpeciesTraitSelection{Trait: "Draconic Ancestry", Option: "Sonic"}), k(character.SpeciesChoiceOptionUnknown), false},
		{"trait with no choice", sp, "unhappy (invalid input)", species("elf", nil, character.SpeciesTraitSelection{Trait: "Elven Lineage", Option: "Drow"}, character.SpeciesTraitSelection{Trait: "Fiendish Legacy", Option: "Abyssal"}), k(character.SpeciesChoiceNotOffered), false},
		{"same trait twice", sp, "unhappy (invalid input)", species("elf", nil, character.SpeciesTraitSelection{Trait: "Elven Lineage", Option: "Drow"}, character.SpeciesTraitSelection{Trait: "Elven Lineage", Option: "High Elf"}), k(character.SpeciesChoiceRepeated), false},
		{"spellcasting ability for a trait without a choice", sp, "unhappy (invalid input)", grants(0, "", map[string]ability.Ability{"Fiendish Legacy": ability.Charisma}), k(character.SpeciesGrantNotOffered), false},
		{"species feat that is not an Origin feat", sp, "unhappy (invalid input)", grants(0, "ability_score_improvement", nil), k(character.SpeciesFeatNotOrigin), false},
		{"species feat repeated", sp, "unhappy (rule interaction)", grants(0, "savage_attacker", nil), k(character.SpeciesFeatRejected), false},
		{"level 3 plan without its subclass", "p. 19", "unhappy (state error)", plan(3, fixed, fixed), k(character.LevelAdvancementRejected), false},
		{"multiclass prerequisite short", "p. 24", "unhappy (state error)", plan(2, character.StartingLevelStep{ClassID: "wizard", HitPointChoice: character.TakeFixedValue{}}), k(character.LevelAdvancementRejected), false},
	}
}

// Requirement: per case (doc/specs/character-creation.md FR-004, FR-005, FR-009, FR-015 to FR-021, FR-025, FR-032 to FR-035, FR-039, FR-040,
// FR-046 to FR-048, FR-054, FR-068, FR-073, FR-074, FR-077, FR-105, FR-106, FR-108, FR-109, FR-110, FR-111 and the tables of section 4).
// Path: per case.
func TestCreateCharacterRejections(t *testing.T) {
	t.Parallel()
	for _, c := range rejectionRows() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			c.mutate(&cmd)
			src := crSource(t)
			roller := dicetest.NewFixedRoller()
			res, err := crUseCase(t, src, roller).Run(context.Background(), cmd)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(c.want) == 0 {
				crCreated(t, res)
				return
			}
			if got := crKinds(t, res); !reflect.DeepEqual(got, c.want) {
				t.Errorf("kinds = %v, want %v [%s]", got, c.want, c.requirement)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("rolled %v for a rejected command", roller.Requests())
			}
			if c.phase1 && len(src.Queries()) != 0 {
				t.Errorf("a phase 1 rejection looked up %v", src.Queries())
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-094, section 5 - every problem kind except the defensive BackgroundIncreaseExceedsMaximum
// is produced by some row, and every kind has a distinct snake_case name.
// Path: unhappy (invalid input).
func TestCreateCharacterProblemKindsAreAllProduced(t *testing.T) {
	t.Parallel()
	produced := map[character.CreateCharacterProblemKind]bool{}
	for _, row := range rejectionRows() {
		for _, kind := range row.want {
			produced[kind] = true
		}
	}
	names := map[string]bool{}
	for kind := character.ClassSkillCountWrong; kind.Valid(); kind++ {
		if names[kind.String()] {
			t.Errorf("kind %d repeats the name %q", kind, kind.String())
		}
		names[kind.String()] = true
		if !produced[kind] && kind != character.BackgroundIncreaseExceedsMaximum {
			t.Errorf("no row produces %s", kind)
		}
	}
	if character.CreateCharacterProblemKind(0).Valid() || character.CreateCharacterProblemKind(0).String() != "unknown" || len(names) != 31 {
		t.Errorf("kinds = %d", len(names))
	}
}

// Requirement: doc/specs/character-creation.md FR-112 and the Composition row "multiclass start" - a plan step in a second class adds no skill or
// tool choice: the record holds the first class's choices only, and the multiclass start is created when the prerequisites hold.
// Path: unhappy (rule interaction).
func TestCreateCharacterMulticlassStepAddsNoChoices(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.AbilityScores = character.MustAbilityScoreAssignment(character.StandardArray, rawScores(15, 8, 14, 13, 10, 12))
	cmd.BackgroundIncrease = character.PlusOneToAll{}
	cmd.Options = startingLevel(2)
	cmd.Plan = []character.StartingLevelStep{{ClassID: "wizard", HitPointChoice: character.TakeFixedValue{}}}
	got := crCreated(t, crMust(t, cmd))
	if want := []content.Skill{content.SkillAthletics, content.SkillPerception, content.SkillIntimidation}; !reflect.DeepEqual(got.Record().Skills(), want) || len(got.Record().ClassTools()) != 0 {
		t.Errorf("skills = %v, tools = %v", got.Record().Skills(), got.Record().ClassTools())
	}
	if levels := got.Character().ClassLevels().Entries(); len(levels) != 2 {
		t.Errorf("classes = %v", levels)
	}
}

// Requirement: doc/specs/character-creation.md FR-034, FR-111, Q-18 and the table "Class skills cases" - a tool belongs to a category by its group
// (Artisan's Tools) or by the family a variant belongs to; a Gaming Set is not a Musical Instrument or an Artisan's tool, and a tool of the
// Other group is not an Artisan's tool.
// Path: per case.
func TestCreateCharacterToolCategories(t *testing.T) {
	t.Parallel()
	artisanFamily := must(content.NewToolFamily("artisan_set", "Artisan Set", noSource, content.ToolGroupArtisans, ability.Dexterity,
		[]content.ToolUtilizeAction{action("Make", 10)}, nil, idsOf("artisan_variant"), content.CostVaries{}, content.WeightVaries{}))
	artisanVariant := must(content.NewToolVariant("artisan_variant", "Artisan Variant", noSource, "artisan_set", price(gp(1)), lbs(1)))
	oddVariant := must(content.NewToolVariant("odd_variant", "Odd Variant", noSource, "smiths_tools", price(gp(1)), lbs(1)))
	for _, c := range []struct {
		name, requirement, path string
		mutate                  func(*character.CreateCharacterCommand)
		want                    []character.CreateCharacterProblemKind
	}{
		{"Soldier with a Musical Instrument", "p. 83", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { c.BackgroundTool = character.ChosenTool{Tool: "lute"} }, []character.CreateCharacterProblemKind{character.BackgroundToolNotInCategory}},
		{"Monk with a Gaming Set variant", "p. 49", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			monkCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "dice_set"}}
		}, []character.CreateCharacterProblemKind{character.ClassToolNotInCategory}},
		{"Monk with a tool of the Other group", "p. 49", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			monkCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "thieves_tools"}}
		}, []character.CreateCharacterProblemKind{character.ClassToolNotInCategory}},
		{"Monk with a variant of an Artisan's family", "p. 49", "happy", func(c *character.CreateCharacterCommand) {
			monkCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "artisan_variant"}}
		}, nil},
		{"Bard with a variant whose family is not a family", "p. 31", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "flute"}, {Tool: "odd_variant"}}
		}, []character.CreateCharacterProblemKind{character.ClassToolNotInCategory}},
		{"Bard with four skills", "p. 31", "unhappy (boundary)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassSkills = []content.Skill{content.SkillArcana, content.SkillDeception, content.SkillStealth, content.SkillHistory}
		}, []character.CreateCharacterProblemKind{character.ClassSkillCountWrong}},
		{"Bard with four tools", "p. 31", "unhappy (boundary)", func(c *character.CreateCharacterCommand) {
			bardCommand(c)
			c.ClassTools = []character.ChosenTool{{Tool: "lute"}, {Tool: "flute"}, {Tool: "drum"}, {Tool: "horn"}}
		}, []character.CreateCharacterProblemKind{character.ClassToolCountWrong}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			c.mutate(&cmd)
			res := must(crUseCase(t, crSource(t, artisanFamily, artisanVariant, oddVariant), dicetest.NewFixedRoller()).Run(context.Background(), cmd))
			if len(c.want) == 0 {
				crCreated(t, res)
				return
			}
			if got := crKinds(t, res); !reflect.DeepEqual(got, c.want) {
				t.Errorf("kinds = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-040, FR-106 and the table "Species cases" - a selection answers the trait it names only (an Elven
// Lineage selection does not answer a Fiendish Legacy), an option is known by its exact name, and the spellcasting ability of a lineage is
// Intelligence, Wisdom or Charisma and nothing else.
// Path: per case.
func TestCreateCharacterSpeciesSelectionsAndAbilities(t *testing.T) {
	t.Parallel()
	one := func(kind ...character.CreateCharacterProblemKind) []character.CreateCharacterProblemKind { return kind }
	for _, c := range []struct {
		name, requirement, path string
		mutate                  func(*character.CreateCharacterCommand)
		want                    []character.CreateCharacterProblemKind
	}{
		{"a selection of another trait is not an answer", "p. 86", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			c.SpeciesID, c.Size, c.SpeciesSelections = "tiefling", character.ChosenSize{Size: content.SizeSmall}, []character.SpeciesTraitSelection{{Trait: "Elven Lineage", Option: "Drow"}}
		}, one(character.SpeciesChoiceMissing, character.SpeciesChoiceNotOffered)},
		{"an option that sorts before the real ones", "p. 84", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) {
			c.SpeciesID, c.SpeciesSelections = "dragonborn", []character.SpeciesTraitSelection{{Trait: "Draconic Ancestry", Option: "Blue"}}
		}, one(character.SpeciesChoiceOptionUnknown)},
		{"Intelligence for a lineage", "pp. 84-86", "happy", func(c *character.CreateCharacterCommand) { lineageAbility(c, ability.Intelligence) }, nil},
		{"Wisdom for a lineage", "pp. 84-86", "happy", func(c *character.CreateCharacterCommand) { lineageAbility(c, ability.Wisdom) }, nil},
		{"Charisma for a lineage", "pp. 84-86", "happy", func(c *character.CreateCharacterCommand) { lineageAbility(c, ability.Charisma) }, nil},
		{"Strength for a lineage", "pp. 84-86", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { lineageAbility(c, ability.Strength) }, one(character.SpeciesAbilityNotOffered)},
		{"Dexterity for a lineage", "pp. 84-86", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { lineageAbility(c, ability.Dexterity) }, one(character.SpeciesAbilityNotOffered)},
		{"Constitution for a lineage", "pp. 84-86", "unhappy (invalid input)", func(c *character.CreateCharacterCommand) { lineageAbility(c, ability.Constitution) }, one(character.SpeciesAbilityNotOffered)},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			c.mutate(&cmd)
			res := crMust(t, cmd)
			if len(c.want) == 0 {
				crCreated(t, res)
				return
			}
			if got := crKinds(t, res); !reflect.DeepEqual(got, c.want) {
				t.Errorf("kinds = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

func lineageAbility(c *character.CreateCharacterCommand, a ability.Ability) {
	c.SpeciesID, c.SpeciesSelections = "elf", []character.SpeciesTraitSelection{{Trait: "Elven Lineage", Option: "Drow"}}
	c.SpeciesGrants = must(character.NewSpeciesGrantSelections(0, "", map[string]ability.Ability{"Elven Lineage": a}))
}

// Requirement: doc/specs/character-creation.md FR-074, FR-078 - the class of the command is the initial class even when a class of the plan sorts
// before it: a Monk (d8) who takes a Fighter level has the Monk's level 1 Hit Points plus the Fighter's fixed gain.
// Path: unhappy (rule interaction).
func TestCreateCharacterInitialClassIsTheCommandsClass(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	monkCommand(&cmd)
	cmd.ClassTools = []character.ChosenTool{{Tool: "lute"}}
	cmd.AbilityScores = character.MustAbilityScoreAssignment(character.StandardArray, rawScores(13, 15, 12, 8, 14, 10))
	cmd.BackgroundIncrease = character.PlusOneToAll{}
	cmd.Options = startingLevel(2)
	cmd.Plan = []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}}
	got := crCreated(t, crMust(t, cmd)).Character()
	// Constitution 12 (+1): d8 maximum 8 + 1 at level 1, then the d10 fixed value 6 + 1.
	if got.InitialClass() != "monk" || got.HitPointMaximum().Value() != 16 {
		t.Errorf("initial class %q, Hit Point maximum %d, want monk and 16", got.InitialClass(), got.HitPointMaximum().Value())
	}
}
