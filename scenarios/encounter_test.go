package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/encounter-aggregate.md section 8 (scenario test), FR-007, FR-018, FR-023, FR-073, FR-030, FR-042, FR-086, FR-093 - four combatants start; two rounds advance; a player character goes to 0 Hit Points (Unconscious, then DyingCombatantTurnStarted at its next turn), makes a Death Saving Throw and is healed; a monster dies; a grapple started by the host's ApplyCondition is ended by ReleaseGrapple; the encounter ends with EndReasonSideDefeated. Public packages only.
// Path: happy.
func TestEncounterScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	hero, ally, goblin, ogre := id("hero"), id("ally"), id("goblin"), id("ogre")
	speed, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	order := []combat.CombatantID{hero, goblin, ally, ogre}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, c := range order {
		standings = append(standings, combat.InitiativeStanding{Combatant: c, Count: combat.InitiativeCount(20 - 2*i)})
		profiles = append(profiles, combat.CombatantProfile{ID: c, Size: content.SizeMedium, Speeds: speed})
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	established := res.(combat.InitiativeOrderEstablished)
	member := func(c combat.CombatantID, side combat.Side, kind combat.CombatantKind, hp int, rule combat.ZeroHitPointsRule) combat.Combatant {
		s, err := combat.NewHitPointStatus(hp, rule, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: c, Kind: kind}, side, s, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	start := combat.NewStartEncounterCommand(combat.MustEncounterID("scenario"), established, []combat.Combatant{
		member(ogre, combat.SideB, combat.MonsterCombatant{ID: "ogre"}, 59, combat.DiesInstantly),
		member(ally, combat.SideA, combat.MonsterCombatant{ID: "ally"}, 12, combat.FallsUnconscious),
		member(goblin, combat.SideB, combat.MonsterCombatant{ID: "goblin"}, 7, combat.DiesInstantly),
		member(hero, combat.SideA, combat.MonsterCombatant{ID: "hero"}, 6, combat.FallsUnconscious),
	})
	started, err := combat.NewStartEncounter().Run(ctx, start)
	if err != nil {
		t.Fatal(err)
	}
	e := started.(combat.StartEncounterApplied).Encounter()

	advance := func() []combat.Event {
		cmd, err := combat.NewAdvanceEncounterTurnCommand(e, combat.TurnAdvance{})
		if err != nil {
			t.Fatal(err)
		}
		r, err := combat.NewAdvanceEncounterTurn().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		a := r.(combat.AdvanceEncounterTurnApplied)
		e = a.Encounter()
		return a.Events()
	}
	for range 4 { // round 1 ends: back to the hero in round 2
		advance()
	}
	if e.Tracker().Round() != 2 || e.Tracker().Active().Combatant != hero {
		t.Fatalf("round %d active %v, want round 2 with the hero", e.Tracker().Round(), e.Tracker().Active().Combatant)
	}
	damage := func(target combat.CombatantID, n int) combat.ApplyDamageToCombatantApplied {
		cmd, err := combat.NewApplyDamageToCombatantCommand(e, target, []combat.Damage{combat.MustDamage(n, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
		if err != nil {
			t.Fatal(err)
		}
		r, err := combat.NewApplyDamageToCombatant().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		a := r.(combat.ApplyDamageToCombatantApplied)
		e = a.Encounter()
		return a
	}
	// The ogre drops the hero to 0: Dying and Unconscious.
	hit := damage(hero, 6)
	if !slices.Contains(kinds(hit.Events()), "condition_applied") {
		t.Errorf("events = %v, want Unconscious applied", kinds(hit.Events()))
	}
	// Round 2 continues; at the hero's next turn the encounter says a Death Saving Throw is due.
	var due []combat.Event
	for range 4 {
		due = advance()
	}
	if got := kinds(due); got[len(got)-1] != "dying_combatant_turn_started" {
		t.Fatalf("events at the hero's turn = %v, want DyingCombatantTurnStarted last", got)
	}
	save, err := combat.NewApplyDeathSavingThrowToCombatantCommand(e, hero, 14, 14)
	if err != nil {
		t.Fatal(err)
	}
	sr, err := combat.NewApplyDeathSavingThrowToCombatant().Run(ctx, save)
	if err != nil {
		t.Fatal(err)
	}
	e = sr.(combat.ApplyDeathSavingThrowToCombatantApplied).Encounter()
	again, _ := combat.NewApplyDeathSavingThrowToCombatantCommand(e, hero, 14, 14)
	if r, _ := combat.NewApplyDeathSavingThrowToCombatant().Run(ctx, again); !isRejected(r) {
		t.Errorf("a second throw in the turn must be refused")
	}
	heal, _ := combat.NewApplyHealingToCombatantCommand(e, hero, 5)
	hr, err := combat.NewApplyHealingToCombatant().Run(ctx, heal)
	if err != nil {
		t.Fatal(err)
	}
	e = hr.(combat.ApplyHealingToCombatantApplied).Encounter()
	if m, _ := e.Find(hero); m.HitPoints().HitPoints().Current() != 5 || slices.Contains(m.Conditions().Effective(), content.ConditionUnconscious) {
		t.Errorf("the hero is %v, want 5 Hit Points and awake", m.HitPoints())
	}
	// A monster dies: one Died wrapped, the death recorded.
	kill := damage(goblin, 9)
	if got := kinds(kill.Events()); !slices.Contains(got, "died") {
		t.Errorf("events = %v, want Died", got)
	}
	if m, _ := e.Find(goblin); !hasDeath(m) {
		t.Errorf("the goblin has no recorded death")
	}
	// The host starts a grapple with ApplyCondition and ends it with ReleaseGrapple.
	grab, _ := combat.NewApplyConditionToCombatantCommand(e, ally, combat.Grappled{Grappler: ogre})
	gr, err := combat.NewApplyConditionToCombatant().Run(ctx, grab)
	if err != nil {
		t.Fatal(err)
	}
	e = gr.(combat.ApplyConditionToCombatantApplied).Encounter()
	rel, _ := combat.NewReleaseCombatantGrappleCommand(e, ally)
	rr, err := combat.NewReleaseCombatantGrapple().Run(ctx, rel)
	if err != nil {
		t.Fatal(err)
	}
	if got := kinds(rr.(combat.ReleaseCombatantGrappleApplied).Events()); !slices.Equal(got, []string{"condition_removed", "grapple_ended"}) {
		t.Errorf("release events = %v", got)
	}
	e = rr.(combat.ReleaseCombatantGrappleApplied).Encounter()
	esc, _ := combat.NewEscapeCombatantGrappleCommand(e, ally, d20.Success)
	if r, _ := combat.NewEscapeCombatantGrapple().Run(ctx, esc); !isRejected(r) {
		t.Errorf("escaping a grapple that already ended must be refused")
	}
	// The encounter does not end by itself; the host ends it with the side it counts defeated.
	endCmd, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		t.Fatal(err)
	}
	er, err := combat.NewEndEncounter().Run(ctx, endCmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := er.(combat.EndEncounterApplied).Encounter().Status().(combat.EncounterEndedStatus); !ok {
		t.Errorf("status = %#v, want ended", er.(combat.EndEncounterApplied).Encounter().Status())
	}
}

func kinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

func hasDeath(c combat.Combatant) bool { _, ok := c.Death(); return ok }

// isRejected reports whether a use case result is a Rejected variant, by its name.
func isRejected(r any) bool {
	switch r.(type) {
	case combat.ApplyDeathSavingThrowToCombatantRejected, combat.EscapeCombatantGrappleRejected:
		return true
	}
	return false
}
