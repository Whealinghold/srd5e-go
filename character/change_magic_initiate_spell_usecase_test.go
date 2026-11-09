package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func changeCmd(t *testing.T, newSpell string) character.ChangeMagicInitiateSpellCommand {
	t.Helper()
	c := charAt(t, 5)
	return character.ChangeMagicInitiateSpellCommand{Character: c, Feats: heldFeats(t, initiateFeat(), clericInitiate()), List: "cleric", Old: "guidance", New: content.SpellID(newSpell), LevelGained: gainedBy(t, c)}
}

func recordedChange(t *testing.T, src character.ContentSource) (character.ChangeMagicInitiateSpell, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return must(character.NewChangeMagicInitiateSpell(src, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now))), rec
}

// Requirement: doc/specs/feats-behaviour.md FR-070, FR-073, FR-074, Table U "spell change use case" - the replacement is found, missing or domain-rejected, each a result and never a panic; only the new spell is looked up; the no-op and a recording bundle agree; one take span with a load_content child, one counter increment and the right attributes.
// Path: per case.
func TestChangeMagicInitiateSpell(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		src     func() *contenttest.FakeSource
		newName string
		outcome string
		result  string
		reject  string
		status  telemetry.Status
	}{
		{"found", func() *contenttest.FakeSource { return featSource(t) }, "mending", "success", "changed", "", telemetry.StatusUnset},
		{"missing", func() *contenttest.FakeSource { return contenttest.NewFakeSource() }, "mending", "failure", "content_missing", "", telemetry.StatusError},
		{"rejected", func() *contenttest.FakeSource { return featSource(t) }, "fire_bolt", "failure", "rejected", "feat_choices_invalid", telemetry.StatusUnset},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := changeCmd(t, c.newName)
			snapshot := cmd.Feats
			src := c.src()
			uc, rec := recordedChange(t, src)
			res, err := uc.Run(context.Background(), cmd)
			plain, err2 := must(character.NewChangeMagicInitiateSpell(c.src())).Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("= %#v, %v (plain %#v)", res, err, plain)
			}
			if !cmd.Feats.Equal(snapshot) {
				t.Error("the command was altered")
			}
			switch r := res.(type) {
			case character.ChangeMagicInitiateSpellSucceeded:
				if c.result != "changed" || r.Event().Old() != "guidance" || r.Event().New() != "mending" || !r.Feats().All()[0].Choices().(character.MagicInitiateChoices).SpellChangedAt(cmd.Character.TotalLevel()) {
					t.Errorf("= %#v", r)
				}
			case character.ChangeMagicInitiateSpellRejected:
				if c.result != "rejected" || r.Rejection.String() != c.reject {
					t.Errorf("= %#v", r)
				}
			case character.ChangeMagicInitiateSpellContentMissing:
				if c.result != "content_missing" || !slices.Equal(r.Missing(), []content.MissingContent{content.SpellID("mending").Missing()}) {
					t.Errorf("= %#v", r)
				}
				r.Missing()[0] = content.MissingContent{}
				if r.Missing()[0] == (content.MissingContent{}) {
					t.Error("Missing must return a copy")
				}
			}
			if keys := lookupKeys(src); len(keys) != 1 || keys[0] != content.SpellID(c.newName).Key() {
				t.Errorf("lookups = %v", keys)
			}
			attrs := map[string]any{telemetry.AttrOutcome: c.outcome, character.AttrResult: c.result}
			if c.reject != "" {
				attrs[character.AttrRejection] = c.reject
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Name != character.SpanChangeMagicInitiateSpell || spans[1].Name != character.SpanLoadContent || spans[0].Status != c.status {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, attrs)
			counter := map[string]any{character.AttrOperation: "change_magic_initiate_spell"}
			for k, v := range attrs {
				counter[k] = v
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, counter)
			sameAttrs(t, "duration", onlyOne(t, rec, character.MetricDuration).Attrs, map[string]any{character.AttrOperation: "change_magic_initiate_spell"})
			noLogsOrViolations(t, rec)
		})
	}
	if (character.ChangeMagicInitiateSpellContentMissing{}).Missing() != nil {
		t.Error("the zero result names nothing")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-068, FR-074 - a content port failure and a spell key answered with another entry type are a *srderr.PortError recorded once; invalid commands, the zero use case and a cancelled context are refused before any signal.
// Path: unhappy (collaborator failure).
func TestChangeMagicInitiateSpellFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("down")
	for name, src := range map[string]character.ContentSource{"fails": contenttest.NewFakeSource().FailLookup(cause), "another type": answering{entry: alertFeat()}} {
		uc, rec := recordedChange(t, src)
		res, err := uc.Run(context.Background(), changeCmd(t, "mending"))
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != "content_error" || name == "fails" && !errors.Is(err, cause) {
			t.Fatalf("%s: %v, %v", name, res, err)
		}
		if spans := rec.Tracer.Spans(); spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
			t.Errorf("%s: span %+v", name, spans[0])
		}
		sameAttrs(t, name+" operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: "change_magic_initiate_spell", telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"})
		noLogsOrViolations(t, rec)
	}
	rec := telemetrytest.NewRecorder()
	uc := must(character.NewChangeMagicInitiateSpell(featSource(t), character.WithTelemetry(rec.Telemetry())))
	good := changeCmd(t, "mending")
	for name, mutate := range map[string]func(*character.ChangeMagicInitiateSpellCommand){
		"zero character": func(c *character.ChangeMagicInitiateSpellCommand) { c.Character = character.Character{} },
		"invalid list":   func(c *character.ChangeMagicInitiateSpellCommand) { c.List = "" },
		"invalid old":    func(c *character.ChangeMagicInitiateSpellCommand) { c.Old = "" },
		"invalid new":    func(c *character.ChangeMagicInitiateSpellCommand) { c.New = "" },
	} {
		cmd := good
		mutate(&cmd)
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.ChangeMagicInitiateSpell{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	if _, err := character.NewChangeMagicInitiateSpell(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil source: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, good); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a refused call emitted a signal")
	}
}
