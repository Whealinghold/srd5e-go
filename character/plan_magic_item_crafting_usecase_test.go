package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func magicEntry(t *testing.T, id string, category content.MagicItemCategory, rarity content.RaritySpec) content.MagicItemEntry {
	t.Helper()
	return must(content.NewMagicItemEntry(mid(id), id, noSource, content.MagicItemFields{Category: category, Rarity: rarity, Attunement: content.NoAttunement{},
		Description: []content.TextBlock{content.MustTextParagraph("Text.")}}))
}

func single(r content.ItemRarity) content.RaritySpec { return must(content.NewSingleRarity(r)) }

func facts(t *testing.T, id string, f content.MagicItemFactsFields) content.MagicItemFacts {
	t.Helper()
	return must(content.NewMagicItemFacts(mid(id), f))
}

func planUse(t *testing.T, src character.MagicItemCraftingSource, opts ...character.Option) character.PlanMagicItemCrafting {
	t.Helper()
	return must(character.NewPlanMagicItemCrafting(src, opts...))
}

func tools() []content.Entry {
	var out []content.Entry
	for _, tl := range craftTools() {
		out = append(out, tl)
	}
	return out
}

func quals(t *testing.T, arcana bool, tool string, spells ...content.SpellID) []character.CrafterQualification {
	t.Helper()
	return []character.CrafterQualification{must(character.NewCrafterQualification("crafter", arcana, idsOf(tool), spells))}
}

func soloTeam(t *testing.T) character.CraftingTeam {
	t.Helper()
	return must(character.NewCraftingTeam("crafter", nil))
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-074, FR-077 to FR-080, FR-082, FR-084, FR-087, FR-107, FR-121 and tables "Crafting",
// "Crafting tools" - the plan resolves the rarity, looks the item and what it needs up, reports every shortfall or the requirements).
// Path: per case.
func TestPlanMagicItemCrafting(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Crafting Magic Items", "Time and Cost", pp. 206-207`
	ctx := context.Background()
	ring := magicEntry(t, "ring_x", content.MagicItemRing, single(content.RarityRare))
	plate := armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500))
	armor := magicEntry(t, "armor_x", content.MagicItemArmor, single(content.RarityRare))
	spell := spellMap(t, spellSpec{id: "fireball", level: 3, classes: []string{"wizard"}})["fireball"]
	wand := magicEntry(t, "wand_x", content.MagicItemWand, single(content.RarityUncommon))
	artifact := magicEntry(t, "artifact_x", content.MagicItemWondrousItem, single(content.RarityArtifact))
	belt := magicEntry(t, "belt_x", content.MagicItemWondrousItem, content.RarityVaries{})
	source := func(extra ...content.Entry) *contenttest.FakeSource {
		return contenttest.NewFakeSource(append(append([]content.Entry{ring, armor, wand, artifact, belt, plate, spell}, tools()...), extra...)...)
	}
	armorFacts := func() content.MagicItemFacts {
		return facts(t, "armor_x", content.MagicItemFactsFields{BaseItem: must(content.NewSpecificBaseItems([]content.ItemID{"plate_x"})), IncorporatedItem: must(content.NewIncorporatedItem("plate_x", 150000))})
	}
	reqs := func(r character.MagicItemCraftingOutcome) character.MagicItemCraftingRequirements {
		return r.(character.MagicItemCraftingRequirements)
	}
	cases := []struct {
		name, requirement, path string
		src                     *contenttest.FakeSource
		cmd                     character.PlanMagicItemCraftingCommand
		check                   func(t *testing.T, res character.PlanMagicItemCraftingResult)
		queries                 int // 0 means not asserted
	}{
		{"Rare ring, qualified solo crafter", p, "happy", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "ring_x", content.MagicItemFactsFields{}), Team: soloTeam(t), Qualifications: quals(t, true, "jewelers_tools")},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				r := reqs(res.(character.PlanMagicItemCraftingPlanned).Outcome())
				if r.TotalWorkHours() != 400 || r.CostCopperPieces() != 2000*gold || r.WorkHoursPerWorker().Denominator() != 1 {
					t.Errorf("requirements = %+v", r)
				}
			}, 0},
		{"+1 Armor (Plate Armor), pay", p, "happy", source(), character.PlanMagicItemCraftingCommand{Facts: armorFacts(), BaseItem: plate, Team: soloTeam(t), Qualifications: quals(t, true, "smiths_tools"), Incorporated: character.PayPurchaseCost},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if r := reqs(res.(character.PlanMagicItemCraftingPlanned).Outcome()); r.CostCopperPieces() != 3500*gold || r.IncorporatedOutcome() != nil {
					t.Errorf("requirements = %+v", r)
				}
			}, 0},
		{"+1 Armor (Plate Armor), craft the plate", p, "happy", source(), character.PlanMagicItemCraftingCommand{Facts: armorFacts(), BaseItem: plate, Team: soloTeam(t), Qualifications: quals(t, true, "smiths_tools"), Incorporated: character.CraftIncorporatedItem},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				r := reqs(res.(character.PlanMagicItemCraftingPlanned).Outcome())
				nested, ok := r.IncorporatedOutcome().(character.CraftingRequirements)
				if r.CostCopperPieces() != 2000*gold || !ok || nested.RawMaterials().ValueInCopper() != 750*gold {
					t.Errorf("requirements = %+v nested %+v", r, r.IncorporatedOutcome())
				}
			}, 0},
		{"wand with a spell, prepared", p, "happy", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "wand_x", content.MagicItemFactsFields{AllowedSpells: []content.SpellID{"fireball"}}), Team: soloTeam(t), Qualifications: quals(t, true, "woodcarvers_tools", "fireball")},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if r := reqs(res.(character.PlanMagicItemCraftingPlanned).Outcome()); r.TotalWorkHours() != 80 {
					t.Errorf("requirements = %+v", r)
				}
			}, 0},
		{"wand with a spell, not prepared", p, "unhappy (rule interaction)", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "wand_x", content.MagicItemFactsFields{AllowedSpells: []content.SpellID{"fireball"}}), Team: soloTeam(t), Qualifications: quals(t, true, "woodcarvers_tools")},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				s := res.(character.PlanMagicItemCraftingRejected).Shortfalls()
				if len(s) != 1 || s[0].Kind != character.ShortfallSpellNotPrepared || s[0].Spell != "fireball" {
					t.Errorf("shortfalls = %+v", s)
				}
			}, 0},
		{"no Arcana, no tool, no qualification at all", p, "unhappy (invalid input)", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "ring_x", content.MagicItemFactsFields{}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				s := res.(character.PlanMagicItemCraftingRejected).Shortfalls()
				if len(s) != 2 || s[0].Kind != character.ShortfallArcanaProficiency || s[1].Kind != character.ShortfallToolProficiency || !reflect.DeepEqual(s[1].Tools, idsOf("jewelers_tools")) {
					t.Errorf("shortfalls = %+v", s)
				}
			}, 0},
		{"too many assistants", p, "unhappy (boundary)", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "ring_x", content.MagicItemFactsFields{}), Team: must(character.NewCraftingTeam("crafter", []character.CraftingMemberID{"a", "b"})),
			Qualifications: append(quals(t, true, "jewelers_tools"), must(character.NewCrafterQualification("a", true, idsOf("jewelers_tools"), nil)), must(character.NewCrafterQualification("b", true, idsOf("jewelers_tools"), nil)))},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				s := res.(character.PlanMagicItemCraftingRejected).Shortfalls()
				if len(s) != 1 || s[0].Kind != character.ShortfallTooManyAssistants {
					t.Errorf("shortfalls = %+v", s)
				}
			}, 0},
		{"Artifact, no team evaluated", p, "unhappy (missing content)", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "artifact_x", content.MagicItemFactsFields{}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if got := res.(character.PlanMagicItemCraftingPlanned).Outcome(); got != (character.MagicItemNotCraftable{Reason: character.MagicItemNotCraftableArtifact}) {
					t.Errorf("outcome = %#v", got)
				}
			}, 1},
		{"rarity varies, no choice: no further lookup", p, "unhappy (missing content)", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "belt_x", content.MagicItemFactsFields{}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if got := res.(character.PlanMagicItemCraftingRarityUnresolved).Reason(); got != character.RarityChoiceRequired {
					t.Errorf("reason = %v", got)
				}
			}, 1},
		{"rarity varies, chosen", p, "happy", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "belt_x", content.MagicItemFactsFields{}), RarityChoice: character.ChosenRarity(content.RarityLegendary), Team: soloTeam(t), Qualifications: quals(t, true, "tinkers_tools")},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if r := reqs(res.(character.PlanMagicItemCraftingPlanned).Outcome()); r.TotalWorkHours() != 2000 {
					t.Errorf("requirements = %+v", r)
				}
			}, 0},
		{"magic item missing", p, "unhappy (missing content)", contenttest.NewFakeSource(tools()...), character.PlanMagicItemCraftingCommand{Facts: facts(t, "ring_x", content.MagicItemFactsFields{}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if m := res.(character.PlanMagicItemCraftingContentMissing).Missing(); !reflect.DeepEqual(m, []content.MissingContent{mid("ring_x").Missing()}) {
					t.Errorf("missing = %v", m)
				}
			}, 1},
		{"spell missing", p, "unhappy (missing content)", source(), character.PlanMagicItemCraftingCommand{Facts: facts(t, "wand_x", content.MagicItemFactsFields{AllowedSpells: []content.SpellID{"fireball", "ghost"}}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if m := res.(character.PlanMagicItemCraftingContentMissing).Missing(); !reflect.DeepEqual(m, []content.MissingContent{content.SpellID("ghost").Missing()}) {
					t.Errorf("missing = %v", m)
				}
			}, 3},
		{"incorporated item missing", p, "unhappy (missing content)", contenttest.NewFakeSource(append(tools(), armor)...), character.PlanMagicItemCraftingCommand{Facts: armorFacts(), Team: soloTeam(t), Incorporated: character.PayPurchaseCost},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if m := res.(character.PlanMagicItemCraftingContentMissing).Missing(); !reflect.DeepEqual(m, []content.MissingContent{content.ItemID("plate_x").Missing()}) {
					t.Errorf("missing = %v", m)
				}
			}, 2},
		{"no tool entries", p, "unhappy (missing content)", contenttest.NewFakeSource(ring), character.PlanMagicItemCraftingCommand{Facts: facts(t, "ring_x", content.MagicItemFactsFields{}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if got := res.(character.PlanMagicItemCraftingContentIncomplete).Reason(); got != character.CraftingToolSetMissing {
					t.Errorf("reason = %v", got)
				}
			}, 2},
		{"Potion of Healing by the equipment rules", p, "unhappy (rule interaction)", source(weaponOf("potion_x", content.WeaponMelee, gp(50))), character.PlanMagicItemCraftingCommand{
			Facts: facts(t, "ring_x", content.MagicItemFactsFields{CraftingBasis: must(content.NewEquipmentRulesCrafting("potion_x"))}), Team: soloTeam(t), Qualifications: quals(t, true, "jewelers_tools")},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				u, ok := res.(character.PlanMagicItemCraftingPlanned).Outcome().(character.MagicItemUsesEquipmentRules)
				if !ok || u.Item != "potion_x" {
					t.Errorf("outcome = %#v", res)
				}
				if _, ok := u.Outcome.(character.CraftingRequirements); !ok {
					t.Errorf("nested = %#v", u.Outcome)
				}
			}, 0},
		{"Spell Scroll without its table", p, "unhappy (missing content)", source(must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(30)), "", "", 3))), character.PlanMagicItemCraftingCommand{
			Facts: facts(t, "ring_x", content.MagicItemFactsFields{CraftingBasis: must(content.NewEquipmentRulesCrafting("scroll_x"))}), Team: soloTeam(t), Qualifications: quals(t, true, "jewelers_tools")},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if got := res.(character.PlanMagicItemCraftingContentIncomplete).Reason(); got != character.CraftingScribingTableMissing {
					t.Errorf("reason = %v", got)
				}
			}, 0},
		{"equipment rules gear item missing", p, "unhappy (missing content)", source(), character.PlanMagicItemCraftingCommand{
			Facts: facts(t, "ring_x", content.MagicItemFactsFields{CraftingBasis: must(content.NewEquipmentRulesCrafting("potion_x"))}), Team: soloTeam(t)},
			func(t *testing.T, res character.PlanMagicItemCraftingResult) {
				t.Helper()
				if m := res.(character.PlanMagicItemCraftingContentMissing).Missing(); !reflect.DeepEqual(m, []content.MissingContent{content.ItemID("potion_x").Missing()}) {
					t.Errorf("missing = %v", m)
				}
			}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			res, err := planUse(t, c.src, character.WithTelemetry(rec.Telemetry())).Do(ctx, c.cmd)
			if err != nil {
				t.Fatalf("err = %v [%s]", err, c.requirement)
			}
			plain, err2 := planUse(t, c.src).Do(ctx, c.cmd)
			if err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Errorf("no-op bundle differs: %#v %v", plain, err2)
			}
			c.check(t, res)
			if c.queries != 0 && len(c.src.Queries()) != 2*c.queries { // the recording and the no-op call
				t.Errorf("queries = %d, want %d per call", len(c.src.Queries()), c.queries)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-109 - the plan records its span, the load_content child span and one counter increment
// with the result and rejection; content failures set error status with no error.type; a rule outcome sets none.
// Path: per case.
func TestPlanMagicItemCraftingSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ring := magicEntry(t, "ring_x", content.MagicItemRing, single(content.RarityRare))
	belt := magicEntry(t, "belt_x", content.MagicItemWondrousItem, content.RarityVaries{})
	cmdFor := func(id string, q []character.CrafterQualification) character.PlanMagicItemCraftingCommand {
		return character.PlanMagicItemCraftingCommand{Facts: facts(t, id, content.MagicItemFactsFields{}), Team: soloTeam(t), Qualifications: q}
	}
	full := append([]content.Entry{ring, belt}, tools()...)
	cases := []struct {
		name   string
		src    *contenttest.FakeSource
		cmd    character.PlanMagicItemCraftingCommand
		status telemetry.Status
		child  telemetry.Status
		want   map[string]any
	}{
		{"planned", contenttest.NewFakeSource(full...), cmdFor("ring_x", quals(t, true, "jewelers_tools")), telemetry.StatusUnset, telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "planned"}},
		{"rejected", contenttest.NewFakeSource(full...), cmdFor("ring_x", nil), telemetry.StatusUnset, telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "arcana"}},
		{"rarity unresolved", contenttest.NewFakeSource(full...), cmdFor("belt_x", nil), telemetry.StatusUnset, telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "rarity_unresolved"}},
		{"content missing", contenttest.NewFakeSource(tools()...), cmdFor("ring_x", nil), telemetry.StatusError, telemetry.StatusError, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing"}},
		{"content incomplete", contenttest.NewFakeSource(ring), cmdFor("ring_x", nil), telemetry.StatusError, telemetry.StatusError, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_incomplete"}},
	}
	for _, c := range cases {
		rec := telemetrytest.NewRecorder()
		if _, err := planUse(t, c.src, character.WithTelemetry(rec.Telemetry())).Do(ctx, c.cmd); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		spans := rec.Tracer.Spans()
		if len(spans) < 1 || spans[0].Name != character.SpanPlanMagicItemCrafting || spans[0].Status != c.status {
			t.Errorf("%s: spans = %+v", c.name, spans)
			continue
		}
		sameAttrs(t, c.name+" span", spans[0].Attrs, c.want)
		sameAttrs(t, c.name+" operations", onlyOne(t, rec, character.MetricOperations).Attrs, attrs(c.want, map[string]any{character.AttrOperation: "plan_magic_item_crafting"}))
		if len(spans) == 2 && (spans[1].Name != character.SpanLoadContent || spans[1].Status != c.child) {
			t.Errorf("%s: child = %+v", c.name, spans[1])
		}
		noLogsOrViolations(t, rec)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-106, FR-107 - a source failure or panic comes back as the *srderr.PortError of the content
// port with a nil result, recorded once on both spans and the counter, with the cause preserved and no log; invalid input is refused
// before any lookup or signal.
// Path: unhappy (failure).
func TestPlanMagicItemCraftingFailuresAndInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ring := magicEntry(t, "ring_x", content.MagicItemRing, single(content.RarityRare))
	cause := errors.New("boom")
	cmd := character.PlanMagicItemCraftingCommand{Facts: facts(t, "ring_x", content.MagicItemFactsFields{}), Team: soloTeam(t)}
	for name, src := range map[string]*contenttest.FakeSource{
		"lookup fails": contenttest.NewFakeSource(ring).FailLookup(cause),
		"list fails":   contenttest.NewFakeSource(ring).FailListKind(cause),
	} {
		rec := telemetrytest.NewRecorder()
		res, err := planUse(t, src, character.WithTelemetry(rec.Telemetry())).Do(ctx, cmd)
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || !errors.Is(err, cause) || srderr.ErrorType(err) != "content_error" {
			t.Errorf("%s: %v %v", name, res, err)
		}
		if spans := rec.Tracer.Spans(); len(spans) == 0 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
			t.Errorf("%s: spans %+v", name, spans)
		}
		sameAttrs(t, name, onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: "plan_magic_item_crafting", telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"})
		noLogsOrViolations(t, rec)
	}
	use := planUse(t, contenttest.NewFakeSource(ring))
	armor := facts(t, "ring_x", content.MagicItemFactsFields{IncorporatedItem: must(content.NewIncorporatedItem("plate_x", 100))})
	for name, c := range map[string]character.PlanMagicItemCraftingCommand{
		"zero facts":                          {Team: soloTeam(t)},
		"zero team":                           {Facts: cmd.Facts},
		"choice without an incorporated item": {Facts: cmd.Facts, Team: soloTeam(t), Incorporated: character.PayPurchaseCost},
		"incorporated item without a choice":  {Facts: armor, Team: soloTeam(t)},
		"unknown choice":                      {Facts: armor, Team: soloTeam(t), Incorporated: 9},
	} {
		if r, err := use.Do(ctx, c); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v %v", name, r, err)
		}
	}
	// A base item the category needs but the command lacks is invalid input found after the lookups.
	armorEntry := magicEntry(t, "armor_x", content.MagicItemArmor, single(content.RarityRare))
	withTools := planUse(t, contenttest.NewFakeSource(append(tools(), armorEntry)...))
	if r, err := withTools.Do(ctx, character.PlanMagicItemCraftingCommand{Facts: facts(t, "armor_x", content.MagicItemFactsFields{}), Team: soloTeam(t)}); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("armor without a base item: %v %v", r, err)
	}
	if _, err := character.NewPlanMagicItemCrafting(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil source: %v", err)
	}
	if r, err := (character.PlanMagicItemCrafting{}).Do(ctx, cmd); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec := telemetrytest.NewRecorder()
	if r, err := planUse(t, contenttest.NewFakeSource(ring), character.WithTelemetry(rec.Telemetry())).Do(cancelled, cmd); r != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v %v", r, err)
	}
}
