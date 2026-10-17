// Package d20 resolves D20 Tests as SRD 5.2.1 defines them ("D20 Tests",
// p. 6): an ability check, a saving throw or an attack roll. A d20 is rolled
// (two under Advantage or Disadvantage), modifiers are added, the total is
// compared with a target number (a Difficulty Class or an Armor Class), and
// the test succeeds or fails.
//
// The package has two layers in one import path. The domain layer is pure:
// [Roll], [RollModifiers], [DifficultyClass], [ArmorClass], and the rules
// functions [ResolveCheck], [ResolveAttack] and [FailWithoutRoll]. It never
// rolls dice, reads a clock or emits telemetry. The use case layer is
// [Resolver]: it rolls the dice through a [Roller], calls the rules and
// emits telemetry.
//
// # Results, errors and panics
//
// A failed check, a missed attack or a failure without a roll is an expected
// rule outcome and a return value, never an error. The [Result] returned by
// [Resolver.Resolve] is a closed set of variants ([RolledCheck],
// [RolledAttack], [ForcedFailure]) to be handled with a type switch.
//
// Invalid input (an out-of-range face, an unknown kind, an attack roll paired
// with a Difficulty Class) yields an [InvalidError] that matches
// [srderr.ErrInvalidInput] with errors.Is. A failing or contract-breaking
// roller yields a [srderr.PortError]. No function in this package panics on
// untrusted input.
//
// # Decoupling
//
// The package imports only kernel packages (dice, ability, modifier, srderr,
// telemetry). The roller it needs is the one-method [Roller] interface
// declared here; any dice.Roller satisfies it, and so does a three-line fake.
// Telemetry defaults to no-ops.
package d20
