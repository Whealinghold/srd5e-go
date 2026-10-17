package d20

import "strconv"

const (
	naturalTwenty        = 20
	naturalOne           = 1
	lowestCriticalRange  = 18
	highestCriticalRange = 20
)

// DefaultCriticalRange is the critical range of an attack roll without a
// class feature: a natural 20 only ("Rolling 20 or 1", SRD 5.2.1, p. 7).
const DefaultCriticalRange = highestCriticalRange

// ResolveCheck resolves an ability check or saving throw of kind for roll with
// modifiers against the Difficulty Class target ("D20 Tests", SRD 5.2.1,
// p. 6). The outcome is [Success] when the total (the natural value plus the
// modifiers, unclamped) equals or exceeds target, and [Failure] otherwise,
// whatever the natural value: the natural 20 and natural 1 rules hold for
// attack rolls only.
//
// It returns an *InvalidError with ReasonKindMismatch when kind is
// [AttackRoll], ReasonUnknownKind when kind is not valid, and ReasonZeroRoll
// when roll is the zero value. A failed check is a normal result, not an error.
func ResolveCheck(kind Kind, roll Roll, modifiers RollModifiers, target DifficultyClass) (CheckResult, error) {
	switch {
	case !kind.Valid():
		return CheckResult{}, invalid(ReasonUnknownKind, "kind "+strconv.Itoa(int(kind)))
	case kind == AttackRoll:
		return CheckResult{}, invalid(ReasonKindMismatch, "an attack roll is compared with an Armor Class")
	case roll.IsZero():
		return CheckResult{}, invalid(ReasonZeroRoll, "")
	}
	total := modifiers.Total().ApplyTo(roll.natural)
	outcome := Failure
	if total >= int(target) {
		outcome = Success
	}
	return CheckResult{kind: kind, roll: roll, modifiers: modifiers, total: total, target: target, outcome: outcome}, nil
}

// ResolveAttack resolves an attack roll for roll with modifiers against the
// Armor Class target ("Rolling 20 or 1", SRD 5.2.1, p. 7).
//
// A natural 20 hits and is a Critical Hit whatever the modifiers or the AC; a
// natural 1 misses whatever they are. Any other natural value at or above
// criticalRange is also a Critical Hit that hits. Below it the total is
// compared with target and the attack is never a Critical Hit. criticalRange
// is [DefaultCriticalRange], or 19 or 18 with the features at p. 49.
//
// It returns an *InvalidError with ReasonCriticalRange when criticalRange is
// not 18, 19 or 20, and ReasonZeroRoll when roll is the zero value. A miss is
// a normal result, not an error.
func ResolveAttack(roll Roll, modifiers RollModifiers, target ArmorClass, criticalRange int) (AttackRollResult, error) {
	switch {
	case criticalRange < lowestCriticalRange || criticalRange > highestCriticalRange:
		return AttackRollResult{}, invalid(ReasonCriticalRange, "range "+strconv.Itoa(criticalRange)+" not in 18..20")
	case roll.IsZero():
		return AttackRollResult{}, invalid(ReasonZeroRoll, "")
	}
	total := modifiers.Total().ApplyTo(roll.natural)
	var critical bool
	var outcome Outcome
	switch {
	case roll.natural == naturalTwenty:
		critical, outcome = true, Success
	case roll.natural == naturalOne:
		outcome = Failure
	case roll.natural >= criticalRange:
		critical, outcome = true, Success
	case total >= int(target):
		outcome = Success
	default:
		outcome = Failure
	}
	return AttackRollResult{
		roll: roll, modifiers: modifiers, total: total, target: target,
		outcome: outcome, critical: critical, criticalRange: criticalRange,
	}, nil
}

// FailWithoutRoll resolves the failure of a check or saving throw without a
// roll: a saving throw the creature chooses to fail ([ChosenToFail]) or an
// automatic failure the caller declares ([Automatic]).
//
// It returns an *InvalidError with ReasonUnknownKind or ReasonUnknownReason
// for an invalid argument, and ReasonForcedFailureNotAllowed when kind is
// [AttackRoll] or when an [AbilityCheck] is given [ChosenToFail]: the SRD
// states the choice for saving throws only ("Saving Throws", p. 7).
func FailWithoutRoll(kind Kind, reason ForcedFailureReason) (ForcedFailureResult, error) {
	switch {
	case !kind.Valid():
		return ForcedFailureResult{}, invalid(ReasonUnknownKind, "kind "+strconv.Itoa(int(kind)))
	case reason != ChosenToFail && reason != Automatic:
		return ForcedFailureResult{}, invalid(ReasonUnknownReason, "reason "+strconv.Itoa(int(reason)))
	case kind == AttackRoll:
		return ForcedFailureResult{}, invalid(ReasonForcedFailureNotAllowed, "an attack roll cannot fail without rolling")
	case kind == AbilityCheck && reason == ChosenToFail:
		return ForcedFailureResult{}, invalid(ReasonForcedFailureNotAllowed, "only a saving throw can be chosen to fail")
	}
	return ForcedFailureResult{kind: kind, reason: reason}, nil
}
