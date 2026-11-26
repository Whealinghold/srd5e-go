package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestResolutionRejectionNames(t *testing.T) {
	t.Parallel()
	a := cid("a")
	cases := []struct {
		name, requirement, path string
		got                     interface{ String() string }
		want                    string
	}{
		{"ended", resSpec + "FR-100", "unhappy (state)", combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, "encounter_already_ended"},
		{"dead", resSpec + "FR-100", "unhappy (state)", combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: a}}, "combatant_is_dead"},
		{"not dying", resSpec + "FR-100", "unhappy (state)", combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: a, Reason: combat.NotDying}}, "not_dying"},
		{"immune", resSpec + "FR-100", "unhappy (state)", combat.EncounterStateRejection{Rejection: combat.ImmuneToConditionRejection{ID: a, Kind: content.ConditionProne}}, "immune_to_condition"},
		{"second throw", resSpec + "FR-100", "unhappy (state)", combat.EncounterStateRejection{Rejection: combat.DeathSavingThrowAlreadyMadeRejection{ID: a}}, "death_saving_throw_already_made"},
		{"no rejection", resSpec + "FR-100", "unhappy (empty and minimal)", combat.EncounterStateRejection{}, "unknown"},
		{"ledger", resSpec + "FR-100", "unhappy (state)", combat.LedgerRejection{Refusal: combat.LedgerNoAttackActionInProgress{}}, "ledger_refused"},
		{"tracker", resSpec + "FR-100", "unhappy (state)", combat.TrackerRejection{Rejection: combat.TurnIncapacitatedRejection{}}, "turn_refused"},
		{"targeting", resSpec + "FR-100", "unhappy (rule interaction)", combat.TargetingForbiddenRejection{}, "targeting_forbidden"},
		{"not possible", resSpec + "FR-100", "unhappy (state)", combat.AttackNotPossibleRejection{Reason: combat.AttackNotPossibleReasonTotalCover}, "attack_not_possible"},
		{"movement", resSpec + "FR-100", "unhappy (rule interaction)", combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionProneAllowsOnlyCrawl}, "movement_restricted"},
		{"posture", resSpec + "FR-100", "unhappy (rule interaction)", combat.PostureMismatchRejection{Mismatch: combat.PostureMismatchNotProne}, "posture_mismatch"},
		{"out of turn", resSpec + "FR-100", "unhappy (state)", combat.DeathSavingThrowOutOfTurnRejection{}, "death_saving_throw_out_of_turn"},
		{"unknown combatant problem", resSpec + "FR-100", "unhappy (invalid input)", combat.UnknownCombatantProblem{ID: a}, "unknown_combatant"},
		{"limit exceeded problem", resSpec + "FR-100, plan P10", "unhappy (boundary)", combat.LimitExceededProblem{}, "limit_exceeded"},
		{"no attack action", resSpec + "FR-113", "unhappy (state)", combat.LedgerNoAttackActionInProgress{}, "no_attack_action_in_progress"},
		{"attack limit", resSpec + "FR-113", "unhappy (boundary)", combat.LedgerAttackRefused{Reason: combat.AttackRefusalLimitReached}, "attack_limit_reached"},
		{"weapon change", resSpec + "FR-118", "unhappy (state)", combat.LedgerWeaponChangeRefused{Reason: combat.WeaponChangeRefusalNoAttackMade}, "no_attack_made"},
		{"check help", resSpec + "FR-117", "unhappy (rule interaction)", combat.LedgerAbilityCheckHelpRefused{Reason: combat.AbilityCheckHelpRefusalCannotHelpSelf}, "cannot_help_self"},
		{"attack help", resSpec + "FR-117", "unhappy (rule interaction)", combat.LedgerAttackRollHelpRefused{Reason: combat.AttackRollHelpRefusalEnemyNotWithinFiveFeet}, "enemy_not_within_five_feet"},
		{"restriction speed", resSpec + "FR-057", "unhappy (rule interaction)", combat.MovementRestrictionSpeedZeroedByCondition, "speed_zeroed_by_condition"},
		{"restriction crawl", resSpec + "FR-058", "unhappy (rule interaction)", combat.MovementRestrictionProneAllowsOnlyCrawl, "prone_allows_only_crawl"},
		{"restriction stand", resSpec + "FR-061", "unhappy (boundary)", combat.MovementRestrictionCannotRightItself, "cannot_right_itself"},
		{"posture already", resSpec + "FR-060", "unhappy (state)", combat.PostureMismatchAlreadyProne, "already_prone"},
		{"posture not", resSpec + "FR-061", "unhappy (state)", combat.PostureMismatchNotProne, "not_prone"},
		{"restriction zero", resSpec + "FR-057", "unhappy (invalid input)", combat.MovementRestriction(0), "unknown"},
		{"posture zero", resSpec + "FR-061", "unhappy (invalid input)", combat.PostureMismatch(0), "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.got.String(); got != c.want {
				t.Errorf("String() = %q, want %q [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-004, FR-009 - a rejection and a problem hold the unchanged encounter and no event, which is the contract the results below pin; here the zero KnockOutChoice is NoKnockOut and an undefined choice is not valid.
// Path: unhappy (invalid input).
func TestKnockOutChoice(t *testing.T) {
	t.Parallel()
	var zero combat.KnockOutChoice
	if zero != combat.NoKnockOut || !zero.Valid() || !combat.KnockOut.Valid() || combat.KnockOutChoice(2).Valid() || combat.KnockOutChoice(-1).Valid() {
		t.Errorf("zero %v, KnockOut valid %v: want the zero value NoKnockOut and no third choice", zero, combat.KnockOut.Valid())
	}
	if combat.KnockOut.String() != "knock_out" || combat.NoKnockOut.String() != "no_knock_out" || combat.KnockOutChoice(7).String() != "unknown" {
		t.Errorf("names = %q, %q, %q", combat.KnockOut, combat.NoKnockOut, combat.KnockOutChoice(7))
	}
}
