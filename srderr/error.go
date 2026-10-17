package srderr

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors owned by this package. Adding one is a specification change.
var (
	// ErrInvalidInput is matched by every error that reports untrusted input
	// failing validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrPortPanic is in the chain of every PanicError.
	ErrPortPanic = errors.New("port panicked")
	// ErrContractViolation is in the chain of every error built by
	// ContractViolation.
	ErrContractViolation = errors.New("port contract violation")
)

// PortError reports that the port named Port failed while performing Op.
//
// Err is the underlying error and is returned unchanged by Unwrap. Err may be
// nil; the error then still renders. Build values with WrapPort, Guard,
// GuardValue or ContractViolation; direct construction is for tests.
type PortError struct {
	Port string
	Op   string
	Err  error
}

// Error renders "<port> <op>: <cause>". The text is not a contract.
func (e *PortError) Error() string {
	head := strings.TrimSpace(e.Port + " " + e.Op)
	if e.Err == nil {
		return head + ": failed"
	}
	return head + ": " + e.Err.Error()
}

// Unwrap returns Err so errors.Is and errors.As reach the original cause.
func (e *PortError) Unwrap() error { return e.Err }

// ContractError is the cause carried by ContractViolation. It matches
// ErrContractViolation with errors.Is and holds the detail as data.
type ContractError struct {
	Detail string
}

// Error renders the detail after the sentinel text.
func (e *ContractError) Error() string {
	return ErrContractViolation.Error() + ": " + e.Detail
}

// Unwrap returns ErrContractViolation.
func (e *ContractError) Unwrap() error { return ErrContractViolation }

// WrapPort attributes err to the port named port, performing op.
//
// It returns nil when err is nil. When the outermost *PortError in err's chain
// is already for the same port, err is returned unchanged, so a failure is
// never wrapped twice; a *PortError for a different port is wrapped. The
// original err stays reachable with errors.Is and errors.As. It never panics.
func WrapPort(port, op string, err error) error {
	if err == nil {
		return nil
	}
	var pe *PortError
	if errors.As(err, &pe) && pe.Port == port {
		return err
	}
	return &PortError{Port: port, Op: op, Err: err}
}

// ContractViolation reports that the port named port broke its documented
// contract while performing op, for example by returning a result for a
// different request. The returned *PortError wraps a *ContractError holding
// detail, which matches ErrContractViolation. It never panics.
func ContractViolation(port, op, detail string) error {
	return &PortError{Port: port, Op: op, Err: &ContractError{Detail: detail}}
}

// ErrorType returns the value of the error.type telemetry attribute for err:
// "<port>_error" for the outermost *PortError in the chain, or "" when there is
// none (including a nil err). The value is one per port, never per reason or
// instance, and is a public telemetry contract.
func ErrorType(err error) string {
	var pe *PortError
	if errors.As(err, &pe) {
		return pe.Port + "_error"
	}
	return ""
}

// PanicError is the cause of a *PortError built from a recovered port panic.
type PanicError struct {
	// Value is the value passed to panic.
	Value any
	// Stack is the goroutine stack captured at recovery.
	Stack []byte
}

// Error renders the panic value.
func (e *PanicError) Error() string { return fmt.Sprintf("port panicked: %v", e.Value) }

// Unwrap returns ErrPortPanic and, when Value is an error, that error.
func (e *PanicError) Unwrap() []error {
	if err, ok := e.Value.(error); ok {
		return []error{ErrPortPanic, err}
	}
	return []error{ErrPortPanic}
}
