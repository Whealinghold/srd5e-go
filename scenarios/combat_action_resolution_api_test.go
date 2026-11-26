package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// resolutionVariant names the variant of a result of the eight use cases of combat action resolution with a type switch that has a
// case for every variant and no default arm.
func resolutionVariant(r any) string {
	switch r.(type) {
	case combat.PerformAttackCompleted, combat.PerformUnarmedStrikeCompleted, combat.TakeCombatantActionCompleted, combat.MoveCombatantCompleted,
		combat.AdvanceCombatTurnCompleted, combat.RollCombatantDeathSavingThrowCompleted, combat.AttemptToStabilizeCombatantCompleted, combat.DealDamageToCombatantCompleted:
		return "completed"
	case combat.PerformAttackRejected, combat.PerformUnarmedStrikeRejected, combat.TakeCombatantActionRejected, combat.MoveCombatantRejected,
		combat.AdvanceCombatTurnRejected, combat.RollCombatantDeathSavingThrowRejected, combat.AttemptToStabilizeCombatantRejected, combat.DealDamageToCombatantRejected:
		return "rejected"
	case combat.PerformAttackInvalid, combat.PerformUnarmedStrikeInvalid, combat.TakeCombatantActionInvalid, combat.MoveCombatantInvalid,
		combat.AdvanceCombatTurnInvalid, combat.RollCombatantDeathSavingThrowInvalid, combat.AttemptToStabilizeCombatantInvalid, combat.DealDamageToCombatantInvalid:
		return "invalid"
	}
	return "unknown"
}

// resolutionEventName names a ResolutionEvent with a type switch that covers every event of the family.
func resolutionEventName(e combat.ResolutionEvent) string {
	switch e.(type) {
	case combat.AttackPerformed:
		return "attack_performed"
	case combat.UnarmedStrikePerformed:
		return "unarmed_strike_performed"
	case combat.MovementSpeedsReviewBecameDue:
		return "movement_speeds_review_became_due"
	case combat.GrappleCheckBecameDue:
		return "grapple_check_became_due"
	case combat.ConcentrationChecked:
		return "concentration_checked"
	}
	return "unknown"
}

// Requirement: doc/specs/combat-action-resolution.md FR-146, FR-107 - a test that imports only the public packages builds the command of each of the eight use cases, runs it, and names the variant of each result and of each resolution event with a switch that has no default arm; the closed result of each use case is reachable as Completed, Rejected and Invalid.
// Path: happy.
func TestCombatActionResolutionPublicAPI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	a, b := id("a"), id("b")
	walk, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: a, Count: 20}, {Combatant: b, Count: 10}},
		[]combat.CombatantProfile{{ID: a, Size: content.SizeMedium, Speeds: walk}, {ID: b, Size: content.SizeMedium, Speeds: walk}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	member := func(c combat.CombatantID, side combat.Side) combat.Combatant {
		hp, err := combat.RestoreHitPointStatus(combat.MustHitPoints(20, 20, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: c, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, hp, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("api"), order.(combat.InitiativeOrderEstablished),
		[]combat.Combatant{member(a, combat.SideA), member(b, combat.SideB)}))
	if err != nil {
		t.Fatal(err)
	}
	e := started.(combat.StartEncounterApplied).Encounter()
	scene, err := combat.NewAttackScene(5, combat.SightRelationCanSee, combat.SightRelationCanSee, combat.LineOfSightBlocked)
	if err != nil {
		t.Fatal(err)
	}
	roller := dicetest.NewFixedRoller(14, 5, 14, 12)
	got := map[string]string{}

	attack := combat.MustResolveAttackCommand(combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: 15, Distance: 5,
		Modifiers: d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2))})
	acmd, err := combat.NewPerformAttackCommand(combat.PerformAttackParams{Encounter: e, AttackerID: a, TargetID: b, Scene: scene, Attack: attack,
		Cost: combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansOtherWeapon}})
	if err != nil {
		t.Fatal(err)
	}
	pa, err := combat.NewPerformAttack(roller)
	if err != nil {
		t.Fatal(err)
	}
	r1, _ := pa.Run(ctx, acmd)
	got["perform_attack"] = resolutionVariant(r1)

	grapple := combat.MustResolveUnarmedStrikeCommand(combat.ResolveUnarmedStrikeParams{
		Option:          combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: d20.ChooseToFail{}},
		AbilityModifier: ability.MustScore(16).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
		AttackerSize: content.SizeMedium, TargetSize: content.SizeMedium, Hand: combat.HandAvailabilityHandFree, Distance: 5})
	ucmd, err := combat.NewPerformUnarmedStrikeCommand(combat.PerformUnarmedStrikeParams{Encounter: e, AttackerID: a, TargetID: b, Scene: scene, Strike: grapple,
		Cost: combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansUnarmedStrike}})
	if err != nil {
		t.Fatal(err)
	}
	pu, err := combat.NewPerformUnarmedStrike(roller)
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := pu.Run(ctx, ucmd)
	got["perform_unarmed_strike"] = resolutionVariant(r2)

	tcmd, err := combat.NewTakeCombatantActionCommand(e, combat.ActionLedger{}, a, combat.SpendInteraction{})
	if err != nil {
		t.Fatal(err)
	}
	r3, _ := combat.NewTakeCombatantAction().Run(ctx, tcmd)
	got["take_combatant_action"] = resolutionVariant(r3)

	mcmd, err := combat.NewMoveCombatantCommand(e, a, combat.DropProne{})
	if err != nil {
		t.Fatal(err)
	}
	r4, _ := combat.NewMoveCombatant().Run(ctx, mcmd)
	got["move_combatant"] = resolutionVariant(r4)

	vcmd, err := combat.NewAdvanceCombatTurnCommand(e, combat.ActionLedger{}, []combat.SpeedUpdate{{Combatant: id("ghost"), Speeds: walk}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	r5, _ := combat.NewAdvanceCombatTurn().Run(ctx, vcmd)
	got["advance_combat_turn"] = resolutionVariant(r5)

	dcmd, err := combat.NewRollCombatantDeathSavingThrowCommand(e, a, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := combat.NewRollCombatantDeathSavingThrow(roller)
	if err != nil {
		t.Fatal(err)
	}
	r6, _ := rt.Run(ctx, dcmd)
	got["roll_combatant_death_saving_throw"] = resolutionVariant(r6)

	scmd, err := combat.NewAttemptToStabilizeCombatantCommand(combat.AttemptToStabilizeCombatantParams{Encounter: e, HelperID: a, PatientID: b,
		Check: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}})
	if err != nil {
		t.Fatal(err)
	}
	as, err := combat.NewAttemptToStabilizeCombatant(roller)
	if err != nil {
		t.Fatal(err)
	}
	r7, _ := as.Run(ctx, scmd)
	got["attempt_to_stabilize_combatant"] = resolutionVariant(r7)

	gcmd, err := combat.NewDealDamageToCombatantCommand(e, b, []combat.Damage{combat.MustDamage(3, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	dd, err := combat.NewDealDamageToCombatant(roller)
	if err != nil {
		t.Fatal(err)
	}
	r8, _ := dd.Run(ctx, gcmd)
	got["deal_damage_to_combatant"] = resolutionVariant(r8)

	want := map[string]string{"perform_attack": "completed", "perform_unarmed_strike": "completed", "take_combatant_action": "completed", "move_combatant": "completed",
		"advance_combat_turn": "invalid", "roll_combatant_death_saving_throw": "rejected", "attempt_to_stabilize_combatant": "rejected", "deal_damage_to_combatant": "completed"}
	for op, w := range want {
		if got[op] != w {
			t.Errorf("%s gave %s, want %s", op, got[op], w)
		}
	}
	for _, ev := range r1.(combat.PerformAttackCompleted).Events() {
		if re, ok := ev.(combat.ResolutionEvent); ok && resolutionEventName(re) != re.Kind() {
			t.Errorf("event %s was named %s", re.Kind(), resolutionEventName(re))
		}
	}

	// A Concentration call: damage to a creature that holds Concentration ends with a ConcentrationChecked event.
	levels, err := character.NewClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 1}})
	if err != nil {
		t.Fatal(err)
	}
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("b"), Levels: levels})
	if err != nil {
		t.Fatal(err)
	}
	began, err := character.NewBeginConcentration().Do(ctx, character.BeginConcentrationCommand{State: created.(character.CreateSpellcastingCreated).State(),
		Subject: character.NewConcentrationSubjectFeature(content.MustFeatureID("synthetic_focus"))})
	if err != nil {
		t.Fatal(err)
	}
	tracks, err := combat.NewTracksConcentration(began.(character.BeginConcentrationBegan).State(), d20.NewRollModifiers(modifier.New(0)), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ccmd, err := gcmd.WithTargetConcentration(tracks)
	if err != nil {
		t.Fatal(err)
	}
	cd, err := combat.NewDealDamageToCombatant(dicetest.NewFixedRoller(15))
	if err != nil {
		t.Fatal(err)
	}
	r9, err := cd.Run(ctx, ccmd)
	if err != nil {
		t.Fatal(err)
	}
	checked := false
	for _, ev := range r9.(combat.DealDamageToCombatantCompleted).Events() {
		if re, ok := ev.(combat.ResolutionEvent); ok {
			checked = checked || resolutionEventName(re) == "concentration_checked"
			if resolutionEventName(re) != re.Kind() {
				t.Errorf("event %s was named %s", re.Kind(), resolutionEventName(re))
			}
		}
	}
	if !checked {
		t.Errorf("no ConcentrationChecked event after damage to a concentrating creature")
	}
}
