package srderr_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/srderr"
)

var errBackend = errors.New("backend failed")

func TestWrapPort(t *testing.T) {
	t.Parallel()
	inner := &srderr.PortError{Port: "content", Op: "load", Err: errBackend}
	tests := []struct {
		name     string
		err      error
		wantNil  bool
		wantSame bool
		wantPort string
	}{
		{name: "nil stays nil", err: nil, wantNil: true},
		{name: "plain error is wrapped", err: errBackend, wantPort: "dice"},
		{name: "same port is not wrapped twice", err: inner, wantSame: true, wantPort: "content"},
		{name: "joined same port is not wrapped twice", err: fmt.Errorf("ctx: %w", inner), wantSame: true, wantPort: "content"},
		{name: "different port is wrapped", err: inner, wantPort: "dice"},
		{name: "context canceled survives", err: context.Canceled, wantPort: "dice"},
		{name: "deadline survives", err: context.DeadlineExceeded, wantPort: "dice"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := srderr.WrapPort(tc.wantPort, "roll", tc.err)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("WrapPort(nil) = %v, want nil", got)
				}
				return
			}
			if tc.wantSame && got != tc.err { //nolint:errorlint // identity is the point
				t.Fatalf("WrapPort returned a new error, want same instance")
			}
			var pe *srderr.PortError
			if !errors.As(got, &pe) || pe.Port != tc.wantPort {
				t.Fatalf("outermost port = %+v, want %q", pe, tc.wantPort)
			}
			if !errors.Is(got, tc.err) {
				t.Fatalf("errors.Is(got, original) = false")
			}
		})
	}
}

func TestWrapPortOuterWrapsDifferentPort(t *testing.T) {
	t.Parallel()
	inner := srderr.WrapPort("content", "load", errBackend)
	outer := srderr.WrapPort("dice", "roll", inner)
	var pe *srderr.PortError
	if !errors.As(outer, &pe) || pe.Port != "dice" || pe.Err != inner { //nolint:errorlint // identity is the point
		t.Fatalf("outer = %+v, want dice wrapping the content error", pe)
	}
}

func TestPortError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  *srderr.PortError
		want string
	}{
		{"full", &srderr.PortError{Port: "dice", Op: "roll", Err: errBackend}, "dice roll: backend failed"},
		{"nil cause renders", &srderr.PortError{Port: "dice", Op: "roll"}, "dice roll: failed"},
		{"empty op", &srderr.PortError{Port: "dice", Err: errBackend}, "dice: backend failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
	if (&srderr.PortError{Port: "dice"}).Unwrap() != nil {
		t.Fatal("Unwrap of nil cause must be nil")
	}
}

func TestContractViolation(t *testing.T) {
	t.Parallel()
	err := srderr.ContractViolation("dice", "roll", "wrong expression")
	var pe *srderr.PortError
	var ce *srderr.ContractError
	switch {
	case !errors.As(err, &pe) || pe.Port != "dice" || pe.Op != "roll":
		t.Fatalf("not a dice roll PortError: %v", err)
	case !errors.Is(err, srderr.ErrContractViolation):
		t.Fatal("does not match ErrContractViolation")
	case !errors.As(err, &ce) || ce.Detail != "wrong expression":
		t.Fatalf("detail not carried as data: %+v", ce)
	}
	if errors.Is(err, srderr.ErrPortPanic) {
		t.Fatal("must not match ErrPortPanic")
	}
	if ce.Error() != "port contract violation: wrong expression" {
		t.Fatalf("Error() = %q", ce.Error())
	}
}

func TestErrorType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"plain", errBackend, ""},
		{"port error", srderr.WrapPort("dice", "roll", errBackend), "dice_error"},
		{"wrapped port error", fmt.Errorf("x: %w", srderr.WrapPort("content", "load", errBackend)), "content_error"},
		{"outermost wins", srderr.WrapPort("dice", "roll", srderr.WrapPort("content", "load", errBackend)), "dice_error"},
		{"contract violation", srderr.ContractViolation("dice", "roll", "d"), "dice_error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := srderr.ErrorType(tc.err); got != tc.want {
				t.Fatalf("ErrorType = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPanicErrorUnwrap(t *testing.T) {
	t.Parallel()
	cause := errors.New("nil dereference")
	tests := []struct {
		name      string
		value     any
		wantCause bool
	}{
		{"error value", cause, true},
		{"string value", "boom", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pe := &srderr.PanicError{Value: tc.value}
			if !errors.Is(pe, srderr.ErrPortPanic) {
				t.Fatal("must match ErrPortPanic")
			}
			if got := errors.Is(pe, cause); got != tc.wantCause {
				t.Fatalf("errors.Is(cause) = %v, want %v", got, tc.wantCause)
			}
			if pe.Error() == "" {
				t.Fatal("empty Error()")
			}
		})
	}
}

func TestWrapPortDifferentPortOrdering(t *testing.T) {
	t.Parallel()
	// Requirement: none - pins current behaviour of WrapPort port comparison.
	// Ports on either side of the existing one, lexically, are still wrapped.
	for _, port := range []string{"a", "c"} {
		inner := srderr.WrapPort("b", "load", errBackend)
		var pe *srderr.PortError
		if !errors.As(srderr.WrapPort(port, "op", inner), &pe) || pe.Port != port {
			t.Errorf("port %q: got %+v, want wrapped by %q", port, pe, port)
		}
	}
}
