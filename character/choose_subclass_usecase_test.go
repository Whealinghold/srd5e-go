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

func subclassFixture(t *testing.T, levels ...character.ClassLevel) (character.Character, *contenttest.FakeSource) {
	t.Helper()
	return levelsChar(t, levels...), contenttest.NewFakeSource(advCleric(t, "life_domain", "light_domain"), advRogue(t))
}

func chooseUseCase(t *testing.T, src character.ContentSource, opts ...character.Option) character.ChooseSubclass {
	t.Helper()
	uc, err := character.NewChooseSubclass(append([]character.Option{character.WithSource(src)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func recordedChoose(t *testing.T, src character.ContentSource) (character.ChooseSubclass, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return chooseUseCase(t, src, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)), rec
}

func chooseCmd(c character.Character, class, subclass string) character.ChooseSubclassCommand {
	return character.ChooseSubclassCommand{Character: c, Class: content.ClassID(class), Subclass: content.SubclassID(subclass)}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-041, FR-048, FR-049,
// FR-050 - the use case looks up only the class, stores the subclass under the class's key
// on a new Character, returns the SubclassChosen event and leaves the command's Character
// unchanged.
// Path: happy.
func TestChooseSubclassSucceeds(t *testing.T) {
	t.Parallel()
	ch, src := subclassFixture(t, cl("cleric", 3), cl("fighter", 4))
	snapshot := ch
	res, err := chooseUseCase(t, src).Run(context.Background(), chooseCmd(ch, "cleric", "life_domain"))
	ok, isOK := res.(character.ChooseSubclassSucceeded)
	if err != nil || !isOK || ok.Event().Subclass() != "life_domain" || ok.Event().ClassLevel() != 3 || ok.Event().CharacterID() != ch.ID() {
		t.Fatalf("= %#v, %v", res, err)
	}
	got, found, _ := character.ReadFeatureState(ok.Character().FeatureStates(), must(character.SubclassStateKey{}.ForClass("cleric")))
	if !found || got.Subclass() != "life_domain" || !ch.Equal(snapshot) {
		t.Error("the subclass must be stored on the new character only")
	}
	if keys := lookupKeys(src); len(keys) != 1 || keys[0] != content.ClassID("cleric").Key() {
		t.Errorf("only the class is looked up, got %v", keys)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-042, FR-044 to FR-046,
// FR-051 - a refused choice is a Rejected result holding the reason; a class not held is
// refused without looking up any other class.
// Path: per case.
func TestChooseSubclassRejected(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		levels   []character.ClassLevel
		subclass string
		want     character.SubclassRejectionReason
	}{
		{"class not held", []character.ClassLevel{cl("fighter", 4)}, "life_domain", character.SubclassNotInClass},
		{"below the subclass level", []character.ClassLevel{cl("cleric", 2)}, "life_domain", character.SubclassBelowSubclassLevel},
		{"unknown subclass", []character.ClassLevel{cl("cleric", 3)}, "thief", character.SubclassUnknownSubclass},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch, src := subclassFixture(t, c.levels...)
			res, err := chooseUseCase(t, src).Run(context.Background(), chooseCmd(ch, "cleric", c.subclass))
			r, ok := res.(character.ChooseSubclassRejected)
			if err != nil || !ok || r.Rejection.Reason != c.want {
				t.Errorf("= %#v, %v, want %v", res, err, c.want)
			}
			if n := len(lookupKeys(src)); n != 1 {
				t.Errorf("%d lookups, want 1", n)
			}
		})
	}
	ch, src := subclassFixture(t, cl("cleric", 3))
	ch = holding(t, ch, "cleric", "life_domain")
	if r, ok := mustRun(t, chooseUseCase(t, src), chooseCmd(ch, "cleric", "light_domain")).(character.ChooseSubclassRejected); !ok || r.Rejection.Reason != character.SubclassAlreadyChosen {
		t.Errorf("already chosen: %#v", r)
	}
}

func mustRun(t *testing.T, uc character.ChooseSubclass, cmd character.ChooseSubclassCommand) character.ChooseSubclassResult {
	t.Helper()
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-050 - a class the source
// lacks is ContentMissing naming it, and a port failure is a *srderr.PortError with the
// cause reachable; neither builds a result.
// Path: per case.
func TestChooseSubclassContentMissingAndPortFailure(t *testing.T) {
	t.Parallel()
	ch, _ := subclassFixture(t, cl("cleric", 3))
	res := mustRun(t, chooseUseCase(t, contenttest.NewFakeSource()), chooseCmd(ch, "cleric", "life_domain"))
	m, ok := res.(character.ChooseSubclassContentMissing)
	if !ok || len(m.Missing()) != 1 || m.Missing()[0].ID().String() != "cleric" {
		t.Fatalf("= %#v", res)
	}
	m.Missing()[0] = content.MissingContent{}
	if m.Missing()[0].ID().String() != "cleric" {
		t.Error("Missing must return a copy")
	}
	if (character.ChooseSubclassContentMissing{}).Missing() != nil {
		t.Error("the zero result names nothing")
	}
	cause := errors.New("down")
	got, err := chooseUseCase(t, contenttest.NewFakeSource().FailLookup(cause)).Run(context.Background(), chooseCmd(ch, "cleric", "life_domain"))
	var pe *srderr.PortError
	if got != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != "content_error" || !errors.Is(err, cause) {
		t.Errorf("port failure: %v, %v", got, err)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-048, FR-098 - invalid
// commands, the zero use case, a cancelled context and a stored state of another type
// are errors; the state error is recorded on the span only; Noop and a recorder agree.
// Path: per case.
func TestChooseSubclassInvalidCommands(t *testing.T) {
	t.Parallel()
	ch, src := subclassFixture(t, cl("cleric", 3))
	uc, rec := recordedChoose(t, src)
	for name, cmd := range map[string]character.ChooseSubclassCommand{
		"zero character": {Class: "cleric", Subclass: "life_domain"}, "invalid class": chooseCmd(ch, "", "life_domain"), "invalid subclass": chooseCmd(ch, "cleric", ""),
	} {
		if _, err := uc.Run(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, chooseCmd(ch, "cleric", "life_domain")); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()); n != 0 {
		t.Errorf("%d signals before validation", n)
	}
	if _, err := (character.ChooseSubclass{}).Run(context.Background(), chooseCmd(ch, "cleric", "life_domain")); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := character.NewChooseSubclass(); !isInvalid(err) {
		t.Errorf("missing source: %v", err)
	}
	bad, err := character.WithFeatureState(ch, character.MustFeatureStateKey[otherState]("subclass_cleric"), otherState{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Run(context.Background(), chooseCmd(bad, "cleric", "life_domain")); !isInvalid(err) {
		t.Errorf("state of another type: %v", err)
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 2 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 0 {
		t.Errorf("spans = %+v", spans)
	}
	noLogsOrViolations(t, rec)
	a, _ := chooseUseCase(t, src).Run(context.Background(), chooseCmd(ch, "cleric", "life_domain"))
	b, _ := chooseUseCase(t, src).Run(context.Background(), chooseCmd(ch, "cleric", "life_domain"))
	if !reflect.DeepEqual(a, b) {
		t.Error("equal inputs must give equal results")
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-098 to FR-102 - a
// success is outcome success, result chosen; a refusal is a failure without error status
// carrying the rejection; content missing is a failure with error status and no error.type;
// a port failure carries error.type and no result; the duration carries the operation only;
// nothing is logged.
// Path: per case.
func TestChooseSubclassSignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name       string
		levels     []character.ClassLevel
		source     func(*contenttest.FakeSource) *contenttest.FakeSource
		attrs      map[string]any
		spanStatus telemetry.Status
		errors     int
		loadStatus telemetry.Status
	}{
		{"chosen", []character.ClassLevel{cl("cleric", 3)}, nil, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "chosen"}, telemetry.StatusUnset, 0, telemetry.StatusUnset},
		{"rejected", []character.ClassLevel{cl("cleric", 2)}, nil, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "below_subclass_level"}, telemetry.StatusUnset, 0, telemetry.StatusUnset},
		{"content missing", []character.ClassLevel{cl("cleric", 3)}, func(*contenttest.FakeSource) *contenttest.FakeSource { return contenttest.NewFakeSource() },
			map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing"}, telemetry.StatusError, 0, telemetry.StatusError},
		{"port failure", []character.ClassLevel{cl("cleric", 3)}, func(s *contenttest.FakeSource) *contenttest.FakeSource { return s.FailLookup(errors.New("down")) },
			map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"}, telemetry.StatusError, 1, telemetry.StatusError},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch, src := subclassFixture(t, c.levels...)
			if c.source != nil {
				src = c.source(src)
			}
			uc, rec := recordedChoose(t, src)
			_, err := uc.Run(context.Background(), chooseCmd(ch, "cleric", "life_domain"))
			if (err != nil) != (c.errors == 1) {
				t.Fatalf("err = %v", err)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Name != character.SpanAdvancementChooseSubclass || spans[1].Name != character.SpanAdvancementLoadContent || spans[1].ParentID != spans[0].ID ||
				spans[0].Status != c.spanStatus || len(spans[0].Errors) != c.errors || spans[1].Status != c.loadStatus {
				t.Fatalf("spans = %+v", spans)
			}
			ops := onlyOne(t, rec, character.MetricOperations)
			sameAttrs(t, "operations", ops.Attrs, opAttrs("choose_subclass", c.attrs))
			if ops.Value != 1 {
				t.Errorf("operations = %+v", ops)
			}
			d := onlyOne(t, rec, character.MetricDuration)
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "choose_subclass"})
			if d.Value != 0.25 || d.Unit != "s" {
				t.Errorf("duration = %+v", d)
			}
			noLogsOrViolations(t, rec)
		})
	}
}
