package combat

import "github.com/Whealinghold/srd5e-go/content"

// wellFormed reports whether the tracker holds together (plan P14): an order of 1 or more
// distinct, named combatants, a round of 1 or more, an active index inside the order, one profile for
// every combatant and no more, non-nil collections, and per-combatant state (spent Reactions, Dodge,
// readied actions, feat uses) that names a combatant of the order only. It does not re-rank the order
// (encounter-aggregate FR-009). A tracker from EstablishInitiativeOrder and the operations on it always
// pass; the zero tracker and a hand-built one with a broken part do not. It is one O(n) pass.
func (t TurnTracker) wellFormed() bool {
	n := len(t.order)
	if n == 0 || t.round < 1 || t.active < 0 || t.active >= n ||
		len(t.profiles) != n || t.profiles == nil || t.spent == nil || t.dodging == nil || t.readied == nil || t.feats == nil {
		return false
	}
	seen := make(map[CombatantID]bool, n)
	for _, s := range t.order {
		if s.Combatant.IsZero() || seen[s.Combatant] {
			return false
		}
		if p, ok := t.profiles[s.Combatant]; !ok || p.ID != s.Combatant {
			return false
		}
		seen[s.Combatant] = true
	}
	for id := range t.spent {
		if !seen[id] {
			return false
		}
	}
	for id := range t.dodging {
		if !seen[id] {
			return false
		}
	}
	for id := range t.readied {
		if !seen[id] {
			return false
		}
	}
	for k := range t.feats {
		if !seen[k.id] {
			return false
		}
	}
	return true
}

// wellFormed reports whether every part of the combatant passes its own validation (plan P14): an id, a kind,
// a side, Hit Points inside their bounds with a life state and rule that agree (the checks of
// RestoreHitPointStatus) and a condition set of valid conditions held under their own kinds. It does not
// check consistency between the life state, the markers and the conditions (FR-083).
func (c Combatant) wellFormed() bool {
	if c.ref.ID.IsZero() || c.ref.Kind == nil || !c.side.Valid() || c.hp.IsZero() || !c.conditions.wellFormed() {
		return false
	}
	h := c.hp.hp
	if _, err := NewHitPoints(h.current, h.maximum, h.temporary); err != nil {
		return false
	}
	_, err := RestoreHitPointStatus(h, c.hp.life, c.hp.rule, c.hp.aff)
	return err == nil
}

// wellFormed reports whether every held condition is valid and stored under its own kind, and the
// immunity bits name valid kinds only.
func (s ConditionSet) wellFormed() bool {
	var valid kindMask
	for k := content.ConditionBlinded; k <= content.ConditionUnconscious; k++ {
		valid |= bit(k)
	}
	if s.immune&^valid != 0 {
		return false
	}
	for i, c := range s.slots {
		if c != nil && (!validCondition(c) || int(c.Kind()) != i) {
			return false
		}
	}
	return true
}
