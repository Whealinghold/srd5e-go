package combat

import "slices"

// Encounter is the aggregate root of one combat encounter, from its first turn to
// its end (SRD 5.2.1, "Combat", p. 13): its id, its status, the TurnTracker that
// holds the turn order, round and active combatant, and the Combatants stored in
// the tracker's turn order. It is opaque: every field is unexported, every accessor
// returns a copy, and every transition returns a new Encounter and leaves the
// receiver unchanged, so a host alias cannot change state during a resolution.
// Build one with Start; the zero Encounter, and any state that breaks an invariant,
// is invalid (IsZero) and every transition on it returns an error matching
// srderr.ErrInvalidInput. Each transition checks the invariants at entry and at exit of the call (plan P14), an
// O(n) pass over the n combatants, never inside a cascade: the tracker (round, active index,
// profiles, per-combatant state) and the parts of every Combatant (Hit Points, life state,
// conditions) are validated, not the order's ranking and not the consistency between a Combatant's life
// state, markers and conditions.
// Compare with Equal, since it holds a slice. Safe for concurrent use as a value.
type Encounter struct {
	id         EncounterID
	status     EncounterStatus
	tracker    TurnTracker
	combatants []Combatant
}

// Start begins an encounter from the turn order the initiative step established and
// the combatants, which it stores in the tracker's order whatever order they were
// passed in, taking its own copy. It returns the encounter, the events
// EncounterStarted, the established tracker's events wrapped, and
// DyingCombatantTurnStarted when the first combatant is Dying. The receiver is
// ignored: call it on the zero Encounter.
//
// It returns an error satisfying srderr.ErrInvalidInput for a blank id, a zero
// tracker or a zero Combatant, and otherwise, for an input that is not valid, the
// errors.Join of one EncounterProblem per problem found (ProblemsOf lists them):
// NoCombatantsProblem, MissingSideProblem, DuplicateCombatantIDProblem,
// CombatantNotInTurnOrderProblem and TurnOrderEntryWithoutCombatantProblem, or CombatantLimitProblem alone for more than 256 combatants (plan P10). It never panics.
func (Encounter) Start(id EncounterID, established InitiativeOrderEstablished, combatants []Combatant) (Encounter, []Event, error) {
	return start(id, established, combatants, true)
}

// start is Start; capped false skips the cap of 256 combatants so a test can build the large encounters that
// the event cap of a call is written for.
func start(id EncounterID, established InitiativeOrderEstablished, combatants []Combatant, capped bool) (Encounter, []Event, error) {
	tracker := established.tracker
	if id.IsZero() || tracker.IsZero() || slices.ContainsFunc(combatants, Combatant.IsZero) {
		return Encounter{}, nil, invalidInput("an encounter needs an id, an established order and valid combatants")
	}
	if capped && (len(combatants) > maxCombatantsPerEncounter || len(tracker.order) > maxCombatantsPerEncounter) {
		return Encounter{}, nil, joinEncounterProblems([]EncounterProblem{CombatantLimitProblem{}})
	}
	if !tracker.wellFormed() || slices.ContainsFunc(combatants, func(c Combatant) bool { return !c.wellFormed() }) {
		return Encounter{}, nil, invalidInput("an encounter needs a well formed tracker and well formed combatants")
	}
	var problems []EncounterProblem
	if len(combatants) == 0 {
		problems = append(problems, NoCombatantsProblem{})
	}
	for _, s := range []Side{SideA, SideB} {
		if len(combatants) > 0 && !slices.ContainsFunc(combatants, func(c Combatant) bool { return c.side == s }) {
			problems = append(problems, MissingSideProblem{s})
		}
	}
	inOrder := map[CombatantID]bool{}
	for _, s := range tracker.order {
		inOrder[s.Combatant] = true
	}
	byID := map[CombatantID]Combatant{}
	for _, c := range combatants {
		switch _, dup := byID[c.ref.ID]; {
		case dup:
			problems = append(problems, DuplicateCombatantIDProblem{c.ref.ID})
		case !inOrder[c.ref.ID]:
			problems = append(problems, CombatantNotInTurnOrderProblem{c.ref.ID})
			byID[c.ref.ID] = c
		default:
			byID[c.ref.ID] = c
		}
	}
	for _, s := range tracker.order {
		if _, ok := byID[s.Combatant]; !ok {
			problems = append(problems, TurnOrderEntryWithoutCombatantProblem{s.Combatant})
		}
	}
	if len(problems) > 0 {
		return Encounter{}, nil, joinEncounterProblems(problems)
	}
	stored := make([]Combatant, len(tracker.order))
	ids := make([]CombatantID, len(tracker.order))
	for i, s := range tracker.order {
		stored[i], ids[i] = byID[s.Combatant], s.Combatant
	}
	stored[0].dstMade = false
	enc := Encounter{id, EncounterInProgress{}, tracker, stored}
	events := []Event{EncounterStarted{encounterEventBase{id, tracker.round}, ids}}
	events = append(events, enc.wrapTurn(tracker.round, established.events)...)
	return enc, append(events, enc.dyingTurnEvent(0)...), nil
}

// MustStart is Start for input known to be valid. It panics with the error Start
// returns, and only then.
func (e Encounter) MustStart(id EncounterID, established InitiativeOrderEstablished, combatants []Combatant) (Encounter, []Event) {
	enc, events, err := e.Start(id, established, combatants)
	if err != nil {
		panic(err)
	}
	return enc, events
}

// IsZero reports whether e is invalid: the zero Encounter, or one whose parts break
// an invariant (no id, status or tracker, a zero Combatant, combatants that are not
// exactly the tracker's ids in its order).
func (e Encounter) IsZero() bool {
	if e.id.IsZero() || !e.tracker.wellFormed() || len(e.combatants) != len(e.tracker.order) {
		return true
	}
	switch s := e.status.(type) {
	case EncounterInProgress:
	case EncounterEndedStatus:
		if !validEndReason(s.Reason) {
			return true
		}
	default:
		return true
	}
	seen := map[CombatantID]bool{}
	for i, c := range e.combatants {
		if !c.wellFormed() || seen[c.ref.ID] || e.tracker.order[i].Combatant != c.ref.ID {
			return true
		}
		seen[c.ref.ID] = true
	}
	return false
}

// usable is the O(1) entry check of the root methods that run inside a cascade (plan P14): an id, a status, a
// tracker with an order and one combatant per order entry. The O(n) validation of every part is IsZero; it runs
// at entry and exit of a call (runTransition, resolution.exitInvalid), never once per committed step.
func (e Encounter) usable() bool {
	if e.id.IsZero() || len(e.tracker.order) == 0 || len(e.combatants) != len(e.tracker.order) {
		return false
	}
	switch s := e.status.(type) {
	case EncounterInProgress:
		return true
	case EncounterEndedStatus:
		return validEndReason(s.Reason)
	}
	return false
}

// ID returns the encounter's id.
func (e Encounter) ID() EncounterID { return e.id }

// Status returns whether the encounter is in progress or ended.
func (e Encounter) Status() EncounterStatus { return e.status }

// Tracker returns the turn tracker: the turn order, round and active combatant.
func (e Encounter) Tracker() TurnTracker { return e.tracker }

// Combatants returns a copy of the combatants in the tracker's turn order.
func (e Encounter) Combatants() []Combatant { return slices.Clone(e.combatants) }

// Find returns the combatant with the id and whether the encounter holds it.
func (e Encounter) Find(id CombatantID) (Combatant, bool) {
	if i := e.index(id); i >= 0 {
		return e.combatants[i], true
	}
	return Combatant{}, false
}

// Equal reports whether e and o have the same id, status, tracker and combatants in
// the same order. Two starts that differ only in the order of the combatants passed
// are equal, because combatants are stored in turn order.
func (e Encounter) Equal(o Encounter) bool {
	return e.id == o.id && e.status == o.status && e.tracker.Equal(o.tracker) &&
		slices.EqualFunc(e.combatants, o.combatants, Combatant.Equal)
}

// index returns the position of id in the combatants, or -1.
func (e Encounter) index(id CombatantID) int {
	return slices.IndexFunc(e.combatants, func(c Combatant) bool { return c.ref.ID == id })
}

// replace returns e with the combatant at i replaced, on a new slice.
func (e Encounter) replace(i int, c Combatant) Encounter {
	e.combatants = slices.Clone(e.combatants)
	e.combatants[i] = c
	return e
}

// ended reports whether the encounter has ended.
func (e Encounter) ended() bool {
	_, ok := e.status.(EncounterEndedStatus)
	return ok
}

// base is the id and current round every event of a transition carries.
func (e Encounter) base() encounterEventBase { return encounterEventBase{e.id, e.tracker.round} }

// wrapTurn wraps tracker events in the order given. Each carries the round it happened
// in: start until a TrackerRoundStarted, then that round.
func (e Encounter) wrapTurn(start int, events []TurnEvent) []Event {
	out := make([]Event, len(events))
	round := start
	for i, ev := range events {
		if rs, ok := ev.(TrackerRoundStarted); ok {
			round = rs.round
		}
		out[i] = TurnEventOccurred{encounterEventBase{e.id, round}, ev}
	}
	return out
}

// wrapHitPoint wraps hit point events about id.
func (e Encounter) wrapHitPoint(events []HitPointEvent) []Event {
	out := make([]Event, len(events))
	for i, ev := range events {
		out[i] = HitPointEventOccurred{e.base(), ev}
	}
	return out
}

// wrapCondition wraps conditions events.
func (e Encounter) wrapCondition(events []ConditionEvent) []Event {
	out := make([]Event, len(events))
	for i, ev := range events {
		out[i] = ConditionEventOccurred{e.base(), ev}
	}
	return out
}

// dyingTurnEvent is DyingCombatantTurnStarted for the combatant at i when it is Dying
// (FR-018), and no event otherwise.
func (e Encounter) dyingTurnEvent(i int) []Event {
	if !e.combatants[i].hp.MakesDeathSavingThrows() {
		return nil
	}
	return []Event{DyingCombatantTurnStarted{e.base(), e.combatants[i].ref.ID}}
}
