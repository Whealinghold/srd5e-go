package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Spans and metrics of the higher-level start use cases StartAtHigherLevel,
// RollHigherLevelStartingMoney and RollTrinket. All three record on the shared
// MetricOperations counter and MetricDuration histogram; StartAtHigherLevel looks
// content up in the child span SpanAdvancementLoadContent and records
// MetricStartSteps.
const (
	// SpanAdvancementStartAtHigherLevel wraps one StartAtHigherLevel.Start call.
	SpanAdvancementStartAtHigherLevel = "srd5e.character.advancement.start_at_higher_level"
	// SpanAdvancementStartingMoney wraps one RollHigherLevelStartingMoney.Roll call.
	SpanAdvancementStartingMoney = "srd5e.character.advancement.starting_money"
	// SpanAdvancementTrinket wraps one RollTrinket.Roll call.
	SpanAdvancementTrinket = "srd5e.character.advancement.trinket"
	// MetricStartSteps records the number of steps of each higher-level start plan ({step}).
	MetricStartSteps = "srd5e.character.start_steps"
)

// Attribute values of the higher-level start use cases. The rejection values name
// the variant of HigherLevelStartRejection. ValueResultStarted, ValueResultRolled,
// ValueResultNotApplicable, ValueResultRejected and ValueResultContentMissing are
// shared.
const (
	ValueOperationStartAtHigherLevel = "start_at_higher_level"
	ValueOperationStartingMoney      = "starting_money"
	ValueOperationTrinket            = "trinket"

	ValueRejectionPlanLengthMismatch = "plan_length_mismatch"
	ValueRejectionCreationInvalid    = "creation_invalid"
	ValueRejectionLevelUpRejected    = "level_up_rejected"
	ValueRejectionSubclassRejected   = "subclass_rejected"
	ValueRejectionIncreaseRejected   = "increase_rejected"
	ValueRejectionIncreaseNotGranted = "increase_not_granted"
	ValueRejectionSubclassRequired   = "subclass_required"
)

// startSignals are the instruments StartAtHigherLevel shares with the other
// character use cases, plus its steps histogram.
type startSignals struct {
	operationSignals
	steps telemetry.Histogram
}

func newStartSignals(o options) startSignals {
	return startSignals{
		operationSignals: newOperationSignals(o),
		steps:            o.tel.Meter.Histogram(telemetry.Descriptor{Name: MetricStartSteps, Unit: "{step}", Description: "Number of steps of each higher-level start plan."}),
	}
}

// recordSteps records the size of the plan of one call.
func (s startSignals) recordSteps(ctx context.Context, n int) { s.steps.Record(ctx, float64(n)) }

// higherLevelStartRejectionValue maps a rejection to its ValueRejection* value.
func higherLevelStartRejectionValue(r HigherLevelStartRejection) string {
	switch r.(type) {
	case HigherLevelStartPlanLengthMismatch:
		return ValueRejectionPlanLengthMismatch
	case HigherLevelStartCreationInvalid:
		return ValueRejectionCreationInvalid
	case HigherLevelStartLevelUpRejected:
		return ValueRejectionLevelUpRejected
	case HigherLevelStartSubclassRejected:
		return ValueRejectionSubclassRejected
	case HigherLevelStartIncreaseRejected:
		return ValueRejectionIncreaseRejected
	case HigherLevelStartIncreaseNotGranted:
		return ValueRejectionIncreaseNotGranted
	}
	return ValueRejectionSubclassRequired
}
