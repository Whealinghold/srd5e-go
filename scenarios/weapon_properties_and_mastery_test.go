package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// must2 returns v, or panics with err: a broken fixture, not a failure of the code under test.
func must2[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// weapon builds a synthetic weapon for the scenario.
func weapon(t *testing.T, id string, cat content.WeaponCategory, dmg content.WeaponDamage, m content.MasteryProperty, props ...content.WeaponPropertyUse) content.Weapon {
	t.Helper()
	return must2(content.NewWeapon(content.ItemID(id), id, content.SourceCitation{}, cat, content.WeaponMelee, dmg, props, m,
		content.MustWeight(1, ""), content.MustFixedCost(content.MustCost(1, content.CoinGP, 0))))
}

// A level 4 Fighter chooses Dagger, Shortsword, Greatsword and Maul, plans a
// Shortsword attack, takes the Nick extra attack as part of the Attack action with a
// negative modifier, gets a Graze offer with the Greatsword on a miss and a Topple
// offer with the Maul that the Resolver turns into a saving throw (SRD 5.2.1,
// "Weapon Mastery", p. 48; "Light", "Nick", "Graze", "Topple", pp. 89-90). Cites
// weapon-properties-and-mastery.md FR-014, FR-022, FR-030, FR-057, FR-064, FR-065,
// FR-078, FR-090, FR-097, FR-098, FR-110, FR-123.
func TestWeaponMasteryFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := func(count int, die dice.Die, typ content.DamageType) content.WeaponDamage {
		return must2(content.NewWeaponDiceDamage(dice.MustExpression(count, die, 0), typ))
	}
	finesse := must2(content.NewPlainUse(content.WeaponPropertyFinesse))
	lightUse := must2(content.NewPlainUse(content.WeaponPropertyLight))
	heavyUse := must2(content.NewPlainUse(content.WeaponPropertyHeavy))
	twoHanded := must2(content.NewTwoHandedUse(content.TwoHandedAlways))
	dagger := weapon(t, "dagger", content.WeaponSimple, d(1, dice.D4(), content.DamagePiercing), content.MasteryNick, finesse, lightUse)
	shortsword := weapon(t, "shortsword", content.WeaponMartial, d(1, dice.D6(), content.DamagePiercing), content.MasteryVex, finesse, lightUse)
	greatsword := weapon(t, "greatsword", content.WeaponMartial, d(2, dice.D6(), content.DamageSlashing), content.MasteryGraze, heavyUse, twoHanded)
	maul := weapon(t, "maul", content.WeaponMartial, d(2, dice.D6(), content.DamageBludgeoning), content.MasteryTopple, heavyUse, twoHanded)
	resolved := map[content.ItemID]content.Weapon{"dagger": dagger, "shortsword": shortsword, "greatsword": greatsword, "maul": maul}

	// Level 1: three kinds; level 4: one more (Fighter, SRD 5.2.1, p. 48).
	kinds := []content.ItemID{"dagger", "shortsword", "greatsword"}
	first := must2(character.CreateWeaponMasteryChoice(kinds, resolved, 3, character.SimpleOrMartialEligibility{}))
	current := first.(character.WeaponMasteryChoiceAccepted).Choice()
	second := must2(character.ExtendWeaponMasteryChoice(current, append(kinds, "maul"), 4, resolved, character.SimpleOrMartialEligibility{}))
	choice := second.(character.WeaponMasteryChoiceAccepted).Choice()
	states := must2(character.SetFeatureState(character.FeatureStates{}, character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(choice)))
	access := must2(character.WeaponMasteryAccessFor(states))

	// Plan a Shortsword attack with Dexterity 14, Strength 10.
	strength, dexterity := must2(ability.NewScore(10)), must2(ability.NewScore(14))
	planned := character.PlanWeaponAttack(character.WeaponAttackInput{
		Weapon: shortsword, Mode: character.MeleeAttackMode{}, Strength: strength, Dexterity: dexterity,
		Preference: character.MustChosenAbility(ability.Dexterity), Wielding: character.WeaponWielding{Grip: character.GripOneHanded},
		Proficiencies: character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponSimple), character.CategoryProficiency(content.WeaponMartial)),
	})
	plan := planned.(character.WeaponAttackPlanned).Plan
	if plan.Ability() != ability.Dexterity || plan.Proficiency() != character.WeaponProficient || plan.DamageModifier() != character.AddModifierAlways {
		t.Fatalf("plan = %+v", plan)
	}

	// Nick: the Dagger extra attack is part of the Attack action; a -1 modifier is still added.
	turn := character.WeaponTurnState{}.RecordAttackAction().RecordAttack(character.MustWeaponInstanceKey("shortsword"), true)
	extra := character.PlanLightExtraAttack(character.LightExtraAttackInput{
		State: turn, Weapon: dagger, InstanceKey: character.MustWeaponInstanceKey("dagger"), Access: access,
		AbilityModifier: must2(ability.NewScore(8)).Modifier(), DamageSource: character.NoExtraAttackDamageEffect{},
	}).(character.LightExtraAttackAllowed)
	if extra.Cost != character.ExtraAttackPartOfAttackAction || extra.AddedModifier != modifier.New(-1) {
		t.Fatalf("extra attack = %+v", extra)
	}

	// A Greatsword attack that misses offers Graze for the ability modifier.
	missRoll := must2(d20.ResolveAttack(d20.MustRoll(modifier.Normal, 4), d20.NewRollModifiers(modifier.New(5)), 18, d20.DefaultCriticalRange))
	graze := must2(character.OfferMastery(character.MasteryAttackFacts{
		WeaponID: "greatsword", Weapon: greatsword, Access: access, Outcome: character.MasteryAttackRolled{Roll: missRoll},
		Mode: character.MeleeAttackMode{}, AbilityModifier: strength.Modifier(),
	}))
	if g, ok := graze.(character.GrazeOffer); !ok || g.Amount != 0 || g.DamageType != content.DamageSlashing {
		t.Fatalf("graze = %#v", graze)
	}

	// A Maul hit offers Topple at DC 8 + 3 + 2 = 13; the Resolver rolls a 12 and the target falls Prone.
	hitRoll := must2(d20.ResolveAttack(d20.MustRoll(modifier.Normal, 15), d20.NewRollModifiers(modifier.New(5)), 15, d20.DefaultCriticalRange))
	offer := must2(character.OfferMastery(character.MasteryAttackFacts{
		WeaponID: "maul", Weapon: maul, Access: access, Outcome: character.MasteryAttackRolled{Roll: hitRoll}, Mode: character.MeleeAttackMode{},
		AbilityModifier: must2(ability.NewScore(16)).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(4)), DamageDealt: 7,
	})).(character.ToppleOffer)
	if offer.DC() != 13 {
		t.Fatalf("DC = %d", offer.DC())
	}
	resolver := must2(character.NewToppleSaveResolver(dicetest.NewFixedRoller(12)))
	res := must2(resolver.ResolveToppleSave(ctx, character.ResolveToppleSaveCommand{
		Offer: offer, Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))},
	}))
	if prone, ok := res.(character.ToppleSaveTargetProne); !ok || prone.Condition() != content.ConditionProne {
		t.Fatalf("result = %#v", res)
	}
}

// Every closed result of the weapon rules can be switched over exhaustively
// with public identifiers only (FR-110).
func TestWeaponClosedResultsAreExhaustiveForCallers(t *testing.T) {
	t.Parallel()
	results := []any{
		character.WeaponMasteryChoiceAccepted{}, character.WeaponMasteryChoiceRejected{}, character.WeaponAttackPlanned{}, character.WeaponAttackPlanBlocked{},
		character.LightExtraAttackAllowed{}, character.LightExtraAttackBlocked{}, character.CleaveTargetAccepted{}, character.CleaveTargetRejected{},
		character.PushDistanceAccepted{}, character.PushDistanceRejected{}, character.ToppleSaveTargetProne{}, character.ToppleSaveTargetResisted{},
	}
	for _, r := range results {
		switch r.(type) {
		case character.WeaponMasteryChoiceAccepted, character.WeaponMasteryChoiceRejected, character.WeaponAttackPlanned, character.WeaponAttackPlanBlocked,
			character.LightExtraAttackAllowed, character.LightExtraAttackBlocked, character.CleaveTargetAccepted, character.CleaveTargetRejected,
			character.PushDistanceAccepted, character.PushDistanceRejected, character.ToppleSaveTargetProne, character.ToppleSaveTargetResisted:
		default:
			t.Errorf("%T is not a known variant", r)
		}
	}
	for _, o := range []character.MasteryOffer{character.CleaveOffer{}, character.GrazeOffer{}, character.PushOffer{}, character.SapOffer{}, character.SlowOffer{}, character.ToppleOffer{}, character.VexOffer{}, character.MasteryNone{}} {
		if o == nil {
			t.Error("nil offer")
		}
	}
}
