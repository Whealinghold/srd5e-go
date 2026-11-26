package combat

import "github.com/Whealinghold/srd5e-go/d20"

// TestResolution is how one d20 test of a combat action resolution was settled, in d20's own
// values (combat-action-resolution.md FR-121). Variants: TestRolled, TestForcedFailure.
type TestResolution interface {
	isTestResolution()
	// Outcome returns d20.Success or d20.Failure.
	Outcome() d20.Outcome
}

// TestRolled: the test was rolled.
type TestRolled struct{ result d20.CheckResult }

// TestForcedFailure: the test failed without a die.
type TestForcedFailure struct{ result d20.ForcedFailureResult }

func (TestRolled) isTestResolution()        {}
func (TestForcedFailure) isTestResolution() {}

// Result returns the test as d20.ResolveCheck resolved it.
func (t TestRolled) Result() d20.CheckResult { return t.result }

// Outcome returns the outcome of the roll.
func (t TestRolled) Outcome() d20.Outcome { return t.result.Outcome() }

// Result returns the forced failure d20.FailWithoutRoll returned.
func (t TestForcedFailure) Result() d20.ForcedFailureResult { return t.result }

// Outcome returns d20.Failure.
func (TestForcedFailure) Outcome() d20.Outcome { return d20.Failure }

// AttackPerformedOutcome is what an attack came to, the payload of AttackPerformed: domain values
// only (combat-action-resolution.md FR-121). Variants: AttackHitOutcome, AttackMissOutcome,
// AttackAutomaticallyMissedOutcome.
type AttackPerformedOutcome interface{ isAttackPerformedOutcome() }

// AttackHitOutcome: the attack hit; HitKind says whether it was a Critical Hit.
type AttackHitOutcome struct {
	roll d20.AttackRollResult
	kind HitKind
}

// AttackMissOutcome: the attack missed, by the roll or by a forced miss.
type AttackMissOutcome struct{ roll d20.AttackRollResult }

// AttackAutomaticallyMissedOutcome: the attack missed without a roll.
type AttackAutomaticallyMissedOutcome struct{ reason AutomaticMissReason }

func (AttackHitOutcome) isAttackPerformedOutcome()                 {}
func (AttackMissOutcome) isAttackPerformedOutcome()                {}
func (AttackAutomaticallyMissedOutcome) isAttackPerformedOutcome() {}

// Roll returns the attack roll exactly as d20.ResolveAttack returned it.
func (o AttackHitOutcome) Roll() d20.AttackRollResult { return o.roll }

// HitKind returns NormalHit or CriticalHit.
func (o AttackHitOutcome) HitKind() HitKind { return o.kind }

// Roll returns the attack roll exactly as d20.ResolveAttack returned it.
func (o AttackMissOutcome) Roll() d20.AttackRollResult { return o.roll }

// Reason returns why no roll was made.
func (o AttackAutomaticallyMissedOutcome) Reason() AutomaticMissReason { return o.reason }

// UnarmedStrikePerformedOutcome is what an Unarmed Strike came to, the payload of
// UnarmedStrikePerformed. Variants: UnarmedDamageOptionPerformed, UnarmedSavePerformed.
type UnarmedStrikePerformedOutcome interface{ isUnarmedStrikePerformedOutcome() }

// UnarmedDamageOptionPerformed: the Damage option rolled an attack.
type UnarmedDamageOptionPerformed struct{ outcome AttackPerformedOutcome }

// UnarmedSavePerformed: the Grapple or Shove option made the target save against the DC.
type UnarmedSavePerformed struct {
	dc   d20.DifficultyClass
	save TestResolution
}

func (UnarmedDamageOptionPerformed) isUnarmedStrikePerformedOutcome() {}
func (UnarmedSavePerformed) isUnarmedStrikePerformedOutcome()         {}

// Outcome returns the attack outcome of the Damage option.
func (o UnarmedDamageOptionPerformed) Outcome() AttackPerformedOutcome { return o.outcome }

// DC returns the Grapple or Shove DC.
func (o UnarmedSavePerformed) DC() d20.DifficultyClass { return o.dc }

// Save returns the target's saving throw.
func (o UnarmedSavePerformed) Save() TestResolution { return o.save }

// UnarmedEffectOutcome is what the effect of a Grapple or Shove came to
// (combat-action-resolution.md FR-046 to FR-048). The Damage option has no effect and reports
// UnarmedEffectNotTriggered. Variants: UnarmedEffectApplied, UnarmedEffectBlocked,
// UnarmedEffectNotTriggered, UnarmedEffectPushRecorded.
type UnarmedEffectOutcome interface{ isUnarmedEffectOutcome() }

// UnarmedEffectApplied: the condition was applied to the target through the encounter.
type UnarmedEffectApplied struct{ condition Condition }

// UnarmedEffectBlocked: the save failed but the encounter refused the condition (the target already
// has it, or is Immune to it). The call still completes.
type UnarmedEffectBlocked struct{ rejection EncounterRejection }

// UnarmedEffectNotTriggered: no effect applies (the target saved, or the option has none).
type UnarmedEffectNotTriggered struct{}

// UnarmedEffectPushRecorded: the Shove pushed the target 5 feet. The engine holds no position, so no
// state changes; the host moves the creature.
type UnarmedEffectPushRecorded struct{}

func (UnarmedEffectApplied) isUnarmedEffectOutcome()      {}
func (UnarmedEffectBlocked) isUnarmedEffectOutcome()      {}
func (UnarmedEffectNotTriggered) isUnarmedEffectOutcome() {}
func (UnarmedEffectPushRecorded) isUnarmedEffectOutcome() {}

// Condition returns the condition applied.
func (o UnarmedEffectApplied) Condition() Condition { return o.condition }

// Rejection returns why the encounter refused the condition.
func (o UnarmedEffectBlocked) Rejection() EncounterRejection { return o.rejection }
