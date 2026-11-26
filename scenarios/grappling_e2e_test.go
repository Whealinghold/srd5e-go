package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpAEncounter starts an encounter of Medium combatants (the first half on side A, the rest on side B) with the
// ids in turn order, each with 30 Hit Points.
func grpAEncounter(t *testing.T, names ...string) combat.Encounter {
	t.Helper()
	speed, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	var members []combat.Combatant
	for i, n := range names {
		id := combat.MustCombatantID(n)
		standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(20 - 2*i)})
		profiles = append(profiles, combat.CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: speed})
		side := combat.SideA
		if i >= len(names)/2 {
			side = combat.SideB
		}
		hp, err := combat.NewHitPointStatus(30, combat.FallsUnconscious, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		m, err := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: content.MonsterID(n)}}, side, hp, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		members = append(members, m)
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), est)
	if err != nil {
		t.Fatal(err)
	}
	started, err := combat.NewStartEncounter().Run(context.Background(),
		combat.NewStartEncounterCommand(combat.MustEncounterID("grpa"), res.(combat.InitiativeOrderEstablished), members))
	if err != nil {
		t.Fatal(err)
	}
	return started.(combat.StartEncounterApplied).Encounter()
}

// grpAEscapeCommand builds the command of target escaping set with the Acrobatics option and a +3 modifier.
func grpAEscapeCommand(t *testing.T, set combat.Grapples, target string, actions combat.ConditionActionAvailability, attempt d20.CheckAttempt) combat.EscapeGrappleFromRecordCommand {
	t.Helper()
	cmd, err := combat.NewEscapeGrappleFromRecordCommand(set, combat.MustCombatantID(target), combat.GrappleEscapeOptionDexterityAcrobatics, actions, attempt)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// Requirement: doc/specs/grappling.md FR-022 to FR-026, FR-066, FR-007 - an ogre grabs a hero with one hand and a goblin with the other; a second grab with a used hand, and a grab of an already grappled target, are refused (the part rule first); equal ids are an Invalid result.
// Path: unhappy (rule interaction).
func TestGrapplingBeginRefusalsScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	begin := func(set combat.Grapples, grappler, target string, hand int, dc int, feet int) combat.BeginGrappleResult {
		t.Helper()
		res, err := combat.NewBeginGrapple().Run(ctx, combat.BeginGrappleCommand{Grapples: set, Grappler: id(grappler), Target: id(target),
			Part: combat.MustGraspingHand(hand), EscapeDC: d20.DifficultyClass(dc), RangeFeet: feet})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	first, ok := begin(combat.Grapples{}, "ogre", "hero", 0, 13, 5).(combat.BeginGrappleApplied)
	if !ok || first.Grapples().Len() != 1 || !slices.Equal(kinds(first.Events()), []string{"grapple_started"}) || first.Grapple().EscapeDC != 13 {
		t.Fatalf("first grab = %+v, want one grapple and one grapple_started", first)
	}
	both, ok := begin(first.Grapples(), "ogre", "goblin", 1, 12, 5).(combat.BeginGrappleApplied)
	if !ok || both.Grapples().Len() != 2 || len(both.Grapples().HeldBy(id("ogre"))) != 2 {
		t.Fatalf("a second hand on another creature = %+v, want two grapples held by the ogre", both)
	}
	cases := map[string]struct {
		res  combat.BeginGrappleResult
		want combat.GrappleRejection
	}{
		"the same hand again":              {begin(both.Grapples(), "ogre", "wolf", 0, 13, 5), combat.GrappleRejectionPartInUse},
		"a held target through a new part": {begin(first.Grapples(), "troll", "hero", 0, 13, 5), combat.GrappleRejectionTargetAlreadyGrappled},
		"both rules at once: part first":   {begin(both.Grapples(), "ogre", "hero", 0, 13, 5), combat.GrappleRejectionPartInUse},
	}
	for name, c := range cases {
		if r, ok := c.res.(combat.BeginGrappleRejected); !ok || r.Rejection != c.want {
			t.Errorf("%s: result = %#v, want rejection %v", name, c.res, c.want)
		}
	}
	if inv, ok := begin(first.Grapples(), "ogre", "ogre", 1, 13, 5).(combat.BeginGrappleInvalid); !ok || inv.Problem != combat.GrappleProblemSameGrapplerAndTarget {
		t.Errorf("a self-grapple = %#v, want Invalid SameGrapplerAndTarget", inv)
	}
	if first.Grapples().Len() != 1 {
		t.Errorf("a refused or invalid grab changed the set it was given")
	}
}

// Requirement: doc/specs/grappling.md FR-048 to FR-052, FR-057, FR-054, FR-055, FR-068 - the host reviews the grapples after a turn: the ogre's grapple holds at the range exactly, the troll's ends out of range, the bear's ends because it is Incapacitated; endings follow the order the grapples began; an unknown or repeated observation is Invalid.
// Path: unhappy (boundary).
func TestGrapplingReviewScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	set := combat.MustGrapples([]combat.Grapple{
		combat.MustGrapple(id("ogre"), id("hero"), combat.MustGraspingHand(0), 13, 5),
		combat.MustGrapple(id("troll"), id("ally"), combat.MustGraspingHand(0), 13, 5),
		combat.MustGrapple(id("bear"), id("scout"), combat.MustGraspingBodyPart("maw"), 14, 5),
	})
	free := combat.ConditionSet{}
	stunned := combat.MustConditionSet([]combat.Condition{combat.Incapacitated{}}, nil)
	obs := func(grappler, target string, conditions combat.ConditionSet, feet int) combat.GrappleObservation {
		return combat.MustGrappleObservation(id(grappler), id(target), conditions, feet)
	}
	res, err := combat.NewReviewGrapples().Run(ctx, combat.ReviewGrapplesCommand{Grapples: set, Observations: []combat.GrappleObservation{
		obs("bear", "scout", stunned, 5), obs("troll", "ally", free, 10), obs("ogre", "hero", free, 5),
	}})
	if err != nil {
		t.Fatal(err)
	}
	applied := res.(combat.ReviewGrapplesApplied)
	endings := applied.Endings()
	if applied.Grapples().Len() != 1 || len(endings) != 2 ||
		endings[0].Grapple().Grappler != id("troll") || endings[0].Cause() != combat.GrappleEndCauseOutOfRange ||
		endings[1].Grapple().Grappler != id("bear") || endings[1].Cause() != combat.GrappleEndCauseGrapplerIncapacitated {
		t.Fatalf("review = %d left, endings %+v, want the ogre's alone, then troll out of range before bear incapacitated", applied.Grapples().Len(), endings)
	}
	if applied.Grapples().IsHolding(id("ogre"), combat.MustGraspingHand(0)) != combat.PartStateHolding {
		t.Errorf("the ogre's hand must still hold at exactly the range")
	}
	for name, observations := range map[string][]combat.GrappleObservation{
		"unknown grapple":       {obs("ogre", "ally", free, 5)},
		"the same one twice":    {obs("ogre", "hero", free, 5), obs("ogre", "hero", free, 10)},
		"a missing observation": nil,
	} {
		r, err := combat.NewReviewGrapples().Run(ctx, combat.ReviewGrapplesCommand{Grapples: set, Observations: observations})
		if err != nil {
			t.Fatal(err)
		}
		if name == "a missing observation" {
			if a, ok := r.(combat.ReviewGrapplesApplied); !ok || !a.Grapples().Equal(set) || len(a.Endings()) != 0 {
				t.Errorf("%s: result = %#v, want the set unchanged and no ending", name, r)
			}
			continue
		}
		inv, ok := r.(combat.ReviewGrapplesInvalid)
		want := combat.GrappleProblemUnknownGrapple
		if name == "the same one twice" {
			want = combat.GrappleProblemDuplicateObservation
		}
		if !ok || inv.Problem != want {
			t.Errorf("%s: result = %#v, want Invalid %v", name, r, want)
		}
	}
	if _, err := combat.NewGrappleObservation(id("ogre"), id("hero"), free, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative distance err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/grappling.md FR-037 to FR-047, FR-041, FR-044 - a hero held by an ogre (escape DC 13) rolls Dexterity (Acrobatics) with +3: a 10 escapes, a 9 stays held, an Incapacitated creature has no action to spend, a creature not held has nothing to escape, an automatic failure rolls nothing, and a failing roller is one wrapped port error.
// Path: unhappy (failure).
func TestGrapplingEscapeScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	set := combat.MustGrapples([]combat.Grapple{combat.MustGrapple(id("ogre"), id("hero"), combat.MustGraspingHand(0), 13, 5)})
	roll := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(3))}
	run := func(roller dice.Roller, target string, actions combat.ConditionActionAvailability, attempt d20.CheckAttempt) (combat.EscapeGrappleFromRecordResult, error) {
		uc, err := combat.NewEscapeGrappleFromRecord(roller)
		if err != nil {
			t.Fatal(err)
		}
		return uc.Run(ctx, grpAEscapeCommand(t, set, target, actions, attempt))
	}
	available := combat.ConditionActionAvailabilityAvailable

	res, err := run(dicetest.NewFixedRoller(10), "hero", available, roll)
	if err != nil {
		t.Fatal(err)
	}
	escaped, ok := res.(combat.EscapeGrappleFromRecordEscaped)
	if !ok || escaped.Grapples().Len() != 0 || escaped.Ending().Cause() != combat.GrappleEndCauseEscaped || !escaped.ActionSpent() ||
		escaped.Check().(combat.GrappleEscapeRolled).Result().Total() != 13 {
		t.Fatalf("a total of 13 against DC 13 = %#v, want escaped with the action spent", res)
	}
	held, err := run(dicetest.NewFixedRoller(9), "hero", available, roll)
	if err != nil {
		t.Fatal(err)
	}
	if h, ok := held.(combat.EscapeGrappleFromRecordHeld); !ok || !h.Grapples().Equal(set) || !h.ActionSpent() {
		t.Errorf("a total of 12 = %#v, want held, set unchanged, action spent", held)
	}
	none := dicetest.NewFixedRoller(1)
	for name, c := range map[string]struct {
		target  string
		actions combat.ConditionActionAvailability
		want    combat.GrappleRejection
	}{
		"no action available":            {"hero", combat.ConditionActionAvailabilityNone, combat.GrappleRejectionNoAction},
		"not grappled":                   {"ally", available, combat.GrappleRejectionNotGrappled},
		"not grappled and no action too": {"ally", combat.ConditionActionAvailabilityNone, combat.GrappleRejectionNotGrappled},
	} {
		r, err := run(none, c.target, c.actions, roll)
		if err != nil {
			t.Fatal(err)
		}
		if rej, ok := r.(combat.EscapeGrappleFromRecordRejected); !ok || rej.Rejection != c.want || rej.ActionSpent() {
			t.Errorf("%s: result = %#v, want rejected %v with no action spent", name, r, c.want)
		}
	}
	if len(none.Requests()) != 0 {
		t.Errorf("refusals rolled %v, want no die", none.Requests())
	}
	forced, err := run(none, "hero", available, d20.AutomaticFailure{})
	if err != nil {
		t.Fatal(err)
	}
	if h, ok := forced.(combat.EscapeGrappleFromRecordHeld); !ok || !h.ActionSpent() {
		t.Errorf("an automatic failure = %#v, want held with the action spent", forced)
	} else if _, ok := h.Check().(combat.GrappleEscapeForced); !ok {
		t.Errorf("the check = %#v, want GrappleEscapeForced", h.Check())
	}
	if len(none.Requests()) != 0 {
		t.Errorf("an automatic failure rolled %v, want no die", none.Requests())
	}
	cause := errors.New("dice source down")
	_, err = run(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), "hero", available, roll)
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, cause) {
		t.Errorf("a failing roller err = %v, want one *PortError of the dice port wrapping the cause", err)
	}
	if _, err := combat.NewEscapeGrappleFromRecordCommand(set, id("hero"), combat.GrappleEscapeOption(0), available, roll); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an unset escape option err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/grappling.md FR-029 to FR-032, FR-060, FR-061, FR-067, FR-059 - the ogre lets the hero go while it is itself Incapacitated (release costs no action), the same hand grabs again, the host drops the record of a Grappled condition that ended elsewhere, and a creature that is not held has no record to drop.
// Path: happy.
func TestGrapplingReleaseAndDropScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	set := combat.MustGrapples([]combat.Grapple{
		combat.MustGrapple(id("ogre"), id("hero"), combat.MustGraspingHand(0), 13, 5),
		combat.MustGrapple(id("ogre"), id("ally"), combat.MustGraspingHand(1), 13, 5),
	})
	rel, err := combat.NewReleaseGrapple().Run(ctx, combat.ReleaseGrappleCommand{Grapples: set, Grappler: id("ogre"), Target: id("hero")})
	if err != nil {
		t.Fatal(err)
	}
	released := rel.(combat.ReleaseGrappleApplied)
	if released.Ending().Cause() != combat.GrappleEndCauseReleased || released.Grapples().Len() != 1 ||
		released.Grapples().IsHolding(id("ogre"), combat.MustGraspingHand(0)) != combat.PartStateNotHoldingGrapple {
		t.Fatalf("release = %+v, want one ending Released and hand 0 free", released)
	}
	again, err := combat.NewBeginGrapple().Run(ctx, combat.BeginGrappleCommand{Grapples: released.Grapples(), Grappler: id("ogre"), Target: id("scout"),
		Part: combat.MustGraspingHand(0), EscapeDC: 14, RangeFeet: 5})
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := again.(combat.BeginGrappleApplied); !ok || a.Grapples().Len() != 2 {
		t.Fatalf("the freed hand grabbing again = %#v, want applied", again)
	}
	for name, c := range map[string]combat.ReleaseGrappleCommand{
		"the target is free":        {Grapples: released.Grapples(), Grappler: id("ogre"), Target: id("hero")},
		"another grappler holds it": {Grapples: released.Grapples(), Grappler: id("troll"), Target: id("ally")},
	} {
		r, err := combat.NewReleaseGrapple().Run(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		if rej, ok := r.(combat.ReleaseGrappleRejected); !ok || rej.Rejection != combat.GrappleRejectionNotHeld {
			t.Errorf("%s: result = %#v, want rejected NotHeld", name, r)
		}
	}
	if inv, ok := grpAMustRelease(t, combat.ReleaseGrappleCommand{Grapples: set, Grappler: id("ogre"), Target: id("ogre")}).(combat.ReleaseGrappleInvalid); !ok || inv.Problem != combat.GrappleProblemSameGrapplerAndTarget {
		t.Errorf("releasing oneself = %#v, want Invalid SameGrapplerAndTarget", inv)
	}
	dropped, err := combat.NewDropGrappleRecord().Run(ctx, combat.DropGrappleRecordCommand{Grapples: set, Target: id("ally")})
	if err != nil {
		t.Fatal(err)
	}
	d := dropped.(combat.DropGrappleRecordApplied)
	if d.Grapples().Len() != 1 || !slices.Equal(kinds(d.Events()), []string{"grapple_record_dropped"}) {
		t.Errorf("drop = %d left, events %v, want one left and grapple_record_dropped", d.Grapples().Len(), kinds(d.Events()))
	}
	missing, err := combat.NewDropGrappleRecord().Run(ctx, combat.DropGrappleRecordCommand{Grapples: d.Grapples(), Target: id("ally")})
	if err != nil {
		t.Fatal(err)
	}
	if rej, ok := missing.(combat.DropGrappleRecordRejected); !ok || rej.Rejection != combat.GrappleRejectionNotGrappled {
		t.Errorf("dropping a record twice = %#v, want rejected NotGrappled", missing)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := combat.NewReleaseGrapple().Run(cancelled, combat.ReleaseGrappleCommand{Grapples: set, Grappler: id("ogre"), Target: id("hero")}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context err = %v, want context.Canceled", err)
	}
}

func grpAMustRelease(t *testing.T, cmd combat.ReleaseGrappleCommand) combat.ReleaseGrappleResult {
	t.Helper()
	r, err := combat.NewReleaseGrapple().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Requirement: doc/specs/grappling.md FR-049, FR-050, FR-051, FR-059 (encounter side, encounter-aggregate.md FR-073) - an encounter holds a hero grappled by an ogre: the grapple holds at 5 feet of a 5-foot range, ends out of range at 10 feet with the Grappled condition removed and one grapple_ended; a creature that is not grappled, a missing grappler and a negative distance are each refused or invalid.
// Path: unhappy (boundary).
func TestGrapplingEncounterCheckScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	e := grpAEncounter(t, "hero", "ally", "ogre", "troll")
	grab, err := combat.NewApplyConditionToCombatantCommand(e, id("hero"), combat.Grappled{Grappler: id("ogre")})
	if err != nil {
		t.Fatal(err)
	}
	gr, err := combat.NewApplyConditionToCombatant().Run(ctx, grab)
	if err != nil {
		t.Fatal(err)
	}
	e = gr.(combat.ApplyConditionToCombatantApplied).Encounter()
	check := func(target string, distance, rng int) combat.CheckCombatantGrappleResult {
		t.Helper()
		cmd, err := combat.NewCheckCombatantGrappleCommand(e, id(target), distance, rng)
		if err != nil {
			t.Fatal(err)
		}
		r, err := combat.NewCheckCombatantGrapple().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	holds, ok := check("hero", 5, 5).(combat.CheckCombatantGrappleApplied)
	if !ok || len(holds.Events()) != 0 {
		t.Fatalf("a distance equal to the range = %#v, want applied with no event", holds)
	}
	if m, _ := holds.Encounter().Find(id("hero")); !slices.Contains(m.Conditions().Effective(), content.ConditionGrappled) {
		t.Errorf("the hero must still be Grappled at the range")
	}
	out, ok := check("hero", 10, 5).(combat.CheckCombatantGrappleApplied)
	if !ok || !slices.Contains(kinds(out.Events()), "grapple_ended") {
		t.Fatalf("a distance beyond the range = %#v, want grapple_ended", out)
	}
	if m, _ := out.Encounter().Find(id("hero")); slices.Contains(m.Conditions().Effective(), content.ConditionGrappled) {
		t.Errorf("the hero must be free once the grapple ended out of range")
	}
	e = out.Encounter()
	if _, ok := check("hero", 10, 5).(combat.CheckCombatantGrappleRejected); !ok {
		t.Errorf("checking a creature that is no longer grappled must be refused")
	}
	if _, err := combat.NewCheckCombatantGrappleCommand(e, id("hero"), -1, 5); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative distance err = %v, want ErrInvalidInput", err)
	}
}
