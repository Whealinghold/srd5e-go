package srderr_test

import (
	"errors"
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

func ExampleGuardValue() {
	_, err := srderr.GuardValue("dice", "roll", func() (int, error) {
		return 0, errors.New("source unavailable")
	})
	var pe *srderr.PortError
	if errors.As(err, &pe) {
		fmt.Println(pe.Port, pe.Op, srderr.ErrorType(err))
	}
	// Output: dice roll dice_error
}

func ExampleGuard_panic() {
	err := srderr.Guard("dice", "roll", func() error { panic("boom") })
	fmt.Println(errors.Is(err, srderr.ErrPortPanic))
	// Output: true
}

func ExampleContractViolation() {
	err := srderr.ContractViolation("dice", "roll", "wrong expression")
	fmt.Println(errors.Is(err, srderr.ErrContractViolation), err)
	// Output: true dice roll: port contract violation: wrong expression
}
