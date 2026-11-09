package character_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func warSpells(tb testing.TB, ids ...string) []content.Spell {
	tb.Helper()
	var out []content.Spell
	for _, id := range ids {
		out = append(out, warSpell(tb, id))
	}
	return out
}

var (
	warBookCantrips = []string{"acid_splash", "dancing_lights", "druidcraft"}
	warBookSpells   = []string{"detect_magic", "speak_with_animals"}
)

// Requirement: doc/specs/class-warlock.md FR-097, FR-098, FR-099, FR-101, FR-152 - a Book of Shadows is conjured after a Short or Long Rest with exactly three cantrips and two Ritual level 1 spells that are not already prepared; the page exists only when the Warlock knew Gift of the Protectors; conjuring while holding a book emits BookOfShadowsLost before BookOfShadowsConjured, and the replaced book's spells do not count as prepared; the input slices are not changed.
// Path: per case (happy).
func TestConjureBook(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		s    character.WarlockState
		rest character.RestKind
		page bool
	}{{"short rest", warKnows("pact_of_the_tome"), character.ShortRest, false}, {"long rest with the page", warKnows("pact_of_the_tome", "gift_of_the_protectors"), character.LongRest, true}} {
		prepared := []content.SpellID{"bless"}
		ch := warPactChanged(t, character.ConjureBook(warID, c.s, character.WarlockPactState{}, c.rest, warSpells(t, warBookCantrips...), warSpells(t, warBookSpells...), prepared))
		book := ch.State().Book()
		ev, ok := ch.Events()[0].(character.BookOfShadowsConjured)
		if len(ch.Events()) != 1 || !ok || !ev.Book().Conjured() || ev.CharacterID() != warID || book.HasPage() != c.page ||
			!slices.Equal(book.Cantrips(), []content.SpellID{"acid_splash", "dancing_lights", "druidcraft"}) || !slices.Equal(book.Spells(), []content.SpellID{"detect_magic", "speak_with_animals"}) || len(book.Names()) != 0 {
			t.Errorf("%s: book = %+v, events %v", c.name, book, ch.Events())
		}
		if !slices.Equal(prepared, []content.SpellID{"bless"}) {
			t.Errorf("%s: the prepared slice changed: %v", c.name, prepared)
		}
	}
	old := warPact(character.WarlockPactStateParts{BookCantrips: []content.SpellID{"druidcraft", "fire_bolt", "ray_of_frost"}, BookSpells: []content.SpellID{"detect_magic", "alarm"}, BookPage: true,
		PageNames: []character.Identifier{warCreature}})
	ch := warPactChanged(t, character.ConjureBook(warID, warKnows("pact_of_the_tome"), old, character.ShortRest, warSpells(t, warBookCantrips...), warSpells(t, warBookSpells...),
		[]content.SpellID{"druidcraft", "detect_magic"}))
	lost, ok := ch.Events()[0].(character.BookOfShadowsLost)
	if _, ok2 := ch.Events()[1].(character.BookOfShadowsConjured); len(ch.Events()) != 2 || !ok || !ok2 || lost.Cause() != character.BookLostToAnotherBook || ch.State().Book().HasPage() || len(ch.State().Book().Names()) != 0 {
		t.Errorf("replacing: events %v, book %+v", ch.Events(), ch.State().Book())
	}
}

// Requirement: doc/specs/class-warlock.md FR-097, FR-098, FR-099 - the Book of Shadows is refused without Pact of the Tome, outside a rest, with 2 or 4 cantrips or 1 or 3 level 1 spells (detail: expected 3 and 2, and the given count), a level 1 spell without the Ritual tag, a spell already prepared or chosen twice, and a spell of the wrong level (detail: the SpellID and the BookSpellRule); nothing changes.
// Path: per case (unhappy: boundary, invalid input, state error).
func TestConjureBookRefusals(t *testing.T) {
	t.Parallel()
	knows := warKnows("pact_of_the_tome")
	rule := func(id string, r character.BookSpellRule) character.WarlockRejection {
		return character.WarlockRejection{Kind: character.WarlockBookSpellRefused, Spell: content.SpellID(id), BookRule: r}
	}
	cases := []struct {
		name             string
		s                character.WarlockState
		rest             character.RestKind
		cantrips, spells []string
		prepared         []content.SpellID
		want             character.WarlockRejection
	}{
		{"not known", warKnows("pact_of_the_chain"), character.ShortRest, warBookCantrips, warBookSpells, nil, character.WarlockRejection{Kind: character.WarlockInvocationNotKnown, Option: "pact_of_the_tome"}},
		{"no rest", knows, 0, warBookCantrips, warBookSpells, nil, character.WarlockRejection{Kind: character.WarlockNotAfterRest}},
		{"two cantrips", knows, character.ShortRest, warBookCantrips[:2], warBookSpells, nil, character.WarlockRejection{Kind: character.WarlockCantripCountWrong, Required: 3, Given: 2}},
		{"four cantrips", knows, character.ShortRest, append(slices.Clone(warBookCantrips), "fire_bolt"), warBookSpells, nil, character.WarlockRejection{Kind: character.WarlockCantripCountWrong, Required: 3, Given: 4}},
		{"one spell", knows, character.ShortRest, warBookCantrips, warBookSpells[:1], nil, character.WarlockRejection{Kind: character.WarlockSpellCountWrong, Required: 2, Given: 1}},
		{"three spells", knows, character.ShortRest, warBookCantrips, append(slices.Clone(warBookSpells), "detect_magic"), nil, character.WarlockRejection{Kind: character.WarlockSpellCountWrong, Required: 2, Given: 3}},
		{"not a ritual", knows, character.ShortRest, warBookCantrips, []string{"detect_magic", "bless"}, nil, rule("bless", character.BookSpellNotRitual)},
		{"cantrip already prepared", knows, character.ShortRest, warBookCantrips, warBookSpells, []content.SpellID{"druidcraft"}, rule("druidcraft", character.BookSpellAlreadyPrepared)},
		{"spell already prepared", knows, character.ShortRest, warBookCantrips, warBookSpells, []content.SpellID{"speak_with_animals"}, rule("speak_with_animals", character.BookSpellAlreadyPrepared)},
		{"chosen twice", knows, character.ShortRest, []string{"acid_splash", "acid_splash", "druidcraft"}, warBookSpells, nil, rule("acid_splash", character.BookSpellAlreadyPrepared)},
		{"cantrip of level 1", knows, character.ShortRest, []string{"acid_splash", "dancing_lights", "bless"}, warBookSpells, nil, rule("bless", character.BookSpellWrongLevel)},
		{"spell of level 0", knows, character.ShortRest, warBookCantrips, []string{"detect_magic", "fire_bolt"}, nil, rule("fire_bolt", character.BookSpellWrongLevel)},
	}
	for _, c := range cases {
		got := warPactRefused(t, character.ConjureBook(warID, c.s, character.WarlockPactState{}, c.rest, warSpells(t, c.cantrips...), warSpells(t, c.spells...), c.prepared))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: rejection = %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-100, FR-101 - while the book is on the Warlock's person its five spells are prepared and it serves as a Spellcasting Focus; elsewhere, or with no book, neither; EndBookOfShadows removes the book with its cause in BookOfShadowsLost and does nothing without a book.
// Path: per case (happy; unhappy: state error).
func TestApplyAndEndBookOfShadows(t *testing.T) {
	t.Parallel()
	pact := warPact(warBook(true, warCreature))
	on := character.ApplyBookOfShadows(pact.Book(), character.BookOnPerson)
	if !on.ServesAsFocus || !slices.Equal(on.PreparedSpells, []content.SpellID{"light", "mage_hand", "prestidigitation", "alarm", "detect_magic"}) {
		t.Errorf("on person = %+v", on)
	}
	on.PreparedSpells[0] = "changed"
	if pact.Book().Cantrips()[0] != "light" {
		t.Error("PreparedSpells must be a copy")
	}
	for name, e := range map[string]character.BookOfShadowsEffect{"elsewhere": character.ApplyBookOfShadows(pact.Book(), character.BookElsewhere),
		"no book": character.ApplyBookOfShadows(character.BookOfShadows{}, character.BookOnPerson)} {
		if e.ServesAsFocus || e.PreparedSpells == nil || len(e.PreparedSpells) != 0 {
			t.Errorf("%s = %+v, want nothing", name, e)
		}
	}
	for _, cause := range []character.BookLossCause{character.BookLostToAnotherBook, character.BookLostToDeath} {
		ch := character.EndBookOfShadows(warID, pact, cause)
		ev, ok := ch.Events()[0].(character.BookOfShadowsLost)
		if !ok || ev.Cause() != cause || ev.CharacterID() != warID || ch.State().Book().Conjured() || len(ch.Events()) != 1 {
			t.Errorf("cause %v: events %v, book %+v", cause, ch.Events(), ch.State().Book())
		}
	}
	if ch := character.EndBookOfShadows(warID, character.WarlockPactState{}, character.BookLostToDeath); len(ch.Events()) != 0 {
		t.Errorf("no book: events %v", ch.Events())
	}
}

// Requirement: doc/specs/class-warlock.md FR-102, FR-152, FR-153 - a name is written on the page only with the invocation, a page, permission and an action, up to the Charisma modifier (minimum 1) names (Charisma 10: one; Charisma 16: three, the fourth is refused with the capacity as detail); a name already there is refused; each refusal changes nothing and the checks run in that order.
// Path: per case (happy; unhappy: boundary, state error).
func TestWriteGiftOfTheProtectorsName(t *testing.T) {
	t.Parallel()
	knows := warKnows("gift_of_the_protectors")
	page := warPact(warBook(true))
	one, two := character.MustIdentifier("npc-1"), character.MustIdentifier("npc-2")
	write := func(s character.WarlockState, p character.WarlockPactState, who character.Identifier, f character.PageWriteFacts, cha int) character.WarlockPactTransition {
		return character.WriteGiftOfTheProtectorsName(warID, s, p, who, f, warMod(cha))
	}
	ch := warPactChanged(t, write(knows, page, one, character.PageWriteFacts{}, 10))
	ev, ok := ch.Events()[0].(character.GiftOfTheProtectorsNameWritten)
	if !ok || ev.Creature() != one || ev.CharacterID() != warID || !slices.Equal(ch.State().Book().Names(), []character.Identifier{one}) || len(page.Book().Names()) != 0 {
		t.Errorf("first name: events %v, names %v", ch.Events(), ch.State().Book().Names())
	}
	for _, c := range []struct {
		name string
		s    character.WarlockState
		pact character.WarlockPactState
		who  character.Identifier
		f    character.PageWriteFacts
		cha  int
		want character.WarlockRejection
	}{
		{"not known", warKnows("pact_of_the_tome"), page, one, character.PageWriteFacts{}, 10, character.WarlockRejection{Kind: character.WarlockInvocationNotKnown, Option: "gift_of_the_protectors"}},
		{"no page", knows, warPact(warBook(false)), one, character.PageWriteFacts{}, 10, character.WarlockRejection{Kind: character.WarlockNoPage}},
		{"no book", knows, character.WarlockPactState{}, one, character.PageWriteFacts{}, 10, character.WarlockRejection{Kind: character.WarlockNoPage}},
		{"no permission", knows, page, one, character.PageWriteFacts{Permission: character.PermissionNotGiven}, 10, character.WarlockRejection{Kind: character.WarlockPermissionNotGiven}},
		{"no action", knows, page, one, character.PageWriteFacts{Action: character.ActionSpent}, 10, character.WarlockRejection{Kind: character.WarlockNoActionAvailable}},
		{"already there", knows, warPact(warBook(true, one)), one, character.PageWriteFacts{}, 16, character.WarlockRejection{Kind: character.WarlockNameAlreadyOnPage}},
		{"full at Charisma 10", knows, warPact(warBook(true, one)), two, character.PageWriteFacts{}, 10, character.WarlockRejection{Kind: character.WarlockPageFull, Required: 1}},
		{"full at Charisma 8", knows, warPact(warBook(true, one)), two, character.PageWriteFacts{}, 8, character.WarlockRejection{Kind: character.WarlockPageFull, Required: 1}},
		{"full at Charisma 16", knows, warPact(warBook(true, one, two, character.MustIdentifier("npc-3"))), character.MustIdentifier("npc-4"), character.PageWriteFacts{}, 16, character.WarlockRejection{Kind: character.WarlockPageFull, Required: 3}},
	} {
		if got := warPactRefused(t, write(c.s, c.pact, c.who, c.f, c.cha)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: rejection = %+v, want %+v", c.name, got, c.want)
		}
	}
	three := warPactChanged(t, write(knows, warPact(warBook(true, one, two)), character.MustIdentifier("npc-3"), character.PageWriteFacts{}, 16))
	if len(three.State().Book().Names()) != 3 {
		t.Errorf("third name at Charisma 16: %v", three.State().Book().Names())
	}
}

// Requirement: doc/specs/class-warlock.md FR-103 - a name is erased by touching the page as a Magic action; erasing needs a page, the name on it and the Magic action, and leaves the other names.
// Path: per case (happy; unhappy: state error).
func TestEraseGiftOfTheProtectorsName(t *testing.T) {
	t.Parallel()
	one, two := character.MustIdentifier("npc-1"), character.MustIdentifier("npc-2")
	pact := warPact(warBook(true, one, two))
	ch := warPactChanged(t, character.EraseGiftOfTheProtectorsName(warID, pact, one, character.WarlockTurnFacts{}))
	ev, ok := ch.Events()[0].(character.GiftOfTheProtectorsNameErased)
	if !ok || ev.Creature() != one || ev.CharacterID() != warID || !slices.Equal(ch.State().Book().Names(), []character.Identifier{two}) || len(pact.Book().Names()) != 2 {
		t.Errorf("erase: events %v, names %v", ch.Events(), ch.State().Book().Names())
	}
	for name, c := range map[string]struct {
		pact character.WarlockPactState
		who  character.Identifier
		turn character.WarlockTurnFacts
		want character.WarlockRejectionKind
	}{
		"no page":      {warPact(warBook(false)), one, character.WarlockTurnFacts{}, character.WarlockNoPage},
		"not on page":  {pact, character.MustIdentifier("npc-9"), character.WarlockTurnFacts{}, character.WarlockNameNotOnPage},
		"magic action": {pact, one, character.WarlockTurnFacts{MagicAction: character.ActionSpent}, character.WarlockNoMagicAction},
	} {
		if got := warPactRefused(t, character.EraseGiftOfTheProtectorsName(warID, c.pact, c.who, c.turn)); got.Kind != c.want {
			t.Errorf("%s: %v, want %v", name, got.Kind, c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-103 - a named creature reduced to 0 Hit Points (not killed outright) drops to 1 once until a Long Rest: the use is spent and GiftOfTheProtectorsTriggered names the creature; a second trigger is refused as spent, a name not on the page, no page and a creature killed outright are refused, and the Long Rest restores the use.
// Path: per case (happy; unhappy: state error).
func TestResolveGiftOfTheProtectors(t *testing.T) {
	t.Parallel()
	pact := warPact(warBook(true, warCreature))
	ch := warChanged(t, character.ResolveGiftOfTheProtectors(warID, character.WarlockState{}, pact, warCreature, false))
	ev, ok := ch.Events()[0].(character.GiftOfTheProtectorsTriggered)
	if !ok || ev.Creature() != warCreature || ev.CharacterID() != warID || !ch.State().GiftOfTheProtectorsSpent() {
		t.Errorf("trigger: events %v, state %+v", ch.Events(), ch.State())
	}
	for name, c := range map[string]struct {
		s      character.WarlockState
		pact   character.WarlockPactState
		who    character.Identifier
		killed bool
		want   character.WarlockRejectionKind
	}{
		"spent":      {ch.State(), pact, warCreature, false, character.WarlockGiftAlreadySpent},
		"no page":    {character.WarlockState{}, warPact(warBook(false)), warCreature, false, character.WarlockNoPage},
		"not named":  {character.WarlockState{}, pact, character.MustIdentifier("npc-9"), false, character.WarlockNameNotOnPage},
		"killed out": {character.WarlockState{}, pact, warCreature, true, character.WarlockKilledOutright},
	} {
		if got := warRefused(t, character.ResolveGiftOfTheProtectors(warID, c.s, c.pact, c.who, c.killed)); got.Kind != c.want {
			t.Errorf("%s: %v, want %v", name, got.Kind, c.want)
		}
	}
	again := warChanged(t, must(character.ApplyWarlockRest(warID, ch.State(), character.LongRest)))
	if again.State().GiftOfTheProtectorsSpent() {
		t.Error("a Long Rest must restore the use")
	}
}
