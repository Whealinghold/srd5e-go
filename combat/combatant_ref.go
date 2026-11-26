package combat

// CombatantRef is a combatant's id and kind: what a host gives the engine to
// introduce a combatant. It is comparable and equal over both fields. Construct
// with NewCombatantRef or a keyed literal. The zero value is invalid.
type CombatantRef struct {
	ID   CombatantID
	Kind CombatantKind
}

// NewCombatantRef returns the reference for id and kind. It returns an error
// matching srderr.ErrInvalidInput for a zero id or a nil kind. It never panics.
func NewCombatantRef(id CombatantID, kind CombatantKind) (CombatantRef, error) {
	if id.IsZero() {
		return CombatantRef{}, invalidInput("combatant ref has a zero id")
	}
	if kind == nil {
		return CombatantRef{}, invalidInput("combatant ref has no kind")
	}
	return CombatantRef{ID: id, Kind: kind}, nil
}
