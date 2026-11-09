package character_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// evkSpellbookCases drives the two content-loading Evoker use cases in the three modes of wzMode.
func evkSpellbookCases() []wzCase {
	return []wzCase{
		{name: "AddEvocationSavantSpells", span: character.SpanAddEvocationSavantSpells, feature: "evocation_savant", loads: true, hasLevels: true, rejection: "not_evoker",
			okType: "SpellbookChanged", badType: "AddEvocationSavantSpellsRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				subclass := content.SubclassID("evoker")
				if mode == wzBad {
					subclass = ""
				}
				return must(character.NewAddEvocationSavantSpells(src, opts...)).Run(t.Context(), character.AddEvocationSavantSpellsCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 3, 3, 3),
					Subclass: subclass, Spellbook: wzBook("k1"), Spells: []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 2)}})
			}},
		{name: "AddEvokerFreeSpell", span: character.SpanAddEvokerFreeSpell, feature: "evoker_free_spell", loads: true, hasLevels: true, rejection: "no_new_spell_level",
			okType: "SpellbookChanged", badType: "AddEvokerFreeSpellRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return must(character.NewAddEvokerFreeSpell(src, opts...)).Run(t.Context(), character.AddEvokerFreeSpellCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 5, 6, 5),
					Subclass: "evoker", Spellbook: wzBook("k1"), Spell: evkEvoFact("s", 3)})
			}},
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-001 to FR-004, FR-018 - through the use cases an Evoker adds its two Evocation Savant spells at level 3 and one free spell at level 5 (the book grows, one event), and a
// level 3 free addition, a level 2 Evocation Savant, a character without the subclass and an unlisted spell each give their rejection.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestEvokerSpellbookUseCases(t *testing.T) {
	t.Parallel()
	src := wzSource(t)
	savant := must(character.NewAddEvocationSavantSpells(src))
	free := must(character.NewAddEvokerFreeSpell(src))
	two := []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 2)}
	res, err := savant.Run(t.Context(), character.AddEvocationSavantSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(3), Subclass: "evoker", Spellbook: wzBook("k1"), Spells: two})
	if got, ok := res.(character.SpellbookChanged); err != nil || !ok || len(got.Spellbook().Spells()) != 3 || len(got.Events()) != 1 {
		t.Fatalf("level 3 Evocation Savant = %#v, %v", res, err)
	}
	for _, c := range []struct {
		name  string
		level int
		sub   content.SubclassID
		want  character.EvokerSpellbookRejectionKind
	}{
		{"level 2", 2, "evoker", character.EvokerSpellbookFeatureNotAvailable},
		{"no subclass", 3, "", character.EvokerSpellbookNotEvoker},
	} {
		res, _ = savant.Run(t.Context(), character.AddEvocationSavantSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Subclass: c.sub, Spellbook: wzBook("k1"), Spells: two})
		if r, ok := res.(character.AddEvocationSavantSpellsRejected); !ok || r.Rejection.Kind != c.want {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
	for _, c := range []struct {
		name  string
		level int
		spell character.WizardSpellFacts
		want  character.EvokerSpellbookRejectionKind
	}{
		{"FA-02", 5, evkEvoFact("s", 3), 0},
		{"FA-01", 3, evkEvoFact("s", 2), character.EvokerSpellbookNoNewSpellLevel},
		{"FA-03", 5, evkEvoFact("s", 4), character.EvokerSpellbookAboveHighestSpellLevel},
		{"FA-04", 6, evkEvoFact("s", 3), character.EvokerSpellbookNoNewSpellLevel},
		{"FA-05", 17, evkEvoFact("s", 9), 0},
		{"level 2", 2, evkEvoFact("s", 1), character.EvokerSpellbookFeatureNotAvailable},
	} {
		freeRes, err := free.Run(t.Context(), character.AddEvokerFreeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Subclass: "evoker", Spellbook: wzBook("k1"), Spell: c.spell})
		if c.want == 0 {
			if got, ok := freeRes.(character.SpellbookChanged); err != nil || !ok || !got.Spellbook().Contains("s") {
				t.Errorf("%s = %#v, %v", c.name, freeRes, err)
			}
			continue
		}
		if r, ok := freeRes.(character.AddEvokerFreeSpellRejected); !ok || r.Rejection.Kind != c.want {
			t.Errorf("%s = %#v", c.name, freeRes)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-018 - one span, counter and duration per call with the shared signals: an accepted result is success with one FINE log, a rule refusal is failure with the snake_case rejection,
// NotAWizard is decided before any lookup, and an unlisted subclass id gives <UseCase>SubclassUnknown carrying the id with outcome failure, rejection subclass_unknown and no error status.
// Path: per case (happy / unhappy: state error, missing content).
func TestEvokerSpellbookSignals(t *testing.T) {
	t.Parallel()
	for _, c := range evkSpellbookCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			src := wzSource(t)
			for _, m := range []struct {
				mode              wzMode
				result, rejection string
				loaded, logged    bool
			}{
				{wzOK, c.okType, "", true, true},
				{wzBad, c.badType, c.rejection, true, true},
				{wzNotWizard, c.name + "NotAWizard", "not_a_wizard", false, true},
			} {
				rec, opts := wzTelemetry()
				res, err := c.run(t, m.mode, src, opts...)
				if err != nil || typeName(res) != m.result {
					t.Fatalf("mode %d = %T, %v; want %s", m.mode, res, err, m.result)
				}
				want := wzWant{span: c.span, feature: c.feature, outcome: "success", status: telemetry.StatusUnset, logged: m.logged, loaded: m.loaded}
				if m.rejection != "" {
					want.outcome, want.rejection = "failure", m.rejection
				}
				wzCheck(t, rec, want)
			}
		})
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-018 - an unlisted subclass id gives <UseCase>SubclassUnknown carrying that id (not a rejection and not an error): outcome failure, rejection subclass_unknown, no error status,
// no errors increment; the wizard class absent gives <UseCase>ContentMissing; a failing source is propagated as a content *srderr.PortError, recorded once.
// Path: per case (unhappy: missing content, collaborator failure).
func TestEvokerSpellbookContentOutcomes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	rec, opts := wzTelemetry()
	savant, err := must(character.NewAddEvocationSavantSpells(wzSource(t), opts...)).Run(ctx, character.AddEvocationSavantSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(3), Subclass: "bogus",
		Spellbook: wzBook(), Spells: []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 2)}})
	if u, ok := savant.(character.AddEvocationSavantSpellsSubclassUnknown); err != nil || !ok || u.Subclass() != "bogus" {
		t.Fatalf("Evocation Savant with an unlisted subclass = %#v, %v", savant, err)
	}
	wzCheck(t, rec, wzWant{span: character.SpanAddEvocationSavantSpells, feature: "evocation_savant", outcome: "failure", rejection: "subclass_unknown", status: telemetry.StatusUnset, logged: true, loaded: true})
	rec, opts = wzTelemetry()
	freeRes, err := must(character.NewAddEvokerFreeSpell(wzSource(t), opts...)).Run(ctx, character.AddEvokerFreeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(5), Subclass: "bogus", Spell: evkEvoFact("s", 3)})
	if u, ok := freeRes.(character.AddEvokerFreeSpellSubclassUnknown); err != nil || !ok || u.Subclass() != "bogus" {
		t.Fatalf("free spell with an unlisted subclass = %#v, %v", freeRes, err)
	}
	wzCheck(t, rec, wzWant{span: character.SpanAddEvokerFreeSpell, feature: "evoker_free_spell", outcome: "failure", rejection: "subclass_unknown", status: telemetry.StatusUnset, logged: true, loaded: true})

	for _, c := range evkSpellbookCases() {
		rec, opts = wzTelemetry()
		res, err := c.run(t, wzOK, wzMissing(), opts...)
		if err != nil || typeName(res) != c.name+"ContentMissing" {
			t.Fatalf("%s missing = %T, %v", c.name, res, err)
		}
		missing := reflect.ValueOf(res).MethodByName("Missing").Call(nil)[0].Interface().([]content.MissingContent)
		if len(missing) != 1 || missing[0].Kind() != content.KindCharacterClass || missing[0].ID().String() != "wizard" {
			t.Errorf("%s Missing = %v", c.name, missing)
		}
		wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "content_missing", status: telemetry.StatusError, loaded: true})
		for name, src := range map[string]character.ContentSource{"an error": wzMissingFailing(errBoom), "wrong entry": wzWrongEntry(t)} {
			rec, opts = wzTelemetry()
			res, err = c.run(t, wzOK, src, opts...)
			var pe *srderr.PortError
			if res != nil || !asPortError(err, &pe) || pe.Port != content.Port {
				t.Fatalf("%s %s = %v, %v; want a content PortError", c.name, name, res, err)
			}
			wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err), loaded: true})
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-018, class-wizard.md FR-091, FR-096 - the Evoker spellbook use cases give identical results with the no-op and a recording telemetry and leave the input book unchanged;
// a nil source, the zero use case, a zero CharacterID, spell facts without a valid id and a cancelled context are errors before any signal.
// Path: per case (unhappy: determinism, invalid input).
func TestEvokerSpellbookInvalidCommandsAndDeterminism(t *testing.T) {
	t.Parallel()
	for _, c := range evkSpellbookCases() {
		for _, mode := range []wzMode{wzOK, wzBad, wzNotWizard} {
			plain, err1 := c.run(t, mode, wzSource(t))
			_, opts := wzTelemetry()
			recorded, err2 := c.run(t, mode, wzSource(t), opts...)
			if err1 != nil || err2 != nil || !reflect.DeepEqual(plain, recorded) {
				t.Errorf("%s mode %d: %#v (%v) versus %#v (%v)", c.name, mode, plain, err1, recorded, err2)
			}
		}
	}
	src := wzSource(t)
	rec, opts := wzTelemetry()
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	for name, err := range map[string]error{
		"nil source (Savant)": errOf(character.NewAddEvocationSavantSpells(nil)),
		"nil source (free)":   errOf(character.NewAddEvokerFreeSpell(nil)),
		"zero Savant":         errOf(character.AddEvocationSavantSpells{}.Run(t.Context(), character.AddEvocationSavantSpellsCommand{CharacterID: wzID})),
		"zero free":           errOf(character.AddEvokerFreeSpell{}.Run(t.Context(), character.AddEvokerFreeSpellCommand{CharacterID: wzID})),
		"zero id (Savant)":    errOf(must(character.NewAddEvocationSavantSpells(src, opts...)).Run(t.Context(), character.AddEvocationSavantSpellsCommand{ClassLevels: wzLevels(3)})),
		"zero id (free)":      errOf(must(character.NewAddEvokerFreeSpell(src, opts...)).Run(t.Context(), character.AddEvokerFreeSpellCommand{ClassLevels: wzLevels(5)})),
		"no spell id (Savant)": errOf(must(character.NewAddEvocationSavantSpells(src, opts...)).Run(t.Context(), character.AddEvocationSavantSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(3),
			Spells: []character.WizardSpellFacts{{}}})),
		"no spell id (free)": errOf(must(character.NewAddEvokerFreeSpell(src, opts...)).Run(t.Context(), character.AddEvokerFreeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(5)})),
	} {
		if !srderrIsInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := must(character.NewAddEvokerFreeSpell(src, opts...)).Run(cancelled, character.AddEvokerFreeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(5), Spell: evkEvoFact("s", 3)}); err == nil {
		t.Error("a cancelled context was accepted (free)")
	}
	if _, err := must(character.NewAddEvocationSavantSpells(src, opts...)).Run(cancelled, character.AddEvocationSavantSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(3)}); err == nil {
		t.Error("a cancelled context was accepted (Savant)")
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("an invalid command left signals: %+v", rec.Tracer.Spans())
	}
}

// fighterAsWizardSource answers every lookup with the bundled Fighter, a class with no Spell Slot rows.
type fighterAsWizardSource struct{ bundled character.ContentSource }

func (s fighterAsWizardSource) Lookup(ctx context.Context, _ content.Key) (content.LookupResult, error) {
	return s.bundled.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("fighter")))
}

// Requirement: doc/specs/class-wizard-evoker.md FR-003, FR-018 - a "wizard" class definition without Spell Slot rows cannot give the highest Spell Level, so AddEvokerFreeSpell returns an error matching ErrInvalidInput, recorded
// on the span only, and no result.
// Path: unhappy (invalid input, bad data).
func TestAddEvokerFreeSpellWithAClassWithoutSlots(t *testing.T) {
	t.Parallel()
	rec, opts := wzTelemetry()
	res, err := must(character.NewAddEvokerFreeSpell(fighterAsWizardSource{wzSource(t)}, opts...)).Run(t.Context(), character.AddEvokerFreeSpellCommand{CharacterID: wzID, ClassLevels: wzLevels(5), Subclass: "", Spell: evkEvoFact("s", 3)})
	if res != nil || !srderrIsInvalid(err) {
		t.Fatalf("= %v, %v; want an input error", res, err)
	}
	if spans := rec.Tracer.SpansNamed(character.SpanAddEvokerFreeSpell); len(spans) != 1 || spans[0].Status != telemetry.StatusError {
		t.Errorf("spans = %+v, want one with error status", spans)
	}
}
