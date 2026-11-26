package combat

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/modifier"
)

// antitoxinDuration is how long Antitoxin works (SRD 5.2.1, "Antitoxin", p. 96).
const antitoxinDuration = time.Hour

// PoisonedSaveIntent says what a saving throw is for, as the host knows it. The zero value is
// invalid and gets nothing from Antitoxin.
type PoisonedSaveIntent int

// The three intents.
const (
	// PoisonedSaveAvoid: a save to avoid the Poisoned condition.
	PoisonedSaveAvoid PoisonedSaveIntent = iota + 1
	// PoisonedSaveEnd: a save to end the Poisoned condition.
	PoisonedSaveEnd
	// PoisonedSaveOther: any other saving throw.
	PoisonedSaveOther
)

// String returns "avoid", "end", "other" or "unknown".
func (i PoisonedSaveIntent) String() string {
	return enumName([]string{"avoid", "end", "other"}, int(i))
}

// AntitoxinEffect is the hour of Advantage a creature has after drinking Antitoxin (SRD 5.2.1,
// "Antitoxin", p. 96). It never touches a condition: it gives no immunity and removes nothing.
// It holds the time since the drink, which the host supplies through Advance. Immutable and
// comparable. The zero value is not an effect: its methods report an *ItemUseError.
type AntitoxinEffect struct {
	drinker CombatantID
	elapsed time.Duration
}

// IsDrunkBy reports whether id is the creature that drank the Antitoxin.
func (e AntitoxinEffect) IsDrunkBy(id CombatantID) bool { return !id.IsZero() && e.drinker == id }

// Elapsed returns the time since the drink.
func (e AntitoxinEffect) Elapsed() time.Duration { return e.elapsed }

// Active reports whether the effect is a drunk Antitoxin whose hour has not passed.
func (e AntitoxinEffect) Active() bool { return !e.drinker.IsZero() && e.elapsed < antitoxinDuration }

// AntitoxinDrink is the result of drinking Antitoxin. Not comparable.
type AntitoxinDrink struct {
	effect AntitoxinEffect
	events []ItemUseEvent
}

// Effect returns the new effect, elapsed zero. Drinking while active restarts the hour; the
// Advantage never stacks (Q-9).
func (d AntitoxinDrink) Effect() AntitoxinEffect { return d.effect }

// Cost returns ItemActionCostBonusAction.
func (AntitoxinDrink) Cost() ItemActionCost { return ItemUseRules{}.CostOf(ItemUseAntitoxin) }

// Consumption returns the one vial used up.
func (AntitoxinDrink) Consumption() ItemConsumption { return consumedOne(ItemUseAntitoxin) }

// Events returns a copy of the events: one AntitoxinDrunk.
func (d AntitoxinDrink) Events() []ItemUseEvent { return slices.Clone(d.events) }

// AntitoxinRules drinks Antitoxin. The zero value is ready to use. Pure; safe for concurrent use.
type AntitoxinRules struct{}

// Drink returns the effect of drinker drinking Antitoxin. It returns an *ItemUseError
// (matching srderr.ErrInvalidInput) with ItemUseFieldUser for a zero drinker. It never panics.
func (AntitoxinRules) Drink(drinker CombatantID) (AntitoxinDrink, error) {
	if drinker.IsZero() {
		return AntitoxinDrink{}, itemUseInvalid(ItemUseFieldUser, 0)
	}
	return AntitoxinDrink{AntitoxinEffect{drinker: drinker}, []ItemUseEvent{AntitoxinDrunk{itemUseEventBase{first: drinker}}}}, nil
}

// AntitoxinUpdate is an effect after time passed and the events it produced. Not comparable.
type AntitoxinUpdate struct {
	effect AntitoxinEffect
	events []ItemUseEvent
}

// Effect returns the new effect.
func (u AntitoxinUpdate) Effect() AntitoxinEffect { return u.effect }

// Events returns a copy of the events: one AntitoxinWoreOff when the hour ended.
func (u AntitoxinUpdate) Events() []ItemUseEvent { return slices.Clone(u.events) }

// Advance adds d to the elapsed time and, on reaching 1 hour, emits one AntitoxinWoreOff
// (exactly one hour is expired, Q-8). An expired effect is returned unchanged with no event.
// The sum saturates.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with ItemUseFieldElapsed for
// a negative d and ItemUseFieldEffect for the zero effect. It never panics.
func (e AntitoxinEffect) Advance(d time.Duration) (AntitoxinUpdate, error) {
	switch {
	case e.drinker.IsZero():
		return AntitoxinUpdate{}, itemUseInvalid(ItemUseFieldEffect, 0)
	case d < 0:
		return AntitoxinUpdate{}, itemUseInvalid(ItemUseFieldElapsed, int64(d))
	case !e.Active():
		return AntitoxinUpdate{effect: e}, nil
	}
	e.elapsed = satAddDuration(e.elapsed, d)
	if e.Active() {
		return AntitoxinUpdate{effect: e}, nil
	}
	return AntitoxinUpdate{e, []ItemUseEvent{AntitoxinWoreOff{itemUseEventBase{first: e.drinker}}}}, nil
}

// AntitoxinAdvantage is what the effect gives a saving throw. Variants:
// AntitoxinGivesAdvantage, AntitoxinGivesNothing.
type AntitoxinAdvantage interface{ isAntitoxinAdvantage() }

// AntitoxinGivesAdvantage: the save has Advantage. It does not stack with Advantage from
// another source: the roll mode is resolved by modifier.Resolve.
type AntitoxinGivesAdvantage struct{}

// Circumstance returns modifier.Advantage, for the save's circumstances.
func (AntitoxinGivesAdvantage) Circumstance() modifier.RollCircumstance { return modifier.Advantage{} }

// AntitoxinNothingReason is why the effect gives nothing. The zero value is no reason.
type AntitoxinNothingReason int

// The reasons.
const (
	// AntitoxinNothingNotAPoisonedSave: the save is not to avoid or end Poisoned.
	AntitoxinNothingNotAPoisonedSave AntitoxinNothingReason = iota + 1
	// AntitoxinNothingExpired: the hour has passed.
	AntitoxinNothingExpired
)

// String returns "not_a_poisoned_save", "expired" or "unknown".
func (r AntitoxinNothingReason) String() string {
	return enumName([]string{"not_a_poisoned_save", "expired"}, int(r))
}

// AntitoxinGivesNothing: the save gets nothing from the effect.
type AntitoxinGivesNothing struct{ reason AntitoxinNothingReason }

// Reason returns why.
func (a AntitoxinGivesNothing) Reason() AntitoxinNothingReason { return a.reason }

func (AntitoxinGivesAdvantage) isAntitoxinAdvantage() {}
func (AntitoxinGivesNothing) isAntitoxinAdvantage()   {}

// AdvantageFor returns AntitoxinGivesAdvantage for a save to avoid or end Poisoned while the
// effect is active (a save against any source of Poisoned, not only poison), else
// AntitoxinGivesNothing: not a poisoned save is checked first, then an expired effect.
func (e AntitoxinEffect) AdvantageFor(intent PoisonedSaveIntent) AntitoxinAdvantage {
	switch {
	case intent != PoisonedSaveAvoid && intent != PoisonedSaveEnd:
		return AntitoxinGivesNothing{AntitoxinNothingNotAPoisonedSave}
	case !e.Active():
		return AntitoxinGivesNothing{AntitoxinNothingExpired}
	}
	return AntitoxinGivesAdvantage{}
}
