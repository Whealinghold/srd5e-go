package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// OpenHandEffect is one of the effects of Open Hand Technique (SRD 5.2.1, "Level 3: Open
// Hand Technique", p. 52). The zero value is not an effect.
type OpenHandEffect int

// The Open Hand Technique effects.
const (
	// OpenHandAddle: the target can't make Opportunity Attacks until the start of its next turn.
	OpenHandAddle OpenHandEffect = iota + 1
	// OpenHandPush: a Strength saving throw or be pushed up to 15 feet away.
	OpenHandPush
	// OpenHandTopple: a Dexterity saving throw or have the Prone condition.
	OpenHandTopple
)

// String returns "addle", "push", "topple" or "unknown".
func (e OpenHandEffect) String() string {
	switch e {
	case OpenHandAddle:
		return "addle"
	case OpenHandPush:
		return "push"
	case OpenHandTopple:
		return "topple"
	}
	return "unknown"
}

// Valid reports whether e is one of the three effects.
func (e OpenHandEffect) Valid() bool { return e >= OpenHandAddle && e <= OpenHandTopple }

const openHandPushFeet = 15

// OpenHandTechniqueFacts is the hit Open Hand Technique rides on and the effects asked
// for. Construct it with a keyed literal.
type OpenHandTechniqueFacts struct {
	// HitByFlurryOfBlows reports a hit made with an attack granted by Flurry of Blows.
	HitByFlurryOfBlows bool
	// Effects are the effects the Monk imposes; exactly one is allowed per hit.
	Effects []OpenHandEffect
}

// OpenHandTechniqueResolution is the outcome of an Open Hand Technique effect. Only
// OpenHandRules.ResolveTechnique builds one.
type OpenHandTechniqueResolution struct {
	effect                  OpenHandEffect
	dc                      d20.DifficultyClass
	pushedFeet              int
	prone, noOpportunityAtk bool
}

// Effect returns the effect imposed.
func (r OpenHandTechniqueResolution) Effect() OpenHandEffect { return r.effect }

// DifficultyClass returns the save DC used (the Monk's Focus save DC, an interpretation
// recorded in class-monk.md section 9); 0 for Addle, which has no save.
func (r OpenHandTechniqueResolution) DifficultyClass() d20.DifficultyClass { return r.dc }

// PushedFeet returns the distance the target is pushed: 15 on a failed Strength save, else 0.
func (r OpenHandTechniqueResolution) PushedFeet() int { return r.pushedFeet }

// Prone reports that the target has the Prone condition: Topple on a failed Dexterity save.
func (r OpenHandTechniqueResolution) Prone() bool { return r.prone }

// NoOpportunityAttacks reports that the target can't make Opportunity Attacks until the
// start of its next turn: Addle.
func (r OpenHandTechniqueResolution) NoOpportunityAttacks() bool { return r.noOpportunityAtk }

// MonkBonusAction says which Bonus Action the Monk took, for Fleet Step. The zero value is not an action.
type MonkBonusAction int

// The Bonus Actions Fleet Step tells apart.
const (
	// MonkBonusActionStepOfTheWind: the Bonus Action was Step of the Wind.
	MonkBonusActionStepOfTheWind MonkBonusAction = iota + 1
	// MonkBonusActionOther: a Bonus Action other than Step of the Wind.
	MonkBonusActionOther
)

// FleetStepResult is the closed outcome of OpenHandRules.FleetStep. Variants:
// FleetStepAvailable, FleetStepRejected.
type FleetStepResult interface{ isFleetStepResult() }

// FleetStepAvailable: Step of the Wind is also available immediately after the Bonus
// Action; it keeps its own Focus Point cost and adds no Bonus Action cost.
type FleetStepAvailable struct{}

// Granted returns MonkStepOfTheWind, the feature that is also available.
func (FleetStepAvailable) Granted() MonkFocusFeature { return MonkStepOfTheWind }

// FleetStepRejected: Fleet Step does not apply. Construct it with a keyed literal.
type FleetStepRejected struct{ Reason MonkReason }

func (FleetStepAvailable) isFleetStepResult() {}
func (FleetStepRejected) isFleetStepResult()  {}

// OpenHandRules are the Warrior of the Open Hand's features that need no state: Open
// Hand Technique and Fleet Step (SRD 5.2.1, "Monk Subclass: Warrior of the Open Hand", p.
// 52). The subclass is an explicit input because the Character holds none (class-monk.md
// FR-148). It is stateless and pure; the zero value is ready to use. Safe for concurrent use.
type OpenHandRules struct{}

// TechniqueAvailability reports whether the Monk can impose an effect: MonkNotAMonk for
// zero stats, MonkLevelTooLow below level 3, MonkNotEligible for a subclass other than
// the Warrior of the Open Hand, a hit not made with an attack granted by Flurry of
// Blows, or other than exactly one valid effect.
func (OpenHandRules) TechniqueAvailability(stats MonkLevelStats, subclass content.SubclassID, facts OpenHandTechniqueFacts) MonkAvailability {
	return monkAvailability(
		monkGate(stats, openHandTechniqueLevel),
		monkIf(subclass != OpenHandSubclassID || !facts.HitByFlurryOfBlows || len(facts.Effects) != 1 || !facts.Effects[0].Valid(), MonkNotEligible),
	)
}

// ResolveTechnique imposes the effect. outcome is the target's saving throw for Push and
// Topple, and ignored for Addle. A failed save pushes the target up to 15 feet or knocks
// it Prone.
//
// It returns an error matching srderr.ErrInvalidInput for an outcome other than
// d20.Success or d20.Failure on a Push or Topple, or when TechniqueAvailability says
// unavailable. It never panics.
func (o OpenHandRules) ResolveTechnique(stats MonkLevelStats, subclass content.SubclassID, abilities MonkAbilities, facts OpenHandTechniqueFacts, outcome d20.Outcome) (OpenHandTechniqueResolution, error) {
	if _, ok := o.TechniqueAvailability(stats, subclass, facts).(MonkAvailable); !ok {
		return OpenHandTechniqueResolution{}, invalidInput("Open Hand Technique resolved while unavailable")
	}
	r := OpenHandTechniqueResolution{effect: facts.Effects[0]}
	if r.effect == OpenHandAddle {
		r.noOpportunityAtk = true
		return r, nil
	}
	if outcome != d20.Success && outcome != d20.Failure {
		return OpenHandTechniqueResolution{}, invalidInput("Open Hand Technique saving throw outcome is neither success nor failure")
	}
	r.dc = abilities.SaveDC()
	if outcome == d20.Failure && r.effect == OpenHandPush {
		r.pushedFeet = openHandPushFeet
	}
	r.prone = outcome == d20.Failure && r.effect == OpenHandTopple
	return r, nil
}

// FleetStep reports whether Step of the Wind is also available after the Bonus Action
// taken: it is after a Bonus Action other than Step of the Wind, from level 11 in the
// Warrior of the Open Hand. Otherwise FleetStepRejected carries the first reason:
// MonkNotAMonk, MonkLevelTooLow, MonkNotEligible (a different subclass, or Step of the
// Wind or an unknown action taken).
func (OpenHandRules) FleetStep(stats MonkLevelStats, subclass content.SubclassID, taken MonkBonusAction) FleetStepResult {
	if r := monkAvailability(monkGate(stats, openHandFleetStepLevel), monkIf(subclass != OpenHandSubclassID || taken != MonkBonusActionOther, MonkNotEligible)); r != (MonkAvailable{}) {
		return FleetStepRejected{Reason: r.(MonkUnavailable).Reason}
	}
	return FleetStepAvailable{}
}

// WholenessOfBodyResolution is the outcome of Wholeness of Body: the die rolled, the Hit
// Points to regain and the new state. The Hit Points are not capped by the Hit Point
// maximum, which the host does. Only WholenessOfBody.Resolve builds one.
type WholenessOfBodyResolution struct {
	rolled, hitPoints int
	state             MonkState
}

// Rolled returns the Martial Arts die rolled.
func (r WholenessOfBodyResolution) Rolled() int { return r.rolled }

// HitPointsToRegain returns the die rolled plus the Wisdom modifier, at least 1, uncapped.
func (r WholenessOfBodyResolution) HitPointsToRegain() int { return r.hitPoints }

// State returns the new state: one more use spent.
func (r WholenessOfBodyResolution) State() MonkState { return r.state }

// WholenessOfBody decides and resolves Wholeness of Body (SRD 5.2.1, "Level 6: Wholeness
// of Body", p. 52). It is stateless and pure; the zero value is ready to use. Safe for
// concurrent use.
type WholenessOfBody struct{}

// Availability reports whether the Monk can heal itself: MonkNotAMonk for zero stats,
// MonkLevelTooLow below level 6, MonkNotEligible for a subclass other than the Warrior of
// the Open Hand, MonkNoUsesRemaining with every use spent (the uses are the Wisdom
// modifier, minimum once).
func (WholenessOfBody) Availability(stats MonkLevelStats, state MonkState, subclass content.SubclassID, wisdom ability.Modifier) MonkAvailability {
	return monkAvailability(
		monkGate(stats, openHandWholenessLevel),
		monkIf(subclass != OpenHandSubclassID, MonkNotEligible),
		monkIf(state.WholenessOfBodyRemaining(wisdom) < 1, MonkNoUsesRemaining),
	)
}

// Resolve heals with the Martial Arts die roll rolled: the Hit Points to regain are rolled
// plus the Wisdom modifier, at least 1; one use is spent.
//
// It returns an error matching srderr.ErrInvalidInput for a rolled outside the Martial
// Arts die, or when Availability says unavailable. It never panics.
func (w WholenessOfBody) Resolve(stats MonkLevelStats, state MonkState, subclass content.SubclassID, wisdom ability.Modifier, rolled int) (WholenessOfBodyResolution, error) {
	if stats.IsZero() || rolled < 1 || rolled > stats.martialArts.Sides() {
		return WholenessOfBodyResolution{}, invalidInput("Wholeness of Body roll " + strconv.Itoa(rolled) + " outside the Martial Arts die, or zero stats")
	}
	if _, ok := w.Availability(stats, state, subclass, wisdom).(MonkAvailable); !ok {
		return WholenessOfBodyResolution{}, invalidInput("Wholeness of Body resolved while unavailable")
	}
	state.wholenessSpent++
	return WholenessOfBodyResolution{rolled: rolled, hitPoints: max(rolled+wisdom.Value(), 1), state: state}, nil
}
