package combat

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// invalidInput returns an error matching srderr.ErrInvalidInput. The detail is
// for humans; it is not a contract.
func invalidInput(detail string) error {
	return fmt.Errorf("combat: %s: %w", detail, srderr.ErrInvalidInput)
}

// unreachableError is the panic value of a branch no valid value reaches.
func unreachableError(what string, v any) error {
	return fmt.Errorf("combat: unreachable %s %T", what, v)
}

// typeName is the dynamic type of v, for a human-readable detail.
func typeName(v any) string { return fmt.Sprintf("%T", v) }

// ErrDuplicateSource is matched, with srderr.ErrInvalidInput, by the error of every spell modifier query given two
// sources with one spell and one target: the host skipped the combining step of Combining Spell Effects
// (spell-granted-modifiers.md FR-008).
var ErrDuplicateSource = fmt.Errorf("combat: two spell modifier sources with one spell and target: %w", srderr.ErrInvalidInput)

// ErrFaceOutOfRange is matched, with srderr.ErrInvalidInput, by the error of every spell outcome table lookup given a
// face the die of the table does not have (spell-outcome-tables.md FR-002, FR-044).
var ErrFaceOutOfRange = fmt.Errorf("combat: face not on the die of the table: %w", srderr.ErrInvalidInput)

// ErrWallDestroyedCount is matched, with srderr.ErrInvalidInput, by the error of NewPrismaticWallStateAfter given a
// count of destroyed layers outside 0 to 7 (spell-outcome-tables.md FR-075).
var ErrWallDestroyedCount = fmt.Errorf("combat: destroyed layers outside 0 to 7: %w", srderr.ErrInvalidInput)

// ErrSpellTableLimit is matched, with srderr.ErrInvalidInput, by the error of a spell outcome command that holds a
// collection over its cap: more than 256 Prismatic Spray targets or 1000 Confusion candidates (plan P10; the
// result value limit_exceeded of the other use cases is an error here because this use case returns malformed
// commands as errors, spell-outcome-tables.md FR-090).
var ErrSpellTableLimit = fmt.Errorf("combat: spell outcome collection over its cap (limit_exceeded): %w", srderr.ErrInvalidInput)
