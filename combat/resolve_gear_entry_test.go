package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// requested lists the dice a FixedRoller was asked for, in notation.
func requested(r *dicetest.FixedRoller) []string {
	out := []string{}
	for _, e := range r.Requests() {
		out = append(out, e.String())
	}
	return out
}

// Requirement: per case
// Path: per case.
func TestResolveGearEntrySave(t *testing.T) {
	t.Parallel()
	const caltrops, bearings, trap = combat.DeployedGearKindCaltrops, combat.DeployedGearKindBallBearings, combat.DeployedGearKindHuntingTrap
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	cases := []struct {
		name, requirement, path string
		kind                    combat.DeployedGearKind
		attempt                 d20.SaveAttempt
		faces                   []int
		saved                   bool
		total                   int // of the save, -1 for a forced failure
		dice                    []string
	}{
		{"caltrops, total equals the DC", gearSpec + "FR-019 FR-024", "unhappy (boundary)", caltrops, rollWith(0), []int{15}, true, 15, []string{"1d20"}},
		{"caltrops, one below the DC", gearSpec + "FR-019 FR-024", "unhappy (boundary)", caltrops, rollWith(0), []int{14}, false, 14, []string{"1d20"}},
		{"caltrops, a modifier lifts the total", gearSpec + "FR-019", "happy", caltrops, rollWith(5), []int{10}, true, 15, []string{"1d20"}},
		{"caltrops, natural 20 is only a 20", gearSpec + "FR-024", "unhappy (boundary)", caltrops, rollWith(-6), []int{20}, false, 14, []string{"1d20"}},
		{"caltrops, natural 1 is only a 1", gearSpec + "FR-024", "unhappy (boundary)", caltrops, rollWith(14), []int{1}, true, 15, []string{"1d20"}},
		{"ball bearings, total equals the DC", gearSpec + "FR-020 FR-024", "unhappy (boundary)", bearings, rollWith(0), []int{10}, true, 10, []string{"1d20"}},
		{"ball bearings, one below the DC", gearSpec + "FR-020 FR-024", "unhappy (boundary)", bearings, rollWith(0), []int{9}, false, 9, []string{"1d20"}},
		{"hunting trap, total equals the DC", gearSpec + "FR-023 FR-025", "unhappy (boundary)", trap, rollWith(0), []int{13}, true, 13, []string{"1d20"}},
		{"hunting trap, one below the DC, then 1d4", gearSpec + "FR-023 FR-025", "unhappy (boundary)", trap, rollWith(0), []int{12, 3}, false, 12, []string{"1d20", "1d4"}},
		{"advantage rolls two d20 and keeps the higher", gearSpec + "FR-025", "unhappy (rule interaction)", caltrops, rollWith(0, adv), []int{5, 18}, true, 18, []string{"2d20"}},
		{"disadvantage keeps the lower", gearSpec + "FR-025", "unhappy (rule interaction)", caltrops, rollWith(0, dis), []int{5, 18}, false, 5, []string{"2d20"}},
		{"advantage and disadvantage cancel", gearSpec + "FR-025", "unhappy (rule interaction)", caltrops, rollWith(0, adv, dis), []int{16}, true, 16, []string{"1d20"}},
		{"caltrops, chosen failure draws no die", gearSpec + "FR-024", "unhappy (rule interaction)", caltrops, d20.ChooseToFail{}, nil, false, -1, []string{}},
		{"ball bearings, automatic failure draws no die", gearSpec + "FR-024", "unhappy (rule interaction)", bearings, d20.AutomaticFailure{}, nil, false, -1, []string{}},
		{"hunting trap, chosen failure rolls only the 1d4", gearSpec + "FR-024 FR-025", "unhappy (rule interaction)", trap, d20.ChooseToFail{}, []int{4}, false, -1, []string{"1d4"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			set := gearOne(t, c.kind)
			in := gearEntryIn(set)
			in.Attempt = c.attempt
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := gearEntry(t, in, roller)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if !slices.Equal(requested(roller), c.dice) {
				t.Errorf("dice requested = %v, want %v [%s]", requested(roller), c.dice, c.requirement)
			}
			var state interface {
				Ledger() combat.EntryLedger
				Test() (combat.GearTest, bool)
				Gears() combat.DeployedGears
			}
			if c.saved {
				s, ok := res.(combat.ResolveGearEntrySaved)
				if !ok || s.Outcome() != (combat.GearEntrySaved{}) {
					t.Fatalf("result = %#v, want saved [%s]", res, c.requirement)
				}
				state = s
			} else {
				f, ok := res.(combat.ResolveGearEntryFailed)
				if !ok {
					t.Fatalf("result = %#v, want failed [%s]", res, c.requirement)
				}
				state = f
			}
			test, ok := state.Test()
			if !ok || (test.Outcome() == d20.Success) != c.saved {
				t.Fatalf("test = %#v, want saved=%v", test, c.saved)
			}
			if r, rolled := test.(combat.GearTestRolled); rolled != (c.total >= 0) || (rolled && r.Result().Total() != c.total) {
				t.Errorf("test = %#v, want total %d", test, c.total)
			}
			if f, forced := test.(combat.GearTestForced); forced && c.total >= 0 || forced && f.Result().Kind() != d20.SavingThrow {
				t.Errorf("test = %#v, want a rolled save", test)
			}
			wantRecords := 1
			if c.kind == trap {
				wantRecords = 0
			}
			if state.Ledger().Len() != wantRecords {
				t.Errorf("ledger holds %d records, want %d [FR-017]", state.Ledger().Len(), wantRecords)
			}
			if c.kind == trap {
				if gearOf(t, state.Gears()).State() != (combat.GearSprung{}) {
					t.Error("a trap set off must be sprung whether or not the creature saved [FR-023]")
				}
			} else if !state.Gears().Equal(set) {
				t.Error("other gear must be unchanged")
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-019, FR-020, FR-023 - a failed save returns the effects of the gear without applying them: Caltrops 1 Piercing and Speed 0, Ball Bearings Prone, a Hunting Trap the 1d4 as Piercing damage, Speed 0 and a caught hold with the given id and the trap's gear id, plus one HuntingTrapSprung for [deployer, creature] (SRD 5.2.1, pp. 96-98).
// Path: happy.
func TestResolveGearEntryEffects(t *testing.T) {
	t.Parallel()
	piercing := func(n int) combat.GearEffect {
		return combat.GearDamageEffect{Damage: combat.MustDamage(n, content.DamagePiercing)}
	}
	failed := func(kind combat.DeployedGearKind, faces ...int) combat.ResolveGearEntryFailed {
		res, err := gearEntry(t, gearEntryIn(gearOne(t, kind)), dicetest.NewFixedRoller(faces...))
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.ResolveGearEntryFailed)
	}
	if fx := failed(combat.DeployedGearKindCaltrops, 1).Outcome().(combat.GearEntryFailed).Effects(); !slices.Equal(fx, []combat.GearEffect{piercing(1), combat.GearSpeedZeroEffect{}}) {
		t.Errorf("caltrops effects = %#v", fx)
	}
	if fx := failed(combat.DeployedGearKindBallBearings, 1).Outcome().(combat.GearEntryFailed).Effects(); !slices.Equal(fx, []combat.GearEffect{combat.GearConditionEffect{Condition: content.ConditionProne}}) {
		t.Errorf("ball bearings effects = %#v", fx)
	}
	r := failed(combat.DeployedGearKindHuntingTrap, 1, 4)
	fx := r.Outcome().(combat.GearEntryFailed).Effects()
	hold := fx[2].(combat.GearCaughtEffect).Hold
	if len(fx) != 3 || fx[0] != piercing(4) || fx[1] != (combat.GearSpeedZeroEffect{}) || hold.GearID() != "g" || hold.ID() != "h" || !hold.IsOn(gid("walker")) ||
		hold.Kind() != combat.HoldKindHuntingTrap || hold.EscapeDifficultyClass() != 13 || hold.MovementLimitFeet() != 3 || hold.Restraint() != combat.HoldRestraintDoesNotRestrain {
		t.Errorf("trap effects = %#v, want 4 Piercing, Speed 0 and a caught hold", fx)
	}
	ev := r.Events()
	combattest.CheckEvent(t, ev[0], "hunting_trap_sprung", gid("owner"), gid("walker"))
	if len(ev) != 1 {
		t.Errorf("events = %v, want one HuntingTrapSprung", ev)
	}
	if caltrops := failed(combat.DeployedGearKindCaltrops, 1); len(caltrops.Events()) != 0 {
		t.Error("Caltrops produce no event")
	}
	ev[0] = nil
	if r.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-021, FR-022, FR-025 - lit Oil damages with 5 Fire and no die, once per turn across entering and ending a turn, and returns the new ledger; Test reports no save (SRD 5.2.1, "Oil", p. 98).
// Path: happy; unhappy (rule interaction).
func TestResolveGearEntryOil(t *testing.T) {
	t.Parallel()
	set := gearBurning(t, 1, "walker")
	roller := dicetest.NewFixedRoller()
	in := gearEntryIn(set)
	res, err := gearEntry(t, in, roller)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := res.(combat.ResolveGearEntryDamaged)
	if !ok || len(requested(roller)) != 0 || d.Ledger().Len() != 1 || !d.Gears().Equal(set) {
		t.Fatalf("result = %#v, dice %v; want 5 Fire with no die and one record", res, requested(roller))
	}
	if _, rolled := d.Test(); rolled {
		t.Error("Oil allows no saving throw")
	}
	fx := d.Outcome().(combat.GearEntryDamaged).Effects()
	if !slices.Equal(fx, []combat.GearEffect{combat.GearDamageEffect{Damage: combat.MustDamage(5, content.DamageFire)}}) {
		t.Errorf("effects = %#v, want 5 Fire", fx)
	}
	in.Ledger, in.Trigger = d.Ledger(), combat.AreaTriggerEndsTurn
	again, err := gearEntry(t, in, roller)
	if r, ok := again.(combat.ResolveGearEntryRejected); err != nil || !ok || r.Reason() != combat.GearRejectionAlreadyTriggeredThisTurn {
		t.Errorf("ending the turn after entering = %#v, %v; want alreadyTriggeredThisTurn", again, err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-016, FR-017, FR-022, FR-060, table 4.9.1 - refusals return their reason with nothing rolled and nothing recorded: an unknown gear id, ending a turn in Caltrops, a second entry on one turn, unlit Oil, a sprung Hunting Trap; an ending of a turn stamped with another creature's turn is Invalid with end_turn_stamp_mismatch.
// Path: per case.
func TestResolveGearEntryRefusals(t *testing.T) {
	t.Parallel()
	roll := rollWith(0)
	first, _ := gearEntry(t, gearEntryIn(gearOne(t, combat.DeployedGearKindCaltrops)), dicetest.NewFixedRoller(20))
	ledger := first.(combat.ResolveGearEntrySaved).Ledger()
	cases := []struct {
		name   string
		set    combat.DeployedGears
		change func(*combat.ResolveGearEntryInput)
		want   combat.GearRejection
	}{
		{"unknown gear", gearOne(t, combat.DeployedGearKindCaltrops), func(in *combat.ResolveGearEntryInput) { in.GearID = "zz" }, combat.GearRejectionUnknownGear},
		{"ending a turn in caltrops", gearOne(t, combat.DeployedGearKindCaltrops), func(in *combat.ResolveGearEntryInput) { in.Trigger = combat.AreaTriggerEndsTurn }, combat.GearRejectionTriggerNotApplicable},
		{"second entry on one turn", gearOne(t, combat.DeployedGearKindCaltrops), func(in *combat.ResolveGearEntryInput) { in.Ledger = ledger }, combat.GearRejectionAlreadyTriggeredThisTurn},
		{"unlit oil", gearOne(t, combat.DeployedGearKindPouredOil), func(*combat.ResolveGearEntryInput) {}, combat.GearRejectionOilNotBurning},
		{"sprung trap", gearSprung(t), func(*combat.ResolveGearEntryInput) {}, combat.GearRejectionTrapAlreadySprung},
		{"oil burned through", gearBurning(t, 1, "walker"), func(in *combat.ResolveGearEntryInput) { in.Stamp = gearStamp(4, "other") }, combat.GearRejectionOilNotBurning},
	}
	for _, c := range cases {
		in := gearEntryIn(c.set)
		in.Attempt = roll
		c.change(&in)
		roller := dicetest.NewFixedRoller()
		res, err := gearEntry(t, in, roller)
		if r, ok := res.(combat.ResolveGearEntryRejected); err != nil || !ok || r.Reason() != c.want || len(requested(roller)) != 0 {
			t.Errorf("%s: result = %#v, %v, dice %v; want rejected %v with nothing rolled [%s FR-022]", c.name, res, err, requested(roller), c.want, gearSpec)
		}
	}
	in := gearEntryIn(gearBurning(t, 1, "walker"))
	in.Trigger, in.Stamp = combat.AreaTriggerEndsTurn, gearStamp(1, "other")
	res, err := gearEntry(t, in, dicetest.NewFixedRoller())
	if i, ok := res.(combat.ResolveGearEntryInvalid); err != nil || !ok || i.Problem() != combat.GearProblemEndTurnStampMismatch {
		t.Errorf("mismatched stamp = %#v, %v; want invalid end_turn_stamp_mismatch [FR-021]", res, err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-061, FR-062 - a roller that fails, panics or answers another expression, at the d20 or at the Hunting Trap's 1d4, propagates a *srderr.PortError with no result; equal commands give equal results and leave the command unchanged.
// Path: unhappy (failure of collaborator, determinism, immutability).
func TestResolveGearEntryFailureAndDeterminism(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller(), "no value for the 1d4": dicetest.NewFixedRoller(1)} {
		res, err := gearEntry(t, gearEntryIn(gearOne(t, combat.DeployedGearKindHuntingTrap)), roller)
		envPortFailure(t, err)
		if res != nil {
			t.Errorf("%s: result = %#v, want none on a port failure", name, res)
		}
	}
	set := gearOne(t, combat.DeployedGearKindHuntingTrap)
	in := gearEntryIn(set)
	cmd, err := combat.NewResolveGearEntryCommand(in)
	if err != nil {
		t.Fatal(err)
	}
	use := gearEntryUse(t, dicetest.NewFixedRoller(1, 2, 1, 2))
	one, err1 := use.Run(context.Background(), cmd)
	two, err2 := use.Run(context.Background(), cmd)
	if err1 != nil || err2 != nil || one.(combat.ResolveGearEntryFailed).Gears().Len() != 1 || gearOf(t, one.(combat.ResolveGearEntryFailed).Gears()).State() != (combat.GearSprung{}) ||
		!one.(combat.ResolveGearEntryFailed).Gears().Equal(two.(combat.ResolveGearEntryFailed).Gears()) {
		t.Errorf("equal commands must give equal results: %#v %#v", one, two)
	}
	if gearOf(t, set).State() != (combat.GearArmed{}) {
		t.Error("Run must not change the gear it was given")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-061, FR-071 - the constructors return a GearError for a blank gear or hold id, a zero creature, a trigger that is not valid, the zero stamp and an attempt that is nil; the use case rejects a nil roller; Run returns ErrInvalidInput for the zero command or use case and ctx.Err for a cancelled ctx, before any roll.
// Path: unhappy (invalid input, cancellation).
func TestResolveGearEntryInvalidInput(t *testing.T) {
	t.Parallel()
	set := gearOne(t, combat.DeployedGearKindCaltrops)
	cases := map[string]struct {
		change func(*combat.ResolveGearEntryInput)
		want   combat.GearProblem
	}{
		"blank gear id":   {func(in *combat.ResolveGearEntryInput) { in.GearID = "" }, combat.GearProblemBlankID},
		"blank hold id":   {func(in *combat.ResolveGearEntryInput) { in.HoldID = "" }, combat.GearProblemBlankID},
		"zero creature":   {func(in *combat.ResolveGearEntryInput) { in.Creature = "" }, combat.GearProblemBlankID},
		"invalid trigger": {func(in *combat.ResolveGearEntryInput) { in.Trigger = 0 }, combat.GearProblemInvalidValue},
		"zero stamp":      {func(in *combat.ResolveGearEntryInput) { in.Stamp = combat.TurnStamp{} }, combat.GearProblemInvalidValue},
		"nil attempt":     {func(in *combat.ResolveGearEntryInput) { in.Attempt = nil }, combat.GearProblemInvalidValue},
	}
	for name, c := range cases {
		in := gearEntryIn(set)
		c.change(&in)
		if _, err := combat.NewResolveGearEntryCommand(in); !errors.Is(err, &combat.GearError{Problem: c.want}) {
			t.Errorf("%s: err = %v, want a %v GearError", name, err, c.want)
		}
	}
	if _, err := combat.NewResolveGearEntry(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v, want ErrInvalidInput", err)
	}
	roller := dicetest.NewFixedRoller(20)
	if _, err := gearEntryUse(t, roller).Run(context.Background(), combat.ResolveGearEntryCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	cmd, _ := combat.NewResolveGearEntryCommand(gearEntryIn(set))
	if _, err := (combat.ResolveGearEntry{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: err = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := gearEntryUse(t, roller).Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil || len(requested(roller)) != 0 {
		t.Errorf("cancelled ctx: %v, %v, dice %v; want context.Canceled and no roll", res, err, requested(roller))
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-017, FR-018 - the command copies the circumstances of the attempt, so changing the slice afterwards changes nothing.
// Path: unhappy (immutability).
func TestResolveGearEntryCopiesAttempt(t *testing.T) {
	t.Parallel()
	circumstances := []modifier.RollCircumstance{modifier.Advantage{}}
	in := gearEntryIn(gearOne(t, combat.DeployedGearKindCaltrops))
	in.Attempt = d20.RollAttempt{Circumstances: circumstances}
	cmd, err := combat.NewResolveGearEntryCommand(in)
	if err != nil {
		t.Fatal(err)
	}
	circumstances[0] = modifier.Disadvantage{}
	roller := dicetest.NewFixedRoller(1, 20)
	res, err := gearEntryUse(t, roller).Run(context.Background(), cmd)
	if err != nil || !slices.Equal(requested(roller), []string{"2d20"}) {
		t.Fatalf("Run = %v, %v, dice %v; want advantage (2d20)", res, err, requested(roller))
	}
	if _, ok := res.(combat.ResolveGearEntrySaved); !ok {
		t.Errorf("result = %#v, want saved with the higher die", res)
	}
}
