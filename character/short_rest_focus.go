package character

import "github.com/Whealinghold/srd5e-go/srderr"

// RestPurpose is what a Short Rest focused on an item is for. The zero value is
// invalid.
type RestPurpose int

// The purposes of a Short Rest focused on one item (SRD 5.2.1, "Magic Items", p. 102).
const (
	RestPurposeLearnProperties RestPurpose = iota + 1
	RestPurposeAttune
	RestPurposeEndAttunement
)

// Valid reports whether p is a purpose.
func (p RestPurpose) Valid() bool {
	return p >= RestPurposeLearnProperties && p <= RestPurposeEndAttunement
}

// PhysicalContact is whether the creature touched the item during the rest.
// The zero value is invalid.
type PhysicalContact int

// Physical contact with the item.
const (
	InContact PhysicalContact = iota + 1
	NotInContact
)

// Valid reports whether c is a contact value.
func (c PhysicalContact) Valid() bool { return c == InContact || c == NotInContact }

// ShortRestFocusReason says why NewShortRestFocus refused a rest report. The
// zero value is not a reason.
type ShortRestFocusReason int

// The reasons a rest report is not a Short Rest focus.
const (
	ShortRestFocusKindResolutionMismatch ShortRestFocusReason = iota + 1
	ShortRestFocusNotAShortRest
	ShortRestFocusRestReportRejected
)

// String returns the snake_case name of the reason, or "unknown".
func (r ShortRestFocusReason) String() string {
	names := []string{"kind_resolution_mismatch", "not_a_short_rest", "rest_report_rejected"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// ShortRestFocusRejected is the error NewShortRestFocus returns for a rest that
// cannot be a Short Rest focus. It matches srderr.ErrInvalidInput.
//
//nolint:errname // the name is the spec's (FR-119); it is an error with typed data
type ShortRestFocusRejected struct{ Reason ShortRestFocusReason }

// Error renders the error. The text is not a contract.
func (e *ShortRestFocusRejected) Error() string {
	return "character: short rest focus: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ShortRestFocusRejected) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ShortRestFocusRejected with the same Reason.
func (e *ShortRestFocusRejected) Is(target error) bool {
	t, ok := target.(*ShortRestFocusRejected)
	return ok && t != nil && e != nil && *t == *e
}

// ShortRestFocus is a Short Rest spent focused on one item: which item, why,
// whether in contact, and whether the rest was completed. Immutable and
// comparable; the zero value is invalid.
type ShortRestFocus struct {
	instance    MagicItemInstanceID
	purpose     RestPurpose
	contact     PhysicalContact
	isCompleted bool
}

// NewShortRestFocus builds the focus from a rest's kind and resolution, applying
// in order: a RestCompleted of another kind, or a Short Rest with benefits of a
// Long Rest, is ShortRestFocusKindResolutionMismatch; any Long Rest is
// ShortRestFocusNotAShortRest; a RestReportRejected is ShortRestFocusRestReportRejected;
// a Short Rest then is completed only when RestCompleted (SRD 5.2.1,
// "Attune during a Short Rest", p. 102).
//
// It returns a *ShortRestFocusRejected for those cases and an error matching
// srderr.ErrInvalidInput for a blank instance, an invalid purpose, contact or
// kind, or a nil or foreign resolution. It never panics.
func NewShortRestFocus(instance MagicItemInstanceID, purpose RestPurpose, contact PhysicalContact, kind RestKind, resolution RestResolution) (ShortRestFocus, error) {
	if !instance.Valid() || !purpose.Valid() || !contact.Valid() || !kind.Valid() {
		return ShortRestFocus{}, invalidInput("short rest focus: blank instance or invalid purpose, contact or kind")
	}
	mismatch := &ShortRestFocusRejected{Reason: ShortRestFocusKindResolutionMismatch}
	completed := false
	switch r := resolution.(type) {
	case RestCompleted:
		if r.Kind() != kind {
			return ShortRestFocus{}, mismatch
		}
		completed = true
	case RestStoppedWithShortRestBenefits:
		if kind == ShortRest {
			return ShortRestFocus{}, mismatch
		}
	case RestStoppedWithoutBenefits, RestIncomplete, RestReportRejected:
	default:
		return ShortRestFocus{}, invalidInput("short rest focus: nil or foreign rest resolution")
	}
	if kind == LongRest {
		return ShortRestFocus{}, &ShortRestFocusRejected{Reason: ShortRestFocusNotAShortRest}
	}
	if _, bad := resolution.(RestReportRejected); bad {
		return ShortRestFocus{}, &ShortRestFocusRejected{Reason: ShortRestFocusRestReportRejected}
	}
	return ShortRestFocus{instance: instance, purpose: purpose, contact: contact, isCompleted: completed}, nil
}

// InstanceID returns the item the rest focused on.
func (f ShortRestFocus) InstanceID() MagicItemInstanceID { return f.instance }

// Purpose returns what the rest was for.
func (f ShortRestFocus) Purpose() RestPurpose { return f.purpose }

// Contact returns whether the creature touched the item.
func (f ShortRestFocus) Contact() PhysicalContact { return f.contact }

// IsCompleted reports whether the Short Rest was finished, not interrupted.
func (f ShortRestFocus) IsCompleted() bool { return f.isCompleted }

// IsZero reports whether f is the invalid zero ShortRestFocus.
func (f ShortRestFocus) IsZero() bool { return !f.instance.Valid() }
