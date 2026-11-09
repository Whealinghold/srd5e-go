package character_test

import (
	"context"
	"io/fs"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// pldBundled loads the bundled Paladin ClassDefinition once: decoding the whole bundle per test made the suite slow, and the definition is immutable.
var pldBundled = sync.OnceValues(func() (content.ClassDefinition, error) {
	src, err := data.NewBundledSource()
	if err != nil {
		return content.ClassDefinition{}, err
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("paladin")))
	if err != nil {
		return content.ClassDefinition{}, err
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition), nil
})

// pldDef is the bundled Paladin ClassDefinition (content/data/class_paladin.json).
func pldDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	def, err := pldBundled()
	if err != nil {
		tb.Fatal(err)
	}
	return def
}

// pldOath is the bundled Oath of Devotion subclass.
func pldOath(tb testing.TB) *content.SubclassDefinition {
	tb.Helper()
	sub := pldDef(tb).Subclasses()[0]
	return &sub
}

// pldProgression is the progression of the bundled Paladin and Oath of Devotion.
func pldProgression(tb testing.TB) character.PaladinProgression {
	tb.Helper()
	return must(character.NewPaladinProgression(pldDef(tb), pldOath(tb)))
}

// pldDefFrom decodes the bundled Paladin file after mutate changed its text.
func pldDefFrom(tb testing.TB, mutate func(string) string) content.ClassDefinition {
	tb.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_paladin.json")
	if err != nil {
		tb.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode([]byte(mutate(string(raw))))
	if err != nil {
		tb.Fatalf("mutated Paladin file: %v", err)
	}
	return set.Entries()[0].(content.ClassDefinition)
}

// pldPaladin is a Paladin of level n and nothing else, with flat scores of 10.
func pldPaladin(t *testing.T, n int) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), cl("paladin", n))
}

// pldWith returns c holding the PaladinState.
func pldWith(t *testing.T, c character.Character, s character.PaladinState) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.PaladinStateKey(), s))
}

// pldStateOf reads the PaladinState c stores.
func pldStateOf(t *testing.T, c character.Character) character.PaladinState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.PaladinStateKey())
	if err != nil || !ok {
		t.Fatalf("PaladinState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// pldState builds the state of a Paladin of level n (channel maximum from the bundled table) from parts.
func pldState(t *testing.T, n int, parts character.PaladinStateParts) character.PaladinState {
	t.Helper()
	return must(character.RestorePaladinState(character.MustPaladinLevel(n), pldProgression(t).ChannelDivinityMaximum(character.MustPaladinLevel(n)), parts))
}

// pldFull is the full state of a Paladin of level n.
func pldFull(t *testing.T, n int) character.PaladinState {
	t.Helper()
	return character.InitialPaladinState(character.MustPaladinLevel(n), pldProgression(t).ChannelDivinityMaximum(character.MustPaladinLevel(n)))
}

// pldReasons turns reasons into their String values.
func pldReasons(rs []character.PaladinReason) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}

// pldForeign is c holding a foreign state under the Paladin key, so reading its PaladinState fails.
func pldForeign(t *testing.T, c character.Character) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.MustFeatureStateKey[character.ChosenSubclass]("paladin_state"), must(character.NewChosenSubclass("oath_of_devotion"))))
}

// pldWant is the signals of one Paladin call: a span of the name for the feature, success or the rejection.
func pldWant(span, feature, rejection string) wantCall {
	w := wantCall{span: span, feature: feature, class: character.ValueClassPaladin, status: telemetry.StatusUnset, outcome: "success", logged: true}
	if rejection != "" {
		w.outcome, w.rejection = "failure", rejection
	}
	return w
}
