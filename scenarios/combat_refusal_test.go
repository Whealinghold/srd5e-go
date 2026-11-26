package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/encounter-aggregate.md FR-018, FR-030, FR-042 and doc/specs/conditions.md FR-029 (SRD 5.2.1,
// "Conditions", p. 180; "Dropping to 0 Hit Points", p. 17) - a conditioned combatant through the public encounter use cases: a
// condition is applied once, a repeat, an unknown id and a dead target are each refused with the encounter returned unchanged and
// no event, and the next legal command (healing then removing the condition) is accepted on that same encounter value. The
// recording telemetry sees every refusal as a normal outcome, never as a port error.
// Path: unhappy (state error), then happy.
func TestRefusedCommandsLeaveTheEncounterAndALegalOneFollows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec := telemetrytest.NewRecorder()
	opts := []combat.Option{combat.WithTelemetry(rec.Telemetry())}
	id := combat.MustCombatantID
	hero, goblin := id("hero"), id("goblin")
	speed, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: hero, Count: 15}, {Combatant: goblin, Count: 10}},
		[]combat.CombatantProfile{{ID: hero, Size: content.SizeMedium, Speeds: speed}, {ID: goblin, Size: content.SizeSmall, Speeds: speed}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	member := func(c combat.CombatantID, side combat.Side, hp int) combat.Combatant {
		s, err := combat.NewHitPointStatus(hp, combat.DiesInstantly, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: c, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, s, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	started, err := combat.NewStartEncounter(opts...).Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("scenario-refusal"),
		order.(combat.InitiativeOrderEstablished), []combat.Combatant{member(hero, combat.SideA, 20), member(goblin, combat.SideB, 5)}))
	if err != nil {
		t.Fatal(err)
	}
	e := started.(combat.StartEncounterApplied).Encounter()

	apply := func(target combat.CombatantID, c combat.Condition) combat.ApplyConditionToCombatantResult {
		cmd, err := combat.NewApplyConditionToCombatantCommand(e, target, c)
		if err != nil {
			t.Fatal(err)
		}
		r, err := combat.NewApplyConditionToCombatant(opts...).Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Legal: the hero is Poisoned.
	first, ok := apply(hero, combat.Poisoned{}).(combat.ApplyConditionToCombatantApplied)
	if !ok || !slices.Equal(kinds(first.Events()), []string{"condition_applied"}) {
		t.Fatalf("first Poisoned = %#v, want applied with one condition_applied", first)
	}
	e = first.Encounter()

	// Refused: a repeat, an unknown id.
	before := e
	if r, ok := apply(hero, combat.Poisoned{}).(combat.ApplyConditionToCombatantRejected); !ok || !e.Equal(before) || !r.Encounter().Equal(before) {
		t.Errorf("a repeated Poisoned must be refused and leave the encounter unchanged, got %#v", r)
	} else if _, ok := r.Rejection.(combat.AlreadyHasConditionRejection); !ok {
		t.Errorf("repeat rejection = %#v, want AlreadyHasConditionRejection", r.Rejection)
	}
	if r, ok := apply(id("nobody"), combat.Prone{}).(combat.ApplyConditionToCombatantRejected); !ok || !r.Encounter().Equal(before) {
		t.Errorf("an unknown combatant must be refused, got %#v", r)
	} else if _, ok := r.Rejection.(combat.UnknownCombatantRejection); !ok {
		t.Errorf("unknown rejection = %#v, want UnknownCombatantRejection", r.Rejection)
	}

	// The goblin dies; a condition on a dead combatant is refused too.
	dmg, err := combat.NewApplyDamageToCombatantCommand(e, goblin, []combat.Damage{combat.MustDamage(9, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	killed, err := combat.NewApplyDamageToCombatant(opts...).Run(ctx, dmg)
	if err != nil {
		t.Fatal(err)
	}
	e = killed.(combat.ApplyDamageToCombatantApplied).Encounter()
	before = e
	if r, ok := apply(goblin, combat.Prone{}).(combat.ApplyConditionToCombatantRejected); !ok || !r.Encounter().Equal(before) {
		t.Errorf("a condition on a dead combatant must be refused, got %#v", r)
	} else if _, ok := r.Rejection.(combat.CombatantIsDeadRejection); !ok {
		t.Errorf("dead rejection = %#v, want CombatantIsDeadRejection", r.Rejection)
	}

	// Legal again, on the encounter the refusals returned: the condition ends.
	rm, err := combat.NewRemoveConditionFromCombatantCommand(e, hero, content.ConditionPoisoned)
	if err != nil {
		t.Fatal(err)
	}
	removed, err := combat.NewRemoveConditionFromCombatant(opts...).Run(ctx, rm)
	if err != nil {
		t.Fatal(err)
	}
	ra, ok := removed.(combat.RemoveConditionFromCombatantApplied)
	if !ok || !slices.Equal(kinds(ra.Events()), []string{"condition_removed"}) {
		t.Fatalf("remove Poisoned = %#v, want applied with one condition_removed", removed)
	}
	if m, _ := ra.Encounter().Find(hero); slices.Contains(m.Conditions().Effective(), content.ConditionPoisoned) {
		t.Errorf("the hero is still Poisoned: %v", m.Conditions().Effective())
	}
	for _, m := range rec.Meter.Measurements() {
		if m.Name == "srd5e.combat.errors" {
			t.Errorf("a refusal was counted as an error: %+v", m)
		}
	}
}
