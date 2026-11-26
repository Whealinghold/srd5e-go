package combat

import "maps"

// ZoneLedger records, per creature, the TurnMarker of its latest once-per-turn
// application and the TurnMarker of its latest first entry into the zone
// (persistent-spell-zones FR-025 to FR-031). The zero ZoneLedger is empty. A
// ledger is never changed in place: an update returns a new ledger.
type ZoneLedger struct {
	applied map[CombatantID]TurnMarker
	entered map[CombatantID]TurnMarker
}

// Applied returns the turn of the creature's latest once-per-turn application,
// and whether the ledger holds one.
func (l ZoneLedger) Applied(c CombatantID) (TurnMarker, bool) {
	m, ok := l.applied[c]
	return m, ok
}

// Entered returns the turn of the creature's latest first entry, and whether the
// ledger holds one.
func (l ZoneLedger) Entered(c CombatantID) (TurnMarker, bool) {
	m, ok := l.entered[c]
	return m, ok
}

// Size returns the number of distinct creatures the ledger holds a turn for. It
// never exceeds the number of creatures seen.
func (l ZoneLedger) Size() int {
	n := len(l.applied)
	for c := range l.entered {
		if _, ok := l.applied[c]; !ok {
			n++
		}
	}
	return n
}

// Equal reports whether l and o hold the same turns for the same creatures.
func (l ZoneLedger) Equal(o ZoneLedger) bool {
	return maps.Equal(l.applied, o.applied) && maps.Equal(l.entered, o.entered)
}

// withApplied returns the ledger with c's once-per-turn application at turn m.
func (l ZoneLedger) withApplied(c CombatantID, m TurnMarker) ZoneLedger {
	return ZoneLedger{applied: withTurn(l.applied, c, m), entered: l.entered}
}

// withEntered returns the ledger with c's first entry at turn m.
func (l ZoneLedger) withEntered(c CombatantID, m TurnMarker) ZoneLedger {
	return ZoneLedger{applied: l.applied, entered: withTurn(l.entered, c, m)}
}

// latestRound returns the highest round the ledger holds for c, or 0 when it
// holds none.
func (l ZoneLedger) latestRound(c CombatantID) int {
	r := 0
	if m, ok := l.applied[c]; ok && m.round > r {
		r = m.round
	}
	if m, ok := l.entered[c]; ok && m.round > r {
		r = m.round
	}
	return r
}

// withTurn copies src and sets c to m, so the source map is never written.
func withTurn(src map[CombatantID]TurnMarker, c CombatantID, m TurnMarker) map[CombatantID]TurnMarker {
	out := make(map[CombatantID]TurnMarker, len(src)+1)
	maps.Copy(out, src)
	out[c] = m
	return out
}

// staleTurn reports whether turn is earlier than the zone's cast round or than a
// turn the ledger already holds for c (persistent-spell-zones FR-032, FR-055).
func staleTurn(zone PersistentZone, turn TurnMarker, c CombatantID) bool {
	return turn.round < zone.cast.round || turn.round < zone.ledger.latestRound(c)
}

// maxZoneLedgerCreatures is the cap of plan P10 on the creatures a zone's ledger holds: an encounter holds at most 256
// combatants.
const maxZoneLedgerCreatures = 256
