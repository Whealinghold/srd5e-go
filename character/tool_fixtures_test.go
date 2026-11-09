package character_test

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Synthetic tool entries for the tool proficiency tests; ids and numbers follow
// the spec's "Cases: tool facts" table.

func action(description string, dc int) content.ToolUtilizeAction {
	return content.MustToolUtilizeAction(description, dc)
}

func standaloneTool(id string, ab ability.Ability, acts ...content.ToolUtilizeAction) content.StandaloneTool {
	return must(content.NewStandaloneTool(content.ItemID(id), id, noSource, content.ToolGroupArtisans, ab, acts, nil, price(gp(5)), lbs(1)))
}

func familyTool(id string, ab ability.Ability, variants []string, acts ...content.ToolUtilizeAction) content.ToolFamily {
	return must(content.NewToolFamily(content.ItemID(id), id, noSource, content.ToolGroupOther, ab, acts, nil, idsOf(variants...), content.CostVaries{}, content.WeightVaries{}))
}

func variantTool(id, family string) content.ToolVariant {
	return must(content.NewToolVariant(content.ItemID(id), id, noSource, content.ItemID(family), price(gp(1)), lbs(1)))
}

// gamingSet is the Gaming Set family of the spec's table with two variants.
func gamingSet() content.ToolFamily {
	return familyTool("gaming_set", ability.Wisdom, []string{"gaming_set_dice", "gaming_set_playing_cards"}, action("Discern cheating", 10), action("Win the game", 20))
}

func herbalismKit() content.StandaloneTool {
	return standaloneTool("herbalism_kit", ability.Intelligence, action("Identify a plant", 10))
}

func thievesTools() content.StandaloneTool {
	return standaloneTool("thieves_tools", ability.Dexterity, action("Pick a lock", 15), action("Disarm a trap", 15))
}

func resolvedTool(tool content.Tool, catalogue ...content.Tool) character.ResolvedTool {
	r, ok := character.ResolveTool(tool, catalogue).(character.ToolResolved)
	if !ok {
		panic("fixture tool does not resolve") // a broken fixture, not a failure of the code under test
	}
	return r.Tool()
}

func skillsOf(proficient, expertise []content.Skill) character.SkillProficiencies {
	return character.MustSkillProficiencies(proficient, expertise)
}
