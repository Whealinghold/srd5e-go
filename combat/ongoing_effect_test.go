package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/magical-effects.md FR-006 - an OngoingEffect carries an id, a category and a source, and is equal by value.
// Path: happy.
func TestOngoingEffectCarriesIDCategorySourceAndIsEqualByValue(t *testing.T) {
	t.Parallel()
	e := meEffect("e1", combat.OngoingEffectCurse, meLabeled("hag_curse"))
	if e.ID() != "e1" || e.Category() != combat.OngoingEffectCurse || e.Source() != meLabeled("hag_curse") || e.IsZero() {
		t.Errorf("accessors: %v %v %v", e.ID(), e.Category(), e.Source())
	}
	if e != meEffect("e1", combat.OngoingEffectCurse, meLabeled("hag_curse")) || e == meEffect("e2", combat.OngoingEffectCurse, meLabeled("hag_curse")) ||
		e == meEffect("e1", combat.OngoingEffectCurse, meLabeled("other")) {
		t.Error("equality must follow id and source")
	}
	for cat, want := range map[combat.OngoingEffectCategory]string{combat.OngoingEffectSpell: "spell", combat.OngoingEffectCurse: "curse",
		combat.OngoingEffectMagicalContagion: "magical_contagion", combat.OngoingEffectMagicalEffect: "magical_effect", 0: "unknown"} {
		if cat.String() != want {
			t.Errorf("%d.String() = %q, want %q", int(cat), cat.String(), want)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestNewOngoingEffectInvariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		id                      combat.OngoingEffectID
		category                combat.OngoingEffectCategory
		source                  combat.EffectSource
		ok                      bool
	}{
		{"spell with a non-spell source", meSpec + "FR-007", "unhappy (invalid input)", "x", combat.OngoingEffectSpell, meItem("ring"), false},
		{"spell with a spell source", meSpec + "FR-007", "happy", "x", combat.OngoingEffectSpell, meSpell("bless"), true},
		{"contagion with a non-magical source", meSpec + `FR-008; SRD 5.2.1, "Dead", p. 180`, "unhappy (invalid input)", "x", combat.OngoingEffectMagicalContagion, meNonMagical("mold"), false},
		{"contagion with a labeled source", meSpec + "FR-008", "happy", "x", combat.OngoingEffectMagicalContagion, meLabeled("rot"), true},
		{"magical effect with a non-magical source", meSpec + `FR-008; SRD 5.2.1, "Magical Effect", p. 185`, "unhappy (invalid input)", "x", combat.OngoingEffectMagicalEffect, meNonMagical("mold"), false},
		{"magical effect with an item source", meSpec + "FR-008", "happy", "x", combat.OngoingEffectMagicalEffect, meItem("ring"), true},
		{"curse with a non-magical source is allowed", meSpec + "FR-027: a curse reads only id, category and source", "happy", "x", combat.OngoingEffectCurse, meNonMagical("mold"), true},
		{"blank id", meSpec + "FR-006", "unhappy (invalid input)", "  ", combat.OngoingEffectCurse, meLabeled("c"), false},
		{"undefined category", meSpec + "FR-006", "unhappy (invalid input)", "x", 0, meLabeled("c"), false},
		{"category past the last", meSpec + "FR-006", "unhappy (boundary)", "x", combat.OngoingEffectMagicalEffect + 1, meLabeled("c"), false},
		{"zero source", meSpec + "FR-006", "unhappy (invalid input)", "x", combat.OngoingEffectCurse, combat.EffectSource{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, err := combat.NewOngoingEffect(c.id, c.category, c.source)
			if c.ok != (err == nil) || (!c.ok && (!errors.Is(err, srderr.ErrInvalidInput) || !e.IsZero())) {
				t.Errorf("NewOngoingEffect = %v, %v, ok want %v [%s]", e, err, c.ok, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-009 - OngoingEffects accept an empty list, keep the order given and expose a defensively copied view; the input list is copied too.
// Path: unhappy (empty and minimal).
func TestOngoingEffectsKeepOrderAndCopy(t *testing.T) {
	t.Parallel()
	for _, in := range [][]combat.OngoingEffect{nil, {}} {
		if l := meEffects(in...); l.Len() != 0 || len(l.All()) != 0 {
			t.Errorf("empty input gave %d effects", l.Len())
		}
	}
	a, b, c := meBless(), meCurse("c1"), meEffect("m", combat.OngoingEffectMagicalEffect, meItem("ring"))
	in := []combat.OngoingEffect{a, b, c}
	l := meMust(combat.NewOngoingEffects(in))
	in[0] = c // the caller changes its slice afterwards
	got := l.All()
	if !slices.Equal(got, []combat.OngoingEffect{a, b, c}) {
		t.Fatalf("order = %v", got)
	}
	got[1] = a
	if l.All()[1] != b {
		t.Error("changing the returned view changed the list")
	}
	if !l.Equal(meEffects(a, b, c)) || l.Equal(meEffects(b, a, c)) || l.Equal(meEffects(a, b)) {
		t.Error("Equal must follow content and order")
	}
	if e, ok := l.Find(meEID("c1")); !ok || e != b {
		t.Errorf("Find(c1) = %v, %v", e, ok)
	}
	for _, missing := range []string{"nope", "a", "bless2"} {
		if e, ok := l.Find(meEID(missing)); ok || !e.IsZero() {
			t.Errorf("Find(%s) = %v, %v", missing, e, ok)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-010 - two entries with one OngoingEffectID, or a zero effect, are an ErrInvalidInput error.
// Path: unhappy (invalid input).
func TestOngoingEffectsRejectDuplicateIDsAndZeroEffects(t *testing.T) {
	t.Parallel()
	dup := meEffect("bless", combat.OngoingEffectCurse, meLabeled("c"))
	for name, in := range map[string][]combat.OngoingEffect{
		"duplicate id": {meBless(), dup},
		"zero effect":  {meBless(), {}},
	} {
		l, err := combat.NewOngoingEffects(in)
		if !errors.Is(err, srderr.ErrInvalidInput) || l.Len() != 0 {
			t.Errorf("%s: got %d effects, %v, want ErrInvalidInput", name, l.Len(), err)
		}
	}
}

// Requirement: doc/specs/magical-effects.md section 5 - a blank OngoingEffectID is rejected, and the zero id reports IsZero.
// Path: unhappy (invalid input).
func TestNewOngoingEffectID(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", " ", "\t\n"} {
		if id, err := combat.NewOngoingEffectID(s); !errors.Is(err, srderr.ErrInvalidInput) || !id.IsZero() {
			t.Errorf("NewOngoingEffectID(%q) = %q, %v", s, id, err)
		}
	}
	if id, err := combat.NewOngoingEffectID(" x"); err != nil || id.String() != " x" || id.IsZero() {
		t.Errorf("a non-blank id is kept as given, got %q, %v", id, err)
	}
}
