package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// containmentTerms returns the terms of a Purple Worm (SRD 5.2.1, p. 317), a Behir (p. 263) or a Gelatinous Cube (p. 287).
func containmentTerms(t *testing.T, which string) combat.ContainmentTerms {
	t.Helper()
	acid := func(notation string) combat.DamageTerm {
		return combat.MustDamageTerm(dice.MustParseExpression(notation), content.DamageAcid)
	}
	br := []content.ConditionKind{content.ConditionBlinded, content.ConditionRestrained}
	in := combat.ContainmentTermsInput{Mode: combat.ContainmentModeSwallowed, MaxSize: content.SizeLarge, Imposed: br, TotalCover: true,
		Suffocation: combat.SuffocationRuleNone, ActionKind: combat.ContainmentActionKindBonusAction}
	switch which {
	case "worm":
		in.Capacity = combat.MustContainmentCapacity(3, nil)
		in.Entry = combat.MustEntryBySave(ability.Strength, 19, nil, false, 0, false)
		in.Periodic = combat.MustPeriodicDamage([]combat.DamageTerm{acid("5d6")}, combat.ContainmentTimingStartOfContainerTurn, false)
		in.Regurgitation, in.DeathRule = combat.MustRegurgitation(30, 21, 5), combat.MustEscapeFromCorpse(20)
	case "behir":
		in.Capacity = combat.MustContainmentCapacity(1, nil)
		in.Entry = combat.MustEntryBySave(ability.Dexterity, 18, nil, false, 0, false)
		in.Periodic = combat.MustPeriodicDamage([]combat.DamageTerm{acid("6d6")}, combat.ContainmentTimingStartOfContainerTurn, false)
		in.Regurgitation, in.DeathRule = combat.MustRegurgitation(30, 14, 10), combat.MustEscapeFromCorpse(15)
	default:
		in.Mode, in.Imposed, in.ActionKind = combat.ContainmentModeEngulfed, []content.ConditionKind{content.ConditionRestrained}, combat.ContainmentActionKindAction
		in.Capacity = combat.MustContainmentCapacity(4, map[content.CreatureSize]int{content.SizeLarge: 4})
		in.Entry = combat.MustEntryBySave(ability.Dexterity, 12, []combat.DamageTerm{acid("3d6")}, true, 5, true)
		in.Suffocation, in.NoVerbalSpells, in.DeathRule = combat.SuffocationRuleAlways, true, combat.ReleaseOnDeath{}
		in.Exit = combat.MustExitTerms(12, 12, []combat.DamageTerm{acid("3d6")})
	}
	terms, err := combat.NewContainmentTerms(in)
	if err != nil {
		t.Fatal(err)
	}
	return terms
}

func containmentRoll(mod int) d20.RollAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(mod))}
}

// swallow runs EnterContainer for a creature that fails its save, grappled by the container beforehand.
func swallow(t *testing.T, terms combat.ContainmentTerms, container, victim string, face int) combat.EnterContainerContained {
	t.Helper()
	grapple := combat.MustGrapple(combat.MustCombatantID(container), combat.MustCombatantID(victim), combat.MustGraspingHand(0), 15, 5)
	cmd, err := combat.NewEnterContainerCommand(combat.EnterContainerInput{
		Grapples: combat.MustGrapples([]combat.Grapple{grapple}), Container: combat.MustCombatantID(container), Occupant: combat.MustCombatantID(victim),
		OccupantSize: content.SizeMedium, Terms: terms, OccupantConditions: combat.MustConditionSet([]combat.Condition{grapple.Condition()}, nil),
		Circumstance: combat.EntryCircumstanceNone, Attempt: containmentRoll(0)})
	if err != nil {
		t.Fatal(err)
	}
	uc, err := combat.NewEnterContainer(dicetest.NewFixedRoller(face))
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	held, ok := res.(combat.EnterContainerContained)
	if !ok {
		t.Fatalf("entry = %#v, want contained", res)
	}
	return held
}

// Requirement: doc/specs/monster-engulf-and-swallow.md section 8 (first scenario), FR-022, FR-043, FR-046 to FR-049 (SRD 5.2.1, "Purple Worm", p. 317) - a Purple Worm that holds a fighter in a Grapple swallows it when the Strength save fails; the host applies the acid damage due at each start of the worm's turn; the fighter deals 31 damage in one turn; the worm fails its Constitution save and regurgitates the fighter, who lands Prone; through the public API only.
// Path: happy.
func TestPurpleWormSwallowsAndRegurgitatesAFighter(t *testing.T) {
	t.Parallel()
	ctx, worm, fighter := context.Background(), combat.MustCombatantID("worm"), combat.MustCombatantID("fighter")
	terms := containmentTerms(t, "worm")
	held := swallow(t, terms, "worm", "fighter", 10)
	if held.Grapples().Len() != 0 || !slices.Equal(held.Conditions().Effective(), []content.ConditionKind{content.ConditionBlinded, content.ConditionRestrained}) {
		t.Fatalf("after the swallow: grapples %d conditions %v", held.Grapples().Len(), held.Conditions().Effective())
	}

	start, err := combat.NewAdvanceContainerTurnCommand(combat.AdvanceContainerTurnInput{Containments: held.Containments(), Container: worm, Boundary: combat.ContainerTurnBoundaryStart})
	if err != nil {
		t.Fatal(err)
	}
	turn, err := combat.NewAdvanceContainerTurn().Run(ctx, start)
	if err != nil {
		t.Fatal(err)
	}
	due := turn.(combat.AdvanceContainerTurnAdvanced).DamageDue()
	if len(due) != 1 || due[0].Target != fighter || due[0].Terms()[0].Expression().Notation() != "5d6" {
		t.Fatalf("damage due = %+v, want 5d6 acid for the fighter", due)
	}

	recorded, err := turn.(combat.AdvanceContainerTurnAdvanced).Containments().RecordDamageFromInside(fighter, 31)
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := combat.NewResolveRegurgitationCommand(combat.ResolveRegurgitationInput{Containments: recorded.(combat.ContainmentTransitionApplied).Containments(), Container: worm,
		OccupantConditions: map[combat.CombatantID]combat.ConditionSet{fighter: held.Conditions()}, Attempt: containmentRoll(0)})
	if err != nil {
		t.Fatal(err)
	}
	uc, err := combat.NewResolveRegurgitation(dicetest.NewFixedRoller(20)) // 20 against DC 21
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	out, ok := res.(combat.ResolveRegurgitationRegurgitated)
	if !ok || out.Containments().Len() != 0 || len(out.Releases()) != 1 {
		t.Fatalf("regurgitation = %#v, want the fighter released", res)
	}
	rel := out.Releases()[0]
	if rel.Placement() != (combat.LandingWithin{Feet: 5}) || !slices.Equal(rel.Conditions().Effective(), []content.ConditionKind{content.ConditionProne}) {
		t.Errorf("release = placement %v conditions %v, want within 5 feet and Prone", rel.Placement(), rel.Conditions().Effective())
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md section 8 (second scenario), FR-057, FR-058 (SRD 5.2.1, "Behir", p. 263) - a Behir is killed with a creature inside; the creature is trapped in the corpse with Total Cover and Blinded, loses Restrained, and climbs out Prone by spending 15 feet of movement; through the public API only.
// Path: happy.
func TestKilledBehirTrapsItsMealUntilItClimbsOut(t *testing.T) {
	t.Parallel()
	ctx, victim := context.Background(), combat.MustCombatantID("victim")
	held := swallow(t, containmentTerms(t, "behir"), "behir", "victim", 3)
	died, err := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{Containments: held.Containments(),
		OccupantConditions: map[combat.CombatantID]combat.ConditionSet{victim: held.Conditions()}, Trigger: combat.ContainerDied{Container: combat.MustCombatantID("behir")}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewReleaseOccupants().Run(ctx, died)
	if err != nil {
		t.Fatal(err)
	}
	trapped := res.(combat.ReleaseOccupantsReleased)
	if len(trapped.Trapped()) != 1 || trapped.Trapped()[0].Feet() != 15 ||
		!slices.Equal(trapped.Trapped()[0].Conditions().Effective(), []content.ConditionKind{content.ConditionBlinded}) {
		t.Fatalf("death = %+v, want the victim trapped for 15 feet and still Blinded", trapped)
	}
	if (combat.ContainmentRules{}).CoverBetween(trapped.Containments(), combat.MustCombatantID("archer"), victim) != combat.ContainmentCoverTotal {
		t.Error("the corpse still gives Total Cover")
	}
	climb := func(feet int) combat.ReleaseOccupantsResult {
		cmd, err := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{Containments: trapped.Containments(),
			OccupantConditions: map[combat.CombatantID]combat.ConditionSet{victim: trapped.Trapped()[0].Conditions()},
			Trigger:            combat.EscapedCorpse{Occupant: victim, MovementRemainingFeet: feet}})
		if err != nil {
			t.Fatal(err)
		}
		r, err := combat.NewReleaseOccupants().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if r := climb(14); r != (combat.ReleaseOccupantsRejected{Rejection: combat.ContainmentRejectionInsufficientMovement}) {
		t.Errorf("14 feet = %#v, want insufficient movement", r)
	}
	out := climb(15).(combat.ReleaseOccupantsReleased)
	if out.MovementSpentFeet() != 15 || !slices.Equal(out.Releases()[0].Conditions().Effective(), []content.ConditionKind{content.ConditionProne}) {
		t.Errorf("escape = spent %d conditions %v", out.MovementSpentFeet(), out.Releases()[0].Conditions().Effective())
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md section 8 (third scenario), FR-051, FR-052, FR-075 (SRD 5.2.1, "Gelatinous Cube", p. 287) - a creature is engulfed by a Gelatinous Cube; an ally's DC 12 pull fails and costs nothing, then succeeds and reports 3d6 acid due on the ally; the creature leaves with Restrained removed and its breathing restored; through the public API only.
// Path: happy.
func TestGelatinousCubeEngulfsACreatureThatAnAllyPullsOut(t *testing.T) {
	t.Parallel()
	ctx, victim, ally := context.Background(), combat.MustCombatantID("victim"), combat.MustCombatantID("ally")
	terms := containmentTerms(t, "cube")
	cmd, err := combat.NewEnterContainerCommand(combat.EnterContainerInput{Container: combat.MustCombatantID("cube"), Occupant: victim, OccupantSize: content.SizeMedium, Terms: terms,
		Circumstance: combat.EntryCircumstanceNone, Attempt: containmentRoll(0)})
	if err != nil {
		t.Fatal(err)
	}
	enter, err := combat.NewEnterContainer(dicetest.NewFixedRoller(4))
	if err != nil {
		t.Fatal(err)
	}
	res, err := enter.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	held := res.(combat.EnterContainerContained)
	if held.Effects().Suffocation != combat.SuffocationRuleAlways || !held.Effects().NoVerbalSpells || len(held.DamageDue()) != 1 {
		t.Fatalf("effects = %+v damage %v", held.Effects(), held.DamageDue())
	}
	pull := func(face int) combat.ExitContainerResult {
		c, err := combat.NewExitContainerCommand(combat.ExitContainerInput{Containments: held.Containments(), Grapples: held.Grapples(), Occupant: victim, OccupantConditions: held.Conditions(),
			Method: combat.ExitByPull{Puller: ally, PullerWithin5Feet: true, Attempt: containmentRoll(0), Actions: combat.ConditionActionAvailabilityAvailable}})
		if err != nil {
			t.Fatal(err)
		}
		uc, err := combat.NewExitContainer(dicetest.NewFixedRoller(face))
		if err != nil {
			t.Fatal(err)
		}
		r, err := uc.Run(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	failed, ok := pull(11).(combat.ExitContainerHeld)
	if !ok || !failed.ActionSpent() || failed.Containments().Len() != 1 {
		t.Fatalf("the first pull = %#v, want held", failed)
	}
	out, ok := pull(12).(combat.ExitContainerExited)
	if !ok || out.Containments().Len() != 0 || !out.Release().RestoreBreathing() || len(out.Release().Conditions().Effective()) != 0 {
		t.Fatalf("the second pull = %#v, want exited with breathing restored", out)
	}
	if due := out.DamageDue(); len(due) != 1 || due[0].Target != ally || due[0].Terms()[0].Expression().Notation() != "3d6" || due[0].Cause() != combat.ContainmentDamageCausePull {
		t.Errorf("damage due = %+v, want 3d6 acid on the ally", due)
	}
}
