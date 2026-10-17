// Package slogtelemetry adapts a caller-supplied [*slog.Logger] to the
// [telemetry.Logger] port. It uses the standard library only.
//
// The adapter never calls [slog.Default] and never creates a handler: the host
// decides where events go. Levels map as trace to [slog.LevelDebug]-4, debug to
// [slog.LevelDebug], info to [slog.LevelInfo], warn to [slog.LevelWarn] and error to
// [slog.LevelError].
package slogtelemetry
