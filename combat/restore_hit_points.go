package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// RestoreHitPointsCommand is the input of RestoreHitPoints.Run: the creature,
// its status, the Healing amount and an optional ceiling. Build it with
// NewRestoreHitPointsCommand; the zero value is invalid.
type RestoreHitPointsCommand struct {
	subject CombatantID
	status  HitPointStatus
	amount  int
	ceiling HealingCeiling
}

// NewRestoreHitPointsCommand returns the command; a nil ceiling is
// NoHealingCeiling. It returns an error matching srderr.ErrInvalidInput for a
// zero subject or status, a negative amount or an unknown ceiling.
func NewRestoreHitPointsCommand(subject CombatantID, status HitPointStatus, amount int, ceiling HealingCeiling) (RestoreHitPointsCommand, error) {
	if ceiling == nil {
		ceiling = NoHealingCeiling{}
	}
	cmd := RestoreHitPointsCommand{subject, status, amount, ceiling}
	if _, err := cmd.apply(); err != nil {
		return RestoreHitPointsCommand{}, err
	}
	return cmd, nil
}

func (c RestoreHitPointsCommand) apply() (HitPointTransition, error) {
	return c.status.Restore(c.subject, c.amount, c.ceiling)
}

// RestoreHitPointsResult is the closed set of outcomes of RestoreHitPoints.Run.
// Variants: HealingApplied, HealingRejected.
type RestoreHitPointsResult interface{ isRestoreHitPointsResult() }

// HealingApplied is Healing that took effect, possibly regaining nothing.
type HealingApplied struct {
	appliedTransition
	regained int
}

// HitPointsRegained returns the Hit Points actually regained.
func (r HealingApplied) HitPointsRegained() int { return r.regained }

// HealingRejected is Healing the rules refuse: the creature is Dead. Construct it
// with a keyed literal.
type HealingRejected struct{ Rejection HitPointRejection }

func (HealingApplied) isRestoreHitPointsResult()  {}
func (HealingRejected) isRestoreHitPointsResult() {}

// RestoreHitPoints is the use case that heals a creature. Create it with
// NewRestoreHitPoints; the zero value is silent and works.
type RestoreHitPoints struct{ hitPointUse }

// NewRestoreHitPoints returns the use case. Options: WithTelemetry, WithClock.
func NewRestoreHitPoints(opts ...Option) RestoreHitPoints {
	return RestoreHitPoints{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.Restore and records the shared signals
// plus srd5e.combat.hit_points_restored (the amount regained, 0 included), the
// refusal and the deaths.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a HealingRejected result, never an error.
func (u RestoreHitPoints) Run(ctx context.Context, cmd RestoreHitPointsCommand) (RestoreHitPointsResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationRestoreHitPoints, spanRestoreHitPoints}, cmd.apply,
		func(a HitPointTransitionApplied) string {
			switch {
			case has[Recovered](a.events):
				return valueResultRecovered
			case a.regained > 0:
				return valueResultHealed
			}
			return valueResultNoEffect
		})
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) RestoreHitPointsResult {
		u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricHitPointsRestored, Unit: "{hit_point}",
			Description: "Hit Points regained per applied RestoreHitPoints command."}).Record(ctx, float64(a.regained))
		return HealingApplied{appliedTransition{a.status, a.events}, a.regained}
	}, func(r HitPointRejection) RestoreHitPointsResult { return HealingRejected{Rejection: r} }), nil
}
