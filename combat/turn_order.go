package combat

import (
	"cmp"
	"slices"
)

// InitiativeStanding is one combatant's place in the Initiative order: its id and
// count. Build it with a keyed literal. Comparable with ==.
type InitiativeStanding struct {
	Combatant CombatantID
	Count     InitiativeCount
}

// TiedGroup is two or more combatants with the same Initiative count, in the order
// given. Immutable; not comparable with ==.
type TiedGroup struct {
	count InitiativeCount
	ids   []CombatantID
}

// Count returns the shared count.
func (g TiedGroup) Count() InitiativeCount { return g.count }

// IDs returns a copy of the tied combatants in the order given.
func (g TiedGroup) IDs() []CombatantID { return slices.Clone(g.ids) }

// TieBreak places the combatants of one tied group in the order chosen (SRD 5.2.1,
// "Initiative", "Ties", p. 13). Build it with NewTieBreak; the zero value is invalid.
type TieBreak struct{ ids []CombatantID }

// NewTieBreak returns the tie-break naming ids in the chosen order. It returns an
// error matching srderr.ErrInvalidInput for no id or a blank id, and a LimitExceededError for more than 256 ids. A repeated or
// unknown id is not an error here: EstablishInitiativeOrder reports it as a
// TieBreakRejection. The ids are copied. It never panics.
func NewTieBreak(ids []CombatantID) (TieBreak, error) {
	if err := overCap("tie-break ids", len(ids), maxInputList); err != nil {
		return TieBreak{}, err
	}
	if len(ids) == 0 || slices.ContainsFunc(ids, CombatantID.IsZero) {
		return TieBreak{}, invalidInput("a tie-break names at least one combatant")
	}
	return TieBreak{slices.Clone(ids)}, nil
}

// IDs returns a copy of the ids in the chosen order.
func (t TieBreak) IDs() []CombatantID { return slices.Clone(t.ids) }

// TieBreakRejectionReason says why a tie-break was not applied. The zero value is invalid.
type TieBreakRejectionReason int

// The reasons, in the order the first that holds is reported.
const (
	// TieBreakUnknownID: an id is not among the standings.
	TieBreakUnknownID TieBreakRejectionReason = iota + 1
	// TieBreakRepeatedID: an id appears twice in the list.
	TieBreakRepeatedID
	// TieBreakNotTied: an id shares its count with nobody.
	TieBreakNotTied
	// TieBreakSpansGroups: the ids belong to more than one tied group.
	TieBreakSpansGroups
	// TieBreakPartial: the list names part of one group.
	TieBreakPartial
	// TieBreakDuplicateGroup: an earlier accepted list already covers the group.
	TieBreakDuplicateGroup
)

// String returns the snake_case reason or "unknown".
func (r TieBreakRejectionReason) String() string {
	return enumName([]string{"unknown_id", "repeated_id", "not_tied", "spans_groups", "partial", "duplicate_group"}, int(r))
}

// TieBreakRejection is a tie-break that was not applied: its ids as submitted and why.
type TieBreakRejection struct {
	ids    []CombatantID
	reason TieBreakRejectionReason
}

// IDs returns a copy of the submitted ids.
func (r TieBreakRejection) IDs() []CombatantID { return slices.Clone(r.ids) }

// Reason returns why the tie-break was rejected.
func (r TieBreakRejection) Reason() TieBreakRejectionReason { return r.reason }

// TurnOrder is the Initiative order, highest to lowest, complete and tie-free (SRD
// 5.2.1, "Initiative Order", p. 13). It is obtained from a TurnTracker; the zero
// value is empty. Immutable; compare with Equal.
type TurnOrder struct{ standings []InitiativeStanding }

// Standings returns a copy of the order; changing it leaves the TurnOrder unchanged.
func (o TurnOrder) Standings() []InitiativeStanding { return slices.Clone(o.standings) }

// Len returns the number of combatants.
func (o TurnOrder) Len() int { return len(o.standings) }

// StandingOf returns the standing of id. It returns an error matching
// srderr.ErrInvalidInput for an id that is not in the order (a query, not a use case).
func (o TurnOrder) StandingOf(id CombatantID) (InitiativeStanding, error) {
	if i := o.indexOf(id); i >= 0 {
		return o.standings[i], nil
	}
	return InitiativeStanding{}, invalidInput("combatant is not in the order")
}

// Equal reports whether o and p rank the same combatants with the same counts.
func (o TurnOrder) Equal(p TurnOrder) bool { return slices.Equal(o.standings, p.standings) }

func (o TurnOrder) indexOf(id CombatantID) int {
	return slices.IndexFunc(o.standings, func(s InitiativeStanding) bool { return s.Combatant == id })
}

// tie outcomes, the values of srd5e.combat.tie_outcome.
const (
	tieResolved   = "resolved"
	tieUnresolved = "unresolved"
	tieRejected   = "rejected"
)

// ranking is the result of ranking standings under tie-breaks.
type ranking struct {
	order      []InitiativeStanding
	unresolved []TiedGroup
	rejections []TieBreakRejection
	ties       []string // one tie outcome per increment of initiative_ties
}

// rankStandings ranks standings from the highest count to the lowest, keeping the
// order given among equal counts, and places each valid tie-break's ids at the
// position of its group. The standings must hold distinct ids.
func rankStandings(standings []InitiativeStanding, breaks []TieBreak) ranking {
	sorted := slices.Clone(standings)
	slices.SortStableFunc(sorted, func(a, b InitiativeStanding) int { return cmp.Compare(b.Count, a.Count) })
	var groups []TiedGroup
	groupOf := map[CombatantID]int{}
	for i := 0; i < len(sorted); {
		j := i + 1
		for j < len(sorted) && sorted[j].Count == sorted[i].Count {
			j++
		}
		if j-i > 1 {
			g := TiedGroup{count: sorted[i].Count}
			for _, s := range sorted[i:j] {
				g.ids = append(g.ids, s.Combatant)
				groupOf[s.Combatant] = len(groups)
			}
			groups = append(groups, g)
		}
		i = j
	}
	known := map[CombatantID]bool{}
	for _, s := range standings {
		known[s.Combatant] = true
	}
	covered := map[int][]CombatantID{}
	named := map[int]bool{} // groups named by a rejected tie-break
	var out ranking
	extra := 0
	for _, b := range breaks {
		reason := breakReason(b.ids, known, groupOf, groups, covered)
		if reason == 0 {
			covered[groupOf[b.ids[0]]] = b.ids
			continue
		}
		out.rejections = append(out.rejections, TieBreakRejection{slices.Clone(b.ids), reason})
		naming := false
		for _, id := range b.ids {
			if g, ok := groupOf[id]; ok {
				named[g], naming = true, true
			}
		}
		if !naming {
			extra++
		}
	}
	for g, grp := range groups {
		switch _, isCovered := covered[g]; {
		case named[g]:
			out.ties = append(out.ties, tieRejected)
		case isCovered:
			out.ties = append(out.ties, tieResolved)
		default:
			out.ties = append(out.ties, tieUnresolved)
			out.unresolved = append(out.unresolved, grp)
		}
	}
	for range extra {
		out.ties = append(out.ties, tieRejected)
	}
	for i := 0; i < len(sorted); {
		g, tied := groupOf[sorted[i].Combatant]
		if !tied {
			out.order = append(out.order, sorted[i])
			i++
			continue
		}
		ids := groups[g].ids
		if c, ok := covered[g]; ok {
			ids = c
		}
		for _, id := range ids {
			out.order = append(out.order, InitiativeStanding{id, groups[g].count})
		}
		i += len(groups[g].ids)
	}
	return out
}

// breakReason returns why a tie-break is not applied, or 0 when it is valid. The first
// reason in the order of TieBreakRejectionReason is reported.
func breakReason(ids []CombatantID, known map[CombatantID]bool, groupOf map[CombatantID]int, groups []TiedGroup, covered map[int][]CombatantID) TieBreakRejectionReason {
	seen := map[CombatantID]bool{}
	repeated, unknown, notTied := false, false, false
	groupsSeen := map[int]bool{}
	for _, id := range ids {
		repeated = repeated || seen[id]
		seen[id] = true
		unknown = unknown || !known[id]
		g, tied := groupOf[id]
		notTied = notTied || (known[id] && !tied)
		if tied {
			groupsSeen[g] = true
		}
	}
	switch {
	case unknown:
		return TieBreakUnknownID
	case repeated:
		return TieBreakRepeatedID
	case notTied:
		return TieBreakNotTied
	case len(groupsSeen) > 1:
		return TieBreakSpansGroups
	}
	g := groupOf[ids[0]]
	switch _, done := covered[g]; {
	case len(ids) < len(groups[g].ids):
		return TieBreakPartial
	case done:
		return TieBreakDuplicateGroup
	}
	return 0
}

// swapStandings exchanges the counts of swapper and ally, keeping the list order (SRD
// 5.2.1, "Alert", Initiative Swap, p. 87). It reports the first refusal in the order
// of SwapInitiative: unknown swapper, unknown ally, swapper incapacitated, ally
// incapacitated, ally not willing.
func swapStandings(standings []InitiativeStanding, swapper, ally CombatantID, swapperAv, allyAv ConditionActionAvailability, consent AllyConsent) ([]InitiativeStanding, []TurnEvent, TurnRejection) {
	si := slices.IndexFunc(standings, func(s InitiativeStanding) bool { return s.Combatant == swapper })
	ai := slices.IndexFunc(standings, func(s InitiativeStanding) bool { return s.Combatant == ally })
	switch {
	case si < 0:
		return nil, nil, TurnUnknownCombatantRejection{swapper}
	case ai < 0:
		return nil, nil, TurnUnknownCombatantRejection{ally}
	case swapperAv == ConditionActionAvailabilityNone, allyAv == ConditionActionAvailabilityNone:
		return nil, nil, TurnIncapacitatedRejection{}
	case consent == AllyConsentUnwilling:
		return nil, nil, TurnAllyNotWillingRejection{}
	}
	out := slices.Clone(standings)
	out[si].Count, out[ai].Count = standings[ai].Count, standings[si].Count
	return out, []TurnEvent{InitiativeSwapped{first: swapper, second: ally, firstCount: out[si].Count, secondCount: out[ai].Count}}, nil
}
