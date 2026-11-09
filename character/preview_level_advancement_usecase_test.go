package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// previewFixture is a Barbarian 1 with 300 XP and a source holding Barbarian and
// Druid (Strength and Wisdom as Primary Abilities).
type previewFixture struct {
	char   character.Character
	source *contenttest.FakeSource
}

func newPreviewFixture(t *testing.T, str, wis int) previewFixture {
	t.Helper()
	return previewFixture{
		char:   restored(t, 300, 40, scoresOf(t, str, 10, 10, 10, wis, 10), cl("barbarian", 1)),
		source: contenttest.NewFakeSource(advBarbarian(t), advDruid(t)),
	}
}

func (f previewFixture) useCase(t *testing.T, opts ...character.Option) character.PreviewLevelAdvancement {
	t.Helper()
	uc, err := character.NewPreviewLevelAdvancement(append([]character.Option{character.WithSource(f.source)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func (f previewFixture) cmd(class string) character.PreviewLevelAdvancementCommand {
	return character.PreviewLevelAdvancementCommand{Character: f.char, Class: content.ClassID(class)}
}

func recordedPreview(t *testing.T, f previewFixture) (character.PreviewLevelAdvancement, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return f.useCase(t, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)), rec
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-009, FR-010, FR-021,
// FR-022 - the use case previews a held class (one lookup) and a new class (the chosen
// class, then every current class) and returns the previewer's gain.
// Path: happy.
func TestPreviewLevelAdvancementAvailable(t *testing.T) {
	t.Parallel()
	f := newPreviewFixture(t, 13, 13)
	res, err := f.useCase(t).Run(context.Background(), f.cmd("barbarian"))
	g, ok := res.(character.PreviewLevelAdvancementAvailable)
	if err != nil || !ok || g.Gain.NewClassLevel() != 2 || g.Gain.NewTotalLevel().Value() != 2 {
		t.Fatalf("held class: %#v, %v", res, err)
	}
	if keys := lookupKeys(f.source); len(keys) != 1 || keys[0] != content.ClassID("barbarian").Key() {
		t.Errorf("FR-021: a held class looks up only itself, got %v", keys)
	}
	f2 := newPreviewFixture(t, 13, 13)
	res, err = f2.useCase(t).Run(context.Background(), f2.cmd("druid"))
	n, ok := res.(character.PreviewLevelAdvancementAvailable)
	if _, isNew := n.Gain.ClassStatus().(character.AdvancementClassStatusNew); err != nil || !ok || !isNew || n.Gain.NewClassLevel() != 1 {
		t.Fatalf("new class: %#v, %v", res, err)
	}
	keys := lookupKeys(f2.source)
	if len(keys) != 2 || keys[0] != content.ClassID("druid").Key() || keys[1] != content.ClassID("barbarian").Key() {
		t.Errorf("FR-021: a new class looks up the chosen class then the current ones, got %v", keys)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-011, FR-012 - a level
// a rule refuses is an Unavailable result holding LevelUp's rejection.
// Path: per case.
func TestPreviewLevelAdvancementUnavailable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		prep func(*previewFixture)
		cmd  string
		want string
	}{
		{"not enough XP", func(f *previewFixture) { f.char = restored(t, 299, 40, flatScores(t, 13), cl("barbarian", 1)) }, "barbarian", "xp"},
		{"prerequisites", func(f *previewFixture) {}, "druid", "prereq"},
		{"maximum level", func(f *previewFixture) { f.char = restored(t, 400000, 40, flatScores(t, 13), cl("barbarian", 20)) }, "barbarian", "max"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := newPreviewFixture(t, 12, 12)
			c.prep(&f)
			res, err := f.useCase(t).Run(context.Background(), f.cmd(c.cmd))
			u, ok := res.(character.PreviewLevelAdvancementUnavailable)
			if err != nil || !ok || levelUpRejectionTag(u.Block.Rejection()) != c.want {
				t.Errorf("= %#v, %v, want %s", res, err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-020, FR-021 - a class
// the source lacks is ContentMissing with no preview; for a new class every missing class
// (chosen and current) is listed once in id order with no prerequisite evaluated, and a
// held class never looks up another.
// Path: per case.
func TestPreviewLevelAdvancementContentMissing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		source  *contenttest.FakeSource
		cmd     string
		want    []string
		lookups int
	}{
		{"empty source, new class", contenttest.NewFakeSource(), "druid", []string{"barbarian", "druid"}, 2},
		{"chosen class only", contenttest.NewFakeSource(advBarbarian(t)), "druid", []string{"druid"}, 2},
		{"current class only", contenttest.NewFakeSource(advDruid(t)), "druid", []string{"barbarian"}, 2},
		{"held class missing", contenttest.NewFakeSource(advDruid(t)), "barbarian", []string{"barbarian"}, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := newPreviewFixture(t, 10, 10) // prerequisites unmet: they must not be evaluated
			f.source = c.source
			res, err := f.useCase(t).Run(context.Background(), f.cmd(c.cmd))
			m, ok := res.(character.PreviewLevelAdvancementContentMissing)
			if err != nil || !ok || len(m.Missing()) != len(c.want) {
				t.Fatalf("= %#v, %v", res, err)
			}
			for i, w := range c.want {
				if m.Missing()[i].ID().String() != w {
					t.Errorf("missing[%d] = %v, want %s", i, m.Missing()[i], w)
				}
			}
			if n := len(lookupKeys(f.source)); n != c.lookups {
				t.Errorf("%d lookups, want %d", n, c.lookups)
			}
			m.Missing()[0] = content.MissingContent{}
			if m.Missing()[0].ID().String() != c.want[0] {
				t.Error("Missing must return a copy")
			}
		})
	}
	if (character.PreviewLevelAdvancementContentMissing{}).Missing() != nil {
		t.Error("the zero result names nothing")
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-022 - a port failure
// (an error, a panic, a broken contract) comes back as a *srderr.PortError of the
// content port with the cause reachable and no result.
// Path: per case.
func TestPreviewLevelAdvancementPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("down")
	f := newPreviewFixture(t, 13, 13)
	f.source = f.source.FailLookup(cause)
	res, err := f.useCase(t).Run(context.Background(), f.cmd("barbarian"))
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != "content_error" || !errors.Is(err, cause) {
		t.Errorf("res %v err %v", res, err)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-023 - the use case does
// not alter its command, returns an equal result for equal input on every call, and
// invalid commands, the zero use case and a cancelled context are errors before any
// signal.
// Path: per case.
func TestPreviewLevelAdvancementInvalidCommands(t *testing.T) {
	t.Parallel()
	f := newPreviewFixture(t, 13, 13)
	uc, rec := recordedPreview(t, f)
	before := f.char
	first, err := uc.Run(context.Background(), f.cmd("druid"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := uc.Run(context.Background(), f.cmd("druid"))
	if err != nil || !reflect.DeepEqual(first, second) || !f.char.Equal(before) {
		t.Error("FR-023: equal inputs must give equal results and leave the character unchanged")
	}
	rec.Meter.Clear()
	for name, cmd := range map[string]character.PreviewLevelAdvancementCommand{
		"zero character": {Class: "druid"}, "invalid class": {Character: f.char},
	} {
		if _, err := uc.Run(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, f.cmd("druid")); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("%d measurements before validation", n)
	}
	if _, err := (character.PreviewLevelAdvancement{}).Run(context.Background(), f.cmd("druid")); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := character.NewPreviewLevelAdvancement(); !isInvalid(err) {
		t.Errorf("missing source: %v", err)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-098, FR-099, FR-101 -
// an available preview is one preview span with a load_content child, one operations
// increment (outcome success, result available) and one duration with the operation
// attribute only; an unavailable preview is a failure without error status carrying the
// rejection; nothing is logged; Noop and a recorder give the same result.
// Path: per case.
func TestPreviewLevelAdvancementSignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		str   int
		cmd   string
		attrs map[string]any
	}{
		{"available", 13, "barbarian", map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "available"}},
		{"unavailable", 12, "druid", map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "unavailable", character.AttrRejection: "multiclass_prerequisites_not_met"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := newPreviewFixture(t, c.str, 13)
			uc, rec := recordedPreview(t, f)
			got, err := uc.Run(context.Background(), f.cmd(c.cmd))
			plain, perr := f.useCase(t).Run(context.Background(), f.cmd(c.cmd))
			if err != nil || perr != nil || !reflect.DeepEqual(got, plain) {
				t.Error("FR-098: Noop and a recording bundle must agree")
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Name != character.SpanAdvancementPreview || spans[1].Name != character.SpanAdvancementLoadContent || spans[1].ParentID != spans[0].ID ||
				spans[0].Status != telemetry.StatusUnset || spans[1].Status != telemetry.StatusUnset || !spans[0].Ended || !spans[1].Ended || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "preview span", spans[0].Attrs, c.attrs)
			sameAttrs(t, "load_content span", spans[1].Attrs, outcome("success"))
			ops := onlyOne(t, rec, character.MetricOperations)
			if ops.Value != 1 || ops.Unit != "{operation}" || ops.Kind != telemetrytest.KindCounter {
				t.Errorf("operations = %+v", ops)
			}
			sameAttrs(t, "operations", ops.Attrs, opAttrs("preview", c.attrs))
			d := onlyOne(t, rec, character.MetricDuration)
			if d.Value != 0.25 || d.Unit != "s" {
				t.Errorf("duration = %+v", d)
			}
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "preview"})
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-100, FR-102 - content
// missing is recorded once as a failure with error status, result content_missing and
// no error.type; a port failure is recorded once with the recorded error, error.type
// content_error and no result; neither is logged.
// Path: per case.
func TestPreviewLevelAdvancementFailureSignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		prep   func(*previewFixture)
		attrs  map[string]any
		errors int
	}{
		{"content missing", func(f *previewFixture) { f.source = contenttest.NewFakeSource() }, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing"}, 0},
		{"port failure", func(f *previewFixture) { f.source = f.source.FailLookup(errors.New("down")) }, map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"}, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := newPreviewFixture(t, 13, 13)
			c.prep(&f)
			uc, rec := recordedPreview(t, f)
			_, err := uc.Run(context.Background(), f.cmd("druid"))
			if (err != nil) != (c.errors == 1) {
				t.Fatalf("err = %v", err)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != c.errors || spans[1].Status != telemetry.StatusError {
				t.Errorf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("preview", c.attrs))
			sameAttrs(t, "duration", onlyOne(t, rec, character.MetricDuration).Attrs, map[string]any{character.AttrOperation: "preview"})
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-099 - the signal
// names and attribute values are pinned to the catalogue.
// Path: per case.
func TestLevelAdvancementSignalValues(t *testing.T) {
	t.Parallel()
	for _, v := range []struct{ got, want string }{
		{character.SpanAdvancementPreview, "srd5e.character.advancement.preview"},
		{character.SpanAdvancementChooseSubclass, "srd5e.character.advancement.choose_subclass"},
		{character.SpanAdvancementLoadContent, "srd5e.character.advancement.load_content"},
		{character.ValueOperationPreview, "preview"}, {character.ValueOperationChooseSubclass, "choose_subclass"},
		{character.ValueResultAvailable, "available"}, {character.ValueResultUnavailable, "unavailable"}, {character.ValueResultChosen, "chosen"},
		{character.ValueRejectionNotInClass, "not_in_class"}, {character.ValueRejectionBelowSubclassLevel, "below_subclass_level"},
		{character.ValueRejectionAlreadyChosen, "already_chosen"}, {character.ValueRejectionUnknownSubclass, "unknown_subclass"},
	} {
		if v.got != v.want {
			t.Errorf("%q, want %q", v.got, v.want)
		}
	}
}
