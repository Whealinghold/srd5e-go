package combat

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md section 7, FR-074 - the two operations, their spans, the two metrics, the two attribute keys and every result value of the summoned-creature use cases are pinned to their catalogue spelling (renaming one breaks consumers' dashboards), and each operation and span pair is a valid combatOperation.
// Path: per case.
func TestSummonedSignalValuesArePinned(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got, want               string
	}{
		{"summon operation", "FR-074", "happy", valueOperationSummonCreature, "summon_creature"},
		{"summon span", "FR-074", "happy", spanSummonCreature, "srd5e.combat.summon_creature"},
		{"change operation", "FR-074", "happy", valueOperationApplySummonedChange, "apply_summoned_change"},
		{"change span", "FR-074", "happy", spanApplySummonedChange, "srd5e.combat.apply_summoned_change"},
		{"summons metric", "section 7", "happy", metricSummons, "srd5e.combat.summons"},
		{"endings metric", "section 7", "happy", metricSummonEndings, "srd5e.combat.summon_endings"},
		{"summon_kind key", "section 7", "happy", attrSummonKind, "srd5e.combat.summon_kind"},
		{"summon_end_cause key", "section 7", "happy", attrSummonEndCause, "srd5e.combat.summon_end_cause"},
		{"summoned", "section 7", "happy", valueResultSummoned, "summoned"},
		{"form_changed", "section 7", "happy", valueResultFormChanged, "form_changed"},
		{"replaced", "section 7", "happy", valueResultReplaced, "replaced"},
		{"control_reasserted", "section 7", "happy", valueResultControlReasserted, "control_reasserted"},
		{"content_missing", "section 7", "unhappy (missing content)", valueResultContentMissing, "content_missing"},
		{"rejected_ prefix", "section 7", "unhappy (state)", valueResultRejectedPrefix, "rejected_"},
		{"commanded", "section 7", "happy", valueResultCommanded, "commanded"},
		{"departed", "section 7", "happy", valueResultDeparted, "departed"},
		{"lapsed", "section 7", "happy", valueResultLapsed, "lapsed"},
		{"elapsed", "section 7", "happy", valueResultElapsed, "elapsed"},
		{"dismissed", "section 7", "happy", valueResultDismissed, "dismissed"},
		{"recalled", "section 7", "happy", valueResultRecalled, "recalled"},
		{"repaired", "section 7", "happy", valueResultRepaired, "repaired"},
		{"touch_delivered", "section 7", "happy", valueResultTouchDelivered, "touch_delivered"},
		{"senses_shared", "section 7", "happy", valueResultSensesShared, "senses_shared"},
		{"life_bond", "section 7", "happy", valueResultLifeBond, "life_bond"},
		{"turn_reported", "FR-046, FR-013", "happy", valueResultTurnReported, "turn_reported"},
		{"task_completed", "FR-013", "happy", valueResultTaskCompleted, "task_completed"},
		{"action_allowed", "FR-048", "happy", valueResultActionAllowed, "action_allowed"},
		{"unchanged", "section 7", "unhappy (state)", valueResultUnchanged, "unchanged"},
		{"limit_exceeded", "plan P10", "unhappy (boundary)", valueResultLimitExceeded, "limit_exceeded"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("%s = %q, want %q [%s%s]", c.name, c.got, c.want, "doc/specs/summoned-and-companion-creatures.md ", c.requirement)
			}
		})
	}
	for _, pair := range [][2]string{{valueOperationSummonCreature, spanSummonCreature}, {valueOperationApplySummonedChange, spanApplySummonedChange}} {
		if _, err := newCombatOperation(pair[0], pair[1]); err != nil {
			t.Errorf("operation %s with span %s is not valid: %v", pair[0], pair[1], err)
		}
	}
	if maxSummonsPerSummoner != 32 {
		t.Errorf("maxSummonsPerSummoner = %d, want 32 (plan P10)", maxSummonsPerSummoner)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-068, FR-073, Table R - each rejection has the snake_case reason the spec's Table R names, every reason is unique, only the limit is class invalid (and then its result is the shared limit_exceeded, with no rejected_ prefix) and every other is a refusal.
// Path: per case.
func TestSummonRejectionReasonsAndClasses(t *testing.T) {
	t.Parallel()
	all := []struct {
		r      SummonRejection
		reason string
	}{
		{SummonCreatureIDInUseRejection{}, "summon_creature_id_in_use"}, {SummonLinkCycleRejection{}, "summon_link_cycle"}, {LevelBelowSpellRejection{}, "level_below_spell"},
		{NotNightRejection{}, "not_night"}, {FormNotEligibleRejection{}, "form_not_eligible"}, {SteedNotDisappearedRejection{}, "steed_not_disappeared"},
		{TargetNotEligibleRejection{}, "target_not_eligible"}, {ObjectNotEligibleRejection{}, "object_not_eligible"}, {OriginalNotEligibleRejection{}, "original_not_eligible"},
		{DuplicateTargetRejection{}, "duplicate_target"}, {TooManyCreaturesRejection{}, "too_many_creatures"}, {CompositionNotAllowedRejection{}, "composition_not_allowed"},
		{ControlLapsedRejection{}, "control_lapsed"}, {NothingToAnimateRejection{}, "nothing_to_animate"}, {NothingToReassertRejection{}, "nothing_to_reassert"},
		{ResistanceNotOfferedRejection{}, "resistance_not_offered"}, {SpaceUnavailableRejection{}, "space_unavailable"}, {SummonLimitExceededRejection{}, "limit_exceeded"},
		{CommandNotYourCreatureRejection{}, "command_not_your_creature"}, {CommandOutOfRangeRejection{}, "command_out_of_range"}, {CannotGiveVerbalCommandRejection{}, "cannot_give_verbal_command"},
		{FamiliarCannotAttackRejection{}, "familiar_cannot_attack"}, {FamiliarAlreadyDismissedRejection{}, "familiar_already_dismissed"}, {FamiliarNotDismissedRejection{}, "familiar_not_dismissed"},
		{SensesNotSharableRejection{}, "senses_not_sharable"}, {TouchNotDeliverableRejection{}, "touch_not_deliverable"}, {SimulacrumTooFarRejection{}, "simulacrum_too_far"},
		{SimulacrumHealingRefusedRejection{}, "simulacrum_healing_refused"}, {SimulacrumRepairAmountRejection{}, "simulacrum_repair_amount"},
		{SimulacrumRepairNeedsLongRestRejection{}, "simulacrum_repair_needs_long_rest"}, {LifeBondOutOfReachRejection{}, "life_bond_out_of_reach"},
		{TelepathyOnlyWithSummonerRejection{}, "telepathy_only_with_summoner"},
	}
	seen := map[string]bool{}
	for _, c := range all {
		if got := c.r.String(); got != c.reason || seen[got] {
			t.Errorf("%T reason = %q, want the unique %q [Table R]", c.r, got, c.reason)
		}
		seen[c.reason] = true
		_, limit := c.r.(SummonLimitExceededRejection)
		if (summonRejectionClass(c.r) == resultInvalid) != limit || summonRejectionClass(c.r).IsSpanError() != limit {
			t.Errorf("%T class = %v, want invalid only for the limit [FR-073]", c.r, summonRejectionClass(c.r))
		}
		want := valueResultRejectedPrefix + c.reason
		if limit {
			want = valueResultLimitExceeded
		}
		if got := rejectionResultName(c.r); got != want {
			t.Errorf("%T result = %q, want %q", c.r, got, want)
		}
	}
	if len(all) != 32 {
		t.Errorf("%d rejections listed, want 32", len(all))
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-006, FR-026 - a familiar dismissed or ended by a Long Rest has RemovalCauseDeparted and every other ending RemovalCauseDisappeared; a spell, kind and position lookup of an undefined kind is empty.
// Path: per case.
func TestSummonEndCauseRemoval(t *testing.T) {
	t.Parallel()
	for cause, want := range map[SummonEndCause]RemovalCause{
		SummonEndDismissedForever: RemovalCauseDeparted, SummonEndDismissedTemporarily: RemovalCauseDeparted, SummonEndLongRestFinished: RemovalCauseDeparted,
		SummonEndDroppedToZeroHitPoints: RemovalCauseDisappeared, SummonEndSpellEnded: RemovalCauseDisappeared, SummonEndSummonerDied: RemovalCauseDisappeared,
		SummonEndReplaced: RemovalCauseDisappeared, SummonEndControlLapsed: RemovalCauseDisappeared,
	} {
		if got := cause.removal(); got != want {
			t.Errorf("%v.removal() = %v, want %v [FR-006]", cause, got, want)
		}
	}
	if spellLevel(0) != 0 || spellLevel(SummonSimulacrum+1) != 0 || spellLevel(SummonSimulacrum) != 7 || SummonKind(0).Spell() != "" {
		t.Errorf("an undefined kind has a spell level or a spell")
	}
	if got := objectWeight(content.SizeHuge) + objectWeight(content.SizeLarge) + objectWeight(content.SizeTiny); got != 6 {
		t.Errorf("object weights sum to %d, want 3 + 2 + 1 [FR-031]", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-070, errors-and-results.md FR-031 - lookupMonsters returns the monsters found and the entries not found ordered by id, and stops at the first port failure; a source that answers with the zero Monster, a result that is neither found nor not found or a wrong entry breaks the port's contract.
// Path: per case.
func TestLookupMonstersOrdersTheMissing(t *testing.T) {
	t.Parallel()
	got, missing, err := lookupMonsters(t.Context(), emptySource{}, []content.MonsterID{"zombie", "skeleton", "ghoul"})
	if err != nil || len(got) != 0 || len(missing) != 3 || missing[0].ID().String() != "ghoul" || missing[2].ID().String() != "zombie" {
		t.Errorf("lookup = %v, %v, %v, want three missing ordered by id [FR-071]", got, missing, err)
	}
	if _, _, err := lookupMonsters(t.Context(), emptySource{wrong: true}, []content.MonsterID{"zombie"}); !errors.Is(err, srderr.ErrContractViolation) {
		t.Errorf("a found result with the zero Monster: error = %v, want a contract violation", err)
	}
	if got, missing, err := lookupMonsters(t.Context(), emptySource{}, nil); err != nil || got != nil || missing != nil {
		t.Errorf("no ids = %v, %v, %v, want nothing", got, missing, err)
	}
}

// emptySource answers every lookup with not found, or with a found result holding the zero Monster.
type emptySource struct{ wrong bool }

func (s emptySource) Lookup(_ context.Context, key content.Key) (content.LookupResult, error) {
	if s.wrong {
		return content.LookupFound{Entry: content.Monster{}}, nil
	}
	return content.LookupNotFound{Key: key}, nil
}
