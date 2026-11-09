package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-034, Table B (SRD 5.2.1, "Breath Weapon", p. 84) - the Breath Weapon declares the shape, a Dexterity save, DC 8 + Constitution modifier + Proficiency Bonus, the dice of the level (1d10 at 1 to 4, 2d10 at 5 to 10, 3d10 at 11 to 16, 4d10 at 17 to 20), the ancestry's type, half on a success and the replaced attack.
// Path: per case.
func TestBreathWeaponEffect(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		level int
		con   int
		dc    int
		dice  int
	}{
		{"level 1", 1, 14, 12, 1}, {"level 4", 4, 14, 12, 1}, {"level 5", 5, 14, 13, 2},
		{"level 10", 10, 16, 15, 2}, {"level 11", 11, 16, 15, 3}, {"level 16", 16, 16, 16, 3},
		{"level 17", 17, 16, 17, 4}, {"level 20", 20, 16, 17, 4}, {"negative modifier", 1, 8, 9, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := character.SpeciesEffects{}.Of(character.BreathWeaponRequest{Shape: character.BreathCone15}, spChar(t, c.level, c.con), spDragonborn(t, character.DraconicRed))
			want := character.BreathWeaponEffect{Shape: character.BreathCone15, SaveAbility: ability.Dexterity, SaveDC: c.dc, Damage: dice.MustExpression(c.dice, dice.D10(), 0),
				DamageType: content.DamageFire, HalfOnSave: true, ReplacesOneAttack: true}
			if !ok || got != want {
				t.Errorf("Of = %#v, %v; want %#v", got, ok, want)
			}
		})
	}
	for anc, want := range map[character.DraconicAncestry]content.DamageType{character.DraconicSilver: content.DamageCold, character.DraconicGreen: content.DamagePoison} {
		got, _ := character.SpeciesEffects{}.Of(character.BreathWeaponRequest{Shape: character.BreathLine30}, spChar(t, 1, 14), spDragonborn(t, anc))
		if e := got.(character.BreathWeaponEffect); e.DamageType != want || e.Shape != character.BreathLine30 {
			t.Errorf("%v = %#v", anc, e)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-035, FR-036, FR-039, FR-040, FR-037, Table G (SRD 5.2.1, pp. 84-86) - Draconic Flight, Stonecunning, Large Form, Adrenaline Rush and the five declared Giant Ancestry benefits declare the facts of the printed text, and the Stone benefit declares nothing.
// Path: per case.
func TestSpeciesEffectsDeclared(t *testing.T) {
	t.Parallel()
	d := func(n int, die dice.Die) dice.Expression { return dice.MustExpression(n, die, 0) }
	pc := spChar(t, 5, 14)
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		req    character.SpeciesUseRequest
		want   character.SpeciesTraitEffect
	}{
		{"Draconic Flight", spDragonborn(t, character.DraconicRed), character.DraconicFlightRequest{}, character.DraconicFlightEffect{Cost: character.TraitBonusAction, DurationMinutes: 10, FlySpeedEqualsSpeed: true, EndsWhenIncapacitated: true}},
		{"Stonecunning", spDwarf(t), character.StonecunningRequest{}, character.StonecunningEffect{Cost: character.TraitBonusAction, TremorsenseFeet: 60, DurationMinutes: 10}},
		{"Large Form", spGoliath(t, character.AncestryHill), character.LargeFormRequest{HasEnoughSpace: true}, character.LargeFormEffect{Cost: character.TraitBonusAction, Size: content.SizeLarge, DurationMinutes: 10, SpeedBonusFeet: 10,
			StrengthCheckAdvantage: character.SpeciesAdvantage{Source: character.TraitLargeForm, Circumstance: modifier.Advantage{}}}},
		{"Adrenaline Rush", spOrc(t), character.AdrenalineRushRequest{}, character.AdrenalineRushEffect{DashAsBonusAction: true, TemporaryHitPoints: character.TemporaryHitPointsGrant{Amount: 3}}},
		{"Cloud's Jaunt", spGoliath(t, character.AncestryCloud), character.GiantAncestryRequest{Benefit: character.AncestryCloud}, character.CloudsJauntEffect{Cost: character.TraitBonusAction, TeleportFeet: 30}},
		{"Fire's Burn", spGoliath(t, character.AncestryFire), character.GiantAncestryRequest{Benefit: character.AncestryFire}, character.FiresBurnEffect{Damage: d(1, dice.D10()), DamageType: content.DamageFire}},
		{"Frost's Chill", spGoliath(t, character.AncestryFrost), character.GiantAncestryRequest{Benefit: character.AncestryFrost}, character.FrostsChillEffect{Damage: d(1, dice.D6()), DamageType: content.DamageCold, SpeedReductionFeet: 10}},
		{"Hill's Tumble", spGoliath(t, character.AncestryHill), character.GiantAncestryRequest{Benefit: character.AncestryHill}, character.HillsTumbleEffect{MaxTargetSize: content.SizeLarge, Condition: content.ConditionProne}},
		{"Storm's Thunder", spGoliath(t, character.AncestryStorm), character.GiantAncestryRequest{Benefit: character.AncestryStorm}, character.StormsThunderEffect{Cost: character.TraitReaction, RangeFeet: 60, Damage: d(1, dice.D8()), DamageType: content.DamageThunder}},
		{"Drow free cast", spElf(t, character.ElvenDrow), character.LineageSpellRequest{Level: 3}, character.SpeciesFreeCastEffect{Spell: "faerie_fire", Ability: ability.Wisdom}},
		{"Tiefling level 5 free cast", spTiefling(t, character.LegacyInfernal), character.LineageSpellRequest{Level: 5}, character.SpeciesFreeCastEffect{Spell: "darkness", Ability: ability.Charisma}},
		{"Forest Gnome free cast", spGnome(t, character.GnomishForest), character.GnomeSpellRequest{}, character.SpeciesFreeCastEffect{Spell: "speak_with_animals", Ability: ability.Intelligence}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := (character.SpeciesEffects{}).Of(c.req, pc, c.traits); !ok || got != c.want {
				t.Errorf("Of = %#v, %v; want %#v", got, ok, c.want)
			}
		})
	}
	if a := (character.AdrenalineRushEffect{}); a.TemporaryHitPoints.Amount != 0 {
		t.Error("the zero effect grants Temporary Hit Points")
	}
	if got, _ := (character.SpeciesEffects{}).Of(character.AdrenalineRushRequest{}, spChar(t, 9, 14), spOrc(t)); got.(character.AdrenalineRushEffect).TemporaryHitPoints.Amount != 4 {
		t.Errorf("Adrenaline Rush at level 9 = %#v", got)
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-037, FR-038 (SRD 5.2.1, "Giant Ancestry", pp. 85-86) - the Stone benefit, Stone's Endurance and Relentless Endurance declare no effect (their own use cases resolve them), and neither does a nil or invalid request or one the species lacks; GiantAncestry returns false for Stone and an invalid value.
// Path: unhappy (rule interaction).
func TestSpeciesEffectsNone(t *testing.T) {
	t.Parallel()
	pc := spChar(t, 5, 14)
	stone := spGoliath(t, character.AncestryStone)
	for name, c := range map[string]struct {
		traits character.SpeciesTraits
		req    character.SpeciesUseRequest
	}{
		"Stone benefit":      {stone, character.GiantAncestryRequest{Benefit: character.AncestryStone}},
		"Stone's Endurance":  {stone, character.StonesEnduranceRequest{}},
		"Relentless":         {spOrc(t), character.RelentlessEnduranceRequest{}},
		"nil request":        {stone, nil},
		"invalid shape":      {spDragonborn(t, character.DraconicRed), character.BreathWeaponRequest{}},
		"not offered":        {spDwarf(t), character.BreathWeaponRequest{Shape: character.BreathCone15}},
		"benefit not chosen": {spGoliath(t, character.AncestryHill), character.GiantAncestryRequest{Benefit: character.AncestryCloud}},
		"Rock Gnome":         {spGnome(t, character.GnomishRock), character.GnomeSpellRequest{}},
		"zero traits":        {character.SpeciesTraits{}, character.StonecunningRequest{}},
	} {
		if got, ok := (character.SpeciesEffects{}).Of(c.req, pc, c.traits); ok || got != nil {
			t.Errorf("%s: %#v, %v", name, got, ok)
		}
	}
	for _, a := range []character.GiantAncestry{character.AncestryStone, 0, 7} {
		if got, ok := (character.SpeciesEffects{}).GiantAncestry(a); ok || got != nil {
			t.Errorf("GiantAncestry(%v) = %#v, %v", a, got, ok)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-021, FR-031 - a free-cast effect is declared only for a spell the character already has at its level: asked below the level of the lineage spell, Of declares nothing (SpeciesUses.Expend refuses that case first with SpeciesUseNotYetAvailable).
// Path: unhappy (boundary).
func TestSpeciesEffectsFreeCastBelowLevel(t *testing.T) {
	t.Parallel()
	if got, ok := (character.SpeciesEffects{}).Of(character.LineageSpellRequest{Level: 3}, spChar(t, 2, 10), spElf(t, character.ElvenDrow)); ok || got != nil {
		t.Errorf("Of = %#v, %v", got, ok)
	}
}
