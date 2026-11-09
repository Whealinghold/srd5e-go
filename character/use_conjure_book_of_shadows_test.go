package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func warBookCmd(t *testing.T, c character.Character) character.ConjureBookOfShadowsCommand {
	t.Helper()
	return character.ConjureBookOfShadowsCommand{Character: c, Rest: character.LongRest, Cantrips: []content.SpellID{"acid_splash", "dancing_lights", "druidcraft"},
		Spells: []content.SpellID{"detect_magic", "speak_with_animals"}}
}

// Requirement: doc/specs/class-warlock.md FR-097 to FR-101, FR-152 - the use case loads the five chosen spells and stores the book, with the page when the Warlock knows Gift of the Protectors: the result and the stored state hold the same book, BookOfShadowsConjured is the event, a spell already prepared (Prepared) is refused, the Prepared slice is not changed and the command's Character is not altered.
// Path: per case (happy; unhappy: state error).
func TestConjureBookOfShadowsStoresTheBook(t *testing.T) {
	t.Parallel()
	uc := must(character.NewConjureBookOfShadows(warSource(t)))
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_tome", "gift_of_the_protectors"))
	cmd := warBookCmd(t, c)
	cmd.Prepared = []content.SpellID{"bless"}
	res, err := uc.Run(t.Context(), cmd)
	got, ok := res.(character.ConjureBookOfShadowsApplied)
	if err != nil || !ok {
		t.Fatalf("= %#v, %v", res, err)
	}
	pact, _, _ := character.ReadFeatureState(got.Character().FeatureStates(), character.WarlockPactStateKey())
	if !pact.Book().HasPage() || !slices.Equal(pact.Book().Spells(), []content.SpellID{"detect_magic", "speak_with_animals"}) || !got.Book().HasPage() || !slices.Equal(got.Book().Cantrips(), pact.Book().Cantrips()) {
		t.Errorf("book = %+v / %+v", pact.Book(), got.Book())
	}
	if ev, isEv := got.Events()[0].(character.BookOfShadowsConjured); !isEv || !ev.Book().Conjured() {
		t.Errorf("events = %v", got.Events())
	}
	got.Events()[0] = nil
	if got.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	if _, held, _ := character.ReadFeatureState(c.FeatureStates(), character.WarlockPactStateKey()); held || !slices.Equal(cmd.Prepared, []content.SpellID{"bless"}) {
		t.Error("the command was altered")
	}
	cmd.Prepared = []content.SpellID{"druidcraft"}
	res, _ = uc.Run(t.Context(), cmd)
	if rej, ok := res.(character.ConjureBookOfShadowsRejected); !ok || rej.Rejection.Kind != character.WarlockBookSpellRefused || rej.Rejection.BookRule != character.BookSpellAlreadyPrepared || rej.Rejection.Spell != "druidcraft" {
		t.Errorf("already prepared = %#v", res)
	}
	cmd.Spells = []content.SpellID{"bad id", "x"}
	if res, err := uc.Run(t.Context(), cmd); res != nil || !srderrIsInvalid(err) {
		t.Errorf("invalid spell id = %v, %v", res, err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-042, FR-141, FR-142 - every chosen spell the source lacks is listed in ConjureBookOfShadowsContentMissing (here all five, in the order cantrips then spells); a source that fails or answers with another entry is a *srderr.PortError recorded once.
// Path: per case (unhappy: missing content, collaborator failure).
func TestConjureBookOfShadowsContentFailures(t *testing.T) {
	t.Parallel()
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_tome"))
	var want []content.MissingContent
	for _, id := range []content.SpellID{"acid_splash", "dancing_lights", "druidcraft", "detect_magic", "speak_with_animals"} {
		want = append(want, id.Missing())
	}
	warContentFailures(t, character.SpanConjureBookOfShadows, "book_of_shadows", want,
		func(src character.ContentSource, opts ...character.Option) (any, error) {
			return must(character.NewConjureBookOfShadows(src, opts...)).Run(t.Context(), warBookCmd(t, c))
		}, func(res any) ([]content.MissingContent, bool) {
			m, ok := res.(character.ConjureBookOfShadowsContentMissing)
			return m.Missing(), ok
		})
	if !srderrIsInvalid(errOf(character.NewConjureBookOfShadows(nil))) {
		t.Error("a nil source was accepted")
	}
	if _, err := (character.ConjureBookOfShadows{}).Run(t.Context(), warBookCmd(t, c)); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
}
