package dicetest

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// Errors returned by FixedRoller. They are test-script mistakes, reported as
// ordinary errors so a test can assert on them with errors.Is.
var (
	// ErrExhausted: the scripted values ran out.
	ErrExhausted = errors.New("dicetest: scripted values exhausted")
	// ErrValueOutOfRange: a scripted value is not a face of the requested die.
	ErrValueOutOfRange = errors.New("dicetest: scripted value not on die")
)

var _ dice.Roller = (*FixedRoller)(nil)

type supply int

const (
	listed supply = iota
	lowest
	highest
)

// FixedRoller is a deterministic dice.Roller that records every request. It is
// not safe for concurrent use.
type FixedRoller struct {
	supply   supply
	values   []int
	next     int
	requests []dice.Expression
}

// NewFixedRoller returns a roller that answers with values, in order, one per
// die. A request that needs more values than remain, or a value that is not a
// face of the requested die, fails with ErrExhausted or ErrValueOutOfRange and
// consumes nothing.
func NewFixedRoller(values ...int) *FixedRoller {
	return &FixedRoller{supply: listed, values: slices.Clone(values)}
}

// NewMinimumRoller returns a roller whose every die shows 1.
func NewMinimumRoller() *FixedRoller { return &FixedRoller{supply: lowest} }

// NewMaximumRoller returns a roller whose every die shows its top face.
func NewMaximumRoller() *FixedRoller { return &FixedRoller{supply: highest} }

// Roll answers e as described on the constructors and records it. A failed
// request is not recorded.
func (f *FixedRoller) Roll(e dice.Expression) (dice.Result, error) {
	faces, err := f.faces(e)
	if err != nil {
		return dice.Result{}, err
	}
	rolls := make([]dice.DieRoll, len(faces))
	for i, face := range faces {
		rolls[i], err = dice.NewDieRoll(e.Die(), face)
		if err != nil {
			return dice.Result{}, err
		}
	}
	res, err := dice.NewResult(e, rolls)
	if err != nil {
		return dice.Result{}, err
	}
	if f.supply == listed {
		f.next += e.Count()
	}
	f.requests = append(f.requests, e)
	return res, nil
}

func (f *FixedRoller) faces(e dice.Expression) ([]int, error) {
	n := e.Count()
	switch f.supply {
	case lowest:
		return slices.Repeat([]int{1}, n), nil
	case highest:
		return slices.Repeat([]int{e.Die().Sides()}, n), nil
	case listed:
	}
	if len(f.values)-f.next < n {
		return nil, fmt.Errorf("%w: %d left, %s needs %d", ErrExhausted, len(f.values)-f.next, e, n)
	}
	faces := f.values[f.next : f.next+n]
	for _, v := range faces {
		if v < 1 || v > e.Die().Sides() {
			return nil, fmt.Errorf("%w: %d on %s", ErrValueOutOfRange, v, e.Die())
		}
	}
	return faces, nil
}

// Requests returns a copy of the expressions rolled successfully, in order.
func (f *FixedRoller) Requests() []dice.Expression { return slices.Clone(f.requests) }

// Remaining returns how many scripted values are unused. bounded is false for
// minimum and maximum rollers, which never run out.
func (f *FixedRoller) Remaining() (n int, bounded bool) {
	if f.supply != listed {
		return 0, false
	}
	return len(f.values) - f.next, true
}
