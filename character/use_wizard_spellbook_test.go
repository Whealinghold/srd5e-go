package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-wizard.md FR-030, FR-033, FR-034, FR-035, FR-036 (tables 4.5.1, 4.5.2) - through the use cases a level 4 Wizard builds its book: the six-spell start, the two spells of a
// level up (rows LU-03 and LU-05 at the highest level) and a copy at 2 hours and 50 GP per level, each returning the new book and one event, and a refusal carrying no cost.
// Path: per case (happy / unhappy: boundary).
func TestSpellbookUseCases(t *testing.T) {
	t.Parallel()
	src := wzSource(t)
	ctx := t.Context()
	started, err := must(character.NewStartSpellbook(src)).Run(ctx, character.StartSpellbookCommand{CharacterID: wzID, ClassLevels: wzLevels(1), Spells: wzFacts("a", 1, 6)})
	book := started.(character.SpellbookChanged).Spellbook()
	if err != nil || len(book.Spells()) != 6 {
		t.Fatalf("StartSpellbook = %#v, %v", started, err)
	}
	for _, c := range []struct {
		name   string
		level  int
		spells []character.WizardSpellFacts
		okay   bool
	}{
		{"LU-03", 3, []character.WizardSpellFacts{wzFact("n1", 1), wzFact("n2", 2)}, true},
		{"LU-05", 3, []character.WizardSpellFacts{wzFact("n1", 3), wzFact("n2", 1)}, false},
		{"level 1", 1, []character.WizardSpellFacts{wzFact("n1", 1), wzFact("n2", 1)}, false},
	} {
		res, err := must(character.NewAddLevelUpSpells(src)).Run(ctx, character.AddLevelUpSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Spellbook: book, Spells: c.spells})
		if err != nil {
			t.Fatal(err)
		}
		if got, ok := res.(character.SpellbookChanged); ok != c.okay || (ok && len(got.Spellbook().Spells()) != 8) {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
	copied, err := must(character.NewCopySpellIntoBook(src)).Run(ctx, character.CopySpellIntoBookCommand{CharacterID: wzID, ClassLevels: wzLevels(5), Spellbook: book, Spell: wzFact("found", 3)})
	changed, ok := copied.(character.SpellbookChanged)
	cost, costed := changed.Cost()
	if err != nil || !ok || !costed || cost.Hours() != 6 || cost.GoldPieces() != 150 || !changed.Spellbook().Contains("found") {
		t.Fatalf("CopySpellIntoBook = %#v, %v", copied, err)
	}
	refused, _ := must(character.NewCopySpellIntoBook(src)).Run(ctx, character.CopySpellIntoBookCommand{CharacterID: wzID, ClassLevels: wzLevels(5), Spellbook: book, Spell: wzFact("found", 4)})
	if r, ok := refused.(character.CopySpellIntoBookRejected); !ok || r.Rejection.Kind != character.SpellbookAboveHighestSpellLevel || r.Rejection.Highest != 3 {
		t.Errorf("a copy above the highest level = %#v", refused)
	}
	if !slices.Equal(book.Spells(), started.(character.SpellbookChanged).Spellbook().Spells()) {
		t.Error("the input book changed")
	}
}

// Requirement: doc/specs/class-wizard.md FR-037, FR-038, FR-039 - CopySpellbook and ReplaceLostSpellbook are pure: copying the book costs 1 hour and 10 GP per level and a spell outside the source is NotInSourceSpellbook;
// a lost book is rebuilt from the prepared spells at the same rate, and nothing prepared gives an empty book at no cost.
// Path: per case (happy / unhappy: empty and minimal).
func TestCopySpellbookAndReplaceLost(t *testing.T) {
	t.Parallel()
	res, err := character.NewCopySpellbook().Run(t.Context(), character.CopySpellbookCommand{ClassLevels: wzLevels(3), Source: wzBook("a", "b"),
		Spells: []character.WizardSpellFacts{wzFact("a", 1), wzFact("b", 2)}})
	got, ok := res.(character.SpellbookChanged)
	cost, _ := got.Cost()
	if err != nil || !ok || cost.Hours() != 3 || cost.GoldPieces() != 30 || len(got.Events()) != 0 {
		t.Errorf("CopySpellbook = %#v, %v", res, err)
	}
	res, _ = character.NewCopySpellbook().Run(t.Context(), character.CopySpellbookCommand{ClassLevels: wzLevels(3), Source: wzBook("a"), Spells: []character.WizardSpellFacts{wzFact("z", 1)}})
	if r, ok := res.(character.CopySpellbookRejected); !ok || r.Rejection != (character.SpellbookRejection{Kind: character.SpellbookNotInSourceSpellbook, Spell: "z"}) {
		t.Errorf("CopySpellbook(z) = %#v", res)
	}
	lostRes, err := character.NewReplaceLostSpellbook().Run(t.Context(), character.ReplaceLostSpellbookCommand{CharacterID: wzID, ClassLevels: wzLevels(3), Prepared: wzFacts("p", 2, 2)})
	lost := lostRes.(character.SpellbookChanged)
	cost, _ = lost.Cost()
	if err != nil || len(lost.Spellbook().Spells()) != 2 || cost.Hours() != 4 || cost.GoldPieces() != 40 {
		t.Errorf("ReplaceLostSpellbook = %#v, %v", lostRes, err)
	}
	lostRes, _ = character.NewReplaceLostSpellbook().Run(t.Context(), character.ReplaceLostSpellbookCommand{CharacterID: wzID, ClassLevels: wzLevels(3)})
	if empty := lostRes.(character.SpellbookChanged); len(empty.Spellbook().Spells()) != 0 || len(empty.Events()) != 0 {
		t.Errorf("ReplaceLostSpellbook(nothing) = %#v", lostRes)
	}
}
