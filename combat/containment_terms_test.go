package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// invalidTerms asserts err is the typed error of every terms constructor.
func invalidTerms(t *testing.T, err error, what string) {
	t.Helper()
	var ce *combat.ContainmentError
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ce) || ce.Problem != combat.ContainmentProblemInvalidTerms ||
		!errors.Is(err, &combat.ContainmentError{Problem: combat.ContainmentProblemInvalidTerms}) {
		t.Errorf("%s: err = %v, want a *ContainmentError matching ErrInvalidInput with InvalidTerms", what, err)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-001 - ContainmentMode has exactly the three members swallowed, engulfed and engulfed_grappled.
// Path: happy.
func TestContainmentModeHasExactlyThreeMembers(t *testing.T) {
	t.Parallel()
	for m, want := range map[combat.ContainmentMode]string{combat.ContainmentModeSwallowed: "swallowed", combat.ContainmentModeEngulfed: "engulfed",
		combat.ContainmentModeEngulfedGrappled: "engulfed_grappled", 0: "unknown", 4: "unknown"} {
		if m.String() != want || m.Valid() != (want != "unknown") {
			t.Errorf("mode %d = %q valid %v, want %q", int(m), m.String(), m.Valid(), want)
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-002, FR-003 - a capacity has 1 or more slots and each cost between 1 and the slots; Cost returns the mapped cost or 1, so one Large creature or four Medium ones fit the cube.
// Path: per case.
func TestContainmentCapacity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		slots      int
		costs      map[content.CreatureSize]int
		ok         bool
	}{
		{"no slots", "unhappy (boundary)", 0, nil, false},
		{"negative slots", "unhappy (boundary)", -1, nil, false},
		{"one slot", "unhappy (boundary)", 1, nil, true},
		{"cost zero", "unhappy (boundary)", 4, map[content.CreatureSize]int{content.SizeLarge: 0}, false},
		{"cost one above the slots", "unhappy (boundary)", 4, map[content.CreatureSize]int{content.SizeLarge: 5}, false},
		{"cost equal to the slots", "unhappy (boundary)", 4, map[content.CreatureSize]int{content.SizeLarge: 4}, true},
		{"size zero", "unhappy (invalid input)", 4, map[content.CreatureSize]int{0: 1}, false},
		{"size past the last", "unhappy (invalid input)", 4, map[content.CreatureSize]int{content.SizeGargantuan + 1: 1}, false},
		{"maximum slots", "unhappy (boundary)", math.MaxInt, map[content.CreatureSize]int{content.SizeHuge: math.MaxInt}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewContainmentCapacity(c.slots, c.costs)
			if c.ok {
				if err != nil || got.Slots() != c.slots {
					t.Fatalf("NewContainmentCapacity(%d) = %v, %v, want %d slots", c.slots, got, err, c.slots)
				}
				for size, cost := range c.costs {
					if got.Cost(size) != cost {
						t.Errorf("Cost(%v) = %d, want %d", size, got.Cost(size), cost)
					}
				}
				return
			}
			invalidTerms(t, err, c.name)
		})
	}
	cube := ctCap(4, 4)
	if cube.Cost(content.SizeLarge) != 4 || cube.Cost(content.SizeMedium) != 1 || cube.Cost(content.SizeTiny) != 1 || cube.Cost(0) != 1 {
		t.Errorf("cube costs = Large %d Medium %d Tiny %d, want 4, 1, 1 (Q-7)", cube.Cost(content.SizeLarge), cube.Cost(content.SizeMedium), cube.Cost(content.SizeTiny))
	}
	if ctCap(4, 1) != combat.MustContainmentCapacity(4, map[content.CreatureSize]int{content.SizeLarge: 1}) {
		t.Error("an explicit cost of 1 must equal the default: equal capacities compare equal")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-004 - a DamageTerm holds a dice expression with at least one die and a damage type; a fixed number or an invalid type is an error.
// Path: unhappy (invalid input).
func TestDamageTerm(t *testing.T) {
	t.Parallel()
	good, err := combat.NewDamageTerm(dice.MustParseExpression("3d6"), content.DamageAcid)
	if err != nil || good.Expression().Notation() != "3d6" || good.DamageType() != content.DamageAcid {
		t.Fatalf("NewDamageTerm(3d6, acid) = %v, %v", good, err)
	}
	_, err = combat.NewDamageTerm(dice.MustParseExpression("1d4"), content.DamageFire)
	if err != nil {
		t.Errorf("one die is enough: %v", err)
	}
	_, err = combat.NewDamageTerm(dice.Expression{}, content.DamageAcid)
	invalidTerms(t, err, "a fixed number with no dice")
	_, err = combat.NewDamageTerm(dice.MustParseExpression("3d6"), 0)
	invalidTerms(t, err, "an invalid damage type")
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-005 - EntryBySave needs a valid ability, a DC of 1 or more, up to the allowed damage terms, halfOnSuccess only with damage and a relocation of 1 or more when present.
// Path: per case.
func TestEntryBySave(t *testing.T) {
	t.Parallel()
	acid := ctDamages(ctDamage("3d6", content.DamageAcid))
	three := ctDamages(ctDamage("1d4", content.DamageAcid), ctDamage("1d4", content.DamageFire), ctDamage("1d4", content.DamageCold))
	cases := []struct {
		name, path string
		ab         ability.Ability
		dc         d20.DifficultyClass
		damage     []combat.DamageTerm
		half       bool
		relocate   int
		ok         bool
	}{
		{"dc 0", "unhappy (boundary)", ability.Dexterity, 0, acid, false, 0, false},
		{"dc 1", "unhappy (boundary)", ability.Dexterity, 1, acid, false, 0, true},
		{"invalid ability", "unhappy (invalid input)", 0, 12, acid, false, 0, false},
		{"half without damage", "unhappy (rule interaction)", ability.Strength, 15, nil, true, 0, false},
		{"half with damage", "happy", ability.Dexterity, 12, acid, true, 5, true},
		{"negative relocation", "unhappy (boundary)", ability.Dexterity, 12, acid, false, -1, false},
		{"relocation of one foot", "unhappy (boundary)", ability.Dexterity, 12, acid, false, 1, true},
		{"no damage at all", "unhappy (empty and minimal)", ability.Strength, 15, nil, false, 0, true},
		{"a zero damage term", "unhappy (invalid input)", ability.Strength, 15, []combat.DamageTerm{{}}, false, 0, false},
		{"three damage terms", "unhappy (boundary)", ability.Strength, 15, three, false, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, err := combat.NewEntryBySave(c.ab, c.dc, c.damage, c.half, c.relocate, true)
			if !c.ok {
				invalidTerms(t, err, c.name)
				return
			}
			if err != nil || e.Ability() != c.ab || e.DifficultyClass() != c.dc || len(e.Damage()) != len(c.damage) || e.HalfOnSuccess() != c.half ||
				e.RelocateFeet() != c.relocate || !e.TargetAnySize() {
				t.Fatalf("entry = %+v, %v", e, err)
			}
		})
	}
	own := ctDamages(ctDamage("3d6", content.DamageAcid))
	e := combat.MustEntryBySave(ability.Dexterity, 12, own, false, 0, false)
	e.Damage()[0] = combat.DamageTerm{}
	if e.Damage()[0].DamageType() != content.DamageAcid {
		t.Error("Damage must return a copy")
	}
	own[0] = combat.DamageTerm{}
	if e.Damage()[0].DamageType() != content.DamageAcid {
		t.Error("the entry must copy its input")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-006 - PeriodicDamage holds a non-empty list, a timing, and a first-tick deferral that only the end of the turn allows.
// Path: per case.
func TestPeriodicDamage(t *testing.T) {
	t.Parallel()
	start, end := combat.ContainmentTimingStartOfContainerTurn, combat.ContainmentTimingEndOfContainerTurn
	acid := ctDamages(ctDamage("2d4", content.DamageAcid))
	cases := []struct {
		name, path string
		damage     []combat.DamageTerm
		timing     combat.ContainmentTiming
		deferred   bool
		ok         bool
	}{
		{"empty list", "unhappy (empty and minimal)", nil, start, false, false},
		{"nil timing", "unhappy (invalid input)", acid, 0, false, false},
		{"timing past the last", "unhappy (invalid input)", acid, 3, false, false},
		{"deferral at the start of the turn", "unhappy (rule interaction)", acid, start, true, false},
		{"deferral at the end of the turn", "happy", acid, end, true, true},
		{"two terms", "happy", ctDamages(ctDamage("3d6", content.DamageAcid), ctDamage("3d6", content.DamageFire)), start, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := combat.NewPeriodicDamage(c.damage, c.timing, c.deferred)
			if !c.ok {
				invalidTerms(t, err, c.name)
				return
			}
			if err != nil || len(p.Damage()) != len(c.damage) || p.Timing() != c.timing || p.FirstTickNextTurn() != c.deferred {
				t.Fatalf("periodic = %+v, %v", p, err)
			}
		})
	}
	if start.String() != "start_of_container_turn" || end.String() != "end_of_container_turn" || combat.ContainmentTiming(0).String() != "unknown" {
		t.Error("timing names changed")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-007, FR-008 - Regurgitation values are 1 or more; a death rule is ReleaseOnDeath or EscapeFromCorpse with 1 or more feet.
// Path: per case.
func TestRegurgitationAndDeathRule(t *testing.T) {
	t.Parallel()
	for _, v := range [][3]int{{0, 14, 10}, {30, 0, 10}, {30, 14, 0}, {-1, 14, 10}, {30, 14, math.MinInt}} {
		_, err := combat.NewRegurgitation(v[0], v[1], v[2])
		invalidTerms(t, err, "regurgitation")
	}
	r, err := combat.NewRegurgitation(1, 1, 1)
	if err != nil || r.ThresholdDamage() != 1 || r.ConstitutionDC() != 1 || r.LandingFeet() != 1 {
		t.Errorf("minimum regurgitation = %+v, %v", r, err)
	}
	if _, err := combat.NewRegurgitation(math.MaxInt, math.MaxInt, math.MaxInt); err != nil {
		t.Errorf("maximum regurgitation: %v", err)
	}
	_, err = combat.NewEscapeFromCorpse(0)
	invalidTerms(t, err, "corpse escape of 0 feet")
	e, err := combat.NewEscapeFromCorpse(1)
	if err != nil || e.Feet() != 1 {
		t.Errorf("corpse escape of 1 foot = %+v, %v", e, err)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-009 - ExitTerms hold at least one DC and pullerDamage only with a pull DC; GrappleHoldTerms need an escape DC of 1 or more and a trimmed, non-blank part name.
// Path: per case.
func TestExitAndGrappleHoldTerms(t *testing.T) {
	t.Parallel()
	acid := ctDamages(ctDamage("3d6", content.DamageAcid))
	for name, c := range map[string]struct {
		self, pull d20.DifficultyClass
		dmg        []combat.DamageTerm
		ok         bool
	}{
		"no DC":               {0, 0, nil, false},
		"negative self":       {-1, 5, nil, false},
		"negative pull":       {5, -1, nil, false},
		"self only":           {12, 0, nil, true},
		"pull only":           {0, 12, nil, true},
		"both":                {12, 12, acid, true},
		"damage without pull": {12, 0, acid, false},
		"a zero damage term":  {12, 12, []combat.DamageTerm{{}}, false},
	} {
		e, err := combat.NewExitTerms(c.self, c.pull, c.dmg)
		if !c.ok {
			invalidTerms(t, err, name)
			continue
		}
		if err != nil || e.SelfEscapeDC() != c.self || e.PullDC() != c.pull || len(e.PullerDamage()) != len(c.dmg) {
			t.Errorf("%s: exit = %+v, %v", name, e, err)
		}
	}
	for name, c := range map[string]struct {
		dc   d20.DifficultyClass
		part string
		ok   bool
	}{"dc 0": {0, "Maw", false}, "dc 1": {1, "Maw", true}, "empty name": {14, "", false}, "blank name": {14, "  ", false},
		"leading space": {14, " Maw", false}, "trailing space": {14, "Maw ", false}} {
		g, err := combat.NewGrappleHoldTerms(c.dc, c.part)
		if !c.ok {
			invalidTerms(t, err, name)
			continue
		}
		if err != nil || g.EscapeDC() != c.dc || g.PartName() != c.part {
			t.Errorf("%s: grapple hold = %+v, %v", name, g, err)
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-010, FR-011 - ContainmentTerms accepts only Blinded and Restrained as imposed conditions and enforces the mode, grapple, regurgitation, periodic and locked-action rules.
// Path: per case.
func TestContainmentTermsRules(t *testing.T) {
	t.Parallel()
	base := func(mod func(*combat.ContainmentTermsInput)) combat.ContainmentTermsInput {
		in := combat.ContainmentTermsInput{Mode: combat.ContainmentModeSwallowed, MaxSize: content.SizeLarge, Capacity: ctCap(1, 1),
			Entry: combat.EntryAutomatic{}, Suffocation: combat.SuffocationRuleNone, DeathRule: combat.ReleaseOnDeath{}, ActionKind: combat.ContainmentActionKindAction}
		mod(&in)
		return in
	}
	periodic := ctPeriodic(combat.ContainmentTimingEndOfContainerTurn, false, ctDamage("1d4", content.DamageAcid))
	cases := []struct {
		name, path string
		mod        func(*combat.ContainmentTermsInput)
		ok         bool
	}{
		{"the minimum", "happy", func(*combat.ContainmentTermsInput) {}, true},
		{"mode 0", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Mode = 0 }, false},
		{"mode past the last", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Mode = 4 }, false},
		{"size 0", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.MaxSize = 0 }, false},
		{"zero capacity", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Capacity = combat.ContainmentCapacity{} }, false},
		{"nil entry", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Entry = nil }, false},
		{"zero entry by save", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Entry = combat.EntryBySave{} }, false},
		{"nil death rule", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.DeathRule = nil }, false},
		{"zero corpse escape", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.DeathRule = combat.EscapeFromCorpse{} }, false},
		{"suffocation 0", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Suffocation = 0 }, false},
		{"action kind 0", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.ActionKind = 0 }, false},
		{"imposed Blinded and Restrained", "happy", func(in *combat.ContainmentTermsInput) { in.Imposed = ctBR() }, true},
		{"imposed Poisoned", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) {
			in.Imposed = []content.ConditionKind{content.ConditionPoisoned}
		}, false},
		{"imposed Grappled", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) {
			in.Imposed = []content.ConditionKind{content.ConditionGrappled}
		}, false},
		{"imposed invalid kind", "unhappy (invalid input)", func(in *combat.ContainmentTermsInput) { in.Imposed = []content.ConditionKind{0} }, false},
		{"grapple with a swallow", "unhappy (rule interaction)", func(in *combat.ContainmentTermsInput) { in.Grapple = combat.MustGrappleHoldTerms(14, "Maw") }, false},
		{"engulfed-grappled without a grapple", "unhappy (rule interaction)", func(in *combat.ContainmentTermsInput) { in.Mode = combat.ContainmentModeEngulfedGrappled }, false},
		{"engulfed-grappled with a grapple", "happy", func(in *combat.ContainmentTermsInput) {
			in.Mode, in.Grapple = combat.ContainmentModeEngulfedGrappled, combat.MustGrappleHoldTerms(14, "Maw")
		}, true},
		{"grapple with an engulf", "unhappy (rule interaction)", func(in *combat.ContainmentTermsInput) {
			in.Mode, in.Grapple = combat.ContainmentModeEngulfed, combat.MustGrappleHoldTerms(14, "Maw")
		}, false},
		{"regurgitation with a swallow", "happy", func(in *combat.ContainmentTermsInput) { in.Regurgitation = combat.MustRegurgitation(30, 14, 10) }, true},
		{"regurgitation with an engulf", "unhappy (rule interaction)", func(in *combat.ContainmentTermsInput) {
			in.Mode, in.Regurgitation = combat.ContainmentModeEngulfed, combat.MustRegurgitation(30, 14, 10)
		}, false},
		{"disgorge without periodic damage", "unhappy (rule interaction)", func(in *combat.ContainmentTermsInput) { in.DisgorgeAfterFirstDamage = true }, false},
		{"disgorge with periodic damage", "happy", func(in *combat.ContainmentTermsInput) { in.DisgorgeAfterFirstDamage, in.Periodic = true, periodic }, true},
		{"locked action Bite", "happy", func(in *combat.ContainmentTermsInput) { in.LockedAction = "Bite" }, true},
		{"locked action blank", "unhappy (boundary)", func(in *combat.ContainmentTermsInput) { in.LockedAction = "  " }, false},
		{"locked action with a trailing space", "unhappy (boundary)", func(in *combat.ContainmentTermsInput) { in.LockedAction = "Bite " }, false},
		{"a zero periodic damage is absent", "unhappy (empty and minimal)", func(in *combat.ContainmentTermsInput) { in.Periodic = combat.PeriodicDamage{} }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewContainmentTerms(base(c.mod))
			if !c.ok {
				invalidTerms(t, err, c.name)
				if got != (combat.ContainmentTerms{}) {
					t.Error("an error must come with the zero terms")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewContainmentTerms: %v", err)
			}
			if _, ok := got.Periodic(); ok != (c.name == "disgorge with periodic damage") {
				t.Errorf("Periodic present = %v", ok)
			}
		})
	}
	if (combat.ContainmentTerms{}) == combat.MustContainmentTerms(base(func(*combat.ContainmentTermsInput) {})) {
		t.Error("the zero terms must differ from valid ones")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-012 - terms have value equality and are immutable; every Must form panics with the error of its New form exactly when that returns one.
// Path: unhappy (invalid input, immutability).
func TestMustFormsPanicExactlyWhenNewReturnsAnError(t *testing.T) {
	t.Parallel()
	panics := func(f func()) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err, _ = r.(error)
			}
		}()
		f()
		return nil
	}
	acid := ctDamages(ctDamage("3d6", content.DamageAcid))
	for name, c := range map[string]struct {
		bad, good func()
	}{
		"capacity": {func() { combat.MustContainmentCapacity(0, nil) }, func() { combat.MustContainmentCapacity(1, nil) }},
		"damage":   {func() { combat.MustDamageTerm(dice.Expression{}, content.DamageAcid) }, func() { ctDamage("1d4", content.DamageAcid) }},
		"entry":    {func() { combat.MustEntryBySave(ability.Strength, 0, nil, false, 0, false) }, func() { combat.MustEntryBySave(ability.Strength, 1, nil, false, 0, false) }},
		"periodic": {func() { combat.MustPeriodicDamage(nil, 1, false) }, func() { combat.MustPeriodicDamage(acid, 1, false) }},
		"regurg":   {func() { combat.MustRegurgitation(0, 1, 1) }, func() { combat.MustRegurgitation(1, 1, 1) }},
		"corpse":   {func() { combat.MustEscapeFromCorpse(0) }, func() { combat.MustEscapeFromCorpse(1) }},
		"exit":     {func() { combat.MustExitTerms(0, 0, nil) }, func() { combat.MustExitTerms(1, 0, nil) }},
		"grapple":  {func() { combat.MustGrappleHoldTerms(0, "x") }, func() { combat.MustGrappleHoldTerms(1, "x") }},
		"terms":    {func() { combat.MustContainmentTerms(combat.ContainmentTermsInput{}) }, func() { ctTerms(t, "cube") }},
	} {
		if err := panics(c.bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Must%s on bad input panicked with %v, want an error matching ErrInvalidInput", name, err)
		}
		if err := panics(c.good); err != nil {
			t.Errorf("Must%s on good input panicked with %v", name, err)
		}
	}
	in := combat.ContainmentTermsInput{Mode: combat.ContainmentModeEngulfed, MaxSize: content.SizeLarge, Capacity: ctCap(2, 1), Entry: combat.EntryAutomatic{},
		Imposed: ctBR(), Suffocation: combat.SuffocationRuleNone, DeathRule: combat.ReleaseOnDeath{}, ActionKind: combat.ContainmentActionKindAction}
	a, b := combat.MustContainmentTerms(in), combat.MustContainmentTerms(in)
	in.Imposed[0] = content.ConditionRestrained // the input slice is not kept
	if a != b || !slices.Equal(a.Imposed(), ctBR()) {
		t.Errorf("equal inputs must give equal terms and copy the imposed slice: %v", a.Imposed())
	}
	a.Imposed()[0] = content.ConditionPoisoned
	if !slices.Equal(a.Imposed(), ctBR()) {
		t.Error("Imposed must return a copy")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-013, FR-041, table 4.7.1 - the terms of the ten stat blocks construct exactly as the table states them.
// Path: per case.
func TestSrdFixturesConstructAsTable471States(t *testing.T) {
	t.Parallel()
	acid, br := content.DamageAcid, ctBR()
	cases := []struct {
		name         string
		mode         combat.ContainmentMode
		size         content.CreatureSize
		slots        int
		large        int
		imposed      []content.ConditionKind
		cover        bool
		timing       combat.ContainmentTiming
		damage       int
		regurgitate  [3]int
		corpse       int
		saves        bool
		damageType   content.DamageType
		deferredTick bool
	}{
		{"cube", combat.ContainmentModeEngulfed, content.SizeLarge, 4, 4, []content.ConditionKind{content.ConditionRestrained}, true, 1, 1, [3]int{}, 0, true, acid, false},
		{"mound", combat.ContainmentModeEngulfedGrappled, content.SizeMedium, 1, 1, br, false, 1, 1, [3]int{}, 0, true, content.DamageLightning, false},
		{"whelm", combat.ContainmentModeEngulfedGrappled, content.SizeLarge, 2, 2, []content.ConditionKind{content.ConditionRestrained}, false, 1, 1, [3]int{}, 0, true, content.DamageBludgeoning, false},
		{"behir", combat.ContainmentModeSwallowed, content.SizeLarge, 1, 1, br, true, 1, 1, [3]int{30, 14, 10}, 15, true, acid, false},
		{"worm", combat.ContainmentModeSwallowed, content.SizeLarge, 3, 1, br, true, 1, 1, [3]int{30, 21, 5}, 20, true, acid, false},
		{"remorhaz", combat.ContainmentModeSwallowed, content.SizeLarge, 2, 1, br, true, 1, 2, [3]int{30, 15, 5}, 15, true, acid, false},
		{"tarrasque", combat.ContainmentModeSwallowed, content.SizeLarge, 6, 1, br, true, 1, 1, [3]int{60, 20, 10}, 20, true, acid, false},
		{"kraken", combat.ContainmentModeSwallowed, content.SizeLarge, 4, 1, []content.ConditionKind{content.ConditionRestrained}, true, 1, 1, [3]int{50, 25, 10}, 15, true, acid, false},
		{"frog", combat.ContainmentModeSwallowed, content.SizeSmall, 1, 1, br, true, 2, 1, [3]int{}, 5, false, acid, true},
		{"toad", combat.ContainmentModeSwallowed, content.SizeMedium, 1, 1, br, true, 2, 1, [3]int{}, 5, false, acid, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := ctTerms(t, c.name)
			p, _ := got.Periodic()
			reg, hasReg := got.Regurgitation()
			corpse, isCorpse := got.DeathRule().(combat.EscapeFromCorpse)
			_, saves := got.Entry().(combat.EntryBySave)
			if got.Mode() != c.mode || got.MaxSize() != c.size || got.Capacity().Slots() != c.slots || got.Capacity().Cost(content.SizeLarge) != c.large ||
				!slices.Equal(got.Imposed(), c.imposed) || got.TotalCover() != c.cover || p.Timing() != c.timing || len(p.Damage()) != c.damage ||
				p.Damage()[0].DamageType() != c.damageType || p.FirstTickNextTurn() != c.deferredTick || saves != c.saves ||
				hasReg != (c.regurgitate[0] > 0) || (hasReg && [3]int{reg.ThresholdDamage(), int(reg.ConstitutionDC()), reg.LandingFeet()} != c.regurgitate) ||
				isCorpse != (c.corpse > 0) || (isCorpse && corpse.Feet() != c.corpse) {
				t.Errorf("%s terms do not match table 4.7.1", c.name)
			}
		})
	}
	cube, mound := ctTerms(t, "cube"), ctTerms(t, "mound")
	exit, _ := cube.Exit()
	if !cube.NoVerbalSpells() || cube.Suffocation() != combat.SuffocationRuleAlways || exit.SelfEscapeDC() != 12 || exit.PullDC() != 12 || len(exit.PullerDamage()) != 1 ||
		mound.NoVerbalSpells() || ctTerms(t, "whelm").Suffocation() != combat.SuffocationRuleUnlessBreathesWater || !ctTerms(t, "tarrasque").NoTeleport() ||
		ctTerms(t, "frog").LockedAction() != "Bite" || !ctTerms(t, "frog").DisgorgeAfterFirstDamage() || ctTerms(t, "behir").ActionKind() != combat.ContainmentActionKindBonusAction ||
		ctTerms(t, "kraken").ActionKind() != combat.ContainmentActionKindAction {
		t.Error("the side facts of table 4.7.1 do not hold")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-010, FR-013 - the optional members report whether they are present (the periodic damage of every block, the regurgitation of the six swallowers that print one, the ways out of the cube and the Whelm, the grapple of the two Engulf-grappled blocks); an out-of-range imposed kind is refused, a repeated one counts once, and a corpse escape of one foot is valid.
// Path: per case.
func TestContainmentTermsOptionalMembersAndImposedKinds(t *testing.T) {
	t.Parallel()
	for name, want := range map[string][4]bool{ // periodic, regurgitation, exit, grapple
		"cube": {true, false, true, false}, "mound": {true, false, false, true}, "whelm": {true, false, true, true}, "behir": {true, true, false, false},
		"worm": {true, true, false, false}, "remorhaz": {true, true, false, false}, "tarrasque": {true, true, false, false}, "kraken": {true, true, false, false},
		"frog": {true, false, false, false}, "toad": {true, false, false, false},
	} {
		terms := ctTerms(t, name)
		_, periodic := terms.Periodic()
		_, regurgitation := terms.Regurgitation()
		_, exit := terms.Exit()
		_, grapple := terms.Grapple()
		if got := [4]bool{periodic, regurgitation, exit, grapple}; got != want {
			t.Errorf("%s presence (periodic, regurgitation, exit, grapple) = %v, want %v", name, got, want)
		}
	}
	base := combat.ContainmentTermsInput{Mode: combat.ContainmentModeSwallowed, MaxSize: content.SizeLarge, Capacity: ctCap(1, 1), Entry: combat.EntryAutomatic{},
		Suffocation: combat.SuffocationRuleNone, DeathRule: combat.MustEscapeFromCorpse(1), ActionKind: combat.ContainmentActionKindAction}
	for _, kind := range []content.ConditionKind{99, 17, 16, -1} {
		in := base
		in.Imposed = []content.ConditionKind{kind}
		_, err := combat.NewContainmentTerms(in)
		invalidTerms(t, err, "imposed kind out of range")
	}
	in := base
	in.Imposed = []content.ConditionKind{content.ConditionBlinded, content.ConditionBlinded}
	got, err := combat.NewContainmentTerms(in)
	if err != nil || !slices.Equal(got.Imposed(), []content.ConditionKind{content.ConditionBlinded}) {
		t.Errorf("a repeated kind = %v, %v, want it counted once", got.Imposed(), err)
	}
	if corpse := got.DeathRule().(combat.EscapeFromCorpse); corpse.Feet() != 1 {
		t.Errorf("corpse escape = %d feet", corpse.Feet())
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-009 - terms whose only way out is the held creature's own escape, or only a pull, report an exit.
// Path: unhappy (boundary).
func TestContainmentTermsExitWithOneDC(t *testing.T) {
	t.Parallel()
	for name, exit := range map[string]combat.ExitTerms{"self only": combat.MustExitTerms(12, 0, nil), "pull only": combat.MustExitTerms(0, 12, nil)} {
		in := combat.ContainmentTermsInput{Mode: combat.ContainmentModeEngulfed, MaxSize: content.SizeLarge, Capacity: ctCap(1, 1), Entry: combat.EntryAutomatic{},
			Suffocation: combat.SuffocationRuleNone, DeathRule: combat.ReleaseOnDeath{}, ActionKind: combat.ContainmentActionKindAction, Exit: exit}
		got := combat.MustContainmentTerms(in)
		if e, ok := got.Exit(); !ok || e != exit {
			t.Errorf("%s: Exit = %+v/%v", name, e, ok)
		}
	}
}
