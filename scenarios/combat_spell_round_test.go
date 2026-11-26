package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/combat-action-resolution.md section 8, FR-092, FR-078, FR-079, FR-088, FR-091, FR-103, FR-134, FR-163, FR-113 and
// plan-phase-4.md 5.10 acceptance - a scripted round of a two-combatant fight through the public packages, each result's encounter and ledger
// the next call's input: an ogre hits a concentrating wizard (one Concentration save, held, then a second hit whose save fails and ends
// Concentration), a second attack past its allowance is refused with no die, the wizard kills the ogre and the turn advances past the dead
// creature. The data of the failed save is accepted by the owner's active-effects commands (FR-163). Synthetic combatants; SRD 5.2.1,
// "Making an Attack", pp. 14-15; "Concentration", p. 179; "Dropping to 0 Hit Points", p. 17.
// Path: happy.
func TestScriptedTwoCombatantRoundWithConcentration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	wizard, ogre := id("wizard"), id("ogre")
	walk, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: ogre, Count: 15}, {Combatant: wizard, Count: 12}},
		[]combat.CombatantProfile{{ID: ogre, Size: content.SizeLarge, Speeds: walk}, {ID: wizard, Size: content.SizeMedium, Speeds: walk}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	member := func(c combat.CombatantID, side combat.Side, hp int, rule combat.ZeroHitPointsRule) combat.Combatant {
		status, err := combat.RestoreHitPointStatus(combat.MustHitPoints(hp, hp, 0), combat.Alive{}, rule, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: c, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, status, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("scenario-spell-round"),
		order.(combat.InitiativeOrderEstablished), []combat.Combatant{member(wizard, combat.SideA, 12, combat.FallsUnconscious), member(ogre, combat.SideB, 7, combat.DiesInstantly)}))
	if err != nil {
		t.Fatal(err)
	}
	e := started.(combat.StartEncounterApplied).Encounter()
	var ledger combat.ActionLedger

	// The wizard's SpellcastingState, concentrating on a feature of its own (synthetic).
	levels, err := character.NewClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 1}})
	if err != nil {
		t.Fatal(err)
	}
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: levels})
	if err != nil {
		t.Fatal(err)
	}
	subject := character.NewConcentrationSubjectFeature(content.MustFeatureID("synthetic_focus"))
	began, err := character.NewBeginConcentration().Do(ctx, character.BeginConcentrationCommand{State: created.(character.CreateSpellcastingCreated).State(), Subject: subject})
	if err != nil {
		t.Fatal(err)
	}
	track := func(state character.SpellcastingState) combat.TracksConcentration {
		in, err := combat.NewTracksConcentration(state, d20.NewRollModifiers(modifier.New(2)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		return in
	}
	state := began.(character.BeginConcentrationBegan).State()

	// The dice, in the order requested: the ogre's attack 12 and damage 4 (12 + 5 = 17 against AC 12; 4 + 3 = 7 would kill a 7 point
	// creature, so the wizard has 12 and takes 7), the Concentration save 10 + 2 = 12 against DC 10 (held); the ogre's second hit is
	// another call below; the wizard's attack 15 and damage 7.
	roller := dicetest.NewFixedRoller(12, 4, 10)
	scene, err := combat.NewAttackScene(5, combat.SightRelationCanSee, combat.SightRelationCanSee, combat.LineOfSightBlocked)
	if err != nil {
		t.Fatal(err)
	}
	strike := func(attacker, target combat.CombatantID, cost combat.AttackCost, ac int, in combat.ConcentrationInput) (combat.PerformAttackResult, error) {
		spec := combat.MustDamageSpec(combat.MustDiceDamage(1, dice.MustDie(8), content.DamageBludgeoning, 0, true))
		atk := combat.MustResolveAttackCommand(combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: d20.ArmorClass(ac),
			Modifiers: d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)), Distance: 5, Damage: spec})
		cmd, err := combat.NewPerformAttackCommand(combat.PerformAttackParams{Encounter: e, Ledger: ledger, AttackerID: attacker, TargetID: target,
			Cost: cost, Scene: scene, Attack: atk, TargetConcentration: in})
		if err != nil {
			t.Fatal(err)
		}
		uc, err := combat.NewPerformAttack(roller)
		if err != nil {
			t.Fatal(err)
		}
		return uc.Run(ctx, cmd)
	}
	one := combat.BaselineAttackAllowance()
	begin := combat.BeginsAttackAction{Allowance: one, Means: combat.AttackMeansOtherWeapon}

	// 1: the ogre hits the concentrating wizard; the save is held and the state is carried to the next call.
	res, err := strike(ogre, wizard, begin, 12, track(state))
	if err != nil {
		t.Fatal(err)
	}
	hit := res.(combat.PerformAttackCompleted)
	e, ledger = hit.Encounter(), hit.Ledger()
	if got, want := withoutBloodied(kinds(hit.Events())), []string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken", "concentration_checked"}; !slices.Equal(got, want) {
		t.Errorf("step 1 events = %v, want %v", got, want)
	}
	if held, ok := hit.Concentration().(combat.ConcentrationHeld); !ok || held.DC() != 10 {
		t.Errorf("step 1 Concentration = %#v, want held against DC 10", hit.Concentration())
	}
	state, _ = hit.TargetSpellcasting()
	if _, going := state.Concentration(); !going {
		t.Fatalf("the wizard lost Concentration on a held save")
	}

	// 2: a second attack past the allowance of one is a refused action: the encounter, the ledger and the dice are untouched.
	asked := len(roller.Requests())
	res, err = strike(ogre, wizard, combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}, 12, track(state))
	if err != nil {
		t.Fatal(err)
	}
	refused, ok := res.(combat.PerformAttackRejected)
	if !ok || refused.Rejection().String() != "ledger_refused" || len(roller.Requests()) != asked || !refused.Encounter().Equal(e) || !refused.Ledger().Equal(ledger) {
		t.Fatalf("step 2 = %#v, want a ledger refusal that rolled nothing and changed nothing", res)
	}
	adv := func() {
		t.Helper()
		cmd, err := combat.NewAdvanceCombatTurnCommand(e, ledger, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewAdvanceCombatTurn().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		done := res.(combat.AdvanceCombatTurnCompleted)
		e, ledger = done.Encounter(), done.Ledger()
	}
	adv() // the wizard's turn

	// 3: the wizard hits the ogre for 7 and kills it; the dice are 15 and 7. A creature that does not concentrate checks nothing.
	roller = dicetest.NewFixedRoller(15, 4)
	res, err = strike(wizard, ogre, begin, 12, nil)
	if err != nil {
		t.Fatal(err)
	}
	kill := res.(combat.PerformAttackCompleted)
	e, ledger = kill.Encounter(), kill.Ledger()
	if got, want := withoutBloodied(kinds(kill.Events())), []string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken", "dropped_to_zero_hit_points", "died"}; !slices.Equal(got, want) {
		t.Errorf("step 3 events = %v, want %v", got, want)
	}
	if _, none := kill.Concentration().(combat.ConcentrationNotChecked); !none {
		t.Errorf("step 3 Concentration = %#v, want not checked", kill.Concentration())
	}
	// 4: the turn advances past the dead ogre, which is not skipped.
	adv()
	if e.Tracker().Active().Combatant != ogre {
		t.Errorf("active combatant = %v, want the dead ogre: its turn is not skipped", e.Tracker().Active().Combatant)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-163, FR-079, FR-132 - the data of a failed Concentration save is what the host forwards to the
// active-effects commands: the ConcentrationEnded event of ConcentrationCheckFailed is the cause ConcentrationLost(caster, subject), and the Total of
// the DamageTaken of the call is SubjectTookDamage(subject, total); both are accepted by NewConcentrationLost and NewSubjectTookDamage, with no event
// of this spec for either.
// Path: unhappy (rule interaction).
func TestFailedConcentrationSaveFeedsTheActiveEffects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	walk, _ := combat.NewMovementSpeeds(30)
	est, err := combat.NewEstablishInitiativeOrderCommand([]combat.InitiativeStanding{{Combatant: id("wizard"), Count: 12}, {Combatant: id("ogre"), Count: 10}},
		[]combat.CombatantProfile{{ID: id("wizard"), Size: content.SizeMedium, Speeds: walk}, {ID: id("ogre"), Size: content.SizeLarge, Speeds: walk}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	status, err := combat.RestoreHitPointStatus(combat.MustHitPoints(20, 20, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	mk := func(c string, side combat.Side) combat.Combatant {
		m, err := combat.NewCombatant(combat.CombatantRef{ID: id(c), Kind: combat.MonsterCombatant{ID: "fixture"}}, side, status, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("scenario-concentration"),
		order.(combat.InitiativeOrderEstablished), []combat.Combatant{mk("wizard", combat.SideA), mk("ogre", combat.SideB)}))
	if err != nil {
		t.Fatal(err)
	}
	levels, _ := character.NewClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 1}})
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: levels})
	if err != nil {
		t.Fatal(err)
	}
	subject := character.NewConcentrationSubjectFeature(content.MustFeatureID("synthetic_focus"))
	began, err := character.NewBeginConcentration().Do(ctx, character.BeginConcentrationCommand{State: created.(character.CreateSpellcastingCreated).State(), Subject: subject})
	if err != nil {
		t.Fatal(err)
	}
	tracks, err := combat.NewTracksConcentration(began.(character.BeginConcentrationBegan).State(), d20.NewRollModifiers(modifier.New(0)), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := combat.NewDealDamageToCombatantCommand(started.(combat.StartEncounterApplied).Encounter(), id("wizard"),
		[]combat.Damage{combat.MustDamage(14, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	if cmd, err = cmd.WithTargetConcentration(tracks); err != nil {
		t.Fatal(err)
	}
	uc, err := combat.NewDealDamageToCombatant(dicetest.NewFixedRoller(3))
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	dealt := res.(combat.DealDamageToCombatantCompleted)
	failed, ok := dealt.Concentration().(combat.ConcentrationCheckFailed)
	if !ok || failed.DC() != 10 {
		t.Fatalf("Concentration = %#v, want a failed save against DC 10", dealt.Concentration())
	}
	ended, ok := failed.Events()[0].(character.ConcentrationEnded)
	if !ok || ended.Reason() != character.EndedFailedSave {
		t.Fatalf("events = %#v, want ConcentrationEnded for the failed save", failed.Events())
	}
	lost, err := combat.NewConcentrationLost(id("wizard"), ended.Subject())
	if err != nil {
		t.Fatalf("NewConcentrationLost: %v [FR-163 a]", err)
	}
	var total int
	for _, ev := range dealt.Events() {
		if o, ok := ev.(combat.HitPointEventOccurred); ok {
			if taken, ok := o.Event().(combat.DamageTaken); ok {
				total = taken.Total()
			}
		}
	}
	tookDamage, err := combat.NewSubjectTookDamage(id("wizard"), total)
	if err != nil || total != 14 {
		t.Fatalf("NewSubjectTookDamage(%d) = %v, %v, want the 14 of the DamageTaken [FR-163 b]", total, tookDamage, err)
	}
	for _, cause := range []combat.EndingCause{lost, tookDamage} {
		res, err := combat.NewEndActiveEffects().Run(ctx, combat.EndActiveEffectsCommand{Cause: cause})
		if _, nothing := res.(combat.EndActiveEffectsNothingEnded); err != nil || !nothing {
			t.Errorf("EndActiveEffects with %T = %#v, %v, want NothingEnded for an empty registry [FR-163]", cause, res, err)
		}
	}
}

// withoutBloodied leaves out BecameBloodied, an optional event of hit-points-and-damage.md table 4.9a.
func withoutBloodied(kinds []string) []string {
	return slices.DeleteFunc(kinds, func(k string) bool { return k == "became_bloodied" })
}
