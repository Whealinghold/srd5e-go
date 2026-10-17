// Package randroller provides a dice.Roller that is safe for concurrent use,
// for hosts (servers) that share one roller between goroutines.
//
// It draws from the process-wide, automatically seeded generator of
// math/rand/v2, so rolls are not reproducible; use dice.SeededRoller with a
// fixed seed for that, and dicetest rollers in tests. The package uses the
// standard library only and emits no telemetry.
package randroller
