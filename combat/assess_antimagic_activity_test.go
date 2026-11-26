package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// aaaVerdict names a verdict: permits, or blocked:<reason>.
func aaaVerdict(res combat.AssessAntimagicActivityResult) string {
	switch v := res.(combat.AntimagicActivityAssessed).Verdict().(type) {
	case combat.AntimagicPermits:
		return "permits"
	case combat.AntimagicBlocks:
		return "blocked:" + v.Block().String()
	default:
		return "unknown"
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-045 to FR-048 and table T-B - the use case returns the verdict of AntimagicField.Blocks for each activity, one row per case (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: per case.
func TestAssessAntimagicActivityTableT_B(t *testing.T) {
	t.Parallel()
	goblin := []combat.SpellTargetRef{dspTargetRef(t, "goblin")}
	cases := []struct {
		name, requirement, path string
		activity                combat.AntimagicActivity
		actor                   bool
		targets                 []combat.SpellTargetRef
		want                    string
	}{
		{"cast from inside", "FR-045", "happy", combat.ActivityCastSpell, true, nil, "blocked:actor_inside_aura"},
		{"cast at a target inside", "FR-045", "happy", combat.ActivityCastSpell, false, goblin, "blocked:target_inside_aura"},
		{"both inside, the actor first", "FR-045", "unhappy (rule interaction)", combat.ActivityCastSpell, true, goblin, "blocked:actor_inside_aura"},
		{"cast, nothing inside", "FR-045", "happy", combat.ActivityCastSpell, false, nil, "permits"},
		{"Magic action inside", "FR-045", "happy", combat.ActivityTakeMagicAction, true, nil, "blocked:actor_inside_aura"},
		{"item property on something inside", "FR-046", "unhappy (rule interaction)", combat.ActivityUseMagicItemProperty, true, goblin, "blocked:actor_inside_aura"},
		{"item outside, target inside", "FR-046", "unhappy (boundary)", combat.ActivityUseMagicItemProperty, false, goblin, "permits"},
		{"teleport out of the aura", "FR-047", "happy", combat.ActivityTeleport, true, nil, "blocked:actor_inside_aura"},
		{"a Counterspell cast inside the aura is blocked like any cast", "FR-048", "unhappy (rule interaction)", combat.ActivityCastSpell, true, goblin, "blocked:actor_inside_aura"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := combat.NewAssessAntimagicActivity().Run(context.Background(), combat.AssessAntimagicActivityCommand{Activity: c.activity, ActorInside: c.actor, TargetsInside: c.targets})
			if err != nil {
				t.Fatalf("Run error = %v [%s]", err, c.requirement)
			}
			if got := aaaVerdict(res); got != c.want {
				t.Errorf("verdict = %s, want %s [%s%s]", got, c.want, dispelSpec, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-045, testing.md section 4 (invalid input, immutability) - an activity that is not defined and a target named twice are errors matching srderr.ErrInvalidInput recorded on the span only (no operations sample); the command's target list is not changed and the blocked targets returned are a copy.
// Path: unhappy (invalid input; immutability).
func TestAssessAntimagicActivityInvalidInput(t *testing.T) {
	t.Parallel()
	goblin := dspTargetRef(t, "goblin")
	cases := []struct {
		name string
		cmd  combat.AssessAntimagicActivityCommand
	}{
		{"no activity", combat.AssessAntimagicActivityCommand{}},
		{"an activity beyond the last", combat.AssessAntimagicActivityCommand{Activity: combat.ActivityTeleport + 1}},
		{"duplicate target ids", combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell, TargetsInside: []combat.SpellTargetRef{goblin, goblin}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := combat.NewAssessAntimagicActivity(opts...).Run(context.Background(), c.cmd)
			if !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
				t.Fatalf("Run = %v, %v, want ErrInvalidInput [%sFR-045]", res, err, dispelSpec)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.assess_antimagic_activity")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 {
				t.Errorf("spans = %+v, want one error span and no operations sample", spans)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
	targets := []combat.SpellTargetRef{goblin}
	res, err := combat.NewAssessAntimagicActivity().Run(context.Background(), combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell, TargetsInside: targets})
	if err != nil {
		t.Fatal(err)
	}
	blocked := res.(combat.AntimagicActivityAssessed).Verdict().(combat.AntimagicBlocks).Targets()
	blocked[0] = nil
	if targets[0] != goblin || res.(combat.AntimagicActivityAssessed).Verdict().(combat.AntimagicBlocks).Targets()[0] != goblin {
		t.Error("the command's targets or the verdict's targets were changed through a returned slice")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md testing.md section 4 (cancellation) - a cancelled context is returned unwrapped before any signal.
// Path: unhappy (cancellation).
func TestAssessAntimagicActivityCancelledContext(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := combat.NewAssessAntimagicActivity(opts...).Run(ctx, combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell})
	if !errors.Is(err, context.Canceled) || res != nil || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Errorf("Run = %v, %v with %d spans, want context.Canceled and no signal", res, err, len(rec.Tracer.Spans()))
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md section 7 and FR-056 - the shared signals under assess_antimagic_activity: completed (AntimagicPermits, applied, outcome success) or antimagic_blocked (AntimagicBlocks, refused, outcome failure, no error status), no extra attribute, no event, no log, one duration sample; the result is the same with the zero-value use case (Noop telemetry) and with the recording double.
// Path: per case.
func TestAssessAntimagicActivitySignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		actorInside             bool
		result, class, outcome  string
	}{
		{"permitted", "FR-056", "happy", false, "completed", "applied", "success"},
		{"blocked", "FR-056", "unhappy (rule interaction)", true, "antimagic_blocked", "refused", "failure"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell, ActorInside: c.actorInside}
			rec, opts := recording()
			got, err := combat.NewAssessAntimagicActivity(opts...).Run(context.Background(), cmd)
			plain, perr := combat.AssessAntimagicActivity{}.Run(context.Background(), cmd)
			if err != nil || perr != nil || fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", plain) {
				t.Fatalf("recording and Noop telemetry gave %#v, %v and %#v, %v", got, err, plain, perr)
			}
			want := map[string]any{"srd5e.combat.operation": "assess_antimagic_activity", "srd5e.outcome": c.outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat.assess_antimagic_activity")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 || !mapsEqual(attrMap(spans[0].Attrs), want) {
				t.Fatalf("spans = %+v, want one ended span with %v [%s%s]", spans, want, dispelSpec, c.requirement)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v", ops, want)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.events")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("events/errors samples = %d, want 0", n)
			}
			if d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(d) != 1 {
				t.Errorf("duration samples = %d, want one", len(d))
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-049, FR-043 - the use case holds no state: asking again for the same activity gives the same verdict, so a creature that leaves the aura (the host passes false) is permitted again.
// Path: unhappy (state error).
func TestAssessAntimagicActivityHoldsNoState(t *testing.T) {
	t.Parallel()
	use := combat.NewAssessAntimagicActivity()
	var verdicts []string
	for _, inside := range []bool{true, false, true} {
		res, err := use.Run(context.Background(), combat.AssessAntimagicActivityCommand{Activity: combat.ActivityTakeMagicAction, ActorInside: inside})
		if err != nil {
			t.Fatal(err)
		}
		verdicts = append(verdicts, aaaVerdict(res))
	}
	if want := []string{"blocked:actor_inside_aura", "permits", "blocked:actor_inside_aura"}; !slices.Equal(verdicts, want) {
		t.Errorf("verdicts = %v, want %v [%sFR-049]", verdicts, want, dispelSpec)
	}
}

// ExampleAssessAntimagicActivity asks whether a creature inside an Antimagic Field can cast a spell.
func ExampleAssessAntimagicActivity() {
	res, _ := combat.NewAssessAntimagicActivity().Run(context.Background(), combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell, ActorInside: true})
	_, blocked := res.(combat.AntimagicActivityAssessed).Verdict().(combat.AntimagicBlocks)
	fmt.Println(blocked)
	// Output: true
}
