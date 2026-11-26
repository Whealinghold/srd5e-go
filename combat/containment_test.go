package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-014 - NewContainment returns an ErrInvalidInput error for equal or blank ids, a negative counter, added conditions outside the imposed ones, and the other unusable values.
// Path: per case.
func TestNewContainmentRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	worm, frog, kraken := ctTerms(t, "worm"), ctTerms(t, "frog"), ctTerms(t, "kraken")
	base := combat.ContainmentInput{Container: gid("c"), Occupant: gid("o"), OccupantSize: content.SizeLarge, Terms: worm, Phase: combat.ContainmentInside(), PartOrdinal: 1}
	cases := []struct {
		name string
		mod  func(*combat.ContainmentInput)
		ok   bool
	}{
		{"valid", func(*combat.ContainmentInput) {}, true},
		{"same creature", func(in *combat.ContainmentInput) { in.Occupant = in.Container }, false},
		{"blank container", func(in *combat.ContainmentInput) { in.Container = "" }, false},
		{"blank occupant", func(in *combat.ContainmentInput) { in.Occupant = "" }, false},
		{"size 0", func(in *combat.ContainmentInput) { in.OccupantSize = 0 }, false},
		{"zero terms", func(in *combat.ContainmentInput) { in.Terms = combat.ContainmentTerms{} }, false},
		{"part ordinal 0", func(in *combat.ContainmentInput) { in.PartOrdinal = 0 }, false},
		{"negative damage from inside", func(in *combat.ContainmentInput) { in.DamageFromInside = -1 }, false},
		{"zero damage from inside", func(in *combat.ContainmentInput) { in.DamageFromInside = 0 }, true},
		{"negative ticks", func(in *combat.ContainmentInput) { in.DamageTicks = -1 }, false},
		{"added condition imposed", func(in *combat.ContainmentInput) {
			in.AddedConditions = []content.ConditionKind{content.ConditionBlinded}
		}, true},
		{"added condition not imposed", func(in *combat.ContainmentInput) {
			in.Terms, in.AddedConditions = kraken, []content.ConditionKind{content.ConditionBlinded}
		}, false},
		{"added invalid kind", func(in *combat.ContainmentInput) { in.AddedConditions = []content.ConditionKind{0} }, false},
		{"deferred tick the terms do not defer", func(in *combat.ContainmentInput) { in.TickDeferred = true }, false},
		{"deferred tick the terms defer", func(in *combat.ContainmentInput) { in.Terms, in.TickDeferred = frog, true }, true},
		{"trapped", func(in *combat.ContainmentInput) { in.Phase = mustTrapped(t, 20) }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := base
			c.mod(&in)
			got, err := combat.NewContainment(in)
			if c.ok {
				if err != nil || got.Container != in.Container {
					t.Fatalf("NewContainment = %+v, %v", got, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || got != (combat.Containment{}) {
				t.Errorf("NewContainment = %+v, %v, want the zero record and ErrInvalidInput", got, err)
			}
		})
	}
	if _, err := combat.NewContainmentTrapped(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a trapped phase of 0 feet: %v", err)
	}
	r := ctRecord(t, "c", "o", content.SizeLarge, worm, 2, func(in *combat.ContainmentInput) {
		in.AddedConditions, in.DamageFromInside, in.DamageTicks = ctBR(), 7, 3
	})
	if r.Size() != content.SizeLarge || r.Terms() != worm || r.Phase().IsTrapped() || r.Phase().TrappedFeet() != 0 || r.PartOrdinal() != 2 || r.DamageFromInside() != 7 ||
		r.DamageTicks() != 3 || r.TickDeferred() || len(r.AddedConditions()) != 2 {
		t.Errorf("record accessors = %+v", r)
	}
}

func mustTrapped(t *testing.T, feet int) combat.ContainmentPhase {
	t.Helper()
	p, err := combat.NewContainmentTrapped(feet)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-015, Q-12, plan-phase-4.md P10 - the Containments constructor rejects an occupant twice, unequal terms in one container, a capacity overrun and nested containment in either direction, which also rejects every link cycle, found with sets and no recursion.
// Path: per case.
func TestNewContainmentsInvariantsAndCycles(t *testing.T) {
	t.Parallel()
	worm, cube, toad := ctTerms(t, "worm"), ctTerms(t, "cube"), ctTerms(t, "toad")
	rec := func(container, occupant string, size content.CreatureSize, terms combat.ContainmentTerms, ordinal int) combat.Containment {
		return ctRecord(t, container, occupant, size, terms, ordinal)
	}
	cases := []struct {
		name, path string
		records    []combat.Containment
		ok         bool
	}{
		{"nothing", "unhappy (empty and minimal)", nil, true},
		{"three Medium in a worm", "unhappy (boundary)", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("w", "b", content.SizeMedium, worm, 2), rec("w", "c", content.SizeMedium, worm, 3)}, true},
		{"a fourth in a worm", "unhappy (boundary)", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("w", "b", content.SizeMedium, worm, 2),
			rec("w", "c", content.SizeMedium, worm, 3), rec("w", "d", content.SizeMedium, worm, 4)}, false},
		{"a Large and a Medium in a cube", "unhappy (boundary)", []combat.Containment{rec("c", "a", content.SizeLarge, cube, 1), rec("c", "b", content.SizeMedium, cube, 2)}, false},
		{"four Medium in a cube", "unhappy (boundary)", []combat.Containment{rec("c", "a", content.SizeMedium, cube, 1), rec("c", "b", content.SizeMedium, cube, 2),
			rec("c", "c2", content.SizeMedium, cube, 3), rec("c", "d", content.SizeMedium, cube, 4)}, true},
		{"an occupant twice in one container", "unhappy (invalid input)", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("w", "a", content.SizeMedium, worm, 2)}, false},
		{"an occupant in two containers", "unhappy (invalid input)", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("t", "a", content.SizeMedium, toad, 1)}, false},
		{"unequal terms in one container", "unhappy (rule interaction)", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("w", "b", content.SizeMedium, toad, 2)}, false},
		{"two containers", "happy", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("t", "b", content.SizeMedium, toad, 1)}, true},
		{"a container that is held", "unhappy (rule interaction)", []combat.Containment{rec("w", "a", content.SizeMedium, worm, 1), rec("a", "b", content.SizeSmall, toad, 1)}, false},
		{"a held creature holding, the other way round", "unhappy (rule interaction)", []combat.Containment{rec("a", "b", content.SizeSmall, toad, 1), rec("w", "a", content.SizeMedium, worm, 1)}, false},
		{"a cycle of two", "unhappy (rule interaction)", []combat.Containment{rec("a", "b", content.SizeSmall, toad, 1), rec("b", "a", content.SizeSmall, toad, 1)}, false},
		{"a cycle of three", "unhappy (rule interaction)", []combat.Containment{rec("a", "b", content.SizeSmall, toad, 1), rec("b", "c", content.SizeSmall, toad, 1), rec("c", "a", content.SizeSmall, toad, 1)}, false},
		{"an invalid record", "unhappy (invalid input)", []combat.Containment{{Container: gid("a"), Occupant: gid("b")}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewContainments(c.records)
			if c.ok {
				if err != nil || got.Len() != len(c.records) {
					t.Fatalf("NewContainments = %v, %v", got, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || got.Len() != 0 {
				t.Errorf("NewContainments = %v, %v, want an empty set and ErrInvalidInput", got, err)
			}
		})
	}
	// A long chain of links is a cycle-shaped input: it must be rejected without recursing.
	var chain []combat.Containment
	for i := 0; i < 2000; i++ {
		chain = append(chain, rec(id(i), id(i+1), content.SizeSmall, toad, 1))
	}
	chain = append(chain, rec(id(2000), id(0), content.SizeSmall, toad, 1))
	if _, err := combat.NewContainments(chain); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a cycle of 2001 links: %v", err)
	}
}

func id(i int) string {
	return "n" + string(rune('A'+i%26)) + string(rune('a'+i/26%26)) + string(rune('0'+i/676))
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-016, FR-017 - Containments copies its input and every view, compares equal whatever order the records were taken in, iterates in entry order, and ContainerOf, OccupantsOf and SlotsUsed answer from the records.
// Path: unhappy (immutability).
func TestContainmentsCopiesComparesAndLooksUp(t *testing.T) {
	t.Parallel()
	worm := ctTerms(t, "worm")
	a, b := ctRecord(t, "w", "a", content.SizeLarge, worm, 1), ctRecord(t, "w", "b", content.SizeMedium, worm, 2)
	input := []combat.Containment{a, b}
	s := ctSet(t, input...)
	input[0] = b
	if s.All()[0] != a {
		t.Error("NewContainments must copy its input")
	}
	s.All()[0] = b
	if s.All()[0] != a || s.All()[1] != b {
		t.Error("All must return a copy in entry order")
	}
	if !s.Equal(ctSet(t, b, a)) || s.Equal(ctSet(t, a)) || s.Equal(ctSet(t, a, ctRecord(t, "w", "b", content.SizeMedium, worm, 3))) {
		t.Error("Equal must ignore order and notice any difference")
	}
	var zero combat.Containments
	if zero.Len() != 0 || len(zero.All()) != 0 || !zero.Equal(combat.Containments{}) || zero.SlotsUsed(gid("w")) != 0 || zero.Equal(s) {
		t.Error("the zero Containments is empty and valid")
	}
	if in, ok := s.ContainerOf(gid("b")).(combat.InContainer); !ok || in.Record() != b {
		t.Errorf("ContainerOf(b) = %#v, want InContainer(b)", s.ContainerOf(gid("b")))
	}
	if _, ok := s.ContainerOf(gid("w")).(combat.NotContained); !ok {
		t.Error("a container is not held")
	}
	if got := s.OccupantsOf(gid("w")); len(got) != 2 || got[0] != a || got[1] != b {
		t.Errorf("OccupantsOf(w) = %v, want [a b] in entry order", got)
	}
	if got := s.OccupantsOf(gid("nobody")); got == nil || len(got) != 0 {
		t.Errorf("OccupantsOf(nobody) = %#v, want an empty non-nil slice", got)
	}
	if s.SlotsUsed(gid("w")) != 2 || s.SlotsUsed(gid("x")) != 0 {
		t.Errorf("SlotsUsed = %d, want 2 (Large and Medium cost 1 each in a worm)", s.SlotsUsed(gid("w")))
	}
	if ctSet(t, ctRecord(t, "c", "o", content.SizeLarge, ctTerms(t, "cube"), 1)).SlotsUsed(gid("c")) != 4 {
		t.Error("a Large creature takes the cube's four slots")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-046, Q-1, table 4.7.3 - RecordDamageFromInside adds per occupant (zero allowed, saturating), refuses a creature that is not held or is trapped, and returns an ErrInvalidInput error for a negative amount.
// Path: per case.
func TestRecordDamageFromInside(t *testing.T) {
	t.Parallel()
	worm := ctTerms(t, "worm")
	a, b := ctRecord(t, "w", "a", content.SizeMedium, worm, 1), ctRecord(t, "w", "b", content.SizeMedium, worm, 2)
	trapped := ctRecord(t, "w", "t", content.SizeMedium, worm, 3, func(in *combat.ContainmentInput) { in.Phase = mustTrapped(t, 20) })
	s := ctSet(t, a, b, trapped)
	apply := func(s combat.Containments, who string, n int) combat.Containments {
		t.Helper()
		tr, err := s.RecordDamageFromInside(gid(who), n)
		if err != nil {
			t.Fatalf("RecordDamageFromInside(%s, %d): %v", who, n, err)
		}
		ap, ok := tr.(combat.ContainmentTransitionApplied)
		if !ok || len(ap.Events()) != 0 {
			t.Fatalf("transition = %#v, want applied with no event", tr)
		}
		return ap.Containments()
	}
	got := apply(apply(s, "a", 20), "b", 20)
	tally := func(s combat.Containments, who string) int {
		return s.ContainerOf(gid(who)).(combat.InContainer).Record().DamageFromInside()
	}
	if tally(got, "a") != 20 || tally(got, "b") != 20 {
		t.Errorf("tallies = %d, %d, want 20 each (per occupant, never summed)", tally(got, "a"), tally(got, "b"))
	}
	if tally(apply(apply(s, "a", 20), "a", 10), "a") != 30 || tally(apply(s, "a", 0), "a") != 0 {
		t.Error("damage from one occupant adds up and zero is allowed")
	}
	if tally(s, "a") != 0 {
		t.Error("the input set must not change")
	}
	if tally(apply(apply(s, "a", math.MaxInt), "a", 5), "a") != math.MaxInt {
		t.Error("the tally must saturate at the limits of int")
	}
	for _, who := range []string{"unknown", "w", "t"} {
		tr, err := s.RecordDamageFromInside(gid(who), 40)
		if rej, ok := tr.(combat.ContainmentTransitionRejected); err != nil || !ok || rej.Rejection != combat.ContainmentRejectionNotContained {
			t.Errorf("RecordDamageFromInside(%s) = %#v, %v, want not_contained (a trapped record never counts)", who, tr, err)
		}
	}
	if tr, err := s.RecordDamageFromInside(gid("a"), -1); !errors.Is(err, srderr.ErrInvalidInput) || tr != nil {
		t.Errorf("a negative amount = %v, %v, want an ErrInvalidInput error", tr, err)
	}
	if _, err := s.RecordDamageFromInside(gid("unknown"), -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the negative amount is checked before the occupant: %v", err)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-014, FR-017 - a trapped phase of one foot is valid, an added condition kind out of range is refused and a repeated one counts once, and SlotsUsed and OccupantsOf answer per container.
// Path: unhappy (boundary).
func TestContainmentRecordKindsAndPerContainerLookups(t *testing.T) {
	t.Parallel()
	if p, err := combat.NewContainmentTrapped(1); err != nil || !p.IsTrapped() || p.TrappedFeet() != 1 {
		t.Errorf("a trapped phase of 1 foot = %+v, %v", p, err)
	}
	in := combat.ContainmentInput{Container: gid("c"), Occupant: gid("o"), OccupantSize: content.SizeMedium, Terms: ctTerms(t, "worm"), Phase: combat.ContainmentInside(), PartOrdinal: 1}
	for _, kind := range []content.ConditionKind{99, 16, -1} {
		bad := in
		bad.AddedConditions = []content.ConditionKind{kind}
		if _, err := combat.NewContainment(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("added kind %d: err = %v, want ErrInvalidInput", int(kind), err)
		}
	}
	twice := in
	twice.AddedConditions = []content.ConditionKind{content.ConditionBlinded, content.ConditionBlinded}
	if r, err := combat.NewContainment(twice); err != nil || len(r.AddedConditions()) != 1 {
		t.Errorf("a repeated added kind = %v, %v, want it counted once", r.AddedConditions(), err)
	}
	two := ctSet(t, ctRecord(t, "a", "x", content.SizeMedium, ctTerms(t, "worm"), 1), ctRecord(t, "b", "y", content.SizeMedium, ctTerms(t, "worm"), 1),
		ctRecord(t, "b", "z", content.SizeMedium, ctTerms(t, "worm"), 2))
	for container, want := range map[string]int{"a": 1, "b": 2, "c": 0} {
		if two.SlotsUsed(gid(container)) != want || len(two.OccupantsOf(gid(container))) != want {
			t.Errorf("container %s: slots %d occupants %d, want %d", container, two.SlotsUsed(gid(container)), len(two.OccupantsOf(gid(container))), want)
		}
	}
}
