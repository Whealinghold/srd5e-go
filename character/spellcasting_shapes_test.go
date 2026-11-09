package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spellcasting.md FR-118 - Concentration holds a closed subject (a spell or a
// feature) and a kind; the kind casting applies to a spell subject only.
// Path: unhappy (invalid input).
func TestConcentration(t *testing.T) {
	t.Parallel()
	spell := character.NewConcentrationSubjectSpell("bless")
	feature := character.NewConcentrationSubjectFeature("rage")
	cases := []struct {
		name    string
		subject character.ConcentrationSubject
		kind    character.ConcentrationKind
		ok      bool
	}{
		{"spell effect", spell, character.ConcentrationEffect, true}, {"spell casting", spell, character.ConcentrationCasting, true},
		{"feature effect", feature, character.ConcentrationEffect, true}, {"feature casting", feature, character.ConcentrationCasting, false},
		{"nil subject", nil, character.ConcentrationEffect, false}, {"zero kind", spell, 0, false}, {"kind 3", spell, 3, false},
	}
	for _, c := range cases {
		got, err := character.NewConcentration(c.subject, c.kind)
		if (err == nil) != c.ok || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) || got.IsZero() == c.ok {
			t.Errorf("%s: %v, %v", c.name, got, err)
		}
		if c.ok && (got.Subject() != c.subject || got.Kind() != c.kind) {
			t.Errorf("%s: accessors = %v %v", c.name, got.Subject(), got.Kind())
		}
	}
	if spell.Spell() != "bless" || feature.Feature() != "rage" || !(character.Concentration{}).IsZero() {
		t.Error("subject accessors")
	}
	if character.ConcentrationEffect.String() != "effect" || character.ConcentrationCasting.String() != "casting" || character.ConcentrationKind(0).String() != "unknown" {
		t.Error("ConcentrationKind.String")
	}
}

// Requirement: doc/specs/spellcasting.md FR-148, FR-149 - CastSource has six values, scroll and duplicated
// included.
// Path: happy.
func TestCastSourceNames(t *testing.T) {
	t.Parallel()
	for i, n := range []string{"slot", "cantrip", "ritual", "feature_granted", "scroll", "duplicated"} {
		if got := character.CastSource(i + 1).String(); got != n {
			t.Errorf("CastSource(%d) = %q, want %q", i+1, got, n)
		}
	}
	if character.CastSource(0).String() != "unknown" || character.CastSource(7).String() != "unknown" || character.CastSourceScroll != 5 || character.CastSourceDuplicated != 6 {
		t.Error("CastSource bounds")
	}
}

// Requirement: doc/specs/spellcasting.md section 5 (CastingInProgress) - a casting in progress needs a valid
// spell, class and source other than duplicated, and at least one minute; the slot choice and mode
// are optional.
// Path: unhappy (invalid input).
func TestCastingInProgress(t *testing.T) {
	t.Parallel()
	slot := character.NewSpellSlotChoiceSpellcasting(slotLevel(3))
	ok, err := character.NewCastingInProgress("bless", character.CastSourceSlot, character.CasterCleric, content.CastingTimeMode{}, slot, 10)
	if err != nil || ok.IsZero() || ok.Spell() != "bless" || ok.Source() != character.CastSourceSlot || ok.Class() != character.CasterCleric || ok.TotalMinutes() != 10 || ok.Slot() != slot || ok.Mode() != (content.CastingTimeMode{}) {
		t.Fatalf("casting = %+v, %v", ok, err)
	}
	if got := slot.Level(); got != slotLevel(3) {
		t.Errorf("slot level = %v", got)
	}
	if _, err := character.NewCastingInProgress("bless", character.CastSourceRitual, character.CasterCleric, content.CastingTimeMode{}, character.SpellSlotChoicePactMagic{}, 1); err != nil {
		t.Errorf("one minute with a Pact Magic choice: %v", err)
	}
	for name, f := range map[string]func() error{
		"spell": func() error {
			_, e := character.NewCastingInProgress("", character.CastSourceSlot, character.CasterCleric, content.CastingTimeMode{}, nil, 1)
			return e
		},
		"class": func() error {
			_, e := character.NewCastingInProgress("a", character.CastSourceSlot, 0, content.CastingTimeMode{}, nil, 1)
			return e
		},
		"source 0": func() error {
			_, e := character.NewCastingInProgress("a", 0, character.CasterCleric, content.CastingTimeMode{}, nil, 1)
			return e
		},
		"duplicated": func() error {
			_, e := character.NewCastingInProgress("a", character.CastSourceDuplicated, character.CasterCleric, content.CastingTimeMode{}, nil, 1)
			return e
		},
		"minutes": func() error {
			_, e := character.NewCastingInProgress("a", character.CastSourceScroll, character.CasterCleric, content.CastingTimeMode{}, nil, 0)
			return e
		},
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if !(character.CastingInProgress{}).IsZero() {
		t.Error("zero CastingInProgress is not zero")
	}
}
