package combat

// ResolutionRejection is why a combat action resolution use case stopped: the field of every
// ...Rejected result (combat-action-resolution.md FR-127). A stop is a normal rule outcome, never an
// error. String is the value of srd5e.combat.result for the stop (FR-100). Variants:
// EncounterStateRejection, LedgerRejection, TrackerRejection, TargetingForbiddenRejection,
// AttackNotPossibleRejection, StatBlockRejection, MovementRestrictedRejection, PostureMismatchRejection,
// DeathSavingThrowOutOfTurnRejection.
type ResolutionRejection interface {
	isResolutionRejection()
	String() string
}

// EncounterStateRejection: the encounter refused (it ended, a combatant is dead, a creature is not
// Dying, the root refused a condition or a second Death Saving Throw).
type EncounterStateRejection struct{ Rejection EncounterRejection }

// LedgerRejection: the ActionLedger refused (no Attack action in progress, the attack limit, a weapon
// change or a Help refusal). It costs no resource (FR-010 step 4).
type LedgerRejection struct{ Refusal LedgerRefusal }

// TrackerRejection: the turn tracker refused the action, Bonus Action, Reaction, move or posture.
type TrackerRejection struct{ Rejection TurnRejection }

// TargetingForbiddenRejection: the attacker is Charmed by the target and so may not attack it.
type TargetingForbiddenRejection struct{}

// AttackNotPossibleRejection: the attack cannot be made (Total Cover, beyond range or reach, other
// reasons of attacks.md). No resource was spent and no die rolled.
type AttackNotPossibleRejection struct{ Reason AttackNotPossibleReason }

// StatBlockRejection: the use of a stat block part was refused (FR-164): Refusal is the refusal of UseStatBlock; for a
// Multiattack that is not available (no part, or no plan) it is nil and MultiattackNotAvailable is true. No
// resource was spent.
type StatBlockRejection struct {
	Refusal                 StatBlockUseRefusal
	MultiattackNotAvailable bool
}

// MovementRestrictedRejection: the conditions of the mover restrict its movement.
type MovementRestrictedRejection struct{ Restriction MovementRestriction }

// PostureMismatchRejection: the posture change does not fit the mover's posture.
type PostureMismatchRejection struct{ Mismatch PostureMismatch }

// DeathSavingThrowOutOfTurnRejection: a Death Saving Throw was asked of a combatant that is not the
// active combatant.
type DeathSavingThrowOutOfTurnRejection struct{}

func (EncounterStateRejection) isResolutionRejection()            {}
func (LedgerRejection) isResolutionRejection()                    {}
func (TrackerRejection) isResolutionRejection()                   {}
func (TargetingForbiddenRejection) isResolutionRejection()        {}
func (AttackNotPossibleRejection) isResolutionRejection()         {}
func (StatBlockRejection) isResolutionRejection()                 {}
func (MovementRestrictedRejection) isResolutionRejection()        {}
func (PostureMismatchRejection) isResolutionRejection()           {}
func (DeathSavingThrowOutOfTurnRejection) isResolutionRejection() {}

// String returns the name of the carried EncounterRejection.
func (r EncounterStateRejection) String() string {
	if r.Rejection == nil {
		return "unknown"
	}
	return r.Rejection.String()
}

// String returns "ledger_refused".
func (LedgerRejection) String() string { return valueResultLedgerRefused }

// String returns "turn_refused".
func (TrackerRejection) String() string { return valueResultTurnRefused }

// String returns "targeting_forbidden".
func (TargetingForbiddenRejection) String() string { return valueResultTargetingForbidden }

// String returns "attack_not_possible".
func (AttackNotPossibleRejection) String() string { return valueResultAttackNotPossible }

// String returns "stat_block_refused".
func (StatBlockRejection) String() string { return valueResultStatBlockRefused }

// String returns "movement_restricted".
func (MovementRestrictedRejection) String() string { return valueResultMovementRestricted }

// String returns "posture_mismatch".
func (PostureMismatchRejection) String() string { return valueResultPostureMismatch }

// String returns "death_saving_throw_out_of_turn".
func (DeathSavingThrowOutOfTurnRejection) String() string {
	return valueResultDeathSavingThrowOutOfTurn
}

// MovementRestriction says how conditions restrict a move or a posture change (SRD 5.2.1, "Prone
// [Condition]", p. 186; "Grappled [Condition]", p. 182). The zero value is not a restriction.
type MovementRestriction int

// The three restrictions.
const (
	// MovementRestrictionSpeedZeroedByCondition: a condition sets the mover's Speed to 0.
	MovementRestrictionSpeedZeroedByCondition MovementRestriction = iota + 1
	// MovementRestrictionProneAllowsOnlyCrawl: a Prone mover may only crawl.
	MovementRestrictionProneAllowsOnlyCrawl
	// MovementRestrictionCannotRightItself: a Prone mover whose Speed is 0 cannot stand up.
	MovementRestrictionCannotRightItself
)

// String returns "speed_zeroed_by_condition", "prone_allows_only_crawl", "cannot_right_itself" or "unknown".
func (r MovementRestriction) String() string {
	return enumName([]string{"speed_zeroed_by_condition", "prone_allows_only_crawl", "cannot_right_itself"}, int(r))
}

// PostureMismatch says why a posture change does not fit. The zero value is not a mismatch.
type PostureMismatch int

// The two mismatches.
const (
	// PostureMismatchAlreadyProne: Drop Prone for a mover that is already effectively Prone.
	PostureMismatchAlreadyProne PostureMismatch = iota + 1
	// PostureMismatchNotProne: stand up for a mover that is not effectively Prone.
	PostureMismatchNotProne
)

// String returns "already_prone", "not_prone" or "unknown".
func (m PostureMismatch) String() string {
	return enumName([]string{"already_prone", "not_prone"}, int(m))
}

// LedgerRefusal is why the ActionLedger refused: the reason carried by LedgerRejection. The reason
// enums are the siblings' (attack-and-help-actions.md). Variants: LedgerNoAttackActionInProgress,
// LedgerAttackRefused, LedgerWeaponChangeRefused, LedgerAbilityCheckHelpRefused,
// LedgerAttackRollHelpRefused, LedgerMultiattackRefused.
type LedgerRefusal interface {
	isLedgerRefusal()
	String() string
}

// LedgerNoAttackActionInProgress: an attack continues or a weapon is handled, and the attacker has no
// Attack action in progress.
type LedgerNoAttackActionInProgress struct{}

// LedgerAttackRefused: the attack was refused by the allowance (AttackRefusalLimitReached).
type LedgerAttackRefused struct{ Reason AttackRefusal }

// LedgerWeaponChangeRefused: the weapon change was refused.
type LedgerWeaponChangeRefused struct{ Reason WeaponChangeRefusal }

// LedgerAbilityCheckHelpRefused: the Help with an ability check was refused.
type LedgerAbilityCheckHelpRefused struct{ Reason AbilityCheckHelpRefusal }

// LedgerAttackRollHelpRefused: the Help with an attack roll was refused.
type LedgerAttackRollHelpRefused struct{ Reason AttackRollHelpRefusal }

// LedgerMultiattackRefused: a step of the Multiattack in progress was refused: Outcome is its
// MultiattackNotInAllowance or MultiattackStepUsageRefused variant (FR-165).
type LedgerMultiattackRefused struct{ Outcome MultiattackStepOutcome }

func (LedgerNoAttackActionInProgress) isLedgerRefusal() {}
func (LedgerAttackRefused) isLedgerRefusal()            {}
func (LedgerWeaponChangeRefused) isLedgerRefusal()      {}
func (LedgerAbilityCheckHelpRefused) isLedgerRefusal()  {}
func (LedgerAttackRollHelpRefused) isLedgerRefusal()    {}
func (LedgerMultiattackRefused) isLedgerRefusal()       {}

// String returns "no_attack_action_in_progress".
func (LedgerNoAttackActionInProgress) String() string { return "no_attack_action_in_progress" }

// String returns the name of the carried AttackRefusal.
func (r LedgerAttackRefused) String() string { return r.Reason.String() }

// String returns the name of the carried WeaponChangeRefusal.
func (r LedgerWeaponChangeRefused) String() string { return r.Reason.String() }

// String returns the name of the carried AbilityCheckHelpRefusal.
func (r LedgerAbilityCheckHelpRefused) String() string { return r.Reason.String() }

// String returns the name of the carried AttackRollHelpRefusal.
func (r LedgerAttackRollHelpRefused) String() string { return r.Reason.String() }

// String returns the name of the allowance refusal, or "multiattack_step_usage_refused".
func (r LedgerMultiattackRefused) String() string {
	if n, ok := r.Outcome.(MultiattackNotInAllowance); ok {
		return n.Reason.String()
	}
	return "multiattack_step_usage_refused"
}

// ResolutionProblem is why a call cannot be answered with a rule outcome: the field of every ...Invalid
// result (combat-action-resolution.md FR-011, FR-128). String is the value of srd5e.combat.result.
// Variants: UnknownCombatantProblem, LimitExceededProblem.
type ResolutionProblem interface {
	isResolutionProblem()
	String() string
}

// UnknownCombatantProblem: the host named a combatant the encounter does not hold.
type UnknownCombatantProblem struct{ ID CombatantID }

// LimitExceededProblem: the call would produce more events than the cap of 4096 per call (plan P10). The
// call changed nothing.
type LimitExceededProblem struct{}

func (UnknownCombatantProblem) isResolutionProblem() {}
func (LimitExceededProblem) isResolutionProblem()    {}

// String returns "unknown_combatant".
func (UnknownCombatantProblem) String() string { return valueResultUnknownCombatant }

// String returns "limit_exceeded".
func (LimitExceededProblem) String() string { return valueResultLimitExceeded }

// maxEventsPerCall is the cap of plan P10 on the events one call produces.
const maxEventsPerCall = 4096
