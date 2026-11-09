package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

func useFreeCast(t *testing.T, opts ...character.Option) character.UseFreeCast {
	t.Helper()
	return must(character.NewUseFreeCast(pldProgression(t), opts...))
}

// Requirement: doc/specs/class-paladin.md FR-060 to FR-064 (SRD 5.2.1, "Level 2: Paladin’s Smite", p. 54; "Level 5: Faithful Steed", p. 55) - UseFreeCast spends the free cast from its gate level (2 and 5), returns the spell to
// cast (Divine Smite, Find Steed) with one FreeCastSpent event, changes only that free cast, and rejects a spent use (already_used_since_long_rest) and a level below the gate (feature_not_gained), both when both hold.
// Path: per case (unhappy: boundary levels 1/2 and 4/5, state error, rule interaction).
func TestUseFreeCast(t *testing.T) {
	t.Parallel()
	spentSmite := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, PaladinsSmite: character.LongRestUseSpent})
	for _, c := range []struct {
		name    string
		char    character.Character
		feature character.FreeCastFeature
		spell   string
		reasons []string
	}{
		{"Smite at level 1", pldPaladin(t, 1), character.FreeCastPaladinsSmite, "", []string{"feature_not_gained"}},
		{"Smite at level 2", pldPaladin(t, 2), character.FreeCastPaladinsSmite, "divine_smite", nil},
		{"Steed at level 4", pldPaladin(t, 4), character.FreeCastFaithfulSteed, "", []string{"feature_not_gained"}},
		{"Steed at level 5", pldPaladin(t, 5), character.FreeCastFaithfulSteed, "find_steed", nil},
		{"Smite spent", pldWith(t, pldPaladin(t, 5), spentSmite), character.FreeCastPaladinsSmite, "", []string{"already_used_since_long_rest"}},
		{"Steed with Smite spent", pldWith(t, pldPaladin(t, 5), spentSmite), character.FreeCastFaithfulSteed, "find_steed", nil},
		{"both", pldWith(t, pldPaladin(t, 1), pldState(t, 1, character.PaladinStateParts{LayOnHandsRemaining: 5, PaladinsSmite: character.LongRestUseSpent})), character.FreeCastPaladinsSmite, "", []string{"feature_not_gained", "already_used_since_long_rest"}},
	} {
		before := c.char
		res, err := useFreeCast(t).Run(context.Background(), character.UseFreeCastCommand{Character: c.char, Feature: c.feature})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !c.char.Equal(before) {
			t.Errorf("%s: the input changed", c.name)
		}
		if len(c.reasons) > 0 {
			rj, ok := res.(character.UseFreeCastRejected)
			if !ok || !slices.Equal(pldReasons(rj.Reasons()), c.reasons) {
				t.Errorf("%s: result = %#v, want reasons %v", c.name, res, c.reasons)
			}
			continue
		}
		a, ok := res.(character.UseFreeCastApplied)
		if !ok || string(a.Spell()) != c.spell {
			t.Fatalf("%s: result = %#v", c.name, res)
		}
		s := pldStateOf(t, a.Character())
		other := character.FreeCastPaladinsSmite
		if c.feature == character.FreeCastPaladinsSmite {
			other = character.FreeCastFaithfulSteed
		}
		wantOther := character.LongRestUseAvailable
		if c.name == "Steed with Smite spent" {
			wantOther = character.LongRestUseSpent
		}
		if s.FreeCast(c.feature) != character.LongRestUseSpent || s.FreeCast(other) != wantOther || !s.Equal(a.State()) {
			t.Errorf("%s: state = %+v", c.name, s)
		}
		ev, ok := a.Events()[0].(character.FreeCastSpent)
		if len(a.Events()) != 1 || !ok || ev.Feature() != c.feature || string(ev.Spell()) != c.spell || ev.CharacterID() != c.char.ID() {
			t.Errorf("%s: events = %#v", c.name, a.Events())
		}
		if lvl := c.char.ClassLevels().LevelIn(character.PaladinClassID); s.LayOnHandsRemaining() != 5*lvl {
			t.Errorf("%s: the Lay On Hands pool changed to %d", c.name, s.LayOnHandsRemaining())
		}
		again, err := useFreeCast(t).Run(context.Background(), character.UseFreeCastCommand{Character: a.Character(), Feature: c.feature})
		if rj, ok := again.(character.UseFreeCastRejected); err != nil || !ok || !slices.Equal(pldReasons(rj.Reasons()), []string{"already_used_since_long_rest"}) {
			t.Errorf("%s: a second spend = %#v, %v", c.name, again, err)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-105, FR-109 - NotAPaladin for a character with no Paladin level, an undefined feature, a zero Character or use case and a cancelled context are errors before any signal, and
// the result is the same with a recording telemetry, which records one span for the free cast.
// Path: per case (unhappy: invalid input, state error).
func TestUseFreeCastEdges(t *testing.T) {
	t.Parallel()
	rec, opts := fgtTelemetry()
	res, err := useFreeCast(t, opts...).Run(context.Background(), character.UseFreeCastCommand{Character: pldPaladin(t, 2), Feature: character.FreeCastPaladinsSmite})
	if _, ok := res.(character.UseFreeCastApplied); !ok || err != nil {
		t.Fatalf("result = %#v, %v", res, err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUseFreeCast, "paladins_smite", ""))
	rec, opts = fgtTelemetry()
	if res, err := useFreeCast(t, opts...).Run(context.Background(), character.UseFreeCastCommand{Character: fgtFighter(t, 5), Feature: character.FreeCastFaithfulSteed}); err != nil {
		t.Fatal(err)
	} else if _, ok := res.(character.UseFreeCastNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUseFreeCast, "faithful_steed", "not_a_paladin"))
	for name, cmd := range map[string]character.UseFreeCastCommand{
		"undefined feature": {Character: pldPaladin(t, 5)}, "zero character": {Feature: character.FreeCastPaladinsSmite}, "foreign state": {Character: pldForeign(t, pldPaladin(t, 5)), Feature: character.FreeCastPaladinsSmite},
	} {
		if res, err := useFreeCast(t).Run(context.Background(), cmd); res != nil || err == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		} else {
			wantInvalid(t, err, name)
		}
	}
	if _, err := (character.UseFreeCast{}).Run(context.Background(), character.UseFreeCastCommand{Character: pldPaladin(t, 5), Feature: character.FreeCastPaladinsSmite}); err == nil {
		t.Error("the zero use case ran")
	}
	if _, err := character.NewUseFreeCast(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := useFreeCast(t).Run(ctx, character.UseFreeCastCommand{Character: pldPaladin(t, 5), Feature: character.FreeCastPaladinsSmite}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
