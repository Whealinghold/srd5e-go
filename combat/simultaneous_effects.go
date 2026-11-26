package combat

import (
	"slices"
	"strings"
)

// TurnHolder is whose turn it is: a player's or the Game Master's. It is named so as
// not to collide with the turn owner of the spell-effect data. The zero value is
// invalid.
type TurnHolder int

// The turn holders.
const (
	// TurnHolderPlayer: it is a player character's turn.
	TurnHolderPlayer TurnHolder = iota + 1
	// TurnHolderGameMaster: it is a monster's turn, run by the Game Master.
	TurnHolderGameMaster
)

// OrderingAuthority is who decides the order of simultaneous effects. The zero value
// is invalid.
type OrderingAuthority int

// The ordering authorities.
const (
	// OrderingAuthorityPlayer: the player whose turn it is.
	OrderingAuthorityPlayer OrderingAuthority = iota + 1
	// OrderingAuthorityGameMaster: the Game Master.
	OrderingAuthorityGameMaster
)

// String returns "player", "game_master" or "unknown".
func (a OrderingAuthority) String() string {
	return enumName([]string{"player", "game_master"}, int(a))
}

// PendingEffectID names one effect waiting to happen at the same time as others. It
// is a host-chosen, non-blank string. Build it with NewPendingEffectID; the zero
// value is invalid. Comparable and safe for concurrent use.
type PendingEffectID string

// NewPendingEffectID returns the id for s. It returns an error matching
// srderr.ErrInvalidInput for an empty or all-whitespace s. It never panics.
func NewPendingEffectID(s string) (PendingEffectID, error) {
	if strings.TrimSpace(s) == "" {
		return "", invalidInput("pending effect id must not be blank")
	}
	return PendingEffectID(s), nil
}

// OrderProblemKind is what is wrong with an id in a chosen order. The zero value is
// not a kind.
type OrderProblemKind int

// The problems of a chosen order.
const (
	// OrderProblemDuplicatePending: the pending list holds the id twice.
	OrderProblemDuplicatePending OrderProblemKind = iota + 1
	// OrderProblemRepeated: the chosen order holds the id twice.
	OrderProblemRepeated
	// OrderProblemUnknown: the chosen order names an id that is not pending.
	OrderProblemUnknown
	// OrderProblemMissing: a pending id is not in the chosen order.
	OrderProblemMissing
	// OrderProblemUnknownTurnHolder: the turn holder is not a TurnHolder.
	OrderProblemUnknownTurnHolder
	// OrderProblemLimitExceeded: the pending or chosen list holds more than 256 ids (plan P10); no other problem is looked for.
	OrderProblemLimitExceeded
)

// String returns "duplicate_pending", "repeated", "unknown", "missing", "unknown_turn_holder", "limit_exceeded" or "unknown_kind".
func (k OrderProblemKind) String() string {
	if k < OrderProblemDuplicatePending || k > OrderProblemLimitExceeded {
		return "unknown_kind"
	}
	return []string{"duplicate_pending", "repeated", "unknown", "missing", "unknown_turn_holder", valueResultLimitExceeded}[k-1]
}

// OrderProblem is one problem of a chosen order: its kind and the id it is about
// (blank for OrderProblemUnknownTurnHolder). Construct with a keyed literal.
type OrderProblem struct {
	Kind OrderProblemKind
	ID   PendingEffectID
}

// ResolveEffectOrderResult is the closed set of answers of
// SimultaneousEffects.Order. Variants: OrderSimultaneousEffectsOrdered,
// OrderSimultaneousEffectsRejected.
type ResolveEffectOrderResult interface{ isResolveEffectOrderResult() }

// OrderSimultaneousEffectsOrdered: the effects happen in the order chosen.
type OrderSimultaneousEffectsOrdered struct {
	order     []PendingEffectID
	authority OrderingAuthority
}

// Order returns a copy of the order, exactly as chosen (never sorted).
func (r OrderSimultaneousEffectsOrdered) Order() []PendingEffectID { return slices.Clone(r.order) }

// Authority returns who decides the order.
func (r OrderSimultaneousEffectsOrdered) Authority() OrderingAuthority { return r.authority }

// OrderSimultaneousEffectsRejected: the chosen order or the pending list is
// malformed; every problem is listed and nothing is ordered.
type OrderSimultaneousEffectsRejected struct{ problems []OrderProblem }

// Problems returns a copy of the problems: duplicate pending ids in pending order,
// then repeated and unknown ids in chosen order, then missing ids in pending order.
func (r OrderSimultaneousEffectsRejected) Problems() []OrderProblem { return slices.Clone(r.problems) }

func (OrderSimultaneousEffectsOrdered) isResolveEffectOrderResult()  {}
func (OrderSimultaneousEffectsRejected) isResolveEffectOrderResult() {}

// SimultaneousEffects is the stateless service for Simultaneous Effects. The zero
// value is ready to use. Safe for concurrent use.
type SimultaneousEffects struct{}

// AuthorityFor returns who decides the order for the turn holder: the player for a
// player's turn, the Game Master for a monster's (SRD 5.2.1, "Simultaneous Effects",
// p. 187). An undefined holder gives the zero (invalid) authority.
func (SimultaneousEffects) AuthorityFor(h TurnHolder) OrderingAuthority {
	switch h {
	case TurnHolderPlayer:
		return OrderingAuthorityPlayer
	case TurnHolderGameMaster:
		return OrderingAuthorityGameMaster
	}
	return 0
}

// Order validates the order the host brings back for effects that happen at the same
// time ("decides the order in which those things happen", SRD 5.2.1, "Simultaneous
// Effects", p. 187). Fewer than two pending effects need no decision and give
// Ordered with the pending list as it is. Otherwise chosen must be a permutation of
// pending: it is returned exactly as chosen, with the authority. A repeated pending
// id, an id repeated or unknown in chosen, a pending id missing from chosen, or an
// undefined holder gives Rejected listing every problem. More than 256 pending or 256 chosen ids give Rejected with the one problem OrderProblemLimitExceeded, before a list is read (plan P10). It does not change its
// arguments and never panics.
func (s SimultaneousEffects) Order(h TurnHolder, pending, chosen []PendingEffectID) ResolveEffectOrderResult {
	authority := s.AuthorityFor(h)
	if len(pending) > maxInputList || len(chosen) > maxInputList {
		return OrderSimultaneousEffectsRejected{[]OrderProblem{{Kind: OrderProblemLimitExceeded}}}
	}
	if len(pending) < 2 && authority != 0 {
		return OrderSimultaneousEffectsOrdered{slices.Clone(pending), authority}
	}
	var problems []OrderProblem
	if authority == 0 {
		problems = append(problems, OrderProblem{Kind: OrderProblemUnknownTurnHolder})
	}
	var distinct []PendingEffectID
	for _, id := range pending {
		if slices.Contains(distinct, id) {
			problems = append(problems, OrderProblem{OrderProblemDuplicatePending, id})
		} else {
			distinct = append(distinct, id)
		}
	}
	var placed []PendingEffectID
	for _, id := range chosen {
		switch {
		case slices.Contains(placed, id):
			problems = append(problems, OrderProblem{OrderProblemRepeated, id})
		case !slices.Contains(distinct, id):
			problems = append(problems, OrderProblem{OrderProblemUnknown, id})
		default:
			placed = append(placed, id)
		}
	}
	for _, id := range distinct {
		if !slices.Contains(placed, id) {
			problems = append(problems, OrderProblem{OrderProblemMissing, id})
		}
	}
	if len(problems) > 0 {
		return OrderSimultaneousEffectsRejected{problems}
	}
	return OrderSimultaneousEffectsOrdered{slices.Clone(chosen), authority}
}
