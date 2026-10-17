// Package telemetry defines the observability ports of the module: [Logger],
// [Tracer] and [Meter], plus the vocabulary they share ([Attr], [Level], [Descriptor]).
//
// The ports use only standard-library types and are shaped so an OpenTelemetry,
// slog or zap adapter is a thin translation. They return no errors: a backend
// handles its own failures, and a backend that panics is the host's bug:
// srd5e does not recover.
//
// Everything defaults to a no-op. The zero values of [Noop], [NoopLogger],
// [NoopTracer], [NoopMeter] and [Telemetry] (after [Telemetry.WithDefaults]) are usable, hold no
// state and are safe for concurrent use. Nothing in this package reads the
// clock or a global; durations use an injected clock (see [Timer]).
//
// Test doubles and the contract suites live in package telemetrytest; the
// optional slog adapter lives in package slogtelemetry.
package telemetry
