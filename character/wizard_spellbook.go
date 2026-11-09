package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellCopyCost is the time and money copying spells into a book takes (SRD 5.2.1,
// "Expanding and Replacing a Spellbook", p. 78). Comparable.
type SpellCopyCost struct{ hours, gold int }

// Hours returns the hours of transcription.
func (c SpellCopyCost) Hours() int { return c.hours }

// GoldPieces returns the cost in GP.
func (c SpellCopyCost) GoldPieces() int { return c.gold }

func copyCost(levels, hoursPer, goldPer int) SpellCopyCost {
	return SpellCopyCost{hours: levels * hoursPer, gold: levels * goldPer}
}

// SpellbookSource is why spells were added to a spellbook. The zero value is not a source.
type SpellbookSource int

// The sources. The last two belong to the Evoker.
const (
	SpellbookStart SpellbookSource = iota + 1
	SpellbookLevelUp
	SpellbookCopy
	SpellbookEvocationSavant
	SpellbookEvokerLevel
	SpellbookReplacement
)

// String returns "start", "level_up", "copy", "evocation_savant", "evoker_level", "replacement" or "unknown".
func (s SpellbookSource) String() string {
	return kindName([]string{"start", "level_up", "copy", "evocation_savant", "evoker_level", "replacement"}, int(s))
}

// SpellbookChanged is the outcome of a spellbook operation the rules allowed: the new
// Spellbook, the cost where the operation has one, and the events. Only the
// SpellbookRules functions and the use cases build one.
type SpellbookChanged struct {
	book   Spellbook
	cost   SpellCopyCost
	costed bool
	events []Event
}

// Spellbook returns the new Spellbook.
func (c SpellbookChanged) Spellbook() Spellbook { return NewSpellbook(c.book.Spells()) }

// Cost returns the time and money of a copy and true, or false for an operation without a cost.
func (c SpellbookChanged) Cost() (SpellCopyCost, bool) { return c.cost, c.costed }

// Events returns a new slice of the events: one SpellsAddedToSpellbook, or none for CopySpellbook.
func (c SpellbookChanged) Events() []Event { return slices.Clone(c.events) }

// SpellbookRules are the stateless Wizard operations over a Spellbook. Each returns the
// outcome and a SpellbookRejection; a non-zero Kind means rejected, with no cost and no
// change. No operation returns a spellbook holding a cantrip, a spell off the Wizard
// list or a repeated spell. The zero value is ready to use.
type SpellbookRules struct{}

// checkSpells validates facts in the order of SpellbookRejectionKind: the count (when
// count is positive), repeats, cantrips, the Wizard list, the highest Spell Level and
// spells already in book, naming the first offender in command order.
func checkSpells(facts []WizardSpellFacts, book Spellbook, highest, count int) SpellbookRejection {
	if count > 0 && len(facts) != count {
		return SpellbookRejection{Kind: SpellbookWrongSpellCount, Allowed: count}
	}
	if id, dup := firstRepeat(wizardFactIDs(facts)); dup {
		return SpellbookRejection{Kind: SpellbookDuplicateSpell, Spell: id}
	}
	for _, f := range facts {
		if f.IsCantrip() {
			return SpellbookRejection{Kind: SpellbookIsACantrip, Spell: f.ID}
		}
	}
	for _, f := range facts {
		if !f.OnWizardList() {
			return SpellbookRejection{Kind: SpellbookNotOnWizardList, Spell: f.ID}
		}
	}
	for _, f := range facts {
		if int(f.Level) > highest {
			return SpellbookRejection{Kind: SpellbookAboveHighestSpellLevel, Spell: f.ID, Highest: highest}
		}
	}
	for _, f := range facts {
		if book.Contains(f.ID) {
			return SpellbookRejection{Kind: SpellbookAlreadyInSpellbook, Spell: f.ID}
		}
	}
	return SpellbookRejection{}
}

func added(id Identifier, source SpellbookSource, ids []content.SpellID) []Event {
	return []Event{SpellsAddedToSpellbook{id: id, source: source, spells: slices.Clone(ids)}}
}

// Start returns the starting spellbook: exactly six distinct level 1 spells on the Wizard
// list (SRD 5.2.1, "Level 1: Spellcasting", Spellbook, p. 78).
func (SpellbookRules) Start(id Identifier, spells []WizardSpellFacts) (SpellbookChanged, SpellbookRejection) {
	if rej := checkSpells(spells, Spellbook{}, 1, 6); rej.Kind != 0 {
		return SpellbookChanged{}, rej
	}
	ids := wizardFactIDs(spells)
	return SpellbookChanged{book: NewSpellbook(ids), events: added(id, SpellbookStart, ids)}, SpellbookRejection{}
}

// AddLevelUp adds the two spells a Wizard level after 1 owes: two distinct spells not in
// the book, each up to the highest Spell Level at the Wizard level. Wizard level 1
// owes none and is FeatureNotAvailable.
func (SpellbookRules) AddLevelUp(id Identifier, wizardLevel int, book Spellbook, spells []WizardSpellFacts, highest int) (SpellbookChanged, SpellbookRejection) {
	if wizardRules := (WizardRules{}); wizardRules.SpellsOwedAtLevelUp(wizardLevel) == 0 {
		return SpellbookChanged{}, SpellbookRejection{Kind: SpellbookFeatureNotAvailable}
	}
	if rej := checkSpells(spells, book, highest, 2); rej.Kind != 0 {
		return SpellbookChanged{}, rej
	}
	ids := wizardFactIDs(spells)
	return SpellbookChanged{book: NewSpellbook(append(book.Spells(), ids...)), events: added(id, SpellbookLevelUp, ids)}, SpellbookRejection{}
}

// CopyIn copies a found level 1+ Wizard spell into the book at 2 hours and 50 GP per spell level.
func (SpellbookRules) CopyIn(id Identifier, book Spellbook, spell WizardSpellFacts, highest int) (SpellbookChanged, SpellbookRejection) {
	if rej := checkSpells([]WizardSpellFacts{spell}, book, highest, 0); rej.Kind != 0 {
		return SpellbookChanged{}, rej
	}
	return SpellbookChanged{book: NewSpellbook(append(book.Spells(), spell.ID)), cost: copyCost(int(spell.Level), 2, 50), costed: true,
		events: added(id, SpellbookCopy, []content.SpellID{spell.ID})}, SpellbookRejection{}
}

// CopyOut copies spells from the source book into a new one at 1 hour and 10 GP per spell level.
// A spell that is not in the source is NotInSourceSpellbook. Copying no spells costs nothing
// and returns an empty spellbook. It produces no event.
func (SpellbookRules) CopyOut(source Spellbook, spells []WizardSpellFacts) (SpellbookChanged, SpellbookRejection) {
	var ids []content.SpellID
	levels := 0
	for _, f := range spells {
		if !source.Contains(f.ID) {
			return SpellbookChanged{}, SpellbookRejection{Kind: SpellbookNotInSourceSpellbook, Spell: f.ID}
		}
	}
	for _, f := range spells {
		if !slices.Contains(ids, f.ID) {
			ids = append(ids, f.ID)
			levels += int(f.Level)
		}
	}
	return SpellbookChanged{book: NewSpellbook(ids), cost: copyCost(levels, 1, 10), costed: true}, SpellbookRejection{}
}

// ReplaceLost transcribes the level 1+ Wizard spells the character has prepared into a new
// spellbook at the Copying the Book cost; no prepared spells gives an empty book at no cost.
func (SpellbookRules) ReplaceLost(id Identifier, prepared []WizardSpellFacts) SpellbookChanged {
	var ids []content.SpellID
	levels := 0
	for _, f := range prepared {
		if f.IsCantrip() || !f.OnWizardList() || slices.Contains(ids, f.ID) {
			continue
		}
		ids = append(ids, f.ID)
		levels += int(f.Level)
	}
	c := SpellbookChanged{book: NewSpellbook(ids), cost: copyCost(levels, 1, 10), costed: true}
	if len(ids) > 0 {
		c.events = added(id, SpellbookReplacement, ids)
	}
	return c
}
