package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// GrappleCue says that a grappler became Incapacitated or died, so the grapple of the grappled combatant
// is due for a check (combat-action-resolution.md FR-120). Immutable and comparable.
type GrappleCue struct{ Grappled, Grappler CombatantID }

// ConsequenceRules decides which prompts follow a call from the state before and after it
// (combat-action-resolution.md FR-085, FR-120). It adds no encounter state, is pure and stateless; the
// zero value is usable. The combatants are the encounter's, in turn order.
type ConsequenceRules struct{}

// changesSpeed reports whether the kind changes a Speed (SRD 5.2.1, "Speed", p. 188).
func changesSpeed(k content.ConditionKind) bool {
	return slices.Contains([]content.ConditionKind{content.ConditionExhaustion, content.ConditionGrappled, content.ConditionParalyzed,
		content.ConditionPetrified, content.ConditionRestrained, content.ConditionUnconscious}, k)
}

// speedKinds returns the Speed-changing kinds the set holds, as a bit mask, and the Exhaustion level.
func speedKinds(s ConditionSet) (kindMask, int) {
	var m kindMask
	for _, k := range s.Effective() {
		if changesSpeed(k) {
			m |= bit(k)
		}
	}
	return m, s.effectiveExhaustionLevel()
}

// SpeedReviews returns, in turn order, the combatants whose conditions gained or lost a kind that changes
// Speed (Exhaustion, Grappled, Paralyzed, Petrified, Restrained, Unconscious), or whose Exhaustion level
// changed, between before and after: one entry each, none for Prone, Poisoned or any other kind. Both
// slices hold the same combatants in the same order; a combatant missing from after is ignored.
func (ConsequenceRules) SpeedReviews(before, after []Combatant) []CombatantID {
	var out []CombatantID
	for i := range min(len(before), len(after)) {
		bk, bl := speedKinds(before[i].conditions)
		ak, al := speedKinds(after[i].conditions)
		if bk != ak || bl != al {
			out = append(out, after[i].ref.ID)
		}
	}
	return out
}

// cannotAct reports whether the combatant cannot act: effectively Incapacitated or dead.
func cannotAct(c Combatant) bool {
	return (ConditionEffects{}).ActiveFlags(c.conditions).Actions == ConditionActionAvailabilityNone || c.hp.isDead()
}

// GrappleChecks returns, in turn order of the grappled combatants, one cue for every combatant that holds
// Grappled whose grappler became Incapacitated or died during the call. A grappler already Incapacitated
// or dead before the call gives none. It removes no condition.
func (ConsequenceRules) GrappleChecks(before, after []Combatant) []GrappleCue {
	became := map[CombatantID]bool{}
	for i := range min(len(before), len(after)) {
		if !cannotAct(before[i]) && cannotAct(after[i]) {
			became[after[i].ref.ID] = true
		}
	}
	var cues []GrappleCue
	for _, c := range after {
		if g, ok := c.conditions.slots[content.ConditionGrappled].(Grappled); ok && became[g.Grappler] {
			cues = append(cues, GrappleCue{c.ref.ID, g.Grappler})
		}
	}
	return cues
}

// ConcentrationStep is what the rules decide for the Concentration of a damaged combatant.
type ConcentrationStep int

// The decisions, in the order the rules ask (combat-action-resolution.md FR-078).
const (
	// ConcentrationStepNone: no check applies; no die is rolled.
	ConcentrationStepNone ConcentrationStep = iota + 1
	// ConcentrationStepBreak: Concentration ends without a save, for the reason of the decision.
	ConcentrationStepBreak
	// ConcentrationStepSave: the Constitution saving throw of FR-131 decides.
	ConcentrationStepSave
)

// ConcentrationDecision is the answer of ConsequenceRules.Concentration. Immutable and comparable.
type ConcentrationDecision struct {
	step   ConcentrationStep
	reason character.ConcentrationBreakReason
}

// Step returns what to do.
func (d ConcentrationDecision) Step() ConcentrationStep { return d.step }

// Reason returns why Concentration ends for ConcentrationStepBreak: BreakDead or BreakIncapacitated; it is the
// zero value otherwise.
func (d ConcentrationDecision) Reason() character.ConcentrationBreakReason { return d.reason }

// Concentration decides, for the combatant as it is after the damage was committed and for damageTaken, the
// DamageTaken.Total of the event (FR-132), in this order: no tracking, no Concentration in progress or no
// damage taken is no check; a dead combatant ends it (BreakDead); an Incapacitated one, which Unconscious
// implies, ends it (BreakIncapacitated); damage that cannot break it is no check; otherwise the save is due
// (SRD 5.2.1, "Concentration", p. 179). It rolls nothing and never panics.
func (ConsequenceRules) Concentration(c Combatant, in ConcentrationInput, damageTaken int) ConcentrationDecision {
	t, ok := in.(TracksConcentration)
	if !ok || damageTaken <= 0 {
		return ConcentrationDecision{step: ConcentrationStepNone}
	}
	if _, going := t.state.Concentration(); !going {
		return ConcentrationDecision{step: ConcentrationStepNone}
	}
	switch {
	case c.hp.isDead():
		return ConcentrationDecision{ConcentrationStepBreak, character.BreakDead}
	case (ConditionEffects{}).ActiveFlags(c.conditions).Concentration == ConditionConcentrationBroken:
		return ConcentrationDecision{ConcentrationStepBreak, character.BreakIncapacitated}
	case t.rule == DamageCannotBreakConcentration:
		return ConcentrationDecision{step: ConcentrationStepNone}
	}
	return ConcentrationDecision{step: ConcentrationStepSave}
}
