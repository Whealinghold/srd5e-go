package character_test

import (
	"math"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/wish-spell.md FR-014, E29 to E31 - a Wish statement is any text of 1 to 2,000 characters that is not blank;
// the empty, blank and 2,001-character statements are an *InvalidError, and 2,000 characters (counted as characters, not bytes) are accepted.
// Path: per case.
func TestNewWishStatement(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, text string
		invalid    bool
	}{
		{"text", "I wish for a boon", false},
		{"2,000 characters", strings.Repeat("a", 2000), false},
		{"2,000 multibyte characters", strings.Repeat("é", 2000), false},
		{"empty", "", true},
		{"blank", "   \n\t", true},
		{"2,001 characters", strings.Repeat("a", 2001), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := character.NewWishStatement(c.text)
			if c.invalid {
				if !isInvalid(err) || !s.IsZero() {
					t.Errorf("= %v, %v, want invalid input", s, err)
				}
				return
			}
			if err != nil || s.String() != c.text || s.IsZero() {
				t.Errorf("= %v, %v", s.String(), err)
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-014, section 8 - NewWishStatement never panics and returns an error exactly for an empty,
// blank or over-long text.
// Path: unhappy (invalid input).
func FuzzNewWishStatement(f *testing.F) {
	for _, s := range []string{"", " ", "a", "\xff", strings.Repeat("x", 2001), strings.Repeat("é", 2000)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		s, err := character.NewWishStatement(in)
		want := strings.TrimSpace(in) == "" || len([]rune(in)) > 2000
		if (err != nil) != want || (err != nil && !isInvalid(err)) || (err == nil && s.String() != in) {
			t.Fatalf("NewWishStatement(%q) = %q, %v", in, s.String(), err)
		}
	})
}

// Requirement: per case.
// Path: per case.
func TestWishChoiceConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	perDay := content.MustCost(5, content.CoinGP, content.CostPerDay)
	creatures := wishCreatures(t, 2)
	held := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	cases := []struct {
		name, requirement, path string
		build                   func() error
	}{
		{"object cost per day", wishSpec + " FR-005, E34", "unhappy (invalid input)", func() error { _, e := character.NewObjectCreation(perDay, 10, false); return e }},
		{"object cost zero", wishSpec + " FR-005", "unhappy (invalid input)", func() error { _, e := character.NewObjectCreation(content.Cost{}, 10, false); return e }},
		{"object dimension 0 ft", wishSpec + " FR-007, E11", "unhappy (boundary)", func() error { _, e := character.NewObjectCreation(gpCost(1), 0, false); return e }},
		{"object dimension negative", wishSpec + " FR-007", "unhappy (invalid input)", func() error { _, e := character.NewObjectCreation(gpCost(1), -1, false); return e }},
		{"object dimension NaN", wishSpec + " FR-007, E35", "unhappy (invalid input)", func() error { _, e := character.NewObjectCreation(gpCost(1), math.NaN(), false); return e }},
		{"object dimension infinite", wishSpec + " FR-007", "unhappy (invalid input)", func() error { _, e := character.NewObjectCreation(gpCost(1), math.Inf(1), false); return e }},
		{"instant health negative", wishSpec + " FR-008, E36", "unhappy (invalid input)", func() error { _, e := character.NewInstantHealth(-1); return e }},
		{"resistance 0 creatures", wishSpec + " FR-009, E16", "unhappy (boundary)", func() error { _, e := character.NewResistanceGrant(content.DamageFire); return e }},
		{"resistance repeated creature", wishSpec + " FR-009", "unhappy (invalid input)", func() error {
			_, e := character.NewResistanceGrant(content.DamageFire, creatures[0], creatures[0])
			return e
		}},
		{"resistance zero creature", wishSpec + " FR-009", "unhappy (invalid input)", func() error {
			_, e := character.NewResistanceGrant(content.DamageFire, character.WishCreature{})
			return e
		}},
		{"resistance invalid damage type", wishSpec + " FR-009", "unhappy (invalid input)", func() error { _, e := character.NewResistanceGrant(0, creatures...); return e }},
		{"immunity 0 creatures", wishSpec + " FR-010, E37", "unhappy (boundary)", func() error {
			_, e := character.NewSpellImmunityGrant(character.ImmuneToSpell{Spell: "sleep"})
			return e
		}},
		{"immunity repeated creature", wishSpec + " FR-010, E37", "unhappy (invalid input)", func() error {
			_, e := character.NewSpellImmunityGrant(character.ImmuneToSpell{Spell: "sleep"}, creatures[1], creatures[1])
			return e
		}},
		{"immunity nil subject", wishSpec + " FR-010", "unhappy (invalid input)", func() error { _, e := character.NewSpellImmunityGrant(nil, creatures...); return e }},
		{"immunity to an invalid spell id", wishSpec + " FR-010", "unhappy (invalid input)", func() error {
			_, e := character.NewSpellImmunityGrant(character.ImmuneToSpell{}, creatures...)
			return e
		}},
		{"immunity to a zero effect", wishSpec + " FR-010", "unhappy (invalid input)", func() error {
			_, e := character.NewSpellImmunityGrant(character.ImmuneToOtherEffect{}, creatures...)
			return e
		}},
		{"duplicate without a spell", wishSpec + " FR-002", "unhappy (invalid input)", func() error { _, e := character.NewDuplicateSpell(content.Spell{}); return e }},
		{"roll redo undefined mode", wishSpec + " FR-013", "unhappy (invalid input)", func() error { _, e := character.NewRollRedo(modifier.RollMode(7), true); return e }},
		{"roll redo negative mode", wishSpec + " FR-013", "unhappy (invalid input)", func() error { _, e := character.NewRollRedo(modifier.RollMode(-1), true); return e }},
		{"reshape reality without a statement", wishSpec + " FR-014", "unhappy (invalid input)", func() error { _, e := character.NewReshapeReality(character.WishStatement{}); return e }},
		{"creature without an id", wishSpec + " FR-009", "unhappy (invalid input)", func() error { _, e := character.NewWishCreature(""); return e }},
		{"sudden learning, old feat invalid", wishSpec + " FR-012", "unhappy (invalid input)", func() error {
			_, e := character.NewSuddenLearning("", savageFeat(), character.NoFeatChoices{}, nil, character.HeldFeatures{}, nil, held)
			return e
		}},
		{"sudden learning, zero new feat", wishSpec + " FR-011", "unhappy (invalid input)", func() error {
			_, e := character.NewSuddenLearning("alert", content.Feat{}, character.NoFeatChoices{}, nil, character.HeldFeatures{}, nil, held)
			return e
		}},
		{"sudden learning, nil choices", wishSpec + " FR-011", "unhappy (invalid input)", func() error {
			_, e := character.NewSuddenLearning("alert", savageFeat(), nil, nil, character.HeldFeatures{}, nil, held)
			return e
		}},
		{"sudden learning, a required-by id that is not held", wishSpec + " FR-012, E22b", "unhappy (invalid input)", func() error {
			_, e := character.NewSuddenLearning("alert", savageFeat(), character.NoFeatChoices{}, []content.FeatID{"defense"}, character.HeldFeatures{}, nil, held)
			return e
		}},
		{"unpreventable damage 0", wishSpec + " FR-044", "unhappy (boundary)", func() error { _, e := character.NewUnpreventableDamage(content.DamageNecrotic, 0); return e }},
		{"unpreventable damage of an invalid type", wishSpec + " FR-044", "unhappy (invalid input)", func() error { _, e := character.NewUnpreventableDamage(0, 3); return e }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if err := c.build(); !isInvalid(err) {
				t.Errorf("= %v, want invalid input [%s]", err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-007, section 8 - MustObjectCreation panics with the error NewObjectCreation returns and only then.
// Path: unhappy (invalid input).
func TestMustObjectCreation(t *testing.T) {
	t.Parallel()
	if o := character.MustObjectCreation(gpCost(10), 300, true); o.Value() != gpCost(10) || o.LargestDimensionFeet() != 300 || !o.MagicItem() {
		t.Errorf("= %+v", o)
	}
	defer func() {
		err, ok := recover().(error)
		if !ok || !isInvalid(err) {
			t.Errorf("panic = %v, want an invalid input error", err)
		}
	}()
	character.MustObjectCreation(gpCost(10), 0, false)
}

// Requirement: doc/specs/wish-spell.md FR-065 - creature lists, the required-by list and the Magic Initiate spell map are copied on
// construction and on read, so mutating the input or a returned slice leaves the choice unchanged.
// Path: unhappy (immutability).
func TestWishChoicesCopyTheirCollections(t *testing.T) {
	t.Parallel()
	in := wishCreatures(t, 2)
	want := append([]character.WishCreature(nil), in...)
	res := must(character.NewResistanceGrant(content.DamageCold, in...))
	imm := must(character.NewSpellImmunityGrant(character.ImmuneToSpell{Spell: "sleep"}, in...))
	in[0] = character.WishCreature{}
	for _, got := range [][]character.WishCreature{res.Creatures(), imm.Creatures()} {
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("creatures = %v, want %v", got, want)
		}
		got[0] = character.WishCreature{}
	}
	if res.Creatures()[0] != want[0] || imm.Creatures()[0] != want[0] {
		t.Error("a returned slice aliases the choice")
	}
	feats := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	req := []content.FeatID{"alert"}
	sl := must(character.NewSuddenLearning("alert", savageFeat(), character.NoFeatChoices{}, req, character.HeldFeatures{}, nil, feats))
	req[0] = "x"
	if got := sl.FeatsRequiredByOthers(); len(got) != 1 || got[0] != "alert" {
		t.Errorf("required-by = %v", got)
	}
	sl.FeatsRequiredByOthers()[0] = "y"
	if got := sl.FeatsRequiredByOthers(); got[0] != "alert" || sl.OldFeat() != "alert" || sl.NewFeat().ID() != "savage_attacker" {
		t.Errorf("sudden learning changed: %v", got)
	}
}

// Requirement: doc/specs/wish-spell.md FR-001, FR-008, FR-009, FR-010 (SRD 5.2.1, "Wish", p. 175) - the Wish spell is "wish" at level 9, and the
// handoffs carry their fixed facts: Immunity lasts 8 hours, Resistance is permanent, healing names Greater Restoration.
// Path: happy.
func TestWishHandoffFixedFacts(t *testing.T) {
	t.Parallel()
	if character.WishSpellID() != "wish" || character.WishLevel() != 9 {
		t.Errorf("Wish = %v at %v", character.WishSpellID(), character.WishLevel())
	}
	if (character.WishSpellImmunity{}).DurationHours() != 8 || !(character.WishResistance{}).Permanent() || (character.WishInstantHealth{}).EndsEffectsOf() != "greater_restoration" {
		t.Error("a fixed fact of a handoff changed")
	}
	d := must(character.NewUnpreventableDamage(content.DamageNecrotic, 17))
	if d.Type() != content.DamageNecrotic || d.Amount() != 17 {
		t.Errorf("damage = %+v", d)
	}
	if c := must(character.NewWishCreature("pc-2")); c.String() != "pc-2" {
		t.Errorf("creature = %v", c)
	}
	if h := must(character.NewInstantHealth(3)); h.OtherCreatures() != 3 {
		t.Errorf("health = %+v", h)
	}
	if r := must(character.NewRollRedo(modifier.WithAdvantage, true)); r.Mode() != modifier.WithAdvantage || !r.WithinLastRound() {
		t.Errorf("redo = %+v", r)
	}
	if r := must(character.NewReshapeReality(statement(t, "x"))); r.Statement().String() != "x" {
		t.Errorf("reshape = %+v", r)
	}
	if d := must(character.NewDuplicateSpell(castSpell(t, spellID(2, 1), 2))); d.Spell().ID() != spellID(2, 1) {
		t.Errorf("duplicate = %+v", d)
	}
	g := must(character.NewSpellImmunityGrant(character.ImmuneToOtherEffect{Effect: statement(t, "a curse")}, wishCreatures(t, 1)...))
	if _, ok := g.Subject().(character.ImmuneToOtherEffect); !ok {
		t.Errorf("subject = %v", g.Subject())
	}
}
