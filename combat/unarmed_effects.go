package combat

// UnarmedEffectPlan is what a Grapple or Shove asks of the encounter: a condition to apply to the target,
// the recorded push, or nothing. The zero value asks nothing. Immutable and comparable.
type UnarmedEffectPlan struct {
	condition Condition
	push      bool
}

// Condition returns the condition to apply and whether there is one.
func (p UnarmedEffectPlan) Condition() (Condition, bool) { return p.condition, p.condition != nil }

// Push reports whether the target is pushed 5 feet; the engine holds no position, so nothing changes.
func (p UnarmedEffectPlan) Push() bool { return p.push }

// UnarmedEffects decides what a Grapple or Shove asks of the encounter once the target's saving throw is
// known (combat-action-resolution.md FR-046, FR-047, FR-048). It is pure and stateless; the zero value
// is usable.
type UnarmedEffects struct{}

// Grapple returns the plan of a Grapple: Grappled by the attacker when the save failed, else nothing (SRD
// 5.2.1, "Unarmed Strike", p. 190).
func (UnarmedEffects) Grapple(attacker CombatantID, saveFailed bool) UnarmedEffectPlan {
	if !saveFailed {
		return UnarmedEffectPlan{}
	}
	return UnarmedEffectPlan{condition: Grappled{Grappler: attacker}}
}

// Shove returns the plan of a Shove with the attacker's effect: Prone, or the recorded push, when the
// save failed, else nothing.
func (UnarmedEffects) Shove(effect ShoveEffect, saveFailed bool) UnarmedEffectPlan {
	switch {
	case !saveFailed:
		return UnarmedEffectPlan{}
	case effect == GiveProne:
		return UnarmedEffectPlan{condition: Prone{}}
	case effect == PushFiveFeet:
		return UnarmedEffectPlan{push: true}
	}
	return UnarmedEffectPlan{}
}
