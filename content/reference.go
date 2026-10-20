package content

import "github.com/Whealinghold/srd5e-go/srderr"

// ReferenceConstraint is a condition on the target of a [Reference], beyond
// the target existing (schema FR-126). It is a closed family of comparable
// values with exactly one variant today, [TargetHasFeatCategory]; a further
// variant needs a revision-table row. The nil constraint means existence
// only.
type ReferenceConstraint interface{ isReferenceConstraint() }

// TargetHasFeatCategory is satisfied by a target that implements
// [HasFeatCategory] and whose category equals Category. Construct it with a
// keyed literal. A zero Category is never satisfied.
type TargetHasFeatCategory struct{ Category FeatCategory }

func (TargetHasFeatCategory) isReferenceConstraint() {}

// satisfiedBy reports whether the target entry meets the constraint.
func (c TargetHasFeatCategory) satisfiedBy(target Entry) bool {
	h, ok := target.(HasFeatCategory)
	return ok && c.Category.Valid() && h.FeatCategory() == c.Category
}

// HasFeatCategory is implemented by an entry that has a feat category, so a
// [TargetHasFeatCategory] constraint can be checked against it. the feat
// entry implements it.
type HasFeatCategory interface{ FeatCategory() FeatCategory }

// ReferenceProblemKind says why [NewReference] rejected its arguments. The
// zero value is not a rejection.
type ReferenceProblemKind int

// The reference rejection reasons, in the order they are checked.
const (
	// ReferenceBlankField: the field path is blank.
	ReferenceBlankField ReferenceProblemKind = iota + 1
	// ReferenceInvalidTarget: the target key has an invalid kind or the zero id.
	ReferenceInvalidTarget
	// ReferenceInvalidConstraint: the constraint is not a valid variant (a
	// pointer, or a category that is not a member).
	ReferenceInvalidConstraint
)

// String returns the lower camel case name of k, or "unknown" for an
// undefined value.
func (k ReferenceProblemKind) String() string {
	switch k {
	case ReferenceBlankField:
		return "blankField"
	case ReferenceInvalidTarget:
		return "invalidTarget"
	case ReferenceInvalidConstraint:
		return "invalidConstraint"
	}
	return unknownName
}

// ReferenceError reports arguments rejected by [NewReference]. It matches
// srderr.ErrInvalidInput with errors.Is, and another *ReferenceError with the
// same Problem.
type ReferenceError struct {
	Problem ReferenceProblemKind
}

// Error renders the problem. The text is not a contract.
func (e *ReferenceError) Error() string {
	return "content: invalid reference: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ReferenceError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ReferenceError (not a nil one) with the same
// Problem.
func (e *ReferenceError) Is(target error) bool {
	t, ok := target.(*ReferenceError)
	return ok && t != nil && t.Problem == e.Problem
}

// Reference is one id inside an entry that names another entry, and the
// condition its target must meet: a site of the revision table (schema
// FR-126). It is an immutable value; the zero Reference names no valid
// target, so [ValidateSet] reports it unresolved.
type Reference struct {
	field      string
	target     Key
	constraint ReferenceConstraint
}

// NewReference returns the reference at field (a path inside the owning
// entry, such as "feat/featId") to target, which must satisfy constraint when
// that is not nil.
//
// It returns a *[ReferenceError] wrapping srderr.ErrInvalidInput when field is
// blank, target is not a valid key, or constraint is not a valid
// [TargetHasFeatCategory] value. It never panics.
func NewReference(field string, target Key, constraint ReferenceConstraint) (Reference, error) {
	switch {
	case blankText(field):
		return Reference{}, &ReferenceError{Problem: ReferenceBlankField}
	case !target.valid():
		return Reference{}, &ReferenceError{Problem: ReferenceInvalidTarget}
	case !validConstraint(constraint):
		return Reference{}, &ReferenceError{Problem: ReferenceInvalidConstraint}
	}
	return Reference{field: field, target: target, constraint: constraint}, nil
}

// validConstraint reports whether c is nil or a valid variant value.
func validConstraint(c ReferenceConstraint) bool {
	switch v := c.(type) {
	case nil:
		return true
	case TargetHasFeatCategory:
		return v.Category.Valid()
	}
	return false
}

// MustReference is like [NewReference] but panics with its error when the
// arguments are invalid.
func MustReference(field string, target Key, constraint ReferenceConstraint) Reference {
	r, err := NewReference(field, target, constraint)
	if err != nil {
		panic(err)
	}
	return r
}

// Field returns the path of the id inside the owning entry.
func (r Reference) Field() string { return r.field }

// Target returns the key of the entry the reference names.
func (r Reference) Target() Key { return r.target }

// Constraint returns the condition on the target, or nil for existence only.
func (r Reference) Constraint() ReferenceConstraint { return r.constraint }

// References is implemented by an entry that holds ids of other entries. An
// entry that does not implement it has no reference sites. An entry type
// lists exactly the sites its revision row registers (schema FR-125); the
// returned slice is the caller's to keep.
type References interface{ References() []Reference }
