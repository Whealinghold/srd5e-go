// Package telemetrytest provides test doubles and contract suites for the
// ports in package telemetry, in the manner of net/http/httptest.
//
// [Logger], [Tracer] and [Meter] record what a use case emits so tests can assert on
// it; [Recorder] bundles the three; [Clock] is a deterministic clock for
// [telemetry.Timer]. The recorders are safe for concurrent use and their zero
// values are ready to use.
//
// Misuse is strict by design but never panics: an invalid span or instrument
// name, an invalid attribute, a write to an ended span, a negative counter
// increment and a conflicting instrument redefinition are collected and
// returned by [Recorder.Violations]. Call [RequireNoViolations] at the end of a test to fail
// on them.
//
// [RunLoggerContract], [RunTracerContract] and [RunMeterContract] check that an
// implementation of a port honours its contract. Every adapter, including
// those a consumer writes, should pass them.
//
// Only test files import this package.
package telemetrytest
