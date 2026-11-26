package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// The use cases of the revival and restoration spells (revival-and-restoration-spells.md RR-2; combat-action-resolution.md
// FR-169): each wraps one pure service over host-held values and returns its closed result. They are named apart
// from the owner's services (section 3). An applied variant is completed, but revived, not_revived, increased, ended,
// warded and not_warded where FR-169 names them; Ineligible is ineligible and every other refusal is refused, never
// invalid and never an error.

// ReviveCreatureBySpellCommand is the input of ReviveCreatureBySpell.Run: the command of ReviveBySpell.
type ReviveCreatureBySpellCommand ReviveBySpellCommand

// ReviveCreatureBySpell is the use case of casting a revival spell: it wraps ReviveBySpell. Create it with
// NewReviveCreatureBySpell.
type ReviveCreatureBySpell struct{ in instrumenter }

// NewReviveCreatureBySpell returns the use case. Options: WithTelemetry, WithClock.
func NewReviveCreatureBySpell(opts ...Option) ReviveCreatureBySpell {
	return ReviveCreatureBySpell{newWired(opts)}
}

// Run casts the spell. The result is revived (ReviveBySpellApplied) or not_revived (ReviveBySpellNotRevived), applied,
// ineligible (ReviveBySpellIneligible) or refused (ReviveBySpellInvalidMaximum, ReviveBySpellLimitExceeded). It returns the errors described for
// the package.
func (u ReviveCreatureBySpell) Run(ctx context.Context, cmd ReviveCreatureBySpellCommand) (ReviveCreatureBySpellResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationReviveBySpell, spanReviveBySpell},
		func() (ReviveCreatureBySpellResult, error) { return ReviveBySpell(ReviveBySpellCommand(cmd)), nil },
		func(r ReviveCreatureBySpellResult) description {
			switch v := r.(type) {
			case ReviveBySpellApplied:
				return appliedWith(valueResultRevived, v.Revival.Events()...)
			case ReviveBySpellNotRevived:
				return appliedWith(valueResultNotRevived)
			case ReviveBySpellIneligible:
				return refusedWith(valueResultIneligible)
			case ReviveBySpellInvalidMaximum, ReviveBySpellLimitExceeded:
				return refusedWith(valueResultRefused)
			default:
				panic(unreachableError("revive by spell result", r))
			}
		})
}

// IncreaseHitPointMaximumCommand is the input of IncreaseHitPointMaximum.Run: the arguments of IncreaseMaximum.
type IncreaseHitPointMaximumCommand struct {
	Subject  CombatantID
	Status   HitPointStatus
	Ledger   HitPointMaximumIncreases
	Increase HitPointMaximumIncrease
}

// IncreaseHitPointMaximum is the use case of a spell that raises the Hit Point maximum: it wraps IncreaseMaximum. Create
// it with NewIncreaseHitPointMaximum.
type IncreaseHitPointMaximum struct{ in instrumenter }

// NewIncreaseHitPointMaximum returns the use case. Options: WithTelemetry, WithClock.
func NewIncreaseHitPointMaximum(opts ...Option) IncreaseHitPointMaximum {
	return IncreaseHitPointMaximum{newWired(opts)}
}

// Run raises the maximum. The result is increased (IncreaseMaximumApplied, applied) or refused (a Dead creature, a
// repeated effect or a limit exceeded). It returns the errors of IncreaseMaximum and those described for the package.
func (u IncreaseHitPointMaximum) Run(ctx context.Context, cmd IncreaseHitPointMaximumCommand) (IncreaseHitPointMaximumResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationIncreaseMaximum, spanIncreaseMaximum},
		func() (IncreaseHitPointMaximumResult, error) {
			return IncreaseMaximum(cmd.Subject, cmd.Status, cmd.Ledger, cmd.Increase)
		},
		func(r IncreaseHitPointMaximumResult) description {
			if v, ok := r.(IncreaseMaximumApplied); ok {
				return appliedWith(valueResultIncreased, v.Events()...)
			}
			return refusedWith(valueResultRefused)
		})
}

// EndHitPointMaximumIncreaseCommand is the input of EndHitPointMaximumIncrease.Run: the arguments of EndMaximumIncrease.
type EndHitPointMaximumIncreaseCommand struct {
	Subject  CombatantID
	Status   HitPointStatus
	Ledger   HitPointMaximumIncreases
	EffectID OngoingEffectID
}

// EndHitPointMaximumIncrease is the use case of an increase of the Hit Point maximum that ends: it wraps
// EndMaximumIncrease. Create it with NewEndHitPointMaximumIncrease.
type EndHitPointMaximumIncrease struct{ in instrumenter }

// NewEndHitPointMaximumIncrease returns the use case. Options: WithTelemetry, WithClock.
func NewEndHitPointMaximumIncrease(opts ...Option) EndHitPointMaximumIncrease {
	return EndHitPointMaximumIncrease{newWired(opts)}
}

// Run ends the increase. The result is ended (EndMaximumIncreaseApplied, applied) or refused (not present, or a Dead
// creature). It returns the errors of EndMaximumIncrease and those described for the package.
func (u EndHitPointMaximumIncrease) Run(ctx context.Context, cmd EndHitPointMaximumIncreaseCommand) (EndHitPointMaximumIncreaseResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationEndMaximumIncrease, spanEndMaximumIncrease},
		func() (EndHitPointMaximumIncreaseResult, error) {
			return EndMaximumIncrease(cmd.Subject, cmd.Status, cmd.Ledger, cmd.EffectID)
		},
		func(r EndHitPointMaximumIncreaseResult) description {
			if v, ok := r.(EndMaximumIncreaseApplied); ok {
				return appliedWith(valueResultEnded, v.Events()...)
			}
			return refusedWith(valueResultRefused)
		})
}

// RestoreHitPointMaximumCommand is the input of RestoreHitPointMaximum.Run: the arguments of RestoreMaximum.
type RestoreHitPointMaximumCommand struct {
	Subject CombatantID
	Status  HitPointStatus
	Amount  int
}

// RestoreHitPointMaximum is the use case of restoring a reduced Hit Point maximum: it wraps RestoreMaximum. Create it
// with NewRestoreHitPointMaximum.
type RestoreHitPointMaximum struct{ in instrumenter }

// NewRestoreHitPointMaximum returns the use case. Options: WithTelemetry, WithClock.
func NewRestoreHitPointMaximum(opts ...Option) RestoreHitPointMaximum {
	return RestoreHitPointMaximum{newWired(opts)}
}

// Run restores the maximum. The result is completed (RestoreMaximumApplied, applied) or refused. It returns the errors
// of RestoreMaximum and those described for the package.
func (u RestoreHitPointMaximum) Run(ctx context.Context, cmd RestoreHitPointMaximumCommand) (RestoreHitPointMaximumResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationRestoreMaximum, spanRestoreMaximum},
		func() (RestoreHitPointMaximumResult, error) {
			return RestoreMaximum(cmd.Subject, cmd.Status, cmd.Amount)
		},
		func(r RestoreHitPointMaximumResult) description {
			if v, ok := r.(RestoreMaximumApplied); ok {
				return appliedWith(valueResultCompleted, v.Events()...)
			}
			return refusedWith(valueResultRefused)
		})
}

// CastLesserRestorationCommand is the input of CastLesserRestoration.Run: the arguments of ApplyLesserRestoration.
type CastLesserRestorationCommand struct {
	Subject CombatantID
	Set     ConditionSet
	Kind    content.ConditionKind
}

// CastLesserRestoration is the use case of Lesser Restoration: it wraps ApplyLesserRestoration. Create it with
// NewCastLesserRestoration.
type CastLesserRestoration struct{ in instrumenter }

// NewCastLesserRestoration returns the use case. Options: WithTelemetry, WithClock.
func NewCastLesserRestoration(opts ...Option) CastLesserRestoration {
	return CastLesserRestoration{newWired(opts)}
}

// Run ends the condition. The result is completed (LesserRestorationApplied, applied) or refused (nothing to end, or a
// kind the spell cannot end). It returns the errors described for the package.
func (u CastLesserRestoration) Run(ctx context.Context, cmd CastLesserRestorationCommand) (CastLesserRestorationResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationLesserRestoration, spanLesserRestoration},
		func() (CastLesserRestorationResult, error) {
			return ApplyLesserRestoration(cmd.Subject, cmd.Set, cmd.Kind), nil
		},
		func(r CastLesserRestorationResult) description {
			if v, ok := r.(LesserRestorationApplied); ok {
				return appliedWith(valueResultCompleted, v.Events()...)
			}
			return refusedWith(valueResultRefused)
		})
}

// CastGreaterRestorationCommand is the input of CastGreaterRestoration.Run: the command of ApplyGreaterRestoration.
type CastGreaterRestorationCommand GreaterRestorationCommand

// CastGreaterRestoration is the use case of Greater Restoration: it wraps ApplyGreaterRestoration. Create it with
// NewCastGreaterRestoration.
type CastGreaterRestoration struct{ in instrumenter }

// NewCastGreaterRestoration returns the use case. Options: WithTelemetry, WithClock.
func NewCastGreaterRestoration(opts ...Option) CastGreaterRestoration {
	return CastGreaterRestoration{newWired(opts)}
}

// Run applies the chosen effect. The result is completed (GreaterRestorationApplied, or GreaterRestorationHostResolved:
// the host resolves a choice, applied) or refused (nothing to end, not a curse, rejected or invalid: never invalid,
// FR-169). It returns the errors described for the package.
func (u CastGreaterRestoration) Run(ctx context.Context, cmd CastGreaterRestorationCommand) (CastGreaterRestorationResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationGreaterRestoration, spanGreaterRestoration},
		func() (CastGreaterRestorationResult, error) {
			return ApplyGreaterRestoration(GreaterRestorationCommand(cmd)), nil
		},
		func(r CastGreaterRestorationResult) description {
			switch v := r.(type) {
			case GreaterRestorationApplied:
				return appliedWith(valueResultCompleted, v.Events()...)
			case GreaterRestorationHostResolved:
				return appliedWith(valueResultCompleted)
			default:
				return refusedWith(valueResultRefused)
			}
		})
}

// CastRestorativeHealingCommand is the input of CastRestorativeHealing.Run: the arguments of ApplyRestorativeHealing.
type CastRestorativeHealingCommand struct {
	Spell     RestorativeSpell
	Subject   CombatantID
	Status    HitPointStatus
	Set       ConditionSet
	Amount    int
	CastLevel int
}

// CastRestorativeHealing is the use case of a healing spell that also ends a condition: it wraps
// ApplyRestorativeHealing. Create it with NewCastRestorativeHealing.
type CastRestorativeHealing struct{ in instrumenter }

// NewCastRestorativeHealing returns the use case. Options: WithTelemetry, WithClock.
func NewCastRestorativeHealing(opts ...Option) CastRestorativeHealing {
	return CastRestorativeHealing{newWired(opts)}
}

// Run applies the healing. The result is completed (RestorativeHealingApplied, applied) or refused. It returns the
// errors of ApplyRestorativeHealing and those described for the package.
func (u CastRestorativeHealing) Run(ctx context.Context, cmd CastRestorativeHealingCommand) (CastRestorativeHealingResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationRestorativeHealing, spanRestorativeHealing},
		func() (CastRestorativeHealingResult, error) {
			return ApplyRestorativeHealing(cmd.Spell, cmd.Subject, cmd.Status, cmd.Set, cmd.Amount, cmd.CastLevel)
		},
		func(r CastRestorativeHealingResult) description {
			if v, ok := r.(RestorativeHealingApplied); ok {
				return appliedWith(valueResultCompleted, v.Events()...)
			}
			return refusedWith(valueResultRefused)
		})
}

// DeathWardRequest is what GuardDamageWithDeathWard asks of the owner. Variants: WardDamage, WardInstantKill.
type DeathWardRequest interface{ isDeathWardRequest() }

// WardDamage: damage is dealt to a warded creature (ApplyDamageWithDeathWard).
type WardDamage struct {
	Damages       []Damage
	Adjustments   []DamageAdjustment
	Delivery      DamageDelivery
	Kind          HitKind
	Unpreventable []Damage
}

// WardInstantKill: an effect that would kill the creature instantly without dealing damage (NegateInstantKill).
type WardInstantKill struct{}

func (WardDamage) isDeathWardRequest()      {}
func (WardInstantKill) isDeathWardRequest() {}

// GuardDamageWithDeathWardCommand is the input of GuardDamageWithDeathWard.Run. Construct it with a keyed literal.
type GuardDamageWithDeathWardCommand struct {
	Subject CombatantID
	Status  HitPointStatus
	Ward    DeathWard
	Request DeathWardRequest
}

// GuardDamageWithDeathWard is the use case of Death Ward (SRD 5.2.1, "Death Ward", p. 122): it wraps
// ApplyDamageWithDeathWard and NegateInstantKill. The result is warded (GuardedDamageApplied that spent the ward) or
// not_warded (GuardedDamageNotWarded), both applied, completed (GuardedDamageApplied with the ward kept, applied) or
// refused (GuardedDamageRefused). Create it with NewGuardDamageWithDeathWard.
type GuardDamageWithDeathWard struct{ in instrumenter }

// NewGuardDamageWithDeathWard returns the use case. Options: WithTelemetry, WithClock.
func NewGuardDamageWithDeathWard(opts ...Option) GuardDamageWithDeathWard {
	return GuardDamageWithDeathWard{newWired(opts)}
}

// Run guards the creature. It returns an error matching srderr.ErrInvalidInput for a nil or unknown request and where
// the owner does, and the errors described for the package.
func (u GuardDamageWithDeathWard) Run(ctx context.Context, cmd GuardDamageWithDeathWardCommand) (GuardDamageWithDeathWardResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationGuardWithDeathWard, spanGuardWithDeathWard},
		func() (GuardDamageWithDeathWardResult, error) {
			switch v := cmd.Request.(type) {
			case WardDamage:
				return ApplyDamageWithDeathWard(cmd.Subject, cmd.Status, v.Damages, v.Adjustments, v.Delivery, v.Kind, v.Unpreventable, cmd.Ward)
			case WardInstantKill:
				return NegateInstantKill(cmd.Subject, cmd.Status, cmd.Ward)
			}
			return nil, invalidInput("a Death Ward guard needs a request")
		},
		func(r GuardDamageWithDeathWardResult) description {
			switch v := r.(type) {
			case GuardedDamageApplied:
				events := v.Events()
				for _, e := range events {
					if _, spent := e.(DeathWardSpent); spent {
						return appliedWith(valueResultWarded, events...)
					}
				}
				return appliedWith(valueResultCompleted, events...)
			case GuardedDamageNotWarded:
				return appliedWith(valueResultNotWarded)
			case GuardedDamageRefused:
				return refusedWith(valueResultRefused)
			default:
				panic(unreachableError("death ward result", r))
			}
		})
}
