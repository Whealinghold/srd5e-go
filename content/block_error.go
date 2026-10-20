package content

import "github.com/Whealinghold/srd5e-go/srderr"

// BlockProblem says why a constructor of a building block ([NamedText], a
// [Prerequisite], a [ToolProficiency] or a [TextBlock]) or of an entry shape
// ([Species], [Background], [Feat] and their parts) rejected its arguments.
// The zero value is not a rejection.
type BlockProblem int

// The building-block rejection reasons. The decoder reports each as the
// problem kind noted.
const (
	// BlockInvalidName: a name is empty, has leading or trailing white space,
	// or holds a rejected character (invalidValue).
	BlockInvalidName BlockProblem = iota + 1
	// BlockBlankText: a text, item, cell, line, title or category has no
	// non-white-space character (invalidValue).
	BlockBlankText
	// BlockEmpty: a collection that must not be empty is empty (outOfRange).
	BlockEmpty
	// BlockOutOfRange: a number is outside its range (outOfRange).
	BlockOutOfRange
	// BlockInvalidMember: a value is not a member of its vocabulary
	// (invalidValue).
	BlockInvalidMember
	// BlockDuplicate: a repeat in a collection that is a set (duplicate).
	BlockDuplicate
	// BlockRaggedRow: a table row is not as wide as the header
	// (constraintViolation).
	BlockRaggedRow
	// BlockInvalidCharacter: a text, item, cell, line, title or category holds
	// invalid UTF-8 or a control or bidirectional character (invalidValue).
	BlockInvalidCharacter
	// BlockMissing: a part that must be present is absent (missingField).
	BlockMissing
	// BlockConstraint: a rule between parts is broken, such as a lineage whose
	// levels are out of order (constraintViolation).
	BlockConstraint
)

// String returns the lower camel case name of p, or "unknown" for an
// undefined value.
func (p BlockProblem) String() string {
	switch p {
	case BlockInvalidName:
		return "invalidName"
	case BlockBlankText:
		return "blankText"
	case BlockEmpty:
		return "empty"
	case BlockOutOfRange:
		return "outOfRange"
	case BlockInvalidMember:
		return "invalidMember"
	case BlockDuplicate:
		return "duplicate"
	case BlockRaggedRow:
		return "raggedRow"
	case BlockInvalidCharacter:
		return "invalidCharacter"
	case BlockMissing:
		return "missing"
	case BlockConstraint:
		return "constraint"
	}
	return unknownName
}

// BlockError reports arguments rejected by the constructor of a building
// block. Block names the type (for example "TextTable") and Problem says why.
// It matches srderr.ErrInvalidInput with errors.Is, and another *BlockError
// with the same Block and Problem. Do not branch on the rendered text.
type BlockError struct {
	Block   string
	Problem BlockProblem
}

// Error renders the block and problem. The text is not a contract.
func (e *BlockError) Error() string {
	return "content: invalid " + e.Block + ": " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *BlockError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *BlockError with the same Block and Problem.
func (e *BlockError) Is(target error) bool {
	t, ok := target.(*BlockError)
	return ok && t != nil && e != nil && *t == *e
}

func blockError(block string, p BlockProblem) error { return &BlockError{Block: block, Problem: p} }

// checkName applies the rule of schema FR-027 ([validName]): a name is not
// empty, has no surrounding white space and holds only clean characters.
func checkName(block, name string) error {
	if !validName(name) {
		return blockError(block, BlockInvalidName)
	}
	return nil
}

// checkText applies the rule of schema FR-028: a text has at least one
// non-white-space character (BlockBlankText, checked first) and holds only
// clean characters (BlockInvalidCharacter).
func checkText(block, text string) error {
	switch {
	case blankText(text):
		return blockError(block, BlockBlankText)
	case !cleanCharacters(text, true):
		return blockError(block, BlockInvalidCharacter)
	}
	return nil
}
