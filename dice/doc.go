// Package dice is the dice vocabulary of the module: a [Die], an [Expression] such
// as "2d8+2", the [Result] of rolling one, and the [Roller] port through which all
// randomness enters.
//
// Values are immutable. Constructors and parsers validate their input and
// return an error that matches [srderr.ErrInvalidInput] and, with errors.As, is
// an [*InvalidError] carrying a typed [Reason]. Zero values ([Die], [DieRoll],
// [Expression], [Result]) are explicitly invalid; test them with IsZero.
//
// A [Roller] is a synchronous, in-memory port and takes no context. Use [Roll] to
// call one: it converts a roller's error, panic or wrong answer into a
// [*srderr.PortError]. Test doubles and the roller contract suite live in
// package dicetest.
//
// Nothing in this package is global mutable state; randomness comes only from
// the [Roller] a caller injects.
package dice
