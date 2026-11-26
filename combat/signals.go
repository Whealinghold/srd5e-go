package combat

// The signals the combat use cases emit, defined once. Names and attribute keys
// are a public contract catalogued in doc/observability/signals.md. The shared
// keys telemetry.AttrOutcome and telemetry.AttrErrorType are used as they are.
// Later specs add their own constants here (prefix metric, span, msg, attr or
// value).
const (
	// spanPrefix starts the span name of every operation.
	spanPrefix = "srd5e.combat."

	// metricOperations counts use case calls that returned a result ({operation}).
	metricOperations = "srd5e.combat.operations"
	// metricOperationDuration records call durations (s), failed calls included.
	metricOperationDuration = "srd5e.combat.operation_duration"
	// metricErrors counts errors leaving a use case ({error}): port failures (error.type is the port's) and invalid input (valueErrorTypeInvalidInput).
	metricErrors = "srd5e.combat.errors"
	// metricEvents counts domain events produced ({event}).
	metricEvents = "srd5e.combat.events"
	// valueErrorTypeInvalidInput is the error.type of an error matching srderr.ErrInvalidInput that is not a port failure: a command the host built wrongly.
	valueErrorTypeInvalidInput = "invalid_input"
)

// Attribute keys.
const (
	// attrOperation is the snake_case use case name.
	attrOperation = "srd5e.combat.operation"
	// attrResult is the finer, per use case result name.
	attrResult = "srd5e.combat.result"
	// attrResultClass is the class of the result: a valueResultClass* value.
	attrResultClass = "srd5e.combat.result_class"
	// attrEvent is the Kind of a produced event.
	attrEvent = "srd5e.combat.event"
)

// Attribute values.
const (
	// valueResultClassApplied is the value of srd5e.combat.result_class for an applied result.
	valueResultClassApplied = "applied"
	// valueResultClassRefused is the value of srd5e.combat.result_class for a refused result.
	valueResultClassRefused = "refused"
	// valueResultClassInvalid is the value of srd5e.combat.result_class for an invalid result.
	valueResultClassInvalid = "invalid"
	// valueOutcomeSuccess is the value of srd5e.outcome for an applied result.
	valueOutcomeSuccess = "success"
	// valueOutcomeFailure is the value of srd5e.outcome for a refused or invalid result.
	valueOutcomeFailure = "failure"
)

// Operations of the hit point use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationApplyDamage is the value of srd5e.combat.operation for ApplyDamage.
	valueOperationApplyDamage = "apply_damage"
	// spanApplyDamage is the span of ApplyDamage.
	spanApplyDamage = "srd5e.combat.apply_damage"
	// valueOperationRestoreHitPoints is the value of srd5e.combat.operation for RestoreHitPoints.
	valueOperationRestoreHitPoints = "restore_hit_points"
	// spanRestoreHitPoints is the span of RestoreHitPoints.
	spanRestoreHitPoints = "srd5e.combat.restore_hit_points"
	// valueOperationGainTemporaryHitPoints is the value of srd5e.combat.operation for GainTemporaryHitPoints.
	valueOperationGainTemporaryHitPoints = "gain_temporary_hit_points"
	// spanGainTemporaryHitPoints is the span of GainTemporaryHitPoints.
	spanGainTemporaryHitPoints = "srd5e.combat.gain_temporary_hit_points"
	// valueOperationEndTemporaryHitPoints is the value of srd5e.combat.operation for EndTemporaryHitPoints.
	valueOperationEndTemporaryHitPoints = "end_temporary_hit_points"
	// spanEndTemporaryHitPoints is the span of EndTemporaryHitPoints.
	spanEndTemporaryHitPoints = "srd5e.combat.end_temporary_hit_points"
	// valueOperationMakeDeathSavingThrow is the value of srd5e.combat.operation for MakeDeathSavingThrow.
	valueOperationMakeDeathSavingThrow = "make_death_saving_throw"
	// spanMakeDeathSavingThrow is the span of MakeDeathSavingThrow.
	spanMakeDeathSavingThrow = "srd5e.combat.make_death_saving_throw"
	// valueOperationStabilizeCreature is the value of srd5e.combat.operation for StabilizeCreature.
	valueOperationStabilizeCreature = "stabilize_creature"
	// spanStabilizeCreature is the span of StabilizeCreature.
	spanStabilizeCreature = "srd5e.combat.stabilize_creature"
	// valueOperationRollStableRecoveryDelay is the value of srd5e.combat.operation for RollStableRecoveryDelay.
	valueOperationRollStableRecoveryDelay = "roll_stable_recovery_delay"
	// spanRollStableRecoveryDelay is the span of RollStableRecoveryDelay.
	spanRollStableRecoveryDelay = "srd5e.combat.roll_stable_recovery_delay"
	// valueOperationRecoverStableCreature is the value of srd5e.combat.operation for RecoverStableCreature.
	valueOperationRecoverStableCreature = "recover_stable_creature"
	// spanRecoverStableCreature is the span of RecoverStableCreature.
	spanRecoverStableCreature = "srd5e.combat.recover_stable_creature"
	// valueOperationReduceHitPointMaximum is the value of srd5e.combat.operation for ReduceHitPointMaximum.
	valueOperationReduceHitPointMaximum = "reduce_hit_point_maximum"
	// spanReduceHitPointMaximum is the span of ReduceHitPointMaximum.
	spanReduceHitPointMaximum = "srd5e.combat.reduce_hit_point_maximum"
	// valueOperationRecordExhaustionDeath is the value of srd5e.combat.operation for RecordExhaustionDeath.
	valueOperationRecordExhaustionDeath = "record_exhaustion_death"
	// spanRecordExhaustionDeath is the span of RecordExhaustionDeath.
	spanRecordExhaustionDeath = "srd5e.combat.record_exhaustion_death"
)

// Signals of the hit point use cases (hit-points-and-damage.md section 7).
const (
	// metricDamageResolved counts instances of damage by type and effect ({instance}).
	metricDamageResolved = "srd5e.combat.damage_resolved"
	// metricHitPointsLost records the Hit Points lost per applied ApplyDamage command ({hit_point}).
	metricHitPointsLost = "srd5e.combat.hit_points_lost"
	// metricHitPointsRestored records the Hit Points regained per applied RestoreHitPoints command ({hit_point}).
	metricHitPointsRestored = "srd5e.combat.hit_points_restored"
	// metricDeathSaves counts Death Saving Throws by result ({roll}).
	metricDeathSaves = "srd5e.combat.death_saves"
	// metricHitPointRefusals counts refused hit point operations by reason ({refusal}).
	metricHitPointRefusals = "srd5e.combat.hit_point_refusals"
	// metricDeaths counts creatures that died, by cause ({creature}).
	metricDeaths = "srd5e.combat.deaths"

	// attrDamageType is the lowercase Damage Type of one instance of damage.
	attrDamageType = "srd5e.combat.damage_type"
	// attrDamageEffect is a valueDamageEffect* value.
	attrDamageEffect = "srd5e.combat.damage_effect"
	// attrDeathSaveResult is a valueDeathSave* value.
	attrDeathSaveResult = "srd5e.combat.death_save_result"
	// attrRefusalReason is the snake_case HitPointRejection.
	attrRefusalReason = "srd5e.combat.refusal_reason"
	// attrDeathCause is the snake_case DeathCause.
	attrDeathCause = "srd5e.combat.death_cause"
)

// Values of srd5e.combat.damage_effect.
const (
	// valueDamageEffectUnchanged is the value of srd5e.combat.damage_effect for damage no affinity changed.
	valueDamageEffectUnchanged = "unchanged"
	// valueDamageEffectImmune is the value of srd5e.combat.damage_effect for damage Immunity cancelled.
	valueDamageEffectImmune = "immune"
	// valueDamageEffectReducedToZero is the value of srd5e.combat.damage_effect for damage reduced to nothing.
	valueDamageEffectReducedToZero = "reduced_to_zero"
	// valueDamageEffectResistedAndVulnerable is the value of srd5e.combat.damage_effect for damage that both applied to.
	valueDamageEffectResistedAndVulnerable = "resisted_and_vulnerable"
	// valueDamageEffectResisted is the value of srd5e.combat.damage_effect for damage halved by Resistance.
	valueDamageEffectResisted = "resisted"
	// valueDamageEffectVulnerable is the value of srd5e.combat.damage_effect for damage doubled by Vulnerability.
	valueDamageEffectVulnerable = "vulnerable"
)

// Values of srd5e.combat.death_save_result.
const (
	// valueDeathSaveSuccess is the value of srd5e.combat.death_save_result for one success.
	valueDeathSaveSuccess = "success"
	// valueDeathSaveFailure is the value of srd5e.combat.death_save_result for one failure.
	valueDeathSaveFailure = "failure"
	// valueDeathSaveCriticalFailure is the value of srd5e.combat.death_save_result for a natural 1.
	valueDeathSaveCriticalFailure = "critical_failure"
	// valueDeathSaveCriticalSuccess is the value of srd5e.combat.death_save_result for a natural 20.
	valueDeathSaveCriticalSuccess = "critical_success"
)

// Values of srd5e.combat.result for the hit point use cases (hit-points-and-damage.md table 4.10a).
const (
	// valueResultKilled is the value of srd5e.combat.result when the call killed the creature.
	valueResultKilled = "killed"
	// valueResultKnockedOut is the value of srd5e.combat.result when a melee attack knocked the creature out.
	valueResultKnockedOut = "knocked_out"
	// valueResultDroppedToZero is the value of srd5e.combat.result when the creature dropped to 0 Hit Points.
	valueResultDroppedToZero = "dropped_to_zero"
	// valueResultLostStability is the value of srd5e.combat.result when a Stable creature took damage.
	valueResultLostStability = "lost_stability"
	// valueResultAbsorbed is the value of srd5e.combat.result when Temporary Hit Points took all the damage.
	valueResultAbsorbed = "absorbed"
	// valueResultDamaged is the value of srd5e.combat.result when the creature lost Hit Points.
	valueResultDamaged = "damaged"
	// valueResultNoEffect is the value of srd5e.combat.result when the call changed nothing.
	valueResultNoEffect = "no_effect"
	// valueResultRecovered is the value of srd5e.combat.result when a creature came back from 0 Hit Points.
	valueResultRecovered = "recovered"
	// valueResultHealed is the value of srd5e.combat.result when Hit Points were regained.
	valueResultHealed = "healed"
	// valueResultGained is the value of srd5e.combat.result when Temporary Hit Points were gained.
	valueResultGained = "gained"
	// valueResultEnded is the value of srd5e.combat.result when Temporary Hit Points ended or Exhaustion ended.
	valueResultEnded = "ended"
	// valueResultStabilized is the value of srd5e.combat.result when the creature became Stable.
	valueResultStabilized = "stabilized"
	// valueResultSucceeded is the value of srd5e.combat.result for a Death Saving Throw success.
	valueResultSucceeded = "succeeded"
	// valueResultFailed is the value of srd5e.combat.result for a Death Saving Throw failure.
	valueResultFailed = "failed"
	// valueResultRolled is the value of srd5e.combat.result when the stable recovery delay was rolled.
	valueResultRolled = "rolled"
	// valueResultReduced is the value of srd5e.combat.result when the Hit Point maximum or Exhaustion was reduced.
	valueResultReduced = "reduced"
)

// valueResultRejectedDead is the value of srd5e.combat.result when the creature is Dead.
const valueResultRejectedDead = "rejected_dead"

// Operations of the condition use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationApplyConditionToSet is the value of srd5e.combat.operation for ApplyConditionToSet.
	valueOperationApplyConditionToSet = "apply_condition_to_set"
	// spanApplyConditionToSet is the span of ApplyConditionToSet.
	spanApplyConditionToSet = "srd5e.combat.apply_condition_to_set"
	// valueOperationRemoveConditionFromSet is the value of srd5e.combat.operation for RemoveConditionFromSet.
	valueOperationRemoveConditionFromSet = "remove_condition_from_set"
	// spanRemoveConditionFromSet is the span of RemoveConditionFromSet.
	spanRemoveConditionFromSet = "srd5e.combat.remove_condition_from_set"
	// valueOperationReduceExhaustionOnSet is the value of srd5e.combat.operation for ReduceExhaustionOnSet.
	valueOperationReduceExhaustionOnSet = "reduce_exhaustion_on_set"
	// spanReduceExhaustionOnSet is the span of ReduceExhaustionOnSet.
	spanReduceExhaustionOnSet = "srd5e.combat.reduce_exhaustion_on_set"
)

// Signals of the condition use cases (conditions.md section 7).
const (
	// metricConditionActive records the effective conditions on a creature after a call ({condition}).
	metricConditionActive = "srd5e.combat.condition.active"
	// attrConditionKind is the snake_case ConditionKind a condition call was about.
	attrConditionKind = "srd5e.combat.condition.kind"
)

// Values of srd5e.combat.result for the condition use cases (conditions.md table 4.11).
const (
	// valueResultAdded is the value of srd5e.combat.result when a condition was added.
	valueResultAdded = "added"
	// valueResultAlreadyHeld is the value of srd5e.combat.result when the kind already held.
	valueResultAlreadyHeld = "already_held"
	// valueResultImmune is the value of srd5e.combat.result when the creature is immune to the kind.
	valueResultImmune = "immune"
	// valueResultAtMaximum is the value of srd5e.combat.result when Exhaustion is at level 6.
	valueResultAtMaximum = "at_maximum"
	// valueResultCleared is the value of srd5e.combat.result when an explicit condition was removed.
	valueResultCleared = "cleared"
	// valueResultLowered is the value of srd5e.combat.result when Exhaustion went down one level.
	valueResultLowered = "lowered"
	// valueResultRemovalNotPresent is the value of srd5e.combat.result when the kind to remove was not held.
	valueResultRemovalNotPresent = "removal_not_present"
	// valueResultImpliedBy is the value of srd5e.combat.result when the kind holds only by implication.
	valueResultImpliedBy = "implied_by"
	// valueResultReductionNotPresent is the value of srd5e.combat.result when there was no Exhaustion to reduce.
	valueResultReductionNotPresent = "reduction_not_present"
)

// Operations of the grapple use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationBeginGrapple is the value of srd5e.combat.operation for BeginGrapple.
	valueOperationBeginGrapple = "begin_grapple"
	// spanBeginGrapple is the span of BeginGrapple.
	spanBeginGrapple = "srd5e.combat.begin_grapple"
	// valueOperationReleaseGrapple is the value of srd5e.combat.operation for ReleaseGrapple.
	valueOperationReleaseGrapple = "release_grapple"
	// spanReleaseGrapple is the span of ReleaseGrapple.
	spanReleaseGrapple = "srd5e.combat.release_grapple"
	// valueOperationEscapeGrappleFromRecord is the value of srd5e.combat.operation for EscapeGrappleFromRecord.
	valueOperationEscapeGrappleFromRecord = "escape_grapple_from_record"
	// spanEscapeGrappleFromRecord is the span of EscapeGrappleFromRecord.
	spanEscapeGrappleFromRecord = "srd5e.combat.escape_grapple_from_record"
	// valueOperationReviewGrapples is the value of srd5e.combat.operation for ReviewGrapples.
	valueOperationReviewGrapples = "review_grapples"
	// spanReviewGrapples is the span of ReviewGrapples.
	spanReviewGrapples = "srd5e.combat.review_grapples"
	// valueOperationDropGrappleRecord is the value of srd5e.combat.operation for DropGrappleRecord.
	valueOperationDropGrappleRecord = "drop_grapple_record"
	// spanDropGrappleRecord is the span of DropGrappleRecord.
	spanDropGrappleRecord = "srd5e.combat.drop_grapple_record"
)

// Signals of the grapple use cases (grappling.md section 7).
const (
	// metricGrappleEnded counts grapples that ended, by reason ({grapple}).
	metricGrappleEnded = "srd5e.combat.grapple.ended"
	// metricGrappleInForce records the size of the Grapples value a result carries ({grapple}).
	metricGrappleInForce = "srd5e.combat.grapple.in_force"
	// attrGrappleEndReason is a valueGrappleEnd* value.
	attrGrappleEndReason = "srd5e.combat.grapple.end_reason"
)

// Values of srd5e.combat.grapple.end_reason besides the four GrappleEndCause names
// (released, escaped, grappler_incapacitated, out_of_range).
const (
	// valueGrappleEndConditionEnded is the value of srd5e.combat.grapple.end_reason for a dropped record.
	valueGrappleEndConditionEnded = "condition_ended"
)

// Values of srd5e.combat.result for the grapple use cases (grappling.md section 7);
// refusals and invalid input use the snake_case names of GrappleRejection and
// GrappleProblem.
const (
	// valueResultApplied is the value of srd5e.combat.result when a grapple operation took effect.
	valueResultApplied = "applied"
	// valueResultEscaped is the value of srd5e.combat.result when the escape check succeeded.
	valueResultEscaped = "escaped"
	// valueResultHeld is the value of srd5e.combat.result when the escape check failed.
	valueResultHeld = "held"
)

// Operations of the hazard and object use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationRollBurningDamage is the value of srd5e.combat.operation for RollBurningDamage.
	valueOperationRollBurningDamage = "roll_burning_damage"
	// spanRollBurningDamage is the span of RollBurningDamage.
	spanRollBurningDamage = "srd5e.combat.roll_burning_damage"
	// valueOperationExtinguishBurning is the value of srd5e.combat.operation for ExtinguishBurning.
	valueOperationExtinguishBurning = "extinguish_burning"
	// spanExtinguishBurning is the span of ExtinguishBurning.
	spanExtinguishBurning = "srd5e.combat.extinguish_burning"
	// valueOperationResolveFall is the value of srd5e.combat.operation for ResolveFall.
	valueOperationResolveFall = "resolve_fall"
	// spanResolveFall is the span of ResolveFall.
	spanResolveFall = "srd5e.combat.resolve_fall"
	// valueOperationLandAfterFall is the value of srd5e.combat.operation for LandAfterFall.
	valueOperationLandAfterFall = "land_after_fall"
	// spanLandAfterFall is the span of LandAfterFall.
	spanLandAfterFall = "srd5e.combat.land_after_fall"
	// valueOperationResolveDehydrationDayEnd is the value of srd5e.combat.operation for ResolveDehydrationDayEnd.
	valueOperationResolveDehydrationDayEnd = "resolve_dehydration_day_end"
	// spanResolveDehydrationDayEnd is the span of ResolveDehydrationDayEnd.
	spanResolveDehydrationDayEnd = "srd5e.combat.resolve_dehydration_day_end"
	// valueOperationResolveMalnutritionDayEnd is the value of srd5e.combat.operation for ResolveMalnutritionDayEnd.
	valueOperationResolveMalnutritionDayEnd = "resolve_malnutrition_day_end"
	// spanResolveMalnutritionDayEnd is the span of ResolveMalnutritionDayEnd.
	spanResolveMalnutritionDayEnd = "srd5e.combat.resolve_malnutrition_day_end"
	// valueOperationAdvanceBreath is the value of srd5e.combat.operation for AdvanceBreath.
	valueOperationAdvanceBreath = "advance_breath"
	// spanAdvanceBreath is the span of AdvanceBreath.
	spanAdvanceBreath = "srd5e.combat.advance_breath"
	// valueOperationResolveSuffocationTurnEnd is the value of srd5e.combat.operation for ResolveSuffocationTurnEnd.
	valueOperationResolveSuffocationTurnEnd = "resolve_suffocation_turn_end"
	// spanResolveSuffocationTurnEnd is the span of ResolveSuffocationTurnEnd.
	spanResolveSuffocationTurnEnd = "srd5e.combat.resolve_suffocation_turn_end"
	// valueOperationRestoreBreathing is the value of srd5e.combat.operation for RestoreBreathing.
	valueOperationRestoreBreathing = "restore_breathing"
	// spanRestoreBreathing is the span of RestoreBreathing.
	spanRestoreBreathing = "srd5e.combat.restore_breathing"
	// valueOperationRemoveExhaustionLevels is the value of srd5e.combat.operation for RemoveExhaustionLevels.
	valueOperationRemoveExhaustionLevels = "remove_exhaustion_levels"
	// spanRemoveExhaustionLevels is the span of RemoveExhaustionLevels.
	spanRemoveExhaustionLevels = "srd5e.combat.remove_exhaustion_levels"
	// valueOperationRollObjectHitPoints is the value of srd5e.combat.operation for RollObjectHitPoints.
	valueOperationRollObjectHitPoints = "roll_object_hit_points"
	// spanRollObjectHitPoints is the span of RollObjectHitPoints.
	spanRollObjectHitPoints = "srd5e.combat.roll_object_hit_points"
	// valueOperationApplyDamageToObject is the value of srd5e.combat.operation for ApplyDamageToObject.
	valueOperationApplyDamageToObject = "apply_damage_to_object"
	// spanApplyDamageToObject is the span of ApplyDamageToObject.
	spanApplyDamageToObject = "srd5e.combat.apply_damage_to_object"
)

// Signals of the hazard use cases (environment-hazards-and-objects.md section 7).
const (
	// metricHazardDamageRolled records the hazard damage handed to the host, as rolled ({hit_point}).
	metricHazardDamageRolled = "srd5e.combat.hazard.damage_rolled"
	// metricHazardExhaustionGained counts Exhaustion levels hazards applied ({level}).
	metricHazardExhaustionGained = "srd5e.combat.hazard.exhaustion_gained"
	// attrHazardKind is the lowercase HazardKind a hazard signal is about.
	attrHazardKind = "srd5e.combat.hazard.kind"
)

// Values of srd5e.combat.result for the hazard and object use cases (section 7);
// the Exhaustion results use HazardExhaustionOutcome.String, the refusals the
// String of their rejection, and rolled, damaged and immune are shared with other
// use cases.
const (
	// valueResultDamageRolled is the value of srd5e.combat.result when a hazard rolled damage.
	valueResultDamageRolled = "damage_rolled"
	// valueResultNoDamage is the value of srd5e.combat.result when a fall or an object took no damage.
	valueResultNoDamage = "no_damage"
	// valueResultExtinguished is the value of srd5e.combat.result when the fire went out.
	valueResultExtinguished = "extinguished"
	// valueResultLandedProne is the value of srd5e.combat.result when a fall left the creature Prone.
	valueResultLandedProne = "landed_prone"
	// valueResultLandedStanding is the value of srd5e.combat.result when a fall left the creature standing.
	valueResultLandedStanding = "landed_standing"
	// valueResultSaveRequired is the value of srd5e.combat.result when a Constitution saving throw is needed.
	valueResultSaveRequired = "save_required"
	// valueResultAdvanced is the value of srd5e.combat.result when a breathing change took effect.
	valueResultAdvanced = "advanced"
	// valueResultRestored is the value of srd5e.combat.result when breathing was restored.
	valueResultRestored = "restored"
	// valueResultRemoved is the value of srd5e.combat.result when Exhaustion levels were removed.
	valueResultRemoved = "removed"
	// valueResultBlocked is the value of srd5e.combat.result when no Exhaustion level could be removed.
	valueResultBlocked = "blocked"
	// valueResultSplitRequired is the value of srd5e.combat.result when an object is too big for the table.
	valueResultSplitRequired = "split_required"
	// valueResultDeflected is the value of srd5e.combat.result when a Damage Threshold stopped the damage.
	valueResultDeflected = "deflected"
	// valueResultDestroyed is the value of srd5e.combat.result when an object reached 0 Hit Points.
	valueResultDestroyed = "destroyed"
	// valueResultAlreadyDestroyed is the value of srd5e.combat.result when the object was already destroyed.
	valueResultAlreadyDestroyed = "already_destroyed"
)

// Operations of the attack use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationResolveAttack is the value of srd5e.combat.operation for ResolveAttack.
	valueOperationResolveAttack = "resolve_attack"
	// spanResolveAttack is the span of ResolveAttack.
	spanResolveAttack = "srd5e.combat.resolve_attack"
	// valueOperationResolveUnarmedStrike is the value of srd5e.combat.operation for ResolveUnarmedStrike.
	valueOperationResolveUnarmedStrike = "resolve_unarmed_strike"
	// spanResolveUnarmedStrike is the span of ResolveUnarmedStrike.
	spanResolveUnarmedStrike = "srd5e.combat.resolve_unarmed_strike"
)

// Signals of the attack use cases (attacks.md section 7). The srd5e.d20_test.roll_mode
// and srd5e.d20_test.critical_hit keys are d20's own constants.
const (
	// metricAttackModifiers records the modifiers in the RollModifiers of a returned ResolveAttack call ({modifier}).
	metricAttackModifiers = "srd5e.combat.attack.modifiers"
	// msgAttackResolved is the FINE log message of a returned ResolveAttack call.
	msgAttackResolved = "attack resolved"
	// msgUnarmedStrikeResolved is the FINE log message of a returned ResolveUnarmedStrike call.
	msgUnarmedStrikeResolved = "unarmed strike resolved"

	// attrAttackForm is the AttackForm of a ResolveAttack call: its String.
	attrAttackForm = "srd5e.combat.attack.form"
	// attrAttackReason is the AttackNotPossibleReason, present only for not_possible: its String.
	attrAttackReason = "srd5e.combat.attack.reason"
	// attrAttackCover is the applicable Cover: its String.
	attrAttackCover = "srd5e.combat.attack.cover"
	// attrAttackAutomaticCritical is a bool: whether the AutomaticCritical input was automatic.
	attrAttackAutomaticCritical = "srd5e.combat.attack.automatic_critical"
	// attrUnarmedStrikeOption is the option of an Unarmed Strike: damage, grapple or shove.
	attrUnarmedStrikeOption = "srd5e.combat.unarmed_strike.option"
)

// Values of srd5e.combat.result for the attack use cases (attacks.md FR-122).
const (
	// valueResultHit is the value of srd5e.combat.result for an attack that hit.
	valueResultHit = "hit"
	// valueResultCriticalHit is the value of srd5e.combat.result for an attack that was a Critical Hit.
	valueResultCriticalHit = "critical_hit"
	// valueResultMiss is the value of srd5e.combat.result for an attack that missed, a forced miss included.
	valueResultMiss = "miss"
	// valueResultAutomaticMiss is the value of srd5e.combat.result for an attack that missed without a roll.
	valueResultAutomaticMiss = "automatic_miss"
	// valueResultNotPossible is the value of srd5e.combat.result for an attack that cannot be made.
	valueResultNotPossible = "not_possible"
	// valueResultDamageHit is the value of srd5e.combat.result for an Unarmed Strike Damage option that hit.
	valueResultDamageHit = "damage_hit"
	// valueResultDamageCriticalHit is the value of srd5e.combat.result for an Unarmed Strike Damage option Critical Hit.
	valueResultDamageCriticalHit = "damage_critical_hit"
	// valueResultDamageMiss is the value of srd5e.combat.result for an Unarmed Strike Damage option that missed.
	valueResultDamageMiss = "damage_miss"
	// valueResultGrappleResisted is the value of srd5e.combat.result for a Grapple the target saved against.
	valueResultGrappleResisted = "grapple_resisted"
	// valueResultGrappleApplied is the value of srd5e.combat.result for a Grapple whose save failed.
	valueResultGrappleApplied = "grapple_applied"
	// valueResultShoveResisted is the value of srd5e.combat.result for a Shove the target saved against.
	valueResultShoveResisted = "shove_resisted"
	// valueResultShoveApplied is the value of srd5e.combat.result for a Shove whose save failed.
	valueResultShoveApplied = "shove_applied"
)

// Values of srd5e.combat.unarmed_strike.option.
const (
	// valueUnarmedStrikeDamage is the value of srd5e.combat.unarmed_strike.option for the Damage option.
	valueUnarmedStrikeDamage = "damage"
	// valueUnarmedStrikeGrapple is the value of srd5e.combat.unarmed_strike.option for the Grapple option.
	valueUnarmedStrikeGrapple = "grapple"
	// valueUnarmedStrikeShove is the value of srd5e.combat.unarmed_strike.option for the Shove option.
	valueUnarmedStrikeShove = "shove"
)

// Operations of the mounting use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationMountCreature is the value of srd5e.combat.operation for MountCreature.
	valueOperationMountCreature = "mount_creature"
	// spanMountCreature is the span of MountCreature.
	spanMountCreature = "srd5e.combat.mount_creature"
	// valueOperationDismountCreature is the value of srd5e.combat.operation for DismountCreature.
	valueOperationDismountCreature = "dismount_creature"
	// spanDismountCreature is the span of DismountCreature.
	spanDismountCreature = "srd5e.combat.dismount_creature"
	// valueOperationResolveFallingOff is the value of srd5e.combat.operation for ResolveFallingOff.
	valueOperationResolveFallingOff = "resolve_falling_off"
	// spanResolveFallingOff is the span of ResolveFallingOff.
	spanResolveFallingOff = "srd5e.combat.resolve_falling_off"
)

// Signals of the mounting use cases (mounted-and-underwater-combat.md section 7).
const (
	// metricMountingMovementSpent records the movement cost of each accepted mount or dismount ([ft_i]).
	metricMountingMovementSpent = "srd5e.combat.mounting_movement_spent"
	// attrMountControl is the snake_case MountControl: requested for mount_creature, the ride's otherwise.
	attrMountControl = "srd5e.combat.mount_control"
	// attrFallingOffTrigger is the snake_case FallingOffTrigger of a resolve_falling_off call.
	attrFallingOffTrigger = "srd5e.combat.falling_off_trigger"
)

// Values of srd5e.combat.result for ResolveFallingOff; the mounting refusals use the
// String of their rejection and an accepted mount or dismount uses valueResultApplied.
const (
	// valueResultStayedMounted is the value of srd5e.combat.result when the rider passed the save.
	valueResultStayedMounted = "stayed_mounted"
	// valueResultFellOff is the value of srd5e.combat.result when the rider failed the save.
	valueResultFellOff = "fell_off"
)

// Operations of the gear use cases: the snake_case value of srd5e.combat.operation and
// the matching span srd5e.combat.<operation>.
const (
	// valueOperationDeployGear is the value of srd5e.combat.operation for DeployGear.
	valueOperationDeployGear = "deploy_gear"
	// spanDeployGear is the span of DeployGear.
	spanDeployGear = "srd5e.combat.deploy_gear"
	// valueOperationMaintainDeployedGear is the value of srd5e.combat.operation for MaintainDeployedGear.
	valueOperationMaintainDeployedGear = "maintain_deployed_gear"
	// spanMaintainDeployedGear is the span of MaintainDeployedGear.
	spanMaintainDeployedGear = "srd5e.combat.maintain_deployed_gear"
	// valueOperationResolveGearEntry is the value of srd5e.combat.operation for ResolveGearEntry.
	valueOperationResolveGearEntry = "resolve_gear_entry"
	// spanResolveGearEntry is the span of ResolveGearEntry.
	spanResolveGearEntry = "srd5e.combat.resolve_gear_entry"
	// valueOperationThrowNet is the value of srd5e.combat.operation for ThrowNet.
	valueOperationThrowNet = "throw_net"
	// spanThrowNet is the span of ThrowNet.
	spanThrowNet = "srd5e.combat.throw_net"
	// valueOperationBindCreature is the value of srd5e.combat.operation for BindCreature.
	valueOperationBindCreature = "bind_creature"
	// spanBindCreature is the span of BindCreature.
	spanBindCreature = "srd5e.combat.bind_creature"
	// valueOperationReleaseFromHold is the value of srd5e.combat.operation for ReleaseFromHold.
	valueOperationReleaseFromHold = "release_from_hold"
	// spanReleaseFromHold is the span of ReleaseFromHold.
	spanReleaseFromHold = "srd5e.combat.release_from_hold"
)

// Signals of the gear use cases (deployed-and-restraining-gear.md section 7).
const (
	// metricGearEntries counts area-entry outcomes of ResolveGearEntry ({entry}).
	metricGearEntries = "srd5e.combat.gear.entries"
	// metricGearHoldEnded counts holds that ended, by kind and cause ({hold}).
	metricGearHoldEnded = "srd5e.combat.gear.hold_ended"
	// metricGearHoldsInForce records the size of the GearHolds a result carries ({hold}).
	metricGearHoldsInForce = "srd5e.combat.gear.holds_in_force"

	// attrGearKind is the snake_case DeployedGearKind or HoldKind a gear signal is about.
	attrGearKind = "srd5e.combat.gear.kind"
	// attrGearEntry is a valueGearEntry* value.
	attrGearEntry = "srd5e.combat.gear.entry"
	// attrGearEndCause is the snake_case HoldEndCause.
	attrGearEndCause = "srd5e.combat.gear.end_cause"
)

// Values of srd5e.combat.gear.entry.
const (
	// valueGearEntrySaveFailed is the value of srd5e.combat.gear.entry for a failed save.
	valueGearEntrySaveFailed = "save_failed"
	// valueGearEntrySavePassed is the value of srd5e.combat.gear.entry for a passed save.
	valueGearEntrySavePassed = "save_passed"
	// valueGearEntryDamaged is the value of srd5e.combat.gear.entry for damage with no save.
	valueGearEntryDamaged = "damaged"
	// valueGearEntryAlreadyTriggered is the value of srd5e.combat.gear.entry for a repeated entry on one turn.
	valueGearEntryAlreadyTriggered = "already_triggered"
	// valueGearEntryNotActive is the value of srd5e.combat.gear.entry for gear that is not in force.
	valueGearEntryNotActive = "not_active"
	// valueGearEntryNotApplicable is the value of srd5e.combat.gear.entry for a trigger the gear ignores.
	valueGearEntryNotApplicable = "not_applicable"
)

// Values of srd5e.combat.result for the gear use cases; the refusals use the String of
// GearRejection, the invalid results the String of GearProblem, and applied, failed,
// damaged and held are shared with other use cases.
const (
	// valueResultPlaced is the value of srd5e.combat.result when gear was placed.
	valueResultPlaced = "placed"
	// valueResultSaved is the value of srd5e.combat.result when the creature saved against gear.
	valueResultSaved = "saved"
	// valueResultAutoSucceeded is the value of srd5e.combat.result when a Net's target is too big to be caught.
	valueResultAutoSucceeded = "auto_succeeded"
	// valueResultBound is the value of srd5e.combat.result when a creature was bound.
	valueResultBound = "bound"
	// valueResultReleased is the value of srd5e.combat.result when a hold ended.
	valueResultReleased = "released"
)

// Operations of the containment use cases: the snake_case value of srd5e.combat.operation and the matching
// span srd5e.combat.<operation>.
const (
	// valueOperationEnterContainer is the value of srd5e.combat.operation for EnterContainer.
	valueOperationEnterContainer = "enter_container"
	// spanEnterContainer is the span of EnterContainer.
	spanEnterContainer = "srd5e.combat.enter_container"
	// valueOperationExitContainer is the value of srd5e.combat.operation for ExitContainer.
	valueOperationExitContainer = "exit_container"
	// spanExitContainer is the span of ExitContainer.
	spanExitContainer = "srd5e.combat.exit_container"
	// valueOperationAdvanceContainerTurn is the value of srd5e.combat.operation for AdvanceContainerTurn.
	valueOperationAdvanceContainerTurn = "advance_container_turn"
	// spanAdvanceContainerTurn is the span of AdvanceContainerTurn.
	spanAdvanceContainerTurn = "srd5e.combat.advance_container_turn"
	// valueOperationResolveRegurgitation is the value of srd5e.combat.operation for ResolveRegurgitation.
	valueOperationResolveRegurgitation = "resolve_regurgitation"
	// spanResolveRegurgitation is the span of ResolveRegurgitation.
	spanResolveRegurgitation = "srd5e.combat.resolve_regurgitation"
	// valueOperationReleaseOccupants is the value of srd5e.combat.operation for ReleaseOccupants.
	valueOperationReleaseOccupants = "release_occupants"
	// spanReleaseOccupants is the span of ReleaseOccupants.
	spanReleaseOccupants = "srd5e.combat.release_occupants"
)

// Signals of the containment use cases (monster-engulf-and-swallow.md section 7).
const (
	// metricContainmentEnded counts containments that ended, by reason ({containment}).
	metricContainmentEnded = "srd5e.combat.containment.ended"
	// metricContainmentOccupants records the number of records of the Containments a result carries ({containment}).
	metricContainmentOccupants = "srd5e.combat.containment.occupants"
	// attrContainmentEndReason is the snake_case ContainmentLeaveCause.
	attrContainmentEndReason = "srd5e.combat.containment.end_reason"
	// msgContainmentOperation is the FINE log message of a returned containment call.
	msgContainmentOperation = "containment operation"
)

// Values of srd5e.combat.result for the containment use cases; the refusals use the String of
// ContainmentRejection, the invalid results the String of ContainmentProblem, and held, advanced and
// released are shared with other use cases.
const (
	// valueResultContained is the value of srd5e.combat.result when a creature is now held.
	valueResultContained = "contained"
	// valueResultResisted is the value of srd5e.combat.result when the entry save succeeded.
	valueResultResisted = "resisted"
	// valueResultDamagedOnly is the value of srd5e.combat.result when the entry damaged a creature it could not hold.
	valueResultDamagedOnly = "damaged_only"
	// valueResultExited is the value of srd5e.combat.result when a held creature got out.
	valueResultExited = "exited"
	// valueResultRegurgitated is the value of srd5e.combat.result when a container released everything it held.
	valueResultRegurgitated = "regurgitated"
	// valueResultNotDue is the value of srd5e.combat.result when no regurgitation save was due.
	valueResultNotDue = "not_due"
)

// Operations of the initiative, turn and action use cases: the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationRollInitiative is the value of srd5e.combat.operation for RollInitiative.
	valueOperationRollInitiative = "roll_initiative"
	// spanRollInitiative is the span of RollInitiative.
	spanRollInitiative = "srd5e.combat.roll_initiative"
	// valueOperationEstablishInitiativeOrder is the value of srd5e.combat.operation for EstablishInitiativeOrder.
	valueOperationEstablishInitiativeOrder = "establish_initiative_order"
	// spanEstablishInitiativeOrder is the span of EstablishInitiativeOrder.
	spanEstablishInitiativeOrder = "srd5e.combat.establish_initiative_order"
	// valueOperationAdvanceTrackerTurn is the value of srd5e.combat.operation for AdvanceTrackerTurn.
	valueOperationAdvanceTrackerTurn = "advance_tracker_turn"
	// spanAdvanceTrackerTurn is the span of AdvanceTrackerTurn.
	spanAdvanceTrackerTurn = "srd5e.combat.advance_tracker_turn"
	// valueOperationUpdateMovementSpeeds is the value of srd5e.combat.operation for UpdateMovementSpeeds.
	valueOperationUpdateMovementSpeeds = "update_movement_speeds"
	// spanUpdateMovementSpeeds is the span of UpdateMovementSpeeds.
	spanUpdateMovementSpeeds = "srd5e.combat.update_movement_speeds"
	// valueOperationTakeAction is the value of srd5e.combat.operation for TakeAction.
	valueOperationTakeAction = "take_action"
	// spanTakeAction is the span of TakeAction.
	spanTakeAction = "srd5e.combat.take_action"
	// valueOperationTakeBonusAction is the value of srd5e.combat.operation for TakeBonusAction.
	valueOperationTakeBonusAction = "take_bonus_action"
	// spanTakeBonusAction is the span of TakeBonusAction.
	spanTakeBonusAction = "srd5e.combat.take_bonus_action"
	// valueOperationTakeReaction is the value of srd5e.combat.operation for TakeReaction.
	valueOperationTakeReaction = "take_reaction"
	// spanTakeReaction is the span of TakeReaction.
	spanTakeReaction = "srd5e.combat.take_reaction"
	// valueOperationMove is the value of srd5e.combat.operation for Move.
	valueOperationMove = "move"
	// spanMove is the span of Move.
	spanMove = "srd5e.combat.move"
	// valueOperationInteractWithObject is the value of srd5e.combat.operation for InteractWithObject.
	valueOperationInteractWithObject = "interact_with_object"
	// spanInteractWithObject is the span of InteractWithObject.
	spanInteractWithObject = "srd5e.combat.interact_with_object"
	// valueOperationChangePosture is the value of srd5e.combat.operation for ChangePosture.
	valueOperationChangePosture = "change_posture"
	// spanChangePosture is the span of ChangePosture.
	spanChangePosture = "srd5e.combat.change_posture"
	// valueOperationGrantAdditionalAction is the value of srd5e.combat.operation for GrantAdditionalAction.
	valueOperationGrantAdditionalAction = "grant_additional_action"
	// spanGrantAdditionalAction is the span of GrantAdditionalAction.
	spanGrantAdditionalAction = "srd5e.combat.grant_additional_action"
	// valueOperationRecordFeatUse is the value of srd5e.combat.operation for RecordFeatUse.
	valueOperationRecordFeatUse = "record_feat_use"
	// spanRecordFeatUse is the span of RecordFeatUse.
	spanRecordFeatUse = "srd5e.combat.record_feat_use"
	// valueOperationSwapInitiative is the value of srd5e.combat.operation for SwapInitiative.
	valueOperationSwapInitiative = "swap_initiative"
	// spanSwapInitiative is the span of SwapInitiative.
	spanSwapInitiative = "srd5e.combat.swap_initiative"
)

// Signals of the initiative and turn use cases (initiative-and-actions.md section 7).
// Turns, rounds and actions are counted by srd5e.combat.events with the event kinds, so
// no per-turn counter exists.
const (
	// metricInitiativeCounts counts the Initiative counts RollInitiative produced ({initiative_count}).
	metricInitiativeCounts = "srd5e.combat.initiative_counts"
	// metricInitiativeParticipants records the combatant ids of one RollInitiative call ({combatant}).
	metricInitiativeParticipants = "srd5e.combat.initiative_participants"
	// metricInitiativeTies counts tied groups and unusable tie-breaks by outcome ({tie}).
	metricInitiativeTies = "srd5e.combat.initiative_ties"
	// metricTurnMovementSpent records the movement of each accepted move, stand-up or readied move, one sample per mode ([ft_i]).
	metricTurnMovementSpent = "srd5e.combat.turn_movement_spent"
	// attrInitiativeMethod is the snake_case InitiativeMethod: rolled or score.
	attrInitiativeMethod = "srd5e.combat.initiative_method"
	// attrTieOutcome is resolved, unresolved or rejected.
	attrTieOutcome = "srd5e.combat.tie_outcome"
	// attrMovementMode is the snake_case MovementMode, or posture for a stand-up.
	attrMovementMode = "srd5e.combat.movement_mode"
	// valueMovementModePosture is the value of srd5e.combat.movement_mode for a stand-up.
	valueMovementModePosture = "posture"
)

// Values of srd5e.combat.result for the initiative and turn use cases; RollInitiative uses
// valueResultRolled, the turn refusals the String of their TurnRejection and an accepted
// tracker operation valueResultApplied.
const (
	// valueResultEstablished is the value of srd5e.combat.result when an order was established.
	valueResultEstablished = "established"
	// valueResultTiesUnresolved is the value of srd5e.combat.result when tied groups have no tie-break.
	valueResultTiesUnresolved = "ties_unresolved"
	// valueResultTieBreakRejected is the value of srd5e.combat.result when a tie-break was unusable.
	valueResultTieBreakRejected = "tie_break_rejected"
)

// Operations of the item-use use cases: the snake_case value of srd5e.combat.operation and the matching
// span srd5e.combat.<operation>.
const (
	// valueOperationResolveThrownItem is the value of srd5e.combat.operation for ResolveThrownItem.
	valueOperationResolveThrownItem = "resolve_thrown_item"
	// spanResolveThrownItem is the span of ResolveThrownItem.
	spanResolveThrownItem = "srd5e.combat.resolve_thrown_item"
	// valueOperationUsePotionOfHealing is the value of srd5e.combat.operation for UsePotionOfHealing.
	valueOperationUsePotionOfHealing = "use_potion_of_healing"
	// spanUsePotionOfHealing is the span of UsePotionOfHealing.
	spanUsePotionOfHealing = "srd5e.combat.use_potion_of_healing"
)

// Signals of the item-use use cases (item-use-effects.md section 7). The one extra attribute is
// recorded on the span and the operations counter only.
const (
	// attrItemUseItem is the item of an item-use call: the String of a ThrownItemKind, or potion_of_healing.
	attrItemUseItem = "srd5e.combat.item_use.item"
	// valueItemUsePotionOfHealing is the value of srd5e.combat.item_use.item for UsePotionOfHealing.
	valueItemUsePotionOfHealing = "potion_of_healing"
)

// Values of srd5e.combat.result for the item-use use cases; the refusals use the String of
// ThrownNotPossibleReason or PotionNotPossibleReason, and damaged, saved, no_effect and healed are shared
// with other use cases.
const (
	// valueResultDamagedAndBurning is the value of srd5e.combat.result when a thrown item damaged and ignited the target.
	valueResultDamagedAndBurning = "damaged_and_burning"
	// valueResultCoveredInOil is the value of srd5e.combat.result when thrown Oil covered the target.
	valueResultCoveredInOil = "covered_in_oil"
	// valueResultRestrained is the value of srd5e.combat.result when a thrown Net restrained the target.
	valueResultRestrained = "restrained"
)

// Operations of the monster limited-use use case: the snake_case value of srd5e.combat.operation and
// the matching span srd5e.combat.<operation>.
const (
	// valueOperationApplyMonsterTurnStart is the value of srd5e.combat.operation for ApplyMonsterTurnStart.
	valueOperationApplyMonsterTurnStart = "apply_monster_turn_start"
	// spanApplyMonsterTurnStart is the span of ApplyMonsterTurnStart.
	spanApplyMonsterTurnStart = "srd5e.combat.apply_monster_turn_start"
)

// Signals of ApplyMonsterTurnStart (monster-limited-use.md section 7). The domain files are silent.
const (
	// metricMonsterRechargeRolls counts the Recharge rolls of a returned ApplyMonsterTurnStart result, one per roll ({roll}).
	metricMonsterRechargeRolls = "srd5e.combat.monster_recharge_rolls"
	// attrRechargeOutcome is regained or not_regained, the outcome of a Recharge roll.
	attrRechargeOutcome = "srd5e.combat.recharge_outcome"
	// valueRechargeRegained is the value of srd5e.combat.recharge_outcome for a roll inside the printed range.
	valueRechargeRegained = "regained"
	// valueRechargeNotRegained is the value of srd5e.combat.recharge_outcome for a roll outside the printed range.
	valueRechargeNotRegained = "not_regained"
)

// Values of srd5e.combat.result for ApplyMonsterTurnStart; all are applied.
const (
	// valueResultTurnStartRecharged is the value of srd5e.combat.result when at least one Recharge roll regained its part.
	valueResultTurnStartRecharged = "turn_start_recharged"
	// valueResultTurnStartNotRecharged is the value of srd5e.combat.result when Recharge rolls were made and none regained.
	valueResultTurnStartNotRecharged = "turn_start_not_recharged"
	// valueResultTurnStartNoRolls is the value of srd5e.combat.result when no Recharge roll was needed.
	valueResultTurnStartNoRolls = "turn_start_no_rolls"
)

// Operations of the monster setup use case: the snake_case value of srd5e.combat.operation and the
// matching span srd5e.combat.<operation>.
const (
	// valueOperationSetUpMonsterCombatant is the value of srd5e.combat.operation for SetUpMonsterCombatant.
	valueOperationSetUpMonsterCombatant = "set_up_monster_combatant"
	// spanSetUpMonsterCombatant is the span of SetUpMonsterCombatant.
	spanSetUpMonsterCombatant = "srd5e.combat.set_up_monster_combatant"
)

// Signals of SetUpMonsterCombatant (monster-combatant-setup.md section 7). The domain files are silent.
const (
	// metricMonsterHitPointsClamped counts the rolled Hit Point totals raised to 1 ({roll}); it has no attributes.
	metricMonsterHitPointsClamped = "srd5e.combat.monster_hit_points_clamped"
	// attrMonsterHitPointsMethod is average or rolled, how the Hit Points of an applied setup were decided (span and operations counter only).
	attrMonsterHitPointsMethod = "srd5e.combat.monster_hit_points_method"
	// valueMonsterHitPointsMethodAverage is the value of srd5e.combat.monster_hit_points_method for the printed average.
	valueMonsterHitPointsMethodAverage = "average"
	// valueMonsterHitPointsMethodRolled is the value of srd5e.combat.monster_hit_points_method for a roll of the Hit Point Dice.
	valueMonsterHitPointsMethodRolled = "rolled"
)

// Values of srd5e.combat.result for SetUpMonsterCombatant; the first two are applied, the others invalid.
const (
	// valueResultSetUpAverage is the value of srd5e.combat.result for a setup with the printed average Hit Points.
	valueResultSetUpAverage = "set_up_average"
	// valueResultSetUpRolled is the value of srd5e.combat.result for a setup with rolled Hit Points.
	valueResultSetUpRolled = "set_up_rolled"
	// valueResultSizeChoiceRequired is the value of srd5e.combat.result when a monster with several size options was given none.
	valueResultSizeChoiceRequired = "size_choice_required"
	// valueResultSizeNotOffered is the value of srd5e.combat.result when the chosen size is not among the monster's options.
	valueResultSizeNotOffered = "size_not_offered"
	// valueResultContentNumberOutOfRange is the value of srd5e.combat.result when a number of the stat block is outside the range combat relies on (plan P13).
	valueResultContentNumberOutOfRange = "content_number_out_of_range"
)

// Operations of the spell effects use case: the snake_case value of srd5e.combat.operation and the
// matching span srd5e.combat.<operation>.
const (
	// valueOperationResolveSpellEffects is the value of srd5e.combat.operation for ResolveSpellEffects.
	valueOperationResolveSpellEffects = "resolve_spell_effects"
	// spanResolveSpellEffects is the span of ResolveSpellEffects.
	spanResolveSpellEffects = "srd5e.combat.resolve_spell_effects"
)

// Signals of ResolveSpellEffects (spell-effects.md section 7). The domain files are silent.
const (
	// metricSpellEffectsResolutions counts the calls that were not rejected, by coverage and level ({resolution}).
	metricSpellEffectsResolutions = "srd5e.combat.spell_effects.resolutions"
	// metricSpellEffectsRolls counts the attack rolls and saving throws of resolved calls ({roll}).
	metricSpellEffectsRolls = "srd5e.combat.spell_effects.rolls"
	// metricSpellEffectsTargets records the valid targets of a resolved call ({target}).
	metricSpellEffectsTargets = "srd5e.combat.spell_effects.targets"

	// attrSpellEffectsCoverage is the coverage of the spell's model: a valueSpellCoverage* value (span, operations counter and resolutions).
	attrSpellEffectsCoverage = "srd5e.combat.spell_effects.coverage"
	// attrSpellEffectsRejection is the snake_case first SpellEffectsRejectionKind of a rejection (span and operations counter only).
	attrSpellEffectsRejection = "srd5e.combat.spell_effects.rejection"
	// attrSpellEffectsLevel is the effective level: cantrip or 1 to 9 (resolutions only).
	attrSpellEffectsLevel = "srd5e.combat.spell_effects.level"
	// attrSpellEffectsRoll is a valueSpellRoll* value (rolls only).
	attrSpellEffectsRoll = "srd5e.combat.spell_effects.roll"
)

// Values of srd5e.combat.spell_effects.coverage, .level and .roll.
const (
	// valueSpellCoverageComplete is the coverage of a model that covers the whole description.
	valueSpellCoverageComplete = "complete"
	// valueSpellCoveragePartial is the coverage of a model that leaves part of the description to the GM.
	valueSpellCoveragePartial = "partial"
	// valueSpellCoverageTextOnly is the coverage of a spell with no model.
	valueSpellCoverageTextOnly = "text_only"
	// valueSpellLevelCantrip is the level attribute of a level 0 spell.
	valueSpellLevelCantrip = "cantrip"
	// valueSpellRollAttack is a spell attack roll.
	valueSpellRollAttack = "attack"
	// valueSpellRollSave is a rolled saving throw.
	valueSpellRollSave = "save"
	// valueSpellRollSaveForced is a saving throw failed without a die.
	valueSpellRollSaveForced = "save_forced"
)

// Values of srd5e.combat.result for ResolveSpellEffects (spell-effects.md FR-081).
const (
	// valueResultSpellResolved is the value of srd5e.combat.result for a plan (applied).
	valueResultSpellResolved = "resolved"
	// valueResultSpellTextOnly is the value of srd5e.combat.result for a spell with no model (applied).
	valueResultSpellTextOnly = "text_only"
	// valueResultSpellRejectedInvalid is the value of srd5e.combat.result for host input the engine cannot use (invalid).
	valueResultSpellRejectedInvalid = "rejected_invalid"
	// valueResultSpellRejectedRefused is the value of srd5e.combat.result for legal input the spell's rules do not allow (refused).
	valueResultSpellRejectedRefused = "rejected_refused"
)

// Operations of the stat-block damage use case: the snake_case value of srd5e.combat.operation and
// the matching span srd5e.combat.<operation>.
const (
	// valueOperationRollStatBlockDamage is the value of srd5e.combat.operation for RollStatBlockDamage.
	valueOperationRollStatBlockDamage = "roll_stat_block_damage"
	// spanRollStatBlockDamage is the span of RollStatBlockDamage.
	spanRollStatBlockDamage = "srd5e.combat.roll_stat_block_damage"
)

// Signals of RollStatBlockDamage (monster-stat-block-actions.md section 7). The domain files are silent.
const (
	// attrDamageNotation is number, dice or mixed, how the damage instances of an applied call were rolled (span and operations counter only).
	attrDamageNotation = "srd5e.combat.damage_notation"
	// valueDamageNotationNumber is the value of srd5e.combat.damage_notation when every instance used its printed number.
	valueDamageNotationNumber = "number"
	// valueDamageNotationDice is the value of srd5e.combat.damage_notation when every instance rolled its dice.
	valueDamageNotationDice = "dice"
	// valueDamageNotationMixed is the value of srd5e.combat.damage_notation when some instances used the number and some rolled.
	valueDamageNotationMixed = "mixed"
)

// Values of srd5e.combat.result for RollStatBlockDamage; the first two are applied. A damage instance with a
// number outside the range combat relies on is invalid, with the shared content_number_out_of_range.
const (
	// valueResultStatBlockDamageRolled is the value of srd5e.combat.result when at least one die was rolled.
	valueResultStatBlockDamageRolled = "stat_block_damage_rolled"
	// valueResultStatBlockDamageFixed is the value of srd5e.combat.result when every instance used its printed number.
	valueResultStatBlockDamageFixed = "stat_block_damage_fixed"
)

// Operations of the encounter use cases (encounter-aggregate.md table 4.10.1): the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationStartEncounter is the value of srd5e.combat.operation for StartEncounter.
	valueOperationStartEncounter = "start_encounter"
	// spanStartEncounter is the span of StartEncounter.
	spanStartEncounter = "srd5e.combat.start_encounter"
	// valueOperationAdvanceEncounterTurn is the value of srd5e.combat.operation for AdvanceEncounterTurn.
	valueOperationAdvanceEncounterTurn = "advance_encounter_turn"
	// spanAdvanceEncounterTurn is the span of AdvanceEncounterTurn.
	spanAdvanceEncounterTurn = "srd5e.combat.advance_encounter_turn"
	// valueOperationApplyDamageToCombatant is the value of srd5e.combat.operation for ApplyDamageToCombatant.
	valueOperationApplyDamageToCombatant = "apply_damage_to_combatant"
	// spanApplyDamageToCombatant is the span of ApplyDamageToCombatant.
	spanApplyDamageToCombatant = "srd5e.combat.apply_damage_to_combatant"
	// valueOperationApplyHealingToCombatant is the value of srd5e.combat.operation for ApplyHealingToCombatant.
	valueOperationApplyHealingToCombatant = "apply_healing_to_combatant"
	// spanApplyHealingToCombatant is the span of ApplyHealingToCombatant.
	spanApplyHealingToCombatant = "srd5e.combat.apply_healing_to_combatant"
	// valueOperationApplyConditionToCombatant is the value of srd5e.combat.operation for ApplyConditionToCombatant.
	valueOperationApplyConditionToCombatant = "apply_condition_to_combatant"
	// spanApplyConditionToCombatant is the span of ApplyConditionToCombatant.
	spanApplyConditionToCombatant = "srd5e.combat.apply_condition_to_combatant"
	// valueOperationRemoveConditionFromCombatant is the value of srd5e.combat.operation for RemoveConditionFromCombatant.
	valueOperationRemoveConditionFromCombatant = "remove_condition_from_combatant"
	// spanRemoveConditionFromCombatant is the span of RemoveConditionFromCombatant.
	spanRemoveConditionFromCombatant = "srd5e.combat.remove_condition_from_combatant"
	// valueOperationReduceCombatantExhaustion is the value of srd5e.combat.operation for ReduceCombatantExhaustion.
	valueOperationReduceCombatantExhaustion = "reduce_combatant_exhaustion"
	// spanReduceCombatantExhaustion is the span of ReduceCombatantExhaustion.
	spanReduceCombatantExhaustion = "srd5e.combat.reduce_combatant_exhaustion"
	// valueOperationApplyDeathSavingThrowToCombatant is the value of srd5e.combat.operation for ApplyDeathSavingThrowToCombatant.
	valueOperationApplyDeathSavingThrowToCombatant = "apply_death_saving_throw_to_combatant"
	// spanApplyDeathSavingThrowToCombatant is the span of ApplyDeathSavingThrowToCombatant.
	spanApplyDeathSavingThrowToCombatant = "srd5e.combat.apply_death_saving_throw_to_combatant"
	// valueOperationStabilizeCombatant is the value of srd5e.combat.operation for StabilizeCombatant.
	valueOperationStabilizeCombatant = "stabilize_combatant"
	// spanStabilizeCombatant is the span of StabilizeCombatant.
	spanStabilizeCombatant = "srd5e.combat.stabilize_combatant"
	// valueOperationRecoverStableCombatant is the value of srd5e.combat.operation for RecoverStableCombatant.
	valueOperationRecoverStableCombatant = "recover_stable_combatant"
	// spanRecoverStableCombatant is the span of RecoverStableCombatant.
	spanRecoverStableCombatant = "srd5e.combat.recover_stable_combatant"
	// valueOperationGiveCombatantTemporaryHitPoints is the value of srd5e.combat.operation for GiveCombatantTemporaryHitPoints.
	valueOperationGiveCombatantTemporaryHitPoints = "give_combatant_temporary_hit_points"
	// spanGiveCombatantTemporaryHitPoints is the span of GiveCombatantTemporaryHitPoints.
	spanGiveCombatantTemporaryHitPoints = "srd5e.combat.give_combatant_temporary_hit_points"
	// valueOperationReduceCombatantHitPointMaximum is the value of srd5e.combat.operation for ReduceCombatantHitPointMaximum.
	valueOperationReduceCombatantHitPointMaximum = "reduce_combatant_hit_point_maximum"
	// spanReduceCombatantHitPointMaximum is the span of ReduceCombatantHitPointMaximum.
	spanReduceCombatantHitPointMaximum = "srd5e.combat.reduce_combatant_hit_point_maximum"
	// valueOperationEndEncounter is the value of srd5e.combat.operation for EndEncounter.
	valueOperationEndEncounter = "end_encounter"
	// spanEndEncounter is the span of EndEncounter.
	spanEndEncounter = "srd5e.combat.end_encounter"
	// valueOperationCheckCombatantGrapple is the value of srd5e.combat.operation for CheckCombatantGrapple.
	valueOperationCheckCombatantGrapple = "check_combatant_grapple"
	// spanCheckCombatantGrapple is the span of CheckCombatantGrapple.
	spanCheckCombatantGrapple = "srd5e.combat.check_combatant_grapple"
	// valueOperationReleaseCombatantGrapple is the value of srd5e.combat.operation for ReleaseCombatantGrapple.
	valueOperationReleaseCombatantGrapple = "release_combatant_grapple"
	// spanReleaseCombatantGrapple is the span of ReleaseCombatantGrapple.
	spanReleaseCombatantGrapple = "srd5e.combat.release_combatant_grapple"
	// valueOperationEscapeCombatantGrapple is the value of srd5e.combat.operation for EscapeCombatantGrapple.
	valueOperationEscapeCombatantGrapple = "escape_combatant_grapple"
	// spanEscapeCombatantGrapple is the span of EscapeCombatantGrapple.
	spanEscapeCombatantGrapple = "srd5e.combat.escape_combatant_grapple"
)

// Signals of the encounter use cases (encounter-aggregate.md section 7). The domain files are silent.
// srd5e.combat.deaths and srd5e.combat.death_cause are hit-points-and-damage.md's.
const (
	// metricEncounterSize records the number of combatants of a started encounter ({combatant}).
	metricEncounterSize = "srd5e.combat.encounter_size"
)

// Value of srd5e.combat.result for a start the engine cannot use (invalid). The other results of the encounter use cases
// are the String of an EncounterRejection (encounter-aggregate.md table 4.10.1), applied is valueResultApplied.
const valueResultInvalidEncounter = "invalid_encounter"

// Operations of the active effect use cases (active-effects.md section 7): the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationStartActiveEffect is the value of srd5e.combat.operation for StartActiveEffect.
	valueOperationStartActiveEffect = "start_active_effect"
	// spanStartActiveEffect is the span of StartActiveEffect.
	spanStartActiveEffect = "srd5e.combat.start_active_effect"
	// valueOperationAdvanceActiveEffects is the value of srd5e.combat.operation for AdvanceActiveEffects.
	valueOperationAdvanceActiveEffects = "advance_active_effects"
	// spanAdvanceActiveEffects is the span of AdvanceActiveEffects.
	spanAdvanceActiveEffects = "srd5e.combat.advance_active_effects"
	// valueOperationEndActiveEffects is the value of srd5e.combat.operation for EndActiveEffects.
	valueOperationEndActiveEffects = "end_active_effects"
	// spanEndActiveEffects is the span of EndActiveEffects.
	spanEndActiveEffects = "srd5e.combat.end_active_effects"
	// valueOperationResolveEffectDamage is the value of srd5e.combat.operation for ResolveEffectDamage.
	valueOperationResolveEffectDamage = "resolve_effect_damage"
	// spanResolveEffectDamage is the span of ResolveEffectDamage.
	spanResolveEffectDamage = "srd5e.combat.resolve_effect_damage"
)

// Signals of the active effect use cases (active-effects.md section 7). The domain files are silent.
const (
	// metricActiveEffectsEnded counts the effects that ended, by reason ({effect}).
	metricActiveEffectsEnded = "srd5e.combat.active_effects.ended"
	// metricActiveEffectsSaves counts the repeated, gate and damage saving throws ({save}).
	metricActiveEffectsSaves = "srd5e.combat.active_effects.saves"
	// metricActiveEffectsActive records the effects in the registry after a call ({effect}).
	metricActiveEffectsActive = "srd5e.combat.active_effects.active"

	// attrActiveEffectsRejection is the snake_case first reason of a rejection (span and operations counter only).
	attrActiveEffectsRejection = "srd5e.combat.active_effects.rejection"
	// attrActiveEffectsEndReason is the snake_case EffectEndReason (the ended counter only).
	attrActiveEffectsEndReason = "srd5e.combat.active_effects.end_reason"
	// attrActiveEffectsSaveKind is the EffectSaveKind name: repeat, stage_gate or damage (the saves counter only).
	attrActiveEffectsSaveKind = "srd5e.combat.active_effects.save_kind"
)

// Values of srd5e.combat.result for the active effect use cases (active-effects.md FR-058).
const (
	// valueResultEffectStarted is the value of srd5e.combat.result for a started effect (applied).
	valueResultEffectStarted = "started"
	// valueResultEffectsAdvanced is the value of srd5e.combat.result for an advanced registry (applied).
	valueResultEffectsAdvanced = "advanced"
	// valueResultEffectsEnded is the value of srd5e.combat.result for ended effects (applied).
	valueResultEffectsEnded = "ended"
	// valueResultEffectMoved is the value of srd5e.combat.result for a moved effect (applied).
	valueResultEffectMoved = "moved"
	// valueResultEffectsNothingEnded is the value of srd5e.combat.result for a cause that ended nothing (refused).
	valueResultEffectsNothingEnded = "nothing_ended"
	// valueResultEffectDamageResolved is the value of srd5e.combat.result for resolved effect damage (applied).
	valueResultEffectDamageResolved = "effect_damage_resolved"
	// valueResultEffectsRejectedInvalid is the value of srd5e.combat.result for input the engine cannot use (invalid).
	valueResultEffectsRejectedInvalid = "rejected_invalid"
	// valueResultEffectsRejectedRefused is the value of srd5e.combat.result for legal input the rules do not allow (refused).
	valueResultEffectsRejectedRefused = "rejected_refused"
)

// Operations of the spell-granted modifier use case (spell-granted-modifiers.md section 7): the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>. The domain files are silent.
const (
	// valueOperationRollSpellBonusDice is the value of srd5e.combat.operation for RollSpellBonusDice.
	valueOperationRollSpellBonusDice = "roll_spell_bonus_dice"
	// spanRollSpellBonusDice is the span of RollSpellBonusDice.
	spanRollSpellBonusDice = "srd5e.combat.roll_spell_bonus_dice"
)

// Values of srd5e.combat.result for RollSpellBonusDice, both applied.
const (
	// valueResultSpellBonusDiceRolled is the value of srd5e.combat.result when at least one die was rolled.
	valueResultSpellBonusDiceRolled = "rolled"
	// valueResultSpellBonusDiceNothingToRoll is the value of srd5e.combat.result when the command held no die.
	valueResultSpellBonusDiceNothingToRoll = "nothing_to_roll"
)

// Values of srd5e.combat.result for the Dispel Magic, Counterspell and Antimagic
// Field use cases (dispel-and-counterspell.md section 7).
const (
	// valueResultCompleted is the value of srd5e.combat.result for a Dispel Magic that ended a spell, and for an applied Counterspell or Antimagic Field verdict.
	valueResultCompleted = "completed"
	// valueResultDispelNothingEnded is the value of srd5e.combat.result for a Dispel Magic that ended nothing.
	valueResultDispelNothingEnded = "dispel_nothing_ended"
	// valueResultDispelInvalidChecks is the value of srd5e.combat.result for a Dispel Magic given checks that do not fit its plan.
	valueResultDispelInvalidChecks = "dispel_invalid_checks"
	// valueResultCounterspellResisted is the value of srd5e.combat.result for a Counterspell the caster's save resisted.
	valueResultCounterspellResisted = "counterspell_resisted"
	// valueResultCounterspellIneligible is the value of srd5e.combat.result for a cast Counterspell cannot counter.
	valueResultCounterspellIneligible = "counterspell_ineligible"
	// valueResultCounterspellInvalidSave is the value of srd5e.combat.result for a Counterspell given a save that does not fit it.
	valueResultCounterspellInvalidSave = "counterspell_invalid_save"
	// valueResultAntimagicBlocked is the value of srd5e.combat.result for an activity an Antimagic Field blocks.
	valueResultAntimagicBlocked = "antimagic_blocked"
)

// Operations of the Dispel Magic, Counterspell and Antimagic Field use cases (dispel-and-counterspell.md section 7): the
// snake_case value of srd5e.combat.operation and the matching span.
const (
	// valueOperationDispelOngoingSpells is the value of srd5e.combat.operation for DispelOngoingSpells.
	valueOperationDispelOngoingSpells = "dispel_ongoing_spells"
	// spanDispelOngoingSpells is the span of DispelOngoingSpells.
	spanDispelOngoingSpells = "srd5e.combat.dispel_ongoing_spells"
	// valueOperationCounterCastSpell is the value of srd5e.combat.operation for CounterCastSpell.
	valueOperationCounterCastSpell = "counter_cast_spell"
	// spanCounterCastSpell is the span of CounterCastSpell.
	spanCounterCastSpell = "srd5e.combat.counter_cast_spell"
	// valueOperationAssessAntimagicActivity is the value of srd5e.combat.operation for AssessAntimagicActivity.
	valueOperationAssessAntimagicActivity = "assess_antimagic_activity"
	// spanAssessAntimagicActivity is the span of AssessAntimagicActivity.
	spanAssessAntimagicActivity = "srd5e.combat.assess_antimagic_activity"
)

// valueResultLimitExceeded is the value of srd5e.combat.result for a call that holds a collection over its cap (plan P10).
// It is invalid, and shared by every use case that caps a collection.
const valueResultLimitExceeded = "limit_exceeded"

// Signals of ResolveSpellOutcomeTable (spell-outcome-tables.md section 7). The pure spell_outcome_*.go files are
// silent.
const (
	// valueOperationResolveSpellOutcomeTable is the value of srd5e.combat.operation for ResolveSpellOutcomeTable.
	valueOperationResolveSpellOutcomeTable = "resolve_spell_outcome_table"
	// spanResolveSpellOutcomeTable is the span of ResolveSpellOutcomeTable.
	spanResolveSpellOutcomeTable = "srd5e.combat.resolve_spell_outcome_table"
	// metricSpellTableRolls counts the rolls on a printed spell table, rerolls included ({roll}).
	metricSpellTableRolls = "srd5e.combat.spell_table_rolls"
	// attrSpellTable is the table rolled: a valueSpellTable* value (the span, the operations counter and spell_table_rolls).
	attrSpellTable = "srd5e.combat.spell_table"
	// attrTableOutcome is the closed outcome of one table roll, the String of the enum of the table (spell_table_rolls
	// only; spell-outcome-tables.md section 7.1).
	attrTableOutcome = "srd5e.combat.table_outcome"
)

// Values of srd5e.combat.result for ResolveSpellOutcomeTable.
const (
	// valueResultSpellTableRolled is the value of srd5e.combat.result for every table that was rolled or crossed (applied).
	valueResultSpellTableRolled = "spell_table_rolled"
	// valueResultSpellTableRefused is the value of srd5e.combat.result for a Reincarnate the rules refuse (refused).
	valueResultSpellTableRefused = "spell_table_refused"
)

// Values of srd5e.combat.spell_table.
const (
	// valueSpellTableTeleport is the value of srd5e.combat.spell_table for Teleport.
	valueSpellTableTeleport = "teleport"
	// valueSpellTableConfusion is the value of srd5e.combat.spell_table for Confusion.
	valueSpellTableConfusion = "confusion"
	// valueSpellTablePrismaticSpray is the value of srd5e.combat.spell_table for Prismatic Spray.
	valueSpellTablePrismaticSpray = "prismatic_spray"
	// valueSpellTablePrismaticWall is the value of srd5e.combat.spell_table for Prismatic Wall.
	valueSpellTablePrismaticWall = "prismatic_wall"
	// valueSpellTableReincarnate is the value of srd5e.combat.spell_table for Reincarnate.
	valueSpellTableReincarnate = "reincarnate"
	// valueSpellTableBlink is the value of srd5e.combat.spell_table for Blink.
	valueSpellTableBlink = "blink"
)

// Operations of the combat action resolution use cases (combat-action-resolution.md FR-099): the snake_case
// value of srd5e.combat.operation and the matching span srd5e.combat.<operation>. The domain files are silent.
const (
	// valueOperationPerformAttack is the value of srd5e.combat.operation for PerformAttack.
	valueOperationPerformAttack = "perform_attack"
	// spanPerformAttack is the span of PerformAttack.
	spanPerformAttack = "srd5e.combat.perform_attack"
	// valueOperationPerformUnarmedStrike is the value of srd5e.combat.operation for PerformUnarmedStrike.
	valueOperationPerformUnarmedStrike = "perform_unarmed_strike"
	// spanPerformUnarmedStrike is the span of PerformUnarmedStrike.
	spanPerformUnarmedStrike = "srd5e.combat.perform_unarmed_strike"
	// valueOperationTakeCombatantAction is the value of srd5e.combat.operation for TakeCombatantAction.
	valueOperationTakeCombatantAction = "take_combatant_action"
	// spanTakeCombatantAction is the span of TakeCombatantAction.
	spanTakeCombatantAction = "srd5e.combat.take_combatant_action"
	// valueOperationMoveCombatant is the value of srd5e.combat.operation for MoveCombatant.
	valueOperationMoveCombatant = "move_combatant"
	// spanMoveCombatant is the span of MoveCombatant.
	spanMoveCombatant = "srd5e.combat.move_combatant"
	// valueOperationAdvanceCombatTurn is the value of srd5e.combat.operation for AdvanceCombatTurn.
	valueOperationAdvanceCombatTurn = "advance_combat_turn"
	// spanAdvanceCombatTurn is the span of AdvanceCombatTurn.
	spanAdvanceCombatTurn = "srd5e.combat.advance_combat_turn"
	// valueOperationRollCombatantDeathSavingThrow is the value of srd5e.combat.operation for RollCombatantDeathSavingThrow.
	valueOperationRollCombatantDeathSavingThrow = "roll_combatant_death_saving_throw"
	// spanRollCombatantDeathSavingThrow is the span of RollCombatantDeathSavingThrow.
	spanRollCombatantDeathSavingThrow = "srd5e.combat.roll_combatant_death_saving_throw"
	// valueOperationAttemptToStabilizeCombatant is the value of srd5e.combat.operation for AttemptToStabilizeCombatant.
	valueOperationAttemptToStabilizeCombatant = "attempt_to_stabilize_combatant"
	// spanAttemptToStabilizeCombatant is the span of AttemptToStabilizeCombatant.
	spanAttemptToStabilizeCombatant = "srd5e.combat.attempt_to_stabilize_combatant"
	// valueOperationDealDamageToCombatant is the value of srd5e.combat.operation for DealDamageToCombatant.
	valueOperationDealDamageToCombatant = "deal_damage_to_combatant"
	// spanDealDamageToCombatant is the span of DealDamageToCombatant.
	spanDealDamageToCombatant = "srd5e.combat.deal_damage_to_combatant"
)

// Values of srd5e.combat.result for the combat action resolution use cases (combat-action-resolution.md FR-100):
// valueResultCompleted for a completed call, the String of an EncounterRejection for an EncounterStateRejection, and
// these for the rest. All but valueResultUnknownCombatant are refused; that one is invalid.
const (
	// valueResultUnknownCombatant is the value of srd5e.combat.result when the host named a combatant the encounter does not hold (invalid).
	valueResultUnknownCombatant = "unknown_combatant"
	// valueResultLedgerRefused is the value of srd5e.combat.result when the ActionLedger refused (refused).
	valueResultLedgerRefused = "ledger_refused"
	// valueResultTurnRefused is the value of srd5e.combat.result when the turn tracker refused (refused).
	valueResultTurnRefused = "turn_refused"
	// valueResultTargetingForbidden is the value of srd5e.combat.result for an attacker Charmed by its target (refused).
	valueResultTargetingForbidden = "targeting_forbidden"
	// valueResultAttackNotPossible is the value of srd5e.combat.result for an attack that cannot be made (refused).
	valueResultAttackNotPossible = "attack_not_possible"
	// valueResultMovementRestricted is the value of srd5e.combat.result for a move the conditions restrict (refused).
	valueResultMovementRestricted = "movement_restricted"
	// valueResultPostureMismatch is the value of srd5e.combat.result for a posture change that does not fit (refused).
	valueResultPostureMismatch = "posture_mismatch"
	// valueResultDeathSavingThrowOutOfTurn is the value of srd5e.combat.result for a Death Saving Throw asked out of turn (refused).
	valueResultDeathSavingThrowOutOfTurn = "death_saving_throw_out_of_turn"
)

// Signals of combat action resolution beyond the shared ones (combat-action-resolution.md FR-103, FR-106).
const (
	// metricConcentrationChecks counts the Concentration checks after damage ({check}); one per ConcentrationChecked event.
	metricConcentrationChecks = "srd5e.combat.concentration_checks"
	// attrConcentrationOutcome is the outcome of one Concentration check: a valueConcentration* value
	// (the counter srd5e.combat.concentration_checks only).
	attrConcentrationOutcome = "srd5e.combat.concentration_outcome"
)

// Values of srd5e.combat.concentration_outcome.
const (
	// valueConcentrationHeld is the value for a Constitution saving throw that kept Concentration.
	valueConcentrationHeld = "held"
	// valueConcentrationBroken is the value for a failed Constitution saving throw (ConcentrationCheckFailed).
	valueConcentrationBroken = "broken"
	// valueConcentrationEndedIncapacitated is the value for Concentration ended because the combatant is Incapacitated.
	valueConcentrationEndedIncapacitated = "ended_incapacitated"
	// valueConcentrationEndedDead is the value for Concentration ended because the combatant died.
	valueConcentrationEndedDead = "ended_dead"
)

// Values of srd5e.combat.result added by the stat block use of combat action resolution (FR-100, FR-164).
const (
	// valueResultStatBlockRefused is the value of srd5e.combat.result for a stat block part that cannot be used (refused).
	valueResultStatBlockRefused = "stat_block_refused"
)

// Operation and span of ResolveStatBlockSave, the spec's ResolveStatBlockSaveEffect (FR-166, FR-170).
const (
	// valueOperationResolveStatBlockSaveEffect is the value of srd5e.combat.operation for ResolveStatBlockSave.
	valueOperationResolveStatBlockSaveEffect = "resolve_stat_block_save_effect"
	// spanResolveStatBlockSaveEffect is the span of ResolveStatBlockSave.
	spanResolveStatBlockSaveEffect = "srd5e.combat.resolve_stat_block_save_effect"
)

// Operations of the use cases that wrap the pure services of the domain-only specs (combat-action-resolution.md FR-170):
// the snake_case value of srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationReadyCombatantSpell is the value of srd5e.combat.operation for the use case that wraps the ready_combatant_spell service.
	valueOperationReadyCombatantSpell = "ready_combatant_spell"
	// spanReadyCombatantSpell is its span.
	spanReadyCombatantSpell = "srd5e.combat.ready_combatant_spell"
	// valueOperationReleaseCombatantSpell is the value of srd5e.combat.operation for the use case that wraps the release_combatant_spell service.
	valueOperationReleaseCombatantSpell = "release_combatant_spell"
	// spanReleaseCombatantSpell is its span.
	spanReleaseCombatantSpell = "srd5e.combat.release_combatant_spell"
	// valueOperationDissipateCombatantSpell is the value of srd5e.combat.operation for the use case that wraps the dissipate_combatant_spell service.
	valueOperationDissipateCombatantSpell = "dissipate_combatant_spell"
	// spanDissipateCombatantSpell is its span.
	spanDissipateCombatantSpell = "srd5e.combat.dissipate_combatant_spell"
	// valueOperationChangeTelepathicContact is the value of srd5e.combat.operation for the use case that wraps the change_telepathic_contact service.
	valueOperationChangeTelepathicContact = "change_telepathic_contact"
	// spanChangeTelepathicContact is its span.
	spanChangeTelepathicContact = "srd5e.combat.change_telepathic_contact"
	// valueOperationCreatePersistentZone is the value of srd5e.combat.operation for the use case that wraps the create_persistent_zone service.
	valueOperationCreatePersistentZone = "create_persistent_zone"
	// spanCreatePersistentZone is its span.
	spanCreatePersistentZone = "srd5e.combat.create_persistent_zone"
	// valueOperationAssessZoneTrigger is the value of srd5e.combat.operation for the use case that wraps the assess_zone_trigger service.
	valueOperationAssessZoneTrigger = "assess_zone_trigger"
	// spanAssessZoneTrigger is its span.
	spanAssessZoneTrigger = "srd5e.combat.assess_zone_trigger"
	// valueOperationMovePersistentZone is the value of srd5e.combat.operation for the use case that wraps the move_persistent_zone service.
	valueOperationMovePersistentZone = "move_persistent_zone"
	// spanMovePersistentZone is its span.
	spanMovePersistentZone = "srd5e.combat.move_persistent_zone"
	// valueOperationEndPersistentZone is the value of srd5e.combat.operation for the use case that wraps the end_persistent_zone service.
	valueOperationEndPersistentZone = "end_persistent_zone"
	// spanEndPersistentZone is its span.
	spanEndPersistentZone = "srd5e.combat.end_persistent_zone"
	// valueOperationBeginShapeShift is the value of srd5e.combat.operation for the use case that wraps the begin_shape_shift service.
	valueOperationBeginShapeShift = "begin_shape_shift"
	// spanBeginShapeShift is its span.
	spanBeginShapeShift = "srd5e.combat.begin_shape_shift"
	// valueOperationChangeShapeShiftForm is the value of srd5e.combat.operation for the use case that wraps the change_shape_shift_form service.
	valueOperationChangeShapeShiftForm = "change_shape_shift_form"
	// spanChangeShapeShiftForm is its span.
	spanChangeShapeShiftForm = "srd5e.combat.change_shape_shift_form"
	// valueOperationAssessShapeShift is the value of srd5e.combat.operation for the use case that wraps the assess_shape_shift service.
	valueOperationAssessShapeShift = "assess_shape_shift"
	// spanAssessShapeShift is its span.
	spanAssessShapeShift = "srd5e.combat.assess_shape_shift"
	// valueOperationEndShapeShift is the value of srd5e.combat.operation for the use case that wraps the end_shape_shift service.
	valueOperationEndShapeShift = "end_shape_shift"
	// spanEndShapeShift is its span.
	spanEndShapeShift = "srd5e.combat.end_shape_shift"
	// valueOperationReviveBySpell is the value of srd5e.combat.operation for the use case that wraps the revive_by_spell service.
	valueOperationReviveBySpell = "revive_by_spell"
	// spanReviveBySpell is its span.
	spanReviveBySpell = "srd5e.combat.revive_by_spell"
	// valueOperationReviveCombatant is the value of srd5e.combat.operation for the use case that wraps the revive service (returning-to-life.md FR-016).
	valueOperationReviveCombatant = "revive_combatant"
	// spanReviveCombatant is its span.
	spanReviveCombatant = "srd5e.combat.revive_combatant"
	// valueOperationIncreaseMaximum is the value of srd5e.combat.operation for the use case that wraps the increase_maximum service.
	valueOperationIncreaseMaximum = "increase_maximum"
	// spanIncreaseMaximum is its span.
	spanIncreaseMaximum = "srd5e.combat.increase_maximum"
	// valueOperationEndMaximumIncrease is the value of srd5e.combat.operation for the use case that wraps the end_maximum_increase service.
	valueOperationEndMaximumIncrease = "end_maximum_increase"
	// spanEndMaximumIncrease is its span.
	spanEndMaximumIncrease = "srd5e.combat.end_maximum_increase"
	// valueOperationRestoreMaximum is the value of srd5e.combat.operation for the use case that wraps the restore_maximum service.
	valueOperationRestoreMaximum = "restore_maximum"
	// spanRestoreMaximum is its span.
	spanRestoreMaximum = "srd5e.combat.restore_maximum"
	// valueOperationLesserRestoration is the value of srd5e.combat.operation for the use case that wraps the lesser_restoration service.
	valueOperationLesserRestoration = "lesser_restoration"
	// spanLesserRestoration is its span.
	spanLesserRestoration = "srd5e.combat.lesser_restoration"
	// valueOperationGreaterRestoration is the value of srd5e.combat.operation for the use case that wraps the greater_restoration service.
	valueOperationGreaterRestoration = "greater_restoration"
	// spanGreaterRestoration is its span.
	spanGreaterRestoration = "srd5e.combat.greater_restoration"
	// valueOperationRestorativeHealing is the value of srd5e.combat.operation for the use case that wraps the restorative_healing service.
	valueOperationRestorativeHealing = "restorative_healing"
	// spanRestorativeHealing is its span.
	spanRestorativeHealing = "srd5e.combat.restorative_healing"
	// valueOperationGuardWithDeathWard is the value of srd5e.combat.operation for the use case that wraps the guard_with_death_ward service.
	valueOperationGuardWithDeathWard = "guard_with_death_ward"
	// spanGuardWithDeathWard is its span.
	spanGuardWithDeathWard = "srd5e.combat.guard_with_death_ward"
)

// Values of srd5e.combat.result of the wrapped use cases (FR-100, FR-169, FR-170, and the section 7 tables of the owners).
// The applied and refused names of the owners that are the String of a rejection are not constants here.
const (
	// valueResultReadyNotAnAction is the value of srd5e.combat.result for ready_not_an_action (refused).
	valueResultReadyNotAnAction = "ready_not_an_action"
	// valueResultReleaseNoHeldSpell is the value of srd5e.combat.result for release_no_held_spell (refused).
	valueResultReleaseNoHeldSpell = "release_no_held_spell"
	// valueResultReleaseNotAReadiedReaction is the value of srd5e.combat.result for release_not_a_readied_reaction (refused).
	valueResultReleaseNotAReadiedReaction = "release_not_a_readied_reaction"
	// valueResultReleaseReactionMismatch is the value of srd5e.combat.result for release_reaction_mismatch (refused).
	valueResultReleaseReactionMismatch = "release_reaction_mismatch"
	// valueResultDissipateNoHeldSpell is the value of srd5e.combat.result for dissipate_no_held_spell (refused).
	valueResultDissipateNoHeldSpell = "dissipate_no_held_spell"
	// valueResultDissipateInvalidCause is the value of srd5e.combat.result for dissipate_invalid_cause (invalid).
	valueResultDissipateInvalidCause = "dissipate_invalid_cause"
	// valueResultContactInitiatorNotTelepathic is the value of srd5e.combat.result for contact_initiator_not_telepathic (refused).
	valueResultContactInitiatorNotTelepathic = "contact_initiator_not_telepathic"
	// valueResultContactInitiatorIncapacitated is the value of srd5e.combat.result for contact_initiator_incapacitated (refused).
	valueResultContactInitiatorIncapacitated = "contact_initiator_incapacitated"
	// valueResultContactContactedIncapacitated is the value of srd5e.combat.result for contact_contacted_incapacitated (refused).
	valueResultContactContactedIncapacitated = "contact_contacted_incapacitated"
	// valueResultContactOutOfRange is the value of srd5e.combat.result for contact_out_of_range (refused).
	valueResultContactOutOfRange = "contact_out_of_range"
	// valueResultContactCannotUnderstand is the value of srd5e.combat.result for contact_cannot_understand (refused).
	valueResultContactCannotUnderstand = "contact_cannot_understand"
	// valueResultContactAlreadyMade is the value of srd5e.combat.result for contact_already_made (refused).
	valueResultContactAlreadyMade = "contact_already_made"
	// valueResultContactReplyNotPermitted is the value of srd5e.combat.result for contact_reply_not_permitted (refused).
	valueResultContactReplyNotPermitted = "contact_reply_not_permitted"
	// valueResultContactReplyNotContacted is the value of srd5e.combat.result for contact_reply_not_contacted (refused).
	valueResultContactReplyNotContacted = "contact_reply_not_contacted"
	// valueResultContactNone is the value of srd5e.combat.result for contact_none (refused).
	valueResultContactNone = "contact_none"
	// valueResultContactInvalid is the value of srd5e.combat.result for contact_invalid (invalid).
	valueResultContactInvalid = "contact_invalid"
	// valueResultZoneNoFacts is the value of srd5e.combat.result for zone_no_facts (refused).
	valueResultZoneNoFacts = "zone_no_facts"
	// valueResultZoneCreateInvalid is the value of srd5e.combat.result for zone_create_invalid (invalid).
	valueResultZoneCreateInvalid = "zone_create_invalid"
	// valueResultZoneRefused is the value of srd5e.combat.result for zone_refused (refused).
	valueResultZoneRefused = "zone_refused"
	// valueResultZoneStaleTurn is the value of srd5e.combat.result for zone_stale_turn (invalid).
	valueResultZoneStaleTurn = "zone_stale_turn"
	// valueResultZoneMoveRefused is the value of srd5e.combat.result for zone_move_refused (refused).
	valueResultZoneMoveRefused = "zone_move_refused"
	// valueResultZoneAlreadyEnded is the value of srd5e.combat.result for zone_already_ended (refused).
	valueResultZoneAlreadyEnded = "zone_already_ended"
	// valueResultShapeShiftResisted is the value of srd5e.combat.result for shape_shift_resisted (applied).
	valueResultShapeShiftResisted = "shape_shift_resisted"
	// valueResultShapeShiftEnded is the value of srd5e.combat.result for shape_shift_ended (applied).
	valueResultShapeShiftEnded = "shape_shift_ended"
	// valueResultShapeShiftUnaffected is the value of srd5e.combat.result for shape_shift_unaffected (applied).
	valueResultShapeShiftUnaffected = "shape_shift_unaffected"
	// valueResultRevived is the value of srd5e.combat.result for revived (applied).
	valueResultRevived = "revived"
	// valueResultNotRevived is the value of srd5e.combat.result for not_revived (applied).
	valueResultNotRevived = "not_revived"
	// valueResultIneligible is the value of srd5e.combat.result for ineligible (refused).
	valueResultIneligible = "ineligible"
	// valueResultRefused is the value of srd5e.combat.result for refused (refused).
	valueResultRefused = "refused"
	// valueResultIncreased is the value of srd5e.combat.result for increased (applied).
	valueResultIncreased = "increased"
	// valueResultWarded is the value of srd5e.combat.result for warded (applied).
	valueResultWarded = "warded"
	// valueResultNotWarded is the value of srd5e.combat.result for not_warded (applied).
	valueResultNotWarded = "not_warded"
	// valueResultReviveNotDead is the value of srd5e.combat.result for revive_not_dead (refused).
	valueResultReviveNotDead = "revive_not_dead"
	// valueResultReviveRefused is the value of srd5e.combat.result for revive_refused (refused).
	valueResultReviveRefused = "revive_refused"
	// valueResultReviveHitPointsBelowOne is the value of srd5e.combat.result for revive_hit_points_below_one (refused).
	valueResultReviveHitPointsBelowOne = "revive_hit_points_below_one"
	// valueResultReviveMaximumBelowOne is the value of srd5e.combat.result for revive_maximum_below_one (refused).
	valueResultReviveMaximumBelowOne = "revive_maximum_below_one"
)

// Operations of the roster use cases (combatant-roster-changes.md section 7): the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>. The domain files are silent.
const (
	// valueOperationAddCombatantToEncounter is the value of srd5e.combat.operation for AddCombatantToEncounter.
	valueOperationAddCombatantToEncounter = "add_combatant_to_encounter"
	// spanAddCombatantToEncounter is the span of AddCombatantToEncounter.
	spanAddCombatantToEncounter = "srd5e.combat.add_combatant_to_encounter"
	// valueOperationRemoveCombatantFromEncounter is the value of srd5e.combat.operation for RemoveCombatantFromEncounter.
	valueOperationRemoveCombatantFromEncounter = "remove_combatant_from_encounter"
	// spanRemoveCombatantFromEncounter is the span of RemoveCombatantFromEncounter.
	spanRemoveCombatantFromEncounter = "srd5e.combat.remove_combatant_from_encounter"
	// valueOperationRepositionCombatantInEncounter is the value of srd5e.combat.operation for RepositionCombatantInEncounter.
	valueOperationRepositionCombatantInEncounter = "reposition_combatant_in_encounter"
	// spanRepositionCombatantInEncounter is the span of RepositionCombatantInEncounter.
	spanRepositionCombatantInEncounter = "srd5e.combat.reposition_combatant_in_encounter"
)

// The attribute of the roster use cases (combatant-roster-changes.md FR-043): why the roster changed, on
// the span and the srd5e.combat.operations counter of every call, a refused one included. Its value is the
// JoinReason of an add, the RemovalCause of a removal (their String) or valueRosterReasonRepositioned.
const (
	// attrRosterReason is the closed-set reason of a roster call: summoned, arrived, fled, surrendered,
	// disappeared, departed or repositioned.
	attrRosterReason = "srd5e.combat.roster_reason"
	// valueRosterReasonRepositioned is the value of srd5e.combat.roster_reason for a reposition.
	valueRosterReasonRepositioned = "repositioned"
)

// Values of srd5e.combat.result of the roster use cases (combatant-roster-changes.md table 4.6a):
// valueResultApplied, valueResultUnknownCombatant and valueResultLimitExceeded are shared; these are
// the others. The first three of the refusals are refused, the duplicate is invalid.
const (
	// valueResultRosterEncounterEnded is the value of srd5e.combat.result for a roster change on an ended encounter (refused).
	valueResultRosterEncounterEnded = "encounter_already_ended"
	// valueResultCombatantIsActive is the value of srd5e.combat.result for a removal or move of the active combatant (refused).
	valueResultCombatantIsActive = "combatant_is_active"
	// valueResultWouldEmptySide is the value of srd5e.combat.result for a removal that would leave a side empty (refused).
	valueResultWouldEmptySide = "would_empty_side"
	// valueResultDuplicateCombatant is the value of srd5e.combat.result for an add of an id the encounter already holds (invalid).
	valueResultDuplicateCombatant = "duplicate_combatant"
)

// Operations of the summoned-creature use cases (summoned-and-companion-creatures.md section 7): the snake_case value of
// srd5e.combat.operation and the matching span srd5e.combat.<operation>. The domain files are silent.
const (
	// valueOperationSummonCreature is the value of srd5e.combat.operation for SummonCreature.
	valueOperationSummonCreature = "summon_creature"
	// spanSummonCreature is the span of SummonCreature.
	spanSummonCreature = "srd5e.combat.summon_creature"
	// valueOperationApplySummonedChange is the value of srd5e.combat.operation for ApplySummonedChange.
	valueOperationApplySummonedChange = "apply_summoned_change"
	// spanApplySummonedChange is the span of ApplySummonedChange.
	spanApplySummonedChange = "srd5e.combat.apply_summoned_change"
)

// Signals of the summoned-creature use cases (summoned-and-companion-creatures.md section 7).
const (
	// metricSummons counts the creatures a SummonCreatureApplied result brought, by kind ({creature}).
	metricSummons = "srd5e.combat.summons"
	// metricSummonEndings counts the creatures that left their summoner or whose control lapsed, by kind and cause ({creature}).
	metricSummonEndings = "srd5e.combat.summon_endings"

	// attrSummonKind is the snake_case SummonKind.
	attrSummonKind = "srd5e.combat.summon_kind"
	// attrSummonEndCause is the snake_case SummonEndCause.
	attrSummonEndCause = "srd5e.combat.summon_end_cause"
)

// Values of srd5e.combat.result of SummonCreature: a refusal is valueResultRejectedPrefix plus the reason of its
// SummonRejection (a SummonLimitExceededRejection is the shared valueResultLimitExceeded, class invalid); the others are
// listed here. valueResultContentMissing is class invalid, the rest applied.
const (
	// valueResultSummoned is the value of srd5e.combat.result for creatures that were brought (applied).
	valueResultSummoned = "summoned"
	// valueResultFormChanged is the value of srd5e.combat.result for a familiar whose form changed (applied).
	valueResultFormChanged = "form_changed"
	// valueResultReplaced is the value of srd5e.combat.result for a summons that replaced a steed or simulacrum (applied).
	valueResultReplaced = "replaced"
	// valueResultControlReasserted is the value of srd5e.combat.result for control that was reasserted (applied).
	valueResultControlReasserted = "control_reasserted"
	// valueResultContentMissing is the value of srd5e.combat.result for a monster the content source lacks (invalid).
	valueResultContentMissing = "content_missing"
	// valueResultRejectedPrefix starts the value of srd5e.combat.result of a refusal: this prefix and the reason.
	valueResultRejectedPrefix = "rejected_"
)

// Values of srd5e.combat.result of ApplySummonedChange besides the refusals (valueResultRejectedPrefix and a reason), all applied.
const (
	// valueResultCommanded is the value of srd5e.combat.result for an order that was given.
	valueResultCommanded = "commanded"
	// valueResultDeparted is the value of srd5e.combat.result for creatures or a shared Resistance that ended.
	valueResultDeparted = "departed"
	// valueResultLapsed is the value of srd5e.combat.result for time that lapsed the control of an undead creature.
	valueResultLapsed = "lapsed"
	// valueResultElapsed is the value of srd5e.combat.result for time that passed with no window lapsing.
	valueResultElapsed = "elapsed"
	// valueResultDismissed is the value of srd5e.combat.result for a familiar that was dismissed.
	valueResultDismissed = "dismissed"
	// valueResultRecalled is the value of srd5e.combat.result for a familiar that was recalled.
	valueResultRecalled = "recalled"
	// valueResultRepaired is the value of srd5e.combat.result for a simulacrum that was repaired.
	valueResultRepaired = "repaired"
	// valueResultTouchDelivered is the value of srd5e.combat.result for a touch a familiar delivered.
	valueResultTouchDelivered = "touch_delivered"
	// valueResultSensesShared is the value of srd5e.combat.result for a sight link that began.
	valueResultSensesShared = "senses_shared"
	// valueResultLifeBond is the value of srd5e.combat.result for a Life Bond that was worked out.
	valueResultLifeBond = "life_bond"
	// valueResultTurnReported is the value of srd5e.combat.result for the start or end of a turn that changed the roster.
	valueResultTurnReported = "turn_reported"
	// valueResultTaskCompleted is the value of srd5e.combat.result for a standing order that ended.
	valueResultTaskCompleted = "task_completed"
	// valueResultActionAllowed is the value of srd5e.combat.result for a familiar's action the rules allow.
	valueResultActionAllowed = "action_allowed"
	// valueResultUnchanged is the value of srd5e.combat.result for a change that had nothing to change.
	valueResultUnchanged = "unchanged"
)

// The reasons of the refusals of the summoned-creature use cases (Table R): the snake_case of the SummonRejection, the part of
// srd5e.combat.result after valueResultRejectedPrefix.
const (
	// valueSummonReasonIDInUse is the reason of a SummonCreatureIDInUseRejection.
	valueSummonReasonIDInUse = "summon_creature_id_in_use"
	// valueSummonReasonLinkCycle is the reason of a SummonLinkCycleRejection.
	valueSummonReasonLinkCycle = "summon_link_cycle"
	// valueSummonReasonLevelBelowSpell is the reason of a LevelBelowSpellRejection.
	valueSummonReasonLevelBelowSpell = "level_below_spell"
	// valueSummonReasonNotNight is the reason of a NotNightRejection.
	valueSummonReasonNotNight = "not_night"
	// valueSummonReasonFormNotEligible is the reason of a FormNotEligibleRejection.
	valueSummonReasonFormNotEligible = "form_not_eligible"
	// valueSummonReasonSteedNotDisappeared is the reason of a SteedNotDisappearedRejection.
	valueSummonReasonSteedNotDisappeared = "steed_not_disappeared"
	// valueSummonReasonTargetNotEligible is the reason of a TargetNotEligibleRejection.
	valueSummonReasonTargetNotEligible = "target_not_eligible"
	// valueSummonReasonObjectNotEligible is the reason of an ObjectNotEligibleRejection.
	valueSummonReasonObjectNotEligible = "object_not_eligible"
	// valueSummonReasonOriginalNotEligible is the reason of an OriginalNotEligibleRejection.
	valueSummonReasonOriginalNotEligible = "original_not_eligible"
	// valueSummonReasonDuplicateTarget is the reason of a DuplicateTargetRejection.
	valueSummonReasonDuplicateTarget = "duplicate_target"
	// valueSummonReasonTooManyCreatures is the reason of a TooManyCreaturesRejection.
	valueSummonReasonTooManyCreatures = "too_many_creatures"
	// valueSummonReasonCompositionNotAllowed is the reason of a CompositionNotAllowedRejection.
	valueSummonReasonCompositionNotAllowed = "composition_not_allowed"
	// valueSummonReasonControlLapsed is the reason of a ControlLapsedRejection.
	valueSummonReasonControlLapsed = "control_lapsed"
	// valueSummonReasonNothingToAnimate is the reason of a NothingToAnimateRejection.
	valueSummonReasonNothingToAnimate = "nothing_to_animate"
	// valueSummonReasonNothingToReassert is the reason of a NothingToReassertRejection.
	valueSummonReasonNothingToReassert = "nothing_to_reassert"
	// valueSummonReasonResistanceNotOffered is the reason of a ResistanceNotOfferedRejection.
	valueSummonReasonResistanceNotOffered = "resistance_not_offered"
	// valueSummonReasonSpaceUnavailable is the reason of a SpaceUnavailableRejection.
	valueSummonReasonSpaceUnavailable = "space_unavailable"
	// valueSummonReasonCommandNotYours is the reason of a CommandNotYourCreatureRejection.
	valueSummonReasonCommandNotYours = "command_not_your_creature"
	// valueSummonReasonCommandOutOfRange is the reason of a CommandOutOfRangeRejection.
	valueSummonReasonCommandOutOfRange = "command_out_of_range"
	// valueSummonReasonCannotGiveVerbal is the reason of a CannotGiveVerbalCommandRejection.
	valueSummonReasonCannotGiveVerbal = "cannot_give_verbal_command"
	// valueSummonReasonFamiliarCannotAttack is the reason of a FamiliarCannotAttackRejection.
	valueSummonReasonFamiliarCannotAttack = "familiar_cannot_attack"
	// valueSummonReasonAlreadyDismissed is the reason of a FamiliarAlreadyDismissedRejection.
	valueSummonReasonAlreadyDismissed = "familiar_already_dismissed"
	// valueSummonReasonNotDismissed is the reason of a FamiliarNotDismissedRejection.
	valueSummonReasonNotDismissed = "familiar_not_dismissed"
	// valueSummonReasonSensesNotSharable is the reason of a SensesNotSharableRejection.
	valueSummonReasonSensesNotSharable = "senses_not_sharable"
	// valueSummonReasonTouchNotDeliverable is the reason of a TouchNotDeliverableRejection.
	valueSummonReasonTouchNotDeliverable = "touch_not_deliverable"
	// valueSummonReasonSimulacrumTooFar is the reason of a SimulacrumTooFarRejection.
	valueSummonReasonSimulacrumTooFar = "simulacrum_too_far"
	// valueSummonReasonHealingRefused is the reason of a SimulacrumHealingRefusedRejection.
	valueSummonReasonHealingRefused = "simulacrum_healing_refused"
	// valueSummonReasonRepairAmount is the reason of a SimulacrumRepairAmountRejection.
	valueSummonReasonRepairAmount = "simulacrum_repair_amount"
	// valueSummonReasonRepairNeedsRest is the reason of a SimulacrumRepairNeedsLongRestRejection.
	valueSummonReasonRepairNeedsRest = "simulacrum_repair_needs_long_rest"
	// valueSummonReasonLifeBondOutOfReach is the reason of a LifeBondOutOfReachRejection.
	valueSummonReasonLifeBondOutOfReach = "life_bond_out_of_reach"
	// valueSummonReasonTelepathyOnlySummoner is the reason of a TelepathyOnlyWithSummonerRejection.
	valueSummonReasonTelepathyOnlySummoner = "telepathy_only_with_summoner"
)

// Operations of the use cases that wrap the magical-effects services (magical-effects.md section 7, FR-148): the
// snake_case value of srd5e.combat.operation and the matching span srd5e.combat.<operation>.
const (
	// valueOperationRemoveCombatantCurse is the value of srd5e.combat.operation for RemoveCombatantCurse.
	valueOperationRemoveCombatantCurse = "remove_combatant_curse"
	// spanRemoveCombatantCurse is the span of RemoveCombatantCurse.
	spanRemoveCombatantCurse = "srd5e.combat.remove_combatant_curse"
	// valueOperationShiftCombatantShape is the value of srd5e.combat.operation for ShiftCombatantShape.
	valueOperationShiftCombatantShape = "shift_combatant_shape"
	// spanShiftCombatantShape is the span of ShiftCombatantShape.
	spanShiftCombatantShape = "srd5e.combat.shift_combatant_shape"
	// valueOperationTeleportCombatant is the value of srd5e.combat.operation for TeleportCombatant.
	valueOperationTeleportCombatant = "teleport_combatant"
	// spanTeleportCombatant is the span of TeleportCombatant.
	spanTeleportCombatant = "srd5e.combat.teleport_combatant"
	// valueOperationResolveEffectOrder is the value of srd5e.combat.operation for ResolveEffectOrder.
	valueOperationResolveEffectOrder = "resolve_effect_order"
	// spanResolveEffectOrder is the span of ResolveEffectOrder.
	spanResolveEffectOrder = "srd5e.combat.resolve_effect_order"
	// valueOperationAssessAreaLocation is the value of srd5e.combat.operation for AssessAreaLocation.
	valueOperationAssessAreaLocation = "assess_area_location"
	// spanAssessAreaLocation is the span of AssessAreaLocation.
	spanAssessAreaLocation = "srd5e.combat.assess_area_location"
)

// Values of srd5e.combat.result of the magical-effects use cases (magical-effects.md section 7); valueResultCompleted
// is shared.
const (
	// valueResultCurseNotPresent is the value of srd5e.combat.result for curse_not_present (refused).
	valueResultCurseNotPresent = "curse_not_present"
	// valueResultCurseNotACurse is the value of srd5e.combat.result for curse_not_a_curse (refused).
	valueResultCurseNotACurse = "curse_not_a_curse"
	// valueResultShiftExceptionNotHeld is the value of srd5e.combat.result for shift_exception_not_held (refused).
	valueResultShiftExceptionNotHeld = "shift_exception_not_held"
	// valueResultRevertNotDead is the value of srd5e.combat.result for revert_not_dead (refused).
	valueResultRevertNotDead = "revert_not_dead"
	// valueResultTeleportDestinationNotSeen is the value of srd5e.combat.result for teleport_destination_not_seen (refused).
	valueResultTeleportDestinationNotSeen = "teleport_destination_not_seen"
	// valueResultTeleportInvalidTouchedSet is the value of srd5e.combat.result for teleport_invalid_touched_set (invalid).
	valueResultTeleportInvalidTouchedSet = "teleport_invalid_touched_set"
	// valueResultOrderRejected is the value of srd5e.combat.result for order_rejected (invalid).
	valueResultOrderRejected = "order_rejected"
	// valueResultAreaLocationExcluded is the value of srd5e.combat.result for area_location_excluded (applied).
	valueResultAreaLocationExcluded = "area_location_excluded"
	// valueResultAreaNoLinesConsidered is the value of srd5e.combat.result for area_no_lines_considered (invalid).
	valueResultAreaNoLinesConsidered = "area_no_lines_considered"
)

// Signals recorded by every use case from the events it produced (observability.md section 5).
const (
	// metricEventsPerOperation records the number of events one use case call produced ({event}), by operation.
	metricEventsPerOperation = "srd5e.combat.events_per_operation"
	// metricShapeShiftEnded counts the transformations that ended, by cause ({transformation}).
	metricShapeShiftEnded = "srd5e.combat.shape_shift.ended"
	// metricZoneEnded counts the persistent zones that ended, by reason ({zone}).
	metricZoneEnded = "srd5e.combat.persistent_zone.ended"

	// attrShapeShiftEndCause is the snake_case ShapeShiftEndCause.
	attrShapeShiftEndCause = "srd5e.combat.shape_shift.end_cause"
	// attrZoneEndReason is the snake_case ZoneEndReason.
	attrZoneEndReason = "srd5e.combat.persistent_zone.end_reason"
)
