package d20

// The signals Resolver emits, defined once. The names and attribute keys are a
// public contract: renaming one breaks consumers' dashboards and alerts. They
// are catalogued in doc/observability/signals.md. The shared keys
// telemetry.AttrOutcome and telemetry.AttrErrorType are used as they are.
const (
	// SpanResolve wraps one Resolve call.
	SpanResolve = "srd5e.d20_test.resolve"
	// MetricResolved counts resolved calls ({test}), by outcome.
	MetricResolved = "srd5e.d20_test.resolved"
	// MetricDuration records call durations (s), failed calls included.
	MetricDuration = "srd5e.d20_test.duration"
	// MetricModifiers records the number of modifiers in a rolled call
	// ({modifier}).
	MetricModifiers = "srd5e.d20_test.modifiers"
	// MetricNatural records the d20 face a rolled call used ({face}).
	MetricNatural = "srd5e.d20_test.natural"
	// MetricErrors counts port failures leaving Resolve ({error}).
	MetricErrors = "srd5e.d20_test.errors"
)

// Attribute keys.
const (
	// AttrKind is the kind of D20 Test: a ValueKind* value.
	AttrKind = "srd5e.d20_test.kind"
	// AttrAbility is the command's ability label, lower case.
	AttrAbility = "srd5e.ability"
	// AttrRollMode is the roll mode; absent for a forced failure.
	AttrRollMode = "srd5e.d20_test.roll_mode"
	// AttrCriticalHit is a bool; attack rolls only.
	AttrCriticalHit = "srd5e.d20_test.critical_hit"
	// AttrRollerFailure is why the roller failed: a ValueRoller* value; the
	// errors counter and the error span only.
	AttrRollerFailure = "srd5e.d20_test.roller_failure"
	// AttrForcedFailureReason is why a call was forced to fail: a
	// ValueForced* value; forced failures only.
	AttrForcedFailureReason = "srd5e.d20_test.forced_failure_reason"
)

// Attribute values.
const (
	ValueKindAbilityCheck = "ability_check"
	ValueKindSavingThrow  = "saving_throw"
	ValueKindAttackRoll   = "attack_roll"

	ValueRollerUnavailable   = "unavailable"
	ValueRollerInvalidResult = "invalid_result"

	ValueForcedChosenToFail = "chosen_to_fail"
	ValueForcedAutomatic    = "automatic"
)
