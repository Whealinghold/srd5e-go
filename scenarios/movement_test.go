package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/movement-lifting-and-flight.md S-1, FR-038, FR-040, FR-042 (SRD 5.2.1, "Flying", p. 182; "Hover", p. 183) - a Fly
// Speed 60 creature made Paralyzed by ConditionSet.Apply falls with the causes Incapacitated and FlySpeedZero, and stays aloft overriding both when it can hover.
// Path: happy.
func TestParalyzedFlierFallsUnlessItHovers(t *testing.T) {
	t.Parallel()
	hero := combat.MustCombatantID("flier")
	applied, ok := combat.ConditionSet{}.Apply(hero, combat.Paralyzed{}).(combat.ConditionAdded)
	if !ok {
		t.Fatal("Paralyzed was not applied")
	}
	set := applied.Set()
	fall := func(h combat.HoverAbility) combat.FlightAssessment {
		f, err := combat.NewFlightFacts(set, 60, h)
		if err != nil {
			t.Fatal(err)
		}
		return combat.AssessFlight(f)
	}
	falls, ok := fall(combat.HoverAbilityNone).(combat.FlightFalls)
	want := []combat.FlightFallCause{combat.FlightFallCauseIncapacitated, combat.FlightFallCauseFlySpeedZero}
	if !ok || !equalCauses(falls.Causes.Slice(), want) {
		t.Fatalf("no hover = %#v, want FlightFalls %v", falls, want)
	}
	aloft, ok := fall(combat.HoverAbilityHover).(combat.FlightStaysAloft)
	if !ok || !equalCauses(aloft.OverriddenCauses.Slice(), want) {
		t.Fatalf("hover = %#v, want FlightStaysAloft overriding %v", aloft, want)
	}
}

func equalCauses(a, b []combat.FlightFallCause) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Requirement: doc/specs/movement-lifting-and-flight.md S-2, FR-015, FR-023, FR-034 (SRD 5.2.1, "Long Jump", pp. 184-185) - a Strength 10
// creature runs 10 feet and asks for a 10-foot Long Jump into Difficult Terrain: the jump is permitted at a cost of 10, the landing
// needs the DC 10 Dexterity (Acrobatics) check, and the host feeds that requirement to the d20 resolver with a fixed roller.
// Path: happy.
func TestLongJumpIntoDifficultTerrain(t *testing.T) {
	t.Parallel()
	strength, err := ability.NewScore(10)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := combat.NewJumpAttempt(combat.JumpKindLong, strength, combat.ConditionSet{}, 10, combat.MovementModeWalk, 10, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := combat.AssessJump(attempt).(combat.JumpPermitted); !ok || got.MovementCostFeet != 10 {
		t.Fatalf("AssessJump = %#v, want JumpPermitted costing 10", combat.AssessJump(attempt))
	}
	landing, ok := combat.JumpLandingFor(combat.JumpKindLong, combat.TerrainDifficult).(combat.JumpLandingCheckRequired)
	if !ok {
		t.Fatal("a Long Jump into Difficult Terrain needs no check")
	}
	req := landing.Requirement
	if req.Ability() != ability.Dexterity || req.Skill() != content.SkillAcrobatics || req.Target() != d20.DCEasy {
		t.Fatalf("requirement = %v %v DC %v", req.Ability(), req.Skill(), req.Target())
	}
	resolver, err := d20.NewResolver(dicetest.NewFixedRoller(9))
	if err != nil {
		t.Fatal(err)
	}
	res, err := resolver.Resolve(context.Background(), d20.AbilityCheckCommand{
		Ability: req.Ability(), Target: req.Target(),
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(1))},
	})
	if err != nil {
		t.Fatal(err)
	}
	// A total of 10 meets the DC, so the creature stays on its feet; a total of 9 would leave it Prone.
	if check, ok := res.(d20.RolledCheck); !ok || check.Outcome() != d20.Success || req.OnFailure() != combat.MovementCheckFailureEffectBecomesProne {
		t.Fatalf("landing check = %#v, want a success (failure means Prone)", res)
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md S-3, FR-061, FR-060, FR-038 (SRD 5.2.1, "Draconic Flight", "Large Form", pp. 84-86;
// "Armor", p. 92) - a creature with Speed 40 and a Fly Speed 40 in Chain Mail with Strength 12 has walk 30 and fly 30, and
// Incapacitated makes the flier fall for that cause alone.
// Path: unhappy (rule interaction).
func TestArmoredDraconicFlierLosesTenFeetOfBothSpeeds(t *testing.T) {
	t.Parallel()
	speeds, err := combat.NewMovementSpeeds(40, combat.MovementSpeedEntry{Name: combat.SpeedNameFly, Feet: 40})
	if err != nil {
		t.Fatal(err)
	}
	rule, _ := content.NewBaseRule(16, content.NoDexterity{})
	cost, _ := content.NewCost(75, content.CoinGP, 0)
	price, _ := content.NewFixedCost(cost)
	weight, _ := content.NewWeight(55, "")
	mail, err := content.NewArmor("chain_mail", "Chain Mail", content.SourceCitation{}, content.ArmorHeavy, rule, 13, true, weight, price)
	if err != nil {
		t.Fatal(err)
	}
	str, _ := ability.NewScore(12)
	got := combat.ApplyArmorSpeedPenalty(speeds, mail, str)
	fly, _ := got.Of(combat.SpeedNameFly)
	if got.Walk() != 30 || fly != 30 {
		t.Fatalf("speeds = walk %d fly %d, want 30 and 30", got.Walk(), fly)
	}
	set, err := combat.NewConditionSet([]combat.Condition{combat.Incapacitated{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := combat.NewFlightFacts(set, fly, combat.HoverAbilityNone)
	if err != nil {
		t.Fatal(err)
	}
	falls, ok := combat.AssessFlight(facts).(combat.FlightFalls)
	if !ok || !equalCauses(falls.Causes.Slice(), []combat.FlightFallCause{combat.FlightFallCauseIncapacitated}) {
		t.Fatalf("AssessFlight = %#v, want FlightFalls for Incapacitated alone", falls)
	}
}
