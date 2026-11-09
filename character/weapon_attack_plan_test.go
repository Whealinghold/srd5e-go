package character_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func planned(t *testing.T, res character.WeaponAttackPlanResult) character.WeaponAttackPlan {
	t.Helper()
	p, ok := res.(character.WeaponAttackPlanned)
	if !ok {
		t.Fatalf("result = %#v, want a plan", res)
	}
	return p.Plan
}

func blockedReason(t *testing.T, res character.WeaponAttackPlanResult) character.WeaponAttackBlockReason {
	t.Helper()
	b, ok := res.(character.WeaponAttackPlanBlocked)
	if !ok {
		t.Fatalf("result = %#v, want blocked", res)
	}
	return b.Reason
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-030 to FR-032, FR-036 to FR-040, FR-042, table 4.4a -
// the ability, Heavy Disadvantage, reach and range of a plan (SRD 5.2.1, "Attack Rolls",
// p. 7; "Finesse", "Heavy", "Reach", "Thrown", pp. 89-90).
// Path: per case.
func TestPlanWeaponAttackAbilityHeavyAndReach(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	two := character.GripTwoHanded
	type rangeWant struct{ normal, long int }
	cases := []struct {
		name, requirement, path string
		weapon                  string
		mode                    character.WeaponAttackMode
		pref                    character.AbilityPreference
		str, dex                int
		reach                   int
		grip                    character.Grip
		wantAbility             ability.Ability
		wantDisadvantage        bool
		wantReach               int
		wantRange               rangeWant
		wantKind                character.WeaponAttackKind
	}{
		{"Longsword melee", "FR-031", "happy", "longsword", character.MeleeAttackMode{}, character.DefaultAbility{}, 14, 10, 5, 0, ability.Strength, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Rapier, chosen Dexterity", "FR-032", "happy", "rapier", character.MeleeAttackMode{}, character.MustChosenAbility(ability.Dexterity), 8, 16, 5, 0, ability.Dexterity, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Rapier, chosen Strength", "FR-032", "happy", "rapier", character.MeleeAttackMode{}, character.MustChosenAbility(ability.Strength), 16, 8, 5, 0, ability.Strength, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Rapier, default", "FR-032", "unhappy (boundary)", "rapier", character.MeleeAttackMode{}, character.DefaultAbility{}, 8, 16, 5, 0, ability.Strength, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Rapier, nil preference", "FR-032", "unhappy (empty case)", "rapier", character.MeleeAttackMode{}, nil, 8, 16, 5, 0, ability.Strength, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Dagger thrown, Finesse", "FR-032, FR-042", "happy", "dagger", character.ThrownAttackMode{}, character.MustChosenAbility(ability.Dexterity), 10, 16, 5, 0, ability.Dexterity, false, 0, rangeWant{20, 60}, character.RangedWeaponAttackKind},
		{"Handaxe thrown (no Finesse)", "FR-031", "unhappy (rule interaction)", "handaxe", character.ThrownAttackMode{}, character.DefaultAbility{}, 16, 10, 5, 0, ability.Strength, false, 0, rangeWant{20, 60}, character.RangedWeaponAttackKind},
		{"Javelin thrown 30/120", "FR-042", "happy", "javelin", character.ThrownAttackMode{}, character.DefaultAbility{}, 16, 10, 5, 0, ability.Strength, false, 0, rangeWant{30, 120}, character.RangedWeaponAttackKind},
		{"Dart thrown", "FR-031", "happy", "dart", character.ThrownAttackMode{}, character.DefaultAbility{}, 10, 14, 5, 0, ability.Dexterity, false, 0, rangeWant{20, 60}, character.RangedWeaponAttackKind},
		{"Shortbow ranged", "FR-031, FR-042", "happy", "shortbow", character.RangedAttackMode{}, character.DefaultAbility{}, 10, 14, 5, two, ability.Dexterity, false, 0, rangeWant{80, 320}, character.RangedWeaponAttackKind},
		{"Greatsword Str 13", "FR-036", "unhappy (boundary)", "greatsword", character.MeleeAttackMode{}, character.DefaultAbility{}, 13, 8, 5, two, ability.Strength, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Greatsword Str 12", "FR-036, FR-037", "unhappy (boundary)", "greatsword", character.MeleeAttackMode{}, character.DefaultAbility{}, 12, 20, 5, two, ability.Strength, true, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Heavy Crossbow Dex 13", "FR-036", "unhappy (boundary)", "heavy_crossbow", character.RangedAttackMode{}, character.DefaultAbility{}, 8, 13, 5, two, ability.Dexterity, false, 0, rangeWant{100, 400}, character.RangedWeaponAttackKind},
		{"Heavy Crossbow Dex 12", "FR-036, FR-037", "unhappy (boundary)", "heavy_crossbow", character.RangedAttackMode{}, character.DefaultAbility{}, 20, 12, 5, two, ability.Dexterity, true, 0, rangeWant{100, 400}, character.RangedWeaponAttackKind},
		{"Longsword Str 1", "FR-038", "unhappy (empty case)", "longsword", character.MeleeAttackMode{}, character.DefaultAbility{}, 1, 10, 5, 0, ability.Strength, false, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Heavy tests the score, not the chosen ability", "FR-037", "unhappy (rule interaction)", "greatsword", character.MeleeAttackMode{}, character.DefaultAbility{}, 12, 20, 5, two, ability.Strength, true, 5, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Glaive reach 5", "FR-039", "happy", "glaive", character.MeleeAttackMode{}, character.DefaultAbility{}, 14, 10, 5, two, ability.Strength, false, 10, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Glaive reach 10", "FR-039", "unhappy (boundary)", "glaive", character.MeleeAttackMode{}, character.DefaultAbility{}, 14, 10, 10, two, ability.Strength, false, 15, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Zero reach reads as 5", "FR-039", "unhappy (empty case)", "glaive", character.MeleeAttackMode{}, character.DefaultAbility{}, 14, 10, 0, two, ability.Strength, false, 10, rangeWant{}, character.MeleeWeaponAttackKind},
		{"Whip (Finesse, Reach)", "FR-039", "unhappy (rule interaction)", "whip", character.MeleeAttackMode{}, character.MustChosenAbility(ability.Dexterity), 8, 15, 5, 0, ability.Dexterity, false, 10, rangeWant{}, character.MeleeWeaponAttackKind},
		{"No Reach keeps own reach", "FR-040", "unhappy (boundary)", "longsword", character.MeleeAttackMode{}, character.DefaultAbility{}, 14, 10, 10, 0, ability.Strength, false, 10, rangeWant{}, character.MeleeWeaponAttackKind},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := character.WeaponAttackInput{
				Weapon: ws[c.weapon], Mode: c.mode, Strength: score(t, c.str), Dexterity: score(t, c.dex), Preference: c.pref,
				Wielding:      character.WeaponWielding{Grip: c.grip}.WithAmmunition(character.MustAmmunitionSupply(5, ammoOf(ws[c.weapon]))),
				AttackerReach: reachOf(c.reach), Proficiencies: character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponSimple), character.CategoryProficiency(content.WeaponMartial)),
			}
			p := planned(t, character.PlanWeaponAttack(in))
			if p.Kind() != c.wantKind || p.Ability() != c.wantAbility {
				t.Errorf("kind = %v, ability = %v; want %v, %v [%s, %s]", p.Kind(), p.Ability(), c.wantKind, c.wantAbility, c.requirement, c.path)
			}
			wantMod := score(t, c.str).Modifier()
			if c.wantAbility == ability.Dexterity {
				wantMod = score(t, c.dex).Modifier()
			}
			if p.AbilityModifier() != wantMod {
				t.Errorf("modifier = %v, want %v", p.AbilityModifier(), wantMod)
			}
			var want []modifier.RollCircumstance
			if c.wantDisadvantage {
				want = []modifier.RollCircumstance{modifier.Disadvantage{}}
			}
			if got := p.Circumstances(); !reflect.DeepEqual(got, want) {
				t.Errorf("circumstances = %v, want %v", got, want)
			}
			switch ror := p.ReachOrRange().(type) {
			case character.PlannedReach:
				if ror.Reach.Feet() != c.wantReach || c.wantRange != (rangeWant{}) {
					t.Errorf("reach = %d, want %d", ror.Reach.Feet(), c.wantReach)
				}
			case character.PlannedRange:
				if (rangeWant{ror.Range.Normal(), ror.Range.Long()}) != c.wantRange || c.wantReach != 0 {
					t.Errorf("range = %v, want %v and no reach", ror.Range, c.wantRange)
				}
			}
		})
	}
}

func reachOf(feet int) character.ReachInFeet {
	if feet == 0 {
		return character.ReachInFeet{}
	}
	return character.MustReachInFeet(feet)
}

// ammoOf returns the ammunition type of w, or Arrow for a weapon without Ammunition.
func ammoOf(w content.Weapon) content.AmmunitionType {
	for _, p := range w.Properties() {
		if a, ok := p.(content.AmmunitionUse); ok {
			return a.Ammunition()
		}
	}
	return content.AmmunitionArrow
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-030, FR-034, FR-035, FR-043 to FR-056, FR-114 to FR-116,
// FR-124, table 4.4b and the order rows - the damage, Versatile and Two-Handed rules, Ammunition,
// Loading, improvised and unsupported uses, and which block reason wins.
// Path: per case.
func TestPlanWeaponAttackHandlingTable(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	arrow, bolt, bullet, needle := content.AmmunitionArrow, content.AmmunitionBolt, content.AmmunitionBullet, content.AmmunitionNeedle
	type in struct {
		weapon                string
		mode                  character.WeaponAttackMode
		pref                  character.AbilityPreference
		grip                  character.Grip
		exemption             character.TwoHandedExemption
		hands                 character.HandState
		loading               character.LoadingState
		supply                int // -1 for none carried
		supplyType            content.AmmunitionType
		proficient            bool
		wantDamage, wantBlock string
	}
	m, th, rg := character.MeleeAttackMode{}, character.ThrownAttackMode{}, character.RangedAttackMode{}
	one, twoH := character.GripOneHanded, character.GripTwoHanded
	free, none := character.HandStateFreeHandAvailable, character.HandStateNoFreeHand
	ready, fired := character.LoadingReady, character.LoadingAlreadyFiredThisActivation
	applies, noEx := character.TwoHandedExemptionApplies, character.TwoHandedNoExemption
	def := character.DefaultAbility{}
	cases := []struct {
		name, requirement, path string
		in                      in
	}{
		{"Longsword two hands", "FR-052", "happy", in{weapon: "longsword", mode: m, pref: def, grip: twoH, supply: -1, proficient: true, wantDamage: "1d10 Slashing"}},
		{"Longsword one hand", "FR-052", "happy", in{weapon: "longsword", mode: m, pref: def, grip: one, supply: -1, proficient: true, wantDamage: "1d8 Slashing"}},
		{"Spear thrown, two hands", "FR-052", "unhappy (boundary)", in{weapon: "spear", mode: th, pref: def, grip: twoH, supply: -1, proficient: true, wantDamage: "1d6 Piercing"}},
		{"Spear melee, two hands", "FR-052", "happy", in{weapon: "spear", mode: m, pref: def, grip: twoH, supply: -1, proficient: true, wantDamage: "1d8 Piercing"}},
		{"Mace two hands", "FR-053", "unhappy (rule interaction)", in{weapon: "mace", mode: m, pref: def, grip: twoH, supply: -1, proficient: true, wantDamage: "1d6 Bludgeoning"}},
		{"Greatsword one hand", "FR-050", "unhappy (state error)", in{weapon: "greatsword", mode: m, pref: def, grip: one, supply: -1, wantBlock: "needsTwoHands"}},
		{"Greatsword zero grip", "FR-050", "unhappy (empty case)", in{weapon: "greatsword", mode: m, pref: def, supply: -1, wantBlock: "needsTwoHands"}},
		{"Lance, mounted", "FR-050", "unhappy (rule interaction)", in{weapon: "lance", mode: m, pref: def, grip: one, exemption: applies, supply: -1, proficient: true, wantDamage: "1d10 Piercing"}},
		{"Lance, not mounted", "FR-050", "unhappy (state error)", in{weapon: "lance", mode: m, pref: def, grip: one, exemption: noEx, supply: -1, wantBlock: "needsTwoHands"}},
		{"Greatsword exemption", "FR-051", "unhappy (invalid input)", in{weapon: "greatsword", mode: m, pref: def, grip: one, exemption: applies, supply: -1, wantBlock: "exemptionNotApplicable"}},
		{"Mace exemption (no Two-Handed)", "FR-051", "unhappy (invalid input)", in{weapon: "mace", mode: m, pref: def, grip: one, exemption: applies, supply: -1, wantBlock: "exemptionNotApplicable"}},
		{"Shortbow 0 arrows", "FR-044", "unhappy (boundary)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, supply: 0, supplyType: arrow, wantBlock: "noAmmunition"}},
		{"Shortbow 1 arrow", "FR-044, FR-045", "unhappy (boundary)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, supply: 1, supplyType: arrow, proficient: true, wantDamage: "1d6 Piercing"}},
		{"Pistol, no free hand", "FR-047", "unhappy (state error)", in{weapon: "pistol", mode: rg, pref: def, grip: one, hands: none, supply: 5, supplyType: bullet, wantBlock: "noFreeHandToLoad"}},
		{"Pistol, free hand", "FR-047", "happy", in{weapon: "pistol", mode: rg, pref: def, grip: one, hands: free, supply: 5, supplyType: bullet, proficient: true, wantDamage: "1d10 Piercing"}},
		{"Light Crossbow, no free hand", "FR-047", "unhappy (rule interaction)", in{weapon: "light_crossbow", mode: rg, pref: def, grip: twoH, hands: none, supply: 5, supplyType: bolt, proficient: true, wantDamage: "1d8 Piercing"}},
		{"Hand Crossbow second shot", "FR-048", "unhappy (state error)", in{weapon: "hand_crossbow", mode: rg, pref: def, grip: one, hands: free, loading: fired, supply: 5, supplyType: bolt, wantBlock: "loadingLimit"}},
		{"Hand Crossbow first shot", "FR-048", "happy", in{weapon: "hand_crossbow", mode: rg, pref: def, grip: one, hands: free, loading: ready, supply: 5, supplyType: bolt, proficient: true, wantDamage: "1d6 Piercing"}},
		{"Shortbow second shot", "FR-049", "unhappy (rule interaction)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, loading: fired, supply: 5, supplyType: arrow, proficient: true, wantDamage: "1d6 Piercing"}},
		{"Shortbow first shot", "FR-049", "unhappy (rule interaction)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, loading: ready, supply: 5, supplyType: arrow, proficient: true, wantDamage: "1d6 Piercing"}},
		{"Sling in melee", "FR-034", "unhappy (rule interaction)", in{weapon: "sling", mode: m, pref: def, supply: -1, wantBlock: "improvisedUse"}},
		{"Club thrown", "FR-034", "unhappy (rule interaction)", in{weapon: "club", mode: th, pref: def, supply: -1, wantBlock: "improvisedUse"}},
		{"Dart ranged", "FR-116", "unhappy (rule interaction)", in{weapon: "dart", mode: rg, pref: def, supply: -1, wantBlock: "modeNotSupportedByWeapon"}},
		{"Chosen ability without Finesse", "FR-035", "unhappy (invalid input)", in{weapon: "mace", mode: m, pref: character.MustChosenAbility(ability.Strength), supply: -1, wantBlock: "finesseNotAvailable"}},
		{"Longsword ranged", "FR-116", "unhappy (rule interaction)", in{weapon: "longsword", mode: rg, pref: def, supply: -1, wantBlock: "modeNotSupportedByWeapon"}},
		{"Shortbow thrown", "FR-116", "unhappy (rule interaction)", in{weapon: "shortbow", mode: th, pref: def, supply: -1, wantBlock: "modeNotSupportedByWeapon"}},
		{"nil mode", "FR-116", "unhappy (empty case)", in{weapon: "longsword", pref: def, supply: -1, wantBlock: "modeNotSupportedByWeapon"}},
		{"Dart thrown keeps proficiency", "FR-116", "unhappy (rule interaction)", in{weapon: "dart", mode: th, pref: def, supply: -1, proficient: true, wantDamage: "1d4 Piercing"}},
		{"Thrown, no free hand", "FR-043", "unhappy (rule interaction)", in{weapon: "dagger", mode: th, pref: def, hands: none, supply: -1, proficient: true, wantDamage: "1d4 Piercing"}},
		{"Shortbow, no supply", "FR-124", "unhappy (state error)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, supply: -1, wantBlock: "noAmmunition"}},
		{"Longsword, supply ignored", "FR-124", "unhappy (rule interaction)", in{weapon: "longsword", mode: m, pref: def, supply: 5, supplyType: arrow, proficient: true, wantDamage: "1d8 Slashing"}},
		{"Shortbow, wrong ammunition", "FR-115", "unhappy (invalid input)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, supply: 20, supplyType: bolt, wantBlock: "wrongAmmunitionType"}},
		{"Blowgun, Needle", "FR-114", "unhappy (rule interaction)", in{weapon: "blowgun", mode: rg, pref: def, grip: one, supply: 5, supplyType: needle, loading: ready, proficient: true, wantDamage: "1 Piercing"}},
		{"Blowgun, Strength choice", "FR-114", "unhappy (invalid input)", in{weapon: "blowgun", mode: rg, pref: character.MustChosenAbility(ability.Strength), supply: 5, supplyType: needle, wantBlock: "finesseNotAvailable"}},
		{"Order 1", "FR-055", "unhappy (rule interaction)", in{weapon: "sling", mode: m, pref: character.MustChosenAbility(ability.Strength), supply: -1, wantBlock: "improvisedUse"}},
		{"Order 2", "FR-055", "unhappy (rule interaction)", in{weapon: "longsword", mode: rg, pref: character.MustChosenAbility(ability.Strength), supply: -1, wantBlock: "modeNotSupportedByWeapon"}},
		{"Order 3", "FR-055", "unhappy (rule interaction)", in{weapon: "mace", mode: m, pref: character.MustChosenAbility(ability.Strength), exemption: applies, supply: -1, wantBlock: "finesseNotAvailable"}},
		{"Order 4", "FR-055", "unhappy (rule interaction)", in{weapon: "greatsword", mode: m, pref: def, grip: one, exemption: applies, supply: -1, wantBlock: "exemptionNotApplicable"}},
		{"Order 5", "FR-055", "unhappy (rule interaction)", in{weapon: "longbow", mode: rg, pref: def, grip: one, supply: 5, supplyType: bolt, wantBlock: "needsTwoHands"}},
		{"Order 6", "FR-055", "unhappy (rule interaction)", in{weapon: "shortbow", mode: rg, pref: def, grip: twoH, supply: 0, supplyType: bolt, wantBlock: "wrongAmmunitionType"}},
		{"Order 7", "FR-055", "unhappy (rule interaction)", in{weapon: "pistol", mode: rg, pref: def, grip: one, hands: none, supply: 0, supplyType: bullet, wantBlock: "noAmmunition"}},
		{"Order 8", "FR-055", "unhappy (rule interaction)", in{weapon: "pistol", mode: rg, pref: def, grip: one, hands: none, loading: fired, supply: 5, supplyType: bullet, wantBlock: "noFreeHandToLoad"}},
		{"Heavy and non-proficient", "FR-056", "unhappy (rule interaction)", in{weapon: "greatsword", mode: m, pref: def, grip: twoH, supply: -1, wantDamage: "2d6 Slashing"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			w := character.WeaponWielding{Grip: c.in.grip, Exemption: c.in.exemption, Hands: c.in.hands, Loading: c.in.loading}
			if c.in.supply >= 0 {
				w = w.WithAmmunition(character.MustAmmunitionSupply(c.in.supply, c.in.supplyType))
			}
			rules := character.NewWeaponProficiencies()
			if c.in.proficient {
				rules = character.NewWeaponProficiencies(character.AnyStatBlockWeaponProficiency{})
			}
			res := character.PlanWeaponAttack(character.WeaponAttackInput{
				Weapon: ws[c.in.weapon], Mode: c.in.mode, Strength: score(t, 12), Dexterity: score(t, 14), Preference: c.in.pref,
				Wielding: w, Proficiencies: rules,
			})
			if c.in.wantBlock != "" {
				if got := blockedReason(t, res).String(); got != c.in.wantBlock {
					t.Errorf("blocked = %s, want %s [%s, %s]", got, c.in.wantBlock, c.requirement, c.path)
				}
				return
			}
			p := planned(t, res)
			if got := damageText(p.Damage()); got != c.in.wantDamage {
				t.Errorf("damage = %s, want %s [%s, %s]", got, c.in.wantDamage, c.requirement, c.path)
			}
			wantProf := character.WeaponNotProficient
			if c.in.proficient {
				wantProf = character.WeaponProficient
			}
			if p.Proficiency() != wantProf {
				t.Errorf("proficiency = %v, want %v", p.Proficiency(), wantProf)
			}
			wantMod := character.AddModifierAlways
			if _, fixed := p.Damage().(content.WeaponFixedDamage); fixed {
				wantMod = character.AddModifierNever
			}
			if p.DamageModifier() != wantMod {
				t.Errorf("damage modifier = %v, want %v", p.DamageModifier(), wantMod)
			}
		})
	}
}

func damageText(d content.WeaponDamage) string {
	switch v := d.(type) {
	case content.WeaponDiceDamage:
		return v.Dice().Notation() + " " + v.DamageType().String()
	case content.WeaponFixedDamage:
		return strconv.Itoa(v.Amount()) + " " + v.DamageType().String()
	}
	return "?"
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-045, FR-048, FR-049, FR-124 - the plan carries the supply
// after the shot (one piece fewer), the loading state after it, and no supply for a
// weapon without Ammunition.
// Path: per case.
func TestPlanWeaponAttackCarriesAmmunitionAndLoadingAfter(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	plan := func(weapon string, mode character.WeaponAttackMode, loading character.LoadingState, supply int, typ content.AmmunitionType) character.WeaponAttackPlan {
		w := character.WeaponWielding{Grip: character.GripTwoHanded, Hands: character.HandStateFreeHandAvailable, Loading: loading}
		if supply >= 0 {
			w = w.WithAmmunition(character.MustAmmunitionSupply(supply, typ))
		}
		return planned(t, character.PlanWeaponAttack(character.WeaponAttackInput{Weapon: ws[weapon], Mode: mode, Strength: score(t, 12), Dexterity: score(t, 14), Wielding: w}))
	}
	p := plan("shortbow", character.RangedAttackMode{}, 0, 1, content.AmmunitionArrow)
	if a, ok := p.AmmunitionAfter(); !ok || a.Pieces() != 0 || a.Type() != content.AmmunitionArrow {
		t.Errorf("Shortbow 1 arrow: after = %v, %v", a, ok)
	}
	if p.LoadingAfter() != 0 {
		t.Errorf("a weapon without Loading must keep the input loading, got %v", p.LoadingAfter())
	}
	if p := plan("shortbow", character.RangedAttackMode{}, character.LoadingAlreadyFiredThisActivation, 5, content.AmmunitionArrow); p.LoadingAfter() != character.LoadingAlreadyFiredThisActivation {
		t.Error("Shortbow second shot must keep alreadyFired unchanged")
	}
	if p := plan("shortbow", character.RangedAttackMode{}, character.LoadingReady, 5, content.AmmunitionArrow); p.LoadingAfter() != character.LoadingReady {
		t.Error("Shortbow first shot must keep ready")
	}
	if p := plan("hand_crossbow", character.RangedAttackMode{}, character.LoadingReady, 5, content.AmmunitionBolt); p.LoadingAfter() != character.LoadingAlreadyFiredThisActivation {
		t.Error("Hand Crossbow first shot must plan alreadyFired")
	}
	p = plan("longsword", character.MeleeAttackMode{}, 0, 5, content.AmmunitionArrow)
	if _, ok := p.AmmunitionAfter(); ok {
		t.Error("a weapon without Ammunition must plan no supply")
	}
	p = plan("hand_crossbow", character.RangedAttackMode{}, character.LoadingReady, 5, content.AmmunitionBolt)
	if a, _ := p.AmmunitionAfter(); a.Pieces() != 4 {
		t.Errorf("pieces after = %d, want 4", a.Pieces())
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-033, FR-054, FR-105, FR-106 - the plan holds one ability
// modifier and the weapon damage, never a rolled amount; the circumstances are a copy; the
// input is not altered.
// Path: unhappy (immutability).
func TestPlanWeaponAttackIsImmutableAndPure(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	in := character.WeaponAttackInput{Weapon: ws["greatsword"], Mode: character.MeleeAttackMode{}, Strength: score(t, 12), Dexterity: score(t, 14), Wielding: character.WeaponWielding{Grip: character.GripTwoHanded}}
	p := planned(t, character.PlanWeaponAttack(in))
	p.Circumstances()[0] = modifier.Advantage{}
	if _, ok := p.Circumstances()[0].(modifier.Disadvantage); !ok {
		t.Error("Circumstances exposes the plan")
	}
	again := planned(t, character.PlanWeaponAttack(in))
	if !reflect.DeepEqual(p, again) {
		t.Error("the same input must give the same plan")
	}
	if p.AbilityModifier() != score(t, 12).Modifier() {
		t.Errorf("modifier = %v", p.AbilityModifier())
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-035 - NewChosenAbility accepts Strength and Dexterity
// only; Must panics with the same error.
// Path: unhappy (invalid input).
func TestNewChosenAbility(t *testing.T) {
	t.Parallel()
	for _, a := range []ability.Ability{ability.Strength, ability.Dexterity} {
		c, err := character.NewChosenAbility(a)
		if err != nil || c.Ability() != a {
			t.Errorf("%v: %v, %v", a, c, err)
		}
	}
	for _, a := range []ability.Ability{ability.Constitution, ability.Charisma, 0} {
		_, err := character.NewChosenAbility(a)
		wantInvalid(t, err, a.String())
	}
	defer func() {
		if recover() == nil {
			t.Error("MustChosenAbility did not panic")
		}
	}()
	character.MustChosenAbility(ability.Wisdom)
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-030, FR-055 - the kind and block reason names.
// Path: happy.
func TestWeaponAttackNames(t *testing.T) {
	t.Parallel()
	if character.MeleeWeaponAttackKind.String() != "meleeWeaponAttack" || character.RangedWeaponAttackKind.String() != "rangedWeaponAttack" || character.WeaponAttackKind(0).String() != "unknown" {
		t.Error("attack kind names")
	}
	if character.AttackBlockedLoadingLimit.String() != "loadingLimit" || character.WeaponAttackBlockReason(0).String() != "unknown" || character.WeaponAttackBlockReason(10).String() != "unknown" {
		t.Error("block reason names")
	}
	for m, want := range map[character.AbilityModifierToDamage]string{character.AddModifierOmitUnlessNegative: "omitUnlessNegative", character.AddModifierAlways: "alwaysAdd", character.AddModifierNever: "neverAdd", 0: "unknown"} {
		if m.String() != want {
			t.Errorf("%d = %q, want %q", int(m), m.String(), want)
		}
	}
}
