package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/shape-changing-spells.md FR-008 - NewRatedByLevel never panics for any integer, accepts exactly 1 to 20 and compares as the Challenge Rating of that level.
// Path: fuzz.
func FuzzNewRatedByLevel(f *testing.F) {
	for _, n := range []int{0, 1, 20, 21, -1, math.MaxInt, math.MinInt} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		r, err := combat.NewRatedByLevel(n)
		if (n >= 1 && n <= 20) != (err == nil) {
			t.Fatalf("NewRatedByLevel(%d) error = %v", n, err)
		}
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if r.Level() != n || content.CompareChallengeRating(r.Challenge(), content.MustChallengeRating("1")) < 0 {
			t.Fatalf("rating %+v", r)
		}
	})
}

// Requirement: doc/specs/shape-changing-spells.md FR-007 - NewCreatureForm and NewObjectForm never panic for any size number and accept a size exactly when it is valid and printed.
// Path: fuzz.
func FuzzNewCreatureForm(f *testing.F) {
	for _, n := range []int{0, 1, 3, 4, 6, 7, -1} {
		f.Add(n)
	}
	facts := luMust(combat.NewFormFacts(msMonster(f, func(m *content.MonsterFields) {
		m.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeLarge}
	})))
	f.Fuzz(func(t *testing.T, n int) {
		size := content.CreatureSize(n)
		_, err := combat.NewCreatureForm(facts, size)
		if printed := size == content.SizeMedium || size == content.SizeLarge; printed != (err == nil) {
			t.Fatalf("NewCreatureForm(%d) error = %v", n, err)
		}
		if _, err := combat.NewObjectForm(size); size.Valid() != (err == nil) {
			t.Fatalf("NewObjectForm(%d) error = %v", n, err)
		}
	})
}

// Requirement: doc/specs/shape-changing-spells.md FR-054, FR-055 - NewActiveTransformation never panics for any numbers in its enum fields; it either builds a value or returns an error matching ErrInvalidInput.
// Path: fuzz.
func FuzzNewActiveTransformation(f *testing.F) {
	f.Add(1, 0, 0, 1, 1, false)
	f.Add(3, 0, 1, 1, 1, true)
	f.Add(0, 9, 5, 0, 0, true)
	form := ssBeast(f, "1", 11)
	f.Fuzz(func(t *testing.T, spell, mode, terms, provenance, maintained int, permanent bool) {
		got, err := combat.NewActiveTransformation(combat.ActiveTransformationFields{
			Spell: combat.ShapeShiftSpell(spell), Mode: combat.TrueTransformation(mode), Terms: combat.ShapechangeTerms(terms),
			Target: gid("a"), Caster: gid("a"), Form: form, MaintainedBy: combat.MaintainedBy(maintained),
			Permanent: permanent, TemporaryHitPoints: combat.FormTemporaryHitPoints(provenance),
		})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
				t.Fatalf("error %v, value %+v", err, got)
			}
			return
		}
		if got.IsZero() || !got.Spell().Valid() {
			t.Fatalf("built the invalid value %+v", got)
		}
		if got.IsPermanent() && got.Spell() != combat.TruePolymorphSpell {
			t.Fatalf("permanent %v", got.Spell())
		}
	})
}
