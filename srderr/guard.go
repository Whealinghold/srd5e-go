package srderr

import "runtime/debug"

// Guard calls fn exactly once and reports its failure as a port failure.
//
// It returns nil when fn returns nil. When fn returns an error it returns
// WrapPort(port, op, err). When fn panics it recovers and returns a
// *PortError wrapping a *PanicError (which matches ErrPortPanic) with the
// stack captured at recovery; the panic is not re-raised. A call to
// runtime.Goexit inside fn is neither recovered nor converted. Guard takes no
// context: fn captures one if it needs it.
//
// A panic raised by this module's own code called from inside fn cannot be told
// from a port panic, so call MustX helpers outside fn.
func Guard(port, op string, fn func() error) error {
	_, err := GuardValue(port, op, func() (struct{}, error) { return struct{}{}, fn() })
	return err
}

// GuardValue is Guard for a port call that also returns a value.
//
// On success it returns fn's value and a nil error unchanged. On a returned
// error it returns fn's value and WrapPort(port, op, err). On a panic it
// returns the zero value and the *PortError described by Guard.
func GuardValue[T any](port, op string, fn func() (T, error)) (v T, err error) {
	returned := false
	defer func() {
		if returned {
			return
		}
		// recover returns nil for runtime.Goexit, which must keep unwinding.
		if r := recover(); r != nil {
			var zero T
			v = zero
			err = &PortError{Port: port, Op: op, Err: &PanicError{Value: r, Stack: debug.Stack()}}
		}
	}()
	v, err = fn()
	returned = true
	return v, WrapPort(port, op, err)
}
