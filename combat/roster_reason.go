package combat

// JoinReason is why a combatant joined an encounter that had started (combatant-roster-changes.md
// FR-012). It changes no rule. The zero value is invalid.
type JoinReason int

// The two reasons.
const (
	// JoinReasonSummoned: a spell or an item placed the creature.
	JoinReasonSummoned JoinReason = iota + 1
	// JoinReasonArrived: a reinforcement, or a creature that was absent.
	JoinReasonArrived
)

// Valid reports whether r is one of the two reasons.
func (r JoinReason) Valid() bool { return r == JoinReasonSummoned || r == JoinReasonArrived }

// String returns "summoned", "arrived" or "unknown".
func (r JoinReason) String() string { return enumName([]string{"summoned", "arrived"}, int(r)) }

// RemovalCause is why a combatant left an encounter (SRD 5.2.1, "Ending Combat", p. 14: killed,
// knocked out, "surrendered or fled"; "Find Steed", p. 130: "disappears"). It is reported in
// CombatantRemoved and changes no rule (combatant-roster-changes.md FR-023). The zero value is
// invalid.
type RemovalCause int

// The four causes.
const (
	// RemovalCauseFled: the combatant fled.
	RemovalCauseFled RemovalCause = iota + 1
	// RemovalCauseSurrendered: the combatant surrendered.
	RemovalCauseSurrendered
	// RemovalCauseDisappeared: the combatant disappeared, as a summoned creature does at 0 Hit Points or when its spell ends.
	RemovalCauseDisappeared
	// RemovalCauseDeparted: any other reason the host reports, such as being teleported away.
	RemovalCauseDeparted
)

// Valid reports whether c is one of the four causes.
func (c RemovalCause) Valid() bool { return c >= RemovalCauseFled && c <= RemovalCauseDeparted }

// String returns "fled", "surrendered", "disappeared", "departed" or "unknown".
func (c RemovalCause) String() string {
	return enumName([]string{"fled", "surrendered", "disappeared", "departed"}, int(c))
}
