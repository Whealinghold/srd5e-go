package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The scenarios of tool-proficiency-and-crafting.md. The bundled slice has no tool
// entries yet (Phase 5), so tools and equipment are synthetic with the numbers of
// SRD 5.2.1, pp. 93-94 and 103; spells are the bundled ones.

func grpD3Action(description string, dc int) content.ToolUtilizeAction {
	return content.MustToolUtilizeAction(description, dc)
}

func grpD3Standalone(id string, ab ability.Ability, acts ...content.ToolUtilizeAction) content.StandaloneTool {
	return grpD3Must(content.NewStandaloneTool(content.ItemID(id), id, content.SourceCitation{}, content.ToolGroupArtisans, ab, acts, nil, grpD3Price(grpD3GP(5)), grpD3Lbs(1)))
}

func grpD3Family(id string, ab ability.Ability, variants []string, acts ...content.ToolUtilizeAction) content.ToolFamily {
	return grpD3Must(content.NewToolFamily(content.ItemID(id), id, content.SourceCitation{}, content.ToolGroupOther, ab, acts, nil, grpD3IDs(variants...), content.CostVaries{}, content.WeightVaries{}))
}

func grpD3Variant(id, family string) content.ToolVariant {
	return grpD3Must(content.NewToolVariant(content.ItemID(id), id, content.SourceCitation{}, content.ItemID(family), grpD3Price(grpD3GP(1)), grpD3Lbs(1)))
}

func grpD3Thieves() content.StandaloneTool {
	return grpD3Standalone("thieves_tools", ability.Dexterity, grpD3Action("Pick a lock", 15), grpD3Action("Disarm a trap", 15))
}

func grpD3GamingSet() content.ToolFamily {
	return grpD3Family("gaming_set", ability.Wisdom, []string{"gaming_set_dice", "gaming_set_cards"}, grpD3Action("Discern cheating", 10), grpD3Action("Win the game", 20))
}

func grpD3Resolve(t *testing.T, tool content.Tool, catalogue ...content.Tool) character.ResolvedTool {
	t.Helper()
	r, ok := character.ResolveTool(tool, catalogue).(character.ToolResolved)
	if !ok {
		t.Fatalf("ResolveTool(%v) = %#v", tool.ID(), character.ResolveTool(tool, catalogue))
	}
	return r.Tool()
}

// A bard learns a gaming set: proficiency is per entry id, so dice are not cards
// and neither is the family; a variant inherits the family's Ability and
// Utilize list; a family or a variant of nothing is not a proficiency (SRD 5.2.1,
// "Tools", pp. 93-94).
// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-001 to FR-006, FR-018, FR-048 - proficiency per entry id, copies, monsters, a variant resolves through its family, a family alone needs a variant, a missing family is reported, not thrown.
// Path: happy; unhappy (rule interaction, missing content, empty and minimal, immutability).
func TestGrpD3ToolProficiencyIsPerEntryAndVariantsInheritTheFamily(t *testing.T) {
	t.Parallel()
	gaming := grpD3GamingSet()
	dice, cards := grpD3Variant("gaming_set_dice", "gaming_set"), grpD3Variant("gaming_set_cards", "gaming_set")
	thieves := grpD3Thieves()
	catalogue := []content.Tool{gaming, dice, cards, thieves}

	prof := character.NewToolProficiencies("gaming_set_dice", "thieves_tools")
	if !prof.Contains("gaming_set_dice") || prof.Contains("gaming_set_cards") || prof.Contains("gaming_set") {
		t.Error("dice proficiency must not count for cards or for the family id")
	}
	more := prof.Plus("gaming_set_cards")
	if !more.Contains("gaming_set_cards") || prof.Contains("gaming_set_cards") {
		t.Error("Plus must return a new value and leave the original alone")
	}
	reordered := character.NewToolProficiencies("thieves_tools", "gaming_set_dice")
	if !prof.Equal(reordered) {
		t.Error("equal sets in another order must be Equal")
	}
	ids := prof.IDs()
	ids[0] = "changed"
	if !prof.Contains("gaming_set_dice") {
		t.Error("mutating IDs() changed the proficiencies")
	}
	if got := character.MonsterToolProficiencies("thieves_tools"); !got.Contains("thieves_tools") || got.Contains("gaming_set_dice") {
		t.Errorf("a monster is proficient with exactly its stat block tools, got %v", got.IDs())
	}

	// A standalone tool keeps its own facts; a variant takes the family's and its own id.
	rt := grpD3Resolve(t, thieves, catalogue...)
	if rt.ProficiencyID() != "thieves_tools" || rt.Ability() != ability.Dexterity || len(rt.Utilize()) != 2 || rt.Group() != content.ToolGroupArtisans {
		t.Errorf("Thieves' Tools resolved to %v, %v, %d actions, %v", rt.ProficiencyID(), rt.Ability(), len(rt.Utilize()), rt.Group())
	}
	rd := grpD3Resolve(t, dice, catalogue...)
	if rd.ProficiencyID() != "gaming_set_dice" || rd.Ability() != ability.Wisdom || rd.Group() != content.ToolGroupOther {
		t.Errorf("Dice resolved to %v, %v, %v; want its own id with the family's Wisdom", rd.ProficiencyID(), rd.Ability(), rd.Group())
	}
	if got := character.UtilizeActions(rd); len(got) != 2 || got[0].Description() != "Discern cheating" || got[1].DC() != 20 {
		t.Errorf("Dice Utilize list = %v, want the family's in printed order", got)
	}
	acts := rd.Utilize()
	acts[0] = grpD3Action("changed", 1)
	if rd.Utilize()[0].Description() != "Discern cheating" {
		t.Error("mutating Utilize() changed the tool")
	}

	// A family alone is not a proficiency; a variant whose family is absent or not a family is reported.
	if r, ok := character.ResolveTool(gaming, catalogue).(character.ToolFamilyNeedsVariant); !ok || r.FamilyID() != "gaming_set" {
		t.Errorf("the family alone = %#v", character.ResolveTool(gaming, catalogue))
	}
	orphan := grpD3Variant("kazoo", "musical_instrument")
	if r, ok := character.ResolveTool(orphan, catalogue).(character.ToolFamilyMissing); !ok || r.VariantID() != "kazoo" || r.FamilyID() != "musical_instrument" {
		t.Errorf("a variant with no family = %#v", character.ResolveTool(orphan, catalogue))
	}
	notFamily := grpD3Variant("odd", "thieves_tools")
	if _, ok := character.ResolveTool(notFamily, catalogue).(character.ToolFamilyMissing); !ok {
		t.Errorf("a variant of a standalone tool = %#v", character.ResolveTool(notFamily, catalogue))
	}
	if r, ok := character.ResolveTool(nil, catalogue).(character.ToolFamilyMissing); !ok || r.VariantID() != "" || r.FamilyID() != "" {
		t.Errorf("a nil tool = %#v", character.ResolveTool(nil, catalogue))
	}
}

// A rogue picks a lock: tool proficiency adds the Proficiency Bonus once, a
// matching skill adds Advantage, Expertise in the skill doubles the bonus but
// never triples it, and the check is scored against the Utilize DC through the
// real D20 resolver (SRD 5.2.1, "Tool Proficiency", p. 93; "Expertise", p. 182).
// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-007 to FR-021 - the multiple is 0, 1 or 2 and never 3; Advantage needs both proficiencies; zero SkillUsed reads no skill; the Ability may be overridden; Utilize DC per action; an unlisted action or foreign benefits are results, not errors.
// Path: happy; unhappy (boundary, rule interaction, empty and minimal, invalid input).
func TestGrpD3ToolCheckBenefitsAndUtilizeAgainstTheResolver(t *testing.T) {
	t.Parallel()
	thieves := grpD3Resolve(t, grpD3Thieves())
	tools := character.NewToolProficiencies("thieves_tools")
	sleight, stealth := content.SkillSleightOfHand, content.SkillStealth
	skills := func(proficient, expert []content.Skill) character.SkillProficiencies {
		return character.MustSkillProficiencies(append(slices.Clone(proficient), expert...), expert) // Expertise needs proficiency
	}

	for _, c := range []struct {
		name     string
		in       character.ToolCheckInput
		multiple int
		toolProf bool
		advHere  bool
		abil     ability.Ability
	}{
		{"neither proficiency", character.ToolCheckInput{Tool: thieves, Skills: skills(nil, nil), SkillUsed: sleight}, 0, false, false, ability.Dexterity},
		{"tool and skill: once, with Advantage", character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills([]content.Skill{sleight}, nil), SkillUsed: sleight}, 1, true, true, ability.Dexterity},
		{"skill only: the skill's bonus, no Advantage", character.ToolCheckInput{Tool: thieves, Skills: skills([]content.Skill{sleight}, nil), SkillUsed: sleight}, 1, false, false, ability.Dexterity},
		{"tool only", character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills(nil, nil)}, 1, true, false, ability.Dexterity},
		{"Expertise without the tool: twice", character.ToolCheckInput{Tool: thieves, Skills: skills(nil, []content.Skill{sleight}), SkillUsed: sleight}, 2, false, false, ability.Dexterity},
		{"Expertise and the tool: still twice, never three times", character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills(nil, []content.Skill{sleight}), SkillUsed: sleight}, 2, true, true, ability.Dexterity},
		{"Expertise in another skill changes nothing", character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills(nil, []content.Skill{stealth}), SkillUsed: sleight}, 1, true, false, ability.Dexterity},
		{"no skill given reads no skill", character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills([]content.Skill{sleight}, []content.Skill{stealth})}, 1, true, false, ability.Dexterity},
		{"the Game Master pairs the tool with Intelligence", character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills(nil, nil), Ability: ability.Intelligence}, 1, true, false, ability.Intelligence},
	} {
		b := character.EvaluateToolCheck(c.in)
		if b.ProficiencyBonusMultiple() != c.multiple || b.ToolProficient() != c.toolProf || b.Ability() != c.abil || b.UsesProficiency() != (c.multiple > 0) {
			t.Errorf("%s: multiple %d, tool %v, ability %v, uses %v", c.name, b.ProficiencyBonusMultiple(), b.ToolProficient(), b.Ability(), b.UsesProficiency())
		}
		circ := b.Circumstances()
		if c.advHere != (len(circ) == 1) || (len(circ) == 1 && circ[0] != (modifier.Advantage{})) {
			t.Errorf("%s: circumstances = %v, want Advantage %v", c.name, circ, c.advHere)
		}
		if b != character.EvaluateToolCheck(c.in) {
			t.Errorf("%s: the evaluation is not deterministic", c.name)
		}
	}

	// The Advantage joins others only through the kernel: a condition's Disadvantage cancels it.
	both := character.EvaluateToolCheck(character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills([]content.Skill{sleight}, nil), SkillUsed: sleight})
	if got := modifier.Resolve(both.Circumstances()...); got != modifier.WithAdvantage {
		t.Errorf("Advantage alone = %v", got)
	}
	if got := modifier.Resolve(append(both.Circumstances(), modifier.Disadvantage{})...); got != modifier.Normal {
		t.Errorf("Advantage with a condition's Disadvantage = %v, want normal", got)
	}

	// The story: a level 5 rogue (Proficiency Bonus +3, Dexterity 16) has Expertise in Sleight of Hand and Thieves' Tools.
	pb := character.ProficiencyBonusFor(character.MustLevel(5))
	expert := character.EvaluateToolCheck(character.ToolCheckInput{Tool: thieves, Tools: tools, Skills: skills(nil, []content.Skill{sleight}), SkillUsed: sleight})
	lockAction := grpD3Action("Pick a lock", 15)
	check, ok := character.UtilizeCheck(thieves, lockAction, expert).(character.ToolUtilizeCheck)
	if !ok || check.Ability() != ability.Dexterity || check.DifficultyClass() != d20.DifficultyClass(15) {
		t.Fatalf("UtilizeCheck = %#v", character.UtilizeCheck(thieves, lockAction, expert))
	}
	rm := check.Benefits().RollModifiers(grpD3Score(16).Modifier(), pb)
	if got, hasPB := rm.ProficiencyBonus(); !hasPB || got.Value() != 6 || rm.Total().Value() != 9 {
		t.Errorf("modifiers = %v / %v, want a doubled +6 bonus and +9 in total", got, rm.Total())
	}
	for natural, want := range map[int]d20.Outcome{6: d20.Success, 5: d20.Failure} {
		roll, err := d20.NewRoll(modifier.Normal, []int{natural})
		if err != nil {
			t.Fatal(err)
		}
		res, err := d20.ResolveCheck(d20.AbilityCheck, roll, rm, check.DifficultyClass())
		if err != nil || res.Outcome() != want {
			t.Errorf("natural %d + 9 against DC 15 = %v, %v; want %v", natural, res.Outcome(), err, want)
		}
	}
	// A check with no tool or skill proficiency carries no Proficiency Bonus slot at all.
	none := character.EvaluateToolCheck(character.ToolCheckInput{Tool: thieves})
	if _, hasPB := none.RollModifiers(grpD3Score(16).Modifier(), pb).ProficiencyBonus(); hasPB {
		t.Error("a non-proficient check must not carry a Proficiency Bonus slot")
	}

	// One action per Utilize: each action has its own DC; an action the tool does not list has none.
	if dc, ok := character.UtilizeDC(thieves, grpD3Action("Disarm a trap", 15)).(character.ToolUtilizeDC); !ok || dc.DifficultyClass() != 15 {
		t.Errorf("Disarm a trap DC = %#v", character.UtilizeDC(thieves, grpD3Action("Disarm a trap", 15)))
	}
	stray := grpD3Action("Forge a seal", 20)
	if r, ok := character.UtilizeDC(thieves, stray).(character.ToolUtilizeActionNotListed); !ok || r.Action() != stray {
		t.Errorf("an unlisted action = %#v", character.UtilizeDC(thieves, stray))
	}
	if _, ok := character.UtilizeCheck(thieves, stray, expert).(character.ToolUtilizeActionNotListed); !ok {
		t.Error("UtilizeCheck of an unlisted action must not score a guessed DC")
	}
	// Benefits evaluated for another tool must not be scored against this one.
	herbs := grpD3Resolve(t, grpD3Standalone("herbalism_kit", ability.Intelligence, grpD3Action("Identify a plant", 10)))
	foreign := character.EvaluateToolCheck(character.ToolCheckInput{Tool: herbs})
	if m, ok := character.UtilizeCheck(thieves, lockAction, foreign).(character.ToolUtilizeBenefitsMismatch); !ok || m.ToolID() != "thieves_tools" || m.BenefitsToolID() != "herbalism_kit" {
		t.Errorf("foreign benefits = %#v", character.UtilizeCheck(thieves, lockAction, foreign))
	}
}

func grpD3Worker(id string, tools ...content.ItemID) character.CraftingWorker {
	return character.NewCraftingWorker(character.MustIdentifier(id), character.NewToolProficiencies(tools...))
}

// A smith takes a helper to forge Plate Armor, a brewer tries to bring one to a
// potion, and a club has two possible tools; each party gets a plan or its exact
// shortfalls (SRD 5.2.1, "Crafting Nonmagical Items", p. 103).
// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-023 to FR-033 - every worker needs one shared tool; WorkHours is 8 x days over the workers, unrounded; shortfalls are ordered tool, too many helpers, helpers not allowed; a Game Master tool is read only for items that list none; not craftable and scroll items skip the party.
// Path: happy; unhappy (rule interaction, boundary, invalid input, missing content).
func TestGrpD3CraftingPlansAreReadyOrBlockedWithExactShortfalls(t *testing.T) {
	t.Parallel()
	src := content.SourceCitation{}
	toolbox := grpD3Toolbox()
	plate := grpD3Armor("plate_x", content.ArmorHeavy, grpD3NoDex(18), 15, true, 65, grpD3GP(1500))
	smith := grpD3Worker("smith", "smiths_tools")
	apprentice := grpD3Worker("apprentice", "smiths_tools")
	novice := grpD3Worker("novice", "woodcarvers_tools")

	// Alone, then with one helper: the same requirements, the hours shared out.
	solo, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Party: character.MustCraftingParty(smith)}).(character.ItemCraftingReady)
	if !ok {
		t.Fatal("a proficient smith alone is not ready")
	}
	if solo.Requirements().Days() != 150 || solo.Requirements().RawMaterials() != grpD3CP(75000) || solo.Workers() != 1 ||
		!slices.Equal(solo.UsableTools(), grpD3IDs("smiths_tools")) || solo.Requirements().Basis() != character.CraftingGenericRule {
		t.Errorf("solo plan = %+v", solo)
	}
	if h := solo.WorkHours(); h.Numerator() != 1200 || h.Denominator() != 1 {
		t.Errorf("solo hours = %d/%d, want 1200/1", h.Numerator(), h.Denominator())
	}
	pair, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Party: character.MustCraftingParty(smith, apprentice)}).(character.ItemCraftingReady)
	if !ok || pair.Workers() != 2 || pair.WorkHours().Numerator() != 1200 || pair.WorkHours().Denominator() != 2 {
		t.Fatalf("a smith with a helper = %#v", pair)
	}

	// A helper without the tool, and a second helper over the cap of one.
	blocked, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Party: character.MustCraftingParty(smith, novice)}).(character.ItemCraftingBlocked)
	if !ok {
		t.Fatal("a helper with the wrong tool must block the plan")
	}
	if short := blocked.Shortfalls(); len(short) != 1 {
		t.Errorf("shortfalls = %#v", short)
	} else if w, ok := short[0].(character.ToolShortfallWorker); !ok || w.Role() != character.CraftingRoleHelper || w.WorkerID() != character.MustIdentifier("novice") ||
		!slices.Equal(w.Alternatives(), grpD3IDs("smiths_tools")) {
		t.Errorf("shortfall = %#v", short[0])
	}
	crowded := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Party: character.MustCraftingParty(smith, apprentice, grpD3Worker("third", "smiths_tools"))})
	if b, ok := crowded.(character.ItemCraftingBlocked); !ok || len(b.Shortfalls()) != 1 || !grpD3IsTooManyHelpers(b.Shortfalls()[0], 2, 1) {
		t.Errorf("two helpers = %#v", crowded)
	}
	allowance := grpD3Must(character.NewAssistantAllowance(2))
	if _, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Allowance: allowance,
		Party: character.MustCraftingParty(smith, apprentice, grpD3Worker("third", "smiths_tools"))}).(character.ItemCraftingReady); !ok {
		t.Error("the Game Master's allowance of two helpers should make the party ready")
	}
	_, err := character.NewAssistantAllowance(1)
	grpD3WantInvalid(t, err, "an allowance of 1 is the default, not a choice")

	// Shortfalls come in order: the tool first, then too many helpers.
	both := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Party: character.MustCraftingParty(smith, novice, grpD3Worker("third", "smiths_tools"))}).(character.ItemCraftingBlocked)
	if s := both.Shortfalls(); len(s) != 2 || !grpD3IsTooManyHelpers(s[1], 2, 1) {
		t.Errorf("combined shortfalls = %#v", s)
	} else if _, ok := s[0].(character.ToolShortfallWorker); !ok {
		t.Errorf("the tool shortfall must come first, got %#v", s[0])
	}

	// A club has two tools; the crafter and helper each know a different one, so there is no shared tool.
	club := grpD3Weapon("club_x", content.WeaponMelee, grpD3GP(1))
	carpenter, carver := grpD3Worker("carpenter", "carpenters_tools"), grpD3Worker("carver", "woodcarvers_tools")
	noShared := character.PlanItemCrafting(character.ItemCraftingInput{Item: club, Tools: toolbox, Party: character.MustCraftingParty(carpenter, carver)}).(character.ItemCraftingBlocked)
	if len(noShared.Shortfalls()) != 1 {
		t.Fatalf("shortfalls = %#v", noShared.Shortfalls())
	}
	if n, ok := noShared.Shortfalls()[0].(character.ToolShortfallNoSharedTool); !ok || !slices.Equal(n.Alternatives(), grpD3IDs("carpenters_tools", "woodcarvers_tools")) {
		t.Errorf("no-shared-tool shortfall = %#v", noShared.Shortfalls()[0])
	}
	if r, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: club, Tools: toolbox, Party: character.MustCraftingParty(carpenter, grpD3Worker("both", "carpenters_tools", "woodcarvers_tools"))}).(character.ItemCraftingReady); !ok || !slices.Equal(r.UsableTools(), grpD3IDs("carpenters_tools")) {
		t.Errorf("sharing one of two alternatives = %#v", r)
	}

	// Potion of Healing: the override's tool, 25 GP and a day; a helper is not allowed.
	override := grpD3Must(content.NewCraftingOverride(grpD3GP(25), 1, grpD3IDs("herbalism_kit")))
	potion := grpD3Must(content.NewMagicGear("potion_x", "Potion of Healing", src, grpD3Price(grpD3GP(50)), content.NoWeight{}, "", "Text.", override))
	brewer := grpD3Worker("brewer", "herbalism_kit")
	brew, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: potion, Tools: toolbox, Party: character.MustCraftingParty(brewer)}).(character.ItemCraftingReady)
	if !ok || brew.Requirements().RawMaterials() != grpD3GP(25) || brew.Requirements().Days() != 1 || brew.WorkHours().Numerator() != 8 || brew.Requirements().Basis() != character.CraftingItemOverride {
		t.Fatalf("brewing alone = %#v", brew)
	}
	helped := character.PlanItemCrafting(character.ItemCraftingInput{Item: potion, Tools: toolbox, Party: character.MustCraftingParty(brewer, grpD3Worker("helper", "herbalism_kit"))}).(character.ItemCraftingBlocked)
	if len(helped.Shortfalls()) != 1 || helped.Shortfalls()[0].(character.ToolShortfallHelpersNotAllowed).Basis() != character.CraftingItemOverride {
		t.Errorf("a helper for a potion = %#v", helped.Shortfalls())
	}

	// Items the rule cannot plan never read the party: not craftable, a scroll, an item with no listed tool.
	horse := grpD3Must(content.NewMount("horse_x", "Horse", src, 540, grpD3Price(grpD3GP(50))))
	if n, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: horse, Tools: toolbox, Party: character.MustCraftingParty(smith)}).(character.ItemCraftingNotCraftable); !ok || n.Reason() != character.NotCraftableNotAnItem {
		t.Errorf("a horse = %#v", n)
	}
	scroll := grpD3Must(content.NewScrollVariant("scroll_x", "Scroll", src, "scroll_family", grpD3Price(grpD3GP(30)), "", "", 0))
	if _, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: scroll, Tools: toolbox, Scribing: character.MustScrollScribing(grpD3ScrollRows()), Party: character.MustCraftingParty(smith)}).(character.ItemCraftingUseScribing); !ok {
		t.Error("a Spell Scroll must be sent to scribing")
	}
	shield := grpD3Armor("shield_x", content.ArmorShield, grpD3Must(content.NewBonusRule(2)), 0, false, 6, grpD3GP(10))
	if _, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: shield, Tools: toolbox, Party: character.MustCraftingParty(smith)}).(character.ItemCraftingNeedsGMTool); !ok {
		t.Error("an item with no listed tool and no assigned one needs the Game Master's tool")
	}
	assigned, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: shield, Tools: toolbox, Party: character.MustCraftingParty(smith), GMAssignedTool: "smiths_tools"}).(character.ItemCraftingReady)
	if !ok || !slices.Equal(assigned.UsableTools(), grpD3IDs("smiths_tools")) {
		t.Errorf("a Game Master assigned tool = %#v", assigned)
	}
	if g, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: shield, Tools: toolbox, Party: character.MustCraftingParty(smith), GMAssignedTool: "no_such_tool"}).(character.ItemCraftingGMToolInvalid); !ok || g.ToolID() != "no_such_tool" {
		t.Errorf("an assigned tool the catalogue lacks = %#v", g)
	}
	// An assigned tool is ignored for an item that lists its own.
	if r, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: plate, Tools: toolbox, Party: character.MustCraftingParty(smith), GMAssignedTool: "woodcarvers_tools"}).(character.ItemCraftingReady); !ok || !slices.Equal(r.UsableTools(), grpD3IDs("smiths_tools")) {
		t.Errorf("an assigned tool for a listed item = %#v", r)
	}
}

func grpD3IsTooManyHelpers(s character.ToolCraftingShortfall, count, limit int) bool {
	t, ok := s.(character.ToolShortfallTooManyHelpers)
	return ok && t.Count() == count && t.Limit() == limit
}

// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-023, FR-048 - a party rejects a repeated or zero id; MustCraftingParty panics with that error; helpers are copied on construction and on read.
// Path: unhappy (invalid input, immutability).
func TestGrpD3CraftingPartyRejectsRepeatsAndCopiesHelpers(t *testing.T) {
	t.Parallel()
	smith, helper := grpD3Worker("smith", "smiths_tools"), grpD3Worker("helper", "smiths_tools")
	_, err := character.NewCraftingParty(smith, helper, helper)
	grpD3WantInvalid(t, err, "a repeated helper")
	_, err = character.NewCraftingParty(smith, smith)
	grpD3WantInvalid(t, err, "the crafter as their own helper")
	_, err = character.NewCraftingParty(character.NewCraftingWorker(character.Identifier{}, character.ToolProficiencies{}))
	grpD3WantInvalid(t, err, "a crafter with the zero id")
	func() {
		defer func() {
			if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
				t.Errorf("MustCraftingParty panic = %v, want an ErrInvalidInput error", r)
			}
		}()
		character.MustCraftingParty(smith, smith)
	}()

	source := []character.CraftingWorker{helper}
	party := character.MustCraftingParty(smith, source...)
	source[0] = grpD3Worker("someone-else")
	got := party.Helpers()
	got[0] = grpD3Worker("another")
	if party.Helpers()[0].ID() != character.MustIdentifier("helper") || party.Crafter().ID() != character.MustIdentifier("smith") || party.IsZero() {
		t.Error("the party changed when its inputs or a read were mutated")
	}
	if !(character.CraftingParty{}).IsZero() {
		t.Error("the zero party is invalid")
	}
}

// A wizard scribes scrolls from the bundled SRD spells: the scribe needs Arcana or
// Calligrapher's Supplies, a day only counts when the spell is prepared and the
// materials are at hand, materials are used up only on completion, and a cantrip
// scroll keeps the scribe's level (SRD 5.2.1, "Scribing Spell Scrolls", p. 103).
// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-034 to FR-043 - qualification by Arcana (or Expertise) or Calligrapher's Supplies; rejections are results; days count under the preparation and material rules; the scroll holds DC, bonus and caster level; a finished job accepts no more days.
// Path: happy; unhappy (rule interaction, boundary, missing content, state error, invalid input).
func TestGrpD3ScribingAScrollFromBundledSpells(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	spell := func(id string) content.Spell {
		f, err := src.Lookup(ctx, content.NewKey(content.KindSpell, content.MustID(id)))
		if err != nil {
			t.Fatal(err)
		}
		return f.(content.LookupFound).Entry.(content.Spell)
	}
	fireball, revivify, fireBolt := spell("fireball"), spell("revivify"), spell("fire_bolt")

	// Qualification: Arcana, Expertise in it, or the supplies; either suffices.
	arcana := character.MustSkillProficiencies([]content.Skill{content.SkillArcana}, nil)
	expert := character.MustSkillProficiencies([]content.Skill{content.SkillArcana}, []content.Skill{content.SkillArcana})
	none := character.MustSkillProficiencies([]content.Skill{content.SkillHistory}, nil)
	supplies := character.NewToolProficiencies("calligraphers_supplies")
	for _, c := range []struct {
		name   string
		skills character.SkillProficiencies
		tools  character.ToolProficiencies
		want   bool
	}{
		{"Arcana", arcana, character.ToolProficiencies{}, true}, {"Expertise in Arcana", expert, character.ToolProficiencies{}, true},
		{"Calligrapher's Supplies", none, supplies, true}, {"both", arcana, supplies, true},
		{"neither", none, character.NewToolProficiencies("thieves_tools"), false},
	} {
		_, got := character.QualifyScribe(c.skills, c.tools).(character.ScribeQualified)
		if got != c.want {
			t.Errorf("%s: qualified = %v, want %v", c.name, got, c.want)
		}
	}

	scribe := character.Stats(character.CasterWizard, grpD3Score(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	table := character.MustScrollScribing(grpD3ScrollRows())
	qualified := character.QualifyScribe(arcana, character.ToolProficiencies{})
	begin := func(sp content.Spell, q character.ScribeQualification, tb character.ScrollScribing, level int) character.ScribingBeginResult {
		return character.BeginScribing(character.ScribingStart{Qualification: q, Spell: sp, Table: tb, ScribeLevel: character.MustLevel(max(level, 1)),
			SaveDC: scribe.SaveDC(), AttackBonus: scribe.AttackBonus()})
	}
	if r, ok := begin(fireball, character.QualifyScribe(none, character.ToolProficiencies{}), table, 5).(character.ScribingRejected); !ok || r.Reason() != character.ScribingRejectionNotQualified {
		t.Errorf("an unqualified scribe = %#v", r)
	}
	if r, ok := begin(fireball, qualified, character.ScrollScribing{}, 5).(character.ScribingRejected); !ok || r.Reason() != character.ScribingRejectionTableMissing {
		t.Errorf("no table = %#v", r)
	}
	if r, ok := character.BeginScribing(character.ScribingStart{Qualification: qualified, Spell: fireball, Table: table, SaveDC: scribe.SaveDC(), AttackBonus: scribe.AttackBonus()}).(character.ScribingRejected); !ok || r.Reason() != character.ScribingRejectionInvalidStart {
		t.Errorf("a zero scribe level = %#v", r)
	}

	// Fireball (level 3): 5 days and 150 GP; the spell must be prepared and the materials at hand.
	started, ok := begin(fireball, qualified, table, 5).(character.ScribingBegun)
	if !ok {
		t.Fatal("a qualified scribe could not begin")
	}
	job := started.Job()
	if job.Spell() != "fireball" || job.RequiredDays() != 5 || job.DaysWorked() != 0 || job.DaysRemaining() != 5 || job.Cost() != grpD3GP(150) {
		t.Fatalf("job = %v, %d days, %d worked, %v", job.Spell(), job.RequiredDays(), job.DaysWorked(), job.Cost())
	}
	prepared := character.NewPreparedSpellSet("fireball", "magic_missile")
	if r, ok := job.WorkDay(character.NewPreparedSpellSet("magic_missile"), true).(character.ScribingDayRejected); !ok || !slices.Equal(r.Reasons(), []character.ScribingDayReason{character.ScribingReasonSpellNotPrepared}) {
		t.Errorf("spell not prepared = %#v", r)
	}
	if r, ok := job.WorkDay(character.NewPreparedSpellSet(), false).(character.ScribingDayRejected); !ok ||
		!slices.Equal(r.Reasons(), []character.ScribingDayReason{character.ScribingReasonSpellNotPrepared, character.ScribingReasonMaterialsNotAtHand}) {
		t.Errorf("neither prepared nor materials = %#v", r)
	}
	if r, ok := job.WorkDay(prepared, false).(character.ScribingDayRejected); !ok || !slices.Equal(r.Reasons(), []character.ScribingDayReason{character.ScribingReasonMaterialsNotAtHand}) {
		t.Errorf("no materials = %#v", r)
	}
	if job.DaysWorked() != 0 {
		t.Error("a rejected day must leave the job unchanged")
	}
	// Days need not be consecutive: reject one between the accepted ones; progress is kept.
	for day := 1; day <= 4; day++ {
		w, ok := job.WorkDay(prepared, true).(character.ScribingDayWorked)
		if !ok || w.DaysRemaining() != 5-day {
			t.Fatalf("day %d = %#v", day, job.WorkDay(prepared, true))
		}
		job = w.Job()
		if _, ok := job.WorkDay(prepared, false).(character.ScribingDayRejected); !ok {
			t.Fatalf("day %d: a day without materials counted", day)
		}
	}
	done, ok := job.WorkDay(prepared, true).(character.ScribingCompleted)
	if !ok {
		t.Fatalf("the fifth day = %#v", job.WorkDay(prepared, true))
	}
	scroll := done.Scroll()
	if scroll.Spell() != "fireball" || scroll.SpellLevel() != grpD3Level(3) || scroll.SaveDC() != scribe.SaveDC() || scroll.AttackBonus() != scribe.AttackBonus() || scroll.MaterialsConsumed() {
		t.Errorf("Fireball scroll = %v level %v DC %v bonus %v consumed %v", scroll.Spell(), scroll.SpellLevel(), scroll.SaveDC(), scroll.AttackBonus(), scroll.MaterialsConsumed())
	}
	if _, has := scroll.CasterLevel(); has {
		t.Error("a level 3 scroll must carry no caster level")
	}
	if scribe.SaveDC().Value() != 14 {
		t.Errorf("the scribe's save DC = %d, want 8 + 3 + 3 = 14", scribe.SaveDC().Value())
	}
	// A finished job takes no more days (and the finished job itself is unchanged).
	finished := grpD3Must(character.NewScrollScribingJob(5, 5, fireball, character.MustLevel(5), scribe.SaveDC(), scribe.AttackBonus(), grpD3GP(150)))
	if r, ok := finished.WorkDay(prepared, true).(character.ScribingDayRejected); !ok || !slices.Equal(r.Reasons(), []character.ScribingDayReason{character.ScribingReasonAlreadyCompleted}) {
		t.Errorf("a day after completion = %#v", r)
	}

	// Revivify's diamond is consumed on completion; the flag is the spell's.
	rv := begin(revivify, qualified, table, 9).(character.ScribingBegun).Job()
	for i := 0; i < rv.RequiredDays()-1; i++ {
		rv = rv.WorkDay(character.NewPreparedSpellSet("revivify"), true).(character.ScribingDayWorked).Job()
	}
	if c, ok := rv.WorkDay(character.NewPreparedSpellSet("revivify"), true).(character.ScribingCompleted); !ok || !c.Scroll().MaterialsConsumed() {
		t.Errorf("Revivify completion = %#v", c)
	}

	// A cantrip: one day, no materials needed (the flag is ignored), and the scroll keeps the scribe's level.
	cj := begin(fireBolt, qualified, table, 5).(character.ScribingBegun).Job()
	if cj.RequiredDays() != 1 || cj.Cost() != grpD3GP(15) {
		t.Fatalf("cantrip job = %d days, %v", cj.RequiredDays(), cj.Cost())
	}
	cd, ok := cj.WorkDay(character.NewPreparedSpellSet("fire_bolt"), false).(character.ScribingCompleted)
	if !ok {
		t.Fatalf("a cantrip with no materials = %#v", cj.WorkDay(character.NewPreparedSpellSet("fire_bolt"), false))
	}
	if lvl, has := cd.Scroll().CasterLevel(); !has || lvl != character.MustLevel(5) || cd.Scroll().MaterialsConsumed() {
		t.Errorf("cantrip scroll caster level %v/%v consumed %v, want level 5 and nothing consumed", lvl, has, cd.Scroll().MaterialsConsumed())
	}

	// The constructors refuse impossible progress.
	_, err := character.NewScrollScribingJob(0, 0, fireball, character.MustLevel(5), scribe.SaveDC(), scribe.AttackBonus(), grpD3GP(1))
	grpD3WantInvalid(t, err, "zero required days")
	_, err = character.NewScrollScribingJob(5, 6, fireball, character.MustLevel(5), scribe.SaveDC(), scribe.AttackBonus(), grpD3GP(1))
	grpD3WantInvalid(t, err, "more days worked than required")
	_, err = character.NewScrollScribingJob(5, -1, fireball, character.MustLevel(5), scribe.SaveDC(), scribe.AttackBonus(), grpD3GP(1))
	grpD3WantInvalid(t, err, "negative days worked")
}

// The prepared set a scribe passes to WorkDay can come from the character's
// spellcasting: prepared, always prepared, known cantrips and granted cantrips
// are included, a spellbook spell only when prepared.
// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-042, FR-048 - the union across classes, deduplicated and ordered, a spellbook spell counts only when prepared, the set is copied.
// Path: happy; unhappy (rule interaction, empty and minimal, immutability).
func TestGrpD3PreparedSpellSetFromAWizardsSpellcasting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	spells := map[content.SpellID]content.Spell{}
	for _, id := range []string{"fireball", "fire_bolt", "magic_missile", "revivify"} {
		f, err := src.Lookup(ctx, content.NewKey(content.KindSpell, content.MustID(id)))
		if err != nil {
			t.Fatal(err)
		}
		sp := f.(content.LookupFound).Entry.(content.Spell)
		spells[sp.ID()] = sp
	}
	wizard := character.MustClassSpellcasting(character.ClassSpellcastingSetup{
		Class: character.CasterWizard, Cantrips: []content.SpellID{"fire_bolt"}, Prepared: []content.SpellID{"fireball", "magic_missile"},
		Spellbook: character.NewSpellbook([]content.SpellID{"fireball", "magic_missile", "revivify"}),
	}, 5, spells)
	set := character.PreparedSpellSetFromSpellcasting([]character.ClassSpellcasting{wizard})
	for _, id := range []content.SpellID{"fire_bolt", "fireball", "magic_missile"} {
		if !set.Contains(id) {
			t.Errorf("the prepared set lacks %s", id)
		}
	}
	if set.Contains("revivify") {
		t.Error("a spellbook spell that is not prepared must not count")
	}
	if got := set.IDs(); !slices.IsSorted(got) || len(got) != 3 {
		t.Errorf("IDs = %v, want three, in order", got)
	}
	ids := set.IDs()
	ids[0] = "changed"
	if !set.Equal(character.NewPreparedSpellSet("fire_bolt", "fireball", "magic_missile")) {
		t.Error("mutating IDs() changed the set")
	}
	if empty := character.PreparedSpellSetFromSpellcasting(nil); len(empty.IDs()) != 0 {
		t.Errorf("no classes = %v, want the empty set", empty.IDs())
	}
}
