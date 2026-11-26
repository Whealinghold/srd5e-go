package combat

import (
	"context"
	"slices"
)

// ApplyDamageToObjectCommand is the input of ApplyDamageToObject.Run: the object and
// what is done to it. Build it with a keyed literal; the zero value is invalid.
type ApplyDamageToObjectCommand struct {
	Status ObjectStatus
	Harm   ObjectHarm
}

// ApplyDamageToObjectResult is the closed set of outcomes of ApplyDamageToObject.Run.
// Variants: ObjectDamageApplied, ObjectDamageRejected.
type ApplyDamageToObjectResult interface{ isApplyDamageToObjectResult() }

// ObjectDamageApplied: the harm was handled, possibly changing nothing. It holds the new
// status, the events in order (ObjectDamageDeflected alone when the Damage Threshold was
// not met; otherwise ObjectDamaged and, at 0 Hit Points, ObjectDestroyed; none when
// everything was Immune), the resolution of each Damage and the threshold outcome.
type ObjectDamageApplied struct {
	status      ObjectStatus
	events      []EnvironmentEvent
	resolutions []DamageResolution
	threshold   ThresholdOutcome
}

// Status returns the object after the call.
func (r ObjectDamageApplied) Status() ObjectStatus { return r.status }

// Events returns a copy of the events, in the order they happened.
func (r ObjectDamageApplied) Events() []EnvironmentEvent {
	return append([]EnvironmentEvent{}, r.events...)
}

// Resolutions returns a copy of the per-Damage resolutions; none when the damage was
// deflected or the object was broken.
func (r ObjectDamageApplied) Resolutions() []DamageResolution { return slices.Clone(r.resolutions) }

// Threshold returns what the Damage Threshold did.
func (r ObjectDamageApplied) Threshold() ThresholdOutcome { return r.threshold }

// ObjectDamageRejected: the rules refuse the harm, because the object is already
// destroyed. Construct it with a keyed literal.
type ObjectDamageRejected struct{ Rejection ObjectDamageRejection }

func (ObjectDamageApplied) isApplyDamageToObjectResult()  {}
func (ObjectDamageRejected) isApplyDamageToObjectResult() {}

// ApplyDamageToObject is the use case that damages or breaks an object: the Damage
// Threshold is tested on the incoming amounts first, then each Damage goes through the
// damage pipeline (SRD 5.2.1, "Breaking Objects", pp. 177-178; "Damage Threshold", p.
// 180). It rolls no die and does not call ApplyDamage. Create it with
// NewApplyDamageToObject; the zero value is silent and works. Safe for concurrent use.
type ApplyDamageToObject struct{ environmentUse }

// NewApplyDamageToObject returns the use case. Options: WithTelemetry, WithClock.
func NewApplyDamageToObject(opts ...Option) ApplyDamageToObject {
	return ApplyDamageToObject{newEnvironmentUse(opts)}
}

// Run damages the object and records the shared signals.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// status, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. A Harm that
// is nil, the zero ObjectDamageInstance or an ObjectBrokenAutomatically with no basis is
// an error matching srderr.ErrInvalidInput that the span records; no operations or
// events are counted. A destroyed object is an ObjectDamageRejected result.
func (u ApplyDamageToObject) Run(ctx context.Context, cmd ApplyDamageToObjectCommand) (ApplyDamageToObjectResult, error) {
	if cmd.Status.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationApplyDamageToObject, spanApplyDamageToObject},
		func(context.Context) (ApplyDamageToObjectResult, error) {
			d, rejection, err := damageObject(cmd.Status, cmd.Harm)
			switch {
			case err != nil:
				return nil, err
			case rejection != 0:
				return ObjectDamageRejected{rejection}, nil
			}
			return ObjectDamageApplied(d), nil
		},
		func(r ApplyDamageToObjectResult) description {
			if _, ok := r.(ObjectDamageRejected); ok {
				return description{Class: resultRefused, Name: valueResultAlreadyDestroyed}
			}
			v := r.(ObjectDamageApplied)
			name := valueResultDamaged
			switch {
			case len(v.events) == 0:
				name = valueResultNoDamage
			case v.events[0] == ObjectDamageDeflected{}:
				name = valueResultDeflected
			case v.events[len(v.events)-1] == ObjectDestroyed{}:
				name = valueResultDestroyed
			}
			return description{Class: resultApplied, Name: name, Events: widen(v.events)}
		})
}
