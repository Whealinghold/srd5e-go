package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// levelUpFixture is a fighter at level 5 with the XP for level 6 and a source
// holding the fighter and wizard classes (both with Intelligence as their
// Primary Ability, from the bundled Wizard document).
type levelUpFixture struct {
	char   character.Character
	source *contenttest.FakeSource
}

func newLevelUpFixture(t *testing.T, scores ability.Scores) levelUpFixture {
	t.Helper()
	return levelUpFixture{
		char:   restored(t, 14000, 40, scores, cl("fighter", 5)),
		source: contenttest.NewFakeSource(fighterClass(t), wizardClass(t)),
	}
}

func (f levelUpFixture) useCase(t *testing.T, roller character.Roller, opts ...character.Option) character.LevelUpCharacter {
	t.Helper()
	uc, err := character.NewLevelUpCharacter(append([]character.Option{character.WithSource(f.source), character.WithRoller(roller)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func (f levelUpFixture) cmd(class string, choice character.HitPointChoice) character.LevelUpCharacterCommand {
	return character.LevelUpCharacterCommand{Character: f.char, Class: content.ClassID(class), HitPointChoice: choice}
}

func runLevelUp(t *testing.T, uc character.LevelUpCharacter, cmd character.LevelUpCharacterCommand) character.LevelUpCharacterResult {
	t.Helper()
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

func lookupKeys(src *contenttest.FakeSource) []content.Key {
	var keys []content.Key
	for _, q := range src.Queries() {
		if l, ok := q.(content.LookupQuery); ok {
			keys = append(keys, l.Key)
		}
	}
	return keys
}

// Requirement: doc/specs/character-aggregate.md FR-064, FR-066 - the fixed choice
// never calls the roller and a held class looks up only the chosen class.
// Path: happy.
func TestLevelUpCharacterFixed(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 10))
	roller := dicetest.NewFixedRoller()
	res := runLevelUp(t, f.useCase(t, roller), f.cmd("fighter", character.TakeFixedValue{}))
	ok, isOK := res.(character.LevelUpCharacterSucceeded)
	if !isOK || ok.Character().TotalLevel().Value() != 6 || ok.Character().HitPointMaximum().Value() != 46 || len(ok.Events()) != 1 {
		t.Fatalf("result = %#v", res)
	}
	if len(roller.Requests()) != 0 {
		t.Error("FR-064: the fixed choice must not roll")
	}
	if keys := lookupKeys(f.source); len(keys) != 1 || keys[0] != content.ClassID("fighter").Key() {
		t.Errorf("FR-066: lookups = %v", keys)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-063, FR-065 - the rolled
// choice rolls exactly one die of the class's size and uses the value; rolls of 1
// and of the maximum give the clamped minimum and the maximum gain.
// Path: per case.
func TestLevelUpCharacterRolled(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		value int
		con   int
		want  int
	}{{"minimum roll", 1, 10, 1}, {"maximum roll", 10, 10, 10}, {"clamped", 1, 6, 1}, {"with modifier", 4, 14, 6}} {
		f := newLevelUpFixture(t, scoresOf(t, 10, 10, c.con, 10, 10, 10))
		roller := dicetest.NewFixedRoller(c.value)
		res := runLevelUp(t, f.useCase(t, roller), f.cmd("fighter", character.RollHitDie{}))
		got := res.(character.LevelUpCharacterSucceeded).Character().HitPointMaximum().Value() - 40
		reqs := roller.Requests()
		if got != c.want || len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
			t.Errorf("%s: gain %d, requests %v", c.name, got, reqs)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-066, FR-026 - a new class looks
// up the chosen class and every current class, and evaluates the prerequisites.
// Path: happy.
func TestLevelUpCharacterNewClass(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 13)) // Intelligence 13 meets both classes
	res := runLevelUp(t, f.useCase(t, dicetest.NewFixedRoller()), f.cmd("wizard", character.TakeFixedValue{}))
	n := res.(character.LevelUpCharacterSucceeded).Character()
	if n.ClassLevels().LevelIn("wizard") != 1 || n.TotalLevel().Value() != 6 {
		t.Errorf("classes = %v", n.ClassLevels().Entries())
	}
	keys := lookupKeys(f.source)
	if len(keys) != 2 || keys[0] != content.ClassID("wizard").Key() || keys[1] != content.ClassID("fighter").Key() {
		t.Errorf("lookups = %v", keys)
	}
	weak := newLevelUpFixture(t, flatScores(t, 10))
	rej, ok := runLevelUp(t, weak.useCase(t, dicetest.NewFixedRoller()), weak.cmd("wizard", character.TakeFixedValue{})).(character.LevelUpCharacterRejected)
	short, isShort := rej.Rejection.(character.LevelUpPrerequisitesNotMet)
	if !ok || !isShort || len(short.ShortAbilities()) != 1 || short.ShortAbilities()[0] != ability.Intelligence {
		t.Errorf("prerequisites: %#v", rej)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-067 - each LevelUpRejection
// reaches LevelUpCharacterRejected.
// Path: per case.
func TestLevelUpCharacterRejections(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 10))
	max := f
	max.char = restored(t, 400000, 300, flatScores(t, 10), cl("fighter", 20))
	poor := f
	poor.char = restored(t, 0, 40, flatScores(t, 10), cl("fighter", 5))
	for _, c := range []struct {
		name string
		fx   levelUpFixture
		want string
	}{{"maximum", max, "max"}, {"experience", poor, "xp"}} {
		res := runLevelUp(t, c.fx.useCase(t, dicetest.NewFixedRoller()), c.fx.cmd("fighter", character.TakeFixedValue{}))
		rej, ok := res.(character.LevelUpCharacterRejected)
		if !ok || levelUpRejectionTag(rej.Rejection) != c.want {
			t.Errorf("%s: %#v", c.name, res)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-061, FR-067, FR-076, FR-077 -
// content missing wins over every rejection, names every missing class once in id
// order, and does not evaluate prerequisites.
// Path: per case.
func TestLevelUpCharacterContentMissing(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 10))
	f.source = contenttest.NewFakeSource()
	res := runLevelUp(t, f.useCase(t, dicetest.NewFixedRoller()), f.cmd("wizard", character.TakeFixedValue{}))
	missing, ok := res.(character.LevelUpCharacterContentMissing)
	if !ok || len(missing.Missing()) != 2 || missing.Missing()[0].ID().String() != "fighter" || missing.Missing()[1].ID().String() != "wizard" {
		t.Fatalf("both classes missing: %#v", res)
	}
	missing.Missing()[0] = content.MissingContent{}
	if missing.Missing()[0].ID().String() != "fighter" {
		t.Error("Missing must return a copy")
	}
	// A level 20 character whose chosen class is missing gets content missing, not the maximum.
	max := newLevelUpFixture(t, flatScores(t, 10))
	max.char = restored(t, 400000, 300, flatScores(t, 10), cl("fighter", 20))
	max.source = contenttest.NewFakeSource(fighterClass(t))
	if _, ok := runLevelUp(t, max.useCase(t, dicetest.NewFixedRoller()), max.cmd("wizard", character.TakeFixedValue{})).(character.LevelUpCharacterContentMissing); !ok {
		t.Error("FR-067: content missing must win over LevelUpMaximumLevel")
	}
	// Only the current class missing.
	cur := newLevelUpFixture(t, flatScores(t, 13))
	cur.source = contenttest.NewFakeSource(wizardClass(t))
	r := runLevelUp(t, cur.useCase(t, dicetest.NewFixedRoller()), cur.cmd("wizard", character.TakeFixedValue{}))
	if m, ok := r.(character.LevelUpCharacterContentMissing); !ok || len(m.Missing()) != 1 || m.Missing()[0].ID().String() != "fighter" {
		t.Errorf("FR-076: %#v", r)
	}
	if (character.LevelUpCharacterContentMissing{}).Missing() != nil {
		t.Error("the zero result names nothing")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-062, FR-078, FR-079 - a source
// failure, a roller failure and a roller breaking its contract come back as
// *srderr.PortError of the right port, with no result.
// Path: per case.
func TestLevelUpCharacterPortFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	badAnswer := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dice.NewResult(dice.MustExpression(1, dice.D6(), 0), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 3)})
	})
	for _, c := range []struct {
		name     string
		source   func(*contenttest.FakeSource) *contenttest.FakeSource
		roller   character.Roller
		wantType string
		contract bool
	}{
		{"source fails", func(s *contenttest.FakeSource) *contenttest.FakeSource { return s.FailLookup(cause) }, dicetest.NewFixedRoller(1), "content_error", false},
		{"roller fails", func(s *contenttest.FakeSource) *contenttest.FakeSource { return s }, dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), "dice_roller_error", false},
		{"roller answers another expression", func(s *contenttest.FakeSource) *contenttest.FakeSource { return s }, badAnswer, "dice_roller_error", true},
		{"roller is exhausted", func(s *contenttest.FakeSource) *contenttest.FakeSource { return s }, dicetest.NewFixedRoller(), "dice_roller_error", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := newLevelUpFixture(t, flatScores(t, 10))
			f.source = c.source(f.source)
			res, err := f.useCase(t, c.roller).Run(context.Background(), f.cmd("fighter", character.RollHitDie{}))
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != c.wantType || errors.Is(err, srderr.ErrContractViolation) != c.contract {
				t.Errorf("res %v err %v", res, err)
			}
			if c.name == "source fails" && !errors.Is(err, cause) {
				t.Error("the cause must stay reachable")
			}
		})
	}
}

// Requirement: doc/specs/character-aggregate.md FR-079 - a roll outside the die
// cannot reach the use case: dice.DieRoll refuses to hold one.
// Path: unhappy (collaborator failure).
func TestRollOutsideTheDieCannotBeBuilt(t *testing.T) {
	t.Parallel()
	for _, v := range []int{0, 11} {
		if _, err := dice.NewDieRoll(dice.D10(), v); err == nil {
			t.Errorf("a roll of %d on a d10 must be refused", v)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-087, FR-059 - Run does not alter
// the command's Character; invalid commands are validation errors, never
// results; a cancelled context is returned unwrapped before any signal.
// Path: per case.
func TestLevelUpCharacterInvalidCommands(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 10))
	rec := telemetrytest.NewRecorder()
	uc := f.useCase(t, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry()))
	before := f.char
	runLevelUp(t, uc, f.cmd("fighter", character.TakeFixedValue{}))
	if !f.char.Equal(before) {
		t.Error("FR-087: the command's character must be unchanged")
	}
	rec.Meter.Clear()
	for name, cmd := range map[string]character.LevelUpCharacterCommand{
		"zero character": {Class: "fighter", HitPointChoice: character.TakeFixedValue{}},
		"invalid class":  {Character: f.char, Class: "", HitPointChoice: character.TakeFixedValue{}},
		"nil choice":     {Character: f.char, Class: "fighter"},
	} {
		if _, err := uc.Run(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, f.cmd("fighter", character.TakeFixedValue{})); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("%d measurements before validation", n)
	}
	if _, err := (character.LevelUpCharacter{}).Run(context.Background(), f.cmd("fighter", character.TakeFixedValue{})); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := character.NewLevelUpCharacter(character.WithSource(f.source)); !isInvalid(err) {
		t.Errorf("missing roller: %v", err)
	}
	if _, err := character.NewLevelUpCharacter(character.WithRoller(dicetest.NewFixedRoller())); !isInvalid(err) {
		t.Errorf("missing source: %v", err)
	}
}

func recordedLevelUp(t *testing.T, f levelUpFixture, roller character.Roller) (character.LevelUpCharacter, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return f.useCase(t, roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)), rec
}

func opAttrs(op string, rest map[string]any) map[string]any {
	m := map[string]any{character.AttrOperation: op}
	for k, v := range rest {
		m[k] = v
	}
	return m
}

// Requirement: doc/specs/character-aggregate.md FR-069, FR-070, FR-072, FR-103 - a
// success is one level_up span with a load_content child, one operations
// increment (outcome success, result changed, choice) and one duration with the
// operation attribute only; nothing is logged; Noop and a recorder give the same
// result.
// Path: happy.
func TestLevelUpCharacterSuccessSignals(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 10))
	uc, rec := recordedLevelUp(t, f, dicetest.NewFixedRoller(4))
	got := runLevelUp(t, uc, f.cmd("fighter", character.RollHitDie{}))
	plain := runLevelUp(t, f.useCase(t, dicetest.NewFixedRoller(4)), f.cmd("fighter", character.RollHitDie{}))
	if !got.(character.LevelUpCharacterSucceeded).Character().Equal(plain.(character.LevelUpCharacterSucceeded).Character()) {
		t.Error("FR-069: Noop and a recording bundle must agree")
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 2 || spans[0].Name != character.SpanLevelUp || spans[1].Name != character.SpanLoadContent || spans[1].ParentID != spans[0].ID || spans[0].Status != telemetry.StatusUnset || !spans[0].Ended || !spans[1].Ended {
		t.Fatalf("spans = %+v", spans)
	}
	sameAttrs(t, "level_up span", spans[0].Attrs, map[string]any{
		character.AttrHitPointChoice: "rolled", telemetry.AttrOutcome: "success", character.AttrResult: "changed",
	})
	sameAttrs(t, "load_content span", spans[1].Attrs, outcome("success"))
	ops := onlyOne(t, rec, character.MetricOperations)
	if ops.Value != 1 || ops.Unit != "{operation}" || ops.Kind != telemetrytest.KindCounter {
		t.Errorf("operations = %+v", ops)
	}
	sameAttrs(t, "operations", ops.Attrs, opAttrs("level_up", map[string]any{
		telemetry.AttrOutcome: "success", character.AttrResult: "changed", character.AttrHitPointChoice: "rolled",
	}))
	d := onlyOne(t, rec, character.MetricDuration)
	if d.Value != 0.25 || d.Unit != "s" {
		t.Errorf("duration = %+v", d)
	}
	sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "level_up"})
	noLogsOrViolations(t, rec)
}

// Requirement: doc/specs/character-aggregate.md FR-071, FR-072, FR-088 - a
// rejection is a failure without error status carrying the rejection; content
// missing is a failure with error status and no error.type; a port failure
// carries error.type, the recorded error and no result.
// Path: per case.
func TestLevelUpCharacterFailureSignals(t *testing.T) {
	t.Parallel()
	type want struct {
		attrs      map[string]any
		spanStatus telemetry.Status
		errors     int
		loadStatus telemetry.Status
	}
	for _, c := range []struct {
		name string
		prep func(*levelUpFixture)
		want want
	}{
		{"rejection", func(f *levelUpFixture) { f.char = restored(t, 0, 40, flatScores(t, 10), cl("fighter", 5)) }, want{
			opAttrs("level_up", map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "insufficient_experience", character.AttrHitPointChoice: "fixed"}),
			telemetry.StatusUnset, 0, telemetry.StatusUnset}},
		{"content missing", func(f *levelUpFixture) { f.source = contenttest.NewFakeSource() }, want{
			opAttrs("level_up", map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing", character.AttrHitPointChoice: "fixed"}),
			telemetry.StatusError, 0, telemetry.StatusError}},
		{"port failure", func(f *levelUpFixture) { f.source = f.source.FailLookup(errors.New("down")) }, want{
			opAttrs("level_up", map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error", character.AttrHitPointChoice: "fixed"}),
			telemetry.StatusError, 1, telemetry.StatusError}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := newLevelUpFixture(t, flatScores(t, 10))
			c.prep(&f)
			uc, rec := recordedLevelUp(t, f, dicetest.NewFixedRoller())
			_, err := uc.Run(context.Background(), f.cmd("fighter", character.TakeFixedValue{}))
			if (err != nil) != (c.want.errors == 1) {
				t.Fatalf("err = %v", err)
			}
			spans := rec.Tracer.Spans()
			if spans[0].Status != c.want.spanStatus || len(spans[0].Errors) != c.want.errors || spans[1].Status != c.want.loadStatus {
				t.Errorf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, c.want.attrs)
			sameAttrs(t, "duration", onlyOne(t, rec, character.MetricDuration).Attrs, map[string]any{character.AttrOperation: "level_up"})
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/character-aggregate.md FR-078 - a roller failure is
// recorded once on the level_up span and the operations counter with error.type
// dice_roller_error and no result.
// Path: unhappy (telemetry on failure).
func TestLevelUpCharacterRollerFailureSignals(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 10))
	uc, rec := recordedLevelUp(t, f, dicetest.NewFixedRoller()) // exhausted at once
	if _, err := uc.Run(context.Background(), f.cmd("fighter", character.RollHitDie{})); srderr.ErrorType(err) != "dice_roller_error" {
		t.Fatalf("err = %v", err)
	}
	spans := rec.Tracer.Spans()
	if spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || spans[1].Status != telemetry.StatusUnset {
		t.Errorf("spans = %+v", spans)
	}
	sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("level_up", map[string]any{
		telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error", character.AttrHitPointChoice: "rolled",
	}))
	noLogsOrViolations(t, rec)
}

// Requirement: doc/specs/character-aggregate.md FR-072 - every rejection value is
// recorded as its snake_case string.
// Path: per case.
func TestRejectionValues(t *testing.T) {
	t.Parallel()
	for _, v := range []struct{ got, want string }{
		{character.ValueRejectionMaximumLevel, "maximum_level"}, {character.ValueRejectionInsufficientExperience, "insufficient_experience"},
		{character.ValueRejectionMulticlassPrerequisitesNotMet, "multiclass_prerequisites_not_met"}, {character.ValueRejectionExceedsMaximumScore, "exceeds_maximum_score"},
		{character.ValueRejectionConstitutionModifierDecrease, "constitution_modifier_decrease"},
	} {
		if v.got != v.want {
			t.Errorf("%q, want %q", v.got, v.want)
		}
	}
	f := newLevelUpFixture(t, flatScores(t, 10))
	for _, c := range []struct {
		fx    func() levelUpFixture
		class string
		want  string
	}{
		{func() levelUpFixture {
			g := f
			g.char = restored(t, 400000, 300, flatScores(t, 10), cl("fighter", 20))
			return g
		}, "fighter", "maximum_level"},
		{func() levelUpFixture { return f }, "wizard", "multiclass_prerequisites_not_met"},
	} {
		fx := c.fx()
		uc, rec := recordedLevelUp(t, fx, dicetest.NewFixedRoller())
		if _, err := uc.Run(context.Background(), fx.cmd(c.class, character.TakeFixedValue{})); err != nil {
			t.Fatal(err)
		}
		if got := attrMap(onlyOne(t, rec, character.MetricOperations).Attrs)[character.AttrRejection]; got != c.want {
			t.Errorf("rejection = %v, want %s", got, c.want)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-066 - the chosen class's
// definition is the one it was looked up for, even when a current class sorts
// after it; its hit die sizes the fixed gain and every current class's Primary
// Ability reaches the prerequisites.
// Path: happy and rejection.
func TestLevelUpCharacterNewClassSortingBeforeCurrent(t *testing.T) {
	t.Parallel()
	f := newLevelUpFixture(t, flatScores(t, 13))
	f.char = restored(t, 14000, 30, flatScores(t, 13), cl("wizard", 5))
	res := runLevelUp(t, f.useCase(t, dicetest.NewFixedRoller()), f.cmd("fighter", character.TakeFixedValue{}))
	ok, isOK := res.(character.LevelUpCharacterSucceeded)
	if !isOK || ok.Character().ClassLevels().LevelIn("fighter") != 1 || ok.Character().HitPointMaximum().Value() != 30+6+1 {
		t.Fatalf("fighter d10 fixed gain 6 plus Constitution 1 expected: %#v", res)
	}
	weak := newLevelUpFixture(t, flatScores(t, 10))
	weak.char = restored(t, 14000, 30, scoresOf(t, 13, 13, 10, 8, 10, 10), cl("wizard", 5))
	rej, isRej := runLevelUp(t, weak.useCase(t, dicetest.NewFixedRoller()), weak.cmd("fighter", character.TakeFixedValue{})).(character.LevelUpCharacterRejected)
	if !isRej || levelUpRejectionTag(rej.Rejection) != "prereq" {
		t.Errorf("the current wizard's Intelligence requirement must apply: %#v", rej)
	}
}
