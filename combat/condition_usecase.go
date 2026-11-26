package combat

import (
	"context"
	"fmt"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// conditionUse is what every condition use case holds: the instrumenter for the
// shared signals and the telemetry for srd5e.combat.condition.active. The zero
// value is silent and usable.
type conditionUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newConditionUse(opts []Option) conditionUse {
	o := newOptions(opts)
	return conditionUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// begin returns the errors every Run reports before any signal: a zero target
// (a zero command) and an already cancelled ctx.
func (u conditionUse) begin(ctx context.Context, target CombatantID) error {
	if target.IsZero() {
		return invalidInput("zero command")
	}
	return ctx.Err()
}

// recordActive records the effective-condition count of set after a call that
// returned a result, with the snake_case kind the call was about.
func (u conditionUse) recordActive(ctx context.Context, op combatOperation, kind content.ConditionKind, set ConditionSet) {
	u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricConditionActive, Unit: "{condition}",
		Description: "Effective conditions on a creature after a condition call."}).
		Record(ctx, float64(len(set.Effective())),
			telemetry.String(attrOperation, op.Value()), telemetry.String(attrConditionKind, strings.ToLower(kind.String())))
}

// asConditionEvents widens condition events for the instrumenter.
func asConditionEvents(events []ConditionEvent) []Event {
	out := make([]Event, len(events))
	for i, e := range events {
		out[i] = e
	}
	return out
}

// describeApplyCondition classifies a result of ApplyConditionToSet.
func describeApplyCondition(r ApplyConditionToSetResult) description {
	switch v := r.(type) {
	case ConditionAdded:
		return description{Class: resultApplied, Name: valueResultAdded, Events: asConditionEvents(v.events)}
	case ConditionAlreadyHeld:
		return description{Class: resultRefused, Name: valueResultAlreadyHeld}
	case ConditionImmune:
		return description{Class: resultRefused, Name: valueResultImmune}
	case ExhaustionAtMaximum:
		return description{Class: resultRefused, Name: valueResultAtMaximum}
	default:
		panic(fmt.Errorf("combat: unreachable apply condition result %T", r))
	}
}

// describeRemoveCondition classifies a result of RemoveConditionFromSet.
func describeRemoveCondition(r RemoveConditionFromSetResult) description {
	switch v := r.(type) {
	case ConditionCleared:
		return description{Class: resultApplied, Name: valueResultCleared, Events: asConditionEvents(v.events)}
	case ExhaustionLowered:
		return description{Class: resultApplied, Name: valueResultLowered, Events: asConditionEvents(v.events)}
	case ConditionRemovalNotPresent:
		return description{Class: resultRefused, Name: valueResultRemovalNotPresent}
	case ConditionRemovalImpliedBy:
		return description{Class: resultRefused, Name: valueResultImpliedBy}
	default:
		panic(fmt.Errorf("combat: unreachable remove condition result %T", r))
	}
}

// describeReduceExhaustion classifies a result of ReduceExhaustionOnSet.
func describeReduceExhaustion(r ReduceExhaustionOnSetResult) description {
	switch v := r.(type) {
	case ExhaustionReduced:
		return description{Class: resultApplied, Name: valueResultReduced, Events: asConditionEvents(v.events)}
	case ExhaustionEnded:
		return description{Class: resultApplied, Name: valueResultEnded, Events: asConditionEvents(v.events)}
	case ExhaustionReductionNotPresent:
		return description{Class: resultRefused, Name: valueResultReductionNotPresent}
	default:
		panic(fmt.Errorf("combat: unreachable reduce exhaustion result %T", r))
	}
}
