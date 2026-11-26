package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-001 - a blank ActiveEffectID is an error matching ErrInvalidInput, any other value is kept unchanged (no trimming, no case folding), MustActiveEffectID panics with that error, and the zero id is invalid.
// Path: unhappy (invalid input).
func TestActiveEffectID(t *testing.T) {
	t.Parallel()
	for _, blank := range []string{"", " ", "\t\n"} {
		if _, err := combat.NewActiveEffectID(blank); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewActiveEffectID(%q) err = %v, want ErrInvalidInput", blank, err)
		}
	}
	id, err := combat.NewActiveEffectID(" Bless ")
	if err != nil || id != " Bless " || id.IsZero() {
		t.Errorf("NewActiveEffectID(' Bless ') = %q, %v, want the value unchanged", id, err)
	}
	if !combat.ActiveEffectID("").IsZero() {
		t.Error("the zero ActiveEffectID is not zero")
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustActiveEffectID(blank) panic = %v, want an ErrInvalidInput error", r)
		}
	}()
	combat.MustActiveEffectID(" ")
}

// Requirement: doc/specs/active-effects.md FR-004 - an EffectPlaceID is opaque and blank is refused.
// Path: unhappy (invalid input).
func TestEffectPlaceIDAndPlaceSubject(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewEffectPlaceID(" "); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank place id err = %v", err)
	}
	if id, err := combat.NewEffectPlaceID("sphere"); err != nil || id != "sphere" {
		t.Errorf("NewEffectPlaceID = %q, %v", id, err)
	}
	if _, err := combat.NewPlaceEffectSubject(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank place subject err = %v", err)
	}
	if s, err := combat.NewPlaceEffectSubject("sphere"); err != nil || s.Place() != "sphere" {
		t.Errorf("NewPlaceEffectSubject = %v, %v", s, err)
	}
}

// Requirement: doc/specs/active-effects.md FR-003 (SRD 5.2.1, "Magical Effect", p. 185; "Using a Higher-Level Spell Slot", p. 104) - a spell origin takes a cast level of 0 to 9 and a valid spell; another origin needs a key.
// Path: unhappy (boundary).
func TestEffectOrigins(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		spell content.SpellID
		level int
		ok    bool
	}{
		{"level 0", "bless", 0, true}, {"level 9", "bless", 9, true},
		{"level -1", "bless", -1, false}, {"level 10", "bless", 10, false}, {"invalid spell", "", 1, false},
	}
	for _, c := range cases {
		o, err := combat.NewSpellEffectOrigin(c.spell, c.level)
		if (err == nil) != c.ok || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
		if c.ok && (o.SpellID() != c.spell || o.CastAtLevel() != c.level) {
			t.Errorf("%s: origin = %v/%d", c.name, o.SpellID(), o.CastAtLevel())
		}
	}
	if _, err := combat.NewOtherEffectOrigin("  "); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank key err = %v", err)
	}
	if o, err := combat.NewOtherEffectOrigin("ring"); err != nil || o.Key() != "ring" {
		t.Errorf("NewOtherEffectOrigin = %v, %v", o, err)
	}
}

// Requirement: doc/specs/active-effects.md FR-004 - a creature subject needs a combatant id and answers Is.
// Path: unhappy (invalid input).
func TestCreatureEffectSubject(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewCreatureEffectSubject(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero creature err = %v", err)
	}
	s := aeCreature(t, "a")
	if !s.Is(combat.MustCombatantID("a")) || s.Is(combat.MustCombatantID("b")) {
		t.Error("Is does not match the creature")
	}
}

// Requirement: doc/specs/active-effects.md FR-012 - a Concentration binding needs a caster and a subject; the zero binding is none; CasterIs and Subject read it.
// Path: unhappy (invalid input).
func TestConcentrationBinding(t *testing.T) {
	t.Parallel()
	subject := character.NewConcentrationSubjectSpell("hold_person")
	if _, err := combat.NewConcentrationBinding("", subject); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero caster err = %v", err)
	}
	if _, err := combat.NewConcentrationBinding("a", nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil subject err = %v", err)
	}
	var none combat.ConcentrationBinding
	if !none.IsZero() || none.Subject() != nil {
		t.Error("the zero binding is not none")
	}
	b := aeBinding(t, "a", "hold_person")
	if b.IsZero() || !b.CasterIs("a") || b.CasterIs("b") || b.Subject() != subject {
		t.Errorf("binding = %#v", b)
	}
}
