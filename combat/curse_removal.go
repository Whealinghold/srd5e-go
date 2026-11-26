package combat

import "github.com/Whealinghold/srd5e-go/content"

// CurseRemovalMeans is what ended a curse: the Remove Curse spell, the Greater
// Restoration spell, or other magic that explicitly ends curses (SRD 5.2.1,
// "Curses", p. 179). The zero value is not a means.
type CurseRemovalMeans int

// The means of removing a curse.
const (
	// RemoveCurseSpell: the Remove Curse spell.
	RemoveCurseSpell CurseRemovalMeans = iota + 1
	// GreaterRestorationSpell: the Greater Restoration spell.
	GreaterRestorationSpell
	// OtherCurseEndingMagic: other magic that explicitly ends curses.
	OtherCurseEndingMagic
)

// String returns "remove_curse", "greater_restoration", "other_magic" or "unknown".
func (m CurseRemovalMeans) String() string {
	return enumName([]string{"remove_curse", "greater_restoration", "other_magic"}, int(m))
}

// CurseRemovalMeansFromSpell classifies a spell as a means of removing curses:
// RemoveCurseSpell for remove_curse, GreaterRestorationSpell for
// greater_restoration, and the zero value with false for any other spell (SRD
// 5.2.1, "Curses", p. 179). It never panics.
func CurseRemovalMeansFromSpell(id content.SpellID) (CurseRemovalMeans, bool) {
	switch id {
	case "remove_curse":
		return RemoveCurseSpell, true
	case "greater_restoration":
		return GreaterRestorationSpell, true
	}
	return 0, false
}

// MagicThatEndsCurses returns OtherCurseEndingMagic for a magical source: "other
// magic that explicitly ends curses" (SRD 5.2.1, "Curses", p. 179). It returns the
// zero value and an error matching srderr.ErrInvalidInput when the source is not
// magical (including the zero source). It never panics.
func MagicThatEndsCurses(source EffectSource) (CurseRemovalMeans, error) {
	if !source.IsMagical() {
		return 0, invalidInput("only a magical source can end curses")
	}
	return OtherCurseEndingMagic, nil
}

// RemoveCombatantCurseResult is the closed set of answers of CurseRemoval.Remove. Every
// variant reports the creature's OngoingEffects after the call. Variants:
// RemoveCurseApplied, RemoveCurseNotPresent, RemoveCurseNotACurse.
type RemoveCombatantCurseResult interface {
	isRemoveCombatantCurseResult()
	Effects() OngoingEffects
}

// RemoveCurseApplied: the curse was removed. Effects lacks it and keeps every other
// entry in order; Events holds one CurseRemoved.
type RemoveCurseApplied struct {
	effects OngoingEffects
	events  []MagicalEffectEvent
}

// RemoveCurseNotPresent: the creature has no effect with that id; nothing changed.
type RemoveCurseNotPresent struct{ effects OngoingEffects }

// RemoveCurseNotACurse: the effect with that id is not a curse; nothing changed.
type RemoveCurseNotACurse struct {
	effects  OngoingEffects
	category OngoingEffectCategory
}

// Effects returns the effects after the call.
func (r RemoveCurseApplied) Effects() OngoingEffects { return r.effects }

// Events returns a copy of the events: one CurseRemoved.
func (r RemoveCurseApplied) Events() []MagicalEffectEvent { return cloneMagicalEvents(r.events) }

// Effects returns the effects, unchanged.
func (r RemoveCurseNotPresent) Effects() OngoingEffects { return r.effects }

// Effects returns the effects, unchanged.
func (r RemoveCurseNotACurse) Effects() OngoingEffects { return r.effects }

// Category returns the category of the effect that was named.
func (r RemoveCurseNotACurse) Category() OngoingEffectCategory { return r.category }

func (RemoveCurseApplied) isRemoveCombatantCurseResult()    {}
func (RemoveCurseNotPresent) isRemoveCombatantCurseResult() {}
func (RemoveCurseNotACurse) isRemoveCombatantCurseResult()  {}

// CurseRemoval is the stateless service that removes a curse from a creature's
// effects. The zero value is ready to use. Safe for concurrent use.
type CurseRemoval struct{}

// Remove removes the curse named id from the creature's effects (SRD 5.2.1,
// "Curses", p. 179). The engine applies no curse behaviour: only the id and the
// category are read. An id not in the list is NotPresent, an effect that is not a
// curse is NotACurse, and neither changes anything or emits an event. A zero
// creature or an undefined means changes nothing and gives NotPresent. It never
// panics.
func (CurseRemoval) Remove(creature CombatantID, effects OngoingEffects, id OngoingEffectID, means CurseRemovalMeans) RemoveCombatantCurseResult {
	e, ok := effects.Find(id)
	switch {
	case !ok || creature.IsZero() || means < RemoveCurseSpell || means > OtherCurseEndingMagic:
		return RemoveCurseNotPresent{effects}
	case e.Category() != OngoingEffectCurse:
		return RemoveCurseNotACurse{effects, e.Category()}
	}
	return RemoveCurseApplied{effects.without([]OngoingEffectID{id}), []MagicalEffectEvent{CurseRemoved{magicalEffectEventBase{creature}, id, means}}}
}
