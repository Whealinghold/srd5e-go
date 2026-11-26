package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	cInc   = combat.FlightFallCauseIncapacitated
	cProne = combat.FlightFallCauseProne
	cZero  = combat.FlightFallCauseFlySpeedZero
)

func flightFacts(tb testing.TB, fly int, hover combat.HoverAbility, conds ...combat.Condition) combat.FlightFacts {
	tb.Helper()
	f, err := combat.NewFlightFacts(moveSet(tb, conds...), fly, hover)
	if err != nil {
		tb.Fatalf("NewFlightFacts: %v", err)
	}
	return f
}

// Requirement: per case
// Path: per case.
func TestAssessFlight(t *testing.T) {
	t.Parallel()
	const r = `doc/specs/movement-lifting-and-flight.md cases 4.5a; SRD 5.2.1, "Flying", p. 182; "Hover", p. 183`
	const rule, empty, edge = "unhappy (rule interaction)", "unhappy (empty case)", "unhappy (boundary)"
	none, hover := combat.HoverAbilityNone, combat.HoverAbilityHover
	grapple := combat.Grappled{Grappler: idA()}
	cases := []struct {
		name, path string
		f          combat.FlightFacts
		falls      []combat.FlightFallCause // the causes of a fall, or the overridden ones when stays
		stays      bool
	}{
		{"nothing wrong (FR-037)", "happy", flightFacts(t, 60, none), nil, true},
		{"Incapacitated (FR-038)", rule, flightFacts(t, 60, none, combat.Incapacitated{}), []combat.FlightFallCause{cInc}, false},
		{"Paralyzed implies Incapacitated and Speed 0", rule, flightFacts(t, 60, none, combat.Paralyzed{}), []combat.FlightFallCause{cInc, cZero}, false},
		{"Stunned implies Incapacitated, keeps its Speed", rule, flightFacts(t, 60, none, combat.Stunned{}), []combat.FlightFallCause{cInc}, false},
		{"Prone (FR-039)", rule, flightFacts(t, 60, none, combat.Prone{}), []combat.FlightFallCause{cProne}, false},
		{"Unconscious implies all three (FR-041)", rule, flightFacts(t, 60, none, combat.Unconscious{}), []combat.FlightFallCause{cInc, cProne, cZero}, false},
		{"Fly Speed 0 (FR-040)", edge, flightFacts(t, 0, none), []combat.FlightFallCause{cZero}, false},
		{"Fly Speed 1 (FR-040)", edge, flightFacts(t, 1, none), nil, true},
		{"Prone and Fly Speed 0", rule, flightFacts(t, 0, none, combat.Prone{}), []combat.FlightFallCause{cProne, cZero}, false},
		{"all three", rule, flightFacts(t, 0, none, combat.Incapacitated{}, combat.Prone{}), []combat.FlightFallCause{cInc, cProne, cZero}, false},
		{"Grappled has Speed 0", rule, flightFacts(t, 60, none, grapple), []combat.FlightFallCause{cZero}, false},
		{"Restrained has Speed 0", rule, flightFacts(t, 60, none, combat.Restrained{}), []combat.FlightFallCause{cZero}, false},
		{"Exhaustion takes Fly Speed to 0", edge, flightFacts(t, 15, none, condLevel(4)), []combat.FlightFallCause{cZero}, false},
		{"Exhaustion leaves some Fly Speed", edge, flightFacts(t, 10, none, condLevel(1)), nil, true},
		{"Grappled with Exhaustion", rule, flightFacts(t, 30, none, grapple, condLevel(2)), []combat.FlightFallCause{cZero}, false},
		{"Grappled, hover (FR-042)", rule, flightFacts(t, 60, hover, grapple), []combat.FlightFallCause{cZero}, true},
		{"hover, Incapacitated", rule, flightFacts(t, 60, hover, combat.Incapacitated{}), []combat.FlightFallCause{cInc}, true},
		{"hover, Prone and Fly Speed 0", rule, flightFacts(t, 0, hover, combat.Prone{}), []combat.FlightFallCause{cProne, cZero}, true},
		{"hover, nothing wrong (FR-043)", empty, flightFacts(t, 60, hover), nil, true},
		{"the largest Fly Speed", edge, flightFacts(t, math.MaxInt, none), nil, true},
		{"the zero facts are a Fly Speed of 0", edge, combat.FlightFacts{}, []combat.FlightFallCause{cZero}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.AssessFlight(c.f)
			var causes []combat.FlightFallCause
			var stays bool
			switch v := got.(type) {
			case combat.FlightStaysAloft:
				causes, stays = v.OverriddenCauses.Slice(), true
			case combat.FlightFalls:
				causes = v.Causes.Slice()
			}
			if stays != c.stays || !slices.Equal(causes, c.falls) {
				t.Errorf("AssessFlight = %#v, want stays %v with causes %v [%s, %s]", got, c.stays, c.falls, r, c.path)
			}
			if again := combat.AssessFlight(c.f); again != got {
				t.Errorf("a second call gave %#v, want %#v (FR-045, FR-047)", again, got)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-036 - a negative base Fly Speed and an invalid hover ability are invalid-input errors.
// Path: unhappy (invalid input).
func TestNewFlightFactsRejectsInvalid(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		fly   int
		hover combat.HoverAbility
	}{"negative": {-1, 0}, "most negative": {math.MinInt, 0}, "hover one past": {10, combat.HoverAbilityHover + 1}, "hover negative": {10, -1}} {
		if got, err := combat.NewFlightFacts(moveSet(t), c.fly, c.hover); !errors.Is(err, srderr.ErrInvalidInput) || got != (combat.FlightFacts{}) {
			t.Errorf("%s: = %#v, %v; want zero facts and invalid input", name, got, err)
		}
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-041, FR-046 - FlightFallCauses is a comparable set listing each cause once in the fixed order,
// Slice returns a copy, an invalid cause is never in a set, and the cause names are fixed.
// Path: unhappy (immutability).
func TestFlightFallCauses(t *testing.T) {
	t.Parallel()
	falls, ok := combat.AssessFlight(flightFacts(t, 0, combat.HoverAbilityNone, combat.Unconscious{})).(combat.FlightFalls)
	if !ok {
		t.Fatal("an Unconscious flier did not fall")
	}
	first := falls.Causes.Slice()
	first[0] = cZero
	if got := falls.Causes.Slice(); !slices.Equal(got, []combat.FlightFallCause{cInc, cProne, cZero}) {
		t.Errorf("Slice after mutating a copy = %v", got)
	}
	if falls.Causes.Has(0) || falls.Causes.Has(cZero+1) || !falls.Causes.Has(cProne) || (combat.FlightFallCauses{}).Has(cInc) {
		t.Error("Has is wrong for an invalid, present or absent cause")
	}
	if cInc.String() != "incapacitated" || cProne.String() != "prone" || cZero.String() != "fly_speed_zero" || combat.FlightFallCause(0).String() != "unknown" || combat.FlightFallCause(0).Valid() {
		t.Error("cause names are wrong")
	}
	if !combat.HoverAbilityNone.Valid() || !combat.HoverAbilityHover.Valid() || combat.HoverAbility(2).Valid() {
		t.Error("HoverAbility.Valid is wrong")
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-036, FR-040 - the effective Fly Speed is the base after the conditions, applied on its own:
// Exhaustion level 1 takes 5 feet off, so a base of 5 falls and a base of 6 does not.
// Path: unhappy (boundary).
func TestAssessFlightUsesTheEffectiveFlySpeed(t *testing.T) {
	t.Parallel()
	if _, fell := combat.AssessFlight(flightFacts(t, 5, combat.HoverAbilityNone, condLevel(1))).(combat.FlightFalls); !fell {
		t.Error("base 5 less 5 for Exhaustion is 0 and should fall")
	}
	if _, aloft := combat.AssessFlight(flightFacts(t, 6, combat.HoverAbilityNone, condLevel(1))).(combat.FlightStaysAloft); !aloft {
		t.Error("base 6 less 5 for Exhaustion is 1 and should stay aloft")
	}
}
