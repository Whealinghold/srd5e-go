package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// scrollOf builds a scroll of a synthetic spell of the given level on the listed classes.
func scrollOf(tb testing.TB, level int, origin character.SpellScrollOrigin, classes ...string) character.SpellScroll {
	tb.Helper()
	id := content.MustSpellID("scroll_spell")
	set, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":1,"spells":[` + spellJSON(spellSpec{id: id.String(), level: level, classes: classes}) + `]}`))
	if err != nil {
		tb.Fatalf("fixture spell: %v", err)
	}
	return must(character.NewSpellScroll(set.Entries()[0].(content.Spell), origin))
}

func classRoute(class string, a ability.Ability) character.ScrollClassRoute {
	return must(character.NewScrollClassRoute(content.MustClassID(class), a))
}

// readerFacts is a reader who understands a written language, with the given routes.
func readerFacts(slots int, routes ...character.ScrollClassRoute) character.ScrollReaderFacts {
	return character.ScrollReaderFacts{UnderstandsWrittenLanguage: true, HighestSlotLevel: content.SpellLevel(slots), ClassRoutes: routes}
}

func w5() character.ScrollReaderFacts {
	return readerFacts(3, classRoute("wizard", ability.Intelligence))
}

func useMagicDevice() character.AnyScrollRoute {
	return must(character.NewAnyScrollRoute(ability.Intelligence, 1, content.SkillArcana))
}

func intOption() character.ScrollCheckOption {
	return must(character.NewScrollCheckOption(ability.Intelligence, 0))
}

// abilityCheck is an ability check with the natural value total against dc.
func abilityCheck(tb testing.TB, kind d20.Kind, total, dc int) d20.CheckResult {
	tb.Helper()
	roll := must(d20.NewRoll(modifier.Normal, []int{total}))
	return must(d20.ResolveCheck(kind, roll, d20.NewRollModifiers(modifier.New(0)), d20.DifficultyClass(dc)))
}

func scribedStats(tb testing.TB, level int) character.ScribedScrollStats {
	tb.Helper()
	l := character.MustLevel(level)
	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(l))
	return must(character.NewScribedScrollStats(stats, l))
}

func scribedOrigin(tb testing.TB, level int) character.SpellScrollOrigin {
	tb.Helper()
	return must(character.NewScribedScrollOrigin(scribedStats(tb, level)))
}
