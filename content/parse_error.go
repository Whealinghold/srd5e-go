package content

import "github.com/Whealinghold/srd5e-go/srderr"

// ParseError reports a string that is not a member of a closed vocabulary. It
// matches srderr.ErrInvalidInput with errors.Is; the decoder reports it as an
// invalidValue problem. Vocabulary names the type (for example
// "DurationUnit") and Value is the rejected text. Do not branch on the
// rendered text.
type ParseError struct {
	Vocabulary string
	Value      string
}

// Error renders the vocabulary name. The text is not a contract.
func (e *ParseError) Error() string { return "content: not a " + e.Vocabulary + " member" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *ParseError) Unwrap() error { return srderr.ErrInvalidInput }
