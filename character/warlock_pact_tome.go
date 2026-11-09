package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

const giftOfTheProtectors content.FeatureOptionID = "gift_of_the_protectors"

const (
	bookCantripCount = 3
	bookSpellCount   = 2
)

// BookSpellRule is the rule a spell chosen for the Book of Shadows broke, a closed set (class-warlock.md FR-099).
type BookSpellRule int

const (
	// BookSpellNotRitual: a level 1 spell must have the Ritual tag.
	BookSpellNotRitual BookSpellRule = iota + 1
	// BookSpellAlreadyPrepared: the spell is already prepared (or chosen twice).
	BookSpellAlreadyPrepared
	// BookSpellWrongLevel: a cantrip must be level 0 and the other spells level 1.
	BookSpellWrongLevel
)

// ConjureBook conjures a Book of Shadows at the end of a Short or Long Rest (class-warlock.md FR-097 to FR-101, FR-152). cantrips and
// spells are the loaded spells chosen; prepared are the spells already prepared, known cantrips included (FR-099), and the spells of the
// book being replaced are not counted in it. The book has the Gift of the Protectors page when the Warlock knows that invocation. A
// book already held disappears first (BookOfShadowsLost, then BookOfShadowsConjured). It refuses with WarlockInvocationNotKnown (Option
// Pact of the Tome), WarlockNotAfterRest (rest is neither ShortRest nor LongRest), WarlockCantripCountWrong and WarlockSpellCountWrong
// (Required and Given) and WarlockBookSpellRefused (Spell and BookRule).
func ConjureBook(id Identifier, s WarlockState, pact WarlockPactState, rest RestKind, cantrips, spells []content.Spell, prepared []content.SpellID) WarlockPactTransition {
	switch {
	case !s.Knows(pactOfTheTome):
		return pactNotKnown(pactOfTheTome)
	case !rest.Valid():
		return pactRefused(warlockRefuse(WarlockNotAfterRest))
	case len(cantrips) != bookCantripCount:
		return pactRefused(WarlockRejection{Kind: WarlockCantripCountWrong, Required: bookCantripCount, Given: len(cantrips)})
	case len(spells) != bookSpellCount:
		return pactRefused(WarlockRejection{Kind: WarlockSpellCountWrong, Required: bookSpellCount, Given: len(spells)})
	}
	held := slices.Concat(prepared) // a copy, so the caller's slice is untouched
	for _, old := range slices.Concat(pact.book.cantrips, pact.book.spells) {
		held = slices.DeleteFunc(held, func(p content.SpellID) bool { return p == old })
	}
	var ids []content.SpellID
	for _, group := range []struct {
		spells []content.Spell
		level  content.SpellLevel
	}{{cantrips, 0}, {spells, 1}} {
		for _, sp := range group.spells {
			rule := BookSpellRule(0)
			switch {
			case sp.Level() != group.level:
				rule = BookSpellWrongLevel
			case group.level == 1 && !sp.Ritual():
				rule = BookSpellNotRitual
			case slices.Contains(held, sp.ID()) || slices.Contains(ids, sp.ID()):
				rule = BookSpellAlreadyPrepared
			}
			if rule != 0 {
				return pactRefused(WarlockRejection{Kind: WarlockBookSpellRefused, Spell: sp.ID(), BookRule: rule})
			}
			ids = append(ids, sp.ID())
		}
	}
	var events []Event
	if pact.book.Conjured() {
		events = append(events, BookOfShadowsLost{id: id, cause: BookLostToAnotherBook})
	}
	pact.book = BookOfShadows{cantrips: ids[:bookCantripCount], spells: ids[bookCantripCount:], page: s.Knows(giftOfTheProtectors)}
	return pactChanged(pact, append(events, BookOfShadowsConjured{id: id, book: pact.book})...)
}

// BookOfShadowsPossession says where the book is, a closed set. The zero value is read as BookOnPerson.
type BookOfShadowsPossession int

const (
	// BookOnPerson: the book is on the Warlock's person.
	BookOnPerson BookOfShadowsPossession = iota
	// BookElsewhere: the book is somewhere else.
	BookElsewhere
)

// BookOfShadowsEffect is what the book gives the Warlock now: the spells it keeps prepared and whether it serves as a Spellcasting Focus.
type BookOfShadowsEffect struct {
	// PreparedSpells are the cantrips and spells of the book, to pass to the spellcasting service as granted Warlock spells.
	PreparedSpells []content.SpellID
	// ServesAsFocus is true while the book is on the Warlock's person.
	ServesAsFocus bool
}

// ApplyBookOfShadows returns the spells prepared and the focus use while the book is on the Warlock's person, and neither while it is
// elsewhere or when there is no book (class-warlock.md FR-100). The spells count against neither the Prepared Spells number nor the
// cantrips known. It returns new slices.
func ApplyBookOfShadows(book BookOfShadows, possession BookOfShadowsPossession) BookOfShadowsEffect {
	if !book.Conjured() || possession != BookOnPerson {
		return BookOfShadowsEffect{PreparedSpells: []content.SpellID{}}
	}
	return BookOfShadowsEffect{PreparedSpells: slices.Concat(book.cantrips, book.spells), ServesAsFocus: true}
}

// EndBookOfShadows makes the book disappear because the Warlock conjured another or died, so its spells stop being prepared
// (class-warlock.md FR-101). It never refuses; with no book it changes nothing and has no events.
func EndBookOfShadows(id Identifier, pact WarlockPactState, cause BookLossCause) WarlockPactChanged {
	if !pact.book.Conjured() {
		return pactChanged(pact)
	}
	pact.book = BookOfShadows{}
	return pactChanged(pact, BookOfShadowsLost{id: id, cause: cause})
}

// PageWritePermission says whether the Warlock gave permission to write a name, a closed set. The zero value is read as PermissionGiven.
type PageWritePermission int

const (
	// PermissionGiven: the Warlock gave permission.
	PermissionGiven PageWritePermission = iota
	// PermissionNotGiven: the Warlock did not.
	PermissionNotGiven
)

// PageWriteFacts are the facts about a name being written, supplied by the host. Construct it with a keyed literal.
type PageWriteFacts struct {
	Permission PageWritePermission
	// Action says whether the writing creature has its action.
	Action WarlockActionAvailability
}

// WriteGiftOfTheProtectorsName writes a creature's name on the page (class-warlock.md FR-102, FR-152, FR-153). The page holds as many
// names as GiftOfTheProtectorsCapacity says for charisma. It refuses with WarlockInvocationNotKnown (Option Gift of the Protectors),
// WarlockNoPage, WarlockPermissionNotGiven, WarlockNoActionAvailable, WarlockNameAlreadyOnPage and WarlockPageFull (Required is the capacity).
func WriteGiftOfTheProtectorsName(id Identifier, s WarlockState, pact WarlockPactState, creature Identifier, facts PageWriteFacts, charisma ability.Modifier) WarlockPactTransition {
	capacity := GiftOfTheProtectorsCapacity(charisma)
	switch {
	case !s.Knows(giftOfTheProtectors):
		return pactNotKnown(giftOfTheProtectors)
	case !pact.book.page:
		return pactRefused(warlockRefuse(WarlockNoPage))
	case facts.Permission == PermissionNotGiven:
		return pactRefused(warlockRefuse(WarlockPermissionNotGiven))
	case facts.Action == ActionSpent:
		return pactRefused(warlockRefuse(WarlockNoActionAvailable))
	case slices.Contains(pact.book.names, creature):
		return pactRefused(warlockRefuse(WarlockNameAlreadyOnPage))
	case len(pact.book.names) >= capacity:
		return pactRefused(WarlockRejection{Kind: WarlockPageFull, Required: capacity})
	}
	pact.book.names = append(slices.Clone(pact.book.names), creature)
	return pactChanged(pact, GiftOfTheProtectorsNameWritten{id: id, creature: creature})
}

// EraseGiftOfTheProtectorsName erases a creature's name by touching the page, as a Magic action (class-warlock.md FR-103). It refuses with
// WarlockNoPage, WarlockNameNotOnPage and WarlockNoMagicAction.
func EraseGiftOfTheProtectorsName(id Identifier, pact WarlockPactState, creature Identifier, turn WarlockTurnFacts) WarlockPactTransition {
	i := slices.Index(pact.book.names, creature)
	switch {
	case !pact.book.page:
		return pactRefused(warlockRefuse(WarlockNoPage))
	case i < 0:
		return pactRefused(warlockRefuse(WarlockNameNotOnPage))
	case turn.MagicAction == ActionSpent:
		return pactRefused(warlockRefuse(WarlockNoMagicAction))
	}
	pact.book.names = slices.Delete(slices.Clone(pact.book.names), i, i+1)
	return pactChanged(pact, GiftOfTheProtectorsNameErased{id: id, creature: creature})
}

// ResolveGiftOfTheProtectors lets a named creature reduced to 0 Hit Points drop to 1 instead, once until a Long Rest (class-warlock.md
// FR-103); the host sets the Hit Points. It refuses with WarlockNoPage, WarlockNameNotOnPage, WarlockKilledOutright and
// WarlockGiftAlreadySpent.
func ResolveGiftOfTheProtectors(id Identifier, s WarlockState, pact WarlockPactState, creature Identifier, killedOutright bool) WarlockTransition {
	switch {
	case !pact.book.page:
		return warlockRefused(warlockRefuse(WarlockNoPage))
	case !slices.Contains(pact.book.names, creature):
		return warlockRefused(warlockRefuse(WarlockNameNotOnPage))
	case killedOutright:
		return warlockRefused(warlockRefuse(WarlockKilledOutright))
	case s.protectorsSpent:
		return warlockRefused(warlockRefuse(WarlockGiftAlreadySpent))
	}
	s.protectorsSpent = true
	return warlockChanged(s, GiftOfTheProtectorsTriggered{id: id, creature: creature})
}
