package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-traits.md FR-002, FR-003 - RecogniseTraits never panics for any text and name, gives equal results for equal input, and every part appears at most once.
// Path: unhappy (invalid input).
func FuzzRecogniseTraits(f *testing.F) {
	for _, s := range []string{mtRegenTwo, mtParry, mtPack, mtBurst, "", " ", "The ", "The  explodes when it dies.", "If the a fails", "The a regains 99999999999 Hit Points", "’’’", "The a has Advantage"} {
		f.Add("Regeneration", s)
		f.Add("Parry", s)
		f.Add("Death Burst", s)
		f.Add("Pack Tactics", s)
	}
	f.Fuzz(func(t *testing.T, name, text string) {
		part, err := content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: text})
		if err != nil {
			return
		}
		m := mtMonster(t, []content.StatBlockPart{part}, []content.StatBlockPart{part})
		a, b := combat.RecogniseTraits(m), combat.RecogniseTraits(m)
		if !a.Equal(b) || len(a.Traits())+len(a.Unrecognised()) > 2 {
			t.Fatalf("name %q text %q: %v %v", name, text, a, b)
		}
	})
}

// Requirement: doc/specs/monster-traits.md FR-007, FR-030 - the trait and fact constructors never panic and an accepted value is in range.
// Path: unhappy (invalid input).
func FuzzTraitConstructors(f *testing.F) {
	for _, n := range []int{0, 1, -1, 2, 1 << 20, content.MonsterMaxNumber, mtOverMax()} {
		f.Add(n, n%14, n%3)
	}
	f.Fuzz(func(t *testing.T, n, ty, hit int) {
		ref := mtRef(t, 0)
		if r, err := combat.NewRegenerationTrait(ref, n, []content.DamageType{content.DamageType(ty)}); err == nil && (r.Amount() < 1 || len(r.SuppressingTypes()) != 1) {
			t.Fatalf("regeneration %d %d: %+v", n, ty, r)
		}
		if p, err := combat.NewParryTrait(ref, n); err == nil && p.Bonus() != n {
			t.Fatalf("parry %d", n)
		}
		if u, err := combat.NewUndeadFortitudeFacts(n, []content.DamageType{content.DamageType(ty)}, combat.HitKind(hit)); err == nil && u.DamageTaken() != n {
			t.Fatalf("facts %d", n)
		}
	})
}
