package scenarios_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/combat-action-resolution.md section 8 table 8.1, FR-092, FR-091, FR-134 - the scripted round of two characters
// against two monsters runs through the public packages with each result's encounter and ledger as the next call's input: attacks (one a
// miss under Disadvantage from Dodge, one a Grapple, one a Critical-free kill), turn advances (a dead creature's turn is not skipped), a
// Death Saving Throw once the turn starts, healing by the encounter's own use case and the end of the encounter. The events of each step
// are those of the table, optional BecameBloodied excepted.
// Path: happy.
func TestCombatActionResolutionScenario(t *testing.T) {
	t.Parallel()
	var w strings.Builder
	runTable81(t, &w)
	if *transcriptMode {
		fmt.Fprint(os.Stdout, transcriptBegin+w.String()+transcriptEnd)
	}
}

// runTable81 runs the scripted round of table 8.1 and writes one line per step (the step and its event kinds) to w.
func runTable81(tb testing.TB, w io.Writer) {
	tb.Helper()
	t := tb
	ctx := context.Background()
	id := combat.MustCombatantID
	pc1, mon1, pc2, mon2 := id("pc-1"), id("monster-1"), id("pc-2"), id("monster-2")
	walk, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, c := range []combat.CombatantID{pc1, mon1, pc2, mon2} {
		standings = append(standings, combat.InitiativeStanding{Combatant: c, Count: combat.InitiativeCount([]int{18, 14, 10, 6}[i])})
		profiles = append(profiles, combat.CombatantProfile{ID: c, Size: content.SizeMedium, Speeds: walk})
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	r0, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	member := func(c combat.CombatantID, side combat.Side, current, maximum int, rule combat.ZeroHitPointsRule) combat.Combatant {
		hp, err := combat.RestoreHitPointStatus(combat.MustHitPoints(current, maximum, 0), combat.Alive{}, rule, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: c, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, hp, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("scenario-8-1"),
		r0.(combat.InitiativeOrderEstablished), []combat.Combatant{
			member(pc1, combat.SideA, 12, 12, combat.FallsUnconscious), member(pc2, combat.SideA, 6, 12, combat.FallsUnconscious),
			member(mon1, combat.SideB, 7, 7, combat.DiesInstantly), member(mon2, combat.SideB, 7, 7, combat.DiesInstantly)}))
	if err != nil {
		t.Fatal(err)
	}
	startedOK := started.(combat.StartEncounterApplied)
	e := startedOK.Encounter()
	var ledger combat.ActionLedger

	// The dice are the script of table 8.1, in the order requested.
	roller := dicetest.NewFixedRoller(14, 4, 17, 5, 5, 15, 12, 3, 4, 18, 4, 12)
	var _ combat.Roller = roller
	expect := func(step string, events []combat.Event, want ...string) {
		t.Helper()
		got := slices.DeleteFunc(kinds(events), func(k string) bool { return k == "became_bloodied" })
		if !slices.Equal(got, want) {
			t.Errorf("step %s events = %v, want %v", step, got, want)
		}
		fmt.Fprintf(w, "%s: %s\n", step, strings.Join(got, " "))
	}
	advance := func(step string, want ...string) {
		t.Helper()
		cmd, err := combat.NewAdvanceCombatTurnCommand(e, ledger, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		uc := combat.NewAdvanceCombatTurn()
		res, err := uc.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		done := res.(combat.AdvanceCombatTurnCompleted)
		e, ledger = done.Encounter(), done.Ledger()
		expect(step, done.Events(), want...)
	}
	mods := func(ability, proficiency int) d20.RollModifiers {
		return d20.NewRollModifiers(modifier.New(ability)).WithProficiencyBonus(modifier.New(proficiency))
	}
	attack := func(step string, attacker, target combat.CombatantID, cost combat.AttackCost, armorClass, proficiency int, damage combat.DamageSpec, want ...string) combat.PerformAttackCompleted {
		t.Helper()
		atk := combat.MustResolveAttackCommand(combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: d20.ArmorClass(armorClass),
			Modifiers: mods(3, proficiency), Distance: 5, Damage: damage})
		scene, err := combat.NewAttackScene(5, combat.SightRelationCanSee, combat.SightRelationCanSee, combat.LineOfSightBlocked)
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := combat.NewPerformAttackCommand(combat.PerformAttackParams{Encounter: e, Ledger: ledger, AttackerID: attacker, TargetID: target,
			Cost: cost, Scene: scene, Attack: atk})
		if err != nil {
			t.Fatal(err)
		}
		uc, err := combat.NewPerformAttack(roller)
		if err != nil {
			t.Fatal(err)
		}
		res, err := uc.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		done := res.(combat.PerformAttackCompleted)
		e, ledger = done.Encounter(), done.Ledger()
		expect(step, done.Events(), want...)
		return done
	}
	slash := func(sides int) combat.DamageSpec {
		return combat.MustDamageSpec(combat.MustDiceDamage(1, dice.MustDie(sides), content.DamageSlashing, 0, true))
	}
	two := mustAllowance(t, 2, 2)
	one := combat.BaselineAttackAllowance()

	// 1: the encounter started with pc-1 first.
	expect("1", startedOK.Events(), "encounter_started", "tracker_round_started", "tracker_turn_started")
	if e.Tracker().Active().Combatant != pc1 || e.Tracker().Round() != 1 {
		t.Fatalf("active %v in round %d, want pc-1 in round 1", e.Tracker().Active().Combatant, e.Tracker().Round())
	}
	// 2: pc-1 hits monster-1 for 7 Slashing and it dies.
	attack("2", pc1, mon1, combat.BeginsAttackAction{Allowance: two, Means: combat.AttackMeansOtherWeapon}, 12, 2, slash(8),
		"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken", "dropped_to_zero_hit_points", "died")
	if c, _ := e.Find(mon1); c.HitPoints().HitPoints().Current() != 0 {
		t.Errorf("monster-1 has %d Hit Points, want 0", c.HitPoints().HitPoints().Current())
	}
	// 3, 4: a dead creature's turn is not skipped, and the Attack action ends with the turn.
	advance("3", "tracker_turn_ended", "tracker_turn_started")
	if _, in := ledger.AttackAction().(combat.AttackActionInProgress); in {
		t.Errorf("the Attack action must end with the turn")
	}
	advance("4", "tracker_turn_ended", "tracker_turn_started")
	// 5: pc-2 takes the Dodge action.
	take := func(step string, who combat.CombatantID, req combat.TurnRequest, want ...string) {
		t.Helper()
		cmd, err := combat.NewTakeCombatantActionCommand(e, ledger, who, req)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewTakeCombatantAction().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		done := res.(combat.TakeCombatantActionCompleted)
		e, ledger = done.Encounter(), done.Ledger()
		expect(step, done.Events(), want...)
	}
	take("5", pc2, combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}}, "action_taken")
	advance("6", "tracker_turn_ended", "tracker_turn_started")
	// 7: monster-2 attacks the dodging pc-2: Disadvantage, the lower die 5 + 4 = 9 against AC 14 misses.
	miss := attack("7", mon2, pc2, combat.BeginsAttackAction{Allowance: one, Means: combat.AttackMeansOtherWeapon}, 14, 1, slash(6),
		"action_taken", "attack_made_in_attack_action", "attack_performed")
	if got := miss.Attack().(combat.AttackResolved).Attack().Roll().Mode(); got != modifier.WithDisadvantage {
		t.Errorf("roll mode = %v, want Disadvantage from Dodge", got)
	}
	advance("8", "tracker_turn_ended", "tracker_round_started", "tracker_turn_started")
	// 9: pc-1 grapples monster-2: DC 13 against Strength +0 with a 5.
	grapple := combat.MustResolveUnarmedStrikeCommand(combat.ResolveUnarmedStrikeParams{
		Option:          combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}},
		AbilityModifier: ability.MustScore(16).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
		AttackerSize: content.SizeMedium, TargetSize: content.SizeMedium, Hand: combat.HandAvailabilityHandFree, Distance: 5})
	scene, err := combat.NewAttackScene(5, combat.SightRelationCanSee, combat.SightRelationCanSee, combat.LineOfSightBlocked)
	if err != nil {
		t.Fatal(err)
	}
	ucmd, err := combat.NewPerformUnarmedStrikeCommand(combat.PerformUnarmedStrikeParams{Encounter: e, Ledger: ledger, AttackerID: pc1, TargetID: mon2,
		Cost: combat.BeginsAttackAction{Allowance: two, Means: combat.AttackMeansUnarmedStrike}, Scene: scene, Strike: grapple})
	if err != nil {
		t.Fatal(err)
	}
	uuc, err := combat.NewPerformUnarmedStrike(roller)
	if err != nil {
		t.Fatal(err)
	}
	ures, err := uuc.Run(ctx, ucmd)
	if err != nil {
		t.Fatal(err)
	}
	grabbed := ures.(combat.PerformUnarmedStrikeCompleted)
	e, ledger = grabbed.Encounter(), grabbed.Ledger()
	expect("9", grabbed.Events(), "action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "condition_applied", "movement_speeds_review_became_due")
	if applied, ok := grabbed.Effect().(combat.UnarmedEffectApplied); !ok || applied.Condition() != (combat.Grappled{Grappler: pc1}) {
		t.Errorf("effect = %#v, want Grappled by pc-1", grabbed.Effect())
	}
	if g := grabbed.Strike().(combat.UnarmedStrikeGrappleResolved); g.DC() != 13 {
		t.Errorf("grapple DC = %d, want 13", g.DC())
	}
	for i := 1; i <= 3; i++ {
		advance("10", "tracker_turn_ended", "tracker_turn_started")
	}
	// 11: monster-2, Grappled, attacks pc-2 (not its grappler): Disadvantage, 12 + 4 = 16 hits for 6 and drops pc-2.
	attack("11", mon2, pc2, combat.BeginsAttackAction{Allowance: one, Means: combat.AttackMeansOtherWeapon}, 14, 1, slash(6),
		"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken", "dropped_to_zero_hit_points", "fell_unconscious",
		"condition_applied", "movement_speeds_review_became_due")
	// 12, 13: round 3, pc-1 misses and then kills monster-2.
	advance("12a", "tracker_turn_ended", "tracker_round_started", "tracker_turn_started")
	attack("12", pc1, mon2, combat.BeginsAttackAction{Allowance: two, Means: combat.AttackMeansOtherWeapon}, 12, 2, slash(8),
		"action_taken", "attack_made_in_attack_action", "attack_performed")
	attack("13", pc1, mon2, combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, 12, 2, slash(8),
		"attack_made_in_attack_action", "attack_performed", "damage_taken", "dropped_to_zero_hit_points", "died")
	advance("14a", "tracker_turn_ended", "tracker_turn_started")
	advance("14b", "tracker_turn_ended", "tracker_turn_started", "dying_combatant_turn_started")
	// 15: pc-2's Death Saving Throw, 12 is a success.
	dcmd, err := combat.NewRollCombatantDeathSavingThrowCommand(e, pc2, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	duc, err := combat.NewRollCombatantDeathSavingThrow(dicetest.NewFixedRoller(12))
	if err != nil {
		t.Fatal(err)
	}
	dres, err := duc.Run(ctx, dcmd)
	if err != nil {
		t.Fatal(err)
	}
	saved := dres.(combat.RollCombatantDeathSavingThrowCompleted)
	e = saved.Encounter()
	expect("15", saved.Events(), "death_saving_throw_made")
	if saved.Natural() != 12 || saved.Total() != 12 {
		t.Errorf("natural %d total %d, want 12 and 12", saved.Natural(), saved.Total())
	}
	// 16: the encounter's own healing use case brings pc-2 back; 17: the host ends the encounter.
	hcmd, _ := combat.NewApplyHealingToCombatantCommand(e, pc2, 4)
	hres, err := combat.NewApplyHealingToCombatant().Run(ctx, hcmd)
	if err != nil {
		t.Fatal(err)
	}
	healed := hres.(combat.ApplyHealingToCombatantApplied)
	e = healed.Encounter()
	expect("16", healed.Events(), "hit_points_restored", "recovered", "condition_removed")
	endCmd, _ := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	eres, err := combat.NewEndEncounter().Run(ctx, endCmd)
	if err != nil {
		t.Fatal(err)
	}
	expect("17", eres.(combat.EndEncounterApplied).Events(), "encounter_ended")
}

// mustAllowance returns the attack allowance or fails the test.
func mustAllowance(tb testing.TB, general, pact int) combat.AttackAllowance {
	tb.Helper()
	a, err := combat.NewAttackAllowance(general, pact)
	if err != nil {
		tb.Fatal(err)
	}
	return a
}
