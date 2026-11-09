package character

// Spans and the attribute values of the Barbarian use cases. They record on the shared class
// feature instruments MetricFeatureUsed, MetricFeatureUsedDuration and MetricFeatureUsedErrors
// (see feature_signals.go), which every class reuses, and on the log event
// LogClassFeatureResolved. The rejection values are the String of BarbarianReason.
const (
	// SpanUseRelentlessRage wraps one UseRelentlessRage.Run call.
	SpanUseRelentlessRage = "srd5e.character.use_relentless_rage"
	// SpanApplyBarbarianOption wraps one ApplyBarbarianOptionChoice.Run call, lookups included.
	SpanApplyBarbarianOption = "srd5e.character.apply_barbarian_option"
)

// Attribute values of the Barbarian use cases.
const (
	ValueClassBarbarian = "barbarian"

	ValueFeatureRelentlessRage = "relentless_rage"
	// ValueFeatureBarbarianWeaponMastery is the srd5e.feature of a Barbarian Weapon Mastery choice.
	ValueFeatureBarbarianWeaponMastery = ValueFeatureWeaponMastery
)
