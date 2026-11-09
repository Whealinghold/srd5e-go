package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-warlock.md FR-158 - the Warlock state round-trips through its parts and through the character's FeatureStates under the key named warlock; the invocations come back in the printed order with a Repeatable one once per time gained, arcanum spells by spell level, and the getters read every field.
// Path: happy.
func TestWarlockStateRoundTrips(t *testing.T) {
	t.Parallel()
	p := warFiend(17)
	parts := character.WarlockStateParts{
		Invocations: []content.FeatureOptionID{"witch_sight", "agonizing_blast", "eldritch_spear", "agonizing_blast"},
		Arcanum:     [4]content.SpellID{"circle_of_death", "", "", "power_word_kill"}, ArcanumCast: [4]bool{true, false, false, false},
		MagicalCunningUsed: true, ContactPatronUsed: true, GiftOfTheDepthsCast: true, GiftOfTheProtectorsSpent: true, HurlThroughHellSpent: true,
		DarkOnesOwnLuckSpent: 2, FiendishResilience: content.DamageFire,
	}
	s := warState(p, parts)
	if got, want := s.Invocations(), []content.FeatureOptionID{"agonizing_blast", "agonizing_blast", "eldritch_spear", "witch_sight"}; !slices.Equal(got, want) {
		t.Errorf("Invocations = %v, want %v", got, want)
	}
	if !s.Knows("witch_sight") || s.Knows("eldritch_mind") || s.Knows("not_an_invocation") {
		t.Error("Knows must be true for a known invocation only")
	}
	if a, ok := s.Arcanum(6); !ok || a != "circle_of_death" {
		t.Errorf("Arcanum(6) = %v, %v", a, ok)
	}
	if a, ok := s.Arcanum(9); !ok || a != "power_word_kill" {
		t.Errorf("Arcanum(9) = %v, %v", a, ok)
	}
	for _, l := range []int{5, 7, 10, -1} {
		if _, ok := s.Arcanum(l); ok {
			t.Errorf("Arcanum(%d) must be empty", l)
		}
	}
	if !s.ArcanumCast(6) || s.ArcanumCast(9) || s.ArcanumCast(5) || s.ArcanumCast(10) {
		t.Error("ArcanumCast must be true for the cast spell of level 6 only")
	}
	if !s.MagicalCunningUsed() || !s.ContactPatronUsed() || !s.GiftOfTheDepthsCast() || !s.GiftOfTheProtectorsSpent() || !s.HurlThroughHellSpent() || s.DarkOnesOwnLuckSpent() != 2 {
		t.Error("a getter lost a field")
	}
	if d, ok := s.FiendishResilience(); !ok || d != content.DamageFire {
		t.Errorf("FiendishResilience = %v, %v", d, ok)
	}
	if _, ok := (character.WarlockState{}).FiendishResilience(); ok {
		t.Error("the zero state has no Fiendish Resilience")
	}
	back := warState(p, s.Parts())
	if back != s || !back.Equal(s) {
		t.Errorf("Parts did not round-trip: %+v", back)
	}
	c := warChar(t, 17, 16, "fiend_patron", s)
	if warStateOf(t, c) != s {
		t.Error("the state did not round-trip through FeatureStates")
	}
	if s.Equal(character.WarlockState{}) || s.Equal(character.BardicInspirationPool{}) {
		t.Error("Equal must compare the type and every field")
	}
}

// Requirement: doc/specs/class-warlock.md FR-158, doc/rules/architecture.md section 3 (constructors validate) - a state outside its ranges is refused with ErrInvalidInput: the zero profile, an unknown invocation, an arcanum spell beyond the level's count or with an invalid id, a cast of an unchosen spell, uses spent below 0 or above 20, and a Fiendish Resilience of Force or an invalid type.
// Path: per case (invalid input, boundary).
func TestRestoreWarlockStateRefusals(t *testing.T) {
	t.Parallel()
	p := warFiend(11)
	cases := []struct {
		name  string
		p     character.WarlockProfile
		parts character.WarlockStateParts
	}{
		{"zero profile", character.WarlockProfile{}, character.WarlockStateParts{}},
		{"unknown invocation", p, character.WarlockStateParts{Invocations: []content.FeatureOptionID{"fireball"}}},
		{"arcanum beyond the count", p, character.WarlockStateParts{Arcanum: [4]content.SpellID{"", "finger_of_death", "", ""}}},
		{"invalid arcanum id", p, character.WarlockStateParts{Arcanum: [4]content.SpellID{"Not Valid", "", "", ""}}},
		{"cast of an unchosen arcanum", p, character.WarlockStateParts{ArcanumCast: [4]bool{true, false, false, false}}},
		{"negative uses spent", p, character.WarlockStateParts{DarkOnesOwnLuckSpent: -1}},
		{"too many uses spent", p, character.WarlockStateParts{DarkOnesOwnLuckSpent: 21}},
		{"Force resilience", p, character.WarlockStateParts{FiendishResilience: content.DamageForce}},
		{"invalid resilience", p, character.WarlockStateParts{FiendishResilience: content.DamageType(99)}},
	}
	for _, c := range cases {
		if _, err := character.RestoreWarlockState(c.p, c.parts); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", c.name, err)
		}
	}
	for _, ok := range []character.WarlockStateParts{{DarkOnesOwnLuckSpent: 20}, {DarkOnesOwnLuckSpent: 0}, {Arcanum: [4]content.SpellID{"circle_of_death", "", "", ""}, ArcanumCast: [4]bool{true}}} {
		if _, err := character.RestoreWarlockState(p, ok); err != nil {
			t.Errorf("%+v: %v", ok, err)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-158, FR-159 - WarlockCreationChoices seeds an empty state under the warlock key, leaves every other entry and the input untouched, names no content, and returns states that already hold the key unchanged.
// Path: per case (happy, state already held).
func TestWarlockCreationChoicesSeedAnEmptyState(t *testing.T) {
	t.Parallel()
	var choices character.ClassCreationChoices = character.WarlockCreationChoices{}
	empty := character.FeatureStates{}
	seeded := choices.SeedFeatureStates(empty)
	got, held, err := character.ReadFeatureState(seeded, character.WarlockStateKey())
	if err != nil || !held || got != (character.WarlockState{}) {
		t.Errorf("seeded = %v, %v, %v", got, held, err)
	}
	if _, held, _ := character.ReadFeatureState(empty, character.WarlockStateKey()); held {
		t.Error("the input states must not change")
	}
	again := choices.SeedFeatureStates(seeded)
	if again.Len() != seeded.Len() {
		t.Errorf("a second seeding changed the states: %d versus %d", again.Len(), seeded.Len())
	}
	if len(choices.ContentKeys()) != 0 {
		t.Errorf("ContentKeys = %v, want none", choices.ContentKeys())
	}
}
