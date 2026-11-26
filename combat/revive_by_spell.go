package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ReviveBySpellCommand is the host's facts for a revival spell cast on a creature.
// Construct it with a keyed literal; every slice is read and never changed.
type ReviveBySpellCommand struct {
	// Spell is the revival spell cast; the zero value is not one.
	Spell RevivalSpell
	// Subject is the creature the spell is cast on.
	Subject CombatantID
	// Status is the creature's Hit Point status; it must be Dead to be revived.
	Status HitPointStatus
	// RestoredMaximum is the Hit Point maximum a GM rules the spell restores for a
	// creature that died of a maximum of 0, or nil. It is read only for such a creature.
	RestoredMaximum *int
	// Conditions and Effects are what the creature had at death.
	Conditions ConditionSet
	Effects    OngoingEffects
	// Circumstances are the facts of the death; the zero value is valid.
	Circumstances DeathCircumstances
	// NameSpoken is whether the caster speaks the creature's name (True Resurrection).
	NameSpoken bool
	// Consent is the spirit's answer; only SpiritAccepts revives.
	Consent RevivalConsent
	// Poisons says which held effects are poisons.
	Poisons PoisonsAtDeath
	// ElapsedConditions and ElapsedEffects are what the host reports elapsed on its own.
	ElapsedConditions []content.ConditionKind
	ElapsedEffects    []OngoingEffectID
}

// ReviveCreatureBySpellResult is the closed set of answers of ReviveBySpell. Variants:
// ReviveBySpellApplied, ReviveBySpellIneligible, ReviveBySpellNotRevived,
// ReviveBySpellInvalidMaximum, ReviveBySpellLimitExceeded.
type ReviveCreatureBySpellResult interface{ isReviveCreatureBySpellResult() }

// ReviveBySpellApplied: the creature returned to life. Revival holds the status,
// conditions, effects and events of the return; Ordeal and Strain are the ordeal
// the creature and the strain the caster now carry; RestoredToNonUndead is true when
// True Resurrection returned a creature that was Undead in its non-Undead form.
// Construct it with a keyed literal.
type ReviveBySpellApplied struct {
	Revival             ReviveCreatureApplied
	Ordeal              ReviveOrdeal
	Strain              CasterStrain
	RestoredToNonUndead bool
}

// ReviveBySpellIneligible: the spell cannot work on this creature; nothing changed,
// no event. A command whose Spell is not a revival spell gives it with no reasons.
type ReviveBySpellIneligible struct{ reasons []RevivalIneligibility }

// Reasons returns a new slice of every failed reason, in the fixed order of
// RevivalIneligibility.
func (r ReviveBySpellIneligible) Reasons() []RevivalIneligibility { return slices.Clone(r.reasons) }

// ReviveBySpellNotRevived: the cast was eligible but the creature did not return.
// Revival is the unchanged answer of Revive (not Dead, the spirit refused, or a
// maximum below 1). Strain is Strained when the spirit refused a Resurrection of
// a creature dead 365 days or more, StrainNone otherwise. Construct it with a keyed
// literal.
type ReviveBySpellNotRevived struct {
	Revival ReviveCreatureResult
	Strain  CasterStrain
}

// ReviveBySpellInvalidMaximum: the status is Dead of a Hit Point maximum of 0 and
// RestoredMaximum is below 1; nothing changed.
type ReviveBySpellInvalidMaximum struct{ RestoredMaximum int }

// ReviveBySpellLimitExceeded: the command holds more than 256 ElapsedConditions, 256 ElapsedEffects or 256 Poisons.Effects
// (plan P10); nothing changed. The shared limit_exceeded value, class refused through the wired use case.
type ReviveBySpellLimitExceeded struct{}

func (ReviveBySpellApplied) isReviveCreatureBySpellResult()        {}
func (ReviveBySpellIneligible) isReviveCreatureBySpellResult()     {}
func (ReviveBySpellNotRevived) isReviveCreatureBySpellResult()     {}
func (ReviveBySpellInvalidMaximum) isReviveCreatureBySpellResult() {}
func (ReviveBySpellLimitExceeded) isReviveCreatureBySpellResult()  {}

// ReviveBySpell casts a revival spell on a creature (SRD 5.2.1, "Dead", p. 180, and
// the four spells). It decides in this order: the spell is not a revival spell or
// the cast is ineligible (ReviveBySpellIneligible, every failed reason, before the
// spirit's consent); a RestoredMaximum below 1 for a creature that died of a
// maximum of 0 (ReviveBySpellInvalidMaximum); then Revive with 1 Hit Point (Revivify,
// Raise Dead) or the Hit Point maximum (Resurrection, True Resurrection), the poisons
// of Raise Dead, Resurrection and True Resurrection added to what the host reports
// elapsed and, for True Resurrection, every curse and magical contagion. A returned
// creature is ReviveBySpellApplied; anything else is ReviveBySpellNotRevived.
//
// A creature that died of a maximum of 0 returns only when RestoredMaximum is set:
// it is ignored for every other status. The function is pure, never panics, never
// changes its inputs, returns copied slices and emits only the events of Revive.
func ReviveBySpell(c ReviveBySpellCommand) ReviveCreatureBySpellResult {
	if len(c.ElapsedConditions) > maxInputList || len(c.ElapsedEffects) > maxInputList || len(c.Poisons.Effects) > maxInputList {
		return ReviveBySpellLimitExceeded{}
	}
	if !c.Spell.Valid() {
		return ReviveBySpellIneligible{}
	}
	if reasons := c.Spell.ineligibilities(c.Circumstances, c.NameSpoken); len(reasons) > 0 {
		return ReviveBySpellIneligible{reasons}
	}
	status := c.Status
	if d, isDead := status.life.(Dead); isDead && d.cause == HitPointMaximumZero && status.hp.maximum == 0 && c.RestoredMaximum != nil {
		if *c.RestoredMaximum < 1 {
			return ReviveBySpellInvalidMaximum{*c.RestoredMaximum}
		}
		// A Dead status of maximum 0 holds 0 Hit Points and no Temporary Hit Points, so this cannot fail.
		status, _ = RestoreHitPointStatus(HitPoints{0, *c.RestoredMaximum, 0}, d, status.rule, status.aff)
	}
	f := c.Spell.Facts()
	hitPoints := 1
	if f.allHitPoints {
		hitPoints = max(1, status.hp.maximum)
	}
	elapsedConditions := slices.Clone(c.ElapsedConditions)
	elapsedEffects := slices.Clone(c.ElapsedEffects)
	if f.neutralisesPoison {
		elapsedEffects = append(elapsedEffects, c.Poisons.Effects...)
		if c.Poisons.PoisonedConditionFromPoison {
			elapsedConditions = append(elapsedConditions, content.ConditionPoisoned)
		}
	}
	if f.liftsCurses {
		for _, e := range c.Effects.list {
			if e.category == OngoingEffectCurse || e.category == OngoingEffectMagicalContagion {
				elapsedEffects = append(elapsedEffects, e.id)
			}
		}
	}
	revival := Revive(ReviveCreatureCommand{Subject: c.Subject, Status: status, HitPoints: hitPoints, Consent: c.Consent,
		Conditions: c.Conditions, Effects: c.Effects, ElapsedConditions: elapsedConditions, ElapsedEffects: elapsedEffects})
	strain := StrainNone
	if f.strainAfter > 0 && c.Circumstances.timeSinceDeath >= f.strainAfter {
		strain = Strained
	}
	switch r := revival.(type) {
	case ReviveCreatureApplied:
		ordeal := ReviveOrdeal{}
		if f.ordeal {
			ordeal = FreshOrdeal()
		}
		return ReviveBySpellApplied{r, ordeal, strain, c.Spell == TrueResurrection && c.Circumstances.wasUndead}
	case ReviveCreatureRefused:
		return ReviveBySpellNotRevived{r, strain}
	}
	return ReviveBySpellNotRevived{revival, StrainNone}
}
