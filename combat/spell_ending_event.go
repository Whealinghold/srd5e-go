package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellEndingEvent is the closed family of events that Dispel Magic and
// Counterspell produce (dispel-and-counterspell.md FR-054). Its variants are
// OngoingSpellEnded, SpellCountered and CounterspellResisted, which embed the base
// spellEndingEventBase that carries the marker methods. Each is an immutable value
// named in the past tense; it is also a combat Event.
type SpellEndingEvent interface {
	Event
	spellEndingEvent()
}

// spellEndingEventBase puts an event in the Event and SpellEndingEvent families.
type spellEndingEventBase struct{}

func (spellEndingEventBase) event()            {}
func (spellEndingEventBase) spellEndingEvent() {}

// SpellEndCause is why an ongoing spell ended by a Dispel Magic conclusion. The
// zero value is invalid.
type SpellEndCause int

const (
	// SpellEndDispelled: Dispel Magic ended the spell (SRD 5.2.1, "Dispel Magic", p. 124).
	SpellEndDispelled SpellEndCause = iota + 1
	// SpellEndAllPartsEnded: every part of a multi-part spell is gone, so the parent ended
	// (SRD 5.2.1, "Guards and Wards", p. 138).
	SpellEndAllPartsEnded
)

// String returns "dispelled", "all_parts_ended" or "unknown".
func (c SpellEndCause) String() string {
	return enumName([]string{"dispelled", "all_parts_ended"}, int(c))
}

// OngoingSpellEnded is the event of an ongoing spell ending by Dispel Magic. It
// carries what the host needs to end the spell's effects: the target, the effect
// id, the spell and its level, the cause, the conditions the spell imposed (which
// the host removes, FR-025), and whether the caster's Concentration ends with it
// (the host then calls BreakConcentration once, FR-026). Combatants names the holder
// and the caster.
type OngoingSpellEnded struct {
	spellEndingEventBase
	target                  DispelTarget
	effect                  OngoingEffectID
	spell                   content.SpellID
	castAtLevel             content.SpellLevel
	cause                   SpellEndCause
	conditions              []content.ConditionKind
	endsCasterConcentration bool
	caster                  CombatantID
}

// Kind returns "ongoing_spell_ended".
func (OngoingSpellEnded) Kind() string { return "ongoing_spell_ended" }

// Combatants returns the holder when it is a creature, then the caster when known
// and different from the holder.
func (e OngoingSpellEnded) Combatants() []CombatantID {
	var out []CombatantID
	if c, ok := e.target.(DispelCreature); ok {
		out = append(out, c.id)
	}
	if !e.caster.IsZero() && (len(out) == 0 || out[0] != e.caster) {
		out = append(out, e.caster)
	}
	return out
}

// Target returns the target the spell was on.
func (e OngoingSpellEnded) Target() DispelTarget { return e.target }

// EffectID returns the id of the ended effect.
func (e OngoingSpellEnded) EffectID() OngoingEffectID { return e.effect }

// SpellID returns the id of the ended spell.
func (e OngoingSpellEnded) SpellID() content.SpellID { return e.spell }

// CastAtLevel returns the level the ended spell took when cast.
func (e OngoingSpellEnded) CastAtLevel() content.SpellLevel { return e.castAtLevel }

// Cause returns why the spell ended.
func (e OngoingSpellEnded) Cause() SpellEndCause { return e.cause }

// ConditionsToRemove returns a copy of the conditions the spell imposed on its holder.
func (e OngoingSpellEnded) ConditionsToRemove() []content.ConditionKind {
	return slices.Clone(e.conditions)
}

// EndsCasterConcentration reports whether the spell required Concentration and its
// caster is known, so the caster's Concentration ends with it.
func (e OngoingSpellEnded) EndsCasterConcentration() bool { return e.endsCasterConcentration }

// Equal reports whether e and o are equal in every field.
func (e OngoingSpellEnded) Equal(o OngoingSpellEnded) bool {
	return e.target == o.target && e.effect == o.effect && e.spell == o.spell &&
		e.castAtLevel == o.castAtLevel && e.cause == o.cause &&
		slices.Equal(e.conditions, o.conditions) &&
		e.endsCasterConcentration == o.endsCasterConcentration && e.caster == o.caster
}

// SpellCountered is the event of a cast stopped by Counterspell. Combatants names the
// counterspeller, then the casting creature.
type SpellCountered struct {
	spellEndingEventBase
	counterspeller CombatantID
	caster         CombatantID
	spell          content.SpellID
}

// Kind returns "spell_countered".
func (SpellCountered) Kind() string { return "spell_countered" }

// Combatants returns the counterspeller, then the casting creature.
func (e SpellCountered) Combatants() []CombatantID {
	return []CombatantID{e.counterspeller, e.caster}
}

// SpellID returns the spell that was countered.
func (e SpellCountered) SpellID() content.SpellID { return e.spell }

// CounterspellResisted is the event of a Counterspell whose cast the caster's save
// resisted: it names the counterspeller, then the casting creature, and the spell. Its
// Go name is fixed by FR-054 and its Kind is "counterspell_resisted"; the result of
// the same name is CounterspellResistedResult.
type CounterspellResisted struct {
	spellEndingEventBase
	counterspeller CombatantID
	caster         CombatantID
	spell          content.SpellID
}

// Kind returns "counterspell_resisted".
func (CounterspellResisted) Kind() string { return "counterspell_resisted" }

// Combatants returns the counterspeller, then the casting creature.
func (e CounterspellResisted) Combatants() []CombatantID {
	return []CombatantID{e.counterspeller, e.caster}
}

// SpellID returns the spell Counterspell targeted.
func (e CounterspellResisted) SpellID() content.SpellID { return e.spell }
