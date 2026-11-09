package character_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// spellSpec describes a synthetic spell: only its id, level and class list matter
// to the spellcasting rules; the rest is a fixed valid body.
type spellSpec struct {
	id      string
	level   int
	classes []string
}

var allSpellClasses = []string{"bard", "cleric", "druid", "paladin", "ranger", "sorcerer", "warlock", "wizard"}

func spellJSON(s spellSpec) string {
	quoted := make([]string, len(s.classes))
	for i, c := range s.classes {
		quoted[i] = `"` + c + `"`
	}
	return fmt.Sprintf(`{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":[%s],"castingTimes":[{"kind":"magicAction"}],"ritual":false,"range":{"kind":"self"},"components":{"verbal":true,"somatic":false},
"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}`, s.id, s.level, strings.Join(quoted, ","))
}

// spellMap decodes the specs into the map the spellcasting functions take.
func spellMap(t *testing.T, specs ...spellSpec) map[content.SpellID]content.Spell {
	t.Helper()
	docs := make([]string, len(specs))
	for i, s := range specs {
		docs[i] = spellJSON(s)
	}
	set, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":1,"spells":[` + strings.Join(docs, ",") + `]}`))
	if err != nil {
		t.Fatalf("fixture spells: %v", err)
	}
	out := map[content.SpellID]content.Spell{}
	for _, e := range set.Entries() {
		sp := e.(content.Spell)
		out[sp.ID()] = sp
	}
	return out
}

// spellID is a valid synthetic spell id for the given level and index.
func spellID(level, i int) content.SpellID {
	return content.MustSpellID(fmt.Sprintf("s%d_%02d", level, i))
}

// spellIDs returns the synthetic ids of level, indexes 1..n.
func spellIDs(level, n int) []content.SpellID {
	out := make([]content.SpellID, n)
	for i := range out {
		out[i] = spellID(level, i+1)
	}
	return out
}

// spellCatalogue is 12 spells of every level 0 to 9 on every class list.
func spellCatalogue(t *testing.T) map[content.SpellID]content.Spell {
	t.Helper()
	var specs []spellSpec
	for lvl := 0; lvl <= 9; lvl++ {
		for i := 1; i <= 12; i++ {
			specs = append(specs, spellSpec{id: string(spellID(lvl, i)), level: lvl, classes: allSpellClasses})
		}
	}
	return spellMap(t, specs...)
}

// with returns the spellCatalogue plus extra spells.
func withSpells(t *testing.T, base map[content.SpellID]content.Spell, specs ...spellSpec) map[content.SpellID]content.Spell {
	t.Helper()
	for id, s := range spellMap(t, specs...) {
		base[id] = s
	}
	return base
}

func classOf(s string) content.ClassID { return content.MustClassID(s) }

// classLevelsOf builds ClassLevels from alternating class names and levels.
func classLevelsOf(t *testing.T, pairs ...any) character.ClassLevels {
	t.Helper()
	var es []character.ClassLevel
	for i := 0; i < len(pairs); i += 2 {
		es = append(es, character.ClassLevel{Class: classOf(pairs[i].(string)), Level: pairs[i+1].(int)})
	}
	l, err := character.NewClassLevels(es)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

var caster = character.MustIdentifier("caster")

// castingState builds a SpellcastingState through the public use case.
func castingState(t *testing.T, levels character.ClassLevels, spells map[content.SpellID]content.Spell, setups ...character.ClassSpellcastingSetup) character.SpellcastingState {
	t.Helper()
	res, err := character.NewCreateSpellcasting().Do(t.Context(), character.CreateSpellcastingCommand{Character: caster, Levels: levels, Classes: setups, Spells: spells})
	if err != nil {
		t.Fatal(err)
	}
	c, ok := res.(character.CreateSpellcastingCreated)
	if !ok {
		t.Fatalf("create = %#v", res)
	}
	return c.State()
}

// recordedCasting returns options that record into a fresh recorder with a clock stepping 250 ms.
func recordedCasting() (*telemetrytest.Recorder, []character.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return rec, []character.Option{character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)}
}

// wantSignals is what one successful or rejected spellcasting call must emit: one span
// (with the status), the counter increment (nil counter attrs: none expected) and one
// duration carrying only the operation.
type wantSignals struct {
	span, operation, counter string
	spanAttrs, counterAttrs  map[string]any
	status                   telemetry.Status
}

func checkSignals(t *testing.T, rec *telemetrytest.Recorder, w wantSignals) {
	t.Helper()
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Name != w.span || !spans[0].Ended || spans[0].Status != w.status {
		t.Fatalf("spans = %+v, want one %s with status %v", spans, w.span, w.status)
	}
	sameAttrs(t, "span", spans[0].Attrs, w.spanAttrs)
	if w.counter != "" {
		sameAttrs(t, "counter", onlyOne(t, rec, w.counter).Attrs, w.counterAttrs)
	}
	d := onlyOne(t, rec, character.MetricDuration)
	if d.Value != 0.25 {
		t.Errorf("duration = %v, want 0.25", d.Value)
	}
	sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: w.operation})
	noLogsOrViolations(t, rec)
}

// attrs merges attribute maps.
func attrs(maps ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func res(result, outcomeValue string) map[string]any {
	return map[string]any{telemetry.AttrOutcome: outcomeValue, character.AttrResult: result}
}
