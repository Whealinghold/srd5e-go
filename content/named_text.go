package content

// NamedText is a name and its verbatim text, such as a species special trait
// (SRD 5.2.1, "Parts of a Species: Special Traits", p. 84). Both parts are
// kept character for character. The zero NamedText is invalid. Immutable,
// comparable and safe for concurrent use.
type NamedText struct {
	name string
	text string
}

// NewNamedText returns a NamedText. The name must be non-empty with no leading
// or trailing white space, and the text must contain a non-white-space
// character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName or BlockBlankText, checked in that order. It never panics.
func NewNamedText(name, text string) (NamedText, error) {
	if err := checkName("NamedText", name); err != nil {
		return NamedText{}, err
	}
	if err := checkText("NamedText", text); err != nil {
		return NamedText{}, err
	}
	return NamedText{name: name, text: text}, nil
}

// MustNamedText is like [NewNamedText] but panics with its error when the
// arguments are invalid.
func MustNamedText(name, text string) NamedText {
	n, err := NewNamedText(name, text)
	if err != nil {
		panic(err)
	}
	return n
}

// Name returns the name.
func (n NamedText) Name() string { return n.name }

// Text returns the text.
func (n NamedText) Text() string { return n.text }
