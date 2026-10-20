package content

// ToolProficiency is the closed set of tool proficiencies a background grants
// (SRD 5.2.1, "Parts of a Background: Tool Proficiency", p. 83). Variants:
// SpecificToolProficiency, ChoiceToolProficiency. Consumers switch over it
// exhaustively; the variants are immutable, comparable and safe for
// concurrent use.
type ToolProficiency interface{ isToolProficiency() }

// SpecificToolProficiency is proficiency with one named tool. The zero value
// is invalid; build it with [NewSpecificToolProficiency].
type SpecificToolProficiency struct{ name string }

// NewSpecificToolProficiency returns proficiency with the tool called name,
// which must be non-empty with no leading or trailing white space.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName otherwise. It never panics.
func NewSpecificToolProficiency(name string) (SpecificToolProficiency, error) {
	if err := checkName("SpecificToolProficiency", name); err != nil {
		return SpecificToolProficiency{}, err
	}
	return SpecificToolProficiency{name: name}, nil
}

// Name returns the tool name.
func (s SpecificToolProficiency) Name() string { return s.name }

func (SpecificToolProficiency) isToolProficiency() {}

// ChoiceToolProficiency is proficiency with one tool of a category of the
// player's choice, such as a Gaming Set. The zero value is invalid; build it
// with [NewChoiceToolProficiency].
type ChoiceToolProficiency struct{ category string }

// NewChoiceToolProficiency returns a choice among the tools of category,
// which must contain a non-white-space character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText otherwise. It never panics.
func NewChoiceToolProficiency(category string) (ChoiceToolProficiency, error) {
	if err := checkText("ChoiceToolProficiency", category); err != nil {
		return ChoiceToolProficiency{}, err
	}
	return ChoiceToolProficiency{category: category}, nil
}

// Category returns the tool category.
func (c ChoiceToolProficiency) Category() string { return c.category }

func (ChoiceToolProficiency) isToolProficiency() {}
