package combat

import "github.com/Whealinghold/srd5e-go/content"

// ConcentrationHold is the request a host passes to the spellcasting rules to
// begin the Concentration that holding a readied spell requires (SRD 5.2.1, "Ready
// [Action]", p. 187). It names the spell and nothing else. The zero value is not a
// request.
type ConcentrationHold struct{ Spell content.SpellID }

// ReleaseHoldRequest is the request a host passes to the spellcasting rules to end
// the Concentration of a held spell that was released. The zero value is not a
// request.
type ReleaseHoldRequest struct{ Spell content.SpellID }

// DissipationCause is why a held spell dissipated, a closed enum. The zero value is
// not a cause.
type DissipationCause int

// The causes of a dissipation.
const (
	// DissipationConcentrationBroken: the caster's Concentration on the held spell ended.
	DissipationConcentrationBroken DissipationCause = iota + 1
	// DissipationReadyElapsed: the readied action expired at the start of the caster's next turn.
	DissipationReadyElapsed
)

// String returns "concentration_broken", "ready_elapsed" or "unknown".
func (c DissipationCause) String() string {
	return enumName([]string{"concentration_broken", "ready_elapsed"}, int(c))
}

// Valid reports whether c is one of the two causes.
func (c DissipationCause) Valid() bool {
	return c == DissipationConcentrationBroken || c == DissipationReadyElapsed
}

// HeldSpellEvent is the closed family of events the rules for readying a spell
// produce. Each is an immutable value with a constant snake_case Kind in the past
// tense whose Combatants is a fresh copy holding the caster. Variants: SpellHeld,
// HeldSpellReleased, HeldSpellDissipated. Each embeds heldSpellEventBase.
type HeldSpellEvent interface {
	Event
	heldSpellEvent()
}

// heldSpellEventBase carries the caster and the spell and puts an event in the
// Event and HeldSpellEvent families.
type heldSpellEventBase struct {
	caster CombatantID
	spell  content.SpellID
}

func (heldSpellEventBase) event()          {}
func (heldSpellEventBase) heldSpellEvent() {}

// Combatants returns a new slice holding the caster.
func (b heldSpellEventBase) Combatants() []CombatantID { return []CombatantID{b.caster} }

// Spell returns the held spell.
func (b heldSpellEventBase) Spell() content.SpellID { return b.spell }

// SpellHeld: the caster readied the spell and holds it. Its Kind is spell_held.
// Combatants is [caster].
type SpellHeld struct{ heldSpellEventBase }

// Kind returns "spell_held".
func (SpellHeld) Kind() string { return "spell_held" }

// HeldSpellReleased: the caster released the held spell with a Reaction. Its Kind is
// held_spell_released. Combatants is [caster].
type HeldSpellReleased struct{ heldSpellEventBase }

// Kind returns "held_spell_released".
func (HeldSpellReleased) Kind() string { return "held_spell_released" }

// HeldSpellDissipated: the held spell dissipated without taking effect and no
// resource is refunded. Its Kind is held_spell_dissipated. Combatants is [caster].
type HeldSpellDissipated struct {
	heldSpellEventBase
	cause DissipationCause
}

// Kind returns "held_spell_dissipated".
func (HeldSpellDissipated) Kind() string { return "held_spell_dissipated" }

// Cause returns why the spell dissipated.
func (e HeldSpellDissipated) Cause() DissipationCause { return e.cause }

// ReadyCombatantSpellResult is the closed set of outcomes of ReadySpell. Variants:
// ReadySpellApplied, ReadySpellNotAnAction.
type ReadyCombatantSpellResult interface{ isReadyCombatantSpellResult() }

// ReadySpellApplied: the spell is held. Spell is the readied spell for the host to
// keep beside the readied action (HoldsSpell), Hold the Concentration request, and
// Events exactly [SpellHeld].
type ReadySpellApplied struct {
	Spell  ReadiedSpell
	Hold   ConcentrationHold
	Events []HeldSpellEvent
}

// ReadySpellNotAnAction: the spell cannot be readied because its casting time is not
// a Magic action. CastingTime is the casting time given. No events.
type ReadySpellNotAnAction struct{ CastingTime content.CastingTime }

func (ReadySpellApplied) isReadyCombatantSpellResult()     {}
func (ReadySpellNotAnAction) isReadyCombatantSpellResult() {}

// ReadySpell readies spell for caster against trigger (SRD 5.2.1, "Ready [Action]",
// p. 187). Only a spell whose casting time is a Magic action can be readied: it
// returns ReadySpellApplied with the ReadiedSpell, a ConcentrationHold (also for a
// spell that needs no Concentration of its own) and the one event SpellHeld. Any
// other casting time, a nil one included, returns ReadySpellNotAnAction and no
// events. The cast and its resources, and ending another Concentration, belong to
// the spellcasting rules; the ids are not validated here, NewReadiedAction rejects
// invalid ones. It never panics.
func ReadySpell(caster CombatantID, trigger ReadyTriggerID, spell content.SpellID, castingTime content.CastingTime) ReadyCombatantSpellResult {
	if _, ok := castingTime.(content.CastingTimeMagicAction); !ok {
		return ReadySpellNotAnAction{castingTime}
	}
	return ReadySpellApplied{
		Spell:  ReadiedSpell{Spell: spell, Caster: caster, Trigger: trigger},
		Hold:   ConcentrationHold{Spell: spell},
		Events: []HeldSpellEvent{SpellHeld{heldSpellEventBase{caster, spell}}},
	}
}

// ReleaseCombatantSpellResult is the closed set of outcomes of ReleaseHeldSpell. Variants:
// ReleaseHeldSpellApplied, ReleaseHeldSpellNoHeldSpell,
// ReleaseHeldSpellNotAReadiedReaction, ReleaseHeldSpellReactionMismatch.
type ReleaseCombatantSpellResult interface{ isReleaseCombatantSpellResult() }

// ReleaseHeldSpellApplied: the held spell was released. Release asks the host to end
// the hold's Concentration; Events is exactly [HeldSpellReleased].
type ReleaseHeldSpellApplied struct {
	Release ReleaseHoldRequest
	Events  []HeldSpellEvent
}

// ReleaseHeldSpellNoHeldSpell: the readied action holds no spell. No events.
type ReleaseHeldSpellNoHeldSpell struct{}

// ReleaseHeldSpellNotAReadiedReaction: the Reaction was not spent on a readied
// action. No events.
type ReleaseHeldSpellNotAReadiedReaction struct{}

// ReleaseHeldSpellReactionMismatch: the Reaction is another caster's or answers
// another trigger. No events.
type ReleaseHeldSpellReactionMismatch struct{}

func (ReleaseHeldSpellApplied) isReleaseCombatantSpellResult()             {}
func (ReleaseHeldSpellNoHeldSpell) isReleaseCombatantSpellResult()         {}
func (ReleaseHeldSpellNotAReadiedReaction) isReleaseCombatantSpellResult() {}
func (ReleaseHeldSpellReactionMismatch) isReleaseCombatantSpellResult()    {}

// ReleaseHeldSpell releases a held spell with the caster's Reaction (SRD 5.2.1,
// "Ready [Action]", p. 187). It checks in order: reaction is not a ReactionTaken
// for a readied action (ReleaseHeldSpellNotAReadiedReaction); held holds no spell,
// a nil held included (ReleaseHeldSpellNoHeldSpell); the spell's caster or trigger
// differs from the reaction's combatant or ReadyTriggerID
// (ReleaseHeldSpellReactionMismatch); otherwise ReleaseHeldSpellApplied with the one
// event HeldSpellReleased. A refusal carries no events. It spends no Reaction. It
// never panics.
func ReleaseHeldSpell(held HeldSpellInput, reaction ReactionTaken) ReleaseCombatantSpellResult {
	fired, ok := reaction.Record().(ReadiedActionRecord)
	if !ok {
		return ReleaseHeldSpellNotAReadiedReaction{}
	}
	holds, ok := held.(HoldsSpell)
	if !ok {
		return ReleaseHeldSpellNoHeldSpell{}
	}
	s := holds.Spell
	if s.Caster != reaction.Combatants()[0] || s.Trigger != fired.Trigger {
		return ReleaseHeldSpellReactionMismatch{}
	}
	return ReleaseHeldSpellApplied{
		Release: ReleaseHoldRequest{Spell: s.Spell},
		Events:  []HeldSpellEvent{HeldSpellReleased{heldSpellEventBase{s.Caster, s.Spell}}},
	}
}

// DissipateCombatantSpellResult is the closed set of outcomes of DissipateHeldSpell.
// Variants: DissipateHeldSpellApplied, DissipateHeldSpellNoHeldSpell, DissipateHeldSpellInvalidCause.
type DissipateCombatantSpellResult interface{ isDissipateCombatantSpellResult() }

// DissipateHeldSpellApplied: the held spell dissipated. Events is exactly
// [HeldSpellDissipated].
type DissipateHeldSpellApplied struct{ Events []HeldSpellEvent }

// DissipateHeldSpellNoHeldSpell: the readied action holds no spell. No events.
type DissipateHeldSpellNoHeldSpell struct{}

// DissipateHeldSpellInvalidCause: the cause is not a DissipationCause (host input the engine cannot use). No events.
type DissipateHeldSpellInvalidCause struct{}

func (DissipateHeldSpellApplied) isDissipateCombatantSpellResult()      {}
func (DissipateHeldSpellNoHeldSpell) isDissipateCombatantSpellResult()  {}
func (DissipateHeldSpellInvalidCause) isDissipateCombatantSpellResult() {}

// DissipateHeldSpell dissipates a held spell without effect and with no resource
// refunded, because the caster's Concentration broke or the readied action
// elapsed (SRD 5.2.1, "Ready [Action]", p. 187). It returns
// DissipateHeldSpellApplied with the one event HeldSpellDissipated for a held spell,
// and DissipateHeldSpellNoHeldSpell and no events for HoldsNoSpell or a nil held.
// It returns DissipateHeldSpellInvalidCause and no events for a cause that is not valid, checked first.
// The host ends the hold's Concentration. It never panics.
func DissipateHeldSpell(held HeldSpellInput, cause DissipationCause) DissipateCombatantSpellResult {
	if !cause.Valid() {
		return DissipateHeldSpellInvalidCause{}
	}
	holds, ok := held.(HoldsSpell)
	if !ok {
		return DissipateHeldSpellNoHeldSpell{}
	}
	s := holds.Spell
	return DissipateHeldSpellApplied{Events: []HeldSpellEvent{HeldSpellDissipated{heldSpellEventBase{s.Caster, s.Spell}, cause}}}
}
