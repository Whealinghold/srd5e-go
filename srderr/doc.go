// Package srderr is the shared error vocabulary of the module.
//
// It separates three things that other languages tend to blur:
//
//   - A port failure (a host-supplied collaborator returned an error,
//     panicked or broke its contract) is a [*PortError], built by [WrapPort],
//     [Guard], [GuardValue] or [ContractViolation]. Branch on it with errors.As and
//     on the cause with errors.Is.
//   - Untrusted input that fails validation wraps [ErrInvalidInput]; each
//     package adds its own typed error for finer branching.
//   - Expected rule outcomes are never errors; packages return them as values.
//
// Error strings are for humans. Never branch on them; use errors.Is, errors.As
// and [ErrorType].
//
// [Guard] and [GuardValue] are the only code in the module that calls recover.
// All helpers are stateless and safe for concurrent use.
package srderr
