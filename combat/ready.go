package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ReadyTriggerID names the trigger of a readied action, an id the host chooses. It
// is compared by exact value (no trimming, no case folding). The zero ReadyTriggerID
// is invalid; it is never a telemetry attribute.
type ReadyTriggerID string

// ReadyTriggerIDRejection says why ParseReadyTriggerID rejected a value. The zero
// value is not a rejection.
type ReadyTriggerIDRejection int

// ReadyTriggerEmpty: the value is empty or only whitespace.
const ReadyTriggerEmpty ReadyTriggerIDRejection = 1

// String returns "empty" or "unknown".
func (r ReadyTriggerIDRejection) String() string { return enumName([]string{"empty"}, int(r)) }

// ReadyTriggerIDError reports a rejected ReadyTriggerID. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type ReadyTriggerIDError struct{ Reason ReadyTriggerIDRejection }

// Error renders the error. The text is not a contract.
func (e *ReadyTriggerIDError) Error() string {
	return "combat: invalid ready trigger id: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ReadyTriggerIDError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ReadyTriggerIDError with the same Reason.
func (e *ReadyTriggerIDError) Is(target error) bool {
	t, ok := target.(*ReadyTriggerIDError)
	return ok && t != nil && e != nil && *t == *e
}

// ParseReadyTriggerID returns s unchanged as a ReadyTriggerID. It returns a
// *ReadyTriggerIDError (matching srderr.ErrInvalidInput) with ReadyTriggerEmpty when
// strings.TrimSpace(s) is empty. It never panics.
func ParseReadyTriggerID(s string) (ReadyTriggerID, error) {
	if strings.TrimSpace(s) == "" {
		return "", &ReadyTriggerIDError{ReadyTriggerEmpty}
	}
	return ReadyTriggerID(s), nil
}

// MustReadyTriggerID is ParseReadyTriggerID for literals known to be valid. It
// panics with the error ParseReadyTriggerID returns, and only then.
func MustReadyTriggerID(s string) ReadyTriggerID {
	id, err := ParseReadyTriggerID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// ReadiedResponse is what a readied action does when its trigger fires, a closed
// set. A readied action cannot be a Ready. Variants: ReadiedActionResponse,
// ReadiedMoveResponse.
type ReadiedResponse interface{ isReadiedResponse() }

// ReadiedActionResponse: take the action in Choice as a Reaction.
type ReadiedActionResponse struct{ Choice ReadyableChoice }

// ReadiedMoveResponse: move in response; judged at the trigger, not at Ready. Build it
// with NewReadiedMoveResponse.
type ReadiedMoveResponse struct{ segments []MoveSegment }

func (ReadiedActionResponse) isReadiedResponse() {}
func (ReadiedMoveResponse) isReadiedResponse()   {}

// NewReadiedMoveResponse returns a move response. It returns an error matching
// srderr.ErrInvalidInput for no segment or a zero segment, and a LimitExceededError for more than 256 segments (plan P10). The segments are copied.
// It does not judge whether the move is affordable or legal.
func NewReadiedMoveResponse(segments []MoveSegment) (ReadiedMoveResponse, error) {
	if err := overCap("move segments", len(segments), maxInputList); err != nil {
		return ReadiedMoveResponse{}, err
	}
	if len(segments) == 0 || slices.ContainsFunc(segments, MoveSegment.IsZero) {
		return ReadiedMoveResponse{}, invalidInput("a readied move needs segments")
	}
	return ReadiedMoveResponse{slices.Clone(segments)}, nil
}

// Segments returns a copy of the segments.
func (r ReadiedMoveResponse) Segments() []MoveSegment { return slices.Clone(r.segments) }

// ReadiedSpell is a spell held by a readied action (SRD 5.2.1, "Ready [Action]",
// pp. 186-187). Comparable with ==.
type ReadiedSpell struct {
	Spell   content.SpellID
	Caster  CombatantID
	Trigger ReadyTriggerID
}

// HeldSpellInput says whether a readied action holds a spell, a closed set.
// Variants: HoldsSpell, HoldsNoSpell.
type HeldSpellInput interface{ isHeldSpellInput() }

// HoldsSpell: the readied action holds the spell.
type HoldsSpell struct{ Spell ReadiedSpell }

// HoldsNoSpell: the readied action holds no spell.
type HoldsNoSpell struct{}

func (HoldsSpell) isHeldSpellInput()   {}
func (HoldsNoSpell) isHeldSpellInput() {}

// ReadiedAction is the trigger, the response and the held spell of a Ready. Build it with
// NewReadiedAction; the zero value is invalid. Immutable; compare with Equal.
type ReadiedAction struct {
	trigger  ReadyTriggerID
	response ReadiedResponse
	held     HeldSpellInput
}

// NewReadiedAction returns the readied action. A nil held is HoldsNoSpell. It returns an
// error matching srderr.ErrInvalidInput for an empty trigger, a nil or invalid
// response, or a held spell with an invalid id or caster or a trigger other than
// trigger. It never panics.
func NewReadiedAction(trigger ReadyTriggerID, response ReadiedResponse, held HeldSpellInput) (ReadiedAction, error) {
	if _, err := ParseReadyTriggerID(string(trigger)); err != nil {
		return ReadiedAction{}, err
	}
	switch r := response.(type) {
	case ReadiedActionResponse:
		if r.Choice == nil || !validActionChoice(r.Choice) {
			return ReadiedAction{}, invalidInput("readied action needs a valid choice")
		}
	case ReadiedMoveResponse:
		if len(r.segments) == 0 {
			return ReadiedAction{}, invalidInput("readied move needs segments")
		}
	default:
		return ReadiedAction{}, invalidInput("readied action needs a response")
	}
	switch h := held.(type) {
	case nil:
		held = HoldsNoSpell{}
	case HoldsSpell:
		if !h.Spell.Spell.Valid() || h.Spell.Caster.IsZero() || h.Spell.Trigger != trigger {
			return ReadiedAction{}, invalidInput("held spell needs an id, a caster and the action's trigger")
		}
	}
	return ReadiedAction{trigger, response, held}, nil
}

// IsZero reports whether a is the invalid zero readied action.
func (a ReadiedAction) IsZero() bool { return a.trigger == "" }

// Trigger returns the trigger id.
func (a ReadiedAction) Trigger() ReadyTriggerID { return a.trigger }

// Response returns the response.
func (a ReadiedAction) Response() ReadiedResponse { return a.response }

// Held returns the held spell input; HoldsNoSpell for the zero value.
func (a ReadiedAction) Held() HeldSpellInput {
	if a.held == nil {
		return HoldsNoSpell{}
	}
	return a.held
}

// Equal reports whether a and o are the same readied action.
func (a ReadiedAction) Equal(o ReadiedAction) bool {
	if a.trigger != o.trigger || a.Held() != o.Held() {
		return false
	}
	switch r := a.response.(type) {
	case ReadiedActionResponse:
		p, ok := o.response.(ReadiedActionResponse)
		return ok && r == p
	case ReadiedMoveResponse:
		p, ok := o.response.(ReadiedMoveResponse)
		return ok && segmentsEqual(r.segments, p.segments)
	}
	return o.response == nil
}
