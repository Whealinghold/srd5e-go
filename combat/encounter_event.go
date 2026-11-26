package combat

import "slices"

// EncounterEvent is the closed family of events an Encounter produces. Each is an
// immutable past-tense value carrying the EncounterID and the round it happened in,
// with a constant snake_case Kind and a Combatants list. EncounterStarted,
// EncounterEnded and DyingCombatantTurnStarted are the encounter's own facts. The
// four ...Occurred types are transport envelopes: each carries one event of a
// sibling spec unchanged and reports the wrapped event's Kind and Combatants, so
// srd5e.combat.events counts the wrapped kind. Events carrying a slice are not
// comparable with ==. Each embeds encounterEventBase. Variants: EncounterStarted,
// EncounterEnded, DyingCombatantTurnStarted, TurnEventOccurred, HitPointEventOccurred,
// ConditionEventOccurred, GrappleEventOccurred.
type EncounterEvent interface {
	Event
	encounterEvent()
	// EncounterID returns the id of the encounter the event happened in.
	EncounterID() EncounterID
	// Round returns the round the event happened in, 1 or more.
	Round() int
}

// encounterEventBase puts an event in the Event and EncounterEvent families.
type encounterEventBase struct {
	id    EncounterID
	round int
}

func (encounterEventBase) event()          {}
func (encounterEventBase) encounterEvent() {}

// EncounterID returns the id of the encounter.
func (b encounterEventBase) EncounterID() EncounterID { return b.id }

// Round returns the round, 1 or more.
func (b encounterEventBase) Round() int { return b.round }

// EncounterStarted: the encounter began. Kind encounter_started; Combatants lists
// every combatant in turn order.
type EncounterStarted struct {
	encounterEventBase
	ids []CombatantID
}

// Kind returns "encounter_started".
func (EncounterStarted) Kind() string { return "encounter_started" }

// Combatants returns a copy of every combatant id, in turn order.
func (e EncounterStarted) Combatants() []CombatantID { return slices.Clone(e.ids) }

// EncounterEnded: the encounter ended. Kind encounter_ended; Combatants lists every
// combatant in turn order.
type EncounterEnded struct {
	encounterEventBase
	ids    []CombatantID
	reason EndReason
}

// Kind returns "encounter_ended".
func (EncounterEnded) Kind() string { return "encounter_ended" }

// Combatants returns a copy of every combatant id, in turn order.
func (e EncounterEnded) Combatants() []CombatantID { return slices.Clone(e.ids) }

// Reason returns why the encounter ended.
func (e EncounterEnded) Reason() EndReason { return e.reason }

// DyingCombatantTurnStarted: the turn of a Dying combatant began, so a Death Saving
// Throw is due. It states a fact, not a prompt: the host decides to roll. Kind
// dying_combatant_turn_started.
type DyingCombatantTurnStarted struct {
	encounterEventBase
	combatant CombatantID
}

// Kind returns "dying_combatant_turn_started".
func (DyingCombatantTurnStarted) Kind() string { return "dying_combatant_turn_started" }

// Combatants returns the combatant whose turn began.
func (e DyingCombatantTurnStarted) Combatants() []CombatantID { return one(e.combatant) }

// TurnEventOccurred wraps one tracker event unchanged. Its Kind and Combatants are
// the wrapped event's. It is a transport envelope.
type TurnEventOccurred struct {
	encounterEventBase
	wrapped TurnEvent
}

// Kind returns the wrapped event's Kind.
func (e TurnEventOccurred) Kind() string { return e.wrapped.Kind() }

// Combatants returns the wrapped event's Combatants.
func (e TurnEventOccurred) Combatants() []CombatantID { return e.wrapped.Combatants() }

// Event returns the wrapped tracker event.
func (e TurnEventOccurred) Event() TurnEvent { return e.wrapped }

func (TurnEventOccurred) transportEnvelope() {}

// HitPointEventOccurred wraps one hit point event about one combatant unchanged. Its
// Kind and Combatants are the wrapped event's. It is a transport envelope.
type HitPointEventOccurred struct {
	encounterEventBase
	wrapped HitPointEvent
}

// Kind returns the wrapped event's Kind.
func (e HitPointEventOccurred) Kind() string { return e.wrapped.Kind() }

// Combatants returns the wrapped event's Combatants.
func (e HitPointEventOccurred) Combatants() []CombatantID { return e.wrapped.Combatants() }

// Event returns the wrapped hit point event.
func (e HitPointEventOccurred) Event() HitPointEvent { return e.wrapped }

func (HitPointEventOccurred) transportEnvelope() {}

// ConditionEventOccurred wraps one conditions event about one combatant unchanged.
// Its Kind and Combatants are the wrapped event's. It is a transport envelope.
type ConditionEventOccurred struct {
	encounterEventBase
	wrapped ConditionEvent
}

// Kind returns the wrapped event's Kind.
func (e ConditionEventOccurred) Kind() string { return e.wrapped.Kind() }

// Combatants returns the wrapped event's Combatants.
func (e ConditionEventOccurred) Combatants() []CombatantID { return e.wrapped.Combatants() }

// Event returns the wrapped conditions event.
func (e ConditionEventOccurred) Event() ConditionEvent { return e.wrapped }

func (ConditionEventOccurred) transportEnvelope() {}

// GrappleEventOccurred wraps one grapple event unchanged, about the grappled
// combatant and its grappler. Its Kind and Combatants are the wrapped event's. It
// is a transport envelope.
type GrappleEventOccurred struct {
	encounterEventBase
	wrapped GrappleEvent
}

// Kind returns the wrapped event's Kind.
func (e GrappleEventOccurred) Kind() string { return e.wrapped.Kind() }

// Combatants returns the wrapped event's Combatants.
func (e GrappleEventOccurred) Combatants() []CombatantID { return e.wrapped.Combatants() }

// Event returns the wrapped grapple event.
func (e GrappleEventOccurred) Event() GrappleEvent { return e.wrapped }

func (GrappleEventOccurred) transportEnvelope() {}

// envelopeBase checks the encounter id and round every wrapper carries.
func envelopeBase(id EncounterID, round int) (encounterEventBase, error) {
	if id.IsZero() || round < 1 {
		return encounterEventBase{}, invalidInput("an event needs an encounter id and a round of 1 or more")
	}
	return encounterEventBase{id, round}, nil
}

// NewTurnEventOccurred wraps a tracker event. It returns an error matching
// srderr.ErrInvalidInput for a zero id, a round below 1 or a nil event. It never panics.
func NewTurnEventOccurred(id EncounterID, round int, e TurnEvent) (TurnEventOccurred, error) {
	b, err := envelopeBase(id, round)
	if err != nil || e == nil {
		return TurnEventOccurred{}, invalidInput("a wrapper needs an encounter, a round and an event")
	}
	return TurnEventOccurred{b, e}, nil
}

// NewHitPointEventOccurred wraps a hit point event about combatant. It returns an
// error matching srderr.ErrInvalidInput for a zero id, a round below 1, a nil event or
// an event whose Combatants is not [combatant]. It never panics.
func NewHitPointEventOccurred(id EncounterID, round int, combatant CombatantID, e HitPointEvent) (HitPointEventOccurred, error) {
	b, err := envelopeBase(id, round)
	if err != nil || e == nil || !slices.Equal(e.Combatants(), one(combatant)) {
		return HitPointEventOccurred{}, invalidInput("the wrapped event is not about the combatant")
	}
	return HitPointEventOccurred{b, e}, nil
}

// NewConditionEventOccurred wraps a conditions event about combatant. It returns an
// error matching srderr.ErrInvalidInput for a zero id, a round below 1, a nil event or
// an event whose Combatants is not [combatant]. It never panics.
func NewConditionEventOccurred(id EncounterID, round int, combatant CombatantID, e ConditionEvent) (ConditionEventOccurred, error) {
	b, err := envelopeBase(id, round)
	if err != nil || e == nil || !slices.Equal(e.Combatants(), one(combatant)) {
		return ConditionEventOccurred{}, invalidInput("the wrapped event is not about the combatant")
	}
	return ConditionEventOccurred{b, e}, nil
}

// NewGrappleEventOccurred wraps a grapple event under the grappled combatant. It
// returns an error matching srderr.ErrInvalidInput for a zero id, a round below 1, a
// nil event or an event whose first combatant is not grappled. It never panics.
func NewGrappleEventOccurred(id EncounterID, round int, grappled CombatantID, e GrappleEvent) (GrappleEventOccurred, error) {
	b, err := envelopeBase(id, round)
	if err != nil || e == nil || len(e.Combatants()) == 0 || e.Combatants()[0] != grappled {
		return GrappleEventOccurred{}, invalidInput("the wrapped event is not about the grappled combatant")
	}
	return GrappleEventOccurred{b, e}, nil
}
