package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func key(s string) character.WeaponInstanceKey { return character.MustWeaponInstanceKey(s) }

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-057 to FR-064, FR-067, FR-070, table 4.5a (first part) -
// the Light extra attack is allowed or blocked with the first reason in order, and costs a
// Bonus Action unless Nick is usable (SRD 5.2.1, "Light", p. 89; "Nick", p. 90).
// Path: per case.
func TestPlanLightExtraAttack(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	var start character.WeaponTurnState
	taken := start.RecordAttackAction()
	daggerA, daggerB, swordKey, axeKey := key("dagger-a"), key("dagger-b"), key("shortsword"), key("longsword")
	chosen := func(names ...string) character.WeaponMasteryAccess {
		return character.MasteryFromChoice{Choice: choiceOf(t, names...)}
	}
	cases := []struct {
		name, requirement, path string
		state                   character.WeaponTurnState
		weapon                  string
		instance                character.WeaponInstanceKey
		access                  character.WeaponMasteryAccess
		inPlay                  character.MasteryInPlay
		wantCost                character.ExtraAttackCost
		wantBlock               character.LightExtraAttackBlockReason
	}{
		{"Shortsword then Dagger", "FR-057, FR-063", "happy", taken.RecordAttack(swordKey, true), "dagger", daggerA, character.NoMasteryFeature{}, nil, character.ExtraAttackSpendsBonusAction, 0},
		{"Dagger then Dagger (two instances)", "FR-061", "unhappy (boundary)", taken.RecordAttack(daggerA, true), "dagger", daggerB, character.NoMasteryFeature{}, nil, character.ExtraAttackSpendsBonusAction, 0},
		{"another recorded Light weapon allows it", "FR-061", "unhappy (boundary)", taken.RecordAttack(daggerA, true).RecordAttack(swordKey, true), "dagger", daggerA, nil, nil, character.ExtraAttackSpendsBonusAction, 0},
		{"Same dagger", "FR-061", "unhappy (boundary)", taken.RecordAttack(daggerA, true), "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedSameLightWeapon},
		{"Non-Light weapon", "FR-059", "unhappy (rule interaction)", taken.RecordAttack(swordKey, true), "longsword", axeKey, character.NoMasteryFeature{}, nil, 0, character.LightBlockedWeaponNotLight},
		{"No Attack action", "FR-058", "unhappy (state error)", start.RecordAttack(swordKey, true), "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedAttackActionNotTaken},
		{"Non-Light first", "FR-060, FR-070", "unhappy (state error)", taken.RecordAttack(axeKey, false), "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedNoLightWeaponAttackMade},
		{"Attack action alone", "FR-060", "unhappy (empty case)", taken, "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedNoLightWeaponAttackMade},
		{"Already made", "FR-062", "unhappy (state error)", taken.RecordAttack(swordKey, true).RecordLightExtraAttack(), "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedAlreadyMadeThisTurn},
		{"Nick in the chosen set", "FR-064", "unhappy (rule interaction)", taken.RecordAttack(swordKey, true), "dagger", daggerA, chosen("dagger", "longsword"), nil, character.ExtraAttackPartOfAttackAction, 0},
		{"Nick not chosen", "FR-064", "unhappy (state error)", taken.RecordAttack(swordKey, true), "dagger", daggerA, chosen("longsword"), nil, character.ExtraAttackSpendsBonusAction, 0},
		{"Non-Nick Light weapon", "FR-064", "unhappy (rule interaction)", taken.RecordAttack(daggerA, true), "handaxe", key("axe"), chosen("handaxe"), nil, character.ExtraAttackSpendsBonusAction, 0},
		{"Replaced mastery is Nick", "FR-064", "unhappy (rule interaction)", taken.RecordAttack(daggerA, true), "handaxe", key("axe"), chosen("handaxe"), character.ReplacedMastery{Property: content.MasteryNick}, character.ExtraAttackPartOfAttackAction, 0},
		{"Replaced mastery hides Nick", "FR-064", "unhappy (rule interaction)", taken.RecordAttack(swordKey, true), "dagger", daggerA, chosen("dagger"), character.ReplacedMastery{Property: content.MasteryVex}, character.ExtraAttackSpendsBonusAction, 0},
		{"Order 0", "FR-067", "unhappy (rule interaction)", start.RecordAttack(axeKey, false), "longsword", axeKey, character.NoMasteryFeature{}, nil, 0, character.LightBlockedAttackActionNotTaken},
		{"Order 1", "FR-067", "unhappy (rule interaction)", taken.RecordLightExtraAttack().RecordAttack(swordKey, true), "longsword", axeKey, character.NoMasteryFeature{}, nil, 0, character.LightBlockedWeaponNotLight},
		{"Order 2", "FR-067", "unhappy (rule interaction)", taken.RecordLightExtraAttack(), "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedAlreadyMadeThisTurn},
		{"Order 3", "FR-067", "unhappy (rule interaction)", taken.RecordLightExtraAttack().RecordAttack(daggerA, true), "dagger", daggerA, character.NoMasteryFeature{}, nil, 0, character.LightBlockedAlreadyMadeThisTurn},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := character.PlanLightExtraAttack(character.LightExtraAttackInput{State: c.state, Weapon: ws[c.weapon], InstanceKey: c.instance, Access: c.access, InPlay: c.inPlay})
			switch r := res.(type) {
			case character.LightExtraAttackAllowed:
				if c.wantBlock != 0 || r.Cost != c.wantCost {
					t.Errorf("allowed at %v; want cost %v, block %v [%s, %s]", r.Cost, c.wantCost, c.wantBlock, c.requirement, c.path)
				}
			case character.LightExtraAttackBlocked:
				if r.Reason != c.wantBlock {
					t.Errorf("blocked %v, want %v [%s, %s]", r.Reason, c.wantBlock, c.requirement, c.path)
				}
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-065, FR-066, table 4.5a (damage rows) - the extra
// attack omits the ability modifier unless negative, and an active Two-Weapon Fighting
// effect adds it whatever its sign.
// Path: per case.
func TestPlanLightExtraAttackDamageModifier(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	state := character.WeaponTurnState{}.RecordAttackAction().RecordAttack(key("a"), true)
	cases := []struct {
		name, requirement string
		mod               int
		source            character.ExtraAttackDamageSource
		wantMode          character.AbilityModifierToDamage
		wantAdded         int
	}{
		{"positive omitted", "FR-065", 3, character.NoExtraAttackDamageEffect{}, character.AddModifierOmitUnlessNegative, 0},
		{"zero omitted", "FR-065", 0, character.NoExtraAttackDamageEffect{}, character.AddModifierOmitUnlessNegative, 0},
		{"negative added", "FR-065", -1, character.NoExtraAttackDamageEffect{}, character.AddModifierOmitUnlessNegative, -1},
		{"nil source", "FR-065", 3, nil, character.AddModifierOmitUnlessNegative, 0},
		{"feature adds positive", "FR-066", 3, character.ExtraAttackDamageEffectActive{}, character.AddModifierAlways, 3},
		{"feature adds negative", "FR-066", -2, character.ExtraAttackDamageEffectActive{}, character.AddModifierAlways, -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := character.PlanLightExtraAttack(character.LightExtraAttackInput{State: state, Weapon: ws["dagger"], InstanceKey: key("b"), AbilityModifier: modifierOf(t, c.mod), DamageSource: c.source})
			a, ok := res.(character.LightExtraAttackAllowed)
			if !ok || a.DamageModifier != c.wantMode || a.AddedModifier != modifier.New(c.wantAdded) {
				t.Errorf("result = %#v; want %v and %d [%s]", res, c.wantMode, c.wantAdded, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-068, FR-069, FR-106 - the zero state records nothing, each
// Record method returns a new instance and leaves the receiver unchanged.
// Path: unhappy (immutability, empty and minimal case).
func TestWeaponTurnStateIsImmutable(t *testing.T) {
	t.Parallel()
	var zero character.WeaponTurnState
	a := zero.RecordAttackAction()
	b := a.RecordAttack(key("a"), true)
	c := b.RecordLightExtraAttack()
	d := c.RecordCleaveAttack()
	if !zero.Equal(character.WeaponTurnState{}) || zero.Equal(a) || a.Equal(b) || b.Equal(c) || c.Equal(d) {
		t.Error("a Record method changed its receiver or recorded nothing")
	}
	if !d.Equal(character.WeaponTurnState{}.RecordAttackAction().RecordAttack(key("a"), true).RecordLightExtraAttack().RecordCleaveAttack()) {
		t.Error("equal histories must be equal")
	}
	b2 := a.RecordAttack(key("b"), true)
	if b.Equal(b2) {
		t.Error("different weapons must differ")
	}
	if _, err := character.NewWeaponInstanceKey("  "); err == nil {
		t.Error("a blank key must fail")
	} else {
		wantInvalid(t, err, "blank key")
	}
	x := key("x")
	if x != key("x") || x == key("y") || x.String() != "x" {
		t.Error("keys compare by value")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustWeaponInstanceKey did not panic")
		}
	}()
	character.MustWeaponInstanceKey("")
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-067, FR-063 - the cost and reason names.
// Path: happy.
func TestLightExtraAttackNames(t *testing.T) {
	t.Parallel()
	if character.ExtraAttackSpendsBonusAction.String() != "spendsBonusAction" || character.ExtraAttackPartOfAttackAction.String() != "partOfAttackAction" || character.ExtraAttackCost(0).String() != "unknown" {
		t.Error("cost names")
	}
	if character.LightBlockedSameLightWeapon.String() != "sameLightWeapon" || character.LightExtraAttackBlockReason(0).String() != "unknown" || character.LightExtraAttackBlockReason(6).String() != "unknown" {
		t.Error("reason names")
	}
}
