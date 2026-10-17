// Package ability holds the six abilities and the ability scores of a creature.
//
// "All creatures - characters and monsters - have six abilities that measure
// physical and mental characteristics" (SRD 5.2.1, "The Six Abilities", p. 5).
//
// [Ability] is a closed enum, [Score] is one validated value from 1 to 30, and
// [Scores] holds one [Score] for every [Ability]. All values are immutable and
// comparable. Untrusted input is validated by [NewScore], [NewScores], [Scores.With]
// and [ParseAbility], which return an [*InvalidError] (matching
// [srderr.ErrInvalidInput]) and never panic. [MustScore] panics for programmer
// error only. Zero values ([Ability], [Score], [Scores]) are explicitly invalid; test
// them with Valid or IsZero.
//
// This package imports no other rules package. A [Modifier] here is not the
// [modifier.Modifier] of package modifier; callers convert with
// modifier.New(m.Value()).
package ability
