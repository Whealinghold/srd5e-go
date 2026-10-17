// Package srd5e is a pure-Go implementation of the fifth-edition rules in
// System Reference Document 5.2.1, intended to be imported by other projects.
//
// This package contains documentation only: no logic, aliases or re-exports.
// Import the packages below directly.
//
// # Package map
//
// Kernel packages (shared vocabulary, no context imports):
//
//   - [github.com/Whealinghold/srd5e-go/dice]: dice expressions, rolls, and the Roller port
//   - [github.com/Whealinghold/srd5e-go/ability]: ability enum, scores and derived modifiers
//   - [github.com/Whealinghold/srd5e-go/modifier]: modifiers, advantage/disadvantage, roll circumstances
//   - [github.com/Whealinghold/srd5e-go/srderr]: shared error vocabulary: port failures and the invalid-input sentinel
//   - [github.com/Whealinghold/srd5e-go/telemetry]: Logger, Tracer and Meter ports with no-op implementations
//
// Bounded contexts (import kernel packages and upstream contexts only):
//
//   - [github.com/Whealinghold/srd5e-go/d20]: d20 test resolution
//
// Test support, imported by tests only: [github.com/Whealinghold/srd5e-go/dice/dicetest],
// [github.com/Whealinghold/srd5e-go/telemetry/telemetrytest].
// Optional adapters: [github.com/Whealinghold/srd5e-go/telemetry/slogtelemetry] (stdlib log/slog),
// [github.com/Whealinghold/srd5e-go/dice/randroller] (unseeded, concurrency-safe roller).
//
// # Usage
//
// Randomness and time enter only through injected ports; there is no global
// state. Expected rule outcomes are returned as values, invalid input is
// reported as an error, and panics are reserved for programmer errors in
// MustX helpers. See the Example functions in each package.
//
// # Compatibility
//
// The module follows semantic versioning through git tags and stays on v0/v1
// until a breaking v2. See doc/rules/go.md for the Go compatibility promise
// and CHANGELOG.md for release history.
//
// # Licensing
//
// Code is MIT licensed (LICENSE). Rules content derives from SRD 5.2.1,
// licensed under CC BY 4.0; the required attribution is in NOTICE.md.
package srd5e
