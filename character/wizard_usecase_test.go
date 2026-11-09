package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

type wzMode int

const (
	wzOK wzMode = iota
	wzBad
	wzNotWizard
)

// wzCase drives one Wizard use case in the three modes (an accepted command, one a rule refuses, a non-Wizard character).
type wzCase struct {
	name, span, feature string
	loads               bool
	hasLevels           bool   // the command carries ClassLevels, so NotAWizard applies
	rejection           string // the rejection value of the wzBad command
	run                 func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error)
	okType, badType     string
}

var wzFighterLevels = levels(cl("fighter", 5))

func wzPick(mode wzMode, ok, bad int, wizard int) character.ClassLevels {
	if mode == wzNotWizard {
		return wzFighterLevels
	}
	if mode == wzBad {
		return wzLevels(bad)
	}
	return wzLevels(wizard)
}

func wzMasteryHeld(t *testing.T) character.WizardResources {
	t.Helper()
	res, err := must(character.NewChooseSpellMastery(wzSource(t))).Run(t.Context(), character.ChooseSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzLevels(18),
		Spellbook: wzBook("one", "two", "one2"), Spells: []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2)}})
	if err != nil {
		t.Fatal(err)
	}
	return res.(character.SpellMasteryApplied).Resources()
}

func wzCases() []wzCase {
	short := func(t *testing.T) character.RestApplied {
		t.Helper()
		return wzRest(t, character.ShortRest, false)
	}
	return []wzCase{
		{name: "ChooseWizardSkills", span: character.SpanChooseWizardSkills, feature: "skills", loads: true, rejection: "wrong_skill_count",
			okType: "WizardSkillsApplied", badType: "ChooseWizardSkillsRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				skills := []content.Skill{content.SkillArcana, content.SkillHistory}
				if mode == wzBad {
					skills = skills[:1]
				}
				return must(character.NewChooseWizardSkills(src, opts...)).Run(t.Context(), character.ChooseWizardSkillsCommand{CharacterID: wzID, Skills: skills})
			}},
		{name: "ChooseWizardEquipment", span: character.SpanChooseWizardEquipment, feature: "equipment", loads: true, rejection: "unknown_equipment_option",
			okType: "WizardEquipmentApplied", badType: "ChooseWizardEquipmentRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				option := "B"
				if mode == wzBad {
					option = "a"
				}
				return must(character.NewChooseWizardEquipment(src, opts...)).Run(t.Context(), character.ChooseWizardEquipmentCommand{CharacterID: wzID, Option: option})
			}},
		{name: "ChooseWizardCantrips", span: character.SpanChooseWizardCantrips, feature: "cantrips", loads: true, hasLevels: true, rejection: "wrong_cantrip_count",
			okType: "WizardCantripsApplied", badType: "ChooseWizardCantripsRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				chosen := wzFacts("c", 0, 3)
				if mode == wzBad {
					chosen = chosen[:2]
				}
				return must(character.NewChooseWizardCantrips(src, opts...)).Run(t.Context(), character.ChooseWizardCantripsCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 1, 1, 1), Chosen: chosen})
			}},
		{name: "ChooseScholarExpertise", span: character.SpanChooseScholar, feature: "scholar", loads: true, hasLevels: true, rejection: "feature_not_available",
			okType: "ScholarApplied", badType: "ChooseScholarExpertiseRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return must(character.NewChooseScholarExpertise(src, opts...)).Run(t.Context(), character.ChooseScholarExpertiseCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 2, 1, 2),
					Proficient: []content.Skill{content.SkillArcana}, Skill: content.SkillArcana})
			}},
		{name: "StartSpellbook", span: character.SpanStartSpellbook, feature: "spellbook_start", loads: true, hasLevels: true, rejection: "wrong_spell_count",
			okType: "SpellbookChanged", badType: "StartSpellbookRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				spells := wzFacts("a", 1, 6)
				if mode == wzBad {
					spells = spells[:5]
				}
				return must(character.NewStartSpellbook(src, opts...)).Run(t.Context(), character.StartSpellbookCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 1, 1, 1), Spells: spells})
			}},
		{name: "AddLevelUpSpells", span: character.SpanAddLevelUpSpells, feature: "spellbook_level_up", loads: true, hasLevels: true, rejection: "feature_not_available",
			okType: "SpellbookChanged", badType: "AddLevelUpSpellsRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return must(character.NewAddLevelUpSpells(src, opts...)).Run(t.Context(), character.AddLevelUpSpellsCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 2, 1, 2),
					Spellbook: wzBook("k1"), Spells: wzFacts("n", 1, 2)})
			}},
		{name: "CopySpellIntoBook", span: character.SpanCopySpellIntoBook, feature: "spellbook_copy_in", loads: true, hasLevels: true, rejection: "above_highest_spell_level",
			okType: "SpellbookChanged", badType: "CopySpellIntoBookRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				level := 3
				if mode == wzBad {
					level = 4
				}
				return must(character.NewCopySpellIntoBook(src, opts...)).Run(t.Context(), character.CopySpellIntoBookCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 5, 5, 5),
					Spellbook: wzBook("k1"), Spell: wzFact("s", level)})
			}},
		{name: "CopySpellbook", span: character.SpanCopySpellbook, feature: "spellbook_copy_book", hasLevels: true, rejection: "not_in_source_spellbook",
			okType: "SpellbookChanged", badType: "CopySpellbookRejected",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				id := "a"
				if mode == wzBad {
					id = "z"
				}
				return character.NewCopySpellbook(opts...).Run(t.Context(), character.CopySpellbookCommand{ClassLevels: wzPick(mode, 1, 1, 1), Source: wzBook("a"), Spells: []character.WizardSpellFacts{wzFact(id, 1)}})
			}},
		{name: "ReplaceLostSpellbook", span: character.SpanReplaceLostSpellbook, feature: "spellbook_replace", hasLevels: true,
			okType: "SpellbookChanged",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return character.NewReplaceLostSpellbook(opts...).Run(t.Context(), character.ReplaceLostSpellbookCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 1, 1, 1), Prepared: wzFacts("p", 1, 2)})
			}},
		{name: "UseArcaneRecovery", span: character.SpanUseArcaneRecovery, feature: "arcane_recovery", loads: true, hasLevels: true, rejection: "exceeds_maximum",
			okType: "ArcaneRecoveryApplied", badType: "UseArcaneRecoveryRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				chosen := map[int]int{1: 2}
				if mode == wzBad {
					chosen = map[int]int{1: 3}
				}
				return must(character.NewUseArcaneRecovery(src, opts...)).Run(t.Context(), character.UseArcaneRecoveryCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 4, 4, 4),
					Expended: map[int]int{1: 3}, Chosen: chosen, Trigger: short(t)})
			}},
		{name: "MemorizeSpell", span: character.SpanMemorizeSpell, feature: "memorize_spell", loads: true, hasLevels: true, rejection: "not_prepared",
			okType: "MemorizeApplied", badType: "MemorizeSpellRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				replace := content.SpellID("a")
				if mode == wzBad {
					replace = "zz"
				}
				return must(character.NewMemorizeSpell(src, opts...)).Run(t.Context(), character.MemorizeSpellCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 5, 5, 5),
					Prepared: wzIDs("a", "b"), Spellbook: wzBook("a", "b", "c"), Replace: replace, New: wzFact("c", 1), Trigger: short(t)})
			}},
		{name: "CheckRitualAdept", span: character.SpanCheckRitualAdept, feature: "ritual_adept", hasLevels: true, rejection: "not_in_spellbook",
			okType: "RitualAdeptEligible", badType: "CheckRitualAdeptRejected",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				id := "r"
				if mode == wzBad {
					id = "zz"
				}
				return character.NewCheckRitualAdept(opts...).Run(t.Context(), character.CheckRitualAdeptCommand{ClassLevels: wzPick(mode, 1, 1, 1), Spellbook: wzBook("r"),
					Spell: wzFact(id, 1, wzRitual), SpellbookAvailable: true})
			}},
		{name: "ChooseSpellMastery", span: character.SpanChooseSpellMastery, feature: "spell_mastery", loads: true, hasLevels: true, rejection: "feature_not_available",
			okType: "SpellMasteryApplied", badType: "ChooseSpellMasteryRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return must(character.NewChooseSpellMastery(src, opts...)).Run(t.Context(), character.ChooseSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 18, 17, 18),
					Spellbook: wzBook("one", "two"), Spells: []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2)}})
			}},
		{name: "ReplaceSpellMastery", span: character.SpanReplaceSpellMastery, feature: "spell_mastery_replace", loads: true, hasLevels: true, rejection: "not_after_long_rest",
			okType: "SpellMasteryReplaceApplied", badType: "ReplaceSpellMasteryRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				trigger := wzRest(t, character.LongRest, false)
				if mode == wzBad {
					trigger = short(t)
				}
				return must(character.NewReplaceSpellMastery(src, opts...)).Run(t.Context(), character.ReplaceSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 18, 18, 18),
					Resources: wzMasteryHeld(t), Spellbook: wzBook("one", "two", "one2"), Replace: "one", New: wzFact("one2", 1), Trigger: trigger})
			}},
		{name: "ChooseSignatureSpells", span: character.SpanChooseSignatureSpells, feature: "signature_spells", loads: true, hasLevels: true, rejection: "feature_not_available",
			okType: "SignatureSpellsApplied", badType: "ChooseSignatureSpellsRejected",
			run: func(t *testing.T, mode wzMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return must(character.NewChooseSignatureSpells(src, opts...)).Run(t.Context(), character.ChooseSignatureSpellsCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 20, 19, 20),
					Spellbook: wzBook("s1", "s2"), Spells: wzSigFacts()})
			}},
	}
}

func wzSigFacts() []character.WizardSpellFacts {
	return []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 3)}
}

func typeName(v any) string { return reflect.TypeOf(v).Name() }

// Requirement: doc/specs/class-wizard.md FR-097, FR-108 - every Wizard use case, in one call, emits one top-level span named per the spec (with the load_content child when it loads), one
// feature_used increment and one duration with srd5e.class wizard and its srd5e.feature; an accepted result is outcome success with one FINE log; a rule refusal is outcome failure with the snake_case
// rejection, no error.type, no error status, no errors increment and one FINE log; NotAWizard is outcome failure with rejection not_a_wizard. Each result variant is named with its use case.
// Path: per case (happy / unhappy: state error).
func TestWizardUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range wzCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			src := wzSource(t)
			rec, opts := wzTelemetry()
			res, err := c.run(t, wzOK, src, opts...)
			if err != nil || typeName(res) != c.okType {
				t.Fatalf("accepted = %T, %v; want %s", res, err, c.okType)
			}
			wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "success", status: telemetry.StatusUnset, logged: true, loaded: c.loads})
			if c.badType != "" {
				rec, opts = wzTelemetry()
				res, err = c.run(t, wzBad, src, opts...)
				if err != nil || typeName(res) != c.badType {
					t.Fatalf("refused = %T, %v; want %s", res, err, c.badType)
				}
				wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "failure", rejection: c.rejection, status: telemetry.StatusUnset, logged: true, loaded: c.loads})
			}
			if c.hasLevels {
				rec, opts = wzTelemetry()
				res, err = c.run(t, wzNotWizard, src, opts...)
				if err != nil || typeName(res) != c.name+"NotAWizard" {
					t.Fatalf("not a Wizard = %T, %v; want %sNotAWizard", res, err, c.name)
				}
				// Decided from ClassLevels before any lookup, so no load_content span.
				wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "not_a_wizard", status: telemetry.StatusUnset, logged: true})
			}
		})
	}
}

// Requirement: doc/specs/class-wizard.md FR-093, FR-094 - when the wizard class is absent every content-loading use case returns its own ContentMissing variant naming the class, with error
// status on the span and no log; a source that returns its error, panics or answers with another entry is a *srderr.PortError for content, recorded once and propagated with no result.
// Path: per case (unhappy: missing content, collaborator failure).
func TestWizardUseCaseContentFailures(t *testing.T) {
	t.Parallel()
	for _, c := range wzCases() {
		if !c.loads {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := wzTelemetry()
			res, err := c.run(t, wzOK, wzMissing(), opts...)
			if err != nil || typeName(res) != c.name+"ContentMissing" {
				t.Fatalf("missing = %T, %v; want %sContentMissing", res, err, c.name)
			}
			missing := reflect.ValueOf(res).MethodByName("Missing").Call(nil)[0].Interface().([]content.MissingContent)
			if len(missing) != 1 || missing[0].Kind() != content.KindCharacterClass || missing[0].ID().String() != "wizard" {
				t.Errorf("Missing = %v", missing)
			}
			wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "content_missing", status: telemetry.StatusError, loaded: true})

			down := errBoom
			for name, src := range map[string]character.ContentSource{"an error": wzMissingFailing(down), "wrong entry": wzWrongEntry(t)} {
				rec, opts = wzTelemetry()
				res, err = c.run(t, wzOK, src, opts...)
				var pe *srderr.PortError
				if res != nil || !asPortError(err, &pe) || pe.Port != content.Port {
					t.Fatalf("%s: = %v, %v; want a PortError for content", name, res, err)
				}
				wzCheck(t, rec, wzWant{span: c.span, feature: c.feature, outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err), loaded: true})
				if loads := rec.Tracer.SpansNamed(character.SpanLoadContent); len(loads) != 1 || loads[0].Status != telemetry.StatusError {
					t.Errorf("%s: load_content = %+v, want error status", name, loads)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-wizard.md FR-096, FR-091 - every Wizard use case gives identical results with the no-op and a recording telemetry, and does not alter its command inputs.
// Path: per case (unhappy: determinism).
func TestWizardUseCasesAreDeterministic(t *testing.T) {
	t.Parallel()
	for _, c := range wzCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, mode := range []wzMode{wzOK, wzBad, wzNotWizard} {
				plain, err1 := c.run(t, mode, wzSource(t))
				_, opts := wzTelemetry()
				recorded, err2 := c.run(t, mode, wzSource(t), opts...)
				if err1 != nil || err2 != nil || !reflect.DeepEqual(plain, recorded) {
					t.Errorf("mode %d: %#v (%v) versus %#v (%v)", mode, plain, err1, recorded, err2)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-wizard.md FR-093 (programmer errors) - the zero use case, a zero CharacterID, a cancelled context and spell facts without a valid id or level are errors matching
// ErrInvalidInput (or ctx.Err()) before any signal; a nil source is rejected by the constructor.
// Path: unhappy (invalid input).
func TestWizardUseCaseInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec, opts := wzTelemetry()
	src := wzSource(t)
	if _, err := (character.ChooseWizardSkills{}).Run(ctx, character.ChooseWizardSkillsCommand{CharacterID: wzID}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := (character.CheckRitualAdept{}).Run(ctx, character.CheckRitualAdeptCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero pure use case: %v", err)
	}
	if _, err := must(character.NewChooseWizardSkills(src, opts...)).Run(ctx, character.ChooseWizardSkillsCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero id: %v", err)
	}
	if _, err := character.NewReplaceLostSpellbook(opts...).Run(ctx, character.ReplaceLostSpellbookCommand{ClassLevels: wzLevels(1)}); !srderrIsInvalid(err) {
		t.Errorf("zero id (pure): %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := must(character.NewStartSpellbook(src, opts...)).Run(cancelled, character.StartSpellbookCommand{CharacterID: wzID, ClassLevels: wzLevels(1)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if _, err := character.NewCopySpellbook(opts...).Run(cancelled, character.CopySpellbookCommand{ClassLevels: wzLevels(1)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled (pure): %v", err)
	}
	for name, f := range map[string]func() error{
		"no id": func() error {
			_, err := must(character.NewStartSpellbook(src, opts...)).Run(ctx, character.StartSpellbookCommand{CharacterID: wzID, ClassLevels: wzLevels(1), Spells: []character.WizardSpellFacts{{}}})
			return err
		},
		"level 10": func() error {
			_, err := character.NewCheckRitualAdept(opts...).Run(ctx, character.CheckRitualAdeptCommand{ClassLevels: wzLevels(1), Spell: wzFact("a", 10)})
			return err
		},
	} {
		if err := f(); !srderrIsInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("an invalid command left signals: %+v", rec.Tracer.Spans())
	}
	nils := []error{
		errOf(character.NewChooseWizardSkills(nil)), errOf(character.NewChooseWizardEquipment(nil)), errOf(character.NewChooseWizardCantrips(nil)), errOf(character.NewChooseScholarExpertise(nil)),
		errOf(character.NewStartSpellbook(nil)), errOf(character.NewAddLevelUpSpells(nil)), errOf(character.NewCopySpellIntoBook(nil)), errOf(character.NewUseArcaneRecovery(nil)),
		errOf(character.NewMemorizeSpell(nil)), errOf(character.NewChooseSpellMastery(nil)), errOf(character.NewReplaceSpellMastery(nil)), errOf(character.NewChooseSignatureSpells(nil)),
	}
	for i, err := range nils {
		if !srderrIsInvalid(err) {
			t.Errorf("constructor %d accepted a nil source: %v", i, err)
		}
	}
}

func errOf[T any](_ T, err error) error { return err }
