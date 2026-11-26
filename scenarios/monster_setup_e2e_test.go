package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The monsters of the grpC scenarios are synthetic: made-up names carrying printed numbers in the shape of the
// stat blocks of SRD 5.2.1, pp. 254 to 316, never a stat block.

func grpCMust[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// grpCFields is the smallest valid stat block: Medium, Hit Points 9 (2d8), Initiative +1 (11), all scores 10.
func grpCFields() content.MonsterFields {
	scores := map[ability.Ability]ability.Score{}
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	return content.MonsterFields{
		Sizes:             []content.CreatureSize{content.SizeMedium},
		Type:              content.CreatureTypeBeast,
		Alignment:         content.AlignmentUnaligned,
		ArmorClass:        12,
		Initiative:        grpCMust(content.NewStatBlockInitiative(1, 11)),
		HitPoints:         grpCMust(content.NewHitPointsEntry(9, dice.MustExpression(2, dice.D8(), 0))),
		Speeds:            []content.SpeedEntry{grpCMust(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, ""))},
		Abilities:         grpCMust(content.NewAbilityBlock(grpCMust(ability.NewScores(scores)), saves)),
		PassivePerception: 10,
	}
}

// grpCMonster builds a monster from the smallest valid stat block changed by edit.
func grpCMonster(edit func(*content.MonsterFields)) content.Monster {
	f := grpCFields()
	if edit != nil {
		edit(&f)
	}
	return grpCMust(content.NewMonster("fixture_beast", "Fixture Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), f))
}

func grpCPart(name string, f content.StatBlockPartFields) content.StatBlockPart {
	if f.Usage == nil {
		f.Usage = content.PartUnlimited{}
	}
	if f.Text == "" {
		f.Text = "T."
	}
	return grpCMust(content.NewStatBlockPart(name, f))
}

func grpCUsage(name string, usage content.PartUsage) content.StatBlockPart {
	return grpCPart(name, content.StatBlockPartFields{Usage: usage})
}

func grpCPerDay(count, inLair int) content.PartUsage {
	return grpCMust(content.NewPartPerDay(count, inLair))
}

func grpCRecharge(low, high int) content.PartUsage {
	return grpCMust(content.NewPartRecharge(low, high))
}

func grpCRef(s combat.MonsterPartSection, i int) combat.LimitedUsePartRef {
	return grpCMust(combat.NewLimitedUsePartRef(s, i))
}

func grpCID(s string) combat.CombatantID { return combat.MustCombatantID(s) }

// grpCCombatant turns a monster setup into the combatant the encounter holds.
func grpCCombatant(t *testing.T, id string, side combat.Side, s combat.MonsterSetup) combat.Combatant {
	t.Helper()
	c, err := combat.NewCombatant(s.Ref(grpCID(id)), side, s.HitPointStatus(), s.Conditions())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// grpCPlayer is a character-like combatant with the given Hit Points.
func grpCPlayer(t *testing.T, id string, side combat.Side, hp int) combat.Combatant {
	t.Helper()
	status, err := combat.NewHitPointStatus(hp, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	c, err := combat.NewCombatant(combat.CombatantRef{ID: grpCID(id), Kind: combat.MonsterCombatant{ID: "stand_in"}}, side, status, combat.ConditionSet{})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// grpCEncounter starts an encounter of the combatants, in the order given, Medium and 30 ft. each.
func grpCEncounter(t *testing.T, members ...combat.Combatant) combat.Encounter {
	t.Helper()
	speed, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, m := range members {
		id := m.Ref().ID
		standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(20 - 2*i)})
		profiles = append(profiles, combat.CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: speed})
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	est, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	started, err := combat.NewStartEncounter().Run(context.Background(), combat.NewStartEncounterCommand(combat.MustEncounterID("grpc"), est.(combat.InitiativeOrderEstablished), members))
	if err != nil {
		t.Fatal(err)
	}
	return started.(combat.StartEncounterApplied).Encounter()
}

func grpCSetUp(t *testing.T, roller combat.Roller, m content.Monster, cmd combat.SetUpMonsterCombatantCommand, opts ...combat.Option) (combat.SetUpMonsterCombatantResult, error) {
	t.Helper()
	uc, err := combat.NewSetUpMonsterCombatant(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Monster = m
	return uc.Run(context.Background(), cmd)
}

// grpCSetUpMonster is a Large monster with Hit Points 59 (7d10 + 21) that has several size options in its sibling.
func grpCSetUpMonster() content.Monster {
	return grpCMonster(func(f *content.MonsterFields) {
		f.Sizes = []content.CreatureSize{content.SizeLarge}
		f.HitPoints = grpCMust(content.NewHitPointsEntry(59, dice.MustExpression(7, dice.D10(), 21)))
	})
}

// Requirement: doc/specs/monster-combatant-setup.md FR-037, FR-038, FR-008, FR-009, FR-012, FR-013, FR-015 - a host sets up two creatures of one stat block, one with the printed average Hit Points (no die is requested) and one with the Hit Point Dice rolled (exactly one request, for the printed expression); each becomes a combatant of an encounter whose maximum is the average or the rolled total, and the rolled creature dies instantly at 0 Hit Points.
// Path: happy.
func TestGrpCMonsterSetupAverageAndRolledJoinAnEncounter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := grpCSetUpMonster()

	avgRoller := dicetest.NewMinimumRoller()
	res, err := grpCSetUp(t, avgRoller, m, combat.SetUpMonsterCombatantCommand{HitPoints: combat.UseAverageHitPoints{}})
	if err != nil {
		t.Fatal(err)
	}
	avg := res.(combat.SetUpMonsterCombatantApplied)
	if got := avg.Setup().HitPointStatus().HitPoints().Maximum(); got != 59 {
		t.Errorf("average maximum = %d, want the printed 59", got)
	}
	if _, ok := avg.HitPoints().(combat.PrintedAverageHitPoints); !ok || len(avgRoller.Requests()) != 0 {
		t.Errorf("average used %T and %d requests, want PrintedAverageHitPoints and none", avg.HitPoints(), len(avgRoller.Requests()))
	}

	// 7d10 rolling 10, 9, 8, 7, 6, 5, 4 totals 49, plus the printed 21.
	rollRoller := dicetest.NewFixedRoller(10, 9, 8, 7, 6, 5, 4)
	res, err = grpCSetUp(t, rollRoller, m, combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}})
	if err != nil {
		t.Fatal(err)
	}
	rolled := res.(combat.SetUpMonsterCombatantApplied)
	if got := rolled.Setup().HitPointStatus().HitPoints().Maximum(); got != 70 {
		t.Errorf("rolled maximum = %d, want 49 + 21 = 70", got)
	}
	if reqs := rollRoller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(7, dice.D10(), 21) {
		t.Errorf("rolled requests = %v, want one request for 7d10+21", reqs)
	}
	if rolled.Setup().HitPointsRaisedToOne() {
		t.Errorf("a rolled 70 was reported as raised to 1")
	}
	if rolled.Equal(avg) {
		t.Errorf("the rolled and the average setup compare equal")
	}

	// Both creatures fight in one encounter; the rolled one dies instantly at 0 Hit Points (DiesInstantly default).
	hero := grpCPlayer(t, "hero", combat.SideA, 30)
	e := grpCEncounter(t, hero,
		grpCCombatant(t, "average", combat.SideB, avg.Setup()),
		grpCCombatant(t, "rolled", combat.SideB, rolled.Setup()))
	hit, err := combat.NewApplyDamageToCombatantCommand(e, grpCID("rolled"), []combat.Damage{combat.MustDamage(70, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	dr, err := combat.NewApplyDamageToCombatant().Run(ctx, hit)
	if err != nil {
		t.Fatal(err)
	}
	applied := dr.(combat.ApplyDamageToCombatantApplied)
	if !slices.Contains(grpCKinds(applied.Events()), "died") {
		t.Errorf("events = %v, want the rolled creature to die at 0 Hit Points", grpCKinds(applied.Events()))
	}
	if m, _ := applied.Encounter().Find(grpCID("average")); m.HitPoints().HitPoints().Current() != 59 {
		t.Errorf("the other creature of the stat block lost Hit Points: %v", m.HitPoints())
	}
}

func grpCKinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// Requirement: doc/specs/monster-combatant-setup.md FR-010, FR-050, FR-009 - a rolled total below 1 is raised to 1 and the setup says so; the average is never raised.
// Path: unhappy (boundary).
func TestGrpCMonsterSetupRolledTotalBelowOneIsRaisedToOne(t *testing.T) {
	t.Parallel()
	m := grpCMonster(func(f *content.MonsterFields) {
		f.HitPoints = grpCMust(content.NewHitPointsEntry(1, dice.MustExpression(1, dice.D4(), -3)))
	})
	res, err := grpCSetUp(t, dicetest.NewFixedRoller(1), m, combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}})
	if err != nil {
		t.Fatal(err)
	}
	a := res.(combat.SetUpMonsterCombatantApplied)
	if a.Setup().HitPointStatus().HitPoints().Maximum() != 1 || !a.Setup().HitPointsRaisedToOne() {
		t.Errorf("setup = max %d raised %v, want 1 and raised", a.Setup().HitPointStatus().HitPoints().Maximum(), a.Setup().HitPointsRaisedToOne())
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-003, FR-004, FR-039, FR-042 - a stat block with several size options and no choice, and a size the block does not offer, are Invalid results found before any die is rolled.
// Path: unhappy (invalid input).
func TestGrpCMonsterSetupSizeProblemsConsumeNoDie(t *testing.T) {
	t.Parallel()
	several := grpCMonster(func(f *content.MonsterFields) {
		f.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeSmall}
	})
	for _, c := range []struct {
		name string
		cmd  combat.SetUpMonsterCombatantCommand
		want combat.MonsterSetupProblem
	}{
		{"several options, no choice", combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}}, combat.SizeChoiceRequired},
		{"size not offered", combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}, Size: content.SizeHuge}, combat.SizeNotOffered},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewMinimumRoller()
			res, err := grpCSetUp(t, roller, several, c.cmd)
			if err != nil {
				t.Fatal(err)
			}
			inv, ok := res.(combat.SetUpMonsterCombatantInvalid)
			if !ok || len(inv.Problems()) == 0 || inv.Problems()[0] != c.want {
				t.Fatalf("result = %#v, want Invalid starting with %v", res, c.want)
			}
			if n := len(roller.Requests()); n != 0 {
				t.Errorf("%d dice requested before a size problem, want none", n)
			}
		})
	}
	// A chosen size among the options is accepted and carried to the combatant profile.
	res, err := grpCSetUp(t, dicetest.NewMinimumRoller(), several, combat.SetUpMonsterCombatantCommand{HitPoints: combat.UseAverageHitPoints{}, Size: content.SizeSmall})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.(combat.SetUpMonsterCombatantApplied).Setup().Size(); got != content.SizeSmall {
		t.Errorf("size = %v, want Small", got)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-040, FR-037 - a roller that fails is returned wrapped as a port failure for the dice port with no result, a nil roller or a command with no Hit Points method is invalid input, and a cancelled context is returned unwrapped.
// Path: unhappy (failure).
func TestGrpCMonsterSetupFailures(t *testing.T) {
	t.Parallel()
	m := grpCSetUpMonster()
	cause := errors.New("dice service down")
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
	res, err := grpCSetUp(t, failing, m, combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}})
	var port *srderr.PortError
	if res != nil || !errors.As(err, &port) || port.Port != dice.Port || !errors.Is(err, cause) {
		t.Errorf("roller failure = %v, %v, want no result and a dice PortError wrapping the cause", res, err)
	}
	if _, err := combat.NewSetUpMonsterCombatant(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v, want ErrInvalidInput", err)
	}
	uc, err := combat.NewSetUpMonsterCombatant(dicetest.NewMinimumRoller())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Run(context.Background(), combat.SetUpMonsterCombatantCommand{Monster: m}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no Hit Points method error = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, combat.SetUpMonsterCombatantCommand{Monster: m, HitPoints: combat.UseAverageHitPoints{}}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context error = %v, want context.Canceled", err)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-037, FR-052 - a stat block of more than 64 parts is an Invalid result with limit_exceeded, and a stat block of exactly 64 is accepted (plan P10).
// Path: unhappy (boundary).
func TestGrpCMonsterSetupPartCap(t *testing.T) {
	t.Parallel()
	build := func(n int) content.Monster {
		return grpCMonster(func(f *content.MonsterFields) {
			for i := range n {
				f.Traits = append(f.Traits, grpCUsage("Trait "+string(rune('A'+i/26))+string(rune('a'+i%26)), content.PartUnlimited{}))
			}
		})
	}
	res, err := grpCSetUp(t, dicetest.NewMinimumRoller(), build(64), combat.SetUpMonsterCombatantCommand{HitPoints: combat.UseAverageHitPoints{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.(combat.SetUpMonsterCombatantApplied); !ok {
		t.Fatalf("64 parts = %#v, want Applied", res)
	}
	res, err = grpCSetUp(t, dicetest.NewMinimumRoller(), build(65), combat.SetUpMonsterCombatantCommand{HitPoints: combat.UseAverageHitPoints{}})
	if err != nil {
		t.Fatal(err)
	}
	inv, ok := res.(combat.SetUpMonsterCombatantInvalid)
	if !ok || inv.Problems()[0] != combat.SetupLimitExceeded {
		t.Fatalf("65 parts = %#v, want Invalid with SetupLimitExceeded", res)
	}
}
