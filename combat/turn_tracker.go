package combat

import (
	"maps"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// feat use windows.
const (
	featWindowTurn      = 1 // OncePerTurn: until the current turn ends
	featWindowNextStart = 2 // OnceUntilStartOfYourNextTurn: until the combatant's next turn starts
)

type featUseKey struct {
	id     CombatantID
	feat   content.FeatID
	window int
}

// extraAction is the active turn's additional action and whether it is spent.
type extraAction struct {
	grant AdditionalActionGrant
	used  ResourceState
}

// TurnTracker is the state of the turn order of one combat: the Initiative order, the
// round, the active combatant and its resources, the Reactions spent, those dodging, the
// readied actions, the profiles, the turn's additional action and the turn-limited
// feat uses (SRD 5.2.1, "Combat", p. 13). It is an aggregate and a value: every
// operation returns a new tracker and leaves the receiver unchanged. A tracker has no
// exported constructor; it comes from EstablishInitiativeOrder. The zero TurnTracker is
// invalid (IsZero); every id is unknown to it. Immutable; compare with Equal. Safe
// for concurrent use as a value.
type TurnTracker struct {
	order    []InitiativeStanding
	profiles map[CombatantID]CombatantProfile
	round    int
	active   int
	res      TurnResources
	spent    map[CombatantID]bool
	dodging  map[CombatantID]bool
	readied  map[CombatantID]ReadiedAction
	extra    extraAction
	feats    map[featUseKey]bool
}

// establishTracker builds the tracker of round 1 with the first combatant active
// (FR-027). The order must be non-empty, ranked and complete over the profiles.
func establishTracker(order []InitiativeStanding, profiles []CombatantProfile) TurnTracker {
	t := TurnTracker{order: slices.Clone(order), profiles: map[CombatantID]CombatantProfile{}, round: 1,
		res: freshTurnResources(), spent: map[CombatantID]bool{}, dodging: map[CombatantID]bool{},
		readied: map[CombatantID]ReadiedAction{}, feats: map[featUseKey]bool{}}
	for _, p := range profiles {
		t.profiles[p.ID] = p
	}
	return t
}

// clone copies the tracker so an operation can change the copy.
func (t TurnTracker) clone() TurnTracker {
	n := t
	n.order = slices.Clone(t.order)
	n.profiles = maps.Clone(t.profiles)
	n.spent = maps.Clone(t.spent)
	n.dodging = maps.Clone(t.dodging)
	n.readied = maps.Clone(t.readied)
	n.feats = maps.Clone(t.feats)
	return n
}

// IsZero reports whether t is the invalid zero tracker.
func (t TurnTracker) IsZero() bool { return len(t.order) == 0 }

// Round returns the current round, 1 or more.
func (t TurnTracker) Round() int { return t.round }

// Active returns the active combatant's standing.
func (t TurnTracker) Active() InitiativeStanding { return t.order[t.active] }

// Next returns the standing of the combatant whose turn comes next: the first when the
// active one is last, the active one itself when it is alone.
func (t TurnTracker) Next() InitiativeStanding { return t.order[(t.active+1)%len(t.order)] }

// Order returns the Initiative order.
func (t TurnTracker) Order() TurnOrder { return TurnOrder{slices.Clone(t.order)} }

// Resources returns the active combatant's resources this turn.
func (t TurnTracker) Resources() TurnResources { return t.res }

// ProfileOf returns the profile of id and whether it is in the tracker.
func (t TurnTracker) ProfileOf(id CombatantID) (CombatantProfile, bool) {
	p, ok := t.profiles[id]
	return p, ok
}

// ReactionSpent reports whether id has spent its Reaction since its own turn started.
func (t TurnTracker) ReactionSpent(id CombatantID) bool { return t.spent[id] }

// IsDodging reports whether id took Dodge and its next turn has not started.
func (t TurnTracker) IsDodging(id CombatantID) bool { return t.dodging[id] }

// ReadiedActionOf returns the readied action id holds, if any.
func (t TurnTracker) ReadiedActionOf(id CombatantID) (ReadiedAction, bool) {
	a, ok := t.readied[id]
	return a, ok
}

// AdditionalAction returns the additional action granted for the active turn, if any,
// and whether it is spent. The grant is zero when none was granted.
func (t TurnTracker) AdditionalAction() (AdditionalActionGrant, ResourceState) {
	return t.extra.grant, t.extra.used
}

// HeldSpellOf returns the spell id's readied action holds, HoldsNoSpell when it
// holds none. It returns an error matching srderr.ErrInvalidInput for an id that is
// not in the tracker.
func (t TurnTracker) HeldSpellOf(id CombatantID) (HeldSpellInput, error) {
	if _, ok := t.profiles[id]; !ok {
		return nil, invalidInput("combatant is not in the tracker")
	}
	if a, ok := t.readied[id]; ok {
		return a.Held(), nil
	}
	return HoldsNoSpell{}, nil
}

// DodgeBenefits returns whether the Dodge benefits apply to id: it is dodging, its
// availability is not none and the profile Speed is above 0 (SRD 5.2.1, "Dodge
// [Action]", p. 181). It returns an error matching srderr.ErrInvalidInput for an id that
// is not in the tracker or an availability that is not defined.
func (t TurnTracker) DodgeBenefits(id CombatantID, av ConditionActionAvailability) (DodgeBenefits, error) {
	p, ok := t.profiles[id]
	if !ok || !validAvailability(av) {
		return 0, invalidInput("combatant is not in the tracker, or availability is not defined")
	}
	if t.dodging[id] && av == ConditionActionAvailabilityAvailable && p.Speeds.Walk() > 0 {
		return DodgeBenefitsActive, nil
	}
	return DodgeBenefitsNone, nil
}

// Equal reports whether t and o are equal in every field: the order with its counts, the
// round, the active combatant, the resources, the sets of spent Reactions and of
// dodging combatants, the readied actions, the profiles, the additional action and the
// feat uses. Set and map insertion order is irrelevant.
func (t TurnTracker) Equal(o TurnTracker) bool {
	return slices.Equal(t.order, o.order) && t.round == o.round && t.active == o.active && t.res == o.res &&
		maps.Equal(t.profiles, o.profiles) && maps.Equal(setOf(t.spent), setOf(o.spent)) &&
		maps.Equal(setOf(t.dodging), setOf(o.dodging)) && maps.Equal(t.feats, o.feats) &&
		maps.EqualFunc(t.readied, o.readied, ReadiedAction.Equal) &&
		t.extra.used == o.extra.used && t.extra.grant.Equal(o.extra.grant)
}

// setOf drops the false entries of a set held as a map.
func setOf(m map[CombatantID]bool) map[CombatantID]bool {
	out := map[CombatantID]bool{}
	for k, v := range m {
		if v {
			out[k] = true
		}
	}
	return out
}

func validAvailability(a ConditionActionAvailability) bool {
	return a == ConditionActionAvailabilityAvailable || a == ConditionActionAvailabilityNone
}

// TrackerStep is the closed set of results of a tracker operation. Variants:
// TrackerStepApplied, TrackerStepRejected.
type TrackerStep interface{ isTrackerStep() }

// TrackerStepApplied: the operation took effect: the new tracker and the events in
// the order they happened.
type TrackerStepApplied struct {
	tracker TurnTracker
	events  []TurnEvent
}

// TrackerStepRejected: the rules refuse the operation; nothing was applied.
type TrackerStepRejected struct{ Reason TurnRejection }

func (TrackerStepApplied) isTrackerStep()  {}
func (TrackerStepRejected) isTrackerStep() {}

// Tracker returns the new tracker.
func (s TrackerStepApplied) Tracker() TurnTracker { return s.tracker }

// Events returns a copy of the events.
func (s TrackerStepApplied) Events() []TurnEvent { return slices.Clone(s.events) }
