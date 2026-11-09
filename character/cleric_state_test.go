package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-cleric.md FR-047, FR-138, FR-139 - NewClericResources accepts 0 of 0 and the full range and refuses a negative maximum, a
// remaining count below 0 or above the maximum, a nil state and a Recharging count below 1; the zero value is 0 of 0 with Divine Intervention
// available; the two keys are named cleric_choices and cleric_resources.
// Path: per case (unhappy: boundary, invalid input).
func TestNewClericResources(t *testing.T) {
	t.Parallel()
	avail, used, rech := character.DivineInterventionAvailable{}, character.DivineInterventionUnavailableUntilLongRest{}, character.DivineInterventionRecharging{LongRestsRemaining: 2}
	for _, c := range []struct {
		name     string
		max, rem int
		state    character.DivineInterventionState
		wantErr  bool
	}{
		{"0 of 0", 0, 0, avail, false}, {"full", 3, 3, used, false}, {"none left", 3, 0, rech, false}, {"above the maximum", 3, 4, avail, true},
		{"negative remaining", 3, -1, avail, true}, {"negative maximum", -1, 0, avail, true}, {"nil state", 3, 3, nil, true},
		{"recharging 0", 3, 3, character.DivineInterventionRecharging{}, true},
	} {
		r, err := character.NewClericResources(c.max, c.rem, c.state)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: error = %v, want error %v", c.name, err, c.wantErr)
		}
		if err != nil {
			wantInvalid(t, err, c.name)
		} else if r.ChannelDivinityMaximum() != c.max || r.ChannelDivinityRemaining() != c.rem || r.DivineIntervention() != c.state {
			t.Errorf("%s: resources = %+v", c.name, r)
		}
	}
	var zero character.ClericResources
	if zero.DivineIntervention() != (character.DivineInterventionAvailable{}) || !zero.Equal(character.InitialClericResources(0)) || !character.InitialClericResources(-4).Equal(zero) {
		t.Error("zero resources are not 0 of 0 with Divine Intervention available")
	}
	if got := character.InitialClericResources(3); got.ChannelDivinityRemaining() != 3 || got.ChannelDivinityMaximum() != 3 || got.Equal(zero) || zero.Equal(character.ClericChoices{}) {
		t.Errorf("InitialClericResources(3) = %+v", got)
	}
	if character.ClericChoicesKey().Name() != "cleric_choices" || character.ClericResourcesKey().Name() != "cleric_resources" {
		t.Error("key names")
	}
}

// Requirement: doc/specs/class-cleric.md FR-042, FR-043, FR-091, FR-092, FR-095, table 4.11.2 (SRD 5.2.1, "Level 2: Channel Divinity", "Level 10: Divine
// Intervention", "Level 20: Greater Divine Intervention", pp. 37-38) - a Short Rest regains one use, never past the maximum, and leaves Divine Intervention as it is; a Long
// Rest regains every use, makes a used Divine Intervention available, leaves an available one so and counts a recharging one down by one, available at 0 (GD-01 to GD-06).
// Path: per case (the rest rows): happy / unhappy (boundary).
func TestClericResourcesAfterRest(t *testing.T) {
	t.Parallel()
	used, avail := character.DivineInterventionUnavailableUntilLongRest{}, character.DivineInterventionAvailable{}
	rech := func(n int) character.DivineInterventionState {
		return character.DivineInterventionRecharging{LongRestsRemaining: n}
	}
	for _, c := range []struct {
		name  string
		from  character.ClericResources
		kind  character.RestKind
		rem   int
		state character.DivineInterventionState
	}{
		{"short rest regains one", must(character.NewClericResources(3, 1, used)), character.ShortRest, 2, used},
		{"short rest at the maximum", must(character.NewClericResources(3, 3, avail)), character.ShortRest, 3, avail},
		{"short rest never counts a recharge", must(character.NewClericResources(3, 3, rech(5))), character.ShortRest, 3, rech(5)},
		{"long rest regains all", must(character.NewClericResources(3, 0, avail)), character.LongRest, 3, avail},
		{"long rest makes a used one available", must(character.NewClericResources(2, 2, used)), character.LongRest, 2, avail},
		{"GD-01 to GD-02 one before", must(character.NewClericResources(4, 4, rech(2))), character.LongRest, 4, rech(1)},
		{"GD-03 exactly N", must(character.NewClericResources(4, 4, rech(1))), character.LongRest, 4, avail},
		{"GD-04 maximum N, one before", must(character.NewClericResources(4, 4, rech(8))), character.LongRest, 4, rech(7)},
		{"an invalid kind changes nothing", must(character.NewClericResources(3, 1, used)), character.RestKind(0), 1, used},
	} {
		got := c.from.AfterRest(c.kind)
		if got.ChannelDivinityRemaining() != c.rem || got.DivineIntervention() != c.state || got.ChannelDivinityMaximum() != c.from.ChannelDivinityMaximum() {
			t.Errorf("%s: %+v, want %d uses and %v", c.name, got, c.rem, c.state)
		}
	}
}

// Requirement: doc/specs/class-cleric.md FR-037, FR-076, FR-034 - ClericChoices holds the order, the Thaumaturge's extra cantrip and the Blessed Strikes option;
// the extra-cantrip allowance is 1 for a Thaumaturge and 0 otherwise; NewClericChoices refuses values outside the closed sets and an invalid cantrip id.
// Path: per case (unhappy: invalid input).
func TestClericChoices(t *testing.T) {
	t.Parallel()
	th := must(character.NewClericChoices(character.DivineOrderThaumaturge, "light", character.BlessedStrikesPotentSpellcasting))
	pr := must(character.NewClericChoices(character.DivineOrderProtector, "", character.BlessedStrikesDivineStrike))
	if th.DivineOrder() != character.DivineOrderThaumaturge || th.ExtraCantrip() != "light" || th.BlessedStrikes() != character.BlessedStrikesPotentSpellcasting ||
		th.ExtraCantrips() != 1 || pr.ExtraCantrips() != 0 || (character.ClericChoices{}).ExtraCantrips() != 0 {
		t.Errorf("choices = %+v / %+v", th, pr)
	}
	if !th.Equal(th) || th.Equal(pr) || th.Equal(character.ClericResources{}) {
		t.Error("Equal")
	}
	for _, c := range []struct {
		order   character.DivineOrder
		cantrip string
		o       character.BlessedStrikesOption
	}{{3, "", 0}, {-1, "", 0}, {0, "", 3}, {0, "", -1}, {character.DivineOrderThaumaturge, "Not Valid", 0}} {
		_, err := character.NewClericChoices(c.order, contentSpellID(c.cantrip), c.o)
		wantInvalid(t, err, "choices")
	}
	for o, s := range map[character.DivineOrder]string{0: "none", character.DivineOrderProtector: "protector", character.DivineOrderThaumaturge: "thaumaturge"} {
		if o.String() != s {
			t.Errorf("order %d = %q", o, o)
		}
	}
	for o, s := range map[character.BlessedStrikesOption]string{0: "none", character.BlessedStrikesDivineStrike: "divine_strike", character.BlessedStrikesPotentSpellcasting: "potent_spellcasting"} {
		if o.String() != s {
			t.Errorf("option %d = %q", o, o)
		}
	}
}
