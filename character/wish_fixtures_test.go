package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// wishSpec is the spec every Wish test cites.
const wishSpec = "doc/specs/wish-spell.md"

// wishWizard is the baseline of the Wish tables: a Wizard 17 named caster with Strength 14, no stress.
func wishWizard(t *testing.T) character.Character {
	t.Helper()
	lv := levels(cl("wizard", 17))
	c, err := character.Restore(character.Parts{
		ID: caster, ClassLevels: lv, Species: "human", Background: "sage", AbilityScores: scoresOf(t, 14, 10, 10, 10, 10, 10),
		HitPointMaximum: character.MustHitPointMaximum(80), HitDicePool: character.MustHitDicePoolFromLevels(lv, map[content.ClassID]content.HitPointDie{"wizard": dieOf("wizard")}),
		HeroicInspiration: character.NotInspired,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// stressed is c holding the stress of weakened days, the cast damage and the lockout.
func stressed(t *testing.T, c character.Character, days int, active, locked bool) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.WishStressKey(), wishStress(t, days, active, locked)))
}

func wishStress(t *testing.T, days int, active, locked bool) character.WishStress {
	t.Helper()
	var w character.WishWeakness = character.NoWishWeakness{}
	if days > 0 {
		w = must(character.NewWishWeakened(days))
	}
	var d character.WishCastDamage = character.WishCastDamageInactive{}
	if active {
		d = character.WishCastDamageActive{}
	}
	var a character.WishCastingAbility = character.CanCastWish{}
	if locked {
		a = character.CannotCastWishAgain{}
	}
	return must(character.NewWishStress(w, d, a))
}

// stressOfChar reads the stress c holds.
func stressOfChar(t *testing.T, c character.Character) character.WishStress {
	t.Helper()
	s, _, err := character.ReadFeatureState(c.FeatureStates(), character.WishStressKey())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func gpCost(n int) content.Cost { return content.MustCost(n, content.CoinGP, 0) }

func wishCreatures(t *testing.T, n int) []character.WishCreature {
	t.Helper()
	out := make([]character.WishCreature, n)
	for i := range out {
		out[i] = must(character.NewWishCreature(string(rune('a'+i/10)) + string(rune('a'+i%10))))
	}
	return out
}

func statement(t *testing.T, s string) character.WishStatement {
	t.Helper()
	return must(character.NewWishStatement(s))
}

// wishWizardState is a Wizard 17 with one prepared spell of each level 1 to 9 and a Rituals (levels 1 and 2) in the book.
func wishWizardState(t *testing.T) character.SpellcastingState {
	t.Helper()
	var prepared []content.SpellID
	for lvl := 1; lvl <= 9; lvl++ {
		prepared = append(prepared, spellID(lvl, 1))
	}
	return castingState(t, classLevelsOf(t, "wizard", 17), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 5),
		Prepared: prepared, Spellbook: character.NewSpellbook(append(cloneIDs(prepared), ritualInBook, spellID(2, 9)))})
}

// wishCastOf casts for the Wizard 17 and returns the SpellCast event the stress damage reads, with the spell.
func wishCastOf(t *testing.T, cc castCase) (character.SpellCast, content.Spell) {
	t.Helper()
	cc.state = wishWizardState(t)
	got, err := cc.run(t)
	if err != nil {
		t.Fatal(err)
	}
	var events []character.Event
	switch r := got.(type) {
	case character.CastSpellCast:
		events = r.Events()
	case character.CastSpellStarted:
		done, err := character.NewCompleteCasting().Do(t.Context(), character.CompleteCastingCommand{State: r.State(), Spell: cc.spell, Context: castContext(t)})
		completed, ok := done.(character.CompleteCastingCompleted)
		if err != nil || !ok {
			t.Fatalf("complete = %#v, %v", done, err)
		}
		events = completed.Events()
	default:
		t.Fatalf("cast = %#v", got)
	}
	for _, e := range events {
		if sc, ok := e.(character.SpellCast); ok {
			return sc, cc.spell
		}
	}
	t.Fatalf("no SpellCast event in %v", events)
	return character.SpellCast{}, content.Spell{}
}

// slotCast is a level spellLevel spell cast with a slot of slotLevel.
func slotCast(t *testing.T, spellLevel, slotLevel int) (character.SpellCast, content.Spell) {
	t.Helper()
	return wishCastOf(t, castCase{spell: castSpell(t, spellID(spellLevel, 1), spellLevel), slot: choice(slotLevel)})
}

// scriptedRoller answers with the listed die faces, in order.
func scriptedRoller(faces ...int) character.Roller { return dicetest.NewFixedRoller(faces...) }
