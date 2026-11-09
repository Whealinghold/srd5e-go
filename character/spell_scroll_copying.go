package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ScrollNotCopyableReason says why a scroll cannot be copied into a spellbook. The
// zero value is invalid.
type ScrollNotCopyableReason int

// The reasons.
const (
	// NotAWizardSpell: the spell is not on the Wizard spell list.
	NotAWizardSpell ScrollNotCopyableReason = iota + 1
	// Cantrip: a cantrip is not copied into a spellbook (interpretation: only level 1+ spells are).
	Cantrip
)

// String returns "not_a_wizard_spell", "cantrip" or "unknown".
func (r ScrollNotCopyableReason) String() string {
	names := [...]string{"not_a_wizard_spell", "cantrip"}
	if r < NotAWizardSpell || r > Cantrip {
		return "unknown"
	}
	return names[r-1]
}

// ScrollCopyAssessmentResult is the closed set of outcomes of AssessScrollCopying.
// Variants: ScrollCopyCheckRequired, ScrollNotCopyable, ScrollAlreadyUsed.
type ScrollCopyAssessmentResult interface{ isScrollCopyAssessmentResult() }

// ScrollCopyCheckRequired: the scroll may be copied after an Intelligence (Arcana)
// check against DC, 10 plus the spell level.
type ScrollCopyCheckRequired struct{ dc int }

// DC returns 10 plus the spell level.
func (r ScrollCopyCheckRequired) DC() int { return r.dc }

// Option returns the fixed check: Intelligence (Arcana).
func (ScrollCopyCheckRequired) Option() ScrollCheckOption {
	return ScrollCheckOption{ability: ability.Intelligence, skill: content.SkillArcana}
}

// ScrollNotCopyable: the scroll cannot be copied; nothing is destroyed and the scroll
// is retained.
type ScrollNotCopyable struct{ Reason ScrollNotCopyableReason }

func (ScrollCopyCheckRequired) isScrollCopyAssessmentResult() {}
func (ScrollNotCopyable) isScrollCopyAssessmentResult()       {}
func (ScrollAlreadyUsed) isScrollCopyAssessmentResult()       {}

// AssessScrollCopying decides whether a Wizard may try to copy scroll into a
// spellbook (SRD 5.2.1, "Copying a Scroll into a Spellbook", p. 244). A consumed
// scroll gives ScrollAlreadyUsed; a spell not on the Wizard list gives
// ScrollNotCopyable(NotAWizardSpell); a cantrip gives ScrollNotCopyable(Cantrip). The
// Wizard's own conditions for copying are not repeated here. It mutates nothing and
// never panics.
func AssessScrollCopying(scroll SpellScroll, state MagicItemUseState) ScrollCopyAssessmentResult {
	switch {
	case state == ItemConsumed:
		return ScrollAlreadyUsed{}
	case !slices.Contains(scroll.classes, content.ClassID("wizard")):
		return ScrollNotCopyable{Reason: NotAWizardSpell}
	case scroll.level == 0:
		return ScrollNotCopyable{Reason: Cantrip}
	}
	return ScrollCopyCheckRequired{dc: 10 + int(scroll.level)}
}

// ScrollCopyResolved is the outcome of the copy check: whether the spell was copied
// and that the scroll is destroyed either way. Comparable. The zero value is not an
// outcome.
type ScrollCopyResolved struct{ copied bool }

// SpellCopied reports whether the check succeeded.
func (r ScrollCopyResolved) SpellCopied() bool { return r.copied }

// ScrollDestroyed is always true: the scroll is destroyed whether or not the check
// succeeds.
func (ScrollCopyResolved) ScrollDestroyed() bool { return true }

// ResolveScrollCopy settles the copy check (SRD 5.2.1, "Copying a Scroll into a
// Spellbook", p. 244): the spell is copied exactly when the total equals or exceeds the
// DC, and the scroll is destroyed either way.
//
// It returns the zero outcome and an error matching ErrScrollCheckMismatch when the
// result is not an ability check or its target differs from required's DC. It never
// panics.
func ResolveScrollCopy(required ScrollCopyCheckRequired, result d20.CheckResult) (ScrollCopyResolved, error) {
	if result.Kind() != d20.AbilityCheck || result.Target() != d20.DifficultyClass(required.dc) {
		return ScrollCopyResolved{}, ErrScrollCheckMismatch
	}
	return ScrollCopyResolved{copied: result.Outcome() == d20.Success}, nil
}
