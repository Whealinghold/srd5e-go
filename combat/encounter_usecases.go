package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Every use case below wraps exactly one Encounter command: it runs it through the
// instrumenter, adds no rule or event and maps the transition onto its own closed
// result, <UseCase>Applied or <UseCase>Rejected, the latter holding the EncounterRejection
// as its Rejection field. Each is built by NewX(opts ...Option) (WithTelemetry,
// WithClock); its zero value is usable and silent; Run returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero command, and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx. A refusal is a Rejected result,
// never an error. They roll no dice, call no port and are safe for concurrent use if
// their telemetry and clock are. A command takes the Encounter by value: Run does not
// alter it.

// targetCommand is the part every command naming a combatant shares.
type targetCommand struct {
	encounter Encounter
	combatant CombatantID
}

// newTargetCommand checks the encounter and the id.
func newTargetCommand(enc Encounter, id CombatantID) (targetCommand, error) {
	if enc.IsZero() || id.IsZero() {
		return targetCommand{}, invalidInput("a command needs a valid encounter and a combatant id")
	}
	return targetCommand{enc, id}, nil
}

// ApplyHealingToCombatantCommand is the immutable input of ApplyHealingToCombatant.Run. Build it with NewApplyHealingToCombatantCommand; the zero value is invalid.
type ApplyHealingToCombatantCommand struct {
	targetCommand
	amount int
}

// NewApplyHealingToCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or a negative amount. It never panics.
func NewApplyHealingToCombatantCommand(encounter Encounter, combatant CombatantID, amount int) (ApplyHealingToCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ApplyHealingToCombatantCommand{}, err
	}
	if amount < 0 {
		return ApplyHealingToCombatantCommand{}, invalidInput("negative healing")
	}
	return ApplyHealingToCombatantCommand{t, amount}, nil
}

// ApplyHealingToCombatantResult is the closed set of outcomes of ApplyHealingToCombatant.Run. Variants: ApplyHealingToCombatantApplied, ApplyHealingToCombatantRejected.
type ApplyHealingToCombatantResult interface{ isApplyHealingToCombatantResult() }

// ApplyHealingToCombatantApplied: the command took effect, possibly changing nothing.
type ApplyHealingToCombatantApplied struct{ encounterApplied }

// ApplyHealingToCombatantRejected: the encounter refused the command; Rejection says why.
type ApplyHealingToCombatantRejected struct{ encounterRejected }

func (ApplyHealingToCombatantApplied) isApplyHealingToCombatantResult()  {}
func (ApplyHealingToCombatantRejected) isApplyHealingToCombatantResult() {}

// ApplyHealingToCombatant is the use case of applying healing to a combatant (Encounter.ApplyHealing). Create it with NewApplyHealingToCombatant; the zero value is silent and works.
type ApplyHealingToCombatant struct{ encounterUse }

// NewApplyHealingToCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewApplyHealingToCombatant(opts ...Option) ApplyHealingToCombatant {
	return ApplyHealingToCombatant{newEncounterUse(opts)}
}

// Run applying healing to a combatant (Encounter.ApplyHealing) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u ApplyHealingToCombatant) Run(ctx context.Context, cmd ApplyHealingToCombatantCommand) (ApplyHealingToCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationApplyHealingToCombatant, spanApplyHealingToCombatant), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.ApplyHealing(cmd.combatant, cmd.amount) },
		func(a encounterApplied, _ []DamageResolution) ApplyHealingToCombatantResult {
			return ApplyHealingToCombatantApplied{a}
		},
		func(r encounterRejected) ApplyHealingToCombatantResult { return ApplyHealingToCombatantRejected{r} })
}

// ApplyConditionToCombatantCommand is the immutable input of ApplyConditionToCombatant.Run. Build it with NewApplyConditionToCombatantCommand; the zero value is invalid.
type ApplyConditionToCombatantCommand struct {
	targetCommand
	condition Condition
}

// NewApplyConditionToCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or a nil or invalid condition. It never panics.
func NewApplyConditionToCombatantCommand(encounter Encounter, combatant CombatantID, condition Condition) (ApplyConditionToCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ApplyConditionToCombatantCommand{}, err
	}
	if !validCondition(condition) {
		return ApplyConditionToCombatantCommand{}, invalidInput("a nil or invalid condition")
	}
	return ApplyConditionToCombatantCommand{t, condition}, nil
}

// ApplyConditionToCombatantResult is the closed set of outcomes of ApplyConditionToCombatant.Run. Variants: ApplyConditionToCombatantApplied, ApplyConditionToCombatantRejected.
type ApplyConditionToCombatantResult interface{ isApplyConditionToCombatantResult() }

// ApplyConditionToCombatantApplied: the command took effect, possibly changing nothing.
type ApplyConditionToCombatantApplied struct{ encounterApplied }

// ApplyConditionToCombatantRejected: the encounter refused the command; Rejection says why.
type ApplyConditionToCombatantRejected struct{ encounterRejected }

func (ApplyConditionToCombatantApplied) isApplyConditionToCombatantResult()  {}
func (ApplyConditionToCombatantRejected) isApplyConditionToCombatantResult() {}

// ApplyConditionToCombatant is the use case of applying a condition to a combatant (Encounter.ApplyCondition). Create it with NewApplyConditionToCombatant; the zero value is silent and works.
type ApplyConditionToCombatant struct{ encounterUse }

// NewApplyConditionToCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewApplyConditionToCombatant(opts ...Option) ApplyConditionToCombatant {
	return ApplyConditionToCombatant{newEncounterUse(opts)}
}

// Run applying a condition to a combatant (Encounter.ApplyCondition) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u ApplyConditionToCombatant) Run(ctx context.Context, cmd ApplyConditionToCombatantCommand) (ApplyConditionToCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationApplyConditionToCombatant, spanApplyConditionToCombatant), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.ApplyCondition(cmd.combatant, cmd.condition) },
		func(a encounterApplied, _ []DamageResolution) ApplyConditionToCombatantResult {
			return ApplyConditionToCombatantApplied{a}
		},
		func(r encounterRejected) ApplyConditionToCombatantResult { return ApplyConditionToCombatantRejected{r} })
}

// RemoveConditionFromCombatantCommand is the immutable input of RemoveConditionFromCombatant.Run. Build it with NewRemoveConditionFromCombatantCommand; the zero value is invalid.
type RemoveConditionFromCombatantCommand struct {
	targetCommand
	kind content.ConditionKind
}

// NewRemoveConditionFromCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or an invalid kind. It never panics.
func NewRemoveConditionFromCombatantCommand(encounter Encounter, combatant CombatantID, kind content.ConditionKind) (RemoveConditionFromCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return RemoveConditionFromCombatantCommand{}, err
	}
	if !kind.Valid() {
		return RemoveConditionFromCombatantCommand{}, invalidInput("an invalid condition kind")
	}
	return RemoveConditionFromCombatantCommand{t, kind}, nil
}

// RemoveConditionFromCombatantResult is the closed set of outcomes of RemoveConditionFromCombatant.Run. Variants: RemoveConditionFromCombatantApplied, RemoveConditionFromCombatantRejected.
type RemoveConditionFromCombatantResult interface{ isRemoveConditionFromCombatantResult() }

// RemoveConditionFromCombatantApplied: the command took effect, possibly changing nothing.
type RemoveConditionFromCombatantApplied struct{ encounterApplied }

// RemoveConditionFromCombatantRejected: the encounter refused the command; Rejection says why.
type RemoveConditionFromCombatantRejected struct{ encounterRejected }

func (RemoveConditionFromCombatantApplied) isRemoveConditionFromCombatantResult()  {}
func (RemoveConditionFromCombatantRejected) isRemoveConditionFromCombatantResult() {}

// RemoveConditionFromCombatant is the use case of removing a condition from a combatant (Encounter.RemoveCondition). Create it with NewRemoveConditionFromCombatant; the zero value is silent and works.
type RemoveConditionFromCombatant struct{ encounterUse }

// NewRemoveConditionFromCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewRemoveConditionFromCombatant(opts ...Option) RemoveConditionFromCombatant {
	return RemoveConditionFromCombatant{newEncounterUse(opts)}
}

// Run removing a condition from a combatant (Encounter.RemoveCondition) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u RemoveConditionFromCombatant) Run(ctx context.Context, cmd RemoveConditionFromCombatantCommand) (RemoveConditionFromCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationRemoveConditionFromCombatant, spanRemoveConditionFromCombatant), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.RemoveCondition(cmd.combatant, cmd.kind) },
		func(a encounterApplied, _ []DamageResolution) RemoveConditionFromCombatantResult {
			return RemoveConditionFromCombatantApplied{a}
		},
		func(r encounterRejected) RemoveConditionFromCombatantResult {
			return RemoveConditionFromCombatantRejected{r}
		})
}

// ReduceCombatantExhaustionCommand is the immutable input of ReduceCombatantExhaustion.Run. Build it with NewReduceCombatantExhaustionCommand; the zero value is invalid.
type ReduceCombatantExhaustionCommand struct {
	targetCommand
	levels int
}

// NewReduceCombatantExhaustionCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or levels below 1. It never panics.
func NewReduceCombatantExhaustionCommand(encounter Encounter, combatant CombatantID, levels int) (ReduceCombatantExhaustionCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ReduceCombatantExhaustionCommand{}, err
	}
	if levels < 1 {
		return ReduceCombatantExhaustionCommand{}, invalidInput("levels must be at least 1")
	}
	return ReduceCombatantExhaustionCommand{t, levels}, nil
}

// ReduceCombatantExhaustionResult is the closed set of outcomes of ReduceCombatantExhaustion.Run. Variants: ReduceCombatantExhaustionApplied, ReduceCombatantExhaustionRejected.
type ReduceCombatantExhaustionResult interface{ isReduceCombatantExhaustionResult() }

// ReduceCombatantExhaustionApplied: the command took effect, possibly changing nothing.
type ReduceCombatantExhaustionApplied struct{ encounterApplied }

// ReduceCombatantExhaustionRejected: the encounter refused the command; Rejection says why.
type ReduceCombatantExhaustionRejected struct{ encounterRejected }

func (ReduceCombatantExhaustionApplied) isReduceCombatantExhaustionResult()  {}
func (ReduceCombatantExhaustionRejected) isReduceCombatantExhaustionResult() {}

// ReduceCombatantExhaustion is the use case of lowering a combatant's Exhaustion (Encounter.ReduceExhaustion). Create it with NewReduceCombatantExhaustion; the zero value is silent and works.
type ReduceCombatantExhaustion struct{ encounterUse }

// NewReduceCombatantExhaustion returns the use case. Options: WithTelemetry, WithClock.
func NewReduceCombatantExhaustion(opts ...Option) ReduceCombatantExhaustion {
	return ReduceCombatantExhaustion{newEncounterUse(opts)}
}

// Run lowering a combatant's Exhaustion (Encounter.ReduceExhaustion) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u ReduceCombatantExhaustion) Run(ctx context.Context, cmd ReduceCombatantExhaustionCommand) (ReduceCombatantExhaustionResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationReduceCombatantExhaustion, spanReduceCombatantExhaustion), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.ReduceExhaustion(cmd.combatant, cmd.levels) },
		func(a encounterApplied, _ []DamageResolution) ReduceCombatantExhaustionResult {
			return ReduceCombatantExhaustionApplied{a}
		},
		func(r encounterRejected) ReduceCombatantExhaustionResult { return ReduceCombatantExhaustionRejected{r} })
}

// ApplyDeathSavingThrowToCombatantCommand is the immutable input of ApplyDeathSavingThrowToCombatant.Run. Build it with NewApplyDeathSavingThrowToCombatantCommand; the zero value is invalid.
type ApplyDeathSavingThrowToCombatantCommand struct {
	targetCommand
	natural, total int
}

// NewApplyDeathSavingThrowToCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or a natural outside 1 to 20. It never panics.
func NewApplyDeathSavingThrowToCombatantCommand(encounter Encounter, combatant CombatantID, natural, total int) (ApplyDeathSavingThrowToCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ApplyDeathSavingThrowToCombatantCommand{}, err
	}
	if natural < 1 || natural > 20 {
		return ApplyDeathSavingThrowToCombatantCommand{}, invalidInput("natural outside 1 to 20")
	}
	return ApplyDeathSavingThrowToCombatantCommand{t, natural, total}, nil
}

// ApplyDeathSavingThrowToCombatantResult is the closed set of outcomes of ApplyDeathSavingThrowToCombatant.Run. Variants: ApplyDeathSavingThrowToCombatantApplied, ApplyDeathSavingThrowToCombatantRejected.
type ApplyDeathSavingThrowToCombatantResult interface{ isApplyDeathSavingThrowToCombatantResult() }

// ApplyDeathSavingThrowToCombatantApplied: the command took effect, possibly changing nothing.
type ApplyDeathSavingThrowToCombatantApplied struct{ encounterApplied }

// ApplyDeathSavingThrowToCombatantRejected: the encounter refused the command; Rejection says why.
type ApplyDeathSavingThrowToCombatantRejected struct{ encounterRejected }

func (ApplyDeathSavingThrowToCombatantApplied) isApplyDeathSavingThrowToCombatantResult()  {}
func (ApplyDeathSavingThrowToCombatantRejected) isApplyDeathSavingThrowToCombatantResult() {}

// ApplyDeathSavingThrowToCombatant is the use case of applying a Death Saving Throw the host rolled (Encounter.ApplyDeathSavingThrow). Create it with NewApplyDeathSavingThrowToCombatant; the zero value is silent and works.
type ApplyDeathSavingThrowToCombatant struct{ encounterUse }

// NewApplyDeathSavingThrowToCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewApplyDeathSavingThrowToCombatant(opts ...Option) ApplyDeathSavingThrowToCombatant {
	return ApplyDeathSavingThrowToCombatant{newEncounterUse(opts)}
}

// Run applying a Death Saving Throw the host rolled (Encounter.ApplyDeathSavingThrow) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u ApplyDeathSavingThrowToCombatant) Run(ctx context.Context, cmd ApplyDeathSavingThrowToCombatantCommand) (ApplyDeathSavingThrowToCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationApplyDeathSavingThrowToCombatant, spanApplyDeathSavingThrowToCombatant), cmd.encounter,
		func() (encounterTransition, error) {
			return cmd.encounter.ApplyDeathSavingThrow(cmd.combatant, cmd.natural, cmd.total)
		},
		func(a encounterApplied, _ []DamageResolution) ApplyDeathSavingThrowToCombatantResult {
			return ApplyDeathSavingThrowToCombatantApplied{a}
		},
		func(r encounterRejected) ApplyDeathSavingThrowToCombatantResult {
			return ApplyDeathSavingThrowToCombatantRejected{r}
		})
}

// StabilizeCombatantCommand is the immutable input of StabilizeCombatant.Run. Build it with NewStabilizeCombatantCommand; the zero value is invalid.
type StabilizeCombatantCommand struct {
	targetCommand
}

// NewStabilizeCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id or a zero encounter. It never panics.
func NewStabilizeCombatantCommand(encounter Encounter, combatant CombatantID) (StabilizeCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return StabilizeCombatantCommand{}, err
	}
	return StabilizeCombatantCommand{t}, nil
}

// StabilizeCombatantResult is the closed set of outcomes of StabilizeCombatant.Run. Variants: StabilizeCombatantApplied, StabilizeCombatantRejected.
type StabilizeCombatantResult interface{ isStabilizeCombatantResult() }

// StabilizeCombatantApplied: the command took effect, possibly changing nothing.
type StabilizeCombatantApplied struct{ encounterApplied }

// StabilizeCombatantRejected: the encounter refused the command; Rejection says why.
type StabilizeCombatantRejected struct{ encounterRejected }

func (StabilizeCombatantApplied) isStabilizeCombatantResult()  {}
func (StabilizeCombatantRejected) isStabilizeCombatantResult() {}

// StabilizeCombatant is the use case of stabilizing a Dying combatant (Encounter.Stabilize). Create it with NewStabilizeCombatant; the zero value is silent and works.
type StabilizeCombatant struct{ encounterUse }

// NewStabilizeCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewStabilizeCombatant(opts ...Option) StabilizeCombatant {
	return StabilizeCombatant{newEncounterUse(opts)}
}

// Run stabilizing a Dying combatant (Encounter.Stabilize) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u StabilizeCombatant) Run(ctx context.Context, cmd StabilizeCombatantCommand) (StabilizeCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationStabilizeCombatant, spanStabilizeCombatant), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.Stabilize(cmd.combatant) },
		func(a encounterApplied, _ []DamageResolution) StabilizeCombatantResult {
			return StabilizeCombatantApplied{a}
		},
		func(r encounterRejected) StabilizeCombatantResult { return StabilizeCombatantRejected{r} })
}

// RecoverStableCombatantCommand is the immutable input of RecoverStableCombatant.Run. Build it with NewRecoverStableCombatantCommand; the zero value is invalid.
type RecoverStableCombatantCommand struct {
	targetCommand
}

// NewRecoverStableCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id or a zero encounter. It never panics.
func NewRecoverStableCombatantCommand(encounter Encounter, combatant CombatantID) (RecoverStableCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return RecoverStableCombatantCommand{}, err
	}
	return RecoverStableCombatantCommand{t}, nil
}

// RecoverStableCombatantResult is the closed set of outcomes of RecoverStableCombatant.Run. Variants: RecoverStableCombatantApplied, RecoverStableCombatantRejected.
type RecoverStableCombatantResult interface{ isRecoverStableCombatantResult() }

// RecoverStableCombatantApplied: the command took effect, possibly changing nothing.
type RecoverStableCombatantApplied struct{ encounterApplied }

// RecoverStableCombatantRejected: the encounter refused the command; Rejection says why.
type RecoverStableCombatantRejected struct{ encounterRejected }

func (RecoverStableCombatantApplied) isRecoverStableCombatantResult()  {}
func (RecoverStableCombatantRejected) isRecoverStableCombatantResult() {}

// RecoverStableCombatant is the use case of recovering a Stable combatant (Encounter.RecoverStable). Create it with NewRecoverStableCombatant; the zero value is silent and works.
type RecoverStableCombatant struct{ encounterUse }

// NewRecoverStableCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewRecoverStableCombatant(opts ...Option) RecoverStableCombatant {
	return RecoverStableCombatant{newEncounterUse(opts)}
}

// Run recovering a Stable combatant (Encounter.RecoverStable) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u RecoverStableCombatant) Run(ctx context.Context, cmd RecoverStableCombatantCommand) (RecoverStableCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationRecoverStableCombatant, spanRecoverStableCombatant), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.RecoverStable(cmd.combatant) },
		func(a encounterApplied, _ []DamageResolution) RecoverStableCombatantResult {
			return RecoverStableCombatantApplied{a}
		},
		func(r encounterRejected) RecoverStableCombatantResult { return RecoverStableCombatantRejected{r} })
}

// GiveCombatantTemporaryHitPointsCommand is the immutable input of GiveCombatantTemporaryHitPoints.Run. Build it with NewGiveCombatantTemporaryHitPointsCommand; the zero value is invalid.
type GiveCombatantTemporaryHitPointsCommand struct {
	targetCommand
	amount int
	choice TemporaryHitPointsChoice
}

// NewGiveCombatantTemporaryHitPointsCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter, an amount below 1 or an invalid choice. It never panics.
func NewGiveCombatantTemporaryHitPointsCommand(encounter Encounter, combatant CombatantID, amount int, choice TemporaryHitPointsChoice) (GiveCombatantTemporaryHitPointsCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return GiveCombatantTemporaryHitPointsCommand{}, err
	}
	if amount < 1 || !choice.Valid() {
		return GiveCombatantTemporaryHitPointsCommand{}, invalidInput("an amount of 1 or more and a choice")
	}
	return GiveCombatantTemporaryHitPointsCommand{t, amount, choice}, nil
}

// GiveCombatantTemporaryHitPointsResult is the closed set of outcomes of GiveCombatantTemporaryHitPoints.Run. Variants: GiveCombatantTemporaryHitPointsApplied, GiveCombatantTemporaryHitPointsRejected.
type GiveCombatantTemporaryHitPointsResult interface{ isGiveCombatantTemporaryHitPointsResult() }

// GiveCombatantTemporaryHitPointsApplied: the command took effect, possibly changing nothing.
type GiveCombatantTemporaryHitPointsApplied struct{ encounterApplied }

// GiveCombatantTemporaryHitPointsRejected: the encounter refused the command; Rejection says why.
type GiveCombatantTemporaryHitPointsRejected struct{ encounterRejected }

func (GiveCombatantTemporaryHitPointsApplied) isGiveCombatantTemporaryHitPointsResult()  {}
func (GiveCombatantTemporaryHitPointsRejected) isGiveCombatantTemporaryHitPointsResult() {}

// GiveCombatantTemporaryHitPoints is the use case of giving a combatant Temporary Hit Points (Encounter.GainTemporaryHitPoints). Create it with NewGiveCombatantTemporaryHitPoints; the zero value is silent and works.
type GiveCombatantTemporaryHitPoints struct{ encounterUse }

// NewGiveCombatantTemporaryHitPoints returns the use case. Options: WithTelemetry, WithClock.
func NewGiveCombatantTemporaryHitPoints(opts ...Option) GiveCombatantTemporaryHitPoints {
	return GiveCombatantTemporaryHitPoints{newEncounterUse(opts)}
}

// Run giving a combatant Temporary Hit Points (Encounter.GainTemporaryHitPoints) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u GiveCombatantTemporaryHitPoints) Run(ctx context.Context, cmd GiveCombatantTemporaryHitPointsCommand) (GiveCombatantTemporaryHitPointsResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationGiveCombatantTemporaryHitPoints, spanGiveCombatantTemporaryHitPoints), cmd.encounter,
		func() (encounterTransition, error) {
			return cmd.encounter.GainTemporaryHitPoints(cmd.combatant, cmd.amount, cmd.choice)
		},
		func(a encounterApplied, _ []DamageResolution) GiveCombatantTemporaryHitPointsResult {
			return GiveCombatantTemporaryHitPointsApplied{a}
		},
		func(r encounterRejected) GiveCombatantTemporaryHitPointsResult {
			return GiveCombatantTemporaryHitPointsRejected{r}
		})
}

// ReduceCombatantHitPointMaximumCommand is the immutable input of ReduceCombatantHitPointMaximum.Run. Build it with NewReduceCombatantHitPointMaximumCommand; the zero value is invalid.
type ReduceCombatantHitPointMaximumCommand struct {
	targetCommand
	amount int
}

// NewReduceCombatantHitPointMaximumCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or a negative amount. It never panics.
func NewReduceCombatantHitPointMaximumCommand(encounter Encounter, combatant CombatantID, amount int) (ReduceCombatantHitPointMaximumCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ReduceCombatantHitPointMaximumCommand{}, err
	}
	if amount < 0 {
		return ReduceCombatantHitPointMaximumCommand{}, invalidInput("negative maximum reduction")
	}
	return ReduceCombatantHitPointMaximumCommand{t, amount}, nil
}

// ReduceCombatantHitPointMaximumResult is the closed set of outcomes of ReduceCombatantHitPointMaximum.Run. Variants: ReduceCombatantHitPointMaximumApplied, ReduceCombatantHitPointMaximumRejected.
type ReduceCombatantHitPointMaximumResult interface{ isReduceCombatantHitPointMaximumResult() }

// ReduceCombatantHitPointMaximumApplied: the command took effect, possibly changing nothing.
type ReduceCombatantHitPointMaximumApplied struct{ encounterApplied }

// ReduceCombatantHitPointMaximumRejected: the encounter refused the command; Rejection says why.
type ReduceCombatantHitPointMaximumRejected struct{ encounterRejected }

func (ReduceCombatantHitPointMaximumApplied) isReduceCombatantHitPointMaximumResult()  {}
func (ReduceCombatantHitPointMaximumRejected) isReduceCombatantHitPointMaximumResult() {}

// ReduceCombatantHitPointMaximum is the use case of lowering a combatant's Hit Point maximum (Encounter.ReduceHitPointMaximum). Create it with NewReduceCombatantHitPointMaximum; the zero value is silent and works.
type ReduceCombatantHitPointMaximum struct{ encounterUse }

// NewReduceCombatantHitPointMaximum returns the use case. Options: WithTelemetry, WithClock.
func NewReduceCombatantHitPointMaximum(opts ...Option) ReduceCombatantHitPointMaximum {
	return ReduceCombatantHitPointMaximum{newEncounterUse(opts)}
}

// Run lowering a combatant's Hit Point maximum (Encounter.ReduceHitPointMaximum) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u ReduceCombatantHitPointMaximum) Run(ctx context.Context, cmd ReduceCombatantHitPointMaximumCommand) (ReduceCombatantHitPointMaximumResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationReduceCombatantHitPointMaximum, spanReduceCombatantHitPointMaximum), cmd.encounter,
		func() (encounterTransition, error) {
			return cmd.encounter.ReduceHitPointMaximum(cmd.combatant, cmd.amount)
		},
		func(a encounterApplied, _ []DamageResolution) ReduceCombatantHitPointMaximumResult {
			return ReduceCombatantHitPointMaximumApplied{a}
		},
		func(r encounterRejected) ReduceCombatantHitPointMaximumResult {
			return ReduceCombatantHitPointMaximumRejected{r}
		})
}

// CheckCombatantGrappleCommand is the immutable input of CheckCombatantGrapple.Run. Build it with NewCheckCombatantGrappleCommand; the zero value is invalid.
type CheckCombatantGrappleCommand struct {
	targetCommand
	distanceFeet, rangeFeet int
}

// NewCheckCombatantGrappleCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or a negative distance or range. It never panics.
func NewCheckCombatantGrappleCommand(encounter Encounter, combatant CombatantID, distanceFeet, rangeFeet int) (CheckCombatantGrappleCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return CheckCombatantGrappleCommand{}, err
	}
	if distanceFeet < 0 || rangeFeet < 0 {
		return CheckCombatantGrappleCommand{}, invalidInput("negative distance or range")
	}
	return CheckCombatantGrappleCommand{t, distanceFeet, rangeFeet}, nil
}

// CheckCombatantGrappleResult is the closed set of outcomes of CheckCombatantGrapple.Run. Variants: CheckCombatantGrappleApplied, CheckCombatantGrappleRejected.
type CheckCombatantGrappleResult interface{ isCheckCombatantGrappleResult() }

// CheckCombatantGrappleApplied: the command took effect, possibly changing nothing.
type CheckCombatantGrappleApplied struct{ encounterApplied }

// CheckCombatantGrappleRejected: the encounter refused the command; Rejection says why.
type CheckCombatantGrappleRejected struct{ encounterRejected }

func (CheckCombatantGrappleApplied) isCheckCombatantGrappleResult()  {}
func (CheckCombatantGrappleRejected) isCheckCombatantGrappleResult() {}

// CheckCombatantGrapple is the use case of checking whether a grapple still holds (Encounter.CheckGrapple). Create it with NewCheckCombatantGrapple; the zero value is silent and works.
type CheckCombatantGrapple struct{ encounterUse }

// NewCheckCombatantGrapple returns the use case. Options: WithTelemetry, WithClock.
func NewCheckCombatantGrapple(opts ...Option) CheckCombatantGrapple {
	return CheckCombatantGrapple{newEncounterUse(opts)}
}

// Run checking whether a grapple still holds (Encounter.CheckGrapple) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u CheckCombatantGrapple) Run(ctx context.Context, cmd CheckCombatantGrappleCommand) (CheckCombatantGrappleResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationCheckCombatantGrapple, spanCheckCombatantGrapple), cmd.encounter,
		func() (encounterTransition, error) {
			return cmd.encounter.CheckGrapple(cmd.combatant, cmd.distanceFeet, cmd.rangeFeet)
		},
		func(a encounterApplied, _ []DamageResolution) CheckCombatantGrappleResult {
			return CheckCombatantGrappleApplied{a}
		},
		func(r encounterRejected) CheckCombatantGrappleResult { return CheckCombatantGrappleRejected{r} })
}

// ReleaseCombatantGrappleCommand is the immutable input of ReleaseCombatantGrapple.Run. Build it with NewReleaseCombatantGrappleCommand; the zero value is invalid.
type ReleaseCombatantGrappleCommand struct {
	targetCommand
}

// NewReleaseCombatantGrappleCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id or a zero encounter. It never panics.
func NewReleaseCombatantGrappleCommand(encounter Encounter, combatant CombatantID) (ReleaseCombatantGrappleCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ReleaseCombatantGrappleCommand{}, err
	}
	return ReleaseCombatantGrappleCommand{t}, nil
}

// ReleaseCombatantGrappleResult is the closed set of outcomes of ReleaseCombatantGrapple.Run. Variants: ReleaseCombatantGrappleApplied, ReleaseCombatantGrappleRejected.
type ReleaseCombatantGrappleResult interface{ isReleaseCombatantGrappleResult() }

// ReleaseCombatantGrappleApplied: the command took effect, possibly changing nothing.
type ReleaseCombatantGrappleApplied struct{ encounterApplied }

// ReleaseCombatantGrappleRejected: the encounter refused the command; Rejection says why.
type ReleaseCombatantGrappleRejected struct{ encounterRejected }

func (ReleaseCombatantGrappleApplied) isReleaseCombatantGrappleResult()  {}
func (ReleaseCombatantGrappleRejected) isReleaseCombatantGrappleResult() {}

// ReleaseCombatantGrapple is the use case of releasing the grapple holding a combatant (Encounter.ReleaseGrapple). Create it with NewReleaseCombatantGrapple; the zero value is silent and works.
type ReleaseCombatantGrapple struct{ encounterUse }

// NewReleaseCombatantGrapple returns the use case. Options: WithTelemetry, WithClock.
func NewReleaseCombatantGrapple(opts ...Option) ReleaseCombatantGrapple {
	return ReleaseCombatantGrapple{newEncounterUse(opts)}
}

// Run releasing the grapple holding a combatant (Encounter.ReleaseGrapple) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u ReleaseCombatantGrapple) Run(ctx context.Context, cmd ReleaseCombatantGrappleCommand) (ReleaseCombatantGrappleResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationReleaseCombatantGrapple, spanReleaseCombatantGrapple), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.ReleaseGrapple(cmd.combatant) },
		func(a encounterApplied, _ []DamageResolution) ReleaseCombatantGrappleResult {
			return ReleaseCombatantGrappleApplied{a}
		},
		func(r encounterRejected) ReleaseCombatantGrappleResult { return ReleaseCombatantGrappleRejected{r} })
}

// EscapeCombatantGrappleCommand is the immutable input of EscapeCombatantGrapple.Run. Build it with NewEscapeCombatantGrappleCommand; the zero value is invalid.
type EscapeCombatantGrappleCommand struct {
	targetCommand
	outcome d20.Outcome
}

// NewEscapeCombatantGrappleCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero encounter or an outcome that is neither success nor failure. It never panics.
func NewEscapeCombatantGrappleCommand(encounter Encounter, combatant CombatantID, outcome d20.Outcome) (EscapeCombatantGrappleCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return EscapeCombatantGrappleCommand{}, err
	}
	if outcome != d20.Success && outcome != d20.Failure {
		return EscapeCombatantGrappleCommand{}, invalidInput("an outcome that is not success or failure")
	}
	return EscapeCombatantGrappleCommand{t, outcome}, nil
}

// EscapeCombatantGrappleResult is the closed set of outcomes of EscapeCombatantGrapple.Run. Variants: EscapeCombatantGrappleApplied, EscapeCombatantGrappleRejected.
type EscapeCombatantGrappleResult interface{ isEscapeCombatantGrappleResult() }

// EscapeCombatantGrappleApplied: the command took effect, possibly changing nothing.
type EscapeCombatantGrappleApplied struct{ encounterApplied }

// EscapeCombatantGrappleRejected: the encounter refused the command; Rejection says why.
type EscapeCombatantGrappleRejected struct{ encounterRejected }

func (EscapeCombatantGrappleApplied) isEscapeCombatantGrappleResult()  {}
func (EscapeCombatantGrappleRejected) isEscapeCombatantGrappleResult() {}

// EscapeCombatantGrapple is the use case of applying the outcome of an escape check (Encounter.EscapeGrapple). Create it with NewEscapeCombatantGrapple; the zero value is silent and works.
type EscapeCombatantGrapple struct{ encounterUse }

// NewEscapeCombatantGrapple returns the use case. Options: WithTelemetry, WithClock.
func NewEscapeCombatantGrapple(opts ...Option) EscapeCombatantGrapple {
	return EscapeCombatantGrapple{newEncounterUse(opts)}
}

// Run applying the outcome of an escape check (Encounter.EscapeGrapple) and records the shared signals and one srd5e.combat.deaths per wrapped Died. See
// the Encounter method for the rejections.
func (u EscapeCombatantGrapple) Run(ctx context.Context, cmd EscapeCombatantGrappleCommand) (EscapeCombatantGrappleResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationEscapeCombatantGrapple, spanEscapeCombatantGrapple), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.EscapeGrapple(cmd.combatant, cmd.outcome) },
		func(a encounterApplied, _ []DamageResolution) EscapeCombatantGrappleResult {
			return EscapeCombatantGrappleApplied{a}
		},
		func(r encounterRejected) EscapeCombatantGrappleResult { return EscapeCombatantGrappleRejected{r} })
}

// AdvanceEncounterTurnCommand is the immutable input of AdvanceEncounterTurn.Run. Build
// it with NewAdvanceEncounterTurnCommand; the zero value is invalid.
type AdvanceEncounterTurnCommand struct {
	encounter Encounter
	advance   TurnAdvance
}

// NewAdvanceEncounterTurnCommand returns the command, taking its own copy of the lists. It
// returns an error matching srderr.ErrInvalidInput for a zero encounter or a size that
// is not defined. An update naming a combatant the encounter does not hold is not an
// error: Run reports it as UnknownCombatantRejection. It never panics.
func NewAdvanceEncounterTurnCommand(encounter Encounter, advance TurnAdvance) (AdvanceEncounterTurnCommand, error) {
	if encounter.IsZero() || slices.ContainsFunc(advance.SharedSpaceOccupants, func(s content.CreatureSize) bool { return !s.Valid() }) {
		return AdvanceEncounterTurnCommand{}, invalidInput("advance needs a valid encounter and valid sizes")
	}
	advance.Updates, advance.SharedSpaceOccupants = slices.Clone(advance.Updates), slices.Clone(advance.SharedSpaceOccupants)
	return AdvanceEncounterTurnCommand{encounter, advance}, nil
}

// AdvanceEncounterTurnResult is the closed set of outcomes of AdvanceEncounterTurn.Run.
// Variants: AdvanceEncounterTurnApplied, AdvanceEncounterTurnRejected.
type AdvanceEncounterTurnResult interface{ isAdvanceEncounterTurnResult() }

// AdvanceEncounterTurnApplied: the turn ended and the next began.
type AdvanceEncounterTurnApplied struct{ encounterApplied }

// AdvanceEncounterTurnRejected: the encounter refused the command; Rejection says why.
type AdvanceEncounterTurnRejected struct{ encounterRejected }

func (AdvanceEncounterTurnApplied) isAdvanceEncounterTurnResult()  {}
func (AdvanceEncounterTurnRejected) isAdvanceEncounterTurnResult() {}

// AdvanceEncounterTurn is the use case of advancing the turn (Encounter.AdvanceTurn). It
// is a bare transition: shared-space Prone, review prompts and the like are the host's.
// Create it with NewAdvanceEncounterTurn; the zero value is silent and works.
type AdvanceEncounterTurn struct{ encounterUse }

// NewAdvanceEncounterTurn returns the use case. Options: WithTelemetry, WithClock.
func NewAdvanceEncounterTurn(opts ...Option) AdvanceEncounterTurn {
	return AdvanceEncounterTurn{newEncounterUse(opts)}
}

// Run advances the turn and records the shared signals. See Encounter.AdvanceTurn.
func (u AdvanceEncounterTurn) Run(ctx context.Context, cmd AdvanceEncounterTurnCommand) (AdvanceEncounterTurnResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationAdvanceEncounterTurn, spanAdvanceEncounterTurn), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.AdvanceTurn(cmd.advance) },
		func(a encounterApplied, _ []DamageResolution) AdvanceEncounterTurnResult {
			return AdvanceEncounterTurnApplied{a}
		},
		func(r encounterRejected) AdvanceEncounterTurnResult { return AdvanceEncounterTurnRejected{r} })
}

// ApplyDamageToCombatantCommand is the immutable input of ApplyDamageToCombatant.Run: the
// inputs of HitPointStatus.ApplyDamage, passed through unchanged. Knocking a creature out
// is the delivery MeleeAttackKnockOut. Build it with NewApplyDamageToCombatantCommand; the
// zero value is invalid.
type ApplyDamageToCombatantCommand struct {
	targetCommand
	damages     []Damage
	adjustments []DamageAdjustment
	delivery    DamageDelivery
	kind        HitKind
}

// NewApplyDamageToCombatantCommand returns the command, taking its own copy of the
// slices. It returns an error matching srderr.ErrInvalidInput for a blank id, a zero
// encounter, no Damage, an invalid Damage, a nil adjustment or an invalid delivery or
// kind. It never panics.
func NewApplyDamageToCombatantCommand(encounter Encounter, combatant CombatantID, damages []Damage, adjustments []DamageAdjustment,
	delivery DamageDelivery, kind HitKind) (ApplyDamageToCombatantCommand, error) {
	t, err := newTargetCommand(encounter, combatant)
	if err != nil {
		return ApplyDamageToCombatantCommand{}, err
	}
	if err := checkDamageInput(combatant, damages, adjustments, delivery, kind); err != nil {
		return ApplyDamageToCombatantCommand{}, err
	}
	return ApplyDamageToCombatantCommand{t, slices.Clone(damages), slices.Clone(adjustments), delivery, kind}, nil
}

// ApplyDamageToCombatantResult is the closed set of outcomes of ApplyDamageToCombatant.Run.
// Variants: ApplyDamageToCombatantApplied, ApplyDamageToCombatantRejected.
type ApplyDamageToCombatantResult interface{ isApplyDamageToCombatantResult() }

// ApplyDamageToCombatantApplied: the damage took effect, possibly changing nothing. It
// also carries the DamageResolution of each Damage.
type ApplyDamageToCombatantApplied struct {
	encounterApplied
	resolutions []DamageResolution
}

// Resolutions returns a copy of the per-instance damage resolutions.
func (r ApplyDamageToCombatantApplied) Resolutions() []DamageResolution {
	return slices.Clone(r.resolutions)
}

// ApplyDamageToCombatantRejected: the encounter refused the command; Rejection says why.
type ApplyDamageToCombatantRejected struct{ encounterRejected }

func (ApplyDamageToCombatantApplied) isApplyDamageToCombatantResult()  {}
func (ApplyDamageToCombatantRejected) isApplyDamageToCombatantResult() {}

// ApplyDamageToCombatant is the use case of applying damage to a combatant
// (Encounter.ApplyDamage). It is a bare transition: Concentration checks and the like are
// the host's. Create it with NewApplyDamageToCombatant; the zero value is silent and works.
type ApplyDamageToCombatant struct{ encounterUse }

// NewApplyDamageToCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewApplyDamageToCombatant(opts ...Option) ApplyDamageToCombatant {
	return ApplyDamageToCombatant{newEncounterUse(opts)}
}

// Run applies the damage and records the shared signals and one srd5e.combat.deaths per
// wrapped Died. See Encounter.ApplyDamage for the rejections.
func (u ApplyDamageToCombatant) Run(ctx context.Context, cmd ApplyDamageToCombatantCommand) (ApplyDamageToCombatantResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationApplyDamageToCombatant, spanApplyDamageToCombatant), cmd.encounter,
		func() (encounterTransition, error) {
			return cmd.encounter.ApplyDamage(cmd.combatant, cmd.damages, cmd.adjustments, cmd.delivery, cmd.kind)
		},
		func(a encounterApplied, res []DamageResolution) ApplyDamageToCombatantResult {
			return ApplyDamageToCombatantApplied{a, res}
		},
		func(r encounterRejected) ApplyDamageToCombatantResult { return ApplyDamageToCombatantRejected{r} })
}

// EndEncounterCommand is the immutable input of EndEncounter.Run. Build it with
// NewEndEncounterCommand; the zero value is invalid.
type EndEncounterCommand struct {
	encounter Encounter
	reason    EndReason
}

// NewEndEncounterCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a zero encounter, a nil reason or a SideDefeated of a
// side that is not valid. It never panics.
func NewEndEncounterCommand(encounter Encounter, reason EndReason) (EndEncounterCommand, error) {
	if encounter.IsZero() || !validEndReason(reason) {
		return EndEncounterCommand{}, invalidInput("end needs a valid encounter and reason")
	}
	return EndEncounterCommand{encounter, reason}, nil
}

// EndEncounterResult is the closed set of outcomes of EndEncounter.Run. Variants:
// EndEncounterApplied, EndEncounterRejected.
type EndEncounterResult interface{ isEndEncounterResult() }

// EndEncounterApplied: the encounter ended.
type EndEncounterApplied struct{ encounterApplied }

// EndEncounterRejected: the encounter had already ended; Rejection says so.
type EndEncounterRejected struct{ encounterRejected }

func (EndEncounterApplied) isEndEncounterResult()  {}
func (EndEncounterRejected) isEndEncounterResult() {}

// EndEncounter is the use case of ending an encounter. It does not verify a defeat claim.
// Create it with NewEndEncounter; the zero value is silent and works.
type EndEncounter struct{ encounterUse }

// NewEndEncounter returns the use case. Options: WithTelemetry, WithClock.
func NewEndEncounter(opts ...Option) EndEncounter { return EndEncounter{newEncounterUse(opts)} }

// Run ends the encounter and records the shared signals.
func (u EndEncounter) Run(ctx context.Context, cmd EndEncounterCommand) (EndEncounterResult, error) {
	return runTransition(ctx, u.encounterUse, transitionOp(valueOperationEndEncounter, spanEndEncounter), cmd.encounter,
		func() (encounterTransition, error) { return cmd.encounter.end(cmd.reason) },
		func(a encounterApplied, _ []DamageResolution) EndEncounterResult { return EndEncounterApplied{a} },
		func(r encounterRejected) EndEncounterResult { return EndEncounterRejected{r} })
}
