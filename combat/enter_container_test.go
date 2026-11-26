package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ctKinds lists the Kind of each event.
func ctKinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// ctEnterInput is the entry of a Medium creature "p" into the container "m" of the fixture name, with the
// state the rules need: a swallow needs the container to hold p in a grapple, and p is Grappled.
func ctEnterInput(tb testing.TB, name string) combat.EnterContainerInput {
	tb.Helper()
	in := combat.EnterContainerInput{Container: gid("m"), Occupant: gid("p"), OccupantSize: content.SizeMedium, Terms: ctTerms(tb, name),
		Circumstance: combat.EntryCircumstanceNone, Attempt: rollOf(0)}
	if in.Terms.Mode() == combat.ContainmentModeSwallowed {
		in.Grapples = grapplesOf(tb, held("m", "p", hand(0)))
		in.OccupantConditions = condSet(tb, []combat.Condition{combat.Grappled{Grappler: gid("m")}})
	}
	return in
}

// ctEnter runs EnterContainer with a fixed roller and returns the result and the dice it was asked for.
func ctEnter(tb testing.TB, in combat.EnterContainerInput, faces ...int) (combat.EnterContainerResult, *dicetest.FixedRoller) {
	tb.Helper()
	cmd, err := combat.NewEnterContainerCommand(in)
	if err != nil {
		tb.Fatalf("NewEnterContainerCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewEnterContainer(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res, roller
}

func ctContained(tb testing.TB, res combat.EnterContainerResult) combat.EnterContainerContained {
	tb.Helper()
	c, ok := res.(combat.EnterContainerContained)
	if !ok {
		tb.Fatalf("result = %#v, want contained", res)
	}
	return c
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-022, FR-023, FR-033, FR-037, FR-031 - a failed entry save holds the creature: the save is a saving throw against the DC with no modifier of the engine's own, and the result carries the set, the damage due in full, the effects, the placement and the action the monster spends.
// Path: happy.
func TestEnterContainerFailedSaveHoldsTheCreature(t *testing.T) {
	t.Parallel()
	res, roller := ctEnter(t, ctEnterInput(t, "cube"), 11)
	got := ctContained(t, res)
	if reqs := roller.Requests(); len(reqs) != 1 || reqs[0].Notation() != "1d20" {
		t.Errorf("dice requested = %v, want one 1d20", reqs)
	}
	rolled, ok := got.Check().(combat.ContainmentTestRolled)
	if !ok || rolled.Result().Outcome() != d20.Failure || rolled.Result().Target() != 12 {
		t.Fatalf("check = %#v, want a rolled failure against DC 12", got.Check())
	}
	rec := got.Containments().ContainerOf(gid("p")).(combat.InContainer).Record()
	if rec.Container != gid("m") || rec.Terms() != ctTerms(t, "cube") || rec.Phase().IsTrapped() || rec.PartOrdinal() != 1 || rec.Size() != content.SizeMedium ||
		!slices.Equal(rec.AddedConditions(), []content.ConditionKind{content.ConditionRestrained}) {
		t.Errorf("record = %+v", rec)
	}
	due := got.DamageDue()
	if len(due) != 1 || due[0].Container != gid("m") || due[0].Target != gid("p") || due[0].Halved() || due[0].Cause() != combat.ContainmentDamageCauseEntry ||
		len(due[0].Terms()) != 1 || due[0].Terms()[0].Expression().Notation() != "3d6" {
		t.Errorf("damage due = %+v, want 3d6 acid in full", due)
	}
	if !slices.Equal(got.Conditions().Effective(), []content.ConditionKind{content.ConditionRestrained}) || got.Grapples().Len() != 0 {
		t.Errorf("conditions = %v, want Restrained", got.Conditions().Effective())
	}
	want := combat.ContainmentEffects{Suffocation: combat.SuffocationRuleAlways, NoVerbalSpells: true, TotalCover: true}
	if got.Effects() != want || got.Placement() != (combat.ContainedPlacement{Container: gid("m"), Occupant: gid("p")}) || got.ActionKind() != combat.ContainmentActionKindAction {
		t.Errorf("effects %+v placement %+v action %v", got.Effects(), got.Placement(), got.ActionKind())
	}
	if !slices.Equal(ctKinds(got.Events()), []string{"condition_applied", "occupant_contained"}) {
		t.Errorf("events = %v", ctKinds(got.Events()))
	}
	ev := got.Events()[1].(combat.OccupantContained)
	if ev.Mode() != combat.ContainmentModeEngulfed || !slices.Equal(ev.Combatants(), []combat.CombatantID{gid("p"), gid("m")}) {
		t.Errorf("OccupantContained = %+v", ev)
	}
	if !slices.Equal(got.Imposed(), []combat.ImposedCondition{{Kind: content.ConditionRestrained, Result: combat.ConditionImpositionAdded}}) {
		t.Errorf("imposed = %v", got.Imposed())
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-024, FR-025, Q-15 - a successful save leaves the creature free with half the damage due when the entry halves it; with a relocation it moves to a free space, and with none free it fails the save instead.
// Path: per case.
func TestEnterContainerSuccessfulSave(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "cube")
	in.RelocationSpaceAvailable = true
	res, _ := ctEnter(t, in, 12)
	saved, ok := res.(combat.EnterContainerResisted)
	if !ok {
		t.Fatalf("result = %#v, want resisted", res)
	}
	reloc, has := saved.Relocation()
	due := saved.DamageDue()
	if !has || reloc != (combat.RelocationDue{Occupant: gid("p"), WithinFeet: 5}) || len(due) != 1 || !due[0].Halved() || saved.Containments().Len() != 0 ||
		!saved.Conditions().Equal(in.OccupantConditions) || saved.ActionKind() != combat.ContainmentActionKindAction {
		t.Errorf("resisted = relocation %+v/%v damage %+v", reloc, has, due)
	}
	if rolled := saved.Check().(combat.ContainmentTestRolled); rolled.Result().Outcome() != d20.Success {
		t.Error("the check must be the success")
	}
	// Nowhere to relocate to: the target fails the save instead and is held with the full damage.
	in.RelocationSpaceAvailable = false
	got := ctContained(t, mustEnter(t, in, 12))
	if due := got.DamageDue(); len(due) != 1 || due[0].Halved() || got.Containments().Len() != 1 {
		t.Errorf("no free space: damage %+v, containments %d, want held and full damage", due, got.Containments().Len())
	}
	// No relocation in the terms: a success is a success whatever the flag says.
	whelm := ctEnterInput(t, "whelm")
	if res, _ := ctEnter(t, whelm, 15); reflect.TypeOf(res) != reflect.TypeOf(combat.EnterContainerResisted{}) {
		t.Errorf("Whelm success = %#v", res)
	} else if r := res.(combat.EnterContainerResisted); !r.DamageDue()[0].Halved() {
		t.Error("Whelm halves on a success")
	} else if _, has := r.Relocation(); has {
		t.Error("the Whelm relocates nobody")
	}
	// A save without half damage deals none on a success.
	kraken := ctEnterInput(t, "kraken")
	kraken.Attempt = rollOf(10)
	if res, _ := ctEnter(t, kraken, 20); len(res.(combat.EnterContainerResisted).DamageDue()) != 0 {
		t.Error("the Kraken deals no damage to a creature that saved")
	}
}

func mustEnter(tb testing.TB, in combat.EnterContainerInput, faces ...int) combat.EnterContainerResult {
	tb.Helper()
	res, _ := ctEnter(tb, in, faces...)
	return res
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-027, FR-022 - a creature that entered the cube's space saves with Disadvantage, combined with the attempt's own circumstances so one Advantage cancels it; a forced attempt draws no die and fails.
// Path: per case.
func TestEnterContainerRollMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		attempt                 d20.SaveAttempt
		circumstance            combat.EntryCircumstance
		faces                   []int
		request                 string
		held                    bool
	}{
		{"no circumstance", "FR-022", "happy", rollOf(0), combat.EntryCircumstanceNone, []int{12}, "1d20", false},
		{"entered the cube's space keeps the lower die", "FR-027", "unhappy (rule interaction)", rollOf(0), combat.EntryCircumstanceEnteredContainerSpace, []int{19, 5}, "2d20", true},
		{"one Advantage cancels it", "FR-027", "unhappy (rule interaction)", rollOf(0, modifier.Advantage{}), combat.EntryCircumstanceEnteredContainerSpace, []int{19}, "1d20", false},
		{"its own Disadvantage is not doubled", "FR-027", "unhappy (rule interaction)", rollOf(0, modifier.Disadvantage{}), combat.EntryCircumstanceEnteredContainerSpace, []int{19, 5}, "2d20", true},
		{"choosing to fail draws no die", "FR-022", "unhappy (rule interaction)", d20.ChooseToFail{}, combat.EntryCircumstanceNone, nil, "", true},
		{"an automatic failure draws no die", "FR-022", "unhappy (rule interaction)", d20.AutomaticFailure{}, combat.EntryCircumstanceEnteredContainerSpace, nil, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := ctEnterInput(t, "cube")
			in.Attempt, in.Circumstance, in.RelocationSpaceAvailable = c.attempt, c.circumstance, true
			res, roller := ctEnter(t, in, c.faces...)
			_, isContained := res.(combat.EnterContainerContained)
			if isContained != c.held {
				t.Errorf("result = %#v, held want %v [%s]", res, c.held, c.requirement)
			}
			reqs := roller.Requests()
			if (c.request == "") != (len(reqs) == 0) || (len(reqs) > 0 && reqs[0].Notation() != c.request) {
				t.Errorf("dice requested = %v, want %q", reqs, c.request)
			}
		})
	}
	// The caller's attempt is not changed by the Disadvantage added to the save.
	attempt := rollOf(0, modifier.Advantage{})
	in := ctEnterInput(t, "cube")
	in.Attempt, in.Circumstance = attempt, combat.EntryCircumstanceEnteredContainerSpace
	ctEnter(t, in, 12)
	if len(attempt.Circumstances) != 1 {
		t.Error("the command must not alter the attempt it was given")
	}
	bad := ctEnterInput(t, "behir")
	bad.Circumstance = combat.EntryCircumstanceEnteredContainerSpace
	if res, roller := ctEnter(t, bad); res != (combat.EnterContainerInvalid{Problem: combat.ContainmentProblemCircumstanceNotApplicable}) || len(roller.Requests()) != 0 {
		t.Errorf("a swallow with the cube's circumstance = %#v, want invalid and no die", res)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-028, FR-029, FR-030 - a swallow ends the grapple first (ConditionRemoved then GrappleEnded) and then applies the imposed conditions; an immune or already-held condition is reported and never recorded; the Kraken imposes Restrained only.
// Path: per case.
func TestEnterContainerSwallowEndsTheGrappleFirst(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "behir")
	got := ctContained(t, mustEnter(t, in, 1))
	if !slices.Equal(ctKinds(got.Events()), []string{"condition_removed", "grapple_ended", "condition_applied", "condition_applied", "occupant_contained"}) {
		t.Errorf("events = %v", ctKinds(got.Events()))
	}
	end := got.Events()[1].(combat.GrappleEnded)
	if end.Cause() != combat.GrappleEndCauseReleased || got.Grapples().Len() != 0 {
		t.Errorf("grapple end = %v, grapples %d, want released and none left", end.Cause(), got.Grapples().Len())
	}
	if !slices.Equal(got.Conditions().Effective(), []content.ConditionKind{content.ConditionBlinded, content.ConditionRestrained}) {
		t.Errorf("conditions = %v, want Blinded and Restrained and no Grappled", got.Conditions().Effective())
	}
	if len(in.Grapples.All()) != 1 || !in.OccupantConditions.Equal(ctEnterInput(t, "behir").OccupantConditions) {
		t.Error("the inputs must not change")
	}
	kraken := ctContained(t, mustEnter(t, ctEnterInput(t, "kraken"), 1))
	if !slices.Equal(kraken.Conditions().Effective(), []content.ConditionKind{content.ConditionRestrained}) || len(kraken.Imposed()) != 1 || kraken.ActionKind() != combat.ContainmentActionKindAction {
		t.Errorf("Kraken conditions = %v imposed %v", kraken.Conditions().Effective(), kraken.Imposed())
	}
	if ctContained(t, mustEnter(t, ctEnterInput(t, "behir"), 1)).ActionKind() != combat.ContainmentActionKindBonusAction {
		t.Error("the Behir's Swallow is a Bonus Action")
	}

	// Immune to Restrained and already Blinded: reported, never recorded.
	special := ctEnterInput(t, "behir")
	special.OccupantConditions = condSet(t, []combat.Condition{combat.Grappled{Grappler: gid("m")}, combat.Blinded{}}, content.ConditionRestrained)
	held := ctContained(t, mustEnter(t, special, 1))
	rec := held.Containments().ContainerOf(gid("p")).(combat.InContainer).Record()
	wantImposed := []combat.ImposedCondition{{Kind: content.ConditionBlinded, Result: combat.ConditionImpositionAlreadyHeld}, {Kind: content.ConditionRestrained, Result: combat.ConditionImpositionImmune}}
	if !slices.Equal(held.Imposed(), wantImposed) || len(rec.AddedConditions()) != 0 || slices.Contains(ctKinds(held.Events()), "condition_applied") {
		t.Errorf("imposed %v added %v events %v", held.Imposed(), rec.AddedConditions(), ctKinds(held.Events()))
	}
	// A host whose set lacks Grappled still gets the entry: no removal event.
	bare := ctEnterInput(t, "behir")
	bare.OccupantConditions = combat.ConditionSet{}
	if k := ctKinds(ctContained(t, mustEnter(t, bare, 1)).Events()); slices.Contains(k, "condition_removed") || !slices.Contains(k, "occupant_contained") {
		t.Errorf("events = %v", k)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-035, FR-020, FR-080, table 4.7.2 - an Engulf that grapples applies Grappled first and begins a numbered grapple; a creature that cannot be Grappled takes the damage only and nothing is recorded.
// Path: per case.
func TestEnterContainerEngulfGrappled(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "mound")
	got := ctContained(t, mustEnter(t, in, 14))
	g := got.Grapples().GrappleOf(gid("p")).(combat.InGrapple).Grapple()
	if g.Grappler != gid("m") || g.Part != combat.MustGraspingBodyPart("Mound 1") || g.EscapeDC != 14 || g.RangeFeet != 0 {
		t.Errorf("grapple = %+v, want m holds p with Mound 1, DC 14, range 0", g)
	}
	if !slices.Equal(ctKinds(got.Events()), []string{"condition_applied", "grapple_started", "condition_applied", "condition_applied", "occupant_contained"}) {
		t.Errorf("events = %v", ctKinds(got.Events()))
	}
	if !slices.Equal(got.Conditions().Effective(), []content.ConditionKind{content.ConditionBlinded, content.ConditionGrappled, content.ConditionRestrained}) {
		t.Errorf("conditions = %v", got.Conditions().Effective())
	}
	if imp := got.Imposed(); len(imp) != 3 || imp[0] != (combat.ImposedCondition{Kind: content.ConditionGrappled, Result: combat.ConditionImpositionAdded}) {
		t.Errorf("imposed = %v, want Grappled first", imp)
	}
	rec := got.Containments().ContainerOf(gid("p")).(combat.InContainer).Record()
	if slices.Contains(rec.AddedConditions(), content.ConditionGrappled) || len(rec.AddedConditions()) != 2 {
		t.Errorf("added = %v, want Blinded and Restrained only", rec.AddedConditions())
	}
	if len(got.DamageDue()) != 0 {
		t.Error("the mound's Engulf deals no damage")
	}

	// The Whelm holds two Smalls: the second gets ordinal 2, and a part already used in the grapples is skipped.
	whelm := ctEnterInput(t, "whelm")
	whelm.OccupantSize = content.SizeSmall
	first := ctContained(t, mustEnter(t, whelm, 1))
	second := whelm
	second.Occupant, second.Containments, second.Grapples = gid("q"), first.Containments(), first.Grapples()
	two := ctContained(t, mustEnter(t, second, 1))
	if q := two.Grapples().GrappleOf(gid("q")).(combat.InGrapple).Grapple(); q.Part != combat.MustGraspingBodyPart("Whelm 2") {
		t.Errorf("second part = %v, want Whelm 2", q.Part)
	}
	busy := ctEnterInput(t, "whelm")
	busy.Grapples = grapplesOf(t, combat.MustGrapple(gid("m"), gid("z"), combat.MustGraspingBodyPart("Whelm 1"), 14, 0))
	if p := ctContained(t, mustEnter(t, busy, 1)).Grapples().GrappleOf(gid("p")).(combat.InGrapple).Grapple().Part; p != combat.MustGraspingBodyPart("Whelm 2") {
		t.Errorf("part = %v, want Whelm 2 (Whelm 1 is already in use)", p)
	}

	// A Ghost is immune to Grappled: the mound's Engulf holds nothing and the Whelm still deals its damage.
	for _, name := range []string{"mound", "whelm"} {
		ghost := ctEnterInput(t, name)
		ghost.OccupantConditions = condSet(t, nil, content.ConditionGrappled)
		res, _ := ctEnter(t, ghost, 1)
		d, ok := res.(combat.EnterContainerDamagedOnly)
		if !ok || d.Reason() != combat.DamagedOnlyReasonGrappleImmune || d.Containments().Len() != 0 || d.Grapples().Len() != 0 || !d.Conditions().Equal(ghost.OccupantConditions) ||
			!slices.Equal(d.Imposed(), []combat.ImposedCondition{{Kind: content.ConditionGrappled, Result: combat.ConditionImpositionImmune}}) {
			t.Fatalf("%s vs a Ghost = %#v", name, res)
		}
		if (len(d.DamageDue()) == 1) != (name == "whelm") {
			t.Errorf("%s vs a Ghost: damage due %v", name, d.DamageDue())
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-026, FR-078 - an automatic entry draws no die, applies no save and holds the creature with no damage; the frog's first tick is deferred only when it swallows during its own turn, and the toad's never.
// Path: per case.
func TestEnterContainerAutomaticEntry(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		own      bool
		deferred bool
	}{{"frog, own turn", true, true}, {"frog, another creature's turn", false, false}, {"toad, own turn", true, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			fixture := "frog"
			if c.name[0] == 't' {
				fixture = "toad"
			}
			in := ctEnterInput(t, fixture)
			in.OccupantSize, in.DuringContainerTurn, in.Attempt = content.SizeSmall, c.own, nil
			res, roller := ctEnter(t, in)
			got := ctContained(t, res)
			rec := got.Containments().ContainerOf(gid("p")).(combat.InContainer).Record()
			if _, none := got.Check().(combat.ContainmentTestNotRequired); !none || len(roller.Requests()) != 0 || len(got.DamageDue()) != 0 || rec.TickDeferred() != c.deferred ||
				got.ActionKind() != combat.ContainmentActionKindAction {
				t.Errorf("check %#v dice %d damage %v deferred %v", got.Check(), len(roller.Requests()), got.DamageDue(), rec.TickDeferred())
			}
		})
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-019, FR-021, table 4.7.2 - every refusal comes before any die, in the order container contained, occupant a container, already contained, not grappled by the container, grappled elsewhere, too large, no room; the inputs are unchanged.
// Path: per case.
func TestEnterContainerRefusals(t *testing.T) {
	t.Parallel()
	worm := ctTerms(t, "worm")
	// A worm holding three Medium creatures: full.
	fullWorm := ctHolding(t, "worm", "m", "a", "b", "c")
	moundHolds := ctHolding(t, "mound", "m", "a")
	cases := []struct {
		name, path string
		fixture    string
		mod        func(*combat.EnterContainerInput)
		want       combat.ContainmentRejection
	}{
		{"the container is held", "unhappy (rule interaction)", "worm", func(in *combat.EnterContainerInput) {
			in.Containments = ctSet(t, ctRecord(t, "boss", "m", content.SizeLarge, ctTerms(t, "kraken"), 1))
		}, combat.ContainmentRejectionContainerIsContained},
		{"the occupant holds creatures", "unhappy (rule interaction)", "worm", func(in *combat.EnterContainerInput) {
			in.Containments = ctSet(t, ctRecord(t, "p", "x", content.SizeSmall, ctTerms(t, "toad"), 1))
		}, combat.ContainmentRejectionOccupantIsContainer},
		{"the occupant is already held", "unhappy (state error)", "worm", func(in *combat.EnterContainerInput) {
			in.Containments = ctSet(t, ctRecord(t, "other", "p", content.SizeMedium, worm, 1))
		}, combat.ContainmentRejectionOccupantAlreadyContained},
		{"a swallow of a creature nobody grapples", "unhappy (state error)", "worm", func(in *combat.EnterContainerInput) { in.Grapples = combat.Grapples{} }, combat.ContainmentRejectionNotGrappledByContainer},
		{"a swallow of a creature another monster grapples", "unhappy (state error)", "worm", func(in *combat.EnterContainerInput) {
			in.Grapples = grapplesOf(t, held("other", "p", hand(0)))
		}, combat.ContainmentRejectionNotGrappledByContainer},
		{"an engulf of a grappled creature", "unhappy (state error)", "mound", func(in *combat.EnterContainerInput) {
			in.Grapples = grapplesOf(t, held("other", "p", hand(0)))
		}, combat.ContainmentRejectionOccupantGrappledElsewhere},
		{"a Medium frog meal", "unhappy (boundary)", "frog", func(in *combat.EnterContainerInput) {}, combat.ContainmentRejectionTooLarge},
		{"a Large creature for the mound", "unhappy (boundary)", "mound", func(in *combat.EnterContainerInput) { in.OccupantSize = content.SizeLarge }, combat.ContainmentRejectionTooLarge},
		{"a Huge creature for the worm", "unhappy (boundary)", "worm", func(in *combat.EnterContainerInput) { in.OccupantSize = content.SizeHuge }, combat.ContainmentRejectionTooLarge},
		{"a fourth creature for the worm", "unhappy (boundary)", "worm", func(in *combat.EnterContainerInput) { in.Containments = fullWorm }, combat.ContainmentRejectionNoRoom},
		{"a second creature for the mound", "unhappy (boundary)", "mound", func(in *combat.EnterContainerInput) { in.Containments = moundHolds }, combat.ContainmentRejectionNoRoom},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := ctEnterInput(t, c.fixture)
			if c.fixture == "frog" {
				in.Grapples = grapplesOf(t, held("m", "p", hand(0)))
			}
			c.mod(&in)
			beforeSet, beforeGrapples := in.Containments.All(), in.Grapples.All()
			res, roller := ctEnter(t, in)
			if res != (combat.EnterContainerRejected{Rejection: c.want}) || len(roller.Requests()) != 0 {
				t.Errorf("result = %#v with %d dice, want %v and no die", res, len(roller.Requests()), c.want)
			}
			if !slices.Equal(in.Containments.All(), beforeSet) || !slices.Equal(in.Grapples.All(), beforeGrapples) {
				t.Error("a refusal must leave the inputs unchanged")
			}
		})
	}
	// The order is fixed even when everything is wrong at once.
	all := ctEnterInput(t, "frog")
	all.Containments = ctSet(t, ctRecord(t, "boss", "m", content.SizeLarge, ctTerms(t, "kraken"), 1), ctRecord(t, "p", "x", content.SizeSmall, ctTerms(t, "toad"), 1))
	if res := mustEnter(t, all); res != (combat.EnterContainerRejected{Rejection: combat.ContainmentRejectionContainerIsContained}) {
		t.Errorf("result = %#v, want the first rule in the order", res)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-032, FR-025, Q-15, table 4.7.2 - a container with no room or a target above the maximum size still has the save rolled when the entry deals damage: a failure deals the damage in full without holding the creature, a success is the usual success.
// Path: per case.
func TestEnterContainerDamageWithoutContainment(t *testing.T) {
	t.Parallel()
	cubeWith := func(held ...string) combat.Containments { return ctHolding(t, "cube", "m", held...) }
	cases := []struct {
		name, requirement string
		fixture           string
		mod               func(*combat.EnterContainerInput)
		failFaces         int
		reason            combat.DamagedOnlyReason
		damage            string
		grappled          bool
	}{
		{"cube holding a Large, a Medium arrives", "no room", "cube", func(in *combat.EnterContainerInput) {
			in.Containments = ctSet(t, ctRecord(t, "m", "big", content.SizeLarge, ctTerms(t, "cube"), 1))
		}, 1, combat.DamagedOnlyReasonNoRoom, "3d6", false},
		{"cube holding four Small, a fifth arrives", "no room", "cube", func(in *combat.EnterContainerInput) {
			in.OccupantSize, in.Containments = content.SizeSmall, cubeWith("a", "b", "c", "d")
		}, 1, combat.DamagedOnlyReasonNoRoom, "3d6", false},
		{"cube, a Huge creature", "too large", "cube", func(in *combat.EnterContainerInput) { in.OccupantSize = content.SizeHuge }, 1, combat.DamagedOnlyReasonTooLarge, "3d6", false},
		{"Whelm holding a Large, a Small arrives", "no room", "whelm", func(in *combat.EnterContainerInput) {
			in.OccupantSize = content.SizeSmall
			in.Containments = ctSet(t, ctRecord(t, "m", "big", content.SizeLarge, ctTerms(t, "whelm"), 1))
		}, 1, combat.DamagedOnlyReasonNoRoom, "4d8+4", false},
		{"Kraken holding four, a fifth arrives", "no room", "kraken", func(in *combat.EnterContainerInput) {
			in.Containments = ctHolding(t, "kraken", "m", "a", "b", "c", "d")
		}, 1, combat.DamagedOnlyReasonNoRoom, "3d8+10", true},
		{"Kraken, a Huge creature", "too large", "kraken", func(in *combat.EnterContainerInput) { in.OccupantSize = content.SizeHuge }, 1, combat.DamagedOnlyReasonTooLarge, "3d8+10", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := ctEnterInput(t, c.fixture)
			in.RelocationSpaceAvailable = true
			c.mod(&in)
			res, roller := ctEnter(t, in, c.failFaces)
			d, ok := res.(combat.EnterContainerDamagedOnly)
			if !ok || d.Reason() != c.reason || len(roller.Requests()) != 1 {
				t.Fatalf("result = %#v with %d dice, want damaged only (%v) after one save [%s]", res, len(roller.Requests()), c.reason, c.requirement)
			}
			due := d.DamageDue()
			if len(due) != 1 || due[0].Halved() || due[0].Terms()[0].Expression().Notation() != c.damage || d.Containments().Len() != in.Containments.Len() ||
				(d.Grapples().Len() == 1) != c.grappled || !d.Conditions().Equal(in.OccupantConditions) || d.ActionKind() != in.Terms.ActionKind() {
				t.Errorf("damaged only = damage %v containments %d grapples %d", due, d.Containments().Len(), d.Grapples().Len())
			}
			// The same entry with a successful save is a plain success.
			in.Attempt = rollOf(30)
			if res, _ := ctEnter(t, in, 20); reflect.TypeOf(res) != reflect.TypeOf(combat.EnterContainerResisted{}) {
				t.Errorf("a saved entry = %#v, want resisted", res)
			}
		})
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-034, FR-035, FR-012 - the same container and occupant is an Invalid result, not an error and with no die; the command constructor rejects the values no result can describe.
// Path: unhappy (invalid input).
func TestEnterContainerInvalidInput(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "cube")
	in.Occupant = in.Container
	res, roller := ctEnter(t, in)
	if res != (combat.EnterContainerInvalid{Problem: combat.ContainmentProblemSameContainerAndOccupant}) || len(roller.Requests()) != 0 {
		t.Errorf("same creature = %#v", res)
	}
	for name, mod := range map[string]func(*combat.EnterContainerInput){
		"blank container":   func(in *combat.EnterContainerInput) { in.Container = "" },
		"blank occupant":    func(in *combat.EnterContainerInput) { in.Occupant = "" },
		"size 0":            func(in *combat.EnterContainerInput) { in.OccupantSize = 0 },
		"zero terms":        func(in *combat.EnterContainerInput) { in.Terms = combat.ContainmentTerms{} },
		"circumstance 0":    func(in *combat.EnterContainerInput) { in.Circumstance = 0 },
		"circumstance 3":    func(in *combat.EnterContainerInput) { in.Circumstance = 3 },
		"no attempt":        func(in *combat.EnterContainerInput) { in.Attempt = nil },
		"a pointer attempt": func(in *combat.EnterContainerInput) { in.Attempt = &d20.RollAttempt{} },
		"container with other terms": func(in *combat.EnterContainerInput) {
			in.Containments = ctSet(t, ctRecord(t, "m", "x", content.SizeSmall, ctTerms(t, "mound"), 1))
		},
	} {
		bad := ctEnterInput(t, "cube")
		mod(&bad)
		if _, err := combat.NewEnterContainerCommand(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	// An automatic entry needs no attempt.
	auto := ctEnterInput(t, "frog")
	auto.Attempt = nil
	if _, err := combat.NewEnterContainerCommand(auto); err != nil {
		t.Errorf("an automatic entry with no attempt: %v", err)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-036, testing.md section 4 - equal commands with fixed dice give equal results, and the command is not altered by running it.
// Path: unhappy (determinism).
func TestEnterContainerIsDeterministic(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "cube")
	attempt := rollOf(2, modifier.Disadvantage{})
	in.Attempt, in.Circumstance = attempt, combat.EntryCircumstanceEnteredContainerSpace
	cmd, err := combat.NewEnterContainerCommand(in)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := combat.NewEnterContainerCommand(in)
	var results [2]combat.EnterContainerResult
	for i := range results {
		uc, _ := combat.NewEnterContainer(dicetest.NewFixedRoller(18, 3))
		if results[i], err = uc.Run(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(results[0], results[1]) || !reflect.DeepEqual(cmd, snapshot) {
		t.Error("results differ or the command changed")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-021, FR-032, FR-034 - the checks do not depend on how the ids sort: a container whose id sorts after the occupant's still holds it, a swallow of a creature another monster with a smaller id grapples is refused, and a save that deals damage is rolled for a full container even when the entry does not reach targets of any size.
// Path: per case.
func TestEnterContainerChecksDoNotDependOnIdOrder(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "cube")
	in.Container, in.Occupant = gid("z"), gid("a")
	if res := mustEnter(t, in, 1); reflect.TypeOf(res) != reflect.TypeOf(combat.EnterContainerContained{}) {
		t.Errorf("a container sorting after its occupant = %#v, want contained", res)
	}
	swallow := ctEnterInput(t, "worm")
	swallow.Grapples = grapplesOf(t, held("a", "p", hand(0)))
	if res := mustEnter(t, swallow); res != (combat.EnterContainerRejected{Rejection: combat.ContainmentRejectionNotGrappledByContainer}) {
		t.Errorf("a grapple by a monster sorting before the container = %#v", res)
	}
	// An entry with damage that cannot reach a creature too large, and a full container: the save is still rolled.
	terms := combat.MustContainmentTerms(combat.ContainmentTermsInput{Mode: combat.ContainmentModeEngulfed, MaxSize: content.SizeMedium, Capacity: ctCap(1, 1),
		Entry:       combat.MustEntryBySave(1, 12, ctDamages(ctDamage("1d6", content.DamageAcid)), false, 0, false),
		Suffocation: combat.SuffocationRuleNone, DeathRule: combat.ReleaseOnDeath{}, ActionKind: combat.ContainmentActionKindAction})
	full := combat.EnterContainerInput{Containments: ctSet(t, ctRecord(t, "m", "z", content.SizeMedium, terms, 1)), Container: gid("m"), Occupant: gid("p"),
		OccupantSize: content.SizeMedium, Terms: terms, Circumstance: combat.EntryCircumstanceNone, Attempt: rollOf(0)}
	res, roller := ctEnter(t, full, 1)
	if d, ok := res.(combat.EnterContainerDamagedOnly); !ok || d.Reason() != combat.DamagedOnlyReasonNoRoom || len(roller.Requests()) != 1 || len(d.DamageDue()) != 1 {
		t.Errorf("a full container, damage, no any-size = %#v with %d dice, want damaged only (no room) after one save", res, len(roller.Requests()))
	}
	full.OccupantSize = content.SizeLarge
	if res := mustEnter(t, full, 1); res != (combat.EnterContainerRejected{Rejection: combat.ContainmentRejectionTooLarge}) {
		t.Errorf("a too large creature, no any-size = %#v, want refused before any die", res)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-027 - the cube's circumstance is not applicable to an Engulf that grapples either.
// Path: unhappy (invalid input).
func TestEnterContainerCircumstanceNotApplicableToEngulfedGrappled(t *testing.T) {
	t.Parallel()
	in := ctEnterInput(t, "mound")
	in.Circumstance = combat.EntryCircumstanceEnteredContainerSpace
	if res, roller := ctEnter(t, in, 1); res != (combat.EnterContainerInvalid{Problem: combat.ContainmentProblemCircumstanceNotApplicable}) || len(roller.Requests()) != 0 {
		t.Errorf("result = %#v with %d dice", res, len(roller.Requests()))
	}
}
