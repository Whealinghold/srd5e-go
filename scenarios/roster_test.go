package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/combatant-roster-changes.md section 8 (scenario test), FR-008, FR-013, FR-016, FR-019, FR-020, FR-021, FR-025 - four combatants start; the summoner's creature is added after it and acts after it in round 1; a latecomer joins before the active combatant and first acts in round 2; a combatant that holds a grapple leaves (the grapple ends with GrappleEnded) and its Help benefit is dropped from the ledger; the last combatant of a side cannot leave and the host ends the encounter. All through the public packages, each result's encounter and ledger being the next call's input.
// Path: happy.
func TestCombatantRosterScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	a, b, c, d, s, l := id("a"), id("b"), id("c"), id("d"), id("summon"), id("latecomer")
	walk, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	profile := func(who combat.CombatantID) combat.CombatantProfile {
		return combat.CombatantProfile{ID: who, Size: content.SizeMedium, Speeds: walk}
	}
	member := func(who combat.CombatantID, side combat.Side, conds ...combat.Condition) combat.Combatant {
		hp, err := combat.RestoreHitPointStatus(combat.MustHitPoints(10, 10, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		set, err := combat.NewConditionSet(conds, nil)
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: who, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, hp, set)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, who := range []combat.CombatantID{a, b, c, d} {
		standings = append(standings, combat.InitiativeStanding{Combatant: who, Count: combat.InitiativeCount([]int{18, 14, 10, 6}[i])})
		profiles = append(profiles, profile(who))
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	r0, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("roster-scenario"),
		r0.(combat.InitiativeOrderEstablished), []combat.Combatant{
			member(a, combat.SideA), member(b, combat.SideB), member(c, combat.SideA), member(d, combat.SideB, combat.Grappled{Grappler: c})}))
	if err != nil {
		t.Fatal(err)
	}
	e := started.(combat.StartEncounterApplied).Encounter()

	helpC, err := combat.NewAbilityCheckHelp(c, a, combat.AssistedSkill(content.SkillAthletics))
	if err != nil {
		t.Fatal(err)
	}
	helps, err := combat.NewHelpBenefits([]combat.HelpBenefit{helpC})
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := combat.NewActionLedger(nil, helps)
	if err != nil {
		t.Fatal(err)
	}

	order := func(e combat.Encounter) []combat.CombatantID {
		var ids []combat.CombatantID
		for _, st := range e.Tracker().Order().Standings() {
			ids = append(ids, st.Combatant)
		}
		return ids
	}
	// turns advances n times from a copy of the state and lists whose turn started, with its round.
	turns := func(e combat.Encounter, l combat.ActionLedger, n int) (string, combat.Encounter, combat.ActionLedger) {
		t.Helper()
		var out string
		for range n {
			cmd, err := combat.NewAdvanceCombatTurnCommand(e, l, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			res, err := combat.NewAdvanceCombatTurn().Run(ctx, cmd)
			if err != nil {
				t.Fatal(err)
			}
			done := res.(combat.AdvanceCombatTurnCompleted)
			e, l = done.Encounter(), done.Ledger()
			out += string(e.Tracker().Active().Combatant) + "@" + string(rune('0'+e.Tracker().Round())) + " "
		}
		return out, e, l
	}

	// The summoned creature is placed directly after its summoner and acts after it in round 1 (FR-013).
	add, err := combat.NewAddCombatantToEncounterCommand(e, member(s, combat.SideA), profile(s), combat.RosterAfterCombatant{Anchor: a}, combat.JoinReasonSummoned)
	if err != nil {
		t.Fatal(err)
	}
	added, err := combat.NewAddCombatantToEncounter().Run(ctx, add)
	if err != nil {
		t.Fatal(err)
	}
	e = added.(combat.AddCombatantToEncounterApplied).Encounter()
	if got := order(e); !slices.Equal(got, []combat.CombatantID{a, s, b, c, d}) || e.Tracker().Active().Combatant != a {
		t.Fatalf("order after the summon = %v, active %v, want a, summon, b, c, d with a active", got, e.Tracker().Active().Combatant)
	}
	var seq string
	seq, e, ledger = turns(e, ledger, 1)
	if seq != "summon@1 " {
		t.Fatalf("the next turn = %q, want the summoned creature in round 1", seq)
	}

	// A latecomer rolled 20 and joins before the active combatant: it acts first in round 2 (FR-006).
	add, err = combat.NewAddCombatantToEncounterCommand(e, member(l, combat.SideB), profile(l), combat.RosterAtInitiative{Count: 20, Tie: combat.TiePlacementAfterTied}, combat.JoinReasonArrived)
	if err != nil {
		t.Fatal(err)
	}
	added, err = combat.NewAddCombatantToEncounter().Run(ctx, add)
	if err != nil {
		t.Fatal(err)
	}
	e = added.(combat.AddCombatantToEncounterApplied).Encounter()
	if e.Tracker().Active().Combatant != s || e.Tracker().Round() != 1 {
		t.Fatalf("active %v in round %d, want the summoned creature still active in round 1", e.Tracker().Active().Combatant, e.Tracker().Round())
	}
	if seq, _, _ = turns(e, ledger, 7); seq != "b@1 c@1 d@1 latecomer@2 a@2 summon@2 b@2 " {
		t.Errorf("turns = %q, want the latecomer to start round 2", seq)
	}

	// The combatant that holds a grapple leaves: the grapple ends and its Help benefit is dropped (FR-021, FR-025).
	_, e, ledger = turns(e, ledger, 1) // b is active
	remove, err := combat.NewRemoveCombatantFromEncounterCommand(e, ledger, c, combat.RemovalCauseFled)
	if err != nil {
		t.Fatal(err)
	}
	removed, err := combat.NewRemoveCombatantFromEncounter().Run(ctx, remove)
	if err != nil {
		t.Fatal(err)
	}
	gone := removed.(combat.RemoveCombatantFromEncounterApplied)
	if got := kinds(gone.Events()); !slices.Equal(got, []string{"condition_removed", "grapple_ended", "combatant_removed"}) {
		t.Errorf("events = %v, want the grapple ended before combatant_removed", got)
	}
	if ended := gone.Events()[1].(combat.GrappleEventOccurred).Event().(combat.GrappleEnded); ended.Cause() != combat.GrappleEndCauseOutOfRange {
		t.Errorf("the grapple ended with %v, want out of range", ended.Cause())
	}
	if len(gone.LedgerEvents()) != 1 || gone.LedgerEvents()[0].Kind() != "help_benefit_dropped" || !gone.Ledger().HelpBenefits().Equal(combat.HelpBenefits{}) {
		t.Errorf("ledger events %v, benefits %v, want the one benefit of the leaver dropped", gone.LedgerEvents(), gone.Ledger().HelpBenefits())
	}
	e, ledger = gone.Encounter(), gone.Ledger()
	if _, ok := e.Find(c); ok {
		t.Fatalf("c is still a combatant")
	}

	// The summoned creature disappears; the last combatant of side A cannot leave, so the host ends the encounter (FR-019, FR-020).
	step := func(who combat.CombatantID) combat.RemoveCombatantFromEncounterResult {
		t.Helper()
		cmd, err := combat.NewRemoveCombatantFromEncounterCommand(e, ledger, who, combat.RemovalCauseDisappeared)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRemoveCombatantFromEncounter().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	e = step(s).(combat.RemoveCombatantFromEncounterApplied).Encounter()
	refused, ok := step(a).(combat.RemoveCombatantFromEncounterRejected)
	if !ok || refused.Rejection != (combat.RosterWouldEmptySideRejection{Side: combat.SideA}) || !refused.Encounter().Equal(e) {
		t.Fatalf("removing the last of side A = %#v, want would_empty_side with the encounter unchanged", refused)
	}
	if _, inProgress := e.Status().(combat.EncounterInProgress); !inProgress {
		t.Fatalf("the encounter ended by itself")
	}
	end, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideA})
	if err != nil {
		t.Fatal(err)
	}
	over, err := combat.NewEndEncounter().Run(ctx, end)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := over.(combat.EndEncounterApplied); !ok {
		t.Errorf("end = %#v, want applied", over)
	}
}
