package combat

// Combatant is a participant in a battle ("each participant", SRD 5.2.1, "The Order
// of Combat", p. 13): its CombatantRef, its Side, its HitPointStatus and
// ConditionSet and two markers, KnockedOut and DeathSavingThrowMade (both false by
// default). The kind is identity only; every rule reads the HitPointStatus. The
// constructor adds no invariant between the parts: a Dying combatant without
// Unconscious is accepted, as the host supplies them. The zero Combatant is invalid
// (IsZero). Immutable and comparable with Equal; safe for concurrent use.
type Combatant struct {
	ref        CombatantRef
	side       Side
	hp         HitPointStatus
	conditions ConditionSet
	knockedOut bool
	dstMade    bool
}

// NewCombatant returns the combatant. It returns an error matching
// srderr.ErrInvalidInput for a ref with a zero id or no kind, a side that is not
// valid, or a zero HitPointStatus. It never panics.
func NewCombatant(ref CombatantRef, side Side, hp HitPointStatus, conditions ConditionSet) (Combatant, error) {
	if ref.ID.IsZero() || ref.Kind == nil || !side.Valid() || hp.IsZero() {
		return Combatant{}, invalidInput("a combatant needs an id, a kind, a side and a hit point status")
	}
	return Combatant{ref: ref, side: side, hp: hp, conditions: conditions}, nil
}

// IsZero reports whether c is the invalid zero Combatant.
func (c Combatant) IsZero() bool { return c.ref.ID.IsZero() }

// Ref returns the combatant's id and kind.
func (c Combatant) Ref() CombatantRef { return c.ref }

// Side returns the side the combatant fights on.
func (c Combatant) Side() Side { return c.side }

// HitPoints returns the combatant's hit point status.
func (c Combatant) HitPoints() HitPointStatus { return c.hp }

// Conditions returns the combatant's conditions.
func (c Combatant) Conditions() ConditionSet { return c.conditions }

// KnockedOut reports the marker of a knock-out: Unconscious was applied by a
// KnockedOut result (SRD 5.2.1, "Knocking Out a Creature", p. 17), so healing may end it.
func (c Combatant) KnockedOut() bool { return c.knockedOut }

// DeathSavingThrowMade reports whether a Death Saving Throw was made since the
// combatant's turn last started.
func (c Combatant) DeathSavingThrowMade() bool { return c.dstMade }

// WithKnockedOut returns a copy with the knocked-out marker set to v.
func (c Combatant) WithKnockedOut(v bool) Combatant { c.knockedOut = v; return c }

// WithDeathSavingThrowMade returns a copy with the per-turn Death Saving Throw
// marker set to v.
func (c Combatant) WithDeathSavingThrowMade(v bool) Combatant { c.dstMade = v; return c }

// Death returns the cause of death and true when the combatant is Dead, derived from
// its hit point status alone (SRD 5.2.1, "Dead", p. 180).
func (c Combatant) Death() (DeathCause, bool) {
	d, ok := c.hp.life.(Dead)
	return d.cause, ok
}

// Equal reports whether c and o have the same ref (id and kind), side, hit point
// status, conditions and markers.
func (c Combatant) Equal(o Combatant) bool { return c == o }
