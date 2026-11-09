package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A character wearing a storm Belt of Giant Strength, a Ring of Protection and a Cloak of Protection has Strength 29 and Armor Class and saving throw
// bonuses of +2 each; a Wand of Magic Missiles emptied by a 3-charge cast rolls its last-charge check; a Bag of Holding placed in a Portable Hole
// destroys both and opens a gate (SRD 5.2.1, "Belt of Giant Strength", p. 213; "Ring of Protection", p. 238; "Wand of Magic Missiles", p. 251; "Bag of
// Holding", p. 212). Cites magic-item-effects.md FR-011, FR-016, FR-029, FR-033, FR-039, FR-060, FR-062.
func TestMagicItemEffectsThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	table, worn := character.MagicItemEffectTable{}, character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageWorn}
	lookup := func(id string, choice character.EnhancementChoice) []character.MagicItemEffect {
		found, ok := table.Lookup(content.MustMagicItemID(id), choice).(character.MagicItemEffectsFound)
		if !ok {
			t.Fatalf("%s is not modelled", id)
		}
		return found.Effects()
	}
	active, err := character.ActiveEffectsOf([]character.EquippedItem{
		{Instance: "belt-1", Effects: lookup("belt_of_giant_strength", character.EnhancementChoice{Giant: character.GiantStorm}), Context: worn},
		{Instance: "ring-1", Effects: lookup("ring_of_protection", character.EnhancementChoice{}), Context: worn},
		{Instance: "cloak-1", Effects: lookup("cloak_of_protection", character.EnhancementChoice{}), Context: worn},
	})
	if err != nil {
		t.Fatal(err)
	}
	base := map[ability.Ability]ability.Score{}
	for _, a := range ability.All() {
		base[a] = ability.MustScore(10)
	}
	scores, err := ability.NewScores(base)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := character.EffectiveScoresOf(scores, active)
	if err != nil {
		t.Fatal(err)
	}
	total := func(bs []character.MagicItemBonus) (n int) {
		for _, b := range bs {
			n += b.Modifier().Value()
		}
		return n
	}
	if eff.Scores().Score(ability.Strength).Value() != 29 || total(active.ArmorClass()) != 2 || total(active.SavingThrows()) != 2 {
		t.Errorf("Strength %d, AC %d, saves %d", eff.Scores().Score(ability.Strength).Value(), total(active.ArmorClass()), total(active.SavingThrows()))
	}

	// The bonuses reach derived stats as flat bonuses (FR-012; derived-stats.md FR-038, FR-079).
	itemOf := map[character.MagicItemInstanceID]string{"ring-1": "ring_of_protection", "cloak-1": "cloak_of_protection"}
	flat := func(bs []character.MagicItemBonus) (out []character.FlatBonus) {
		for _, b := range bs {
			fb, ferr := character.NewFlatBonus(b.Modifier(), character.MagicItemFlatBonusSource{Item: content.MustMagicItemID(itemOf[b.InstanceID()])})
			if ferr != nil {
				t.Fatal(ferr)
			}
			out = append(out, fb)
		}
		return out
	}
	var saveBonuses []character.FlatSavingThrowBonus
	for _, fb := range flat(active.SavingThrows()) {
		sb, serr := character.NewFlatSavingThrowBonus(fb, ability.All())
		if serr != nil {
			t.Fatal(serr)
		}
		saveBonuses = append(saveBonuses, sb)
	}
	calc, err := character.NewCalculateDerivedStats(character.WithSource(data.MustBundledSource()))
	if err != nil {
		t.Fatal(err)
	}
	withItems, err := calc.Run(context.Background(), character.CalculateDerivedStatsCommand{
		ClassLevels:   character.MustClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 5}}),
		AbilityScores: eff.Scores(), HitPointMaximum: character.MustHitPointMaximum(27),
		FlatArmorClassBonuses: flat(active.ArmorClass()), FlatSavingThrowBonuses: saveBonuses,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, isOK := withItems.(character.CalculateDerivedStatsSucceeded)
	if !isOK {
		t.Fatalf("result = %#v", withItems)
	}
	origins := 0
	for _, term := range got.Stats.ArmorClassBreakdown().Terms() {
		if term.Source == character.FlatBonusTerm && term.Origin != nil {
			origins++
		}
	}
	// Dexterity 10 (+0): Armor Class 10 + 2. Strength 29 (+9) and Charisma 10 (+0), neither proficient, each gain the +2 of the Ring and Cloak.
	if got.Stats.ArmorClass() != 12 || origins != 2 {
		t.Errorf("AC %d with %d Origin terms; want 12 with 2", got.Stats.ArmorClass(), origins)
	}
	if str, cha := got.Stats.SavingThrow(ability.Strength).Value(), got.Stats.SavingThrow(ability.Charisma).Value(); str != 11 || cha != 2 {
		t.Errorf("Strength save %d, Charisma save %d; want 11 and 2", str, cha)
	}

	holder := character.MustIdentifier("mage")
	magicMissile := character.MustChargesPerLevel(content.MustSpellID("magic_missile"), content.MustSpellLevel(1), 3, character.UsersSpellStats())
	plan, err := (character.ChargedSpellCasting{}).Plan(magicMissile, 3, content.MustSpellLevel(1))
	if err != nil {
		t.Fatal(err)
	}
	cast, err := (character.ChargedSpellCasting{}).Cast(character.MustItemCharges(7, 3), plan.(character.ChargedSpellPlan), holder, "wand-1", false)
	if err != nil {
		t.Fatal(err)
	}
	emptied := cast.(character.ChargedSpellCastApplied)
	if emptied.Charges().Current() != 0 || emptied.Spell().CastAtLevel() != content.MustSpellLevel(3) {
		t.Fatalf("cast = %+v", emptied)
	}
	roller := dicetest.NewFixedRoller(1)
	resolve, err := character.NewResolveLastCharge(roller)
	if err != nil {
		t.Fatal(err)
	}
	rule := (character.LastChargeRules{}).For(content.MustMagicItemID("wand_of_magic_missiles"))
	res, err := resolve.Do(context.Background(), character.ResolveLastChargeCommand{CharacterID: holder, Instance: "wand-1", Charges: emptied.Charges(), Rule: rule})
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := res.(character.ResolveLastChargeResolved); !ok || r.Consequence() != character.LastChargeDestroyed || len(roller.Requests()) != 1 || roller.Requests()[0] != dice.MustExpression(1, dice.D20(), 0) {
		t.Errorf("last charge = %+v, requests %v", res, roller.Requests())
	}

	nest, err := (character.ExtradimensionalNesting{}).Place(character.BagOfHoldingSpace{}, character.PortableHoleSpace{})
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := nest.(character.NestingDestroysBoth); !ok || n.Gate().Area().Kind() != character.GateSphere || n.Gate().Area().Feet() != 10 || !n.Gate().OneWay() || n.Gate().Reopenable() {
		t.Errorf("nesting = %+v", nest)
	}
}
