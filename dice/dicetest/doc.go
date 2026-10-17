// Package dicetest provides test doubles and a contract suite for
// [dice.Roller], in the manner of net/http/httptest.
//
// [FixedRoller] supplies scripted, minimum or maximum faces and records the
// requests it saw. [CallbackRoller] adapts a function. [RunRollerContract] and
// [RunFailureContract] are the suites every [dice.Roller] adapter and double must
// pass. None of this is for production use.
package dicetest
