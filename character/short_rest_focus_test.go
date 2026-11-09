package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-114 and table "Short Rest focus").
// Path: per case.
func TestNewShortRestFocus(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185`
	ran := character.NewRestRanItsCourse(1, 0)
	stop := character.NewRestStopped(character.TakingDamage, 0, 0)
	longRan := character.NewRestRanItsCourse(8, 0)
	longStop := character.NewRestStopped(character.TakingDamage, 3, 0)
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		end                     character.RestEnd
		resolution              character.RestResolution // overrides end when set
		wantReason              character.ShortRestFocusReason
		wantCompleted           bool
	}{
		{"finished Short Rest", p, "happy", character.ShortRest, ran, nil, 0, true},
		{"interrupted Short Rest", p, "unhappy (state error)", character.ShortRest, stop, nil, 0, false},
		{"unfinished Short Rest", p, "unhappy (boundary)", character.ShortRest, nil, character.RestIncomplete{}, 0, false},
		{"finished Long Rest", p, "unhappy (rule interaction)", character.LongRest, longRan, nil, character.ShortRestFocusNotAShortRest, false},
		{"Long Rest interrupted with Short Rest benefits", p, "unhappy (rule interaction)", character.LongRest, longStop, nil, character.ShortRestFocusNotAShortRest, false},
		{"rejected report on a Short Rest", p, "unhappy (invalid input)", character.ShortRest, nil, character.RestReportRejected{}, character.ShortRestFocusRestReportRejected, false},
		{"rejected report on a Long Rest is not a Short Rest first", p, "unhappy (invalid input)", character.LongRest, nil, character.RestReportRejected{}, character.ShortRestFocusNotAShortRest, false},
		{"Short Rest with Long Rest benefits", p, "unhappy (invalid input)", character.ShortRest, nil, character.RestStoppedWithShortRestBenefits{}, character.ShortRestFocusKindResolutionMismatch, false},
		{"kind disagrees (long result for short)", p, "unhappy (invalid input)", character.ShortRest, longRan, nil, character.ShortRestFocusKindResolutionMismatch, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := c.resolution
			if res == nil {
				_, res = restOf(t, c.kind, c.end)
			}
			if c.name == "kind disagrees (long result for short)" {
				_, res = restOf(t, character.LongRest, c.end)
			}
			f, err := character.NewShortRestFocus(inst("a"), character.RestPurposeAttune, character.InContact, c.kind, res)
			if c.wantReason != 0 {
				var rej *character.ShortRestFocusRejected
				if !errors.As(err, &rej) || rej.Reason != c.wantReason || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ShortRestFocusRejected{Reason: c.wantReason}) || !f.IsZero() {
					t.Fatalf("err = %v, want reason %v", err, c.wantReason)
				}
				return
			}
			if err != nil || f.IsCompleted() != c.wantCompleted || f.InstanceID() != "a" || f.Purpose() != character.RestPurposeAttune || f.Contact() != character.InContact {
				t.Errorf("focus = %+v, %v", f, err)
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-114 - a blank instance, an invalid purpose, contact or kind, and a nil resolution are invalid input.
// Path: unhappy (invalid input).
func TestNewShortRestFocusInvalidInput(t *testing.T) {
	t.Parallel()
	_, res := restOf(t, character.ShortRest, character.NewRestRanItsCourse(1, 0))
	for name, run := range map[string]func() error{
		"blank instance": func() error {
			_, e := character.NewShortRestFocus("", character.RestPurposeAttune, character.InContact, character.ShortRest, res)
			return e
		},
		"purpose": func() error {
			_, e := character.NewShortRestFocus(inst("a"), 0, character.InContact, character.ShortRest, res)
			return e
		},
		"contact": func() error {
			_, e := character.NewShortRestFocus(inst("a"), character.RestPurposeAttune, 0, character.ShortRest, res)
			return e
		},
		"kind": func() error {
			_, e := character.NewShortRestFocus(inst("a"), character.RestPurposeAttune, character.InContact, 0, res)
			return e
		},
		"nil resolution": func() error {
			_, e := character.NewShortRestFocus(inst("a"), character.RestPurposeAttune, character.InContact, character.ShortRest, nil)
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if (character.ShortRestFocusKindResolutionMismatch).String() == "unknown" || character.ShortRestFocusReason(0).String() != "unknown" {
		t.Error("String")
	}
}
