package srderr_test

import (
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/srderr"
)

func TestGuardValue(t *testing.T) {
	t.Parallel()
	type result struct {
		val       int
		wantErr   bool
		wantPanic bool
		wantIs    error
	}
	tests := []struct {
		name string
		fn   func() (int, error)
		want result
	}{
		{"value passes through", func() (int, error) { return 7, nil }, result{val: 7}},
		{"zero value passes through", func() (int, error) { return 0, nil }, result{}},
		{"error is wrapped", func() (int, error) { return 3, errBackend }, result{val: 3, wantErr: true, wantIs: errBackend}},
		{"port error of same port is unchanged", func() (int, error) { return 0, srderr.WrapPort("dice", "roll", errBackend) }, result{wantErr: true, wantIs: errBackend}},
		{"error panic", func() (int, error) { panic(errBackend) }, result{wantErr: true, wantPanic: true, wantIs: errBackend}},
		{"string panic", func() (int, error) { panic("boom") }, result{wantErr: true, wantPanic: true}},
		{"nil panic", func() (int, error) { panic(nil) }, result{wantErr: true, wantPanic: true}},
		{"runtime error panic", func() (int, error) { s, i := []int{}, 3; return s[i], nil }, result{wantErr: true, wantPanic: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := srderr.GuardValue("dice", "roll", tc.fn)
			if got != tc.want.val {
				t.Fatalf("value = %d, want %d", got, tc.want.val)
			}
			if !tc.want.wantErr {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			var pe *srderr.PortError
			if !errors.As(err, &pe) || pe.Port != "dice" || pe.Op != "roll" {
				t.Fatalf("err = %v, want dice roll PortError", err)
			}
			if tc.want.wantIs != nil && !errors.Is(err, tc.want.wantIs) {
				t.Fatalf("errors.Is(%v) = false", tc.want.wantIs)
			}
			var pan *srderr.PanicError
			if gotPanic := errors.As(err, &pan); gotPanic != tc.want.wantPanic {
				t.Fatalf("PanicError present = %v, want %v", gotPanic, tc.want.wantPanic)
			}
			if tc.want.wantPanic {
				if !errors.Is(err, srderr.ErrPortPanic) || len(pan.Stack) == 0 {
					t.Fatalf("panic chain incomplete: stack=%d", len(pan.Stack))
				}
			}
		})
	}
}

func TestGuard(t *testing.T) {
	t.Parallel()
	calls := 0
	if err := srderr.Guard("dice", "roll", func() error { calls++; return nil }); err != nil || calls != 1 {
		t.Fatalf("success: err=%v calls=%d", err, calls)
	}
	calls = 0
	err := srderr.Guard("dice", "roll", func() error { calls++; return errBackend })
	if !errors.Is(err, errBackend) || srderr.ErrorType(err) != "dice_error" || calls != 1 {
		t.Fatalf("failure: err=%v calls=%d (must not retry)", err, calls)
	}
	err = srderr.Guard("dice", "roll", func() error { panic("x") })
	if !errors.Is(err, srderr.ErrPortPanic) {
		t.Fatalf("panic: err=%v", err)
	}
}

func TestGuardDoesNotSwallowGoexit(t *testing.T) {
	t.Parallel()
	var after bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srderr.Guard("dice", "roll", func() error { runtime.Goexit(); return nil })
		after = true
	}()
	<-done
	if after {
		t.Fatal("code after Guard ran: Goexit was swallowed")
	}
}

func TestGuardConcurrent(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srderr.Guard("dice", "roll", func() error { panic("x") }); err == nil {
				t.Error("want error")
			}
		}()
	}
	wg.Wait()
}
