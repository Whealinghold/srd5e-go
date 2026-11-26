package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// spellModifiersSpec names the spec every spell modifier test cites.
const spellModifiersSpec = "doc/specs/spell-granted-modifiers.md "

// smSrc returns a source of spell cast by caster on target at order, with the choice.
func smSrc(tb testing.TB, spell content.SpellID, caster, target string, order int, choice combat.SpellModifierChoice) combat.SpellModifierSource {
	tb.Helper()
	return luMust(combat.NewSpellModifierSource(spell, cid(caster), cid(target), order, choice))
}

// smPlain returns a source with no choice cast by "c" on "a".
func smPlain(tb testing.TB, spell content.SpellID, order int) combat.SpellModifierSource {
	tb.Helper()
	return smSrc(tb, spell, "c", "a", order, combat.NoChoice{})
}
