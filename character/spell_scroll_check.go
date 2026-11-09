package character

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ErrScrollCheckMismatch is returned when a CheckResult does not answer the check
// that was required: a different kind, a different target, or an option that was
// not offered. It matches srderr.ErrInvalidInput with errors.Is.
var ErrScrollCheckMismatch = fmt.Errorf("character: scroll check does not match the requirement: %w", srderr.ErrInvalidInput)

// ScrollDisposition is what happens to a scroll after a reading attempt. The zero
// value is invalid.
type ScrollDisposition int

// The dispositions. DispositionConsumed and DispositionErased both mean the host
// consumes the scroll (the MagicItemConsumed event); DispositionRetained leaves it
// intact.
const (
	// DispositionRetained: the scroll is kept and stays ItemIntact.
	DispositionRetained ScrollDisposition = iota + 1
	// DispositionConsumed: the spell was cast and the scroll crumbles to dust.
	DispositionConsumed
	// DispositionErased: the check failed and the spell disappears from the scroll.
	DispositionErased
)

// String returns "retained", "consumed", "erased" or "unknown".
func (d ScrollDisposition) String() string {
	names := [...]string{"retained", "consumed", "erased"}
	if d < DispositionRetained || d > DispositionErased {
		return "unknown"
	}
	return names[d-1]
}

// ScrollCastProgress is how far a scroll's casting got. The zero value is invalid.
type ScrollCastProgress int

// The progress values. The host derives them from the cast result: a SpellCast event
// is ProgressCompleted, a CastingFailed event ProgressInterrupted, and a rejection
// or a casting never begun ProgressNotStarted.
const (
	// ProgressNotStarted: the cast was rejected or never begun.
	ProgressNotStarted ScrollCastProgress = iota + 1
	// ProgressInterrupted: the casting began and was interrupted.
	ProgressInterrupted
	// ProgressCompleted: the casting completed and the spell took effect.
	ProgressCompleted
)

// String returns "not_started", "interrupted", "completed" or "unknown".
func (p ScrollCastProgress) String() string {
	names := [...]string{"not_started", "interrupted", "completed"}
	if p < ProgressNotStarted || p > ProgressCompleted {
		return "unknown"
	}
	return names[p-1]
}

// ScrollCheckResolutionResult is the closed set of outcomes of ResolveScrollCheck.
// Variants: ScrollCheckPassed, ScrollSpellErased.
type ScrollCheckResolutionResult interface{ isScrollCheckResolutionResult() }

// ScrollCheckPassed: the check succeeded; the reader casts with Terms.
type ScrollCheckPassed struct{ terms ScrollCastTerms }

// Terms returns the terms of the cast, whose ability is that of the chosen option.
func (r ScrollCheckPassed) Terms() ScrollCastTerms { return r.terms }

// ScrollSpellErased: the check failed and the spell disappears from the scroll with
// no other effect: no spell is cast, no slot is spent and no Concentration begins.
type ScrollSpellErased struct{}

// Disposition returns DispositionErased.
func (ScrollSpellErased) Disposition() ScrollDisposition { return DispositionErased }

func (ScrollCheckPassed) isScrollCheckResolutionResult() {}
func (ScrollSpellErased) isScrollCheckResolutionResult() {}

// ResolveScrollCheck settles the reader's ability check for option (SRD 5.2.1,
// "Spell Scroll", p. 244; "Level 13: Use Magic Device", p. 64). A total equal to or
// above the DC gives ScrollCheckPassed with the terms of option; a lower one gives
// ScrollSpellErased. The host MUST roll the option it passes: a CheckResult has no
// ability or skill, so only its kind and target are validated.
//
// It returns a nil result and an error matching ErrScrollCheckMismatch (and so
// srderr.ErrInvalidInput) when option is not one of required's options, the result is
// not an ability check, or its target differs from required's DC. It never panics.
func ResolveScrollCheck(required ScrollCheckRequired, option ScrollCheckOption, result d20.CheckResult) (ScrollCheckResolutionResult, error) {
	terms, offered := required.TermsFor(option)
	if !offered || result.Kind() != d20.AbilityCheck || result.Target() != d20.DifficultyClass(required.dc) {
		return nil, ErrScrollCheckMismatch
	}
	if result.Outcome() != d20.Success {
		return ScrollSpellErased{}, nil
	}
	return ScrollCheckPassed{terms: terms}, nil
}

// SettleScrollReading returns what happens to the scroll: DispositionConsumed once the
// spell is cast, DispositionRetained for an interrupted casting and for one never
// started (SRD 5.2.1, "Spell Scroll", p. 244). A failed check is ScrollSpellErased,
// not a progress value.
func SettleScrollReading(progress ScrollCastProgress) ScrollDisposition {
	if progress == ProgressCompleted {
		return DispositionConsumed
	}
	return DispositionRetained
}
