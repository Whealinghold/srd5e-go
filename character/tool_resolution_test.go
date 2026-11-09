package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: tool-proficiency-and-crafting.md FR-004, FR-006, FR-018 (SRD
// 5.2.1, "Tools", Artisan's Tools and Other Tools, pp. 93-94) - the "Cases: tool
// facts" rows: a standalone tool's facts are its own, a variant's are its
// family's while its proficiency id stays its own.
// Path: per case (happy; unhappy rule interaction).
func TestResolveToolFacts(t *testing.T) {
	t.Parallel()
	cook := standaloneTool("cooks_utensils", ability.Wisdom, action("Improve flavor", 10), action("Detect spoiled or poisoned food", 15))
	lute := variantTool("musical_instrument_lute", "musical_instrument")
	instruments := familyTool("musical_instrument", ability.Charisma, []string{"musical_instrument_lute"}, action("Play a known tune", 10), action("Improvise a song", 15))
	catalogue := []content.Tool{gamingSet(), instruments}
	cases := []struct {
		name    string
		tool    content.Tool
		wantID  content.ItemID
		wantAb  ability.Ability
		wantDCs []int
	}{
		{"Thieves' Tools", thievesTools(), "thieves_tools", ability.Dexterity, []int{15, 15}},
		{"Herbalism Kit", herbalismKit(), "herbalism_kit", ability.Intelligence, []int{10}},
		{"Carpenter's Tools", standaloneTool("carpenters_tools", ability.Strength, action("Seal or pry open", 20)), "carpenters_tools", ability.Strength, []int{20}},
		{"Cook's Utensils, several actions", cook, "cooks_utensils", ability.Wisdom, []int{10, 15}},
		{"gaming_set_dice takes its family's facts", variantTool("gaming_set_dice", "gaming_set"), "gaming_set_dice", ability.Wisdom, []int{10, 20}},
		{"musical_instrument_lute takes its family's facts", lute, "musical_instrument_lute", ability.Charisma, []int{10, 15}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, ok := character.ResolveTool(c.tool, catalogue).(character.ToolResolved)
			if !ok {
				t.Fatalf("ResolveTool = %#v, want ToolResolved", character.ResolveTool(c.tool, catalogue))
			}
			r := res.Tool()
			var dcs []int
			for _, a := range r.Utilize() {
				dcs = append(dcs, a.DC())
			}
			if r.ProficiencyID() != c.wantID || r.Ability() != c.wantAb || !slices.Equal(dcs, c.wantDCs) || r.IsZero() {
				t.Errorf("resolved = %v %v %v, want %v %v %v", r.ProficiencyID(), r.Ability(), dcs, c.wantID, c.wantAb, c.wantDCs)
			}
		})
	}
	if g := resolvedTool(variantTool("gaming_set_dice", "gaming_set"), gamingSet()).Group(); g != content.ToolGroupOther {
		t.Errorf("variant group = %v, want the family's Other Tools", g)
	}
	if g := resolvedTool(thievesTools()).Group(); g != content.ToolGroupArtisans {
		t.Errorf("standalone group = %v", g)
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-005 (SRD 5.2.1, "Gaming Set",
// "Musical Instrument", p. 94) - a family is not a proficiency, and a variant
// whose family is absent, not a family or ambiguous is missing, whatever the
// catalogue order; no error and no panic.
// Path: per case (unhappy missing content).
func TestResolveToolFamilyProblems(t *testing.T) {
	t.Parallel()
	dice := variantTool("gaming_set_dice", "gaming_set")
	twin := familyTool("gaming_set", ability.Charisma, []string{"gaming_set_dice"}, action("Other", 12))
	cases := []struct {
		name      string
		tool      content.Tool
		catalogue []content.Tool
		want      character.ToolResolution
	}{
		{"family alone", gamingSet(), nil, character.ToolFamilyNeedsVariant{}},
		{"variant with an empty catalogue", dice, nil, character.ToolFamilyMissing{}},
		{"variant naming another variant", dice, []content.Tool{variantTool("gaming_set", "x")}, character.ToolFamilyMissing{}},
		{"variant naming a standalone tool", dice, []content.Tool{standaloneTool("gaming_set", ability.Wisdom, action("A", 10))}, character.ToolFamilyMissing{}},
		{"two families, one order", dice, []content.Tool{gamingSet(), twin}, character.ToolFamilyMissing{}},
		{"two families, the other order", dice, []content.Tool{twin, gamingSet()}, character.ToolFamilyMissing{}},
		{"a family and a standalone tool with one id", dice, []content.Tool{gamingSet(), standaloneTool("gaming_set", ability.Wisdom, action("A", 10))}, character.ToolFamilyMissing{}},
		{"nil entries in the catalogue are skipped", dice, []content.Tool{nil, gamingSet()}, character.ToolResolved{}},
		{"nil tool", nil, []content.Tool{gamingSet()}, character.ToolFamilyMissing{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.ResolveTool(c.tool, c.catalogue)
			switch want := c.want.(type) {
			case character.ToolFamilyNeedsVariant:
				g, ok := got.(character.ToolFamilyNeedsVariant)
				if !ok || g.FamilyID() != "gaming_set" {
					t.Errorf("got %#v, want ToolFamilyNeedsVariant(gaming_set)", got)
				}
			case character.ToolFamilyMissing:
				g, ok := got.(character.ToolFamilyMissing)
				if !ok {
					t.Fatalf("got %#v, want ToolFamilyMissing", got)
				}
				if c.tool != nil && (g.VariantID() != "gaming_set_dice" || g.FamilyID() != "gaming_set") {
					t.Errorf("missing ids = %q, %q", g.VariantID(), g.FamilyID())
				}
				if c.tool == nil && (g.VariantID() != "" || g.FamilyID() != "") {
					t.Errorf("nil tool should give empty ids, got %q, %q", g.VariantID(), g.FamilyID())
				}
			case character.ToolResolved:
				if _, ok := got.(character.ToolResolved); !ok {
					t.Errorf("got %#v, want ToolResolved", got)
				}
			default:
				t.Fatalf("unexpected want %T", want)
			}
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-048 - the Utilize list of a
// resolved tool is a copy on read.
// Path: unhappy (immutability).
func TestResolvedToolUtilizeIsACopy(t *testing.T) {
	t.Parallel()
	r := resolvedTool(thievesTools())
	got := r.Utilize()
	got[0] = action("Changed", 1)
	if again := r.Utilize(); again[0].Description() != "Pick a lock" {
		t.Errorf("Utilize after mutating a returned slice = %v", again[0].Description())
	}
	if !(character.ResolvedTool{}).IsZero() {
		t.Error("the zero ResolvedTool must report IsZero")
	}
}
