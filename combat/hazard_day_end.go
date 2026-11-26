package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
)

const (
	// malnutritionSaveDC is the DC of the Constitution saving throw after eating less than half the food needed.
	malnutritionSaveDC d20.DifficultyClass = d20.DCEasy
	// starvationDays is the day without food on which a creature that eats nothing gains Exhaustion.
	starvationDays = 5
)

// FoodSavingThrow is the Constitution saving throw of a creature that ate less than
// half the food it needs. Variants: FoodSaveNotRolled, FoodSaveRolled. A nil
// FoodSavingThrow is FoodSaveNotRolled.
type FoodSavingThrow interface{ isFoodSavingThrow() }

// FoodSaveNotRolled: the save has not been rolled yet.
type FoodSaveNotRolled struct{}

// FoodSaveRolled: the host rolled the save with d20 and reports the Outcome.
// Construct it with a keyed literal.
type FoodSaveRolled struct{ Outcome d20.Outcome }

func (FoodSaveNotRolled) isFoodSavingThrow() {}
func (FoodSaveRolled) isFoodSavingThrow()    {}

// exhaustionApplication is what one attempt to give a creature an Exhaustion level
// did: the outcome, the new set and the condition events.
type exhaustionApplication struct {
	outcome HazardExhaustionOutcome
	set     ConditionSet
	events  []Event
}

// applyExhaustion gives the creature one Exhaustion level through
// ConditionSet.Apply (SRD 5.2.1, "Exhaustion", p. 181).
func applyExhaustion(set ConditionSet, target CombatantID) exhaustionApplication {
	one := ExhaustionCondition{ExhaustionLevel{minExhaustionLevel}}
	switch a := set.Apply(target, one).(type) {
	case ConditionAdded:
		var events []Event
		for _, e := range a.events {
			events = append(events, e)
		}
		return exhaustionApplication{HazardExhaustionGained, a.set, events}
	case ExhaustionAtMaximum:
		return exhaustionApplication{HazardExhaustionAtMaximum, set, nil}
	case ConditionImmune:
		return exhaustionApplication{HazardImmune, set, nil}
	default:
		panic(unreachableError("exhaustion application", a))
	}
}

// dayEnd is what a day-end rule decided: the outcome, the set, the record and the
// events, in order.
type dayEnd struct {
	outcome HazardExhaustionOutcome
	set     ConditionSet
	record  HazardRecord
	events  []Event
}

// dehydrationDayEnd is the rule of ResolveDehydrationDayEnd (SRD 5.2.1,
// "Dehydration [Hazard]", p. 181) for a creature that needs required eighths and
// drank drunk.
func dehydrationDayEnd(required, drunk int, set ConditionSet, target CombatantID, rec HazardRecord) dayEnd {
	rec = rec.clamped(set.exhaustionLevel())
	out := dayEnd{outcome: HazardNoExhaustion, set: set, record: rec}
	switch {
	case drunk >= required:
		if n := rec.dehydration; n > 0 {
			out.record.dehydration = 0
			out.events = append(out.events, DehydrationRecovered{creatureEnvironmentEventBase{target}, n})
		}
	case isLessThanHalfOf(drunk, required):
		a := applyExhaustion(set, target)
		out.outcome, out.set, out.events = a.outcome, a.set, a.events
		if a.outcome == HazardExhaustionGained {
			out.record.dehydration++
		}
	}
	out.record = out.record.clamped(out.set.exhaustionLevel())
	return out
}

// malnutritionDayEnd is the rule of ResolveMalnutritionDayEnd (SRD 5.2.1,
// "Malnutrition [Hazard]", p. 185). saveRequired is true, with nothing changed,
// when the creature ate some food but less than half and the save is not rolled.
func malnutritionDayEnd(required, eaten int, save FoodSavingThrow, set ConditionSet, target CombatantID,
	rec HazardRecord) (out dayEnd, saveRequired bool) {
	rec = rec.clamped(set.exhaustionLevel())
	out = dayEnd{outcome: HazardNoExhaustion, set: set, record: rec}
	gain := func() {
		a := applyExhaustion(set, target)
		out.outcome, out.set, out.events = a.outcome, a.set, a.events
		if a.outcome == HazardExhaustionGained {
			out.record.malnutrition++
		}
	}
	switch {
	case !isLessThanHalfOf(eaten, required):
		out.record.daysWithoutFood = 0
		if n := rec.malnutrition; eaten >= required && n > 0 {
			out.record.malnutrition = 0
			out.events = append(out.events, MalnutritionRecovered{creatureEnvironmentEventBase{target}, n})
		}
	case eaten > 0:
		rolled, ok := save.(FoodSaveRolled)
		if !ok {
			return out, true
		}
		out.record.daysWithoutFood = 0
		if rolled.Outcome != d20.Success {
			gain()
		}
	default:
		out.record.daysWithoutFood = satAdd(rec.daysWithoutFood, 1)
		if out.record.daysWithoutFood >= starvationDays {
			gain()
		}
	}
	out.record = out.record.clamped(out.set.exhaustionLevel())
	return out, false
}

// malnutritionSave is the saving throw MalnutritionSaveRequired publishes.
func malnutritionSave() (d20.DifficultyClass, ability.Ability) {
	return malnutritionSaveDC, ability.Constitution
}
