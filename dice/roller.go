package dice

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// Port is the srderr port name used for every roller failure.
const Port = "dice_roller"

// Roller is the port through which randomness enters. It is synchronous and
// in-memory, so it takes no context.
//
// Contract: Roll returns a Result whose Expression equals the requested one,
// holding Count() rolls of Die() each within 1..Sides(). On failure it returns
// the zero Result and a non-nil error; it never returns a partial or default
// result. Implementations should wrap failures with srderr.WrapPort(Port,
// "roll", err), but callers must go through Roll, which enforces this. See
// dicetest.RunRollerContract.
//
// Whether a Roller is safe for concurrent use is up to the implementation and
// MUST be documented. A Roller shared by several goroutines (for example
// through one d20.Resolver) MUST be safe for concurrent use; SeededRoller is
// not.
type Roller interface {
	Roll(Expression) (Result, error)
}

// Roll rolls e with r and checks the answer.
//
// It returns r's result when r returns no error and the result's expression
// equals e. A returned error is passed through srderr.WrapPort, so it is a
// *srderr.PortError (Port is the package constant Port) that still matches the
// original cause. A panic in r is recovered into a *srderr.PortError matching
// srderr.ErrPortPanic. A result for a different expression, including the zero
// Result, is a *srderr.PortError matching srderr.ErrContractViolation. On any
// failure the Result is the zero Result. A nil r is a *srderr.PortError matching srderr.ErrInvalidInput; the
// recover is only a backstop for misbehaving rollers.
func Roll(r Roller, e Expression) (Result, error) {
	if r == nil {
		return Result{}, srderr.WrapPort(Port, "roll", fmt.Errorf("nil roller: %w", srderr.ErrInvalidInput))
	}
	res, err := srderr.GuardValue(Port, "roll", func() (Result, error) { return r.Roll(e) })
	if err != nil {
		return Result{}, err
	}
	if res.Expression() != e {
		return Result{}, srderr.ContractViolation(Port, "roll", "result answers "+res.Expression().Notation()+", not "+e.Notation())
	}
	return res, nil
}
