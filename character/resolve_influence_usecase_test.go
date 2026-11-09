package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func influenceTarget(t *testing.T, intelligence int, ct content.CreatureType, att character.Attitude) character.InfluenceTarget {
	t.Helper()
	return must(character.NewInfluenceTarget(score(t, intelligence), ct, character.Understands, character.WithAttitude(att)))
}

// chaRoll is a rolled attempt with Charisma +chaMod and nothing else.
func chaRoll(chaMod int, opts ...character.RollOption) character.InfluenceRoll {
	return character.NewInfluenceRoll(map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(chaMod)}, opts...)
}

func hesitant(t *testing.T, att character.Attitude, a character.InfluenceAttempt, opts ...character.HesitantOption) character.InfluenceHesitantCommand {
	t.Helper()
	return must(character.NewInfluenceHesitantCommand(character.Persuade, influenceTarget(t, 10, content.CreatureTypeHumanoid, att), a, opts...))
}

func newInfluencer(t *testing.T, roller character.Roller, opts ...character.Option) character.InfluenceResolver {
	t.Helper()
	return must(character.NewInfluenceResolver(roller, opts...))
}

func d20Expr(n int) dice.Expression { return dice.MustExpression(n, dice.D20(), 0) }

// Requirement: doc/specs/social-interaction.md FR-013, FR-014, FR-024 - a Willing command
// fulfils and an Unwilling one refuses, each without a request to the roller, as field-less
// results (SRD 5.2.1, "Influence [Action]", p. 184).
// Path: happy.
func TestResolveInfluenceWillingAndUnwillingRollNothing(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		cmd  character.InfluenceCommand
		want character.ResolveInfluenceResult
	}{
		"willing":   {character.InfluenceWillingCommand{}, character.ResolveInfluenceFulfilled{}},
		"unwilling": {character.InfluenceUnwillingCommand{}, character.ResolveInfluenceRefused{}},
	} {
		roller := dicetest.NewFixedRoller(20)
		got, err := newInfluencer(t, roller).ResolveInfluence(context.Background(), c.cmd)
		if err != nil || got != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s: %#v, %v, %d requests", name, got, err, len(roller.Requests()))
		}
	}
}

// Requirement: doc/specs/social-interaction.md FR-015, FR-016, FR-017 - the roll mode is
// modifier.Resolve over the attitude's circumstances then the caller's; exactly one request
// of 1d20 or 2d20 is made and the d20.Roll holds the faces in served order (SRD 5.2.1,
// "Friendly", p. 182; "Hostile", p. 183; "They Don't Stack", p. 8).
// Path: per case.
func TestResolveInfluenceRollMode(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	cases := []struct {
		name, requirement, path string
		att                     character.Attitude
		circs                   []modifier.RollCircumstance
		want                    modifier.RollMode
	}{
		{"Friendly alone", "FR-016", "happy", character.Friendly, nil, modifier.WithAdvantage},
		{"Indifferent alone", "FR-016", "unhappy (empty and minimal)", character.Indifferent, nil, modifier.Normal},
		{"Hostile alone", "FR-016", "happy", character.Hostile, nil, modifier.WithDisadvantage},
		{"Friendly plus Advantage", "FR-016", "unhappy (rule interaction)", character.Friendly, []modifier.RollCircumstance{adv}, modifier.WithAdvantage},
		{"Friendly plus Disadvantage", "FR-016", "unhappy (rule interaction)", character.Friendly, []modifier.RollCircumstance{dis}, modifier.Normal},
		{"Hostile plus Advantage", "FR-016", "unhappy (rule interaction)", character.Hostile, []modifier.RollCircumstance{adv}, modifier.Normal},
		{"Indifferent plus Advantage", "FR-016", "happy", character.Indifferent, []modifier.RollCircumstance{adv}, modifier.WithAdvantage},
		{"Hostile plus two Advantages", "FR-016", "unhappy (rule interaction)", character.Hostile, []modifier.RollCircumstance{adv, adv}, modifier.Normal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			faces := []int{7, 13}[:c.want.DiceCount()]
			roller := dicetest.NewFixedRoller(faces...)
			got, err := newInfluencer(t, roller).ResolveInfluence(context.Background(), hesitant(t, c.att, chaRoll(3, character.WithCircumstances(c.circs...))))
			var check d20.CheckResult
			switch r := got.(type) {
			case character.ResolveInfluenceCheckSucceeded:
				check = r.Check()
			case character.ResolveInfluenceCheckFailed:
				check = r.Check()
			default:
				t.Fatalf("result = %#v, %v", got, err)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != d20Expr(c.want.DiceCount()) {
				t.Errorf("requests = %v, want one %v [%s, %s]", reqs, d20Expr(c.want.DiceCount()), c.requirement, c.path)
			}
			if check.Roll().Mode() != c.want || !reflect.DeepEqual(check.Roll().Dice(), faces) || check.Kind() != d20.AbilityCheck {
				t.Errorf("roll = %v %v kind %v, want %v %v [%s]", check.Roll().Mode(), check.Roll().Dice(), check.Kind(), c.want, faces, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-018, FR-019, FR-020, FR-023, FR-040 - the
// check succeeds when the total reaches the DC (15 for Intelligence 10) and fails below it;
// a natural 20 or 1 changes nothing; a failure carries 24 hours or the GM's wait (zero
// accepted); the kept die follows Advantage and Disadvantage (SRD 5.2.1, "Influence
// [Action]", p. 184; "Compare the Total to a Target Number", p. 6; "Rolling 20 or 1", p. 7).
// Path: per case.
func TestResolveInfluenceResolution(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		att                     character.Attitude
		faces                   []int
		mod                     int
		opts                    []character.HesitantOption
		wantSuccess             bool
		wantTotal               int
		wantWait                time.Duration
	}{
		{"total equals DC", "FR-018", "unhappy (boundary)", character.Indifferent, []int{12}, 3, nil, true, 15, 0},
		{"total one below", "FR-019", "unhappy (boundary)", character.Indifferent, []int{11}, 3, nil, false, 14, 24 * time.Hour},
		{"natural 20, total below DC", "FR-023", "unhappy (boundary)", character.Indifferent, []int{20}, -10, nil, false, 10, 24 * time.Hour},
		{"natural 1, total reaches DC", "FR-023", "unhappy (boundary)", character.Indifferent, []int{1}, 17, nil, true, 18, 0},
		{"Friendly keeps the higher", "FR-016", "happy", character.Friendly, []int{4, 16}, 3, nil, true, 19, 0},
		{"Hostile keeps the lower", "FR-016", "happy", character.Hostile, []int{16, 4}, 3, nil, false, 7, 24 * time.Hour},
		{"GM wait of 1 hour", "FR-020", "happy", character.Indifferent, []int{2}, 3, []character.HesitantOption{character.WithGMRetryAfter(time.Hour)}, false, 5, time.Hour},
		{"GM wait of zero", "FR-020", "unhappy (boundary)", character.Indifferent, []int{2}, 3, []character.HesitantOption{character.WithGMRetryAfter(0)}, false, 5, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := newInfluencer(t, dicetest.NewFixedRoller(c.faces...)).ResolveInfluence(context.Background(), hesitant(t, c.att, chaRoll(c.mod), c.opts...))
			if err != nil {
				t.Fatal(err)
			}
			var (
				check d20.CheckResult
				wait  time.Duration
				base  interface {
					Approach() character.InfluenceApproach
					Attitude() character.Attitude
					DC() d20.DifficultyClass
					DCBasis() character.DCBasis
				}
			)
			switch r := got.(type) {
			case character.ResolveInfluenceCheckSucceeded:
				check, base = r.Check(), r
				if !c.wantSuccess {
					t.Fatalf("succeeded, want failure [%s, %s]", c.requirement, c.path)
				}
			case character.ResolveInfluenceCheckFailed:
				check, wait, base = r.Check(), r.RetryAfter(), r
				if c.wantSuccess {
					t.Fatalf("failed, want success [%s, %s]", c.requirement, c.path)
				}
			default:
				t.Fatalf("result = %#v", got)
			}
			if check.Total() != c.wantTotal || wait != c.wantWait || check.Target() != 15 {
				t.Errorf("total %d wait %v target %d, want %d %v 15 [%s]", check.Total(), wait, check.Target(), c.wantTotal, c.wantWait, c.requirement)
			}
			if base.Approach() != character.Persuade || base.Attitude() != c.att || base.DC() != 15 || base.DCBasis() != character.DefaultRule {
				t.Errorf("carried = %v %v %v %v", base.Approach(), base.Attitude(), base.DC(), base.DCBasis())
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-011, FR-012, FR-009 - a GM-set DC replaces
// the default (8 and 25 against Intelligence 10 and 20) and is recorded as GMSet; a default
// DC is recorded as DefaultRule; an approach not suggested for the target is still resolved
// (SRD 5.2.1, "Influence [Action]", p. 184).
// Path: per case.
func TestResolveInfluenceDifficultyClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		intelligence            int
		opts                    []character.HesitantOption
		approach                character.InfluenceApproach
		ct                      content.CreatureType
		wantDC                  d20.DifficultyClass
		wantBasis               character.DCBasis
		wantSuccess             bool
		face                    int
	}{
		{"GM DC 8 against Intelligence 10", "FR-011", "happy", 10, []character.HesitantOption{character.WithGMDC(8)}, character.Persuade, content.CreatureTypeHumanoid, 8, character.GMSet, true, 15},
		{"GM DC 25 against Intelligence 20", "FR-011", "happy", 20, []character.HesitantOption{character.WithGMDC(25)}, character.Persuade, content.CreatureTypeHumanoid, 25, character.GMSet, false, 20},
		{"default DC for Intelligence 20", "FR-012", "happy", 20, nil, character.Persuade, content.CreatureTypeHumanoid, 20, character.DefaultRule, true, 20},
		{"Coax on a Humanoid is not rejected", "FR-009", "unhappy (rule interaction)", 10, nil, character.Coax, content.CreatureTypeHumanoid, 15, character.DefaultRule, true, 15},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			target := influenceTarget(t, c.intelligence, c.ct, character.Indifferent)
			attempt := character.NewInfluenceRoll(map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(0), ability.Wisdom: modifier.New(0)})
			cmd := must(character.NewInfluenceHesitantCommand(c.approach, target, attempt, c.opts...))
			roller := dicetest.NewFixedRoller(c.face)
			got, err := newInfluencer(t, roller).ResolveInfluence(context.Background(), cmd)
			if err != nil || len(roller.Requests()) != 1 {
				t.Fatalf("%#v, %v, %d requests", got, err, len(roller.Requests()))
			}
			var dc d20.DifficultyClass
			var basis character.DCBasis
			var ok bool
			switch r := got.(type) {
			case character.ResolveInfluenceCheckSucceeded:
				dc, basis, ok = r.DC(), r.DCBasis(), r.Check().Target() == c.wantDC
			case character.ResolveInfluenceCheckFailed:
				dc, basis, ok = r.DC(), r.DCBasis(), r.Check().Target() == c.wantDC
			}
			_, succeeded := got.(character.ResolveInfluenceCheckSucceeded)
			if !ok || dc != c.wantDC || basis != c.wantBasis || succeeded != c.wantSuccess {
				t.Errorf("dc %d basis %v target-ok %v succeeded %v, want %d %v %v [%s]", dc, basis, ok, succeeded, c.wantDC, c.wantBasis, c.wantSuccess, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-017, FR-041, FR-042 - the check uses the
// ability modifier of the approach's ability (Persuade the Charisma entry, Coax the Wisdom
// entry), then the Proficiency Bonus and the other modifiers in order; with only the
// ability modifier the total is the natural d20 plus it (SRD 5.2.1, "D20 Tests", p. 6;
// "Influence Checks" table, p. 184).
// Path: per case.
func TestResolveInfluenceModifiers(t *testing.T) {
	t.Parallel()
	both := map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(4), ability.Wisdom: modifier.New(-1), ability.Strength: modifier.New(9)}
	cases := []struct {
		name, requirement, path string
		approach                character.InfluenceApproach
		opts                    []character.RollOption
		wantTotal               int
		wantOthers              []modifier.Modifier
		wantProf                bool
	}{
		{"Persuade reads Charisma", "FR-041", "happy", character.Persuade, nil, 14, []modifier.Modifier{}, false},
		{"Coax reads Wisdom", "FR-041", "unhappy (rule interaction)", character.Coax, nil, 9, []modifier.Modifier{}, false},
		{"ability, Proficiency Bonus and others", "FR-017", "happy", character.Persuade,
			[]character.RollOption{character.WithProficiencyBonus(modifier.New(2)), character.WithOtherModifiers(modifier.New(3), modifier.New(-1))}, 18,
			[]modifier.Modifier{modifier.New(3), modifier.New(-1)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := must(character.NewInfluenceHesitantCommand(c.approach, influenceTarget(t, 10, content.CreatureTypeBeast, character.Indifferent), character.NewInfluenceRoll(both, c.opts...)))
			got, err := newInfluencer(t, dicetest.NewFixedRoller(10)).ResolveInfluence(context.Background(), cmd)
			var mods d20.RollModifiers
			var total int
			switch r := got.(type) {
			case character.ResolveInfluenceCheckSucceeded:
				mods, total = r.Check().Modifiers(), r.Check().Total()
			case character.ResolveInfluenceCheckFailed:
				mods, total = r.Check().Modifiers(), r.Check().Total()
			default:
				t.Fatalf("%#v, %v", got, err)
			}
			prof, hasProf := mods.ProficiencyBonus()
			if total != c.wantTotal || hasProf != c.wantProf || (hasProf && prof.Value() != 2) || !slices.Equal(mods.Others(), c.wantOthers) {
				t.Errorf("total %d prof %v/%v others %v, want %d %v %v [%s, %s]", total, prof, hasProf, mods.Others(), c.wantTotal, c.wantProf, c.wantOthers, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-022, FR-043 - an InfluenceAutomaticFailure
// makes no request, applies no attitude, and returns the forced failure with reason
// automatic, the approach, attitude, DC, basis and 24 hours (or the GM's wait) (SRD 5.2.1,
// "Blinded", p. 177; "Influence [Action]", p. 184).
// Path: unhappy (rule interaction).
func TestResolveInfluenceAutomaticFailure(t *testing.T) {
	t.Parallel()
	for _, att := range []character.Attitude{character.Friendly, character.Indifferent, character.Hostile} {
		roller := dicetest.NewFixedRoller(20)
		cmd := hesitant(t, att, character.InfluenceAutomaticFailure{}, character.WithGMDC(9))
		got, err := newInfluencer(t, roller).ResolveInfluence(context.Background(), cmd)
		r, ok := got.(character.ResolveInfluenceCheckForcedFailure)
		if err != nil || !ok || len(roller.Requests()) != 0 || r.Check().Reason() != d20.Automatic || r.Check().Kind() != d20.AbilityCheck ||
			r.RetryAfter() != 24*time.Hour || r.DC() != 9 || r.DCBasis() != character.GMSet || r.Attitude() != att || r.Approach() != character.Persuade {
			t.Errorf("%v: %#v, %v", att, got, err)
		}
	}
	wait := hesitant(t, character.Indifferent, character.InfluenceAutomaticFailure{}, character.WithGMRetryAfter(time.Minute))
	got, _ := newInfluencer(t, dicetest.NewFixedRoller()).ResolveInfluence(context.Background(), wait)
	if r, ok := got.(character.ResolveInfluenceCheckForcedFailure); !ok || r.RetryAfter() != time.Minute {
		t.Errorf("GM wait: %#v", got)
	}
}

// Requirement: doc/specs/social-interaction.md FR-021, FR-041, FR-030 - the hesitant
// constructor rejects a negative GM wait, an InfluenceRoll without the approach's ability,
// an invalid approach or target and a nil attempt with ErrInvalidInput and the zero command;
// MustInfluenceHesitantCommand panics with that error; extra abilities are ignored.
// Path: per case.
func TestNewInfluenceHesitantCommandValidation(t *testing.T) {
	t.Parallel()
	target := influenceTarget(t, 10, content.CreatureTypeBeast, character.Hostile)
	cases := []struct {
		name, requirement, path string
		approach                character.InfluenceApproach
		target                  character.InfluenceTarget
		attempt                 character.InfluenceAttempt
		opts                    []character.HesitantOption
	}{
		{"negative GM wait", "FR-021", "unhappy (invalid input)", character.Persuade, target, chaRoll(0), []character.HesitantOption{character.WithGMRetryAfter(-1 * time.Second)}},
		{"no Wisdom entry for Coax", "FR-041", "unhappy (invalid input)", character.Coax, target, chaRoll(0), nil},
		{"empty ability map", "FR-041", "unhappy (empty and minimal)", character.Persuade, target, character.NewInfluenceRoll(nil), nil},
		{"invalid approach", "FR-041", "unhappy (invalid input)", 0, target, character.InfluenceAutomaticFailure{}, nil},
		{"zero target", "FR-030", "unhappy (invalid input)", character.Persuade, character.InfluenceTarget{}, character.InfluenceAutomaticFailure{}, nil},
		{"nil attempt", "FR-041", "unhappy (invalid input)", character.Persuade, target, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewInfluenceHesitantCommand(c.approach, c.target, c.attempt, c.opts...)
			if !errors.Is(err, srderr.ErrInvalidInput) || !got.Equal(character.InfluenceHesitantCommand{}) {
				t.Errorf("got %#v, %v [%s, %s]", got, err, c.requirement, c.path)
			}
			defer func() {
				if p, _ := recover().(error); !errors.Is(p, srderr.ErrInvalidInput) {
					t.Errorf("MustInfluenceHesitantCommand panicked with %v", p)
				}
			}()
			character.MustInfluenceHesitantCommand(c.approach, c.target, c.attempt, c.opts...)
		})
	}
	extra := map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(1), ability.Strength: modifier.New(5)}
	if _, err := character.NewInfluenceHesitantCommand(character.Persuade, target, character.NewInfluenceRoll(extra)); err != nil {
		t.Errorf("extra entries rejected: %v", err)
	}
}

// Requirement: doc/specs/social-interaction.md FR-025, FR-026, FR-034 - a roller that
// errors, panics, answers another expression or runs dry yields the dice port's
// *srderr.PortError, a nil result and one request, recorded once (error status, one error
// count with error.type, one duration, no resolved count, no log) and identical with the
// no-op bundle.
// Path: per case.
func TestResolveInfluenceRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dice.NewResult(d20Expr(2), []dice.DieRoll{dice.MustDieRoll(dice.D20(), 1), dice.MustDieRoll(dice.D20(), 1)})
	})
	cases := []struct {
		name, requirement, path string
		roller                  character.Roller
		wantIs                  error
	}{
		{"roller fails", "FR-025", "unhappy (failure of collaborator)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		{"roller panics", "FR-025", "unhappy (failure of collaborator)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }), srderr.ErrPortPanic},
		{"another expression", "FR-025", "unhappy (failure of collaborator)", wrong, srderr.ErrContractViolation},
		{"drained fixed roller", "FR-026", "unhappy (failure of collaborator)", dicetest.NewFixedRoller(), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := hesitant(t, character.Indifferent, chaRoll(0))
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), time.Second)
			res, err := newInfluencer(t, c.roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).ResolveInfluence(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" || (c.wantIs != nil && !errors.Is(err, c.wantIs)) {
				t.Fatalf("result = %#v, err = %v [%s]", res, err, c.requirement)
			}
			if plain, plainErr := newInfluencer(t, c.roller).ResolveInfluence(context.Background(), cmd); plain != nil || !errors.As(plainErr, &pe) {
				t.Errorf("no-op bundle: %#v, %v", plain, plainErr)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanResolveInfluence || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, map[string]any{character.AttrInfluenceApproach: "persuade", character.AttrInfluenceAttitude: "indifferent"})
			sameAttrs(t, "errors", onlyOne(t, rec, character.MetricInfluenceErrors).Attrs, map[string]any{telemetry.AttrErrorType: "dice_roller_error"})
			if n := len(rec.Meter.MeasurementsNamed(character.MetricInfluenceResolved)); n != 0 {
				t.Errorf("%d resolved increments on a port failure", n)
			}
			if d := onlyOne(t, rec, character.MetricInfluenceDuration); d.Value != 1 || len(d.Attrs) != 0 {
				t.Errorf("duration = %+v", d)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-025 - a roller that fails is asked once
// and the use case does not ask again.
// Path: unhappy (failure of collaborator).
func TestResolveInfluenceAsksTheRollerOnce(t *testing.T) {
	t.Parallel()
	calls := 0
	roller := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { calls++; return dice.Result{}, errors.New("down") })
	if res, err := newInfluencer(t, roller).ResolveInfluence(context.Background(), hesitant(t, character.Friendly, chaRoll(0))); res != nil || err == nil || calls != 1 {
		t.Errorf("%#v, %v, %d calls", res, err, calls)
	}
}

type foreignCommand struct{ character.InfluenceCommand }

// Requirement: doc/specs/social-interaction.md FR-034 - the constructor refuses a nil roller;
// ResolveInfluence refuses the zero use case and a nil, foreign or zero hesitant command with
// ErrInvalidInput, and a cancelled context with ctx.Err(), all before any signal or roll.
// Path: unhappy (invalid input).
func TestResolveInfluenceInvalidInputAndCancellation(t *testing.T) {
	t.Parallel()
	if _, err := character.NewInfluenceResolver(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v", err)
	}
	roller, rec := dicetest.NewFixedRoller(5), telemetrytest.NewRecorder()
	use := newInfluencer(t, roller, character.WithTelemetry(rec.Telemetry()))
	for name, run := range map[string]func() (character.ResolveInfluenceResult, error){
		"zero use case": func() (character.ResolveInfluenceResult, error) {
			return character.InfluenceResolver{}.ResolveInfluence(context.Background(), character.InfluenceWillingCommand{})
		},
		"nil command": func() (character.ResolveInfluenceResult, error) {
			return use.ResolveInfluence(context.Background(), nil)
		},
		"foreign command": func() (character.ResolveInfluenceResult, error) {
			return use.ResolveInfluence(context.Background(), foreignCommand{})
		},
		"zero hesitant command": func() (character.ResolveInfluenceResult, error) {
			return use.ResolveInfluence(context.Background(), character.InfluenceHesitantCommand{})
		},
	} {
		if res, err := run(); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v", name, res, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := use.ResolveInfluence(cancelled, hesitant(t, character.Friendly, chaRoll(0)))
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %#v, %v, spans %d, measurements %d, requests %d", res, err, len(rec.Tracer.Spans()), len(rec.Meter.Measurements()), len(roller.Requests()))
	}
}

// Requirement: doc/specs/social-interaction.md FR-027, FR-036 - equal commands and equal d20
// values give equal results, at all 1s and all 20s for each attitude, identical with the
// no-op bundle and a recording one.
// Path: unhappy (determinism).
func TestResolveInfluenceIsDeterministic(t *testing.T) {
	t.Parallel()
	for _, att := range []character.Attitude{character.Friendly, character.Indifferent, character.Hostile} {
		for _, face := range []int{1, 20} {
			cmd := hesitant(t, att, chaRoll(2))
			faces := []int{face, face}
			a, errA := newInfluencer(t, dicetest.NewFixedRoller(faces...)).ResolveInfluence(context.Background(), cmd)
			b, errB := newInfluencer(t, dicetest.NewFixedRoller(faces...), character.WithTelemetry(telemetrytest.NewRecorder().Telemetry())).ResolveInfluence(context.Background(), cmd)
			if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
				t.Errorf("%v/%d: %#v, %v vs %#v, %v", att, face, a, errA, b, errB)
			}
		}
	}
}

// Requirement: doc/specs/social-interaction.md FR-028 - InfluenceRoll copies the caller's
// slices and map on construction and its accessors return fresh copies; a call leaves the
// command unchanged.
// Path: unhappy (immutability).
func TestInfluenceRollCopiesItsCollections(t *testing.T) {
	t.Parallel()
	abilities := map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(3)}
	others := []modifier.Modifier{modifier.New(1)}
	circs := []modifier.RollCircumstance{modifier.Advantage{}}
	r := character.NewInfluenceRoll(abilities, character.WithOtherModifiers(others...), character.WithCircumstances(circs...))
	want := character.NewInfluenceRoll(map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(3)}, character.WithOtherModifiers(modifier.New(1)), character.WithCircumstances(modifier.Advantage{}))
	abilities[ability.Charisma], others[0], circs[0] = modifier.New(99), modifier.New(99), modifier.Disadvantage{}
	r.AbilityModifiers()[ability.Charisma] = modifier.New(77)
	r.OtherModifiers()[0] = modifier.New(77)
	r.Circumstances()[0] = nil
	if !r.Equal(want) {
		t.Errorf("roll changed: %#v", r)
	}
	cmd := hesitant(t, character.Friendly, r)
	before := hesitant(t, character.Friendly, want)
	if _, err := newInfluencer(t, dicetest.NewFixedRoller(5, 6)).ResolveInfluence(context.Background(), cmd); err != nil || !cmd.Equal(before) {
		t.Errorf("command changed or call failed: %v", err)
	}
}

// Requirement: doc/specs/social-interaction.md FR-029 - rolls, commands and check results
// are equal when every field is equal and unequal when any single field differs.
// Path: per case.
func TestInfluenceEquality(t *testing.T) {
	t.Parallel()
	roll := func(cha int, opts ...character.RollOption) character.InfluenceRoll { return chaRoll(cha, opts...) }
	cases := []struct {
		name, requirement, path string
		a, b                    character.InfluenceRoll
		equal                   bool
	}{
		{"same", "FR-029", "happy", roll(1), roll(1), true},
		{"ability modifier", "FR-029", "unhappy (any differing field)", roll(1), roll(2), false},
		{"extra ability entry", "FR-029", "unhappy (any differing field)", roll(1), character.NewInfluenceRoll(map[ability.Ability]modifier.Modifier{ability.Charisma: modifier.New(1), ability.Wisdom: modifier.New(0)}), false},
		{"proficiency present", "FR-029", "unhappy (any differing field)", roll(1), roll(1, character.WithProficiencyBonus(modifier.New(0))), false},
		{"proficiency value", "FR-029", "unhappy (any differing field)", roll(1, character.WithProficiencyBonus(modifier.New(2))), roll(1, character.WithProficiencyBonus(modifier.New(3))), false},
		{"others order", "FR-029", "unhappy (any differing field)", roll(1, character.WithOtherModifiers(modifier.New(1), modifier.New(2))), roll(1, character.WithOtherModifiers(modifier.New(2), modifier.New(1))), false},
		{"circumstances", "FR-029", "unhappy (any differing field)", roll(1, character.WithCircumstances(modifier.Advantage{})), roll(1, character.WithCircumstances(modifier.Disadvantage{})), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.a.Equal(c.b) != c.equal || c.b.Equal(c.a) != c.equal {
				t.Errorf("Equal = %v, want %v [%s, %s]", c.a.Equal(c.b), c.equal, c.requirement, c.path)
			}
		})
	}
	base := hesitant(t, character.Friendly, chaRoll(1), character.WithGMDC(12), character.WithGMRetryAfter(time.Hour))
	other := func(opts ...character.HesitantOption) character.InfluenceHesitantCommand {
		return hesitant(t, character.Friendly, chaRoll(1), opts...)
	}
	for name, c := range map[string]struct {
		cmd   character.InfluenceHesitantCommand
		equal bool
	}{
		"same":         {other(character.WithGMDC(12), character.WithGMRetryAfter(time.Hour)), true},
		"DC":           {other(character.WithGMDC(13), character.WithGMRetryAfter(time.Hour)), false},
		"no DC":        {other(character.WithGMRetryAfter(time.Hour)), false},
		"wait":         {other(character.WithGMDC(12), character.WithGMRetryAfter(time.Minute)), false},
		"no wait":      {other(character.WithGMDC(12)), false},
		"target":       {hesitant(t, character.Hostile, chaRoll(1), character.WithGMDC(12), character.WithGMRetryAfter(time.Hour)), false},
		"attempt kind": {hesitant(t, character.Friendly, character.InfluenceAutomaticFailure{}, character.WithGMDC(12), character.WithGMRetryAfter(time.Hour)), false},
		"approach": {must(character.NewInfluenceHesitantCommand(character.Intimidate, influenceTarget(t, 10, content.CreatureTypeHumanoid, character.Friendly), chaRoll(1),
			character.WithGMDC(12), character.WithGMRetryAfter(time.Hour))), false},
	} {
		if base.Equal(c.cmd) != c.equal {
			t.Errorf("command %s: Equal = %v, want %v", name, base.Equal(c.cmd), c.equal)
		}
	}
	forcedA, forcedB := hesitant(t, character.Friendly, character.InfluenceAutomaticFailure{}), hesitant(t, character.Friendly, character.InfluenceAutomaticFailure{})
	if !forcedA.Equal(forcedB) || forcedA.Equal(base) || !character.InfluenceHesitantCommand.Equal(character.InfluenceHesitantCommand{}, character.InfluenceHesitantCommand{}) {
		t.Error("forced or zero command equality wrong")
	}
	// check results
	run := func(att character.Attitude, face int, wait time.Duration) character.ResolveInfluenceResult {
		res, err := newInfluencer(t, dicetest.NewFixedRoller(face, face)).ResolveInfluence(context.Background(), hesitant(t, att, chaRoll(0), character.WithGMRetryAfter(wait)))
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	s1, s2 := run(character.Indifferent, 18, 0).(character.ResolveInfluenceCheckSucceeded), run(character.Indifferent, 18, 0).(character.ResolveInfluenceCheckSucceeded)
	s3, s4 := run(character.Friendly, 18, 0).(character.ResolveInfluenceCheckSucceeded), run(character.Indifferent, 19, 0).(character.ResolveInfluenceCheckSucceeded)
	f1, f2 := run(character.Indifferent, 3, time.Hour).(character.ResolveInfluenceCheckFailed), run(character.Indifferent, 3, time.Hour).(character.ResolveInfluenceCheckFailed)
	f3, f4 := run(character.Indifferent, 3, time.Minute).(character.ResolveInfluenceCheckFailed), run(character.Indifferent, 4, time.Hour).(character.ResolveInfluenceCheckFailed)
	if !s1.Equal(s2) || s1.Equal(s3) || s1.Equal(s4) || !f1.Equal(f2) || f1.Equal(f3) || f1.Equal(f4) {
		t.Error("check result equality wrong")
	}
}

// Requirement: doc/specs/social-interaction.md FR-031, FR-032, FR-033, FR-035, FR-043, FR-045
// - each call records one span, one resolved increment and one duration with the attributes
// of its response: hesitant calls add approach and attitude, rolled ones the roll mode,
// forced failures and non-check responses none; a failed check or refusal sets no error
// status; the mapping is closed; both failed-check variants are check_failed; nothing is
// logged; the no-op bundle gives the same result.
// Path: per case.
func TestResolveInfluenceSignals(t *testing.T) {
	t.Parallel()
	hes := map[string]any{character.AttrInfluenceApproach: "persuade", character.AttrInfluenceAttitude: "hostile"}
	with := func(extra map[string]any) map[string]any {
		m := map[string]any{}
		for k, v := range hes {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	cases := []struct {
		name, requirement, path string
		cmd                     character.InfluenceCommand
		faces                   []int
		spanAttrs, counterAttrs map[string]any
	}{
		{"fulfilled", "FR-032", "happy", character.InfluenceWillingCommand{}, nil,
			map[string]any{character.AttrInfluenceResponse: "fulfilled", telemetry.AttrOutcome: "success"},
			map[string]any{character.AttrInfluenceResponse: "fulfilled", telemetry.AttrOutcome: "success"}},
		{"refused", "FR-032", "happy", character.InfluenceUnwillingCommand{}, nil,
			map[string]any{character.AttrInfluenceResponse: "refused", telemetry.AttrOutcome: "failure"},
			map[string]any{character.AttrInfluenceResponse: "refused", telemetry.AttrOutcome: "failure"}},
		{"check succeeded", "FR-035", "happy", hesitant(t, character.Hostile, chaRoll(20)), []int{1, 5},
			with(map[string]any{character.AttrInfluenceResponse: "check_succeeded", telemetry.AttrOutcome: "success", d20.AttrRollMode: "disadvantage"}),
			with(map[string]any{character.AttrInfluenceResponse: "check_succeeded", telemetry.AttrOutcome: "success", d20.AttrRollMode: "disadvantage"})},
		{"check failed", "FR-035", "unhappy (boundary)", hesitant(t, character.Hostile, chaRoll(0, character.WithCircumstances(modifier.Advantage{}))), []int{3},
			with(map[string]any{character.AttrInfluenceResponse: "check_failed", telemetry.AttrOutcome: "failure", d20.AttrRollMode: "normal"}),
			with(map[string]any{character.AttrInfluenceResponse: "check_failed", telemetry.AttrOutcome: "failure", d20.AttrRollMode: "normal"})},
		{"forced failure has no roll mode", "FR-043", "unhappy (boundary: attribute presence)", hesitant(t, character.Hostile, character.InfluenceAutomaticFailure{}), nil,
			with(map[string]any{character.AttrInfluenceResponse: "check_failed", telemetry.AttrOutcome: "failure"}),
			with(map[string]any{character.AttrInfluenceResponse: "check_failed", telemetry.AttrOutcome: "failure"})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			res, err := newInfluencer(t, dicetest.NewFixedRoller(c.faces...), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).ResolveInfluence(context.Background(), c.cmd)
			plain, err2 := newInfluencer(t, dicetest.NewFixedRoller(c.faces...)).ResolveInfluence(context.Background(), c.cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("results differ: %#v, %v vs %#v, %v", res, err, plain, err2)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != "srd5e.character.resolve_influence" || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, c.spanAttrs)
			sameAttrs(t, "resolved", onlyOne(t, rec, character.MetricInfluenceResolved).Attrs, c.counterAttrs)
			if d := onlyOne(t, rec, character.MetricInfluenceDuration); d.Value != 0.25 || len(d.Attrs) != 0 {
				t.Errorf("duration = %+v", d)
			}
			if n := len(rec.Meter.MeasurementsNamed(character.MetricInfluenceErrors)); n != 0 {
				t.Errorf("%d error increments", n)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-033 - the duration is recorded while the
// span is open (the timer stops before span.End).
// Path: happy.
func TestResolveInfluenceStopsTheTimerBeforeEndingTheSpan(t *testing.T) {
	t.Parallel()
	var events []string
	rec := telemetrytest.NewRecorder()
	clock := func() time.Time { events = append(events, "clock"); return time.Unix(0, 0) }
	_, err := newInfluencer(t, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock)).ResolveInfluence(context.Background(), character.InfluenceWillingCommand{})
	if err != nil || !reflect.DeepEqual(events, []string{"clock", "clock"}) || !rec.Tracer.Spans()[0].Ended {
		t.Errorf("events %v, err %v", events, err)
	}
}

type panickingTracer struct{}

func (panickingTracer) Start(context.Context, string, ...telemetry.Attr) (context.Context, telemetry.Span) {
	panic(errors.New("backend down"))
}

// Requirement: doc/specs/social-interaction.md FR-034 - a panic in a telemetry backend is
// the host's bug and propagates.
// Path: unhappy (failure of collaborator).
func TestResolveInfluenceTelemetryPanicPropagates(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Error("the telemetry panic was swallowed")
		}
	}()
	u := newInfluencer(t, dicetest.NewFixedRoller(), character.WithTelemetry(telemetry.Telemetry{Tracer: panickingTracer{}}))
	_, _ = u.ResolveInfluence(context.Background(), character.InfluenceWillingCommand{})
}

// Requirement: doc/specs/social-interaction.md FR-037 - the signal names, attribute keys and
// values are pinned to their literal catalogue values.
// Path: unhappy (boundary: contract pinning).
func TestInfluenceSignalNamesArePinned(t *testing.T) {
	t.Parallel()
	for got, want := range map[string]string{
		character.SpanResolveInfluence:         "srd5e.character.resolve_influence",
		character.MetricInfluenceResolved:      "srd5e.character.influence_resolved",
		character.MetricInfluenceErrors:        "srd5e.character.influence_errors",
		character.MetricInfluenceDuration:      "srd5e.character.influence_duration",
		character.AttrInfluenceApproach:        "srd5e.influence.approach",
		character.AttrInfluenceAttitude:        "srd5e.influence.attitude",
		character.AttrInfluenceResponse:        "srd5e.influence.response",
		character.ValueInfluenceFulfilled:      "fulfilled",
		character.ValueInfluenceRefused:        "refused",
		character.ValueInfluenceCheckSucceeded: "check_succeeded",
		character.ValueInfluenceCheckFailed:    "check_failed",
		d20.AttrRollMode:                       "srd5e.d20_test.roll_mode",
		telemetry.AttrOutcome:                  "srd5e.outcome",
		telemetry.AttrErrorType:                "error.type",
	} {
		if got != want {
			t.Errorf("constant = %q, want %q", got, want)
		}
	}
}
