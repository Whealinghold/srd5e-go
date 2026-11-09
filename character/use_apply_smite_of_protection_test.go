package character_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func applySmite(t *testing.T, opts ...character.Option) character.ApplySmiteOfProtection {
	t.Helper()
	return must(character.NewApplySmiteOfProtection(pldProgression(t), opts...))
}

// Requirement: doc/specs/class-paladin.md FR-084, FR-112, Table SP (SRD 5.2.1, "Level 15: Smite of Protection", p. 57) - ApplySmiteOfProtection gives Half Cover until the start of the Paladin's next turn with one
// SmiteOfProtectionTriggered event at level 15 with the Oath, a Divine Smite cast and an active aura, and a no-effect success with no event for each of the other four rows of Table SP.
// Path: per case (Table SP): unhappy (boundary, state error, rule interaction).
func TestApplySmiteOfProtectionTableSP(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label       string
		level       int
		subclass    content.SubclassID
		cast, incap bool
		effect      bool
	}{
		{"before the feature", 14, oath, true, false, false},
		{"first level", 15, oath, true, false, true},
		{"no cast", 15, oath, false, false, false},
		{"no subclass", 15, "", true, false, false},
		{"Incapacitated", 15, oath, true, true, false},
	} {
		ch := pldPaladin(t, c.level)
		before := ch
		res, err := applySmite(t).Run(context.Background(), character.ApplySmiteOfProtectionCommand{Character: ch, Subclass: c.subclass, DivineSmiteCast: c.cast, Charisma: modifierOf(t, 3), Incapacitated: c.incap})
		if err != nil {
			t.Fatalf("%s: %v", c.label, err)
		}
		if !ch.Equal(before) {
			t.Errorf("%s: the input changed", c.label)
		}
		if !c.effect {
			if _, ok := res.(character.ApplySmiteOfProtectionNoEffect); !ok {
				t.Errorf("%s: result = %#v, want no effect", c.label, res)
			}
			continue
		}
		a, ok := res.(character.ApplySmiteOfProtectionApplied)
		if !ok || a.Effect().Cover() != character.CoverHalf || !a.Effect().UntilNextTurn() {
			t.Fatalf("%s: result = %#v", c.label, res)
		}
		ev, ok := a.Events()[0].(character.SmiteOfProtectionTriggered)
		if len(a.Events()) != 1 || !ok || ev.CharacterID() != ch.ID() {
			t.Errorf("%s: events = %#v", c.label, a.Events())
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-105, FR-109, FR-115, FR-117, FR-118 - a no-effect outcome is recorded as success with no rejection; NotAPaladin is a failure not_a_paladin; the zero use case, a zero Character, a foreign
// state and a cancelled context are errors.
// Path: per case (unhappy: telemetry on rejection, invalid input).
func TestApplySmiteOfProtectionSignalsAndEdges(t *testing.T) {
	t.Parallel()
	rec, opts := fgtTelemetry()
	if _, err := applySmite(t, opts...).Run(context.Background(), character.ApplySmiteOfProtectionCommand{Character: pldPaladin(t, 15), Subclass: oath}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanApplySmiteOfProtection, "smite_of_protection", ""))
	rec, opts = fgtTelemetry()
	if res, _ := applySmite(t, opts...).Run(context.Background(), character.ApplySmiteOfProtectionCommand{Character: fgtFighter(t, 15), DivineSmiteCast: true}); res == nil {
		t.Fatal("no result")
	} else if _, ok := res.(character.ApplySmiteOfProtectionNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanApplySmiteOfProtection, "smite_of_protection", "not_a_paladin"))
	for name, cmd := range map[string]character.ApplySmiteOfProtectionCommand{"zero character": {}, "foreign state": {Character: pldForeign(t, pldPaladin(t, 15))}} {
		if res, err := applySmite(t).Run(context.Background(), cmd); res != nil || err == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		} else {
			wantInvalid(t, err, name)
		}
	}
	if _, err := (character.ApplySmiteOfProtection{}).Run(context.Background(), character.ApplySmiteOfProtectionCommand{Character: pldPaladin(t, 15)}); err == nil {
		t.Error("the zero use case ran")
	}
	if _, err := character.NewApplySmiteOfProtection(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := applySmite(t).Run(ctx, character.ApplySmiteOfProtectionCommand{Character: pldPaladin(t, 15)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
