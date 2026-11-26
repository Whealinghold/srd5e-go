package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-001 to FR-010 - building an ongoing spell and a registry from untrusted ids, levels and times never panics; a refused value is the zero value with an error matching ErrInvalidInput; an accepted registry reads back the spells it holds and hands out copies.
// Path: unhappy (invalid input).
func FuzzNewOngoingSpellAndRegistry(f *testing.F) {
	f.Add("bless", "e1", "wizard", 1, 60, 2)
	f.Add("", "", "", -1, -1, 0)
	f.Add("haste", "e1", "", 9, 0, 300)
	f.Add("x", "e1", "w", 99, 1<<30, 1)
	f.Fuzz(func(t *testing.T, spell, effect, caster string, level, seconds, copies int) {
		id, err := combat.NewOngoingEffectID(effect)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("effect id error %v does not match ErrInvalidInput", err)
			}
			return
		}
		src, err := combat.SpellSource(content.SpellID(spell))
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("spell source error %v does not match ErrInvalidInput", err)
			}
			return
		}
		eff, err := combat.NewOngoingEffect(id, combat.OngoingEffectSpell, src)
		if err != nil {
			return
		}
		lvl, lerr := content.NewSpellLevel(level)
		rem, rerr := combat.NewSpellTimeRemaining(seconds)
		for _, e := range []error{lerr, rerr} { // the zero SpellLevel is a valid cantrip, so only the constructors are checked
			if e != nil && !errors.Is(e, srderr.ErrInvalidInput) {
				t.Fatalf("level or time error %v does not match ErrInvalidInput", e)
			}
		}
		if lerr != nil || rerr != nil {
			return
		}
		sp, err := combat.NewOngoingSpell(combat.OngoingSpellParams{Effect: eff, CastAtLevel: lvl, Caster: combat.CombatantID(caster), Origin: combat.SpellOriginOrdinary,
			Remaining: rem, Restriction: combat.DispelUnrestricted{}, RequiresConcentration: caster != ""})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewOngoingSpell error %v does not match ErrInvalidInput", err)
			}
			return
		}
		target, err := combat.NewDispelCreature(combat.MustCombatantID("target"))
		if err != nil {
			t.Fatal(err)
		}
		held := make([]combat.HeldSpell, 0, 4)
		for i := 0; i < copies%5 && i >= 0; i++ { // the same effect id twice must be refused
			held = append(held, combat.HeldSpell{Target: target, Spell: sp})
		}
		reg, err := combat.NewOngoingSpellRegistry(held)
		switch {
		case len(held) > 1:
			if err == nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("a registry with %d spells sharing one effect id = %v, want ErrInvalidInput", len(held), err)
			}
		case err != nil:
			t.Fatalf("a registry of %d valid spells refused: %v", len(held), err)
		default:
			on := reg.SpellsOn(target)
			if len(on) != len(held) {
				t.Fatalf("SpellsOn = %d spells, want %d", len(on), len(held))
			}
			if len(on) > 0 {
				on[0] = combat.OngoingSpell{} // a copy: the registry must not change
				if len(reg.SpellsOn(target)) != len(held) || reg.SpellsOn(target)[0].IsZero() {
					t.Fatal("SpellsOn returned the registry's own storage")
				}
			}
		}
	})
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031, FR-032 - building a cast observation from untrusted ids and flags never panics; a zero caster, an invalid spell id or an undefined source is refused with ErrInvalidInput; an accepted observation reads back what was given.
// Path: unhappy (invalid input).
func FuzzNewCastObservation(f *testing.F) {
	f.Add("haste", "wizard", 1, true, true, true, true)
	f.Add("", "", 0, false, false, false, false)
	f.Add("x", "w", 99, true, false, true, false)
	f.Fuzz(func(t *testing.T, spell, caster string, source int, v, s, m, conc bool) {
		obs, err := combat.NewCastObservation(content.SpellID(spell), combat.CombatantID(caster), character.CastSource(source), combat.NewUsedComponents(v, s, m), conc, content.NewCastingTimeMagicAction())
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if string(obs.Spell()) != spell || int(obs.Source()) != source || caster == "" {
			t.Fatalf("observation reads back %q source %d (caster %q), want %q source %d and a named caster", obs.Spell(), obs.Source(), caster, spell, source)
		}
	})
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-012 to FR-025 - Dispel Magic over a registry built from untrusted levels and dice never panics; the spells it ends and the spells that survive partition the target's spells, a spell cast at the slot level or lower ends without a die, and a refused command is a result, not an error.
// Path: unhappy (boundary).
func FuzzDispelOngoingSpells(f *testing.F) {
	f.Add(3, 1, 2, 5, 14, 13)
	f.Add(9, 9, 9, 9, 20, 1)
	f.Add(0, 0, 0, 0, 0, 0)
	f.Add(-4, 12, 3, 99, -3, 21)
	f.Fuzz(func(t *testing.T, slot, l1, l2, l3, score, face int) {
		target := gapMust(combat.NewDispelCreature(combat.MustCombatantID("target")))
		var held []combat.HeldSpell
		for i, l := range []int{l1, l2, l3} {
			lvl, err := content.NewSpellLevel(l)
			if err != nil {
				continue
			}
			id := gapMust(combat.NewOngoingEffectID(string(rune('a' + i))))
			eff := gapMust(combat.NewOngoingEffect(id, combat.OngoingEffectSpell, gapMust(combat.SpellSource(content.SpellID("bless")))))
			sp, err := combat.NewOngoingSpell(combat.OngoingSpellParams{Effect: eff, CastAtLevel: lvl, Origin: combat.SpellOriginOrdinary,
				Remaining: gapMust(combat.NewSpellTimeRemaining(60)), Restriction: combat.DispelUnrestricted{}})
			if err == nil {
				held = append(held, combat.HeldSpell{Target: target, Spell: sp})
			}
		}
		reg := gapMust(combat.NewOngoingSpellRegistry(held))
		slotLevel, err := character.NewSlotLevel(slot)
		if err != nil {
			return
		}
		sc, err := ability.NewScore(score)
		if err != nil {
			return
		}
		use := gapMust(combat.NewDispelOngoingSpells(dicetest.NewFixedRoller(gapRepeat(dieFace(face), 12)...)))
		res, err := use.Run(context.Background(), combat.DispelOngoingSpellsCommand{Registry: reg, Target: target, CastLevel: slotLevel,
			AbilityModifier: sc.Modifier(), Modifiers: d20.NewRollModifiers(modifier.New(0))})
		if err != nil {
			return // a roller refusal or a cancelled context: an error is allowed, a panic is not
		}
		got, ok := res.(combat.OngoingSpellsDispelled)
		if !ok {
			return // DispelOngoingSpellsInvalid is a result
		}
		c := got.Conclusion()
		if len(c.Ended())+len(c.Survived()) != len(held) || len(c.Registry().SpellsOn(target)) != len(c.Survived()) {
			t.Fatalf("ended %d + survived %d != %d held, registry keeps %d", len(c.Ended()), len(c.Survived()), len(held), len(c.Registry().SpellsOn(target)))
		}
		for _, e := range c.Ended() {
			if int(e.CastAtLevel()) > slot && len(got.Checks()) == 0 {
				t.Fatalf("a level %d spell ended with a level %d slot and no check", e.CastAtLevel(), slot)
			}
		}
	})
}

func gapMust[T any](v T, err error) T {
	if err != nil {
		panic(err) // a broken fixture, not a failure of the code under test
	}
	return v
}

// dieFace maps any int to a face of a d20.
func dieFace(n int) int {
	if n < 0 {
		n = -(n + 1)
	}
	return n%20 + 1
}

func gapRepeat(n, times int) []int {
	out := make([]int, times)
	for i := range out {
		out[i] = n
	}
	return out
}
