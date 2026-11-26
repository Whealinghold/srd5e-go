package combat

// Side is one of the two sides of a combat encounter ("a clash between two
// sides", SRD 5.2.1, "The Order of Combat", p. 13). Two sides is a design
// decision of the encounter-aggregate spec, not an SRD rule. The zero value is
// invalid.
type Side int

// The two sides.
const (
	// SideA is the first side.
	SideA Side = iota + 1
	// SideB is the second side.
	SideB
)

// Valid reports whether s is SideA or SideB.
func (s Side) Valid() bool { return s == SideA || s == SideB }

// String returns "side_a", "side_b" or "unknown".
func (s Side) String() string { return enumName([]string{"side_a", "side_b"}, int(s)) }

// EndReason is why an Encounter ended (SRD 5.2.1, "Ending Combat", p. 14).
// Variants: EndReasonSideDefeated, EndReasonMutualAgreement.
type EndReason interface{ isEndReason() }

// EndReasonSideDefeated: Side was defeated. It is the side defeated, not the
// winner, and the encounter does not verify the claim.
type EndReasonSideDefeated struct{ Side Side }

// EndReasonMutualAgreement: the two sides agreed to stop.
type EndReasonMutualAgreement struct{}

func (EndReasonSideDefeated) isEndReason()    {}
func (EndReasonMutualAgreement) isEndReason() {}

// validEndReason reports whether r is a usable reason.
func validEndReason(r EndReason) bool {
	switch v := r.(type) {
	case EndReasonSideDefeated:
		return v.Side.Valid()
	case EndReasonMutualAgreement:
		return true
	}
	return false
}

// EncounterStatus is whether an Encounter is running or over. Variants:
// EncounterInProgress, EncounterEndedStatus.
type EncounterStatus interface{ isEncounterStatus() }

// EncounterInProgress: the encounter accepts commands.
type EncounterInProgress struct{}

// EncounterEndedStatus: the encounter ended for Reason and accepts no command.
type EncounterEndedStatus struct{ Reason EndReason }

func (EncounterInProgress) isEncounterStatus()  {}
func (EncounterEndedStatus) isEncounterStatus() {}
