package combat

import (
	"slices"
	"time"
)

// SaveImmunityWindow is the time during which one creature is immune to one part of another
// creature after succeeding on its saving throw (SRD 5.2.1, "Ghost", p. 288: "immune to this
// ghost's Horrific Visage for 24 hours"). It names the individual source creature and the part,
// so another creature of the same kind, or another part of the same creature, is not covered
// (monster-stat-block-actions.md FR-039). Comparable and safe for concurrent use. The zero value
// is invalid; build one with NewSaveImmunityWindow.
type SaveImmunityWindow struct {
	target, source CombatantID
	part           LimitedUsePartRef
	remaining      time.Duration
}

// NewSaveImmunityWindow returns the window in which target is immune to part of source for
// remaining more time.
//
// It returns an error matching srderr.ErrInvalidInput for a zero or equal target and source, a zero
// part, or a remaining that is not positive. It never panics.
func NewSaveImmunityWindow(target, source CombatantID, part LimitedUsePartRef, remaining time.Duration) (SaveImmunityWindow, error) {
	if target.IsZero() || source.IsZero() || target == source || part.IsZero() || remaining <= 0 {
		return SaveImmunityWindow{}, invalidInput("an immunity window needs distinct creatures, a part and a positive duration")
	}
	return SaveImmunityWindow{target, source, part, remaining}, nil
}

// Combatants returns a new slice holding the creature that is immune, then the creature whose part
// it is immune to.
func (w SaveImmunityWindow) Combatants() []CombatantID { return []CombatantID{w.target, w.source} }

// Part returns the part the target is immune to.
func (w SaveImmunityWindow) Part() LimitedUsePartRef { return w.part }

// Remaining returns how long the immunity still lasts.
func (w SaveImmunityWindow) Remaining() time.Duration { return w.remaining }

// IsZero reports whether w is the invalid zero window.
func (w SaveImmunityWindow) IsZero() bool { return w == SaveImmunityWindow{} }

// sameTriple reports whether w and o are about the same target, source and part.
func (w SaveImmunityWindow) sameTriple(o SaveImmunityWindow) bool {
	return w.target == o.target && w.source == o.source && w.part == o.part
}

// SaveImmunities are the immunity windows a combat has granted, in the order they were granted.
// The engine reads no clock: the host reports elapsed time with Elapse, and a rest is not a reset
// (FR-041). Every operation returns a new value and leaves the receiver unchanged. It holds a
// slice, so compare it with Equal. The zero value holds no window and is ready to use. Safe for
// concurrent use.
type SaveImmunities struct{ windows []SaveImmunityWindow }

// Windows returns the windows in the order they were granted, as a new slice.
func (s SaveImmunities) Windows() []SaveImmunityWindow { return slices.Clone(s.windows) }

// Equal reports whether s and o hold the same windows in the same order.
func (s SaveImmunities) Equal(o SaveImmunities) bool { return slices.Equal(s.windows, o.windows) }

// IsImmune reports whether target is immune to part of source: a window for exactly that
// triple has time remaining (FR-040).
func (s SaveImmunities) IsImmune(target, source CombatantID, part LimitedUsePartRef) bool {
	return slices.ContainsFunc(s.windows, func(w SaveImmunityWindow) bool {
		return w.target == target && w.source == source && w.part == part && w.remaining > 0
	})
}

// SaveImmunityTransition is the change of a grant or of elapsed time: the new immunities and
// the events. Immutable; every accessor returns a copy.
type SaveImmunityTransition struct {
	immunities SaveImmunities
	events     []StatBlockActionEvent
}

// Immunities returns the immunities after the change.
func (t SaveImmunityTransition) Immunities() SaveImmunities {
	return SaveImmunities{slices.Clone(t.immunities.windows)}
}

// Events returns the events in order, as a new slice.
func (t SaveImmunityTransition) Events() []StatBlockActionEvent { return slices.Clone(t.events) }

// Grant adds w, or, when a window for the same target, source and part exists, keeps one window
// with the larger remaining time; windows never add (FR-042). It produces one SaveImmunityGranted
// event carrying the remaining time of the window now held. A zero w changes nothing and produces
// no event.
func (s SaveImmunities) Grant(w SaveImmunityWindow) SaveImmunityTransition {
	if w.IsZero() {
		return SaveImmunityTransition{immunities: s}
	}
	windows := slices.Clone(s.windows)
	if i := slices.IndexFunc(windows, w.sameTriple); i >= 0 {
		w.remaining = max(w.remaining, windows[i].remaining)
		windows[i] = w
	} else {
		windows = append(windows, w)
	}
	return SaveImmunityTransition{SaveImmunities{windows}, []StatBlockActionEvent{
		SaveImmunityGranted{statBlockActionEventBase{w.source, w.target}, w.part, w.remaining}}}
}

// Elapse reduces every window by d, drops each window that reaches zero or below and produces one
// SaveImmunityExpired per dropped window, in the order the windows were granted (FR-041). A window
// of 24 hours is still immune after 23 hours 59 minutes and not after 24 hours. Zero changes
// nothing.
//
// It returns an error matching srderr.ErrInvalidInput for a negative d. It never panics.
func (s SaveImmunities) Elapse(d time.Duration) (SaveImmunityTransition, error) {
	if d < 0 {
		return SaveImmunityTransition{}, invalidInput("elapsed time cannot be negative")
	}
	var kept []SaveImmunityWindow
	var events []StatBlockActionEvent
	for _, w := range s.windows {
		if w.remaining -= d; w.remaining > 0 {
			kept = append(kept, w)
		} else {
			events = append(events, SaveImmunityExpired{statBlockActionEventBase{w.source, w.target}, w.part})
		}
	}
	return SaveImmunityTransition{SaveImmunities{kept}, events}, nil
}
