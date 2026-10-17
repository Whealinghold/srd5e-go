// Package modifier holds the d20 vocabulary shared by every rules context:
// [Modifier] (a bonus or penalty), [RollMode] (normal, advantage or disadvantage)
// and [RollCircumstance] (the [Advantage] and [Disadvantage] inputs that [Resolve]
// turns into a [RollMode]).
//
// All values are immutable and comparable and all functions are pure. The
// zero Modifier is +0 and the zero RollMode is Normal, so both are useful.
// [Resolve] never fails; [RollMode.Select] validates untrusted dice values and
// returns an [*InvalidError] (matching [srderr.ErrInvalidInput]).
//
// This package imports no other rules package: it does not roll dice, it
// only combines numbers a caller obtained through a [dice.Roller].
package modifier
