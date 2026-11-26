package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func meSense(s string) combat.IllusionSenseID { return meMust(combat.NewIllusionSenseID(s)) }

// Requirement: doc/specs/magical-effects.md FR-042 - IllusionTraits needs a non-empty set of senses (package design, Q-10), non-blank ids and a manifestation; otherwise an ErrInvalidInput error.
// Path: unhappy (invalid input).
func TestNewIllusionTraitsRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func() error{
		"nil senses":   func() error { return second(combat.NewIllusionTraits(nil, combat.NotInSpace{})) },
		"empty senses": func() error { return second(combat.NewIllusionTraits([]combat.IllusionSenseID{}, combat.NotInSpace{})) },
		"blank sense": func() error {
			return second(combat.NewIllusionTraits([]combat.IllusionSenseID{"visual", " "}, combat.NotInSpace{}))
		},
		"nil manifestation": func() error { return second(combat.NewIllusionTraits([]combat.IllusionSenseID{"visual"}, nil)) },
		"blank id":          func() error { return second(combat.NewIllusionSenseID("")) },
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-043 - DeceivesSight is true exactly when the visual sense is among the senses; an id such as a made-up hearing is accepted and does not make it true (SRD 5.2.1, "Illusions", p. 183; "Truesight", p. 190).
// Path: happy.
func TestIllusionDeceivesSightOnlyForTheVisualSense(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		senses []combat.IllusionSenseID
		want   bool
	}{
		{"visual", []combat.IllusionSenseID{combat.IllusionSenseVisual}, true},
		{"visual among others", []combat.IllusionSenseID{meSense("hearing"), combat.IllusionSenseVisual}, true},
		{"a made-up hearing", []combat.IllusionSenseID{meSense("hearing")}, false},
		{"a look-alike id", []combat.IllusionSenseID{meSense("Visual")}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := combat.NewIllusionTraits(c.senses, combat.NotInSpace{})
			if err != nil || tr.DeceivesSight() != c.want {
				t.Errorf("DeceivesSight = %v (%v), want %v", tr.DeceivesSight(), err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-042, FR-026 style immutability - the senses are copied in and out, a repeated sense is kept once, and first-seen order is kept.
// Path: unhappy (immutability).
func TestIllusionTraitsCopySenses(t *testing.T) {
	t.Parallel()
	in := []combat.IllusionSenseID{"visual", "hearing", "visual"}
	tr := meMust(combat.NewIllusionTraits(in, combat.NotInSpace{}))
	in[0] = "smell"
	got := tr.Senses()
	if !slices.Equal(got, []combat.IllusionSenseID{"visual", "hearing"}) {
		t.Fatalf("senses = %v", got)
	}
	got[0] = "smell"
	if tr.Senses()[0] != "visual" || !tr.DeceivesSight() {
		t.Error("changing a returned view changed the traits")
	}
}

// Requirement: doc/specs/magical-effects.md FR-044, FR-045 - an illusion that ManifestsInSpace has IllusionBodyFacts that are insubstantial with weight 0 pounds, and appears affected by the environment as if real by default, or as the effect specifies (SRD 5.2.1, "Illusions", p. 183).
// Path: happy.
func TestManifestsInSpaceBodyFactsAndAppearance(t *testing.T) {
	t.Parallel()
	def := combat.ManifestsInSpace{}
	if def.Appearance() != combat.AsIfReal || !def.BodyFacts().Insubstantial() || def.BodyFacts().WeightPounds() != 0 {
		t.Errorf("default: %v, %v, %v", def.Appearance(), def.BodyFacts().Insubstantial(), def.BodyFacts().WeightPounds())
	}
	spec := meMust(combat.NewManifestsInSpace(combat.SpecifiedByEffect))
	if spec.Appearance() != combat.SpecifiedByEffect || !spec.BodyFacts().Insubstantial() || spec == def {
		t.Errorf("specified: %v", spec.Appearance())
	}
	tr := meMust(combat.NewIllusionTraits([]combat.IllusionSenseID{"visual"}, spec))
	if tr.Manifestation() != combat.IllusionManifestation(spec) {
		t.Error("Manifestation must return what was given")
	}
	for a, want := range map[combat.EnvironmentAppearance]string{combat.AsIfReal: "as_if_real", combat.SpecifiedByEffect: "specified_by_effect", 7: "unknown"} {
		if a.String() != want {
			t.Errorf("%d.String() = %q", int(a), a.String())
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-045 - an undefined environment appearance is an ErrInvalidInput error.
// Path: unhappy (invalid input).
func TestNewManifestsInSpaceRejectsUndefinedAppearance(t *testing.T) {
	t.Parallel()
	for _, a := range []combat.EnvironmentAppearance{-1, combat.SpecifiedByEffect + 1} {
		if err := second(combat.NewManifestsInSpace(a)); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewManifestsInSpace(%d) error = %v", a, err)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-046 - IllusionManifestation is closed: NotInSpace declares no body facts and no appearance; only ManifestsInSpace does, shown by an exhaustive switch (SRD 5.2.1, "Illusions", p. 183).
// Path: unhappy (boundary).
func TestIllusionManifestationIsClosed(t *testing.T) {
	t.Parallel()
	for _, m := range []combat.IllusionManifestation{combat.NotInSpace{}, combat.ManifestsInSpace{}} {
		switch v := m.(type) {
		case combat.NotInSpace:
			// A struct with no members: it cannot carry body facts or an appearance.
		case combat.ManifestsInSpace:
			if !v.BodyFacts().Insubstantial() {
				t.Error("ManifestsInSpace must be insubstantial")
			}
		default:
			t.Errorf("unexpected variant %T", m)
		}
	}
}
