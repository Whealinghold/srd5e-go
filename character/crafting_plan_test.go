package character_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func planTools() []content.Tool {
	return append(craftTools(), herbalismKit(), gamingSet(), variantTool("gaming_set_dice", "gaming_set"))
}

func craftPlateArmor(t *testing.T) content.EquipmentEntry {
	t.Helper()
	return armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500))
}

func craftPotion() content.EquipmentEntry {
	override := must(content.NewCraftingOverride(gp(25), 1, idsOf("herbalism_kit")))
	return must(content.NewMagicGear("potion_x", "Potion", noSource, price(gp(50)), content.NoWeight{}, "", "Text.", override))
}

func craftPlan(t *testing.T, item content.EquipmentEntry, party character.CraftingParty) character.ItemCraftingPlan {
	t.Helper()
	return character.PlanItemCrafting(character.ItemCraftingInput{Item: item, Tools: planTools(), Scribing: character.MustScrollScribing(scrollRows()), Party: party})
}

func wantCraftReady(t *testing.T, p character.ItemCraftingPlan) character.ItemCraftingReady {
	t.Helper()
	r, ok := p.(character.ItemCraftingReady)
	if !ok {
		t.Fatalf("plan = %#v, want ItemCraftingReady", p)
	}
	return r
}

func wantCraftBlocked(t *testing.T, p character.ItemCraftingPlan) []character.ToolCraftingShortfall {
	t.Helper()
	b, ok := p.(character.ItemCraftingBlocked)
	if !ok {
		t.Fatalf("plan = %#v, want ItemCraftingBlocked", p)
	}
	return b.Shortfalls()
}

func workHours(n, d int) character.WorkHours { return must(character.NewWorkHours(n, d)) }

// Requirement: per case tool-proficiency-and-crafting.md FR-024, FR-026, FR-030,
// FR-031 (SRD 5.2.1, "Crafting Nonmagical Items" and "Time", p. 103) - the ready
// plans of "Cases: crafting plans": requirements unchanged, usable tools by id,
// the worker count and the exact work hours.
// Path: per case (happy; unhappy rule interaction).
func TestPlanItemCraftingReady(t *testing.T) {
	t.Parallel()
	smith := craftWorker("c", "smiths_tools")
	cases := []struct {
		name      string
		item      content.EquipmentEntry
		party     character.CraftingParty
		wantRaw   content.Cost
		wantDays  int
		wantTools []string
		wantHours character.WorkHours
		wantBasis character.CraftingBasis
	}{
		{"Plate Armor, solo", craftPlateArmor(t), character.MustCraftingParty(smith), cp(75000), 150, []string{"smiths_tools"}, workHours(1200, 1), character.CraftingGenericRule},
		{"Plate Armor with a helper", craftPlateArmor(t), character.MustCraftingParty(smith, craftWorker("h", "smiths_tools")), cp(75000), 150, []string{"smiths_tools"}, workHours(1200, 2), character.CraftingGenericRule},
		{"Heavy Crossbow, crafter and helper", weaponOf("hc_x", content.WeaponRanged, gp(50)), character.MustCraftingParty(craftWorker("c", "woodcarvers_tools"), craftWorker("h", "woodcarvers_tools")), cp(2500), 5, []string{"woodcarvers_tools"}, workHours(40, 2), character.CraftingGenericRule},
		{"Club, a tool both share", weaponOf("club_x", content.WeaponMelee, gp(1)), character.MustCraftingParty(craftWorker("c", "carpenters_tools", "woodcarvers_tools"), craftWorker("h", "woodcarvers_tools")), cp(50), 1, []string{"woodcarvers_tools"}, workHours(8, 2), character.CraftingGenericRule},
		{"Club, every tool shared", weaponOf("club_x", content.WeaponMelee, gp(1)), character.MustCraftingParty(craftWorker("c", "woodcarvers_tools", "carpenters_tools")), cp(50), 1, []string{"carpenters_tools", "woodcarvers_tools"}, workHours(8, 1), character.CraftingGenericRule},
		{"Potion of Healing", craftPotion(), character.MustCraftingParty(craftWorker("c", "herbalism_kit")), gp(25), 1, []string{"herbalism_kit"}, workHours(8, 1), character.CraftingItemOverride},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := wantCraftReady(t, craftPlan(t, c.item, c.party))
			req := r.Requirements()
			if req.RawMaterials() != c.wantRaw || req.Days() != c.wantDays || req.Basis() != c.wantBasis {
				t.Errorf("requirements = %v, %d days, %v", req.RawMaterials(), req.Days(), req.Basis())
			}
			if !slices.Equal(r.UsableTools(), idsOf(c.wantTools...)) || r.WorkHours() != c.wantHours || r.Workers() != 1+len(c.party.Helpers()) {
				t.Errorf("tools %v, hours %v, workers %d", r.UsableTools(), r.WorkHours(), r.Workers())
			}
			got := r.UsableTools()
			got[0] = "changed"
			if r.UsableTools()[0] == "changed" {
				t.Error("UsableTools is not a copy")
			}
		})
	}
}

// describe renders a shortfall as role, id and alternatives for comparison; the
// shortfall types have no public constructors.
func describeShortfall(s character.ToolCraftingShortfall) string {
	switch v := s.(type) {
	case character.ToolShortfallWorker:
		return fmt.Sprintf("worker %s %s %v", v.Role(), v.WorkerID(), v.Alternatives())
	case character.ToolShortfallNoSharedTool:
		return fmt.Sprintf("noShared %v", v.Alternatives())
	case character.ToolShortfallTooManyHelpers:
		return fmt.Sprintf("tooMany %d %d", v.Count(), v.Limit())
	case character.ToolShortfallHelpersNotAllowed:
		return fmt.Sprintf("notAllowed %s", v.Basis())
	}
	return "unknown"
}

func describeShortfalls(ss []character.ToolCraftingShortfall) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = describeShortfall(s)
	}
	return out
}

// Requirement: per case tool-proficiency-and-crafting.md FR-026 to FR-028,
// FR-032, FR-033 (SRD 5.2.1, "Tools", "Time" and "Brewing Potions of Healing",
// p. 103) - every shortfall at once and in order: tools (crafter first, then
// helpers in party order, or no shared tool), too many helpers, helpers not
// allowed; the allowance raises the cap.
// Path: per case (unhappy invalid input, boundary, rule interaction).
func TestPlanItemCraftingShortfalls(t *testing.T) {
	t.Parallel()
	smith := "[smiths_tools]"
	club := weaponOf("club_x", content.WeaponMelee, gp(1))
	clubAlts := "[carpenters_tools woodcarvers_tools]"
	cases := []struct {
		name      string
		item      content.EquipmentEntry
		party     character.CraftingParty
		allowance character.AssistantAllowance
		want      []string
	}{
		{"the helper lacks the tool", craftPlateArmor(t), character.MustCraftingParty(craftWorker("c", "smiths_tools"), craftWorker("h")), 0, []string{"worker helper h " + smith}},
		{"nobody has the tool", craftPlateArmor(t), character.MustCraftingParty(craftWorker("c"), craftWorker("h")), 0, []string{"worker crafter c " + smith, "worker helper h " + smith}},
		{"no shared tool", club, character.MustCraftingParty(craftWorker("c", "carpenters_tools"), craftWorker("h", "woodcarvers_tools")), 0, []string{"noShared " + clubAlts}},
		{"a worker with none hides the shared-tool shortfall", club, character.MustCraftingParty(craftWorker("c", "carpenters_tools"), craftWorker("h", "woodcarvers_tools"), craftWorker("i")), must(character.NewAssistantAllowance(2)), []string{"worker helper i " + clubAlts}},
		{"all at once", craftPlateArmor(t), character.MustCraftingParty(craftWorker("c"), craftWorker("h1"), craftWorker("h2")), 0, []string{"worker crafter c " + smith, "worker helper h1 " + smith, "worker helper h2 " + smith, "tooMany 2 1"}},
		{"two helpers over the standard cap", craftPlateArmor(t), character.MustCraftingParty(craftWorker("c", "smiths_tools"), craftWorker("h1", "smiths_tools"), craftWorker("h2", "smiths_tools")), 0, []string{"tooMany 2 1"}},
		{"one helper at the cap", craftPlateArmor(t), character.MustCraftingParty(craftWorker("c", "smiths_tools"), craftWorker("h1", "smiths_tools")), 0, nil},
		{"three helpers over an allowance of two", craftPlateArmor(t), character.MustCraftingParty(craftWorker("c", "smiths_tools"), craftWorker("h1", "smiths_tools"), craftWorker("h2", "smiths_tools"), craftWorker("h3", "smiths_tools")), must(character.NewAssistantAllowance(2)), []string{"tooMany 3 2"}},
		{"Potion of Healing with a helper", craftPotion(), character.MustCraftingParty(craftWorker("c", "herbalism_kit"), craftWorker("h", "herbalism_kit")), 0, []string{"notAllowed itemOverride"}},
		{"Potion of Healing, crafter without the kit", craftPotion(), character.MustCraftingParty(craftWorker("c")), 0, []string{"worker crafter c [herbalism_kit]"}},
		{"Potion of Healing, two helpers: cap then not allowed", craftPotion(), character.MustCraftingParty(craftWorker("c", "herbalism_kit"), craftWorker("h1", "herbalism_kit"), craftWorker("h2", "herbalism_kit")), 0, []string{"tooMany 2 1", "notAllowed itemOverride"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := character.PlanItemCrafting(character.ItemCraftingInput{Item: c.item, Tools: planTools(), Party: c.party, Allowance: c.allowance})
			if c.want == nil {
				wantCraftReady(t, p)
				return
			}
			if got := describeShortfalls(wantCraftBlocked(t, p)); !slices.Equal(got, c.want) {
				t.Errorf("shortfalls = %q, want %q", got, c.want)
			}
		})
	}
	t.Run("an allowance of two makes three workers ready with exact hours", func(t *testing.T) {
		t.Parallel()
		party := character.MustCraftingParty(craftWorker("c", "smiths_tools"), craftWorker("h1", "smiths_tools"), craftWorker("h2", "smiths_tools"))
		p := character.PlanItemCrafting(character.ItemCraftingInput{Item: craftPlateArmor(t), Tools: planTools(), Party: party, Allowance: must(character.NewAssistantAllowance(2))})
		if r := wantCraftReady(t, p); r.WorkHours() != workHours(1200, 3) || r.Workers() != 3 {
			t.Errorf("hours %v workers %d, want 1200/3 and 3", r.WorkHours(), r.Workers())
		}
	})
	t.Run("the shortfall lists are copies", func(t *testing.T) {
		t.Parallel()
		b := character.PlanItemCrafting(character.ItemCraftingInput{Item: craftPlateArmor(t), Tools: planTools(), Party: character.MustCraftingParty(craftWorker("c"))}).(character.ItemCraftingBlocked)
		b.Shortfalls()[0] = character.ToolShortfallTooManyHelpers{}
		w, ok := b.Shortfalls()[0].(character.ToolShortfallWorker)
		if !ok {
			t.Fatalf("shortfall = %#v, want the stored worker shortfall", b.Shortfalls()[0])
		}
		w.Alternatives()[0] = "changed"
		if w.Alternatives()[0] != "smiths_tools" {
			t.Error("Alternatives aliases the shortfall's own slice")
		}
		if b.Requirements().Days() != 150 {
			t.Errorf("blocked requirements days = %d", b.Requirements().Days())
		}
	})
	t.Run("the zero party is blocked on the crafter", func(t *testing.T) {
		t.Parallel()
		got := describeShortfalls(wantCraftBlocked(t, craftPlan(t, craftPlateArmor(t), character.CraftingParty{})))
		if len(got) != 1 || !strings.HasPrefix(got[0], "worker crafter") {
			t.Errorf("shortfalls = %q", got)
		}
	})
	t.Run("a nil shortfall is not described", func(t *testing.T) {
		t.Parallel()
		if describeShortfall(nil) != "unknown" {
			t.Error("a nil shortfall must not be described")
		}
	})
}

// Requirement: per case tool-proficiency-and-crafting.md FR-029 (SRD 5.2.1,
// "Tools", Crafting, p. 103) - the Game Master's tool is the only alternative for
// an item that lists none, and is ignored or rejected otherwise.
// Path: per case (unhappy missing content).
func TestPlanItemCraftingGMTool(t *testing.T) {
	t.Parallel()
	smith := character.MustCraftingParty(craftWorker("c", "smiths_tools"))
	scribe := must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(50)), "", "", 1))
	cases := []struct {
		name  string
		item  content.EquipmentEntry
		gm    content.ItemID
		party character.CraftingParty
		want  string
	}{
		{"Shield, no assigned tool", shieldOf(t), "", smith, "needsGMTool"},
		{"Shield, assigned Smith's Tools", shieldOf(t), "smiths_tools", smith, "ready [smiths_tools]"},
		{"Shield, assigned family", shieldOf(t), "gaming_set", smith, "invalid gaming_set"},
		{"Shield, assigned id not in the tools", shieldOf(t), "unknown_tools", smith, "invalid unknown_tools"},
		{"Shield, assigned variant", shieldOf(t), "gaming_set_dice", character.MustCraftingParty(craftWorker("c", "gaming_set_dice")), "ready [gaming_set_dice]"},
		{"Plate Armor ignores the assigned tool", craftPlateArmor(t), "woodcarvers_tools", smith, "ready [smiths_tools]"},
		{"Potion of Healing keeps its override tool", craftPotion(), "smiths_tools", character.MustCraftingParty(craftWorker("c", "herbalism_kit")), "ready [herbalism_kit]"},
		{"Spell Scroll goes to scribing", scribe, "smiths_tools", smith, "useScribing"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := character.PlanItemCrafting(character.ItemCraftingInput{Item: c.item, Tools: planTools(), Scribing: character.MustScrollScribing(scrollRows()), Party: c.party, GMAssignedTool: c.gm})
			var got string
			switch v := p.(type) {
			case character.ItemCraftingReady:
				got = fmt.Sprintf("ready %v", v.UsableTools())
			case character.ItemCraftingNeedsGMTool:
				got = "needsGMTool"
			case character.ItemCraftingGMToolInvalid:
				got = "invalid " + string(v.ToolID())
			case character.ItemCraftingUseScribing:
				got = "useScribing"
			default:
				got = fmt.Sprintf("%#v", p)
			}
			if got != c.want {
				t.Errorf("plan = %s, want %s", got, c.want)
			}
		})
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-024, FR-025 (SRD
// 5.2.1, "Crafting Nonmagical Items", p. 103; equipment-rules.md FR-040) - a
// NotCraftable outcome and an entry the rules cannot read pass through with the
// party not evaluated, and the requirements are taken as they come (no
// recomputation).
// Path: per case (unhappy rule interaction, invalid input).
func TestPlanItemCraftingPassThrough(t *testing.T) {
	t.Parallel()
	zero := character.CraftingParty{}
	huge := must(content.NewPlainGear("huge_x", "huge_x", noSource, price(content.MustCost(content.MaxCostAmount, content.CoinPP, 0)), lbs(1), "", "Text."))
	cases := []struct {
		name string
		item content.EquipmentEntry
		want character.NotCraftableReason
	}{
		{"Horse, Draft", mountOf(t, "horse_x", 540), character.NotCraftableNotAnItem},
		{"a nil item", nil, character.NotCraftableNoCraftingRule},
		{"a price too large to halve", huge, character.NotCraftableNoCraftingRule},
		{"Gaming Set family", gamingSet(), character.NotCraftablePriceVaries},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: c.item, Tools: planTools(), Party: zero}).(character.ItemCraftingNotCraftable)
			if !ok || got.Reason() != c.want {
				t.Errorf("plan = %#v, want NotCraftable(%v)", got, c.want)
			}
		})
	}
	t.Run("a Spell Scroll goes to scribing", func(t *testing.T) {
		t.Parallel()
		scroll := must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(50)), "", "", 1))
		if _, ok := craftPlan(t, scroll, zero).(character.ItemCraftingUseScribing); !ok {
			t.Error("want ItemCraftingUseScribing")
		}
	})
	t.Run("a Spell Scroll without a scribing table is not craftable", func(t *testing.T) {
		t.Parallel()
		scroll := must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(50)), "", "", 1))
		got, ok := character.PlanItemCrafting(character.ItemCraftingInput{Item: scroll, Tools: planTools()}).(character.ItemCraftingNotCraftable)
		if !ok || got.Reason() != character.NotCraftableScribingTableMissing {
			t.Errorf("plan = %#v", got)
		}
	})
	t.Run("an override with no tool needs none", func(t *testing.T) {
		t.Parallel()
		synth := must(content.NewCraftingOverride(gp(3), 2, nil))
		item := must(content.NewMagicGear("synth_x", "Synth", noSource, price(gp(50)), content.NoWeight{}, "", "Text.", synth))
		r := wantCraftReady(t, craftPlan(t, item, character.MustCraftingParty(craftWorker("c"))))
		if len(r.UsableTools()) != 0 || r.WorkHours() != workHours(16, 1) {
			t.Errorf("tools %v hours %v", r.UsableTools(), r.WorkHours())
		}
	})
}

// Requirement: tool-proficiency-and-crafting.md FR-030, FR-031, FR-048 (SRD
// 5.2.1, "Time", p. 103) - the work hours are an exact fraction, never reduced,
// and the plan holds the requirements unchanged: a synthetic override with odd
// figures passes through.
// Path: unhappy (rule interaction).
func TestPlanItemCraftingHoursAreExact(t *testing.T) {
	t.Parallel()
	odd := must(content.NewCraftingOverride(gp(7), 3, idsOf("smiths_tools")))
	item := must(content.NewMagicGear("odd_x", "Odd", noSource, price(gp(50)), content.NoWeight{}, "", "Text.", odd))
	r := wantCraftReady(t, craftPlan(t, item, character.MustCraftingParty(craftWorker("c", "smiths_tools"))))
	if h := r.WorkHours(); h.Numerator() != 24 || h.Denominator() != 1 || r.Requirements().RawMaterials() != gp(7) || r.Requirements().Days() != 3 {
		t.Errorf("plan = %v hours, %v, %d days", r.WorkHours(), r.Requirements().RawMaterials(), r.Requirements().Days())
	}
}
