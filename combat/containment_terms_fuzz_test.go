package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzNewContainmentTerms feeds the terms constructors host numbers and text. It asserts properties: no
// constructor panics, every error is a *ContainmentError matching ErrInvalidInput with InvalidTerms, and
// terms that were built are valid, equal to a second build of the same input and make a record and a set.
func FuzzNewContainmentTerms(f *testing.F) {
	f.Add(2, 3, 4, 12, 5, 30, 14, 10, 15, 0, "Bite", "Maw", "3d6", true)
	f.Add(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "", "", "", false)
	f.Add(1, 6, 1, 1, 1, 1, 1, 1, 1, 1, " Bite", " ", "1d4", true)
	f.Add(math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, 7, "x", "y", "99999999999d6", false)
	f.Fuzz(func(t *testing.T, mode, size, slots, dc, relocate, threshold, conDC, landing, corpse, cond int, locked, part, notation string, half bool) {
		typed := func(err error) {
			t.Helper()
			var ce *combat.ContainmentError
			if err != nil && (!errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ce) || ce.Problem != combat.ContainmentProblemInvalidTerms) {
				t.Fatalf("error %v is not an InvalidTerms ContainmentError", err)
			}
		}
		var dmg []combat.DamageTerm
		if expr, err := dice.ParseExpression(notation); err == nil {
			term, err := combat.NewDamageTerm(expr, content.DamageAcid)
			typed(err)
			if err == nil {
				dmg = []combat.DamageTerm{term}
			}
		}
		capacity, err := combat.NewContainmentCapacity(slots, map[content.CreatureSize]int{content.CreatureSize(size): slots})
		typed(err)
		entry, err := combat.NewEntryBySave(ability.Dexterity, d20.DifficultyClass(dc), dmg, half, relocate, half)
		typed(err)
		reg, err := combat.NewRegurgitation(threshold, conDC, landing)
		typed(err)
		death, err := combat.NewEscapeFromCorpse(corpse)
		typed(err)
		exit, err := combat.NewExitTerms(d20.DifficultyClass(dc), d20.DifficultyClass(conDC), dmg)
		typed(err)
		hold, err := combat.NewGrappleHoldTerms(d20.DifficultyClass(dc), part)
		typed(err)
		periodic, err := combat.NewPeriodicDamage(dmg, combat.ContainmentTiming(mode), half)
		typed(err)
		in := combat.ContainmentTermsInput{Mode: combat.ContainmentMode(mode), MaxSize: content.CreatureSize(size), Capacity: capacity, Entry: entry,
			Imposed: []content.ConditionKind{content.ConditionKind(cond)}, Suffocation: combat.SuffocationRule(cond), Periodic: periodic,
			Regurgitation: reg, DeathRule: death, Exit: exit, Grapple: hold, DisgorgeAfterFirstDamage: half, LockedAction: locked,
			ActionKind: combat.ContainmentActionKind(size)}
		terms, err := combat.NewContainmentTerms(in)
		typed(err)
		if err != nil {
			return
		}
		again := combat.MustContainmentTerms(in)
		if terms != again || terms.Mode() == 0 {
			t.Fatalf("a valid input must build equal, non-zero terms: %+v", terms)
		}
		rec, err := combat.NewContainment(combat.ContainmentInput{Container: gid("a"), Occupant: gid("b"), OccupantSize: content.SizeMedium, Terms: terms,
			Phase: combat.ContainmentInside(), PartOrdinal: 1})
		if err != nil {
			t.Fatalf("a record of valid terms: %v", err)
		}
		if _, err := combat.NewContainments([]combat.Containment{rec}); err != nil && terms.Capacity().Cost(content.SizeMedium) <= terms.Capacity().Slots() {
			t.Fatalf("a set of one record that fits: %v", err)
		}
	})
}
