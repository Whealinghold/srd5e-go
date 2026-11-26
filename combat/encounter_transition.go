package combat

import "slices"

// encounterTransition is the closed domain-level outcome of an Encounter command; each
// use case maps it to its own result. Variants: transitionApplied, transitionRejected.
type encounterTransition interface {
	isEncounterTransition()
	// Encounter returns the new encounter, or the unchanged one for a rejection.
	Encounter() Encounter
	// Events returns a copy of the events in the order they happened; none for a rejection.
	Events() []Event
	// Resolutions returns a copy of the damage resolutions: one per Damage of an
	// ApplyDamage, none otherwise.
	Resolutions() []DamageResolution
	// Rejection returns why the command was refused, or nil when it was applied.
	Rejection() EncounterRejection
}

// transitionApplied: the command took effect (possibly changing nothing).
type transitionApplied struct {
	enc         Encounter
	events      []Event
	resolutions []DamageResolution
}

// transitionRejected: the command was refused; the encounter is unchanged.
type transitionRejected struct {
	enc       Encounter
	rejection EncounterRejection
}

func (transitionApplied) isEncounterTransition()  {}
func (transitionRejected) isEncounterTransition() {}

func (t transitionApplied) Encounter() Encounter  { return t.enc }
func (t transitionRejected) Encounter() Encounter { return t.enc }

func (t transitionApplied) Events() []Event { return slices.Clone(t.events) }
func (transitionRejected) Events() []Event  { return nil }
func (t transitionApplied) Resolutions() []DamageResolution {
	return slices.Clone(t.resolutions)
}
func (transitionRejected) Resolutions() []DamageResolution { return nil }

func (transitionApplied) Rejection() EncounterRejection    { return nil }
func (t transitionRejected) Rejection() EncounterRejection { return t.rejection }

// reject is the refusal of a command: the encounter, unchanged.
func (e Encounter) reject(r EncounterRejection) encounterTransition {
	return transitionRejected{e, r}
}

// accept is an applied transition with the events given.
func (e Encounter) accept(events []Event) encounterTransition {
	return transitionApplied{enc: e, events: events}
}

// target is the prelude of every command naming a combatant (FR-078 steps 1 to 3). It
// returns an error matching srderr.ErrInvalidInput for an invalid encounter or a zero
// id; otherwise the combatant's position and the rejection (nil when none): ended,
// unknown combatant, dead.
func (e Encounter) target(id CombatantID) (int, EncounterRejection, error) {
	if id.IsZero() || !e.usable() {
		return 0, nil, invalidInput("a command needs a valid encounter and a combatant id")
	}
	i := e.index(id)
	switch {
	case e.ended():
		return i, EncounterAlreadyEndedRejection{}, nil
	case i < 0:
		return i, UnknownCombatantRejection{id}, nil
	case e.combatants[i].hp.isDead():
		return i, CombatantIsDeadRejection{id}, nil
	}
	return i, nil, nil
}
