package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-022 - a Command Word works signed, and spoken only where sound is audible).
// Path: per case.
func TestCommandWordAttempt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		d      character.CommandWordDelivery
		env    character.SoundEnvironment
		accept bool
	}{
		{character.DeliverySpoken, character.SoundAudible, true},
		{character.DeliverySpoken, character.SoundSuppressed, false},
		{character.DeliverySigned, character.SoundAudible, true},
		{character.DeliverySigned, character.SoundSuppressed, true},
	}
	for _, c := range cases {
		got, err := character.CommandWord{}.Attempt(c.d, c.env)
		_, ok := got.(character.MagicItemCommandWordAccepted)
		if err != nil || ok != c.accept {
			t.Errorf("Attempt(%v, %v) = %#v, %v (SRD 5.2.1, \"Command Word\", p. 206)", c.d, c.env, got, err)
		}
		if f, failed := got.(character.MagicItemCommandWordFailed); failed && f.Environment != c.env {
			t.Errorf("failed environment = %v", f.Environment)
		}
	}
	for _, in := range [][2]int{{0, 1}, {1, 0}, {3, 1}, {1, 3}} {
		if _, err := (character.CommandWord{}).Attempt(character.CommandWordDelivery(in[0]), character.SoundEnvironment(in[1])); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Attempt(%v): %v", in, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-023, FR-024 - using a consumable consumes it and raises MagicItemConsumed;
// using it again is rejected; a non-consumable stays intact).
// Path: per case.
func TestMagicItemUse(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Consumable Items", p. 206`
	cases := []struct {
		name       string
		consumable bool
		state      character.MagicItemUseState
		want       character.MagicItemUseState // 0 means MagicItemAlreadyConsumed
		events     int
	}{
		{"consumable", true, character.ItemIntact, character.ItemConsumed, 1},
		{"not consumable", false, character.ItemIntact, character.ItemIntact, 0},
		{"consumed again", true, character.ItemConsumed, 0, 0},
		{"non-consumable that was consumed", false, character.ItemConsumed, 0, 0},
	}
	for _, c := range cases {
		got, err := character.MagicItemUse{}.Use(holder, inst("p1"), c.consumable, c.state)
		if err != nil {
			t.Fatal(err)
		}
		switch r := got.(type) {
		case character.MagicItemUseApplied:
			ev := r.Events()
			if c.want == 0 || r.State != c.want || len(ev) != c.events {
				t.Errorf("%s: %+v [%s]", c.name, r, p)
			}
			if c.events == 1 {
				if e, ok := ev[0].(character.MagicItemConsumed); !ok || e.CharacterID() != holder || e.InstanceID() != "p1" {
					t.Errorf("%s: event %+v", c.name, ev[0])
				}
			}
		case character.MagicItemAlreadyConsumed:
			if c.want != 0 {
				t.Errorf("%s: already consumed", c.name)
			}
		}
	}
	for name, run := range map[string]func() error{
		"zero holder": func() error {
			_, e := character.MagicItemUse{}.Use(character.Identifier{}, inst("p"), true, character.ItemIntact)
			return e
		},
		"blank instance": func() error { _, e := character.MagicItemUse{}.Use(holder, "", true, character.ItemIntact); return e },
		"invalid state":  func() error { _, e := character.MagicItemUse{}.Use(holder, inst("p"), true, 0); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-008, FR-009 - ammunition is magical only for a magic weapon with the Ammunition
// property; a scroll can be read when a written language is understood and the description does not restrict reading.
// Path: per case.
func TestAmmunitionMagicAndScrollReading(t *testing.T) {
	t.Parallel()
	for _, c := range [][3]bool{{true, true, true}, {true, false, false}, {false, true, false}, {false, false, false}} {
		if got := (character.AmmunitionMagic{}).IsMagical(c[0], c[1]); got != c[2] {
			t.Errorf("IsMagical(%v, %v) = %v (SRD 5.2.1, \"Weapons\", p. 205)", c[0], c[1], got)
		}
	}
	for _, c := range []struct {
		understands, restricts bool
		readable               bool
	}{{true, false, true}, {false, false, false}, {true, true, false}, {false, true, false}} {
		got := character.ScrollReading{}.CanAttempt(c.understands, c.restricts)
		_, ok := got.(character.MagicItemScrollReadable)
		if ok != c.readable {
			t.Errorf("CanAttempt(%v, %v) = %#v (SRD 5.2.1, \"Using a Scroll\", p. 205)", c.understands, c.restricts, got)
		}
		if n, bad := got.(character.MagicItemScrollNotReadable); bad && n.Restricted != c.restricts {
			t.Errorf("Restricted = %v", n.Restricted)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-025, FR-026 - SpellFromItem holds the spell, the level it is cast at (not below the
// lowest), the lowest caster level, the defaults, and needs Concentration exactly when cast from the item for a Concentration spell.
// Path: per case.
func TestSpellFromItem(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Spells Cast from Items", p. 206`
	l1, l3 := levelOf(t, 1), levelOf(t, 3)
	cases := []struct {
		name, requirement, path string
		castAt                  content.SpellLevel
		delivery                character.SpellDelivery
		concentration           bool
		wantConcentrate         bool
	}{
		{"cast from item, Concentration spell", p, "unhappy (rule interaction)", l1, character.CastFromItem, true, true},
		{"cast from item, no Concentration", p, "happy", l1, character.CastFromItem, false, false},
		{"bypasses casting, Concentration spell", p, "unhappy (rule interaction)", l1, character.BypassesCasting, true, false},
		{"charge-to-level item casts higher", p, "happy", l3, character.CastFromItem, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := character.NewSpellFromItem("magic_missile", l1, c.castAt, character.NoComponentsRequired, character.StandardCastingTerms, c.delivery, c.concentration)
			if err != nil || s.UserMustConcentrate() != c.wantConcentrate || s.SpellID() != "magic_missile" || s.CastAtLevel() != c.castAt || s.CasterLevel() != character.ItemCasterLevelLowest ||
				s.Components() != character.NoComponentsRequired || s.CastingTerms() != character.StandardCastingTerms || s.Delivery() != c.delivery {
				t.Errorf("= %+v, %v [%s]", s, err, c.requirement)
			}
		})
	}
	bad := map[string]func() error{
		"below the lowest level": func() error {
			_, e := character.NewSpellFromItem("x", l3, l1, character.NoComponentsRequired, character.StandardCastingTerms, character.CastFromItem, false)
			return e
		},
		"invalid spell": func() error {
			_, e := character.NewSpellFromItem("", l1, l1, character.NoComponentsRequired, character.StandardCastingTerms, character.CastFromItem, false)
			return e
		},
		"invalid components": func() error {
			_, e := character.NewSpellFromItem("x", l1, l1, 0, character.StandardCastingTerms, character.CastFromItem, false)
			return e
		},
		"invalid terms": func() error {
			_, e := character.NewSpellFromItem("x", l1, l1, character.NoComponentsRequired, 0, character.CastFromItem, false)
			return e
		},
		"invalid delivery": func() error {
			_, e := character.NewSpellFromItem("x", l1, l1, character.NoComponentsRequired, character.StandardCastingTerms, 0, false)
			return e
		},
	}
	for name, run := range bad {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	s := must(character.NewSpellFromItem("x", l1, l1, character.ComponentsPerDescription, character.CastingTermsPerDescription, character.CastFromItem, false))
	if s.Components() != character.ComponentsPerDescription || s.CastingTerms() != character.CastingTermsPerDescription {
		t.Errorf("per description: %+v", s)
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-027, FR-028 and table "Item spellcasting ability").
// Path: per case.
func TestItemSpellcastingAbilityResolve(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Spells Cast from Items", p. 206`
	ua := func(a ability.Ability, mod int) character.UserSpellAbility {
		return character.UserSpellAbility{Ability: a, Modifier: modifierOf(t, mod)}
	}
	pb := func(n int) character.ProficiencyBonus {
		switch n {
		case 2:
			return character.ProficiencyBonusFor(character.MustLevel(1))
		case 3:
			return character.ProficiencyBonusFor(character.MustLevel(5))
		}
		return character.ProficiencyBonusFor(character.MustLevel(17))
	}
	cases := []struct {
		name, requirement, path string
		user                    []character.UserSpellAbility
		choice                  ability.Ability
		pb                      int
		wantAbility             ability.Ability
		wantMod                 int
		notAvailable            bool
	}{
		{"one ability", p, "happy", []character.UserSpellAbility{ua(ability.Intelligence, 3)}, 0, 2, ability.Intelligence, 3, false},
		{"one ability, choice ignored", p, "happy", []character.UserSpellAbility{ua(ability.Intelligence, 3)}, ability.Wisdom, 2, ability.Intelligence, 3, false},
		{"two abilities, user chooses", p, "happy", []character.UserSpellAbility{ua(ability.Wisdom, 2), ua(ability.Charisma, 4)}, ability.Charisma, 3, ability.Charisma, 4, false},
		{"no ability", p, "unhappy (empty case)", nil, 0, 2, 0, 0, false},
		{"no ability, choice ignored", p, "unhappy (empty case)", nil, ability.Strength, 2, 0, 0, false},
		{"no ability, higher Proficiency Bonus", p, "unhappy (boundary)", nil, 0, 6, 0, 0, false},
		{"choice not among the user's", p, "unhappy (invalid input)", []character.UserSpellAbility{ua(ability.Wisdom, 2), ua(ability.Charisma, 4)}, ability.Intelligence, 2, 0, 0, true},
		{"two abilities, no choice", p, "unhappy (invalid input)", []character.UserSpellAbility{ua(ability.Wisdom, 2), ua(ability.Charisma, 4)}, 0, 2, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.ItemSpellcastingAbility{}.Resolve(c.user, c.choice, pb(c.pb))
			if c.notAvailable {
				if got != (character.ItemSpellcastingAbilityNotAvailable{}) {
					t.Errorf("= %#v [%s]", got, c.requirement)
				}
				return
			}
			r, ok := got.(character.ItemSpellcastingResolved)
			if !ok || r.Ability != c.wantAbility || r.Modifier.Value() != c.wantMod || r.ProficiencyBonus.Value() != c.pb {
				t.Errorf("= %#v [%s]", got, c.requirement)
			}
		})
	}
}
