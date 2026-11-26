package combat

import "github.com/Whealinghold/srd5e-go/d20"

// HidingEvent is the closed family of events the Hide rules produce. Each is an
// immutable value comparable with ==, about the hider (its Combatants is exactly
// [hider]) with a constant snake_case Kind. Variants: CreatureHid, HidingEnded.
// Each embeds hidingEventBase. There is no id accessor (no function of this
// package returns a CombatantID); read the hider from Combatants.
type HidingEvent interface {
	Event
	hidingEvent()
}

// hidingEventBase carries the hider and puts the event in the Event and HidingEvent families.
type hidingEventBase struct{ hider CombatantID }

func (hidingEventBase) event()       {}
func (hidingEventBase) hidingEvent() {}

// Combatants returns a new slice [hider].
func (b hidingEventBase) Combatants() []CombatantID { return []CombatantID{b.hider} }

// CreatureHid: the creature hid with a check total that others must meet to find
// it. Its Kind is creature_hid.
type CreatureHid struct {
	hidingEventBase
	findDC d20.DifficultyClass
}

// NewCreatureHid returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank hider or a find DC below the Hide DC of 15.
func NewCreatureHid(hider CombatantID, findDC d20.DifficultyClass) (CreatureHid, error) {
	if hider.IsZero() {
		return CreatureHid{}, invalidInput("hiding event needs a hider")
	}
	if findDC < hideDC {
		return CreatureHid{}, invalidInput("find DC below the Hide DC")
	}
	return CreatureHid{hidingEventBase{hider}, findDC}, nil
}

// Kind returns "creature_hid".
func (CreatureHid) Kind() string { return "creature_hid" }

// FindDC returns the DC for a creature to find the hider.
func (e CreatureHid) FindDC() d20.DifficultyClass { return e.findDC }

// HidingEnded: the creature stopped being hidden because of the activity. Its
// Kind is hiding_ended.
type HidingEnded struct {
	hidingEventBase
	activity HidingActivity
}

// NewHidingEnded returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank hider or an activity that is not one of the seven.
func NewHidingEnded(hider CombatantID, activity HidingActivity) (HidingEnded, error) {
	if hider.IsZero() {
		return HidingEnded{}, invalidInput("hiding event needs a hider")
	}
	if !activity.Valid() {
		return HidingEnded{}, invalidInput("hiding ended needs an activity")
	}
	return HidingEnded{hidingEventBase{hider}, activity}, nil
}

// Kind returns "hiding_ended".
func (HidingEnded) Kind() string { return "hiding_ended" }

// Activity returns what ended the hiding.
func (e HidingEnded) Activity() HidingActivity { return e.activity }
