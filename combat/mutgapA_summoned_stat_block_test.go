package combat

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-036 - the summoner's save DC and Proficiency Bonus are accepted at their
// lowest allowed value, 1, and refused at 0; the spell attack bonus and the modifier may be 0.
// Path: unhappy (boundary).
func TestSummonerSpellFactsAcceptTheLowestSaveDCAndProficiencyBonus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                                            string
		attackBonus, saveDC, modifier, proficiencyBonus int
		ok                                              bool
	}{
		{"save DC and Proficiency Bonus of 1", 0, 1, 0, 1, true},
		{"save DC of 0", 0, 0, 0, 1, false},
		{"Proficiency Bonus of 0", 0, 1, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f, err := NewSummonerSpellFacts(c.attackBonus, c.saveDC, c.modifier, c.proficiencyBonus)
			if c.ok {
				if err != nil || f.IsZero() || f.SaveDC() != c.saveDC || f.ProficiencyBonus() != c.proficiencyBonus {
					t.Fatalf("NewSummonerSpellFacts = %+v, %v, want the facts and no error", f, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !f.IsZero() {
				t.Fatalf("NewSummonerSpellFacts = %+v, %v, want the zero facts and ErrInvalidInput", f, err)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-032 to FR-034 - exactly the Venomous Spew, Fell Glare, Fey Step and
// Healing Touch entries are taken with a Bonus Action; no other kind and no undefined value is.
// Path: per case (every kind).
func TestOnlyTheFourBonusActionKindsAreBonusActions(t *testing.T) {
	t.Parallel()
	bonus := map[SummonedAttackKind]bool{
		SummonedAttackVenomousSpew: true, SummonedAttackFellGlare: true, SummonedAttackFeyStep: true, SummonedAttackHealingTouch: true,
	}
	for k := SummonedAttackKind(0); k <= SummonedAttackHealingTouch+1; k++ {
		if got := k.IsBonusAction(); got != bonus[k] {
			t.Errorf("%v.IsBonusAction() = %v, want %v", k, got, bonus[k])
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008 - a built stat block is not the zero value and the zero value is.
// Path: happy and unhappy (zero value).
func TestStatBlockIsZeroOnlyForTheZeroValue(t *testing.T) {
	t.Parallel()
	block, rej, err := NewGiantInsectStatBlock(InsectWasp, 4, MustSummonerSpellFacts(5, 14, 3, 2))
	if rej != nil || err != nil {
		t.Fatalf("NewGiantInsectStatBlock = %v, %v", rej, err)
	}
	if block.IsZero() {
		t.Error("a built stat block reports IsZero")
	}
	if !(SummonedStatBlock{}).IsZero() {
		t.Error("the zero stat block does not report IsZero")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008 - two stat blocks are Equal only when every number and list is the
// same; a change of any one field, up or down, makes them differ, in either order of comparison.
// Path: per case (one field at a time).
func TestStatBlockEqualComparesEveryField(t *testing.T) {
	t.Parallel()
	d6, d8 := dice.MustExpression(2, dice.D6(), 1), dice.MustExpression(2, dice.D8(), 1)
	attack := SummonedAttack{kind: SummonedAttackSlam, attackBonus: 5, damage: d6, damageType: content.DamageForce, rangeFeet: 5}
	base := SummonedStatBlock{
		monster: "m1", size: content.SizeMedium, typ: content.CreatureTypeBeast, armorClass: 12, hitPointMaximum: 20, hitDice: d6,
		walk: 30, climb: 20, fly: 10, swim: 5, resistances: []content.DamageType{content.DamageFire},
		conditionImmunities: []content.ConditionKind{content.ConditionCharmed}, attacks: []SummonedAttack{attack},
		multiattack: 2, proficiencyBonus: 3, experience: 100, forbidden: []content.SpellID{"simulacrum"}, telepathy: MustTelepathic(5280),
	}
	if !base.Equal(base) {
		t.Fatal("a stat block is not Equal to itself")
	}
	edits := map[string]func(*SummonedStatBlock){
		"monster":            func(b *SummonedStatBlock) { b.monster = "m2" },
		"size":               func(b *SummonedStatBlock) { b.size = content.SizeLarge },
		"type":               func(b *SummonedStatBlock) { b.typ = content.CreatureTypeFiend },
		"armor class":        func(b *SummonedStatBlock) { b.armorClass = 13 },
		"hit point maximum":  func(b *SummonedStatBlock) { b.hitPointMaximum = 21 },
		"hit dice":           func(b *SummonedStatBlock) { b.hitDice = d8 },
		"walk":               func(b *SummonedStatBlock) { b.walk = 31 },
		"climb":              func(b *SummonedStatBlock) { b.climb = 21 },
		"fly":                func(b *SummonedStatBlock) { b.fly = 11 },
		"swim":               func(b *SummonedStatBlock) { b.swim = 6 },
		"resistances":        func(b *SummonedStatBlock) { b.resistances = []content.DamageType{content.DamageCold} },
		"condition immunity": func(b *SummonedStatBlock) { b.conditionImmunities = nil },
		"attacks":            func(b *SummonedStatBlock) { b.attacks = []SummonedAttack{{kind: SummonedAttackRend}} },
		"multiattack":        func(b *SummonedStatBlock) { b.multiattack = 3 },
		"proficiency bonus":  func(b *SummonedStatBlock) { b.proficiencyBonus = 4 },
		"experience":         func(b *SummonedStatBlock) { b.experience = 101 },
		"forbidden spells":   func(b *SummonedStatBlock) { b.forbidden = nil },
		"cannot gain levels": func(b *SummonedStatBlock) { b.cannotGainLevels = true },
		"cannot take rests":  func(b *SummonedStatBlock) { b.cannotTakeRests = true },
		"life bond":          func(b *SummonedStatBlock) { b.lifeBond = true },
		"telepathy":          func(b *SummonedStatBlock) { b.telepathy = MustTelepathic(10) },
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			other := base
			edit(&other)
			if base.Equal(other) || other.Equal(base) {
				t.Errorf("a stat block that differs in %s is Equal", name)
			}
		})
	}
	// A change down is as visible as a change up.
	for name, edit := range map[string]func(*SummonedStatBlock){
		"monster": func(b *SummonedStatBlock) { b.monster = "m0" }, "size": func(b *SummonedStatBlock) { b.size = content.SizeSmall },
		"type": func(b *SummonedStatBlock) { b.typ = content.CreatureTypeAberration }, "armor class": func(b *SummonedStatBlock) { b.armorClass = 11 },
		"hit point maximum": func(b *SummonedStatBlock) { b.hitPointMaximum = 19 }, "walk": func(b *SummonedStatBlock) { b.walk = 29 },
		"climb": func(b *SummonedStatBlock) { b.climb = 19 }, "fly": func(b *SummonedStatBlock) { b.fly = 9 }, "swim": func(b *SummonedStatBlock) { b.swim = 4 },
		"multiattack": func(b *SummonedStatBlock) { b.multiattack = 1 }, "proficiency bonus": func(b *SummonedStatBlock) { b.proficiencyBonus = 2 },
		"experience": func(b *SummonedStatBlock) { b.experience = 99 },
	} {
		other := base
		edit(&other)
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("a stat block with a lower %s is Equal", name)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-060 - a simulacrum can copy a Beast or a Humanoid within 10 feet and
// no other creature type, whatever the type's position among the types; a Beast out of reach is refused too.
// Path: per case (every creature type).
func TestSimulacrumOriginalMustBeABeastOrHumanoidWithinTenFeet(t *testing.T) {
	t.Parallel()
	for ty := content.CreatureTypeAberration; ty.Valid(); ty++ {
		eligible := ty == content.CreatureTypeBeast || ty == content.CreatureTypeHumanoid
		block, rej, err := NewSimulacrumStatBlock(SimulacrumOriginal{Monster: "hero", Size: content.SizeMedium, Type: ty, HitPointMaximum: 20, WithinTenFeet: true})
		if err != nil {
			t.Fatalf("%v: NewSimulacrumStatBlock error = %v", ty, err)
		}
		if _, refused := rej.(OriginalNotEligibleRejection); refused == eligible || block.IsZero() != !eligible {
			t.Errorf("%v: block zero = %v, rejection = %v, want eligible = %v", ty, block.IsZero(), rej, eligible)
		}
	}
	_, rej, err := NewSimulacrumStatBlock(SimulacrumOriginal{Monster: "hero", Size: content.SizeMedium, Type: content.CreatureTypeBeast, HitPointMaximum: 20})
	if _, refused := rej.(OriginalNotEligibleRejection); !refused || err != nil {
		t.Errorf("a Beast out of reach: rejection = %v, error = %v, want OriginalNotEligibleRejection", rej, err)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-007 - the stat block of a monster takes the monster's first size, and
// Medium only when the monster names none.
// Path: unhappy (no size on the monster).
func TestStatBlockOfAMonsterWithNoSizeIsMedium(t *testing.T) {
	t.Parallel()
	if got := statBlockOfMonster(content.Monster{}, content.CreatureTypeBeast); got.size != content.SizeMedium || got.typ != content.CreatureTypeBeast {
		t.Errorf("statBlockOfMonster(zero Monster) = size %v, type %v, want Medium and Beast", got.size, got.typ)
	}
}
