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

const hunter = content.SubclassID("hunter")

// rgrBundled loads the bundled Ranger ClassDefinition once: decoding the whole bundle per test made the suite slow, and the definition is immutable.
var rgrBundled = sync.OnceValues(func() (content.ClassDefinition, error) {
	src, err := data.NewBundledSource()
	if err != nil {
		return content.ClassDefinition{}, err
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("ranger")))
	if err != nil {
		return content.ClassDefinition{}, err
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition), nil
})

// rgrDef is the bundled Ranger ClassDefinition (content/data/class_ranger.json).
func rgrDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	def, err := rgrBundled()
	if err != nil {
		tb.Fatal(err)
	}
	return def
}

// rgrHunter is the bundled Hunter subclass.
func rgrHunter(tb testing.TB) *content.SubclassDefinition {
	tb.Helper()
	sub := rgrDef(tb).Subclasses()[0]
	return &sub
}

// rgrProgression is the progression of the bundled Ranger and Hunter.
func rgrProgression(tb testing.TB) character.RangerProgression {
	tb.Helper()
	return must(character.NewRangerProgression(rgrDef(tb), rgrHunter(tb)))
}

// rgrDefFrom decodes the bundled Ranger file after mutate changed its text.
func rgrDefFrom(tb testing.TB, mutate func(string) string) content.ClassDefinition {
	tb.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_ranger.json")
	if err != nil {
		tb.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode([]byte(mutate(string(raw))))
	if err != nil {
		tb.Fatalf("mutated Ranger file: %v", err)
	}
	return set.Entries()[0].(content.ClassDefinition)
}

// rgrRanger is a Ranger of level n and nothing else, with flat scores of 10.
func rgrRanger(t *testing.T, n int) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), cl("ranger", n))
}

// rgrWith returns c holding the RangerState.
func rgrWith(t *testing.T, c character.Character, s character.RangerState) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.RangerStateKey(), s))
}

// rgrStateOf reads the RangerState c stores.
func rgrStateOf(t *testing.T, c character.Character) character.RangerState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.RangerStateKey())
	if err != nil || !ok {
		t.Fatalf("RangerState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// rgrState builds a RangerState from parts.
func rgrState(t *testing.T, parts character.RangerStateParts) character.RangerState {
	t.Helper()
	return must(character.RestoreRangerState(parts))
}

// rgrFacts are the facts of a Ranger of level n holding the state and subclass.
func rgrFacts(t *testing.T, n int, s character.RangerState, subclass content.SubclassID) character.RangerFacts {
	t.Helper()
	return character.RangerFacts{Progression: rgrProgression(t), Level: character.MustRangerLevel(n), State: s, Subclass: subclass}
}

// rgrHunterFacts are the facts of a Hunter of level n holding the Hunter’s Prey option and Defensive Tactics option.
func rgrHunterFacts(t *testing.T, n int, prey, tactics string) character.RangerFacts {
	t.Helper()
	return rgrFacts(t, n, rgrState(t, character.RangerStateParts{HuntersPrey: prey, DefensiveTactics: tactics}), hunter)
}

// rgrReasons turns reasons into their String values.
func rgrReasons(rs []character.RangerReason) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}

// rgrForeign is c holding a foreign state under the Ranger key, so reading its RangerState fails.
func rgrForeign(t *testing.T, c character.Character) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.MustFeatureStateKey[character.ChosenSubclass]("ranger_state"), must(character.NewChosenSubclass("hunter"))))
}

// rgrWant is the signals of one Ranger call: a span of the name for the feature, success or the rejection.
func rgrWant(span, feature, rejection string) wantCall {
	w := wantCall{span: span, feature: feature, class: character.ValueClassRanger, status: telemetry.StatusUnset, outcome: "success", logged: true}
	if rejection != "" {
		w.outcome, w.rejection = "failure", rejection
	}
	return w
}

// rgrCantrip is a synthetic cantrip of the id on the class lists.
func rgrCantrip(t *testing.T, id string, classes ...string) content.Spell {
	t.Helper()
	return clcSpell(t, id, 0, classes, "")
}
