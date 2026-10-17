package d20_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func TestInvalidError(t *testing.T) {
	t.Parallel()
	err := error(&d20.InvalidError{Reason: d20.ReasonZeroRoll, Detail: "x"})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Error("does not match ErrInvalidInput")
	}
	if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonZeroRoll}) {
		t.Error("does not match same reason")
	}
	if errors.Is(err, &d20.InvalidError{Reason: d20.ReasonUnknownKind}) {
		t.Error("matches a different reason")
	}
	if errors.Is(err, errors.New("other")) {
		t.Error("matches a foreign error")
	}
	if got := err.Error(); got != "d20: zero roll: x" {
		t.Errorf("Error() = %q", got)
	}
	if got := (&d20.InvalidError{Reason: d20.ReasonZeroRoll}).Error(); got != "d20: zero roll" {
		t.Errorf("Error() without detail = %q", got)
	}
}

func TestReasonString(t *testing.T) {
	t.Parallel()
	for r := d20.ReasonIndexOutOfRange; r <= d20.ReasonMissingRoller; r++ {
		if s := r.String(); s == "unknown" || s == "" || strings.ToLower(s) != s {
			t.Errorf("Reason(%d).String() = %q", r, s)
		}
	}
	for _, r := range []d20.Reason{0, d20.ReasonMissingRoller + 1} {
		if r.String() != "unknown" {
			t.Errorf("Reason(%d).String() = %q, want unknown", r, r.String())
		}
	}
}
