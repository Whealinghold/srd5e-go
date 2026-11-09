package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func calcFeatures(t *testing.T, opts ...character.Option) character.CalculatePaladinFeatures {
	t.Helper()
	return must(character.NewCalculatePaladinFeatures(pldProgression(t), opts...))
}

// Requirement: doc/specs/class-paladin.md FR-042, FR-058, FR-077 to FR-088, FR-125 - CalculatePaladinFeatures summarises the features at the character's level from the definition: the aura, the attacks per Attack action, Radiant Strikes, the
// always-prepared spells (Oath spells unfiltered), the feat grants up to the level (the Fighting Style excluded), the pending Fighting Style and the Blessed Warrior cantrips; it changes nothing and has no rejection.
// Path: per case (happy / unhappy: boundary levels, empty case, rule interaction).
func TestCalculatePaladinFeatures(t *testing.T) {
	t.Parallel()
	bw := pldState(t, 11, character.PaladinStateParts{LayOnHandsRemaining: 55, ChannelDivinityRemaining: 3, FightingStyle: character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}})
	for _, c := range []struct {
		name       string
		char       character.Character
		subclass   content.SubclassID
		incap      bool
		aura       bool
		radius     int
		immunities int
		attacks    int
		radiant    bool
		spells     int
		feats      []content.FeatCategory
		pending    int
		cantrips   int
	}{
		{"level 1", pldPaladin(t, 1), "", false, false, 0, 0, 1, false, 0, nil, 0, 0},
		{"level 2, no style", pldPaladin(t, 2), "", false, false, 0, 0, 1, false, 1, nil, 1, 0},
		{"level 5", pldPaladin(t, 5), oath, false, false, 0, 0, 2, false, 6, []content.FeatCategory{content.FeatGeneral}, 1, 0},
		{"level 11 with Blessed Warrior", pldWith(t, pldPaladin(t, 11), bw), oath, false, true, 10, 2, 2, true, 8, []content.FeatCategory{content.FeatGeneral, content.FeatGeneral}, 0, 2},
		{"level 11 Incapacitated", pldWith(t, pldPaladin(t, 11), bw), oath, true, false, 0, 0, 2, true, 8, []content.FeatCategory{content.FeatGeneral, content.FeatGeneral}, 0, 2},
		{"level 20", pldPaladin(t, 20), oath, false, true, 30, 2, 2, true, 12, []content.FeatCategory{content.FeatGeneral, content.FeatGeneral, content.FeatGeneral, content.FeatGeneral, content.FeatEpicBoon}, 1, 0},
	} {
		before := c.char
		res, err := calcFeatures(t).Run(context.Background(), character.CalculatePaladinFeaturesCommand{Character: c.char, Subclass: c.subclass, Charisma: modifierOf(t, 3), Incapacitated: c.incap})
		s, ok := res.(character.CalculatePaladinFeaturesSummary)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.name, res, err)
		}
		a := s.Auras()
		if a.Active() != c.aura || a.RadiusFeet() != c.radius || len(a.Immunities()) != c.immunities && !c.incap || s.AttacksPerAttackAction() != c.attacks || s.RadiantStrikes() != c.radiant ||
			len(s.AlwaysPreparedSpells()) != c.spells || (!slices.Equal(s.FeatGrants(), c.feats) && (len(c.feats) != 0 || len(s.FeatGrants()) != 0)) || len(s.PendingChoices()) != c.pending || len(s.BlessedWarriorCantrips()) != c.cantrips {
			t.Errorf("%s: summary = aura %+v, attacks %d, radiant %v, %d spells, feats %v, pending %v, %d cantrips", c.name, a, s.AttacksPerAttackAction(), s.RadiantStrikes(), len(s.AlwaysPreparedSpells()), s.FeatGrants(), s.PendingChoices(), len(s.BlessedWarriorCantrips()))
		}
		if !c.char.Equal(before) {
			t.Errorf("%s: the input changed", c.name)
		}
		if got := s.AlwaysPreparedSpells(); len(got) > 0 {
			got[0] = "x"
			if s.AlwaysPreparedSpells()[0] == "x" {
				t.Errorf("%s: AlwaysPreparedSpells returned shared storage", c.name)
			}
		}
		if got := s.PendingChoices(); len(got) > 0 {
			got[0] = 0
			if s.PendingChoices()[0] == 0 {
				t.Errorf("%s: PendingChoices returned shared storage", c.name)
			}
		}
	}
	if s := must(calcFeatures(t).Run(context.Background(), character.CalculatePaladinFeaturesCommand{Character: pldPaladin(t, 20), Subclass: oath})).(character.CalculatePaladinFeaturesSummary); s.FeatGrants() == nil || s.PendingChoices() == nil || s.BlessedWarriorCantrips() == nil {
		t.Error("an accessor returned nil")
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-109, FR-115 - NotAPaladin for a Fighter; one calculate_features span with success or not_a_paladin; the zero use case, a zero Character, a foreign state and a cancelled context
// are errors.
// Path: per case (unhappy: invalid input, telemetry on rejection).
func TestCalculatePaladinFeaturesSignalsAndEdges(t *testing.T) {
	t.Parallel()
	rec, opts := fgtTelemetry()
	if _, err := calcFeatures(t, opts...).Run(context.Background(), character.CalculatePaladinFeaturesCommand{Character: pldPaladin(t, 6)}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanCalculatePaladinFeatures, "calculate_features", ""))
	rec, opts = fgtTelemetry()
	if res, _ := calcFeatures(t, opts...).Run(context.Background(), character.CalculatePaladinFeaturesCommand{Character: fgtFighter(t, 6)}); res == nil {
		t.Fatal("no result")
	} else if _, ok := res.(character.CalculatePaladinFeaturesNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanCalculatePaladinFeatures, "calculate_features", "not_a_paladin"))
	for name, cmd := range map[string]character.CalculatePaladinFeaturesCommand{"zero character": {}, "foreign state": {Character: pldForeign(t, pldPaladin(t, 6))}} {
		if res, err := calcFeatures(t).Run(context.Background(), cmd); res != nil || err == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		} else {
			wantInvalid(t, err, name)
		}
	}
	if _, err := (character.CalculatePaladinFeatures{}).Run(context.Background(), character.CalculatePaladinFeaturesCommand{Character: pldPaladin(t, 6)}); err == nil {
		t.Error("the zero use case ran")
	}
	if _, err := character.NewCalculatePaladinFeatures(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := calcFeatures(t).Run(ctx, character.CalculatePaladinFeaturesCommand{Character: pldPaladin(t, 6)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
