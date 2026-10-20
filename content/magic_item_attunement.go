package content

// Attunement is the Attunement a magic item's header prints, a closed family
// with three variants: [NoAttunement], [AttunementRequired] and
// [AttunementRequiredBy] (SRD 5.2.1, "Attunement", p. 102). The prerequisite
// is verbatim text; see [AttunementPrerequisite] for the structured host-side
// form. Consumers switch over it exhaustively. Every variant is immutable,
// comparable and safe for concurrent use.
type Attunement interface{ isAttunement() }

// NoAttunement is an item that needs no Attunement. Its zero value is valid.
type NoAttunement struct{}

func (NoAttunement) isAttunement() {}

// AttunementRequired is "Requires Attunement" with no prerequisite. Its zero
// value is valid.
type AttunementRequired struct{}

func (AttunementRequired) isAttunement() {}

// AttunementRequiredBy is "Requires Attunement by" and a prerequisite kept
// verbatim, such as "a Wizard". The zero value is invalid; build it with
// [NewAttunementRequiredBy].
type AttunementRequiredBy struct{ prerequisite string }

// NewAttunementRequiredBy returns the prerequisite text, which must contain a
// non-white-space character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter otherwise. It never panics.
func NewAttunementRequiredBy(prerequisite string) (AttunementRequiredBy, error) {
	if err := checkText("AttunementRequiredBy", prerequisite); err != nil {
		return AttunementRequiredBy{}, err
	}
	return AttunementRequiredBy{prerequisite: prerequisite}, nil
}

// Prerequisite returns the verbatim text after "Requires Attunement by ".
func (a AttunementRequiredBy) Prerequisite() string { return a.prerequisite }

func (AttunementRequiredBy) isAttunement() {}

// validAttunement reports whether a is a variant its constructor would
// accept.
func validAttunement(a Attunement) bool {
	switch v := a.(type) {
	case NoAttunement, AttunementRequired:
		return true
	case AttunementRequiredBy:
		_, err := NewAttunementRequiredBy(v.prerequisite)
		return err == nil
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}
