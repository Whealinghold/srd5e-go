package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-010, FR-015, FR-040 (SRD 5.2.1, "Animate Dead") - Animate Dead cast with a level 2 slot is refused with LevelBelowSpellRejection and leaves the roster empty; cast at level 3 it links the Skeleton; an order from another wizard is refused (CommandNotYourCreature) and one beyond 60 feet is refused (CommandOutOfRange); both refusals leave the roster as it came in.
// Path: unhappy (boundary, state error), then happy.
func TestAnimateDeadRefusalsLeaveTheRosterUnchanged(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	wizard, rival, bones := id("wizard"), id("rival"), id("bones")
	facts := must(combat.NewSummonerSpellFacts(7, 15, 4, 3))
	summon := must(combat.NewSummonCreature(smSource(t)))
	cast := func(level int) combat.SummonCreatureResult {
		t.Helper()
		return must(summon.Run(ctx, must(combat.NewSummonCreatureCommand(combat.SummonedRoster{}, wizard, combat.SideA, combat.AnimateDeadRequest{Level: level, Targets: []combat.UndeadTarget{
			{Creature: bones, Remains: "pile-1", Kind: combat.RemainsBones, Size: content.SizeMedium}}}, facts, combat.SummonScene{}))))
	}
	low, ok := cast(2).(combat.SummonCreatureRejected)
	if !ok || low.Rejection != (combat.LevelBelowSpellRejection{Minimum: 3}) || low.Roster().Len() != 0 {
		t.Fatalf("a level 2 cast = %#v, want LevelBelowSpellRejection{3} and no links", low)
	}
	roster := cast(3).(combat.SummonCreatureApplied).Roster()
	if roster.Len() != 1 {
		t.Fatalf("a level 3 cast linked %d creatures, want 1", roster.Len())
	}
	order := func(who combat.CombatantID, feet int) combat.ApplySummonedChangeResult {
		t.Helper()
		return must(combat.NewApplySummonedChange().Run(ctx, must(combat.NewApplySummonedChangeCommand(roster, combat.IssueCommand{Summoner: who, Kind: combat.SummonAnimatedUndead,
			Targets: []combat.CommandTarget{{Creature: bones, DistanceFeet: feet}}, Order: combat.GeneralOrder{Label: "guard"}}, ""))))
	}
	if r, ok := order(rival, 30).(combat.ApplySummonedChangeRejected); !ok || r.Rejection != (combat.CommandNotYourCreatureRejection{Creature: bones}) {
		t.Errorf("an order from another wizard = %#v, want CommandNotYourCreatureRejection", r)
	}
	if r, ok := order(wizard, 61).(combat.ApplySummonedChangeRejected); !ok {
		t.Errorf("an order from 61 feet = %#v, want a CommandOutOfRange rejection", r)
	} else if _, isRange := r.Rejection.(combat.CommandOutOfRangeRejection); !isRange {
		t.Errorf("rejection = %#v, want CommandOutOfRangeRejection", r.Rejection)
	}
	if _, ok := order(wizard, 60).(combat.ApplySummonedChangeApplied); !ok {
		t.Error("an order from exactly 60 feet must be accepted")
	}
	if roster.Len() != 1 {
		t.Errorf("the roster has %d links after the refusals, want 1", roster.Len())
	}
}
