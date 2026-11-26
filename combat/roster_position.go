package combat

import "slices"

// RosterPosition is where a combatant takes its place in the TurnOrder when it joins or is
// moved (combatant-roster-changes.md FR-001). Variants: RosterAfterCombatant,
// RosterAtInitiative. Both compare by value with ==. The zero value (a nil RosterPosition) is
// invalid.
type RosterPosition interface{ isRosterPosition() }

// RosterAfterCombatant places the combatant directly after Anchor with Anchor's count (SRD
// 5.2.1, "Giant Insect", p. 135; "Summon Dragon", p. 166: "shares your Initiative count, but it
// takes its turn immediately after yours"). With several entries placed after one anchor the
// latest placed is nearest it. Anchor must not be blank. Build it with a keyed literal.
type RosterAfterCombatant struct{ Anchor CombatantID }

// RosterAtInitiative places the combatant by Count, after every entry with a higher count and
// before every entry with a lower one (SRD 5.2.1, "Initiative", p. 13). Count is any integer,
// negative and zero included. Tie says where it goes among entries with an equal count: the
// engine does not decide ties, the host does. Build it with a keyed literal.
type RosterAtInitiative struct {
	Count InitiativeCount
	Tie   TiePlacement
}

func (RosterAfterCombatant) isRosterPosition() {}
func (RosterAtInitiative) isRosterPosition()   {}

// TiePlacement is the host's decision for a combatant that ties with others on the Initiative
// count (SRD 5.2.1, "Initiative", "Ties", p. 13: the Game Master and the players decide). The
// zero value is invalid.
type TiePlacement int

// The two tie placements.
const (
	// TiePlacementBeforeTied puts the combatant before the first entry with an equal count.
	TiePlacementBeforeTied TiePlacement = iota + 1
	// TiePlacementAfterTied puts the combatant after the last entry with an equal count.
	TiePlacementAfterTied
)

// Valid reports whether t is one of the two placements.
func (t TiePlacement) Valid() bool { return t == TiePlacementBeforeTied || t == TiePlacementAfterTied }

// String returns "before_tied", "after_tied" or "unknown".
func (t TiePlacement) String() string { return enumName([]string{"before_tied", "after_tied"}, int(t)) }

// validRosterPosition reports whether p is a usable position: a variant with a non-blank anchor
// or a defined tie placement.
func validRosterPosition(p RosterPosition) bool {
	switch v := p.(type) {
	case RosterAfterCombatant:
		return !v.Anchor.IsZero()
	case RosterAtInitiative:
		return v.Tie.Valid()
	}
	return false
}

// placeIn returns the index at which a combatant takes position p in order, and the count it
// gets (combatant-roster-changes.md FR-002, FR-003). order must hold counts that do not
// increase and not hold the subject. An anchor that is not in order is a
// RosterUnknownCombatantRejection. p must be valid.
func placeIn(order []InitiativeStanding, p RosterPosition) (int, InitiativeCount, RosterRejection) {
	switch v := p.(type) {
	case RosterAfterCombatant:
		i := slices.IndexFunc(order, func(s InitiativeStanding) bool { return s.Combatant == v.Anchor })
		if i < 0 {
			return 0, 0, RosterUnknownCombatantRejection{v.Anchor}
		}
		return i + 1, order[i].Count, nil
	case RosterAtInitiative:
		i := slices.IndexFunc(order, func(s InitiativeStanding) bool {
			if v.Tie == TiePlacementBeforeTied {
				return s.Count <= v.Count
			}
			return s.Count < v.Count
		})
		if i < 0 {
			i = len(order)
		}
		return i, v.Count, nil
	default:
		panic(unreachableError("roster position", p))
	}
}

// precededBy returns the standing of the entry directly before id in order, or the zero standing when id
// is first or absent (FR-034). It returns a standing, not an id, because no function of the package
// returns a CombatantID.
func precededBy(order []InitiativeStanding, id CombatantID) InitiativeStanding {
	if i := slices.IndexFunc(order, func(s InitiativeStanding) bool { return s.Combatant == id }); i > 0 {
		return order[i-1]
	}
	return InitiativeStanding{}
}
