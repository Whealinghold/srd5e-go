package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-cleric.md FR-035, FR-036, FR-137, table 4.5.1 TB-01 to TB-11 (SRD 5.2.1, "Level 1: Divine Order", p. 37; "Ability Modifiers", p. 6) - a
// Thaumaturge adds its Wisdom modifier, at least +1, to Arcana and Religion checks only; another skill, a Protector and no order get nothing.
// Path: per case (TB-01 to TB-11): happy / unhappy (boundary: minimum, lowest and highest score).
func TestThaumaturgeCheckBonus(t *testing.T) {
	t.Parallel()
	th, pr := character.DivineOrderThaumaturge, character.DivineOrderProtector
	for _, c := range []struct {
		label string
		order character.DivineOrder
		wis   int
		skill content.Skill
		want  int
		some  bool
	}{
		{"TB-01", th, 2, content.SkillArcana, 2, true}, {"TB-02", th, 2, content.SkillReligion, 2, true}, {"TB-03", th, 1, content.SkillArcana, 1, true},
		{"TB-04", th, 0, content.SkillReligion, 1, true}, {"TB-05", th, -1, content.SkillArcana, 1, true}, {"TB-06", th, -5, content.SkillReligion, 1, true},
		{"TB-07", th, 5, content.SkillArcana, 5, true}, {"TB-08", th, 10, content.SkillReligion, 10, true}, {"TB-09", th, 2, content.SkillHistory, 0, false},
		{"TB-10", th, 2, content.SkillInvestigation, 0, false}, {"TB-11", pr, 2, content.SkillArcana, 0, false}, {"no order", 0, 2, content.SkillArcana, 0, false},
	} {
		got, ok := character.ThaumaturgeCheckBonus(c.order, modifierOf(t, c.wis), c.skill)
		if ok != c.some || got != modifier.New(c.want) {
			t.Errorf("%s: bonus = %v, %v; want %d, %v", c.label, got, ok, c.want, c.some)
		}
	}
}

// Requirement: doc/specs/class-cleric.md FR-031, FR-032 - the Protector grants Martial weapon proficiency and Heavy armor training; the Thaumaturge and no order grant neither.
// Path: per case (unhappy: rule interaction).
func TestDivineOrderGrants(t *testing.T) {
	t.Parallel()
	if g := character.DivineOrderProtector.Grants(); !g.MartialWeapons || !g.HeavyArmor {
		t.Errorf("Protector grants %+v", g)
	}
	for _, o := range []character.DivineOrder{character.DivineOrderThaumaturge, 0} {
		if g := o.Grants(); g != (character.DivineOrderGrants{}) {
			t.Errorf("%v grants %+v", o, g)
		}
	}
}

func clcLight(t *testing.T) content.Spell {
	t.Helper()
	return clcSpell(t, "light", 0, []string{"cleric", "wizard"}, "")
}

// Requirement: doc/specs/class-cleric.md FR-022, FR-029, FR-030, FR-033, FR-034, FR-037, FR-128, FR-141 (SRD 5.2.1, "Level 1: Divine Order", p. 37) - the choice accepts
// exactly Protector or Thaumaturge; each refusal carries the first reason in the order already_chosen, order_missing, both_orders, order_unknown, extra_cantrip_missing,
// extra_cantrip_not_cleric_cantrip, extra_cantrip_already_known; a Protector ignores any extra cantrip; the extra-cantrip allowance is 1 for a Thaumaturge and 0 for a Protector;
// the choice is stored, works on a multiclass Cleric and leaves the input unchanged.
// Path: per case: happy / unhappy (invalid input, state error, rule interaction).
func TestChooseDivineOrder(t *testing.T) {
	t.Parallel()
	light := clcLight(t)
	level1 := clcSpell(t, "bless", 1, []string{"cleric"}, "")
	offList := clcSpell(t, "minor_illusion", 0, []string{"wizard"}, "")
	already := clcWithChoices(t, clcCleric(t, 1), must(character.NewClericChoices(character.DivineOrderProtector, "", 0)))
	for _, c := range []struct {
		name   string
		char   character.Character
		names  []string
		extra  *content.Spell
		known  []content.SpellID
		reason character.ClericReason
		order  character.DivineOrder
	}{
		{"Protector", clcCleric(t, 1), []string{"protector"}, nil, nil, 0, character.DivineOrderProtector},
		{"Protector ignores an extra cantrip", clcCleric(t, 1), []string{"protector"}, &level1, nil, 0, character.DivineOrderProtector},
		{"Thaumaturge", clcCleric(t, 1), []string{"thaumaturge"}, &light, nil, 0, character.DivineOrderThaumaturge},
		{"multiclass Fighter 3 / Cleric 1", clcChar(t, cl("fighter", 3), cl("cleric", 1)), []string{"protector"}, nil, nil, 0, character.DivineOrderProtector},
		{"RJ-12 already chosen, both named", already, []string{"protector", "thaumaturge"}, nil, nil, character.ClericAlreadyChosen, 0},
		{"none named", clcCleric(t, 1), nil, nil, nil, character.ClericOrderMissing, 0},
		{"both named", clcCleric(t, 1), []string{"protector", "thaumaturge"}, &light, nil, character.ClericBothOrders, 0},
		{"unknown name", clcCleric(t, 1), []string{"warden"}, nil, nil, character.ClericOrderUnknown, 0},
		{"Thaumaturge without a cantrip", clcCleric(t, 1), []string{"thaumaturge"}, nil, nil, character.ClericExtraCantripMissing, 0},
		{"extra cantrip is level 1", clcCleric(t, 1), []string{"thaumaturge"}, &level1, nil, character.ClericExtraCantripNotClericCantrip, 0},
		{"extra cantrip off the list", clcCleric(t, 1), []string{"thaumaturge"}, &offList, nil, character.ClericExtraCantripNotClericCantrip, 0},
		{"extra cantrip already known", clcCleric(t, 1), []string{"thaumaturge"}, &light, []content.SpellID{"light"}, character.ClericExtraCantripAlreadyKnown, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := character.ChooseDivineOrder(c.char, c.names, c.extra, c.known)
			if err != nil {
				t.Fatal(err)
			}
			if c.reason != 0 {
				if r, ok := res.(character.ChooseDivineOrderRejected); !ok || r.Reason != c.reason {
					t.Errorf("result = %#v, want rejected %v", res, c.reason)
				}
				return
			}
			r, ok := res.(character.ChooseDivineOrderChosen)
			if !ok || r.Choices().DivineOrder() != c.order {
				t.Fatalf("result = %#v", res)
			}
			wantExtra := 0
			if c.order == character.DivineOrderThaumaturge {
				wantExtra = 1
			}
			stored, _, _ := character.ReadFeatureState(r.Character().FeatureStates(), character.ClericChoicesKey())
			ev, isEvent := r.Events()[0].(character.ClericOptionChosen)
			if r.ExtraCantrips() != wantExtra || (wantExtra == 1) != (r.ExtraCantrip() == "light") || r.Grants() != c.order.Grants() || stored != r.Choices() || !isEvent || ev.Option() != c.order.String() || ev.Feature() != "divine_order" {
				t.Errorf("chosen = %+v, stored %+v, events %v", r, stored, r.Events())
			}
			if _, held, _ := character.ReadFeatureState(c.char.FeatureStates(), character.ClericChoicesKey()); held {
				t.Error("the input Character was altered")
			}
			r.Events()[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-122, FR-037 - a character with no Cleric levels is ChooseDivineOrderNotACleric; a zero Character and a foreign type of state under the
// key are errors matching srderr.ErrInvalidInput.
// Path: per case (unhappy: state error, invalid input).
func TestChooseDivineOrderNotAClericAndInvalid(t *testing.T) {
	t.Parallel()
	res, err := character.ChooseDivineOrder(fgtFighter(t, 3), []string{"protector"}, nil, nil)
	if _, ok := res.(character.ChooseDivineOrderNotACleric); err != nil || !ok {
		t.Errorf("Fighter = %#v, %v", res, err)
	}
	_, err = character.ChooseDivineOrder(character.Character{}, []string{"protector"}, nil, nil)
	wantInvalid(t, err, "zero character")
	foreign := must(character.WithFeatureState(clcCleric(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_choices"), must(character.NewChosenSubclass("life_domain"))))
	_, err = character.ChooseDivineOrder(foreign, []string{"protector"}, nil, nil)
	wantInvalid(t, err, "foreign state")
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Error("not invalid input")
	}
}
