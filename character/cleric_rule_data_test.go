package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/class-cleric.md FR-027, FR-094, FR-133 (SRD 5.2.1, "Level 1: Spellcasting", pp. 36-37; "Level 20: Greater Divine Intervention", p. 38) -
// the default rule data holds the recommended cantrips (Guidance, Sacred Flame, Thaumaturgy), the recommended level 1 spells (Bless, Cure Wounds, Guiding Bolt, Shield
// of Faith) and the one grant, Wish at level 20 with 2d4; the getters return copies.
// Path: happy / unhappy (immutability).
func TestDefaultClericRuleData(t *testing.T) {
	t.Parallel()
	d := character.DefaultClericRuleData()
	if !slices.Equal(d.RecommendedCantrips(), []content.SpellID{"guidance", "sacred_flame", "thaumaturgy"}) ||
		!slices.Equal(d.RecommendedLevel1Spells(), []content.SpellID{"bless", "cure_wounds", "guiding_bolt", "shield_of_faith"}) {
		t.Errorf("recommended = %v, %v", d.RecommendedCantrips(), d.RecommendedLevel1Spells())
	}
	grants := d.ExtraDivineInterventionGrants()
	if len(grants) != 1 || grants[0].Spell() != "wish" || grants[0].MinLevel() != 20 || grants[0].RechargeDice() != dice.MustExpression(2, dice.D4(), 0) {
		t.Fatalf("grants = %+v", grants)
	}
	d.RecommendedCantrips()[0] = "changed"
	d.RecommendedLevel1Spells()[0] = "changed"
	grants[0] = character.ExtraDivineInterventionGrant{}
	if d.RecommendedCantrips()[0] != "guidance" || d.RecommendedLevel1Spells()[0] != "bless" || d.ExtraDivineInterventionGrants()[0].Spell() != "wish" {
		t.Error("a getter returned shared storage")
	}
}

// Requirement: doc/specs/class-cleric.md FR-133 - the default ids that the bundle holds exist in the bundled content as spells; Guidance, Sacred Flame, Thaumaturgy, Guiding Bolt
// and Wish are Phase 5 data (D) and are named in the PR Testing entry; the default grants unlock Wish only at level 20.
// Path: happy / unhappy (missing content: D ids).
func TestDefaultClericRuleDataIDsExistInTheBundle(t *testing.T) {
	t.Parallel()
	src := must(data.NewBundledSource())
	d := character.DefaultClericRuleData()
	bundled := 0
	for _, id := range slices.Concat(d.RecommendedCantrips(), d.RecommendedLevel1Spells(), []content.SpellID{d.ExtraDivineInterventionGrants()[0].Spell()}) {
		res, err := src.Lookup(t.Context(), content.NewKey(content.KindSpell, content.MustID(string(id))))
		if err != nil {
			t.Fatal(err)
		}
		if f, ok := res.(content.LookupFound); ok {
			bundled++
			if !slices.Contains(f.Entry.(content.Spell).Classes(), character.ClericClassID) && id != "wish" {
				t.Errorf("%s is bundled but not a Cleric spell", id)
			}
		}
	}
	if bundled < 3 { // Bless, Cure Wounds, Shield of Faith
		t.Errorf("only %d default ids resolve in the bundle", bundled)
	}
}

// Requirement: doc/specs/class-cleric.md FR-094, FR-133 - NewClericRuleData and NewExtraDivineInterventionGrant accept house rules and refuse an invalid spell id, a minimum
// level outside 1 to 20, a dice count below 1 and the zero die.
// Path: per case (unhappy: invalid input, boundary).
func TestNewClericRuleDataRefusals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		spell   content.SpellID
		min, n  int
		die     dice.Die
		wantErr bool
	}{
		{"house grant", "plane_shift", 17, 1, dice.D6(), false}, {"level 1", "x", 1, 1, dice.D4(), false}, {"level 20", "x", 20, 1, dice.D4(), false},
		{"level 0", "x", 0, 1, dice.D4(), true}, {"level 21", "x", 21, 1, dice.D4(), true}, {"no dice", "x", 5, 0, dice.D4(), true},
		{"zero die", "x", 5, 1, dice.Die{}, true}, {"bad id", "Not Valid", 5, 1, dice.D4(), true},
	} {
		g, err := character.NewExtraDivineInterventionGrant(c.spell, c.min, c.n, c.die)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: error = %v", c.name, err)
		}
		if err != nil {
			wantInvalid(t, err, c.name)
			continue
		}
		rd, err := character.NewClericRuleData([]content.SpellID{"guidance"}, []content.SpellID{"bless"}, []character.ExtraDivineInterventionGrant{g})
		if err != nil || len(rd.ExtraDivineInterventionGrants()) != 1 || rd.ExtraDivineInterventionGrants()[0] != g {
			t.Errorf("%s: rule data = %+v, %v", c.name, rd, err)
		}
	}
	_, err := character.NewClericRuleData([]content.SpellID{"Not Valid"}, nil, nil)
	wantInvalid(t, err, "bad cantrip id")
	_, err = character.NewClericRuleData(nil, []content.SpellID{"Not Valid"}, nil)
	wantInvalid(t, err, "bad level 1 id")
	_, err = character.NewClericRuleData(nil, nil, []character.ExtraDivineInterventionGrant{{}})
	wantInvalid(t, err, "zero grant")
	if len(character.ClericRuleData{}.ExtraDivineInterventionGrants()) != 0 {
		t.Error("the zero rule data holds a grant")
	}
}
