package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

type sbRej = character.SpellbookRejection

// Requirement: doc/specs/class-wizard.md FR-030, FR-031, FR-032 - StartSpellbook accepts exactly six distinct level 1 Wizard spells and adds them with one event; each
// cause is rejected in the order of SpellbookRejection (count, repeat, cantrip, list, level), the first offender in command order.
// Path: per case (happy / unhappy: invalid input).
func TestSpellbookRulesStart(t *testing.T) {
	t.Parallel()
	six := wzFacts("a", 1, 6)
	with := func(i int, f character.WizardSpellFacts) []character.WizardSpellFacts {
		out := slices.Clone(six)
		out[i] = f
		return out
	}
	for _, c := range []struct {
		name   string
		spells []character.WizardSpellFacts
		want   sbRej
	}{
		{"six", six, sbRej{}},
		{"five", six[:5], sbRej{Kind: character.SpellbookWrongSpellCount, Allowed: 6}},
		{"seven", append(slices.Clone(six), wzFact("x", 1)), sbRej{Kind: character.SpellbookWrongSpellCount, Allowed: 6}},
		{"none", nil, sbRej{Kind: character.SpellbookWrongSpellCount, Allowed: 6}},
		{"repeat", with(5, six[0]), sbRej{Kind: character.SpellbookDuplicateSpell, Spell: "aa"}},
		{"cantrip", with(2, wzFact("c", 0)), sbRej{Kind: character.SpellbookIsACantrip, Spell: "c"}},
		{"off list", with(1, wzFact("o", 1, wzOffList)), sbRej{Kind: character.SpellbookNotOnWizardList, Spell: "o"}},
		{"level 2", with(3, wzFact("h", 2)), sbRej{Kind: character.SpellbookAboveHighestSpellLevel, Spell: "h", Highest: 1}},
		{"cantrip before off list", with(4, wzFact("c", 0, wzOffList)), sbRej{Kind: character.SpellbookIsACantrip, Spell: "c"}},
		{"first offender", append(slices.Clone(six[:3]), wzFact("o1", 1, wzOffList), wzFact("o2", 1, wzOffList), six[5]), sbRej{Kind: character.SpellbookNotOnWizardList, Spell: "o1"}},
	} {
		got, rej := character.SpellbookRules{}.Start(wzID, c.spells)
		if rej != c.want {
			t.Errorf("%s: rejection = %+v, want %+v", c.name, rej, c.want)
			continue
		}
		if c.want.Kind != 0 {
			if len(got.Spellbook().Spells()) != 0 || len(got.Events()) != 0 {
				t.Errorf("%s: a rejection carries %+v", c.name, got)
			}
			continue
		}
		ev, ok := got.Events()[0].(character.SpellsAddedToSpellbook)
		if _, costed := got.Cost(); costed || len(got.Spellbook().Spells()) != 6 || !ok || ev.Source() != character.SpellbookStart || len(ev.Spells()) != 6 || ev.CharacterID() != wzID {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
	empty := character.NewSpellbook(nil)
	if len(empty.Spells()) != 0 {
		t.Error("an empty Spellbook must stay representable")
	}
}

// Requirement: doc/specs/class-wizard.md FR-033, FR-034 (table 4.5.1, LU-01 to LU-12) - the two spells a Wizard level after 1 owes are accepted up to the highest Spell Level and
// rejected, by cause, when they repeat, are already known, are cantrips, are off the list or are too high; level 1 owes none.
// Path: per case (happy / unhappy: boundary at the highest level).
func TestSpellbookRulesAddLevelUp(t *testing.T) {
	t.Parallel()
	known := wzBook("k1", "k2")
	two := func(a, b int) []character.WizardSpellFacts {
		return []character.WizardSpellFacts{wzFact("n1", a), wzFact("n2", b)}
	}
	for _, c := range []struct {
		label          string
		level, highest int
		spells         []character.WizardSpellFacts
		want           sbRej
	}{
		{"LU-01", 2, 1, two(1, 1), sbRej{}},
		{"LU-02", 2, 1, two(1, 2), sbRej{Kind: character.SpellbookAboveHighestSpellLevel, Spell: "n2", Highest: 1}},
		{"LU-03", 3, 2, two(1, 2), sbRej{}},
		{"LU-04", 3, 2, two(2, 2), sbRej{}},
		{"LU-05", 3, 2, two(3, 1), sbRej{Kind: character.SpellbookAboveHighestSpellLevel, Spell: "n1", Highest: 2}},
		{"LU-06", 5, 3, two(3, 3), sbRej{}},
		{"LU-07", 17, 9, two(9, 9), sbRej{}},
		{"LU-08", 20, 9, two(9, 9)[:1], sbRej{Kind: character.SpellbookWrongSpellCount, Allowed: 2}},
		{"LU-09", 4, 2, []character.WizardSpellFacts{wzFact("n1", 1), wzFact("n1", 1)}, sbRej{Kind: character.SpellbookDuplicateSpell, Spell: "n1"}},
		{"LU-10", 4, 2, []character.WizardSpellFacts{wzFact("k1", 1), wzFact("n2", 2)}, sbRej{Kind: character.SpellbookAlreadyInSpellbook, Spell: "k1"}},
		{"LU-11", 4, 2, two(0, 1), sbRej{Kind: character.SpellbookIsACantrip, Spell: "n1"}},
		{"LU-12", 1, 1, two(1, 1), sbRej{Kind: character.SpellbookFeatureNotAvailable}},
		{"off list", 4, 2, []character.WizardSpellFacts{wzFact("n1", 1), wzFact("n2", 1, wzOffList)}, sbRej{Kind: character.SpellbookNotOnWizardList, Spell: "n2"}},
	} {
		got, rej := character.SpellbookRules{}.AddLevelUp(wzID, c.level, known, c.spells, c.highest)
		if rej != c.want {
			t.Errorf("%s: rejection = %+v, want %+v", c.label, rej, c.want)
			continue
		}
		if c.want.Kind == 0 {
			if !slices.Equal(got.Spellbook().Spells(), wzIDs("k1", "k2", "n1", "n2")) || len(got.Events()) != 1 {
				t.Errorf("%s: %+v", c.label, got)
			}
		}
	}
	if len(known.Spells()) != 2 {
		t.Error("AddLevelUp altered its input")
	}
}

// Requirement: doc/specs/class-wizard.md FR-035, FR-036 (table 4.5.2, CC-01 to CC-06) - copying a found spell into the book costs 2 hours and 50 GP per spell level and is rejected,
// without a cost, for a cantrip, an off-list spell, one above the highest level or one already in the book.
// Path: per case (happy / unhappy: boundary).
func TestSpellbookRulesCopyIn(t *testing.T) {
	t.Parallel()
	book := wzBook("k1")
	for _, c := range []struct {
		label       string
		spell       character.WizardSpellFacts
		highest     int
		hours, gold int
		want        sbRej
	}{
		{"CC-01", wzFact("s", 1), 1, 2, 50, sbRej{}},
		{"CC-02", wzFact("s", 3), 3, 6, 150, sbRej{}},
		{"CC-03", wzFact("s", 5), 5, 10, 250, sbRej{}},
		{"CC-04", wzFact("s", 9), 9, 18, 450, sbRej{}},
		{"CC-05", wzFact("s", 2), 1, 0, 0, sbRej{Kind: character.SpellbookAboveHighestSpellLevel, Spell: "s", Highest: 1}},
		{"CC-06", wzFact("s", 0), 3, 0, 0, sbRej{Kind: character.SpellbookIsACantrip, Spell: "s"}},
		{"off list", wzFact("s", 1, wzOffList), 3, 0, 0, sbRej{Kind: character.SpellbookNotOnWizardList, Spell: "s"}},
		{"already known", wzFact("k1", 1), 3, 0, 0, sbRej{Kind: character.SpellbookAlreadyInSpellbook, Spell: "k1"}},
	} {
		got, rej := character.SpellbookRules{}.CopyIn(wzID, book, c.spell, c.highest)
		cost, costed := got.Cost()
		if rej != c.want || costed != (c.want.Kind == 0) || cost.Hours() != c.hours || cost.GoldPieces() != c.gold {
			t.Errorf("%s = cost %+v (%v), %+v; want %dh %dGP, %+v", c.label, cost, costed, rej, c.hours, c.gold, c.want)
		}
		if c.want.Kind == 0 && (!got.Spellbook().Contains(c.spell.ID) || !got.Spellbook().Contains("k1")) {
			t.Errorf("%s: book = %v", c.label, got.Spellbook().Spells())
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-037, FR-038 (table 4.5.2, CC-07 to CC-10) - copying the book costs 1 hour and 10 GP per spell level, summed per spell; copying nothing is
// free and empty; a spell not in the source is rejected.
// Path: per case (happy / unhappy: boundary).
func TestSpellbookRulesCopyOut(t *testing.T) {
	t.Parallel()
	src := wzBook("a", "b", "c", "d")
	for _, c := range []struct {
		label       string
		spells      []character.WizardSpellFacts
		hours, gold int
		want        sbRej
	}{
		{"CC-07", []character.WizardSpellFacts{wzFact("a", 1)}, 1, 10, sbRej{}},
		{"CC-08", []character.WizardSpellFacts{wzFact("b", 3)}, 3, 30, sbRej{}},
		{"CC-09", []character.WizardSpellFacts{wzFact("c", 9)}, 9, 90, sbRej{}},
		{"CC-10", []character.WizardSpellFacts{wzFact("a", 1), wzFact("b", 2), wzFact("c", 3)}, 6, 60, sbRej{}},
		{"nothing", nil, 0, 0, sbRej{}},
		{"repeated counts once", []character.WizardSpellFacts{wzFact("a", 2), wzFact("a", 2)}, 2, 20, sbRej{}},
		{"not in source", []character.WizardSpellFacts{wzFact("a", 1), wzFact("z", 1)}, 0, 0, sbRej{Kind: character.SpellbookNotInSourceSpellbook, Spell: "z"}},
	} {
		got, rej := character.SpellbookRules{}.CopyOut(src, c.spells)
		cost, costed := got.Cost()
		if rej != c.want || costed != (c.want.Kind == 0) || cost.Hours() != c.hours || cost.GoldPieces() != c.gold || len(got.Events()) != 0 {
			t.Errorf("%s = cost %+v (%v), %+v; want %dh %dGP, %+v and no event", c.label, cost, costed, rej, c.hours, c.gold, c.want)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-039, FR-032 - a lost spellbook is replaced by the prepared level 1+ Wizard spells at the Copying the Book cost; cantrips, off-list and repeated
// spells are left out, and with none prepared the book is empty, free and silent.
// Path: per case (happy / unhappy: empty and minimal).
func TestSpellbookRulesReplaceLost(t *testing.T) {
	t.Parallel()
	got := character.SpellbookRules{}.ReplaceLost(wzID, []character.WizardSpellFacts{wzFact("a", 1), wzFact("b", 2), wzFact("a", 1), wzFact("c", 0), wzFact("o", 1, wzOffList)})
	cost, costed := got.Cost()
	ev, _ := got.Events()[0].(character.SpellsAddedToSpellbook)
	if !slices.Equal(got.Spellbook().Spells(), wzIDs("a", "b")) || !costed || cost.Hours() != 3 || cost.GoldPieces() != 30 || ev.Source() != character.SpellbookReplacement {
		t.Errorf("ReplaceLost = %+v", got)
	}
	none := character.SpellbookRules{}.ReplaceLost(wzID, nil)
	if cost, costed := none.Cost(); len(none.Spellbook().Spells()) != 0 || !costed || cost != (character.SpellCopyCost{}) || len(none.Events()) != 0 {
		t.Errorf("ReplaceLost(nothing) = %+v", none)
	}
}

// Requirement: doc/specs/class-wizard.md FR-091 - the Events and Spellbook of a result are copies; a SpellbookSource has its snake_case name.
// Path: unhappy (immutability).
func TestSpellbookChangedIsReadOnly(t *testing.T) {
	t.Parallel()
	got, _ := character.SpellbookRules{}.Start(wzID, wzFacts("a", 1, 6))
	got.Events()[0] = nil
	got.Spellbook().Spells()[0] = content.SpellID("zz")
	if got.Events()[0] == nil || got.Spellbook().Spells()[0] != "aa" {
		t.Error("a result accessor returned shared state")
	}
	names := map[character.SpellbookSource]string{character.SpellbookStart: "start", character.SpellbookLevelUp: "level_up", character.SpellbookCopy: "copy",
		character.SpellbookEvocationSavant: "evocation_savant", character.SpellbookEvokerLevel: "evoker_level", character.SpellbookReplacement: "replacement", 0: "unknown", 7: "unknown"}
	for s, want := range names {
		if s.String() != want {
			t.Errorf("SpellbookSource(%d) = %q, want %q", int(s), s.String(), want)
		}
	}
	ev := got.Events()[0].(character.SpellsAddedToSpellbook)
	ev.Spells()[0] = "zz"
	if ev.Spells()[0] != "aa" {
		t.Error("SpellsAddedToSpellbook.Spells returned shared state")
	}
}
