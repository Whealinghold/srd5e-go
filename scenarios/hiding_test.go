package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/hiding-searching-and-senses.md section 6 hide flow and seeing flow; FR-051, FR-054, FR-059, FR-062, FR-064,
// FR-068, FR-049 (SRD 5.2.1, "Hide [Action]", p. 183) - a rogue behind Three-Quarters Cover with every enemy line blocked hides
// with a rolled Dexterity (Stealth) check of 17; it is Invisible while hidden; a Darkvision-less observer in the dark cannot see it,
// an enemy's search total below 17 leaves it hidden and 17 finds it; and an attack roll ends hiding with one event.
// Path: happy.
func TestHiderRollsHidesAndIsFoundOrAttacks(t *testing.T) {
	t.Parallel()
	rogue, guard := combat.MustCombatantID("rogue"), combat.MustCombatantID("guard")
	var hide combat.HideRules
	scene, err := combat.NewHideScene(rogue, content.ObscurementNone, []combat.HideCoverDegree{combat.HideCoverDegreeThreeQuartersCover}, false,
		[]combat.EnemySighting{{Enemy: guard, Line: combat.LineOfSightBlocked, HiderSeesEnemy: combat.SightRelationCanSee}})
	if err != nil {
		t.Fatal(err)
	}
	a, ok := hide.Assess(scene).(combat.HideAvailable)
	if !ok || len(a.DiscernedSightings()) != 1 {
		t.Fatalf("Assess = %#v, want HideAvailable with the guard discerned", hide.Assess(scene))
	}
	roll, err := d20.NewRoll(modifier.Normal, []int{14})
	if err != nil {
		t.Fatal(err)
	}
	check, err := d20.ResolveCheck(d20.AbilityCheck, roll, d20.NewRollModifiers(modifier.New(3)), a.Check.DC())
	if err != nil {
		t.Fatal(err)
	}
	res, err := hide.Resolve(a, check.Total(), check.Outcome())
	if err != nil {
		t.Fatal(err)
	}
	hid, ok := res.(combat.HideSucceeded)
	if !ok || hid.State().FindDC() != 17 || len(hid.Events()) != 1 || hid.Events()[0].Kind() != "creature_hid" {
		t.Fatalf("Resolve = %#v, want HideSucceeded with find DC 17", res)
	}
	status := combat.HidingStatus(combat.HidingStatusHidden{State: hid.State()})
	if got := hide.ImpliedConditions(status); !slices.Equal(got, []content.ConditionKind{content.ConditionInvisible}) {
		t.Fatalf("ImpliedConditions = %v", got)
	}
	var dark, sight = combat.MustSpaceVisibility(content.LightDarkness, 0, 0), combat.SightRules{}
	unseen, err := combat.NewSightScene(20, combat.LineOfSightClear, dark, combat.NewSenseRanges(combat.MustSenseRange(content.SenseDarkvision, 60)), nil, hide.ImpliedConditions(status))
	if err != nil {
		t.Fatal(err)
	}
	if got := sight.CanSee(unseen); got != combat.SightRelationCannotSee {
		t.Errorf("CanSee a hidden (Invisible) rogue = %v, want CannotSee", got)
	}
	if got := hide.AttemptToFind(status, rogue, combat.HidingFinderEnemy, 16); got.Status() != status {
		t.Error("a total of 16 found a rogue hidden at 17")
	}
	if got := hide.AttemptToFind(status, rogue, combat.HidingFinderEnemy, 17); got.Status() == status || len(got.Events()) != 1 {
		t.Error("a total of 17 did not find the rogue")
	}
	atk := hide.ApplyActivity(status, combat.HidingActivityAttackRoll, rogue)
	if _, hidden := atk.Status().(combat.HidingStatusNotHidden); !hidden || len(atk.Events()) != 1 || atk.Events()[0].Kind() != "hiding_ended" {
		t.Errorf("attack roll change = %#v, want NotHidden and one hiding_ended", atk)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md section 8 (scenario that chains Hide, a roll, an attack and the
// location-given-away flag), FR-068, doc/specs/attacks.md FR-037 (SRD 5.2.1, "Hide [Action]", p. 183; "Unseen Attackers and
// Targets", p. 14) - a hidden attacker's hit and miss both give its location away, and the host ends the hiding with the attack
// roll's activity. The host derives AttackerHidden from the status it read before the roll.
// Path: happy.
func TestHiddenAttackerGivesItsLocationAwayWhateverTheRoll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rogue := combat.MustCombatantID("rogue")
	state, err := combat.NewHiddenState(17)
	if err != nil {
		t.Fatal(err)
	}
	var hide combat.HideRules
	for _, c := range []struct {
		name string
		face int
		hit  bool
	}{{"a hit", 15, true}, {"a miss", 2, false}} {
		status := combat.HidingStatus(combat.HidingStatusHidden{State: state})
		_, hidden := status.(combat.HidingStatusHidden) // the fact is read before the roll (FR-068)
		cmd, err := combat.NewResolveAttackCommand(combat.ResolveAttackParams{
			Form: combat.MeleeWeaponAttack, ArmorClass: 15, Modifiers: d20.NewRollModifiers(modifier.New(5)), Distance: 5,
			Situation: combat.AttackSituation{AttackerHidden: hidden},
		})
		if err != nil {
			t.Fatal(err)
		}
		attack, err := combat.NewResolveAttack(dicetest.NewFixedRoller(c.face))
		if err != nil {
			t.Fatal(err)
		}
		res, err := attack.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		resolved, ok := res.(combat.AttackResolved)
		if !ok || resolved.Hit() != c.hit || !resolved.RevealsAttackerLocation() {
			t.Errorf("%s: result = %#v, want hit %v and the location given away", c.name, res, c.hit)
		}
		ended := hide.ApplyActivity(status, combat.HidingActivityAttackRoll, rogue)
		if _, notHidden := ended.Status().(combat.HidingStatusNotHidden); !notHidden {
			t.Errorf("%s: status after the attack roll = %#v, want not hidden", c.name, ended.Status())
		}
	}
}
