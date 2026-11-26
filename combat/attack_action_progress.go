package combat

import "slices"

// AttackActionProgress is one Attack action in progress (SRD 5.2.1, "Attack [Action]",
// p. 177): the combatant, the allowance, the attacks counted so far and the weapon-change
// slots used. A slot is the 1-based number of an attack. Each Attack action has its own
// progress, so a second Attack action in a turn starts a new one. It records no weapon
// held, no Opportunity Attack and no movement. Build it with StartAttackActionProgress or
// NewAttackActionProgress; every operation returns a new value. The zero value is invalid.
// It holds slices, so compare with Equal, not ==; accessors return copies. Safe for
// concurrent use.
type AttackActionProgress struct {
	attacker  CombatantID
	allowance AttackAllowance
	entries   []AttackEntry
	slots     []int
}

// StartAttackActionProgress returns a progress with no attack recorded and no weapon
// change used. It returns an error matching srderr.ErrInvalidInput for a blank attacker
// or an invalid allowance. It never panics.
func StartAttackActionProgress(attacker CombatantID, allowance AttackAllowance) (AttackActionProgress, error) {
	return NewAttackActionProgress(attacker, allowance, nil, nil)
}

// NewAttackActionProgress returns the progress holding attacks and usedSlots, for a host
// restoring saved state. It returns an error matching srderr.ErrInvalidInput for a blank
// attacker, an invalid allowance or entry, a history that exceeds the limit on any prefix
// (each attack is checked against the limit for the attacks before it), a slot below 1, a
// slot twice, or a slot above the attacks recorded plus one; the slot after the last
// attack is valid only while attacks remain. More than 256 attacks return a
// LimitExceededError (plan P10). It copies its slices and never panics.
func NewAttackActionProgress(attacker CombatantID, allowance AttackAllowance, attacks []AttackEntry, usedSlots []int) (AttackActionProgress, error) {
	if err := overCap("attack action history", len(attacks), maxInputList); err != nil {
		return AttackActionProgress{}, err
	}
	if attacker.IsZero() || !allowance.Valid() {
		return AttackActionProgress{}, invalidInput("attack action progress needs an attacker and an allowance")
	}
	cur := AttackActionProgress{attacker: attacker, allowance: allowance}
	for _, e := range attacks {
		if !e.Valid() || !cur.canRecord(e) {
			return AttackActionProgress{}, invalidInput("attack action history is invalid or exceeds the limit")
		}
		cur.entries = append(cur.entries, e)
	}
	seen := make(map[int]bool, len(usedSlots))
	for _, s := range usedSlots {
		if !cur.slotValid(s, seen) {
			return AttackActionProgress{}, invalidInput("weapon change slot is invalid")
		}
		seen[s] = true
	}
	cur.slots = slices.Clone(usedSlots)
	return cur, nil
}

// slotValid reports whether s may be a used slot: 1 or more, not seen before, no later than the attack
// after the last one recorded, and that one only while an attack remains.
func (p AttackActionProgress) slotValid(s int, seen map[int]bool) bool {
	made := len(p.entries)
	return s >= 1 && !seen[s] && s <= made+1 && (s != made+1 || p.AttacksRemaining() >= 1)
}

// IsFor reports whether the progress was started for the combatant. A host holding one
// progress per combatant checks this before recording. No accessor returns the id itself
// (combat-foundations FR-009).
func (p AttackActionProgress) IsFor(id CombatantID) bool { return !id.IsZero() && p.attacker == id }

// Allowance returns the allowance the progress counts against.
func (p AttackActionProgress) Allowance() AttackAllowance { return p.allowance }

// Attacks returns a copy of the attacks counted so far, in order.
func (p AttackActionProgress) Attacks() []AttackEntry { return append([]AttackEntry{}, p.entries...) }

// UsedWeaponChangeSlots returns a copy of the slots that hold a weapon change, in the order used.
func (p AttackActionProgress) UsedWeaponChangeSlots() []int { return append([]int{}, p.slots...) }

// Valid reports whether p was built by StartAttackActionProgress or NewAttackActionProgress.
func (p AttackActionProgress) Valid() bool { return !p.attacker.IsZero() && p.allowance.Valid() }

// Equal reports whether p and o have the same attacker, allowance and attacks in the same
// order, and the same slots used, whatever order they were used in.
func (p AttackActionProgress) Equal(o AttackActionProgress) bool {
	return p.attacker == o.attacker && p.allowance == o.allowance && slices.Equal(p.entries, o.entries) && sameSet(p.slots, o.slots)
}

// limitWith returns the limit that governs recording e after the attacks so far: the pact
// weapon limit when every attack so far and e are pact weapon attacks or replacements and
// at least one is a pact weapon attack, the general limit otherwise (SRD 5.2.1,
// "Thirsting Blade", p. 74).
func (p AttackActionProgress) limitWith(e AttackEntry) int {
	allPact, anyPact := e.pactCompatible(), e.means == AttackMeansPactWeapon
	for _, h := range p.entries {
		allPact = allPact && h.pactCompatible()
		anyPact = anyPact || h.means == AttackMeansPactWeapon
	}
	if allPact && anyPact {
		return p.allowance.pact
	}
	return p.allowance.general
}

func (p AttackActionProgress) canRecord(e AttackEntry) bool { return len(p.entries) < p.limitWith(e) }

// AttacksRemaining returns the attacks the allowance still gives: the pact weapon limit
// minus the attacks recorded when every recorded attack is a pact weapon attack or a
// replacement (an empty history included), the general limit minus them otherwise. It is
// never negative.
func (p AttackActionProgress) AttacksRemaining() int {
	allPact := true
	for _, h := range p.entries {
		allPact = allPact && h.pactCompatible()
	}
	limit := p.allowance.general
	if allPact {
		limit = p.allowance.pact
	}
	return max(0, limit-len(p.entries))
}

// IsBetweenAttacks reports whether at least one attack is recorded and attacks remain:
// the moment a combatant may move between attacks (SRD 5.2.1, "Moving between Attacks",
// p. 177). With one attack allowed there is never a moment between. It is a pure query;
// the movement itself belongs to the turn tracker.
func (p AttackActionProgress) IsBetweenAttacks() bool {
	return len(p.entries) > 0 && p.AttacksRemaining() >= 1
}

func (p AttackActionProgress) record(e AttackEntry) AttackRecordOutcome {
	if !e.Valid() || !p.canRecord(e) {
		return AttackActionAttackRefused{p, AttackRefusalLimitReached}
	}
	next := p
	next.entries = append(slices.Clone(p.entries), e)
	n, base := len(next.entries), attackActionEventBase{p.attacker}
	var ev AttackActionEvent = AttackMadeInAttackAction{base, e.means, n}
	if e.IsReplacement() {
		ev = AttackReplacedInAttackAction{base, e.replacement, n}
	}
	return AttackActionAttackRecorded{next, []AttackActionEvent{ev}}
}

// RecordAttack counts an attack made with the means against the limit. It returns
// AttackActionAttackRecorded with a new progress and one AttackMadeInAttackAction event, or
// AttackActionAttackRefused with AttackRefusalLimitReached and the progress unchanged when
// the limit is reached or the means is not valid. An Unarmed Strike counts like a weapon
// attack. It never panics.
func (p AttackActionProgress) RecordAttack(means AttackMeans) AttackRecordOutcome {
	return p.record(MadeAttack(means))
}

// RecordReplacement counts one attack replaced or forgone against the limit. It returns
// AttackActionAttackRecorded with an AttackReplacedInAttackAction event, or
// AttackActionAttackRefused with AttackRefusalLimitReached and the progress unchanged when
// the limit is reached or the replacement is not valid. The effect belongs to the owner of
// the feature or item. It never panics.
func (p AttackActionProgress) RecordReplacement(r AttackReplacement) AttackRecordOutcome {
	return p.record(ReplacedAttack(r))
}

// HandleWeapon records one equip or unequip in the slot of an attack: before the next
// attack it uses slot made + 1, after the last attack made it uses slot made (SRD 5.2.1,
// "Equipping and Unequipping Weapons", p. 177). It returns AttackActionWeaponChangeRecorded
// with a new progress and one WeaponChangedInAttackAction event, or
// AttackActionWeaponChangeRefused with the progress unchanged: NoAttackRemaining before an
// attack when none remains, NoAttackMade after an attack when none is made, and
// AlreadyUsedForThisAttack when the slot holds a change. An invalid change or timing is
// refused as NoAttackRemaining. It never panics.
func (p AttackActionProgress) HandleWeapon(change AttackActionWeaponChange, timing AttackActionWeaponChangeTiming) WeaponChangeOutcome {
	made, slot := len(p.entries), 0
	switch {
	case !change.Valid() || !timing.Valid():
		return AttackActionWeaponChangeRefused{p, WeaponChangeRefusalNoAttackRemaining}
	case timing == WeaponChangeBeforeAttack && p.AttacksRemaining() < 1:
		return AttackActionWeaponChangeRefused{p, WeaponChangeRefusalNoAttackRemaining}
	case timing == WeaponChangeBeforeAttack:
		slot = made + 1
	case made == 0:
		return AttackActionWeaponChangeRefused{p, WeaponChangeRefusalNoAttackMade}
	default:
		slot = made
	}
	if slices.Contains(p.slots, slot) {
		return AttackActionWeaponChangeRefused{p, WeaponChangeRefusalAlreadyUsedForThisAttack}
	}
	next := p
	next.slots = append(slices.Clone(p.slots), slot)
	ev := WeaponChangedInAttackAction{attackActionEventBase{p.attacker}, change, timing, slot}
	return AttackActionWeaponChangeRecorded{next, []AttackActionEvent{ev}}
}
