package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// smMonster builds a synthetic monster with made-up numbers, not a stat block of SRD 5.2.1.
func smMonster(t *testing.T, id string, typ content.CreatureType, size content.CreatureSize, cr string) content.Monster {
	t.Helper()
	scores, saves := map[ability.Ability]ability.Score{}, map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	name := id
	return must(content.NewMonster(content.MonsterID(id), name, content.MustSourceCitation("Fixture", content.MonsterSection, 258), content.MonsterFields{
		Sizes: []content.CreatureSize{size}, Type: typ, Alignment: content.AlignmentUnaligned, ArmorClass: 12,
		Initiative: must(content.NewStatBlockInitiative(0, 10)), HitPoints: must(content.NewHitPointsEntry(13, dice.MustExpression(2, dice.D8(), 4))),
		Speeds:    []content.SpeedEntry{must(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, ""))},
		Abilities: must(content.NewAbilityBlock(must(ability.NewScores(scores)), saves)), PassivePerception: 10, ChallengeRating: content.MustChallengeRating(cr),
	}))
}

// smSource serves the skeleton and the cat of these scenarios.
func smSource(t *testing.T) *contenttest.FakeSource {
	t.Helper()
	return contenttest.NewFakeSource(smMonster(t, "skeleton", content.CreatureTypeUndead, content.SizeMedium, "1/4"), smMonster(t, "cat", content.CreatureTypeBeast, content.SizeTiny, "0"))
}

// Requirement: doc/specs/summoned-and-companion-creatures.md section 8 (scenario row), FR-001, FR-002, FR-010 to FR-013, FR-019, FR-077, Table P (active combatant drops; several summons, one summoner) - a wizard summons a Giant Insect and two Animated Objects in combat; the host joins them through AddCombatantToEncounter so that the creatures take their turns after the wizard in the order latest placed first (the insect, then the second object, then the first object); the wizard commands the insect; the insect, active on its turn, drops to 0 Hit Points: the host's removal is refused with RosterCombatantIsActiveRejection, it advances the turn and removes the insect, and the objects outlive their insect. All through the public API, each result being the next call's input.
// Path: happy.
func TestSummonedCreaturesScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	wizard, ogre, cleric := id("wizard"), id("ogre"), id("cleric")
	walk := must(combat.NewMovementSpeeds(30))
	profile := func(who combat.CombatantID, speed int) combat.CombatantProfile {
		return combat.CombatantProfile{ID: who, Size: content.SizeMedium, Speeds: must(combat.NewMovementSpeeds(speed))}
	}
	member := func(ref combat.CombatantRef, side combat.Side, hp int, rule combat.ZeroHitPointsRule) combat.Combatant {
		status := must(combat.RestoreHitPointStatus(combat.MustHitPoints(hp, hp, 0), combat.Alive{}, rule, combat.NoAffinities()))
		return must(combat.NewCombatant(ref, side, status, must(combat.NewConditionSet(nil, nil))))
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, who := range []combat.CombatantID{wizard, ogre, cleric} {
		standings = append(standings, combat.InitiativeStanding{Combatant: who, Count: combat.InitiativeCount([]int{18, 12, 5}[i])})
		profiles = append(profiles, combat.CombatantProfile{ID: who, Size: content.SizeMedium, Speeds: walk})
	}
	est := must(combat.NewEstablishInitiativeOrder().Run(ctx, must(combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil))))
	started := must(combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("summons"), est.(combat.InitiativeOrderEstablished), []combat.Combatant{
		member(combat.CombatantRef{ID: wizard, Kind: combat.MonsterCombatant{ID: "fixture"}}, combat.SideA, 20, combat.FallsUnconscious),
		member(combat.CombatantRef{ID: ogre, Kind: combat.MonsterCombatant{ID: "fixture"}}, combat.SideB, 40, combat.DiesInstantly),
		member(combat.CombatantRef{ID: cleric, Kind: combat.MonsterCombatant{ID: "fixture"}}, combat.SideA, 20, combat.FallsUnconscious)})))
	enc := started.(combat.StartEncounterApplied).Encounter()
	order := func(e combat.Encounter) []combat.CombatantID {
		var ids []combat.CombatantID
		for _, st := range e.Tracker().Order().Standings() {
			ids = append(ids, st.Combatant)
		}
		return ids
	}
	var ledger combat.ActionLedger
	advance := func() {
		t.Helper()
		done := must(combat.NewAdvanceCombatTurn().Run(ctx, must(combat.NewAdvanceCombatTurnCommand(enc, ledger, nil, nil)))).(combat.AdvanceCombatTurnCompleted)
		enc, ledger = done.Encounter(), done.Ledger()
	}
	apply := func(res combat.SummonCreatureApplied) {
		t.Helper()
		for i, act := range res.RosterActions() {
			join := act.(combat.SummonedRosterJoin)
			spec := res.Creatures()[i]
			if spec.Creature().ID != join.Creature {
				t.Fatalf("join %d is for %v, spec is %v", i, join.Creature, spec.Creature().ID)
			}
			speed := spec.StatBlock().Speed(content.SpeedWalk)
			add := must(combat.NewAddCombatantToEncounterCommand(enc, member(spec.Creature(), combat.SideA, spec.StatBlock().HitPointMaximum(), spec.StatBlock().ZeroHitPointsRule()),
				profile(join.Creature, speed), join.Position, join.Reason))
			enc = must(combat.NewAddCombatantToEncounter().Run(ctx, add)).(combat.AddCombatantToEncounterApplied).Encounter()
		}
	}

	facts := must(combat.NewSummonerSpellFacts(7, 15, 4, 3))
	summon := must(combat.NewSummonCreature(smSource(t)))
	cast := func(roster combat.SummonedRoster, request combat.SummonRequest) combat.SummonCreatureApplied {
		t.Helper()
		cmd := must(combat.NewSummonCreatureCommand(roster, wizard, combat.SideA, request, facts, combat.SummonScene{InCombat: true, Active: wizard}))
		return must(summon.Run(ctx, cmd)).(combat.SummonCreatureApplied)
	}
	roster := combat.SummonedRoster{}
	objects := cast(roster, combat.AnimateObjectsRequest{Level: 5, Objects: []combat.ObjectTarget{{Creature: id("x"), Size: content.SizeSmall}, {Creature: id("y"), Size: content.SizeSmall}}})
	apply(objects)
	roster = objects.Roster()
	if got := order(enc); !slices.Equal(got, []combat.CombatantID{wizard, id("y"), id("x"), ogre, cleric}) {
		t.Fatalf("order after two objects = %v, want wizard, y, x, ogre, cleric: X then Y placed after the wizard, the latest nearest it [Table P]", got)
	}
	insect := cast(roster, combat.GiantInsectRequest{Creature: id("wasp"), Level: 5, Form: combat.InsectWasp})
	apply(insect)
	roster = insect.Roster()
	if got := order(enc); !slices.Equal(got, []combat.CombatantID{wizard, id("wasp"), id("y"), id("x"), ogre, cleric}) {
		t.Fatalf("order after the insect = %v, want the wasp directly after the wizard [FR-002]", got)
	}

	// The wizard commands the wasp by voice; the order costs no action.
	order1 := combat.IssueCommand{Summoner: wizard, Kind: combat.SummonGiantInsect, Targets: []combat.CommandTarget{{Creature: id("wasp"), DistanceFeet: 40}}, Order: combat.GeneralOrder{Label: "harry_the_ogre"}, CanGiveVerbalCommand: true}
	changed := must(combat.NewApplySummonedChange().Run(ctx, must(combat.NewApplySummonedChangeCommand(roster, order1, wizard))))
	roster = changed.(combat.ApplySummonedChangeApplied).Roster()
	if plan, ok := roster.TurnPlan(id("wasp")); !ok || plan != (combat.TurnPlanOrdered{Order: combat.GeneralOrder{Label: "harry_the_ogre"}}) {
		t.Fatalf("the wasp's plan = %v, %v, want the order", plan, ok)
	}

	// The wasp's turn: it drops to 0 Hit Points on it. Its departure is pending, so removing it now is refused.
	advance()
	if enc.Tracker().Active().Combatant != id("wasp") {
		t.Fatalf("active = %v, want the wasp", enc.Tracker().Active().Combatant)
	}
	dropped := must(combat.NewApplySummonedChange().Run(ctx, must(combat.NewApplySummonedChangeCommand(roster, combat.ReportZeroHitPoints{Creature: id("wasp")}, id("wasp")))))
	gone := dropped.(combat.ApplySummonedChangeApplied)
	leave := gone.RosterActions()[0].(combat.SummonedRosterLeave)
	if ev := gone.Events()[0].(combat.SummonedCreatureDeparted); !ev.PendingRemoval() || !leave.AfterTurn || leave.Cause != combat.RemovalCauseDisappeared {
		t.Fatalf("departure = %#v, leave %#v, want a pending removal after the turn, Disappeared", ev, leave)
	}
	remove := must(combat.NewRemoveCombatantFromEncounterCommand(enc, ledger, leave.Creature, leave.Cause))
	refused := must(combat.NewRemoveCombatantFromEncounter().Run(ctx, remove))
	if r, ok := refused.(combat.RemoveCombatantFromEncounterRejected); !ok || r.Rejection != (combat.RosterCombatantIsActiveRejection{ID: id("wasp")}) {
		t.Fatalf("removing the active wasp = %#v, want RosterCombatantIsActiveRejection [FR-077]", refused)
	}
	advance()
	remove = must(combat.NewRemoveCombatantFromEncounterCommand(enc, ledger, leave.Creature, leave.Cause))
	enc = must(combat.NewRemoveCombatantFromEncounter().Run(ctx, remove)).(combat.RemoveCombatantFromEncounterApplied).Encounter()
	roster = gone.Roster()
	if got := order(enc); !slices.Equal(got, []combat.CombatantID{wizard, id("y"), id("x"), ogre, cleric}) || roster.Len() != 2 {
		t.Fatalf("order = %v, links %d, want the objects to outlive the wasp", got, roster.Len())
	}

	// The spell that animated the objects ends: they depart and revert to objects.
	ended := must(combat.NewApplySummonedChange().Run(ctx, must(combat.NewApplySummonedChangeCommand(roster, combat.ReportSpellEnded{Summoner: wizard, Spell: "animate_objects", Cause: combat.SummonSpellEndConcentrationEnded}, ""))))
	if got := len(ended.(combat.ApplySummonedChangeApplied).RosterActions()); got != 2 {
		t.Errorf("the end of Animate Objects gave %d leaves, want 2", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md section 8 (scenario row), FR-015, FR-040, FR-041 - a wizard raises a Skeleton with Animate Dead; 23 hours later it reasserts control, which starts 24 hours again; 23 more hours leave it controlled; 24 hours in all lapse the window, an order is refused with ControlLapsedRejection and the creature stays linked; reasserting a lapsed creature is refused too.
// Path: happy.
func TestUndeadControlLapsesUnlessReasserted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	wizard := id("wizard")
	facts := must(combat.NewSummonerSpellFacts(7, 15, 4, 3))
	summon := must(combat.NewSummonCreature(smSource(t)))
	cmd := must(combat.NewSummonCreatureCommand(combat.SummonedRoster{}, wizard, combat.SideA, combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{
		{Creature: id("bones"), Remains: "pile-1", Kind: combat.RemainsBones, Size: content.SizeMedium}}}, facts, combat.SummonScene{}))
	roster := must(summon.Run(ctx, cmd)).(combat.SummonCreatureApplied).Roster()
	change := func(c combat.SummonedChange) combat.ApplySummonedChangeResult {
		t.Helper()
		return must(combat.NewApplySummonedChange().Run(ctx, must(combat.NewApplySummonedChangeCommand(roster, c, ""))))
	}
	elapse := func(d time.Duration) combat.ApplySummonedChangeResult {
		t.Helper()
		res := change(combat.ElapseControlTime{Elapsed: d})
		roster = res.(combat.ApplySummonedChangeApplied).Roster()
		return res
	}
	elapse(23 * time.Hour)
	order := combat.IssueCommand{Summoner: wizard, Kind: combat.SummonAnimatedUndead, Targets: []combat.CommandTarget{{Creature: id("bones"), DistanceFeet: 60}}, Order: combat.GeneralOrder{Label: "guard"}}
	if _, ok := change(order).(combat.ApplySummonedChangeApplied); !ok {
		t.Fatalf("an order with an hour left was refused")
	}
	reassert := must(combat.NewSummonCreatureCommand(roster, wizard, combat.SideA, combat.ReassertControlRequest{Kind: combat.SummonAnimatedUndead, Level: 3, Creatures: []combat.CombatantID{id("bones")}}, facts, combat.SummonScene{}))
	roster = must(summon.Run(ctx, reassert)).(combat.SummonCreatureApplied).Roster()
	elapse(23 * time.Hour)
	if _, ok := change(order).(combat.ApplySummonedChangeApplied); !ok {
		t.Fatalf("an order after the reassert was refused")
	}
	lapsed := elapse(time.Hour).(combat.ApplySummonedChangeApplied)
	if kinds := lapsed.Events(); len(kinds) != 1 || kinds[0].Kind() != "summon_control_lapsed" {
		t.Fatalf("events = %v, want the lapse", kinds)
	}
	refused := change(order).(combat.ApplySummonedChangeRejected)
	if refused.Rejection != (combat.ControlLapsedRejection{Creature: id("bones")}) || roster.Len() != 1 {
		t.Errorf("order after the lapse = %#v with %d links, want ControlLapsedRejection and the creature still linked", refused.Rejection, roster.Len())
	}
	late := must(combat.NewSummonCreatureCommand(roster, wizard, combat.SideA, combat.ReassertControlRequest{Kind: combat.SummonAnimatedUndead, Level: 3, Creatures: []combat.CombatantID{id("bones")}}, facts, combat.SummonScene{}))
	if r, ok := must(summon.Run(ctx, late)).(combat.SummonCreatureRejected); !ok || r.Rejection != (combat.ControlLapsedRejection{Creature: id("bones")}) {
		t.Errorf("a reassert of a lapsed creature = %#v, want ControlLapsedRejection", r)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md section 8 (scenario row), FR-044, FR-048, FR-050, FR-051 - a Warlock's Pact of the Chain familiar is cast in combat at the Initiative the host rolled; the Attack action is refused for it and allowed with the grant of Investment of the Chain Master; the Warlock sends it to its pocket dimension and recalls it, and it rejoins at the same Initiative; a second cast changes its form; Wild Companion's casting is the same familiar with another origin and a Long Rest departs it.
// Path: happy.
func TestAFamiliarIsDismissedAndRecalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	warlock := id("warlock")
	summon := must(combat.NewSummonCreature(smSource(t)))
	at := combat.RosterAtInitiative{Count: 14, Tie: combat.TiePlacementAfterTied}
	cast := combat.FindFamiliarRequest{Creature: id("imp"), Level: 1, Form: character.NormalFamiliarForm("cat"), Type: content.CreatureTypeFiend, Origin: combat.FamiliarOriginPactOfTheChain}
	cmd := must(combat.NewSummonCreatureCommand(combat.SummonedRoster{}, warlock, combat.SideA, cast, combat.SummonerSpellFacts{}, combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{at}}))
	made := must(summon.Run(ctx, cmd)).(combat.SummonCreatureApplied)
	if got := made.RosterActions(); len(got) != 1 || got[0] != (combat.SummonedRosterJoin{Creature: id("imp"), Position: at, Reason: combat.JoinReasonSummoned}) {
		t.Fatalf("actions = %v, want a join at the Initiative rolled", got)
	}
	roster := made.Roster()
	do := func(c combat.SummonedChange) combat.ApplySummonedChangeResult {
		t.Helper()
		res := must(combat.NewApplySummonedChange().Run(ctx, must(combat.NewApplySummonedChangeCommand(roster, c, ""))))
		if a, ok := res.(combat.ApplySummonedChangeApplied); ok {
			roster = a.Roster()
		}
		return res
	}
	if r, ok := do(combat.RequestFamiliarAction{Creature: id("imp"), Action: combat.StandardActionAttack}).(combat.ApplySummonedChangeRejected); !ok || r.Rejection != (combat.FamiliarCannotAttackRejection{}) {
		t.Errorf("the Attack action = %#v, want FamiliarCannotAttackRejection [FR-048]", r)
	}
	if _, ok := do(combat.RequestFamiliarAction{Creature: id("imp"), Action: combat.StandardActionAttack, Grant: combat.FamiliarAttackViaBonusActionCommand}).(combat.ApplySummonedChangeApplied); !ok {
		t.Errorf("the Attack action with a grant was refused [FR-048]")
	}
	pocket := do(combat.DismissFamiliar{Summoner: warlock, How: combat.DismissalTemporary}).(combat.ApplySummonedChangeApplied)
	if got := pocket.RosterActions(); len(got) != 1 || got[0] != (combat.SummonedRosterLeave{Creature: id("imp"), Cause: combat.RemovalCauseDeparted}) {
		t.Fatalf("dismissal actions = %v, want a leave with RemovalCauseDeparted [FR-050]", got)
	}
	if r, ok := do(combat.DismissFamiliar{Summoner: warlock, How: combat.DismissalTemporary}).(combat.ApplySummonedChangeRejected); !ok || r.Rejection != (combat.FamiliarAlreadyDismissedRejection{}) {
		t.Errorf("a second dismissal = %#v, want FamiliarAlreadyDismissedRejection", r)
	}
	back := do(combat.RecallFamiliar{Summoner: warlock, SpaceAvailable: true, InCombat: true, Initiative: combat.RosterAtInitiative{Count: 3, Tie: combat.TiePlacementBeforeTied}}).(combat.ApplySummonedChangeApplied)
	if got := back.RosterActions(); len(got) != 1 || got[0] != (combat.SummonedRosterJoin{Creature: id("imp"), Position: at, Reason: combat.JoinReasonSummoned}) {
		t.Fatalf("recall actions = %v, want a join at the Initiative it had [FR-051]", got)
	}
	recast := must(combat.NewSummonCreatureCommand(roster, warlock, combat.SideA, combat.FindFamiliarRequest{Creature: id("other"), Level: 1, Form: character.NormalFamiliarForm("cat"),
		Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginWildCompanion}, combat.SummonerSpellFacts{}, combat.SummonScene{InCombat: true}))
	changedForm := must(summon.Run(ctx, recast)).(combat.SummonCreatureApplied)
	if got := changedForm.Events(); len(got) != 1 || got[0].Kind() != "familiar_form_changed" || len(changedForm.RosterActions()) != 0 {
		t.Fatalf("events = %v, want only a form change [FR-044]", got)
	}
	roster = changedForm.Roster()
	rest := do(combat.ReportLongRestFinished{Summoner: warlock}).(combat.ApplySummonedChangeApplied)
	if got := rest.RosterActions(); len(got) != 1 || got[0] != (combat.SummonedRosterLeave{Creature: id("imp"), Cause: combat.RemovalCauseDeparted}) || roster.Len() != 0 {
		t.Errorf("Long Rest actions = %v, links %d, want the Wild Companion familiar gone [FR-053]", got, roster.Len())
	}
	if _, err := combat.NewApplySummonedChangeCommand(roster, combat.ElapseControlTime{Elapsed: -time.Second}, ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative time error = %v, want invalid input", err)
	}
}
