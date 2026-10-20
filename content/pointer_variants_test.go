package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/rules/go.md (never panics) - every closed variant set built
// from value variants rejects a nil pointer and a non-nil pointer to a valid
// variant with an error, never a panic and never acceptance, so a caller cannot
// mutate a variant after construction.
// Path: per family.
func TestClosedSetsRejectPointerVariants(t *testing.T) {
	t.Parallel()
	base := decodeClassFixture(t, ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6)))
	build := func(edit func(*content.ClassFields)) error {
		f := fields(base)
		edit(&f)
		return errOf(content.NewClassDefinition("fixture", "Fixture", content.SourceCitation{}, f))
	}
	allOf := mustValue(content.NewAllOf(ability.Strength))
	any1 := mustValue(content.NewSkillChoiceAny(1))
	specific := mustValue(content.NewClassToolProficiencySpecific("Thieves' Tools"))
	grants := func(tools []content.ClassToolProficiency, skills content.SkillChoice) error {
		return errOf(content.NewMulticlassGrants(nil, nil, tools, skills))
	}
	row := func(features []content.RowFeature, sc content.SpellcastingRow, v map[content.ClassColumnID]content.ColumnValue) error {
		return errOf(content.NewProgressionRow(1, 2, features, sc, v))
	}
	cases := []blockCase{
		{"class: nil primary ability pointer", build(func(f *content.ClassFields) { f.PrimaryAbility = (*content.AllOf)(nil) }), "ClassDefinition", content.BlockInvalidMember},
		{"class: primary ability pointer", build(func(f *content.ClassFields) { f.PrimaryAbility = &allOf }), "ClassDefinition", content.BlockInvalidMember},
		{"class: nil skill choice pointer", build(func(f *content.ClassFields) { f.SkillChoice = (*content.SkillChoiceAny)(nil) }), "ClassDefinition", content.BlockInvalidMember},
		{"class: skill choice pointer", build(func(f *content.ClassFields) { f.SkillChoice = &any1 }), "ClassDefinition", content.BlockInvalidMember},
		{"class: nil tool pointer", build(func(f *content.ClassFields) {
			f.ToolProficiencies = []content.ClassToolProficiency{(*content.ClassToolProficiencySpecific)(nil)}
		}), "ClassDefinition", content.BlockInvalidMember},
		{"class: tool pointer", build(func(f *content.ClassFields) { f.ToolProficiencies = []content.ClassToolProficiency{&specific} }), "ClassDefinition", content.BlockInvalidMember},
		{"class: nil spellcasting pointer", build(func(f *content.ClassFields) { f.Spellcasting = (*content.SlotSpellcastingTrait)(nil) }), "ClassDefinition", content.BlockInvalidMember},
		{"class: spellcasting pointer", build(func(f *content.ClassFields) {
			f.Spellcasting = ptr(mustValue(content.NewSlotSpellcastingTrait(ability.Wisdom, true)))
		}), "ClassDefinition", content.BlockInvalidMember},
		{"grants: nil tool pointer", grants([]content.ClassToolProficiency{(*content.ClassToolProficiencyChoice)(nil)}, any1), "MulticlassGrants", content.BlockInvalidMember},
		{"grants: tool pointer", grants([]content.ClassToolProficiency{&specific}, any1), "MulticlassGrants", content.BlockInvalidMember},
		{"grants: nil skills pointer", grants(nil, (*content.SkillChoiceListed)(nil)), "MulticlassGrants", content.BlockInvalidMember},
		{"grants: skills pointer", grants(nil, &any1), "MulticlassGrants", content.BlockInvalidMember},
		{"row: nil feature pointer", row([]content.RowFeature{(*content.RowSubclassFeature)(nil)}, nil, nil), "ProgressionRow", content.BlockInvalidMember},
		{"row: feature pointer", row([]content.RowFeature{&content.RowSubclassFeature{}}, nil, nil), "ProgressionRow", content.BlockInvalidMember},
		{"row: nil spellcasting pointer", row(nil, (*content.SlotSpellcastingRow)(nil), nil), "ProgressionRow", content.BlockInvalidMember},
		{"row: spellcasting pointer", row(nil, ptr(mustValue(content.NewSlotSpellcastingRow(1, make([]int, 9)))), nil), "ProgressionRow", content.BlockInvalidMember},
		{"row: nil value pointer", row(nil, nil, map[content.ClassColumnID]content.ColumnValue{"a": (*content.ColumnNone)(nil)}), "ProgressionRow", content.BlockInvalidMember},
		{"row: value pointer", row(nil, nil, map[content.ClassColumnID]content.ColumnValue{"a": &content.ColumnNone{}}), "ProgressionRow", content.BlockInvalidMember},
		{"potion: nil effect pointer", errOf(content.NewPotionMiscibilityBand(1, 2, (*content.OnePotionPermanent)(nil))), "PotionMiscibilityBand", content.BlockInvalidMember},
		{"potion: effect pointer", errOf(content.NewPotionMiscibilityBand(1, 2, &content.OnePotionPermanent{})), "PotionMiscibilityBand", content.BlockInvalidMember},
		{"fact: nil pointer", errOf(content.NewFeatureEntry("f", "F", content.FeatureEntryFields{Level: 1, Text: "t", Facts: []content.FeatureFact{(*content.GrantsSubclass)(nil)}})), "FeatureEntry", content.BlockInvalidMember},
		{"fact: pointer", errOf(content.NewFeatureEntry("f", "F", content.FeatureEntryFields{Level: 1, Text: "t", Facts: []content.FeatureFact{&content.GrantsSubclass{}}})), "FeatureEntry", content.BlockInvalidMember},
		{"table: nil cell pointer", errOf(content.NewFeatureTable("T", []string{"A"}, [][]content.TableCell{{(*content.TableCellInt)(nil)}})), "FeatureTable", content.BlockInvalidMember},
		{"table: cell pointer", errOf(content.NewFeatureTable("T", []string{"A"}, [][]content.TableCell{{ptr(mustValue(content.NewTableCellInt(1)))}})), "FeatureTable", content.BlockInvalidMember},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantBlockError(t, c.err, c.block, c.want)
		})
	}
}

// Requirement: doc/rules/go.md (never panics) - the spell, magic item and
// feature-free families reject a nil pointer and a pointer to a valid variant
// with the documented BlockError (block and reason pinned) and never panic.
// Path: per family.
func TestClosedSetsRejectPointerVariantsSpellsAndItems(t *testing.T) {
	t.Parallel()
	sphere := mustValue(content.NewSphereArea(10))
	fixed := mustValue(content.NewFixedSpellAmount(1))
	perSlot := mustValue(content.NewPerSlotLevelCount(1, 1))
	only := mustValue(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
	single := mustValue(content.NewSingleRarity(content.RarityCommon))
	minutes := mustValue(content.NewCastingTimeMinutes(1))
	singleTimes := mustValue(content.NewSpellCastingTimesSingle(minutes))
	selfRange := content.NewSpellRangeSelf()
	timed := mustValue(content.NewSpellDurationTimed(1, content.DurationUnitHours, content.LimitFixed))
	modelledFx := fireball(content.CoverageComplete)
	sf := func(edit func(*content.SpellFields)) error {
		_, err := baseSpell(edit)
		return err
	}
	mf := func(edit func(*content.MagicItemFields)) error {
		f := miFields()
		edit(&f)
		return errOf(content.NewMagicItemEntry("rod_x", "Rod", content.SourceCitation{}, f))
	}
	cases := []struct {
		name  string
		err   error
		block string
		want  content.BlockProblem
	}{
		{"area: nil", errOf(content.NewTargetsInArea((*content.SphereArea)(nil), content.SubjectCreature, nil)), "AreaShape", content.BlockMissing},
		{"area: pointer", errOf(content.NewTargetsInArea(&sphere, content.SubjectCreature, nil)), "AreaShape", content.BlockMissing},
		{"restriction: nil", errOf(content.NewTargetsInArea(sphere, content.SubjectCreature, (*content.OnlyCreatureTypes)(nil))), "TargetsInArea", content.BlockInvalidMember},
		{"restriction: pointer", errOf(content.NewTargetsInArea(sphere, content.SubjectCreature, &only)), "TargetsInArea", content.BlockInvalidMember},
		{"count scaling: nil", errOf(content.NewExactlyTargets(1, (*content.PerSlotLevelCount)(nil))), "CountScaling", content.BlockInvalidMember},
		{"count scaling: pointer", errOf(content.NewExactlyTargets(1, &perSlot)), "CountScaling", content.BlockInvalidMember},
		{"amount: nil", errOf(content.NewScalableAmount((*content.FixedSpellAmount)(nil), nil)), "SpellAmount", content.BlockMissing},
		{"amount: pointer", errOf(content.NewScalableAmount(&fixed, nil)), "SpellAmount", content.BlockMissing},
		{"rarity: nil", mf(func(f *content.MagicItemFields) { f.Rarity = (*content.SingleRarity)(nil) }), "MagicItemEntry", content.BlockInvalidMember},
		{"rarity: pointer", mf(func(f *content.MagicItemFields) { f.Rarity = &single }), "MagicItemEntry", content.BlockInvalidMember},
		{"attunement: nil", mf(func(f *content.MagicItemFields) { f.Attunement = (*content.AttunementRequired)(nil) }), "MagicItemEntry", content.BlockInvalidMember},
		{"attunement: pointer", mf(func(f *content.MagicItemFields) { f.Attunement = &content.AttunementRequired{} }), "MagicItemEntry", content.BlockInvalidMember},
		{"block: pointer", mf(func(f *content.MagicItemFields) { f.Description = []content.TextBlock{&content.TextParagraph{}} }), "MagicItemEntry", content.BlockInvalidMember},
		{"casting time: nil", errOf(content.NewSpellCastingTimesSingle((*content.CastingTimeMagicAction)(nil))), "CastingTime", content.BlockMissing},
		{"casting time: pointer", errOf(content.NewSpellCastingTimesSingle(&minutes)), "CastingTime", content.BlockMissing},
		{"mode time: pointer", errOf(content.NewCastingTimeMode("A", &minutes)), "CastingTime", content.BlockMissing},
		{"casting times: nil", sf(func(f *content.SpellFields) { f.CastingTimes = (*content.SpellCastingTimesSingle)(nil) }), "SpellCastingTimes", content.BlockMissing},
		{"casting times: pointer", sf(func(f *content.SpellFields) { f.CastingTimes = &singleTimes }), "SpellCastingTimes", content.BlockMissing},
		{"range: nil", sf(func(f *content.SpellFields) { f.Range = (*content.SpellRangeSelf)(nil) }), "SpellRange", content.BlockMissing},
		{"range: pointer", sf(func(f *content.SpellFields) { f.Range = &selfRange }), "SpellRange", content.BlockMissing},
		{"duration: nil", sf(func(f *content.SpellFields) { f.Duration = (*content.SpellDurationSpecial)(nil) }), "SpellDuration", content.BlockMissing},
		{"duration: pointer", sf(func(f *content.SpellFields) { f.Duration = &timed }), "SpellDuration", content.BlockMissing},
		{"effects: nil", sf(func(f *content.SpellFields) { f.Effects = (*content.TextOnlyEffects)(nil) }), "Spell", content.BlockInvalidMember},
		{"effects: pointer", sf(func(f *content.SpellFields) { f.Effects = &modelledFx }), "Spell", content.BlockInvalidMember},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantBlockError(t, c.err, c.block, c.want)
		})
	}
}
