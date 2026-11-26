package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-026, FR-033, FR-071 - NewGearHold returns a GearError for exactly the invalid cases: a Net that does not restrain, a Hunting Trap that does, a status on a kind other than a Net, a negative chain limit, a Hunting Trap without a gear id and any other kind with one; also a blank id or target, a kind or restraint that is not valid, and a chain limit on a kind other than a Hunting Trap (SRD 5.2.1, "Net", p. 98; "Hunting Trap", pp. 97-98).
// Path: per case.
func TestNewGearHoldInvalid(t *testing.T) {
	t.Parallel()
	status := combat.NetProfile{}.Status()
	cases := []struct {
		name   string
		change func(*combat.GearHoldInput)
		kind   combat.HoldKind
		want   combat.GearProblem
	}{
		{"net that does not restrain", func(in *combat.GearHoldInput) { in.Restraint = combat.HoldRestraintDoesNotRestrain }, combat.HoldKindNet, combat.GearProblemHoldKindMismatch},
		{"hunting trap that restrains", func(in *combat.GearHoldInput) { in.Restraint = combat.HoldRestraintRestrains }, combat.HoldKindHuntingTrap, combat.GearProblemHoldKindMismatch},
		{"status on a chain", func(in *combat.GearHoldInput) { in.Status = status }, combat.HoldKindChain, combat.GearProblemHoldKindMismatch},
		{"status on a hunting trap", func(in *combat.GearHoldInput) { in.Status = status }, combat.HoldKindHuntingTrap, combat.GearProblemHoldKindMismatch},
		{"chain limit on a rope", func(in *combat.GearHoldInput) { in.MovementLimitFeet = 1 }, combat.HoldKindRope, combat.GearProblemHoldKindMismatch},
		{"negative chain limit", func(in *combat.GearHoldInput) { in.MovementLimitFeet = -1 }, combat.HoldKindHuntingTrap, combat.GearProblemNegativeMovementLimit},
		{"smallest chain limit", func(in *combat.GearHoldInput) { in.MovementLimitFeet = math.MinInt }, combat.HoldKindHuntingTrap, combat.GearProblemNegativeMovementLimit},
		{"negative chain limit on a net", func(in *combat.GearHoldInput) { in.MovementLimitFeet = -1 }, combat.HoldKindNet, combat.GearProblemNegativeMovementLimit},
		{"hunting trap without a gear id", func(in *combat.GearHoldInput) { in.GearID = "" }, combat.HoldKindHuntingTrap, combat.GearProblemGearIDMismatch},
		{"hunting trap with a blank gear id", func(in *combat.GearHoldInput) { in.GearID = " " }, combat.HoldKindHuntingTrap, combat.GearProblemGearIDMismatch},
		{"manacles with a gear id", func(in *combat.GearHoldInput) { in.GearID = "trap" }, combat.HoldKindManacles, combat.GearProblemGearIDMismatch},
		{"net with a gear id", func(in *combat.GearHoldInput) { in.GearID = "trap" }, combat.HoldKindNet, combat.GearProblemGearIDMismatch},
		{"blank id", func(in *combat.GearHoldInput) { in.ID = " " }, combat.HoldKindChain, combat.GearProblemBlankID},
		{"zero target", func(in *combat.GearHoldInput) { in.Target = "" }, combat.HoldKindChain, combat.GearProblemBlankID},
		{"invalid kind", func(in *combat.GearHoldInput) { in.Kind = 0 }, 0, combat.GearProblemInvalidValue},
		{"invalid restraint", func(in *combat.GearHoldInput) { in.Restraint = 0 }, combat.HoldKindRope, combat.GearProblemInvalidValue},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := holdIn("h", c.kind)
			c.change(&in)
			h, err := combat.NewGearHold(in)
			if !h.IsZero() || !errors.Is(err, &combat.GearError{Problem: c.want}) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewGearHold = %+v, %v; want a %v GearError [%s FR-026]", h, err, c.want, gearSpec)
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-026, FR-030, FR-033, table 4.9b - NewGearHold keeps every part, defaults the escape Difficulty Class to the profile's and holds any supplied integer as given (also the extremes), gives a Net without a status the Net's profile, and compares by value on all parts.
// Path: happy; unhappy (boundary).
func TestNewGearHold(t *testing.T) {
	t.Parallel()
	for kind, dc := range map[combat.HoldKind]d20.DifficultyClass{combat.HoldKindNet: 10, combat.HoldKindChain: 18, combat.HoldKindManacles: 20, combat.HoldKindRope: 15, combat.HoldKindHuntingTrap: 13} {
		in := holdIn("h", kind)
		if kind == combat.HoldKindManacles {
			in.Restraint = combat.HoldRestraintDoesNotRestrain
		}
		h := gearHold(t, in)
		if h.ID() != "h" || h.Kind() != kind || !h.IsOn(gid("victim")) || h.IsOn(gid("x")) || h.EscapeDifficultyClass() != dc || h.Restraint() != in.Restraint {
			t.Errorf("%v: hold = %+v, want escape DC %d", kind, h, dc)
		}
		if h != gearHold(t, in) {
			t.Errorf("%v: equal inputs must give equal holds", kind)
		}
	}
	for _, dc := range []d20.DifficultyClass{0, -5, 1, math.MaxInt, math.MinInt} {
		in := holdIn("h", combat.HoldKindChain)
		in.EscapeDC = combat.DCOverrideOf(dc)
		if got := gearHold(t, in).EscapeDifficultyClass(); got != dc {
			t.Errorf("override %d held as %d, want it as given", dc, got)
		}
	}
	in := holdIn("h", combat.HoldKindChain)
	in.EscapeDC = combat.NoDCOverride()
	if gearHold(t, in).EscapeDifficultyClass() != 18 || (combat.DCOverride{}) != combat.NoDCOverride() {
		t.Error("no override must give the profile's Difficulty Class")
	}
	net := gearHold(t, holdIn("h", combat.HoldKindNet))
	if net.Status() != (combat.NetProfile{}).Status() || net.GearID() != "" || net.MovementLimitFeet() != 0 {
		t.Errorf("net = %+v, want the Net's profile status", net)
	}
	hurt := combat.NetProfile{}.Status()
	withStatus := holdIn("h", combat.HoldKindNet)
	withStatus.Status = hurt
	if gearHold(t, withStatus) != net {
		t.Error("a Net given the profile status must equal one given none")
	}
	trap := gearHold(t, holdIn("h", combat.HoldKindHuntingTrap))
	if trap.GearID() != "trap" || trap.MovementLimitFeet() != 3 || !trap.Status().IsZero() {
		t.Errorf("trap = %+v, want gear id trap and a limit of 3", trap)
	}
	zeroLimit := holdIn("h", combat.HoldKindHuntingTrap)
	zeroLimit.MovementLimitFeet = 0
	if gearHold(t, zeroLimit).MovementLimitFeet() != 0 {
		t.Error("a limit of 0 is valid for a Hunting Trap")
	}
}

// FuzzNewGearHold checks NewGearHold never panics and, when it succeeds, returns a hold whose parts are the input's.
func FuzzNewGearHold(f *testing.F) {
	f.Add("h", "t", 1, 1, "", 0, 0, false, 0)
	f.Add("h", "t", 5, 2, "trap", 3, 7, true, 1)
	f.Add("", "", 0, 0, " ", -1, math.MinInt, false, 9)
	f.Add("h", "t", 3, 1, "", math.MaxInt, math.MaxInt, true, -3)
	f.Fuzz(func(t *testing.T, id, target string, kind, restraint int, gear string, limit, dc int, override bool, _ int) {
		in := combat.GearHoldInput{ID: combat.HoldID(id), Kind: combat.HoldKind(kind), Target: combat.CombatantID(target),
			Restraint: combat.HoldRestraint(restraint), GearID: combat.DeployedGearID(gear), MovementLimitFeet: limit}
		if override {
			in.EscapeDC = combat.DCOverrideOf(d20.DifficultyClass(dc))
		}
		h, err := combat.NewGearHold(in)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !h.IsZero() {
				t.Fatalf("error %v must match ErrInvalidInput with the zero hold", err)
			}
			return
		}
		if h.ID() != in.ID || h.Kind() != in.Kind || h.MovementLimitFeet() != limit || h.Restraint() != in.Restraint || (override && h.EscapeDifficultyClass() != d20.DifficultyClass(dc)) {
			t.Fatalf("hold = %+v for %+v", h, in)
		}
		if _, err := combat.NewGearHolds([]combat.GearHold{h}); err != nil {
			t.Fatal(err)
		}
	})
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-027, FR-034, Q-10 - NewGearHolds rejects a zero hold, two holds with one id, and two with one target and kind, except Hunting Trap holds, whose key also names the trap; different kinds on one target are allowed; the set copies its input, returns copies, compares equal whatever the order, lists holds in the order begun and answers HoldsOn and HoldOf with an ok boolean.
// Path: unhappy (state error, immutability).
func TestNewGearHolds(t *testing.T) {
	t.Parallel()
	net := gearHold(t, holdIn("n", combat.HoldKindNet))
	cuffs := holdIn("m", combat.HoldKindManacles)
	manacles := gearHold(t, cuffs)
	trap1 := gearHold(t, holdIn("t1", combat.HoldKindHuntingTrap))
	two := holdIn("t2", combat.HoldKindHuntingTrap)
	two.GearID = "trap2"
	trap2 := gearHold(t, two)
	same := holdIn("t3", combat.HoldKindHuntingTrap)
	sameTrap := gearHold(t, same)
	other := holdIn("n2", combat.HoldKindNet)
	other.Target = gid("somebody")
	elsewhere := gearHold(t, other)
	bad := map[string][]combat.GearHold{
		"zero hold":         {{}},
		"one id":            {net, gearHold(t, func() combat.GearHoldInput { in := holdIn("n", combat.HoldKindChain); return in }())},
		"two nets":          {net, gearHold(t, holdIn("n9", combat.HoldKindNet))},
		"one trap twice":    {trap1, sameTrap},
		"two nets (second)": {elsewhere, net, gearHold(t, holdIn("n8", combat.HoldKindNet))},
	}
	for name, hs := range bad {
		if _, err := combat.NewGearHolds(hs); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	in := []combat.GearHold{net, manacles, trap1, trap2, elsewhere}
	set, err := combat.NewGearHolds(in)
	if err != nil {
		t.Fatalf("different kinds, two traps and another target must be allowed: %v", err)
	}
	in[0] = combat.GearHold{}
	if set.Len() != 5 || set.Holds()[0] != net {
		t.Error("the set must copy its input")
	}
	view := set.Holds()
	view[0] = combat.GearHold{}
	if set.Holds()[0].IsZero() || !slices.Equal(set.HoldsOn(gid("victim")), []combat.GearHold{net, manacles, trap1, trap2}) {
		t.Error("views must be copies in the order begun")
	}
	on := set.HoldsOn(gid("victim"))
	on[0] = combat.GearHold{}
	if h, ok := set.HoldOf("n"); !ok || h != net {
		t.Error("HoldOf must find a hold")
	}
	if h, ok := set.HoldOf("zz"); ok || !h.IsZero() {
		t.Error("HoldOf an absent id must give the zero hold and false")
	}
	if len(set.HoldsOn(gid("nobody"))) != 0 {
		t.Error("HoldsOn nobody must be empty")
	}
	rev, _ := combat.NewGearHolds([]combat.GearHold{elsewhere, trap2, trap1, manacles, net})
	if !set.Equal(rev) || !rev.Equal(set) || set.Equal(gearHolds(t, net)) || gearHolds(t, net).Equal(gearHolds(t, manacles)) {
		t.Error("sets must compare equal whatever the order")
	}
	if empty, err := combat.NewGearHolds(nil); err != nil || empty.Len() != 0 || !empty.Equal(combat.GearHolds{}) {
		t.Errorf("nil input = %+v, %v; want the empty set", empty, err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-027, FR-028, FR-029, FR-030, table 4.9.1 (second trap) - Begin emits one GearHoldStarted and reports restraintBegins exactly when the hold restrains and no other hold in the set restrains the target; End removes the hold with one GearHoldEnded and reports restraintEnds exactly when it restrained and no other hold still does; a hold that does not restrain never begins or ends a restraint; a second hold of one kind on a target is refused targetAlreadyHeld (a second Hunting Trap is not), a reused id idInUse, an unknown id unknownHold (SRD 5.2.1, "Net", "Chain", pp. 96, 98; "Restrained", p. 187).
// Path: per case.
func TestGearHoldsBeginAndEnd(t *testing.T) {
	t.Parallel()
	net, chain := holdIn("net", combat.HoldKindNet), holdIn("chain", combat.HoldKindChain)
	loose := holdIn("rope", combat.HoldKindRope)
	loose.Restraint = combat.HoldRestraintDoesNotRestrain
	trap1, trap2 := holdIn("t1", combat.HoldKindHuntingTrap), holdIn("t2", combat.HoldKindHuntingTrap)
	trap2.GearID = "trap2"
	begin := func(set combat.GearHolds, in combat.GearHoldInput) combat.GearHoldsTransition {
		tr, err := set.Begin(gearHold(t, in))
		if err != nil {
			t.Fatal(err)
		}
		return tr
	}
	first := holdsApplied(t, begin(combat.GearHolds{}, net), nil)
	ev := first.Events()
	combattest.CheckEvent(t, ev[0], "gear_hold_started", gid("victim"))
	if !first.RestraintBegins() || first.RestraintEnds() || first.Holds().Len() != 1 || len(ev) != 1 || ev[0].(combat.GearHoldStarted).HoldKind() != combat.HoldKindNet || ev[0].(combat.GearHoldStarted).HoldID() != "net" {
		t.Fatalf("first hold = %+v, want restraintBegins with one event", first)
	}
	second := holdsApplied(t, begin(first.Holds(), chain), nil)
	if second.RestraintBegins() || !first.Holds().RestrainsTarget(gid("victim")) || first.Holds().Len() != 1 {
		t.Error("a second restraining hold must not begin the restraint again and Begin must not change the set")
	}
	if r := holdsApplied(t, begin(combat.GearHolds{}, loose), nil); r.RestraintBegins() || r.Holds().RestrainsTarget(gid("victim")) {
		t.Error("a hold that does not restrain must not begin the restraint")
	}
	both := second.Holds()
	endNet, err := both.End("net", combat.HoldEndCauseEscaped)
	a := holdsApplied(t, endNet, err)
	if a.RestraintEnds() || !a.Holds().RestrainsTarget(gid("victim")) || a.RestraintBegins() {
		t.Error("ending one of two restraining holds must not end the restraint")
	}
	combattest.CheckEvent(t, a.Events()[0], "gear_hold_ended", gid("victim"))
	if e := a.Events()[0].(combat.GearHoldEnded); e.Cause() != combat.HoldEndCauseEscaped || e.HoldKind() != combat.HoldKindNet || e.HoldID() != "net" {
		t.Errorf("event = %+v", e)
	}
	endChain, err := a.Holds().End("chain", combat.HoldEndCauseBurst)
	last := holdsApplied(t, endChain, err)
	if !last.RestraintEnds() || last.Holds().Len() != 0 || last.Holds().RestrainsTarget(gid("victim")) {
		t.Error("ending the last restraining hold must end the restraint")
	}
	onlyLoose := holdsApplied(t, begin(combat.GearHolds{}, loose), nil).Holds()
	endLoose, err := onlyLoose.End("rope", combat.HoldEndCauseEscaped)
	if holdsApplied(t, endLoose, err).RestraintEnds() {
		t.Error("ending a hold that did not restrain must not end a restraint")
	}
	mixed := holdsApplied(t, begin(holdsApplied(t, begin(combat.GearHolds{}, loose), nil).Holds(), chain), nil).Holds()
	endMixed, err := mixed.End("chain", combat.HoldEndCauseBurst)
	if !holdsApplied(t, endMixed, err).RestraintEnds() {
		t.Error("ending the only restraining hold beside one that does not restrain must end the restraint")
	}

	// A second trap on the same creature is a second hold; the same trap, or a second Net, is refused.
	traps := holdsApplied(t, begin(holdsApplied(t, begin(combat.GearHolds{}, trap1), nil).Holds(), trap2), nil).Holds()
	if traps.Len() != 2 {
		t.Error("a creature caught in one trap may be caught by another (Q-10)")
	}
	refusals := []struct {
		name string
		set  combat.GearHolds
		in   combat.GearHoldInput
		want combat.GearRejection
	}{
		{"second net on a target", first.Holds(), holdIn("net2", combat.HoldKindNet), combat.GearRejectionTargetAlreadyHeld},
		{"same trap again", traps, func() combat.GearHoldInput { in := trap1; in.ID = "t3"; return in }(), combat.GearRejectionTargetAlreadyHeld},
		{"id in use", first.Holds(), func() combat.GearHoldInput { in := chain; in.ID = "net"; return in }(), combat.GearRejectionIDInUse},
		{"same id and same kind", first.Holds(), func() combat.GearHoldInput { in := net; return in }(), combat.GearRejectionTargetAlreadyHeld},
	}
	for _, c := range refusals {
		if r, ok := begin(c.set, c.in).(combat.GearHoldsRejected); !ok || r.Reason() != c.want {
			t.Errorf("%s: Begin must be refused with %v", c.name, c.want)
		}
	}
	if r, ok := mustEnd(t, first.Holds(), "zz", combat.HoldEndCauseEscaped).(combat.GearHoldsRejected); !ok || r.Reason() != combat.GearRejectionUnknownHold {
		t.Error("End of an unknown id must be refused with unknownHold")
	}
	if _, err := first.Holds().Begin(combat.GearHold{}); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("Begin of the zero hold: err = %v, want invalid_value", err)
	}
	if _, err := first.Holds().End("net", 0); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("End with an invalid cause: err = %v, want invalid_value", err)
	}
	// A hold on one creature never restrains another.
	other := net
	other.ID, other.Target = "n2", "somebody"
	if r := holdsApplied(t, begin(first.Holds(), other), nil); !r.RestraintBegins() || r.Holds().RestrainsTarget(gid("nobody")) {
		t.Error("restraint is per target")
	}
}

func mustEnd(t *testing.T, s combat.GearHolds, id combat.HoldID, cause combat.HoldEndCause) combat.GearHoldsTransition {
	t.Helper()
	tr, err := s.End(id, cause)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-031 - AttackRollCircumstances returns [Disadvantage] while a Manacles hold is on the target, restraining or not, and an empty list for every other kind and every other creature (SRD 5.2.1, "Manacles", p. 98).
// Path: happy; unhappy (rule interaction).
func TestGearHoldsAttackRollCircumstances(t *testing.T) {
	t.Parallel()
	loose := holdIn("m", combat.HoldKindManacles)
	loose.Restraint = combat.HoldRestraintDoesNotRestrain
	fixed := holdIn("m", combat.HoldKindManacles)
	for name, in := range map[string]combat.GearHoldInput{"loose manacles": loose, "fixed manacles": fixed} {
		got := gearHolds(t, gearHold(t, in)).AttackRollCircumstances(gid("victim"))
		if len(got) != 1 || got[0] != (modifier.Disadvantage{}) {
			t.Errorf("%s: circumstances = %v, want [Disadvantage]", name, got)
		}
	}
	holds := gearHolds(t, gearHold(t, holdIn("n", combat.HoldKindNet)), gearHold(t, holdIn("c", combat.HoldKindChain)))
	if got := holds.AttackRollCircumstances(gid("victim")); got == nil || len(got) != 0 {
		t.Errorf("net and chain: circumstances = %v, want an empty list", got)
	}
	if got := gearHolds(t, gearHold(t, fixed)).AttackRollCircumstances(gid("other")); len(got) != 0 {
		t.Errorf("another creature: circumstances = %v, want none", got)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-032 - Reanchor changes a Manacles hold's restraint with one GearHoldRestraintChanged and reports restraintBegins or restraintEnds by FR-028 and FR-029; the same restraint changes and emits nothing; other kinds are refused methodNotAvailable and an unknown id unknownHold (SRD 5.2.1, "Manacles", p. 98).
// Path: per case.
func TestGearHoldsReanchor(t *testing.T) {
	t.Parallel()
	loose := holdIn("m", combat.HoldKindManacles)
	loose.Restraint = combat.HoldRestraintDoesNotRestrain
	chain := holdIn("c", combat.HoldKindChain)
	reanchor := func(set combat.GearHolds, id combat.HoldID, r combat.HoldRestraint) combat.GearHoldsApplied {
		tr, err := set.Reanchor(id, r)
		return holdsApplied(t, tr, err)
	}
	set := gearHolds(t, gearHold(t, loose))
	fixed := reanchor(set, "m", combat.HoldRestraintRestrains)
	combattest.CheckEvent(t, fixed.Events()[0], "gear_hold_restraint_changed", gid("victim"))
	if e := fixed.Events()[0].(combat.GearHoldRestraintChanged); e.Restraint() != combat.HoldRestraintRestrains || e.HoldID() != "m" ||
		!fixed.RestraintBegins() || fixed.RestraintEnds() || !fixed.Holds().RestrainsTarget(gid("victim")) {
		t.Errorf("attaching loose manacles = %+v, want restraintBegins", fixed)
	}
	freed := reanchor(fixed.Holds(), "m", combat.HoldRestraintDoesNotRestrain)
	if !freed.RestraintEnds() || freed.RestraintBegins() || freed.Holds().RestrainsTarget(gid("victim")) {
		t.Errorf("taking fixed manacles off = %+v, want restraintEnds", freed)
	}
	with := gearHolds(t, gearHold(t, loose), gearHold(t, chain))
	if r := reanchor(with, "m", combat.HoldRestraintRestrains); r.RestraintBegins() {
		t.Error("manacles beside a restraining chain must not begin a second restraint")
	}
	bothFixed := reanchor(with, "m", combat.HoldRestraintRestrains).Holds()
	if r := reanchor(bothFixed, "m", combat.HoldRestraintDoesNotRestrain); r.RestraintEnds() {
		t.Error("loosening manacles beside a restraining chain must not end the restraint")
	}
	same := reanchor(set, "m", combat.HoldRestraintDoesNotRestrain)
	if len(same.Events()) != 0 || same.RestraintBegins() || same.RestraintEnds() || !same.Holds().Equal(set) {
		t.Errorf("no change = %+v, want nothing changed and no event", same)
	}
	for name, c := range map[string]struct {
		id   combat.HoldID
		want combat.GearRejection
	}{"a chain": {"c", combat.GearRejectionMethodNotAvailable}, "unknown": {"zz", combat.GearRejectionUnknownHold}} {
		tr, err := with.Reanchor(c.id, combat.HoldRestraintRestrains)
		if r, ok := tr.(combat.GearHoldsRejected); err != nil || !ok || r.Reason() != c.want {
			t.Errorf("%s: Reanchor = %#v, %v; want %v", name, tr, err, c.want)
		}
	}
	if _, err := set.Reanchor("m", 0); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("Reanchor with an invalid restraint: err = %v, want invalid_value", err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-042, FR-043 - RecordNetDamage stores an undestroyed status, ends the hold with destroyed (and restraintEnds by FR-029) for a destroyed one, refuses a hold that is not a Net notDestructible, and recording a destroyed status twice, or on an absent hold, refuses unknownHold the second time (SRD 5.2.1, "Net", p. 98; "Breaking Objects", p. 178).
// Path: per case.
func TestGearHoldsRecordNetDamage(t *testing.T) {
	t.Parallel()
	net := combat.NetProfile{}.Status()
	hurt, err := combat.NewObjectStatus(net.Size(), net.ArmorClass(), combat.MustHitPoints(2, 5, 0), net.Threshold(), net.Affinities(), nil)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := combat.NewObjectStatus(net.Size(), net.ArmorClass(), combat.MustHitPoints(0, 5, 0), net.Threshold(), net.Affinities(), nil)
	if err != nil {
		t.Fatal(err)
	}
	set := gearHolds(t, gearHold(t, holdIn("n", combat.HoldKindNet)), gearHold(t, holdIn("c", combat.HoldKindChain)))
	record := func(set combat.GearHolds, id combat.HoldID, s combat.ObjectStatus) combat.GearHoldsTransition {
		tr, err := set.RecordNetDamage(id, s)
		if err != nil {
			t.Fatal(err)
		}
		return tr
	}
	a := record(set, "n", hurt).(combat.GearHoldsApplied)
	h, _ := a.Holds().HoldOf("n")
	if h.Status() != hurt || len(a.Events()) != 0 || a.RestraintEnds() || a.Holds().Len() != 2 {
		t.Errorf("undestroyed Net = %+v, want the status stored and nothing else", a)
	}
	if orig, _ := set.HoldOf("n"); orig.Status() != net {
		t.Error("RecordNetDamage must not change the set it was called on")
	}
	d := record(set, "n", dead).(combat.GearHoldsApplied)
	if _, ok := d.Holds().HoldOf("n"); ok || d.RestraintEnds() || d.Events()[0].(combat.GearHoldEnded).Cause() != combat.HoldEndCauseDestroyed {
		t.Errorf("destroyed Net with a restraining chain = %+v, want the hold ended and the restraint kept", d)
	}
	alone := gearHolds(t, gearHold(t, holdIn("n", combat.HoldKindNet)))
	if r := record(alone, "n", dead).(combat.GearHoldsApplied); !r.RestraintEnds() {
		t.Error("destroying the only restraining hold must end the restraint")
	}
	for name, c := range map[string]struct {
		set  combat.GearHolds
		id   combat.HoldID
		want combat.GearRejection
	}{
		"not a net":                {set, "c", combat.GearRejectionNotDestructible},
		"destroyed twice":          {d.Holds(), "n", combat.GearRejectionUnknownHold},
		"absent hold":              {set, "zz", combat.GearRejectionUnknownHold},
		"absent hold, empty set":   {combat.GearHolds{}, "n", combat.GearRejectionUnknownHold},
		"not a net, destroyed one": {set, "c", combat.GearRejectionNotDestructible},
	} {
		if r, ok := record(c.set, c.id, dead).(combat.GearHoldsRejected); !ok || r.Reason() != c.want {
			t.Errorf("%s: must be refused with %v", name, c.want)
		}
	}
	if _, err := set.RecordNetDamage("n", combat.ObjectStatus{}); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("zero status: err = %v, want invalid_value", err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-071 - NewHoldID rejects a blank string with a blank_id GearError, keeps a valid id as given, and MustHoldID panics with that error and only then.
// Path: unhappy (invalid input).
func TestNewHoldID(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "  "} {
		if _, err := combat.NewHoldID(s); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemBlankID}) {
			t.Errorf("NewHoldID(%q) error = %v, want blank_id", s, err)
		}
	}
	if combat.MustHoldID(" h ") != " h " {
		t.Error("MustHoldID must keep a valid id as given")
	}
	defer func() {
		if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
			t.Errorf("recover() = %v, want an error matching ErrInvalidInput", r)
		}
	}()
	combat.MustHoldID("")
}
