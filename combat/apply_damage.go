package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ApplyDamageCommand is the input of ApplyDamage.Run: the creature, its status
// and the damage (see HitPointStatus.ApplyDamage). Build it with
// NewApplyDamageCommand; the zero value is invalid. Run does not alter it.
type ApplyDamageCommand struct {
	subject       CombatantID
	status        HitPointStatus
	damages       []Damage
	adjustments   []DamageAdjustment
	delivery      DamageDelivery
	kind          HitKind
	unpreventable []Damage
}

// NewApplyDamageCommand returns the command. The slices are copied.
//
// It returns an error matching srderr.ErrInvalidInput for a zero subject or
// status, no damage at all (damages and unpreventable are both empty), an
// invalid Damage, nil adjustment, delivery or kind, and a LimitExceededError for more than
// 64 damage instances or 64 adjustments. It never panics.
func NewApplyDamageCommand(subject CombatantID, status HitPointStatus, damages []Damage, adjustments []DamageAdjustment,
	delivery DamageDelivery, kind HitKind, unpreventable []Damage) (ApplyDamageCommand, error) {
	cmd := ApplyDamageCommand{subject, status, slices.Clone(damages), slices.Clone(adjustments), delivery, kind, slices.Clone(unpreventable)}
	if _, err := cmd.apply(); err != nil {
		return ApplyDamageCommand{}, err
	}
	return cmd, nil
}

func (c ApplyDamageCommand) apply() (HitPointTransition, error) {
	return c.status.ApplyDamage(c.subject, c.damages, c.adjustments, c.delivery, c.kind, c.unpreventable)
}

// ApplyDamageResult is the closed set of outcomes of ApplyDamage.Run. Variants:
// DamageApplied, DamageRejected.
type ApplyDamageResult interface{ isApplyDamageResult() }

// DamageApplied is damage that took effect, possibly changing nothing: the new
// status, the events of table 4.9a and the resolution of each instance.
type DamageApplied struct {
	appliedTransition
	resolutions []DamageResolution
}

// Resolutions returns a copy of the per-instance resolutions.
func (r DamageApplied) Resolutions() []DamageResolution { return slices.Clone(r.resolutions) }

// DamageRejected is damage the rules refuse: the creature is Dead. Construct it
// with a keyed literal.
type DamageRejected struct{ Rejection HitPointRejection }

func (DamageApplied) isApplyDamageResult()  {}
func (DamageRejected) isApplyDamageResult() {}

// ApplyDamage is the use case that applies damage to a creature. Create it with
// NewApplyDamage; the zero value is silent and works. It is safe for concurrent
// use only if its telemetry and clock are.
type ApplyDamage struct{ hitPointUse }

// NewApplyDamage returns the use case. Options: WithTelemetry, WithClock.
func NewApplyDamage(opts ...Option) ApplyDamage { return ApplyDamage{newHitPointUse(opts)} }

// Run applies cmd through HitPointStatus.ApplyDamage and records the shared
// signals plus srd5e.combat.damage_resolved (once per instance),
// srd5e.combat.hit_points_lost (once per command), the refusal and the deaths.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a DamageRejected result, never an error. The status in cmd is
// not altered.
func (u ApplyDamage) Run(ctx context.Context, cmd ApplyDamageCommand) (ApplyDamageResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationApplyDamage, spanApplyDamage}, cmd.apply,
		func(a HitPointTransitionApplied) string {
			switch {
			case has[Died](a.events):
				return valueResultKilled
			case has[KnockedOut](a.events):
				return valueResultKnockedOut
			case has[DroppedToZeroHitPoints](a.events):
				return valueResultDroppedToZero
			case has[LostStability](a.events):
				return valueResultLostStability
			case len(a.events) == 0:
				return valueResultNoEffect
			case a.events[0].(DamageTaken).lost == 0:
				return valueResultAbsorbed
			}
			return valueResultDamaged
		})
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) ApplyDamageResult {
		u.recordDamage(ctx, a)
		return DamageApplied{appliedTransition{a.status, a.events}, a.resolutions}
	}, func(r HitPointRejection) ApplyDamageResult { return DamageRejected{Rejection: r} }), nil
}

// recordDamage records the signals only ApplyDamage emits, for an applied result.
func (u ApplyDamage) recordDamage(ctx context.Context, a HitPointTransitionApplied) {
	meter := u.tel.WithDefaults().Meter
	resolved := meter.Counter(telemetry.Descriptor{Name: metricDamageResolved, Unit: "{instance}",
		Description: "Instances of damage resolved, by Damage Type and effect."})
	for _, r := range a.resolutions {
		resolved.Add(ctx, 1, telemetry.String(attrDamageType, damageType(r)), telemetry.String(attrDamageEffect, damageEffect(r)))
	}
	lost := 0
	if len(a.events) > 0 {
		lost = a.events[0].(DamageTaken).lost
	}
	meter.Histogram(telemetry.Descriptor{Name: metricHitPointsLost, Unit: "{hit_point}",
		Description: "Hit Points lost per applied ApplyDamage command."}).Record(ctx, float64(lost))
}
