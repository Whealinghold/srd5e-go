package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// attackRoll resolves a real d20.AttackRollResult for a natural value against AC 15
// with a +5 modifier, so the natural 1 and 20 rules apply.
func attackRoll(t *testing.T, natural int) d20.AttackRollResult {
	t.Helper()
	r := must(d20.ResolveAttack(d20.MustRoll(modifier.Normal, natural), d20.NewRollModifiers(modifier.New(5)), 15, d20.DefaultCriticalRange))
	return r
}

// rolled returns an outcome builder for a roll that stands with the natural value.
func rolled(natural int) outcomeFn {
	return func(t *testing.T) character.MasteryAttackOutcome {
		t.Helper()
		return character.MasteryAttackRolled{Roll: attackRoll(t, natural)}
	}
}

// forcedMiss returns an outcome builder for a roll turned into a miss.
func forcedMiss(natural int) outcomeFn {
	return func(t *testing.T) character.MasteryAttackOutcome {
		t.Helper()
		return character.MasteryAttackForcedMiss{Roll: attackRoll(t, natural)}
	}
}

var (
	hit, crit, miss, nat1 = rolled(12), rolled(20), rolled(3), rolled(1)
	forced, forced20      = forcedMiss(12), forcedMiss(20)
)

func unrolled(*testing.T) character.MasteryAttackOutcome {
	return character.MasteryAttackUnrolledMiss{}
}

type outcomeFn = func(*testing.T) character.MasteryAttackOutcome

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-072 to FR-096, FR-118, FR-119, table 4.6a - the mastery
// offer for each property, the precedence of the reasons, and hit and miss taken from the
// d20 result (SRD 5.2.1, "Mastery Properties", p. 90).
// Path: per case.
func TestOfferMastery(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	usable := func(name string) character.WeaponMasteryAccess {
		return character.MasteryFromChoice{Choice: choiceOf(t, name)}
	}
	melee, thrown := character.MeleeAttackMode{}, character.ThrownAttackMode{}
	type want = character.MasteryOffer
	none := func(r character.MasteryNoneReason) want { return character.MasteryNone{Reason: r} }
	cases := []struct {
		name, requirement, path string
		weapon                  string
		access                  character.WeaponMasteryAccess // nil: the weapon is chosen
		inPlay                  character.MasteryInPlay
		outcome                 outcomeFn
		mode                    character.WeaponAttackMode
		mod, bonus              int
		size                    content.CreatureSize
		damage                  int
		turn                    character.WeaponTurnState
		want                    want
	}{
		{"Cleave, hit", "FR-074", "happy", "greataxe", nil, nil, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, character.CleaveOffer{WeaponID: "greataxe", Kind: character.MeleeWeaponAttackKind, Damage: ws["greataxe"].Damage(), DamageModifier: character.AddModifierOmitUnlessNegative}},
		{"Cleave, Critical Hit", "FR-096", "unhappy (boundary)", "greataxe", nil, nil, crit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, character.CleaveOffer{WeaponID: "greataxe", Kind: character.MeleeWeaponAttackKind, Damage: ws["greataxe"].Damage(), DamageModifier: character.AddModifierOmitUnlessNegative}},
		{"Cleave, miss", "FR-075", "unhappy (rule interaction)", "greataxe", nil, nil, miss, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Cleave, thrown hit", "FR-075", "unhappy (rule interaction)", "greataxe", nil, nil, hit, thrown, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, none(character.MasteryNoneNotMeleeAttack)},
		{"Cleave, nil mode", "FR-075", "unhappy (empty case)", "greataxe", nil, nil, hit, nil, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, none(character.MasteryNoneNotMeleeAttack)},
		{"Cleave, used", "FR-077", "unhappy (state error)", "greataxe", nil, nil, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}.RecordCleaveAttack(), none(character.MasteryNoneAlreadyUsedThisTurn)},
		{"Cleave, miss then ranged", "FR-072", "unhappy (rule interaction)", "greataxe", nil, nil, miss, thrown, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Cleave, ranged and used", "FR-072", "unhappy (rule interaction)", "greataxe", nil, nil, hit, thrown, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}.RecordCleaveAttack(), none(character.MasteryNoneNotMeleeAttack)},
		{"Graze, miss below AC", "FR-078", "unhappy (boundary)", "greatsword", nil, nil, miss, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, character.GrazeOffer{Amount: 3, DamageType: content.DamageSlashing}},
		{"Graze, natural 1", "FR-078", "unhappy (boundary)", "greatsword", nil, nil, nat1, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, character.GrazeOffer{Amount: 3, DamageType: content.DamageSlashing}},
		{"Graze, modifier 0", "FR-078", "unhappy (boundary)", "greatsword", nil, nil, miss, melee, 0, 2, content.SizeMedium, 0, character.WeaponTurnState{}, character.GrazeOffer{Amount: 0, DamageType: content.DamageSlashing}},
		{"Graze, modifier -1", "FR-078", "unhappy (boundary)", "greatsword", nil, nil, miss, melee, -1, 2, content.SizeMedium, 0, character.WeaponTurnState{}, character.GrazeOffer{Amount: 0, DamageType: content.DamageSlashing}},
		{"Graze, hit", "FR-079", "unhappy (rule interaction)", "greatsword", nil, nil, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Graze, Critical Hit", "FR-079", "unhappy (boundary)", "greatsword", nil, nil, crit, melee, 3, 2, content.SizeMedium, 9, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Graze, vacated location", "FR-079", "unhappy (rule interaction)", "greatsword", nil, nil, forced, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Graze, natural 20 at a vacated location", "FR-079, FR-118", "unhappy (rule interaction)", "greatsword", nil, nil, forced20, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Graze, underwater automatic miss", "FR-119", "unhappy (rule interaction)", "greatsword", nil, nil, unrolled, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Push, Large", "FR-082", "unhappy (boundary)", "pike", nil, nil, hit, melee, 3, 2, content.SizeLarge, 5, character.WeaponTurnState{}, nil},
		{"Push, Tiny", "FR-082", "unhappy (boundary)", "pike", nil, nil, hit, melee, 3, 2, content.SizeTiny, 5, character.WeaponTurnState{}, nil},
		{"Push, Huge", "FR-083", "unhappy (boundary)", "pike", nil, nil, hit, melee, 3, 2, content.SizeHuge, 5, character.WeaponTurnState{}, none(character.MasteryNoneTargetTooLarge)},
		{"Push, Gargantuan", "FR-083", "unhappy (boundary)", "pike", nil, nil, hit, melee, 3, 2, content.SizeGargantuan, 5, character.WeaponTurnState{}, none(character.MasteryNoneTargetTooLarge)},
		{"Push, miss", "FR-083", "unhappy (rule interaction)", "pike", nil, nil, miss, melee, 3, 2, content.SizeHuge, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Push, Huge and 0 damage", "FR-072", "unhappy (rule interaction)", "pike", nil, nil, hit, melee, 3, 2, content.SizeHuge, 0, character.WeaponTurnState{}, none(character.MasteryNoneTargetTooLarge)},
		{"Sap, hit, no damage", "FR-085", "unhappy (boundary)", "mace", nil, nil, hit, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, character.SapOffer{}},
		{"Sap, miss", "FR-085", "unhappy (rule interaction)", "mace", nil, nil, miss, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Slow, damage", "FR-086", "unhappy (boundary)", "club", nil, nil, hit, melee, 3, 2, content.SizeMedium, 1, character.WeaponTurnState{}, character.SlowOffer{}},
		{"Slow, no damage", "FR-087", "unhappy (boundary)", "club", nil, nil, hit, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneNoDamageDealt)},
		{"Slow, miss and 0 damage", "FR-072", "unhappy (rule interaction)", "club", nil, nil, miss, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Topple, Gargantuan", "FR-090", "unhappy (boundary)", "maul", nil, nil, hit, melee, 3, 2, content.SizeGargantuan, 5, character.WeaponTurnState{}, nil},
		{"Topple, modifier -1", "FR-090", "unhappy (boundary)", "maul", nil, nil, hit, melee, -1, 2, content.SizeMedium, 5, character.WeaponTurnState{}, nil},
		{"Topple, high level", "FR-090", "unhappy (boundary)", "maul", nil, nil, crit, melee, 5, 6, content.SizeMedium, 5, character.WeaponTurnState{}, nil},
		{"Topple, miss", "FR-091", "unhappy (rule interaction)", "maul", nil, nil, miss, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Vex, damage", "FR-092", "happy", "shortsword", nil, nil, hit, melee, 3, 2, content.SizeMedium, 4, character.WeaponTurnState{}, character.VexOffer{}},
		{"Vex, no damage", "FR-093", "unhappy (boundary)", "shortsword", nil, nil, hit, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneNoDamageDealt)},
		{"Vex, thrown", "FR-073", "unhappy (rule interaction)", "handaxe", nil, nil, hit, thrown, 3, 2, content.SizeMedium, 4, character.WeaponTurnState{}, character.VexOffer{}},
		{"Vex, no roll", "FR-119", "unhappy (rule interaction)", "shortsword", nil, nil, unrolled, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Vex, Blowgun fixed damage", "FR-093", "unhappy (boundary)", "blowgun", nil, nil, hit, character.RangedAttackMode{}, 3, 2, content.SizeMedium, 1, character.WeaponTurnState{}, character.VexOffer{}},
		{"Nick", "FR-081", "unhappy (rule interaction)", "dagger", nil, nil, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, none(character.MasteryNoneAttackDidNotQualify)},
		{"Not chosen", "FR-004", "unhappy (state error)", "shortsword", usable("club"), nil, hit, melee, 3, 2, content.SizeMedium, 4, character.WeaponTurnState{}, none(character.MasteryNoneFeatureNotAvailable)},
		{"No feature", "FR-005", "unhappy (empty case)", "shortsword", character.NoMasteryFeature{}, nil, hit, melee, 3, 2, content.SizeMedium, 4, character.WeaponTurnState{}, none(character.MasteryNoneFeatureNotAvailable)},
		{"nil access", "FR-005", "unhappy (empty case)", "shortsword", nilAccess{}, nil, hit, melee, 3, 2, content.SizeMedium, 4, character.WeaponTurnState{}, none(character.MasteryNoneFeatureNotAvailable)},
		{"Unusable wins", "FR-072", "unhappy (rule interaction)", "shortsword", usable("club"), nil, miss, melee, 3, 2, content.SizeMedium, 0, character.WeaponTurnState{}, none(character.MasteryNoneFeatureNotAvailable)},
		{"Tactical Master", "FR-006", "unhappy (rule interaction)", "longsword", nil, character.ReplacedMastery{Property: content.MasterySlow}, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, character.SlowOffer{}},
		{"Tactical Master, not chosen", "FR-006", "unhappy (state error)", "longsword", usable("club"), character.ReplacedMastery{Property: content.MasterySlow}, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, none(character.MasteryNoneFeatureNotAvailable)},
		{"own mastery in place", "FR-003", "happy", "longsword", nil, character.WeaponsOwnMastery{}, hit, melee, 3, 2, content.SizeMedium, 5, character.WeaponTurnState{}, character.SapOffer{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			access := c.access
			if access == nil {
				access = usable(c.weapon)
			}
			if _, isNil := access.(nilAccess); isNil {
				access = nil
			}
			facts := character.MasteryAttackFacts{
				WeaponID: content.ItemID(c.weapon), Weapon: ws[c.weapon], Access: access, InPlay: c.inPlay, Outcome: c.outcome(t), Mode: c.mode,
				AbilityModifier: modifierOf(t, c.mod), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1 + 4*(c.bonus-2))),
				TargetSize: c.size, DamageDealt: c.damage, TurnState: c.turn,
			}
			got, err := character.OfferMastery(facts)
			if err != nil {
				t.Fatal(err)
			}
			want := c.want
			switch c.name {
			case "Push, Large", "Push, Tiny":
				want = mustPush(t, got)
			case "Topple, Gargantuan":
				want = toppleOf(t, got, 13)
			case "Topple, modifier -1":
				want = toppleOf(t, got, 9)
			case "Topple, high level":
				want = toppleOf(t, got, 19)
			}
			if !offersEqual(got, want) {
				t.Errorf("offer = %#v, want %#v [%s, %s]", got, want, c.requirement, c.path)
			}
		})
	}
}

// nilAccess marks a case that passes a nil WeaponMasteryAccess.
type nilAccess struct{}

func (nilAccess) Permits(content.ItemID) bool { return true }

func mustPush(t *testing.T, got character.MasteryOffer) character.MasteryOffer {
	t.Helper()
	p, ok := got.(character.PushOffer)
	if !ok || p.MaximumFeet() != 10 {
		t.Fatalf("offer = %#v, want a PushOffer of 10 feet", got)
	}
	return p
}

func toppleOf(t *testing.T, got character.MasteryOffer, dc int) character.MasteryOffer {
	t.Helper()
	o, ok := got.(character.ToppleOffer)
	if !ok || o.Ability() != ability.Constitution || int(o.DC()) != dc {
		t.Fatalf("offer = %#v, want a Constitution ToppleOffer with DC %d", got, dc)
	}
	return o
}

func offersEqual(a, b character.MasteryOffer) bool {
	if ca, ok := a.(character.CleaveOffer); ok {
		cb, ok := b.(character.CleaveOffer)
		return ok && ca.WeaponID == cb.WeaponID && ca.Kind == cb.Kind && ca.DamageModifier == cb.DamageModifier && damageText(ca.Damage) == damageText(cb.Damage)
	}
	return a == b
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-107, FR-071 - a negative DamageDealt is an input error
// naming the field, with no offer, even without the feature.
// Path: unhappy (invalid input).
func TestOfferMasteryRejectsNegativeDamage(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	offer, err := character.OfferMastery(character.MasteryAttackFacts{WeaponID: "club", Weapon: ws["club"], DamageDealt: -1})
	if offer != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("offer = %v, error = %v", offer, err)
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-104, FR-106 - OfferMastery gives the same offer for the same
// input and does not alter the turn state.
// Path: unhappy (determinism, immutability).
func TestOfferMasteryIsDeterministic(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	turn := character.WeaponTurnState{}.RecordAttackAction()
	facts := character.MasteryAttackFacts{WeaponID: "greataxe", Weapon: ws["greataxe"], Access: character.MasteryFromChoice{Choice: choiceOf(t, "greataxe")}, Outcome: hit(t), Mode: character.MeleeAttackMode{}, TurnState: turn}
	first, _ := character.OfferMastery(facts)
	second, _ := character.OfferMastery(facts)
	if !offersEqual(first, second) || !turn.Equal(character.WeaponTurnState{}.RecordAttackAction()) {
		t.Error("OfferMastery changed state or varied")
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-076, table 4.4c (placement rows) - a second creature is
// accepted within 5 feet of the first and within reach, and rejected naming the rule broken.
// Path: per case.
func TestCleaveOfferValidateTarget(t *testing.T) {
	t.Parallel()
	offer := character.CleaveOffer{}
	cases := []struct {
		name                         string
		first, attacker, reach, want int
		path                         string
	}{
		{"inside both", 5, 5, 5, 0, "happy"},
		{"6 from the first", 6, 5, 5, int(character.CleaveNotWithinFiveFeetOfFirst), "unhappy (boundary)"},
		{"6 from the attacker", 5, 6, 5, int(character.CleaveNotWithinReach), "unhappy (boundary)"},
		{"Reach weapon", 5, 10, 10, 0, "unhappy (boundary)"},
		{"Reach weapon, beyond", 5, 11, 10, int(character.CleaveNotWithinReach), "unhappy (boundary)"},
		{"both fail", 6, 6, 5, int(character.CleaveBothRulesBroken), "unhappy (rule interaction)"},
		{"default reach", 0, 5, 0, 0, "unhappy (empty case)"},
	}
	for _, c := range cases {
		res := offer.ValidateTarget(character.MustCleaveTargetPlacement(c.first, c.attacker), reachOf(c.reach))
		switch r := res.(type) {
		case character.CleaveTargetAccepted:
			if c.want != 0 {
				t.Errorf("%s: accepted, want failure %d", c.name, c.want)
			}
		case character.CleaveTargetRejected:
			if int(r.Failure) != c.want {
				t.Errorf("%s: failure %d, want %d", c.name, r.Failure, c.want)
			}
		}
	}
	_, err := character.NewCleaveTargetPlacement(-1, 5)
	wantInvalid(t, err, "first -1")
	_, err = character.NewCleaveTargetPlacement(5, -1)
	wantInvalid(t, err, "attacker -1")
	defer func() {
		if recover() == nil {
			t.Error("MustCleaveTargetPlacement did not panic")
		}
	}()
	character.MustCleaveTargetPlacement(-1, -1)
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-084 - ChooseDistance accepts 0 to 10 and rejects -1 and 11
// as results, never an error or panic.
// Path: unhappy (boundary).
func TestPushOfferChooseDistance(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	o, _ := character.OfferMastery(character.MasteryAttackFacts{WeaponID: "pike", Weapon: ws["pike"], Access: character.MasteryFromChoice{Choice: choiceOf(t, "pike")}, Outcome: hit(t), TargetSize: content.SizeLarge})
	push := o.(character.PushOffer)
	for feet, want := range map[int]character.PushDistanceResult{
		-1: character.PushDistanceRejected{Reason: character.PushDistanceNegative}, 0: character.PushDistanceAccepted{Movement: 0},
		10: character.PushDistanceAccepted{Movement: 10}, 11: character.PushDistanceRejected{Reason: character.PushDistanceAboveMaximum},
	} {
		if got := push.ChooseDistance(feet); got != want {
			t.Errorf("ChooseDistance(%d) = %#v, want %#v", feet, got, want)
		}
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-085, FR-086, FR-092, FR-094 - the circumstance and the
// expiry of Sap, Slow and Vex are data.
// Path: happy.
func TestTimedOffersExposeCircumstancesAndExpiry(t *testing.T) {
	t.Parallel()
	if _, ok := (character.SapOffer{}).Circumstance().(modifier.Disadvantage); !ok {
		t.Error("Sap must give Disadvantage")
	}
	if _, ok := (character.VexOffer{}).Circumstance().(modifier.Advantage); !ok {
		t.Error("Vex must give Advantage")
	}
	start, end, next := character.ExpiryStartOfAttackersNextTurn, character.ExpiryEndOfAttackersNextTurn, character.ExpiryWhenNextAttackRollMade
	for name, c := range map[string]struct {
		got, want []character.MasteryEffectExpiry
	}{
		"Sap":  {(character.SapOffer{}).Expiries(), []character.MasteryEffectExpiry{start, next}},
		"Slow": {(character.SlowOffer{}).Expiries(), []character.MasteryEffectExpiry{start}},
		"Vex":  {(character.VexOffer{}).Expiries(), []character.MasteryEffectExpiry{end, next}},
	} {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("%s expiries = %v, want %v", name, c.got, c.want)
		}
	}
	e := (character.SapOffer{}).Expiries()
	e[0] = 0
	if (character.SapOffer{}).Expiries()[0] != start {
		t.Error("Expiries exposes the offer")
	}
	for e, want := range map[character.MasteryEffectExpiry]string{start: "startOfAttackersNextTurn", end: "endOfAttackersNextTurn", next: "whenNextAttackRollMade", 0: "unknown"} {
		if e.String() != want {
			t.Errorf("%d = %q", int(e), e.String())
		}
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-088, FR-089, table 4.4c (Slow rows) - reductions combine
// without exceeding 10 feet and ApplyTo subtracts down to 0.
// Path: per case.
func TestSpeedReduction(t *testing.T) {
	t.Parallel()
	ten := (character.SlowOffer{}).Reduction()
	if ten.Feet() != 10 || ten.Combine(ten).Feet() != 10 || ten.Combine(ten).Combine(ten).Feet() != 10 || (character.SpeedReduction{}).Combine(ten) != ten || (character.SpeedReduction{}).Feet() != 0 {
		t.Error("combination must stay at 10 feet")
	}
	for speed, want := range map[int]int{30: 20, 5: 0, 0: 0} {
		got, err := ten.ApplyTo(speed)
		if err != nil || got != want {
			t.Errorf("ApplyTo(%d) = %d, %v; want %d", speed, got, err, want)
		}
	}
	_, err := ten.ApplyTo(-1)
	wantInvalid(t, err, "speed -1")
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-072 - the reason names.
// Path: happy.
func TestMasteryNoneReasonString(t *testing.T) {
	t.Parallel()
	names := []string{"featureNotAvailable", "attackDidNotQualify", "notMeleeAttack", "alreadyUsedThisTurn", "targetTooLarge", "noDamageDealt"}
	for i, want := range names {
		if got := character.MasteryNoneReason(i + 1).String(); got != want {
			t.Errorf("reason %d = %q, want %q", i+1, got, want)
		}
	}
	if character.MasteryNoneReason(0).String() != "unknown" || character.MasteryNoneReason(7).String() != "unknown" {
		t.Error("out of range reasons must be unknown")
	}
}
