package combat

import "slices"

// HeldSpell is one entry the host gives an OngoingSpellRegistry: an ongoing spell
// and the target it is on.
type HeldSpell struct {
	Target DispelTarget
	Spell  OngoingSpell
}

// OngoingSpellRegistry is the read model of the spells in effect, by target
// (dispel-and-counterspell.md section 4.1). The host builds it from the active
// effects; it is not a lifetime store, and it holds no clock. It is immutable:
// the entries are copied in and out, and it has no way to add, end or age a spell.
// The zero value is the empty registry. Safe for concurrent use.
type OngoingSpellRegistry struct{ held []HeldSpell }

// NewOngoingSpellRegistry returns the registry of held, in the order given. It
// returns an error matching srderr.ErrInvalidInput for an entry with a zero
// target or a zero spell, and when two spells in it share an OngoingEffectID, and a LimitExceededError for more than 256 entries (plan P10).
// It copies the entries and never panics.
func NewOngoingSpellRegistry(held []HeldSpell) (OngoingSpellRegistry, error) {
	if err := overCap("ongoing spells", len(held), maxInputList); err != nil {
		return OngoingSpellRegistry{}, err
	}
	seen := make(map[OngoingEffectID]bool, len(held))
	for _, h := range held {
		if !validDispelTarget(h.Target) || h.Spell.IsZero() {
			return OngoingSpellRegistry{}, invalidInput("a registry entry needs a target and a spell")
		}
		id := h.Spell.Effect().ID()
		if seen[id] {
			return OngoingSpellRegistry{}, invalidInput("two ongoing spells share an effect id")
		}
		seen[id] = true
	}
	return OngoingSpellRegistry{slices.Clone(held)}, nil
}

// SpellsOn returns the spells on target in registry order, or an empty list for a
// target the registry does not hold. The result is a copy.
func (r OngoingSpellRegistry) SpellsOn(target DispelTarget) []OngoingSpell {
	var out []OngoingSpell
	for _, h := range r.held {
		if h.Target == target {
			out = append(out, h.Spell)
		}
	}
	return out
}

// Equal reports whether r and o hold equal entries in the same order.
func (r OngoingSpellRegistry) Equal(o OngoingSpellRegistry) bool {
	if len(r.held) != len(o.held) {
		return false
	}
	for i := range r.held {
		if r.held[i].Target != o.held[i].Target || !r.held[i].Spell.Equal(o.held[i].Spell) {
			return false
		}
	}
	return true
}

// without returns the registry minus the spells whose effect ids are in ids,
// order kept. It is the registry a Dispel Magic conclusion leaves behind.
func (r OngoingSpellRegistry) without(ids []OngoingEffectID) OngoingSpellRegistry {
	var kept []HeldSpell
	for _, h := range r.held {
		if !slices.Contains(ids, h.Spell.Effect().ID()) {
			kept = append(kept, h)
		}
	}
	return OngoingSpellRegistry{kept}
}

// holdsOn reports whether the registry holds the effect id on target.
func (r OngoingSpellRegistry) holdsOn(target DispelTarget, id OngoingEffectID) bool {
	for _, h := range r.held {
		if h.Target == target && h.Spell.Effect().ID() == id {
			return true
		}
	}
	return false
}
