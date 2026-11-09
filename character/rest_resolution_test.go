package character_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	rIn   = character.RollingInitiative
	rCast = character.CastingSpellOtherThanCantrip
	rDmg  = character.TakingDamage
	rEx   = character.PhysicalExertion
)

func ran(h, n int) character.RestEnd { return character.NewRestRanItsCourse(h, n) }
func stopped(i character.RestInterruption, h, n int) character.RestEnd {
	return character.NewRestStopped(i, h, n)
}

// Requirement: per case.
// Path: per case.
func TestResolveRestInterruptions(t *testing.T) {
	t.Parallel()
	pages := `SRD 5.2.1, "Long Rest", p. 185; "Short Rest", p. 187`
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		variant                 character.LongRestVariant
		end                     character.RestEnd
		want                    character.RestResolution
	}{
		{"SI-1 short rest stopped by Rolling Initiative", "FR-018 " + pages, "unhappy (state error)", character.ShortRest, character.StandardLongRest, stopped(rIn, 0, 0), character.RestStoppedWithoutBenefits{}},
		{"SI-2 short rest stopped by casting a spell", "FR-018", "unhappy (state error)", character.ShortRest, character.StandardLongRest, stopped(rCast, 0, 0), character.RestStoppedWithoutBenefits{}},
		{"SI-3 short rest stopped by damage", "FR-018", "unhappy (state error)", character.ShortRest, character.StandardLongRest, stopped(rDmg, 0, 0), character.RestStoppedWithoutBenefits{}},
		{"SI-4 short rest stopped by exertion", "FR-019", "unhappy (state error)", character.ShortRest, character.StandardLongRest, stopped(rEx, 0, 0), character.RestStoppedWithoutBenefits{}},
		{"SI-5 short rest stopped by exertion after an hour", "FR-019", "unhappy (boundary)", character.ShortRest, character.StandardLongRest, stopped(rEx, 1, 0), character.RestStoppedWithoutBenefits{}},
		{"LI-1 long rest, Rolling Initiative", "FR-020 " + pages, "happy", character.LongRest, character.StandardLongRest, stopped(rIn, 3, 0), character.RestStoppedWithShortRestBenefits{}},
		{"LI-2 long rest, casting a spell", "FR-022", "happy", character.LongRest, character.StandardLongRest, stopped(rCast, 3, 0), character.RestStoppedWithShortRestBenefits{}},
		{"LI-3 long rest, damage", "FR-022", "happy", character.LongRest, character.StandardLongRest, stopped(rDmg, 3, 0), character.RestStoppedWithShortRestBenefits{}},
		{"LI-4 long rest, exertion", "FR-022", "happy", character.LongRest, character.StandardLongRest, stopped(rEx, 3, 0), character.RestStoppedWithShortRestBenefits{}},
		{"LI-5 long rest, one hour rested", "FR-021", "unhappy (boundary)", character.LongRest, character.StandardLongRest, stopped(rDmg, 1, 0), character.RestStoppedWithShortRestBenefits{}},
		{"LI-6 long rest, no hours rested", "FR-021", "unhappy (boundary)", character.LongRest, character.StandardLongRest, stopped(rDmg, 0, 0), character.RestStoppedWithoutBenefits{}},
		{"LI-7 long rest, one hour short", "FR-020", "unhappy (boundary: one hour short)", character.LongRest, character.StandardLongRest, stopped(rIn, 7, 0), character.RestStoppedWithShortRestBenefits{}},
		{"LI-8 long rest, stop, resume, stop again", "FR-067", "unhappy (boundary: stop, resume, stop again)", character.LongRest, character.StandardLongRest, stopped(rDmg, 5, 1), character.RestStoppedWithShortRestBenefits{}},
		{"LI-9 long rest, resumed with no hours", "FR-021", "unhappy (boundary: no hours in any segment)", character.LongRest, character.StandardLongRest, stopped(rIn, 0, 1), character.RestStoppedWithoutBenefits{}},
		{"LI-11 long rest stopped at 8 hours", "FR-071", "unhappy (boundary: required hours reached)", character.LongRest, character.StandardLongRest, stopped(rIn, 8, 0), character.RestCompleted{}},
		{"LI-12 long rest stopped at 9 hours", "FR-071", "unhappy (boundary)", character.LongRest, character.StandardLongRest, stopped(rIn, 9, 0), character.RestCompleted{}},
		{"LI-13 trance long rest stopped at 4 hours", "FR-071", "unhappy (boundary)", character.LongRest, character.TranceLongRest, stopped(rDmg, 4, 0), character.RestCompleted{}},
		{"LI-14 resumed once, stopped at 8 hours", "FR-016", "unhappy (boundary: resumed n plus 8, one short)", character.LongRest, character.StandardLongRest, stopped(rDmg, 8, 1), character.RestStoppedWithShortRestBenefits{}},
		{"LI-15 resumed once, stopped at 9 hours", "FR-071", "unhappy (boundary: resumed n plus 8)", character.LongRest, character.StandardLongRest, stopped(rDmg, 9, 1), character.RestCompleted{}},
		{"short rest stopped after the hour still has no benefits", "FR-071", "unhappy (state error)", character.ShortRest, character.StandardLongRest, stopped(rDmg, 2, 0), character.RestStoppedWithoutBenefits{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rep := character.MustRestReport(c.kind, c.variant, c.end)
			got, err := character.ResolveRest(rep)
			if err != nil {
				t.Fatal(err)
			}
			switch want := c.want.(type) {
			case character.RestStoppedWithShortRestBenefits:
				g, ok := got.(character.RestStoppedWithShortRestBenefits)
				if !ok || g.Interruption() != c.end.(character.RestStopped).Interruption() {
					t.Errorf("got %#v, want short rest benefits [%s]", got, c.requirement)
				}
			case character.RestStoppedWithoutBenefits:
				g, ok := got.(character.RestStoppedWithoutBenefits)
				if !ok || g.Interruption() != c.end.(character.RestStopped).Interruption() {
					t.Errorf("got %#v, want no benefits [%s]", got, c.requirement)
				}
			case character.RestCompleted:
				if g, ok := got.(character.RestCompleted); !ok || g.Kind() != character.LongRest {
					t.Errorf("got %#v, want RestCompleted(LongRest) [%s]", got, c.requirement)
				}
			default:
				t.Fatalf("unexpected want %#v", want)
			}
		})
	}
}

// Requirement: per case.
// Path: per case.
func TestResolveRestRequiredHours(t *testing.T) {
	t.Parallel()
	std, trance := character.StandardLongRest, character.TranceLongRest
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		variant                 character.LongRestVariant
		resumed, hours          int
		wantCompleted           character.RestKind // zero when not completed
		wantNeeded              int
		wantRejected            bool
	}{
		{"RH-1 short rest of an hour", "FR-014 (SRD 5.2.1, \"Short Rest\", p. 187)", "happy", character.ShortRest, std, 0, 1, character.ShortRest, 0, false},
		{"RH-2 short rest of no time", "FR-015", "unhappy (boundary)", character.ShortRest, std, 0, 0, 0, 1, false},
		{"RH-3 long rest of 8 hours", "FR-003", "happy", character.LongRest, std, 0, 8, character.LongRest, 0, false},
		{"RH-4 long rest of 7 hours", "FR-015", "unhappy (boundary)", character.LongRest, std, 0, 7, 0, 1, false},
		{"RH-5 resumed once, 9 hours", "FR-016", "happy", character.LongRest, std, 1, 9, character.LongRest, 0, false},
		{"RH-6 resumed once, 8 hours", "FR-016", "unhappy (boundary)", character.LongRest, std, 1, 8, 0, 1, false},
		{"RH-7 resumed three times, 11 hours", "FR-016", "unhappy (rule interaction)", character.LongRest, std, 3, 11, character.LongRest, 0, false},
		{"RH-8 trance, 4 hours", "FR-004", "happy", character.LongRest, trance, 0, 4, character.LongRest, 0, false},
		{"RH-9 trance, 3 hours", "FR-004", "unhappy (boundary)", character.LongRest, trance, 0, 3, 0, 1, false},
		{"RH-10 trance resumed once, 5 hours", "FR-016", "unhappy (rule interaction)", character.LongRest, trance, 1, 5, character.LongRest, 0, false},
		{"RH-11 short rest with the trance variant", "FR-004", "unhappy (rule interaction)", character.ShortRest, trance, 0, 1, character.ShortRest, 0, false},
		{"RH-12 resumed short rest", "FR-017", "unhappy (state error)", character.ShortRest, std, 1, 1, 0, 0, true},
		{"RH-13 resumed twice, total 10 hours", "FR-067", "unhappy (boundary: stop, resume, stop again, resume)", character.LongRest, std, 2, 10, character.LongRest, 0, false},
		{"RH-14 resumed twice, total 9 hours", "FR-067", "unhappy (boundary)", character.LongRest, std, 2, 9, 0, 1, false},
		{"long rest of no hours needs the base", "FR-015", "unhappy (boundary)", character.LongRest, std, 0, 0, 0, 8, false},
		{"huge resumed count cannot wrap, ran its course", "FR-016", "unhappy (boundary: overflow)", character.LongRest, std, math.MaxInt, 0, 0, math.MaxInt, false},
		{"huge resumed count with hours clamps the need", "FR-016", "unhappy (boundary: overflow)", character.LongRest, std, math.MaxInt, 3, 0, math.MaxInt, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.ResolveRest(character.MustRestReport(c.kind, c.variant, ran(c.hours, c.resumed)))
			if err != nil {
				t.Fatal(err)
			}
			switch g := got.(type) {
			case character.RestCompleted:
				if g.Kind() != c.wantCompleted || c.wantCompleted == 0 {
					t.Errorf("completed %v, want %v [%s]", g.Kind(), c.wantCompleted, c.requirement)
				}
			case character.RestIncomplete:
				if g.HoursStillNeeded() != c.wantNeeded || c.wantNeeded == 0 {
					t.Errorf("incomplete needing %d, want %d [%s]", g.HoursStillNeeded(), c.wantNeeded, c.requirement)
				}
			case character.RestReportRejected:
				if !c.wantRejected || g.Problem() != character.ShortRestCannotBeResumed {
					t.Errorf("rejected %v [%s]", g.Problem(), c.requirement)
				}
			default:
				t.Errorf("got %#v [%s]", got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-017 - a resumed Short Rest is
// rejected whether it ran its course or was stopped, as a result and not an error.
// Path: unhappy (state error).
// Requirement: FR-016, FR-071, FR-023 - a huge resumed count never wraps the required hours.
// Path: unhappy (boundary: overflow).
func TestResolveRestHugeResumedStopped(t *testing.T) {
	t.Parallel()
	for _, resumed := range []int{math.MaxInt, math.MaxInt - 3} {
		got, err := character.ResolveRest(character.MustRestReport(character.LongRest, character.StandardLongRest, stopped(rDmg, 0, resumed)))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := got.(character.RestStoppedWithoutBenefits); !ok {
			t.Errorf("resumed %d: got %#v, want RestStoppedWithoutBenefits", resumed, got)
		}
	}
}

func TestResolveRestRejectsAResumedShortRestStopped(t *testing.T) {
	t.Parallel()
	got, err := character.ResolveRest(character.MustRestReport(character.ShortRest, character.StandardLongRest, stopped(rDmg, 1, 2)))
	if r, ok := got.(character.RestReportRejected); err != nil || !ok || r.Problem() != character.ShortRestCannotBeResumed || r.Problem().String() != "short_rest_cannot_be_resumed" {
		t.Errorf("ResolveRest = %#v, %v", got, err)
	}
	if character.RestReportRejection(2).String() != "unknown" {
		t.Error("an undefined rejection has a name")
	}
	if character.RestReportRejection(0).String() != "unknown" {
		t.Error("the zero rejection has a name")
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-066 - the zero report is invalid.
// Path: unhappy (invalid input).
func TestResolveRestRejectsTheZeroReport(t *testing.T) {
	t.Parallel()
	got, err := character.ResolveRest(character.RestReport{})
	if !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
		t.Errorf("ResolveRest(zero) = %v, %v", got, err)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-023, FR-066 - NewRestReport lists
// every problem once each, in input order, each matching its value and ErrInvalidInput.
// Path: per case.
func TestNewRestReportProblems(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		end                     character.RestEnd
		want                    []character.RestReportProblem
	}{
		{"negative hours rested", "FR-023", "unhappy (invalid input)", ran(-1, 0), []character.RestReportProblem{character.NegativeHoursRested}},
		{"negative resumed on a course", "FR-023", "unhappy (invalid input)", ran(0, -1), []character.RestReportProblem{character.NegativeInterruptionsResumed}},
		{"both on a course", "FR-066", "unhappy (invalid input)", ran(-2, -3), []character.RestReportProblem{character.NegativeHoursRested, character.NegativeInterruptionsResumed}},
		{"negative hours before", "FR-023", "unhappy (invalid input)", stopped(rDmg, -1, 0), []character.RestReportProblem{character.NegativeHoursRestedBefore}},
		{"negative resumed on a stop", "FR-023", "unhappy (invalid input)", stopped(rDmg, 0, -1), []character.RestReportProblem{character.NegativeInterruptionsResumed}},
		{"both on a stop", "FR-066", "unhappy (invalid input)", stopped(rDmg, -1, -1), []character.RestReportProblem{character.NegativeHoursRestedBefore, character.NegativeInterruptionsResumed}},
	}
	all := []character.RestReportProblem{character.NegativeHoursRested, character.NegativeHoursRestedBefore, character.NegativeInterruptionsResumed}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := character.NewRestReport(character.LongRest, character.StandardLongRest, c.end)
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput [%s]", err, c.requirement)
			}
			for _, p := range all {
				wanted := false
				for _, w := range c.want {
					wanted = wanted || w == p
				}
				if errors.Is(err, p) != wanted {
					t.Errorf("errors.Is(err, %v) = %v, want %v [%s]", p, !wanted, wanted, c.requirement)
				}
			}
			if joined, ok := err.(interface{ Unwrap() []error }); !ok || len(joined.Unwrap()) != len(c.want) {
				t.Errorf("err = %v, want %d joined problems", err, len(c.want))
			}
		})
	}
	for _, p := range all {
		if p.String() == "unknown" || p.Error() == "" {
			t.Errorf("problem %d has no name", int(p))
		}
	}
	if character.RestReportProblem(0).String() != "unknown" {
		t.Error("the zero problem has a name")
	}
	for p, want := range map[character.RestReportProblem]string{
		character.NegativeHoursRested: "negative_hours_rested", character.NegativeHoursRestedBefore: "negative_hours_rested_before",
		character.NegativeInterruptionsResumed: "negative_interruptions_resumed",
	} {
		if p.String() != want {
			t.Errorf("problem %d = %q, want %q", int(p), p.String(), want)
		}
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-066 - structural mistakes (kind,
// variant, interruption, end) are errors matching ErrInvalidInput; a resumed Short
// Rest and a Short Rest stopped by exertion are valid reports.
// Path: per case.
func TestNewRestReportStructure(t *testing.T) {
	t.Parallel()
	good := ran(1, 0)
	for name, c := range map[string]struct {
		kind    character.RestKind
		variant character.LongRestVariant
		end     character.RestEnd
		valid   bool
	}{
		"invalid kind": {0, character.StandardLongRest, good, false}, "invalid variant": {character.LongRest, 0, good, false},
		"nil end": {character.LongRest, character.StandardLongRest, nil, false}, "invalid interruption": {character.LongRest, character.StandardLongRest, stopped(0, 1, 0), false},
		"resumed short rest": {character.ShortRest, character.StandardLongRest, ran(1, 1), true}, "short rest stopped by exertion": {character.ShortRest, character.StandardLongRest, stopped(rEx, 1, 0), true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r, err := character.NewRestReport(c.kind, c.variant, c.end)
			if c.valid != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || r.IsZero() == c.valid {
				t.Errorf("NewRestReport = %v, %v", r, err)
			}
			if c.valid && (r.Kind() != c.kind || r.Variant() != c.variant || r.End() != c.end) {
				t.Errorf("accessors disagree: %#v", r)
			}
		})
	}
	func() {
		defer func() {
			if e, ok := recover().(error); !ok || !errors.Is(e, srderr.ErrInvalidInput) {
				t.Errorf("MustRestReport did not panic with an ErrInvalidInput error")
			}
		}()
		character.MustRestReport(0, 0, nil)
	}()
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-066 and testing.md section 10 -
// NewRestReport and ResolveRest never panic, and a report that builds always resolves.
// Path: unhappy (invalid input).
func FuzzNewRestReport(f *testing.F) {
	f.Add(1, 1, 0, 0, 0, false)
	f.Add(2, 2, 3, -1, 4, true)
	f.Add(0, 0, -5, 9, 99, true)
	f.Fuzz(func(t *testing.T, kind, variant, hours, resumed, interruption int, stop bool) {
		end := character.RestEnd(character.NewRestRanItsCourse(hours, resumed))
		if stop {
			end = character.NewRestStopped(character.RestInterruption(interruption), hours, resumed)
		}
		r, err := character.NewRestReport(character.RestKind(kind), character.LongRestVariant(variant), end)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		res, err := character.ResolveRest(r)
		if err != nil || res == nil {
			t.Fatalf("ResolveRest(%v) = %v, %v", r, res, err)
		}
		if _, done := res.(character.RestCompleted); done && r.Kind() == character.LongRest && hours == 0 {
			t.Fatalf("Long Rest of 0 hours completed: %v", r)
		}
	})
}
