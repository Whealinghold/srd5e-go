package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	longJump  = combat.JumpKindLong
	highJump  = combat.JumpKindHigh
	walkMode  = combat.MovementModeWalk
	runningAp = combat.JumpApproachRunning
	standing  = combat.JumpApproachStanding
)

// Requirement: doc/specs/movement-lifting-and-flight.md FR-007 - JumpKind has exactly two values in the order Long, High; the zero
// value and one past the last are not kinds (SRD 5.2.1, "Jumping", p. 184).
// Path: happy.
func TestJumpKindValues(t *testing.T) {
	t.Parallel()
	if longJump != 1 || highJump != 2 || longJump.String() != "long" || highJump.String() != "high" {
		t.Errorf("kinds = %d %q, %d %q", longJump, longJump, highJump, highJump)
	}
	for _, k := range []combat.JumpKind{0, highJump + 1, -1} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("kind %d: Valid %v, String %q; want invalid, unknown", k, k.Valid(), k)
		}
	}
	for _, a := range []combat.JumpApproach{0, standing + 1} {
		if a.Valid() || a.String() != "unknown" {
			t.Errorf("approach %d: Valid %v, String %q; want invalid, unknown", a, a.Valid(), a)
		}
	}
	if !runningAp.Valid() || !standing.Valid() || runningAp.String() != "running" || standing.String() != "standing" {
		t.Error("the two approaches are not valid or misnamed")
	}
}

// Requirement: per case
// Path: per case.
func TestJumpApproachFor(t *testing.T) {
	t.Parallel()
	const long, high = `SRD 5.2.1, "Long Jump", pp. 184-185 (FR-008)`, `SRD 5.2.1, "High Jump", p. 183 (FR-009)`
	cases := []struct {
		name, requirement, path string
		kind                    combat.JumpKind
		feet                    int
		mode                    combat.MovementMode
		want                    combat.JumpApproach
	}{
		{"Long Jump, exactly 10", long, "unhappy (boundary)", longJump, 10, walkMode, runningAp},
		{"Long Jump, one short", long, "unhappy (boundary)", longJump, 9, walkMode, standing},
		{"Long Jump, one more", long, "happy", longJump, 11, walkMode, runningAp},
		{"Long Jump, no run-up", long, "unhappy (empty case)", longJump, 0, walkMode, standing},
		{"Long Jump, any mode counts", long, "unhappy (rule interaction)", longJump, 10, combat.MovementModeFly, runningAp},
		{"High Jump, exactly 10 on foot", high, "unhappy (boundary)", highJump, 10, walkMode, runningAp},
		{"High Jump, one short", high, "unhappy (boundary)", highJump, 9, walkMode, standing},
		{"High Jump, flew the 10 feet", high, "unhappy (rule interaction)", highJump, 10, combat.MovementModeFly, standing},
		{"High Jump, crawled the 10 feet", high, "unhappy (rule interaction)", highJump, 10, combat.MovementModeCrawl, standing},
		{"High Jump, climbed", high, "unhappy (rule interaction)", highJump, 15, combat.MovementModeClimb, standing},
		{"High Jump, the largest run-up on foot", high, "unhappy (boundary)", highJump, math.MaxInt, walkMode, runningAp},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.JumpApproachFor(c.kind, c.feet, c.mode)
			if err != nil || got != c.want {
				t.Errorf("JumpApproachFor(%v, %d, %v) = %v, %v; want %v [%s]", c.kind, c.feet, c.mode, got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-010 - a negative run-up, an invalid kind and an invalid mode are
// invalid-input errors, and a run-up of 0 is accepted.
// Path: unhappy (invalid input).
func TestJumpApproachForRejectsInvalid(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		kind combat.JumpKind
		feet int
		mode combat.MovementMode
	}{
		"negative run-up": {highJump, -1, walkMode}, "most negative run-up": {longJump, math.MinInt, walkMode},
		"zero kind": {0, 10, walkMode}, "zero mode": {longJump, 10, 0},
	} {
		if _, err := combat.JumpApproachFor(c.kind, c.feet, c.mode); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want invalid input", name, err)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestJumpMaximumFeet(t *testing.T) {
	t.Parallel()
	const r = `doc/specs/movement-lifting-and-flight.md cases 4.2a (FR-011 to FR-014); SRD 5.2.1, "Jumping", pp. 183-185`
	cases := []struct {
		strength                         int
		longRun, longStand, highRun, hiS int
		path                             string
	}{
		{1, 1, 0, 0, 0, "unhappy (boundary)"}, {4, 4, 2, 0, 0, "unhappy (boundary)"}, {6, 6, 3, 1, 0, "unhappy (boundary)"},
		{8, 8, 4, 2, 1, "unhappy (boundary)"}, {10, 10, 5, 3, 1, "happy"}, {11, 11, 5, 3, 1, "unhappy (boundary)"},
		{12, 12, 6, 4, 2, "happy"}, {15, 15, 7, 5, 2, "unhappy (boundary)"}, {20, 20, 10, 8, 4, "happy"},
		{30, 30, 15, 13, 6, "unhappy (boundary)"},
	}
	for _, c := range cases {
		s := moveScore(t, c.strength)
		for _, w := range []struct {
			kind combat.JumpKind
			ap   combat.JumpApproach
			want int
		}{{longJump, runningAp, c.longRun}, {longJump, standing, c.longStand}, {highJump, runningAp, c.highRun}, {highJump, standing, c.hiS}} {
			if got := combat.JumpMaximumFeet(w.kind, s, w.ap); got != w.want {
				t.Errorf("JumpMaximumFeet(%v, Strength %d, %v) = %d, want %d [%s, %s]", w.kind, c.strength, w.ap, got, w.want, r, c.path)
			}
		}
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-007 - an invalid kind or approach, and the zero Strength score, give a maximum of 0.
// Path: unhappy (invalid input).
func TestJumpMaximumFeetOfInvalidInputsIsZero(t *testing.T) {
	t.Parallel()
	s := moveScore(t, 10)
	for name, got := range map[string]int{
		"zero kind": combat.JumpMaximumFeet(0, s, runningAp), "zero approach": combat.JumpMaximumFeet(longJump, s, 0),
		"zero score": combat.JumpMaximumFeet(longJump, ability.Score{}, runningAp),
	} {
		if got != 0 {
			t.Errorf("%s: maximum = %d, want 0", name, got)
		}
	}
}

func attempt(tb testing.TB, kind combat.JumpKind, strength, runUp int, mode combat.MovementMode, requested, left int, conds ...combat.Condition) combat.JumpAttempt {
	tb.Helper()
	a, err := combat.NewJumpAttempt(kind, moveScore(tb, strength), moveSet(tb, conds...), runUp, mode, requested, left)
	if err != nil {
		tb.Fatalf("NewJumpAttempt: %v", err)
	}
	return a
}

// Requirement: per case
// Path: per case.
func TestAssessJump(t *testing.T) {
	t.Parallel()
	const r = `doc/specs/movement-lifting-and-flight.md cases 4.2c`
	permitted := func(k combat.JumpKind, ap combat.JumpApproach, max, d int) combat.JumpAssessment {
		return combat.JumpPermitted{Kind: k, Approach: ap, MaximumFeet: max, DistanceFeet: d, MovementCostFeet: d}
	}
	refused := func(k combat.JumpKind, ap combat.JumpApproach, why combat.JumpRefusal, max, req, left int) combat.JumpAssessment {
		return combat.JumpRefused{Kind: k, Approach: ap, Reason: why, MaximumFeet: max, RequestedFeet: req, MovementLeftFeet: left}
	}
	const prone, exceeds, short = combat.JumpRefusalProneRestrictedMovement, combat.JumpRefusalExceedsMaximumDistance, combat.JumpRefusalInsufficientMovement
	cases := []struct {
		name, path string
		a          combat.JumpAttempt
		want       combat.JumpAssessment
	}{
		{"at the maximum (FR-016)", "unhappy (boundary)", attempt(t, longJump, 10, 10, walkMode, 10, 30), permitted(longJump, runningAp, 10, 10)},
		{"one over the maximum (FR-016)", "unhappy (boundary)", attempt(t, longJump, 10, 10, walkMode, 11, 30), refused(longJump, runningAp, exceeds, 10, 11, 30)},
		{"shorter than the maximum (FR-015, FR-021)", "happy", attempt(t, longJump, 10, 10, walkMode, 4, 30), permitted(longJump, runningAp, 10, 4)},
		{"standing max (FR-012)", "unhappy (boundary)", attempt(t, longJump, 10, 0, walkMode, 5, 30), permitted(longJump, standing, 5, 5)},
		{"standing one over (FR-016)", "unhappy (boundary)", attempt(t, longJump, 10, 0, walkMode, 6, 30), refused(longJump, standing, exceeds, 5, 6, 30)},
		{"High Jump at the maximum (FR-015)", "unhappy (boundary)", attempt(t, highJump, 10, 10, walkMode, 3, 30), permitted(highJump, runningAp, 3, 3)},
		{"High Jump one over (FR-016)", "unhappy (boundary)", attempt(t, highJump, 10, 10, walkMode, 4, 30), refused(highJump, runningAp, exceeds, 3, 4, 30)},
		{"High Jump standing cost equals height (FR-021)", "happy", attempt(t, highJump, 10, 0, walkMode, 1, 30), permitted(highJump, standing, 1, 1)},
		{"movement exactly enough (FR-017)", "unhappy (boundary)", attempt(t, longJump, 10, 10, walkMode, 10, 10), permitted(longJump, runningAp, 10, 10)},
		{"movement one short (FR-017)", "unhappy (boundary)", attempt(t, longJump, 10, 10, walkMode, 10, 9), refused(longJump, runningAp, short, 10, 10, 9)},
		{"both limits exceeded (FR-018)", "unhappy (rule interaction)", attempt(t, longJump, 10, 10, walkMode, 11, 9), refused(longJump, runningAp, exceeds, 10, 11, 9)},
		{"zero distance, no movement (FR-019)", "unhappy (boundary)", attempt(t, longJump, 10, 10, walkMode, 0, 0), permitted(longJump, runningAp, 10, 0)},
		{"zero maximum, zero height (FR-019)", "unhappy (boundary)", attempt(t, highJump, 1, 10, walkMode, 0, 30), permitted(highJump, runningAp, 0, 0)},
		{"zero maximum, any height (FR-016)", "unhappy (boundary)", attempt(t, highJump, 1, 10, walkMode, 1, 30), refused(highJump, runningAp, exceeds, 0, 1, 30)},
		{"negative movement left, any request (FR-020)", "unhappy (state error)", attempt(t, longJump, 10, 10, walkMode, 5, -5), refused(longJump, runningAp, short, 10, 5, 0)},
		{"negative movement left, no jump (FR-019, FR-020)", "unhappy (state error)", attempt(t, longJump, 10, 10, walkMode, 0, -5), permitted(longJump, runningAp, 10, 0)},
		{"most negative movement left (FR-020)", "unhappy (state error)", attempt(t, longJump, 10, 10, walkMode, 1, math.MinInt), refused(longJump, runningAp, short, 10, 1, 0)},
		{"Prone (FR-056)", "unhappy (rule interaction)", attempt(t, longJump, 10, 10, walkMode, 5, 30, combat.Prone{}), refused(longJump, runningAp, prone, 10, 5, 30)},
		{"Unconscious implies Prone (FR-056)", "unhappy (rule interaction)", attempt(t, highJump, 10, 10, walkMode, 1, 30, combat.Unconscious{}), refused(highJump, runningAp, prone, 3, 1, 30)},
		{"Prone beats the other reasons (FR-018)", "unhappy (rule interaction)", attempt(t, longJump, 10, 10, walkMode, 11, 9, combat.Prone{}), refused(longJump, runningAp, prone, 10, 11, 9)},
		{"Prone, zero distance (FR-019)", "unhappy (boundary)", attempt(t, longJump, 10, 10, walkMode, 0, 30, combat.Prone{}), permitted(longJump, runningAp, 10, 0)},
		{"Strength 30 asks for the most (overflow limit)", "unhappy (boundary)", attempt(t, longJump, 30, 10, walkMode, math.MaxInt, math.MaxInt), refused(longJump, runningAp, exceeds, 30, math.MaxInt, math.MaxInt)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.AssessJump(c.a); got != c.want {
				t.Errorf("AssessJump = %#v, want %#v [%s, %s]", got, c.want, r, c.path)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-022 - equal attempts give equal results on every call and the jump rolls nothing.
// Path: unhappy (determinism).
func TestAssessJumpIsDeterministic(t *testing.T) {
	t.Parallel()
	a := attempt(t, longJump, 10, 10, walkMode, 10, 30)
	if first, second := combat.AssessJump(a), combat.AssessJump(a); first != second {
		t.Errorf("two calls gave %#v and %#v", first, second)
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-010, FR-020 - NewJumpAttempt rejects a negative run-up or distance, an
// invalid kind or mode and the zero Strength score; the zero attempt is invalid and is refused with the zero reason.
// Path: unhappy (invalid input).
func TestNewJumpAttemptRejectsInvalid(t *testing.T) {
	t.Parallel()
	s, none := moveScore(t, 10), moveSet(t)
	for name, build := range map[string]func() (combat.JumpAttempt, error){
		"negative request": func() (combat.JumpAttempt, error) {
			return combat.NewJumpAttempt(longJump, s, none, 10, walkMode, -1, 30)
		},
		"negative run-up": func() (combat.JumpAttempt, error) {
			return combat.NewJumpAttempt(longJump, s, none, -1, walkMode, 1, 30)
		},
		"zero kind": func() (combat.JumpAttempt, error) { return combat.NewJumpAttempt(0, s, none, 10, walkMode, 1, 30) },
		"zero mode": func() (combat.JumpAttempt, error) { return combat.NewJumpAttempt(longJump, s, none, 10, 0, 1, 30) },
		"zero Strength": func() (combat.JumpAttempt, error) {
			return combat.NewJumpAttempt(longJump, ability.Score{}, none, 10, walkMode, 1, 30)
		},
		"most negative ask": func() (combat.JumpAttempt, error) {
			return combat.NewJumpAttempt(longJump, s, none, 10, walkMode, math.MinInt, 30)
		},
	} {
		if got, err := build(); !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
			t.Errorf("%s: = %#v, %v; want the zero attempt and invalid input", name, got, err)
		}
	}
	var zero combat.JumpAttempt
	if !zero.IsZero() || attempt(t, longJump, 10, 10, walkMode, 1, 30).IsZero() {
		t.Error("IsZero is wrong for the zero or a built attempt")
	}
	if got := combat.AssessJump(zero); got != (combat.JumpRefused{}) {
		t.Errorf("AssessJump(zero) = %#v, want the zero JumpRefused", got)
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-018 - the refusals are numbered in the order of precedence and the zero value is invalid.
// Path: unhappy (boundary).
func TestJumpRefusalValues(t *testing.T) {
	t.Parallel()
	want := []string{"prone_restricted_movement", "exceeds_maximum_distance", "insufficient_movement"}
	for i, w := range want {
		r := combat.JumpRefusal(i + 1)
		if !r.Valid() || r.String() != w {
			t.Errorf("refusal %d = %q valid %v, want %q", i+1, r, r.Valid(), w)
		}
	}
	for _, r := range []combat.JumpRefusal{0, 4} {
		if r.Valid() || r.String() != "unknown" {
			t.Errorf("refusal %d: Valid %v, String %q", r, r.Valid(), r)
		}
	}
}
