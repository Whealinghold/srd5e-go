package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

func internalTerms(t *testing.T, mod func(*ContainmentTermsInput)) ContainmentTerms {
	t.Helper()
	in := ContainmentTermsInput{Mode: ContainmentModeEngulfed, MaxSize: content.SizeLarge,
		Capacity:    MustContainmentCapacity(4, map[content.CreatureSize]int{content.SizeLarge: 4}),
		Entry:       MustEntryBySave(ability.Dexterity, 12, []DamageTerm{MustDamageTerm(dice.MustParseExpression("3d6"), content.DamageAcid)}, true, 5, true),
		Suffocation: SuffocationRuleNone, DeathRule: ReleaseOnDeath{}, ActionKind: ContainmentActionKindAction}
	mod(&in)
	got, err := NewContainmentTerms(in)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-018, FR-019 - enter takes a creature in with one OccupantContained in phase inside, and rejects in this order: container is contained, occupant is a container, occupant already contained, too large (equal is allowed), no room; a rejection changes nothing.
// Path: per case.
func TestEnterRejectionOrder(t *testing.T) {
	t.Parallel()
	terms := internalTerms(t, func(*ContainmentTermsInput) {})
	enter := func(s Containments, container, occupant string, size content.CreatureSize) ContainmentTransition {
		return s.enter(CombatantID(container), CombatantID(occupant), size, terms, 0, s.nextPartOrdinal(CombatantID(container)), false)
	}
	first := enter(Containments{}, "cube", "a", content.SizeMedium).(ContainmentTransitionApplied)
	rec, ok := first.containments.ContainerOf("a").(InContainer)
	if !ok || rec.record.phase.IsTrapped() || rec.record.partOrdinal != 1 || len(first.events) != 1 || first.events[0].Kind() != "occupant_contained" ||
		first.events[0].(OccupantContained).mode != ContainmentModeEngulfed {
		t.Fatalf("enter = %+v, want a record inside, ordinal 1 and one OccupantContained", first)
	}
	// A state that breaks every rule at once, then each rule removed in turn.
	held := first.containments
	cases := []struct {
		name                string
		set                 Containments
		container, occupant string
		size                content.CreatureSize
		want                ContainmentRejection
	}{
		{"container is held", held, "a", "x", content.SizeMedium, ContainmentRejectionContainerIsContained},
		{"occupant holds creatures", held, "z", "cube", content.SizeHuge, ContainmentRejectionOccupantIsContainer},
		{"occupant already held, too large", held, "other", "a", content.SizeHuge, ContainmentRejectionOccupantAlreadyContained},
		{"too large before no room", held, "cube", "x", content.SizeHuge, ContainmentRejectionTooLarge},
		{"Large equals the maximum but takes four slots", held, "cube", "x", content.SizeLarge, ContainmentRejectionNoRoom},
		{"Large fits an empty cube", Containments{}, "cube", "x", content.SizeLarge, 0},
		{"a fourth Medium fits", held, "cube", "x", content.SizeMedium, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.set.All()
			got := enter(c.set, c.container, c.occupant, c.size)
			if c.want == 0 {
				if _, ok := got.(ContainmentTransitionApplied); !ok {
					t.Fatalf("enter = %#v, want applied", got)
				}
			} else if rej, ok := got.(ContainmentTransitionRejected); !ok || rej.Rejection != c.want {
				t.Fatalf("enter = %#v, want %v", got, c.want)
			}
			if !c.set.Equal(Containments{before}) {
				t.Error("enter must not change its receiver")
			}
		})
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-020 - partOrdinal is the lowest positive integer no other record of the container uses, 1 for an empty container.
// Path: unhappy (boundary).
func TestNextPartOrdinalIsTheLowestFree(t *testing.T) {
	t.Parallel()
	terms := internalTerms(t, func(in *ContainmentTermsInput) { in.Capacity = MustContainmentCapacity(8, nil) })
	rec := func(container, occupant string, n int) Containment {
		return Containment{CombatantID(container), CombatantID(occupant), content.SizeMedium, terms, ContainmentInside(), 0, n, 0, 0, false}
	}
	s := Containments{[]Containment{rec("c", "a", 1), rec("c", "b", 3), rec("d", "e", 2)}}
	for container, want := range map[string]int{"c": 2, "d": 1, "empty": 1} {
		if got := s.nextPartOrdinal(CombatantID(container)); got != want {
			t.Errorf("nextPartOrdinal(%s) = %d, want %d", container, got, want)
		}
	}
	s = Containments{[]Containment{rec("c", "a", 1), rec("c", "b", 2), rec("c", "e", 3)}}
	if got := s.nextPartOrdinal("c"); got != 4 {
		t.Errorf("nextPartOrdinal with 1, 2, 3 used = %d, want 4", got)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-043, FR-044, FR-078 - advance returns the due damage of a record inside at the matching boundary, increments its ticks, skips a deferred record at the end boundary and clears the flag, and gives nothing for a trapped record or another container.
// Path: per case.
func TestAdvanceDueAtEachBoundary(t *testing.T) {
	t.Parallel()
	start := MustPeriodicDamage([]DamageTerm{MustDamageTerm(dice.MustParseExpression("3d6"), content.DamageAcid)}, ContainmentTimingStartOfContainerTurn, false)
	end := MustPeriodicDamage([]DamageTerm{MustDamageTerm(dice.MustParseExpression("2d4"), content.DamageAcid)}, ContainmentTimingEndOfContainerTurn, true)
	startTerms := internalTerms(t, func(in *ContainmentTermsInput) { in.Periodic = start })
	endTerms := internalTerms(t, func(in *ContainmentTermsInput) { in.Periodic = end })
	noPeriodic := internalTerms(t, func(*ContainmentTermsInput) {})
	rec := func(container, occupant string, terms ContainmentTerms, deferred bool, feet int) Containment {
		return Containment{CombatantID(container), CombatantID(occupant), content.SizeSmall, terms, ContainmentPhase{feet}, 0, 1, 0, 0, deferred}
	}
	s := Containments{[]Containment{rec("c", "a", startTerms, false, 0), rec("c", "b", startTerms, false, 20), rec("d", "x", startTerms, false, 0)}}
	next, dues := s.advance("c", ContainerTurnBoundaryStart)
	if len(dues) != 1 || dues[0].Target != "a" || dues[0].Container != "c" || dues[0].cause != ContainmentDamageCausePeriodic || dues[0].halved {
		t.Fatalf("start dues = %+v, want one for a (b is trapped, x has another container)", dues)
	}
	if next.list[0].damageTicks != 1 || next.list[1].damageTicks != 0 || next.list[2].damageTicks != 0 || s.list[0].damageTicks != 0 {
		t.Error("only a due record's ticks go up, and the receiver does not change")
	}
	if _, dues := s.advance("c", ContainerTurnBoundaryEnd); len(dues) != 0 {
		t.Errorf("end dues for start-timed records = %+v, want none", dues)
	}
	if _, dues := (Containments{[]Containment{rec("c", "a", noPeriodic, false, 0)}}).advance("c", ContainerTurnBoundaryStart); len(dues) != 0 {
		t.Error("terms with no periodic damage give none")
	}
	deferred := Containments{[]Containment{rec("c", "a", endTerms, true, 0), rec("c", "b", endTerms, false, 0)}}
	next, dues = deferred.advance("c", ContainerTurnBoundaryEnd)
	if len(dues) != 1 || dues[0].Target != "b" || next.list[0].tickDeferred || next.list[0].damageTicks != 0 || next.list[1].damageTicks != 1 {
		t.Errorf("end boundary with a deferred record: dues %+v, record %+v", dues, next.list[0])
	}
	if _, dues = next.advance("c", ContainerTurnBoundaryEnd); len(dues) != 2 {
		t.Errorf("the next end boundary = %d dues, want both", len(dues))
	}
	if _, dues := deferred.advance("c", ContainerTurnBoundaryStart); len(dues) != 0 || !deferred.list[0].tickDeferred {
		t.Error("the start boundary leaves an end-timed deferred record alone")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-049 - withoutTallies resets the damage from inside of the container's records and no other.
// Path: unhappy (rule interaction).
func TestWithoutTalliesResetsOneContainer(t *testing.T) {
	t.Parallel()
	terms := internalTerms(t, func(in *ContainmentTermsInput) { in.Capacity = MustContainmentCapacity(8, nil) })
	s := Containments{[]Containment{{"c", "a", content.SizeSmall, terms, ContainmentInside(), 0, 1, 9, 0, false}, {"d", "b", content.SizeSmall, terms, ContainmentInside(), 0, 1, 9, 0, false}}}
	got := s.withoutTallies("c")
	if got.list[0].damageFromInside != 0 || got.list[1].damageFromInside != 9 || s.list[0].damageFromInside != 9 {
		t.Errorf("tallies after the reset = %d, %d", got.list[0].damageFromInside, got.list[1].damageFromInside)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-015, FR-016 - NewContainments rejects a record that fails its own validation even when it would fit, and without and withoutTallies touch only the occupant or container they name.
// Path: unhappy (invalid input).
func TestContainmentsHelpersTouchOnlyWhatTheyName(t *testing.T) {
	t.Parallel()
	terms := internalTerms(t, func(in *ContainmentTermsInput) { in.Capacity = MustContainmentCapacity(8, nil) })
	rec := func(container, occupant string, ordinal, tally int) Containment {
		return Containment{CombatantID(container), CombatantID(occupant), content.SizeSmall, terms, ContainmentInside(), 0, ordinal, tally, 0, false}
	}
	if _, err := NewContainments([]Containment{rec("c", "a", 0, 0)}); err == nil {
		t.Error("a record with part ordinal 0 fits the capacity but is invalid")
	}
	s := Containments{[]Containment{rec("c", "a", 1, 5), rec("c", "b", 2, 5), rec("d", "e", 1, 5)}}
	if got := s.without("b").list; len(got) != 2 || got[0].Occupant != "a" || got[1].Occupant != "e" {
		t.Errorf("without(b) = %v, want a and e", got)
	}
	if got := s.without("a").list; len(got) != 2 || got[0].Occupant != "b" || got[1].Occupant != "e" {
		t.Errorf("without(a) = %v, want b and e", got)
	}
	if got := s.without("zz").list; len(got) != 3 {
		t.Errorf("without(zz) = %v, want all three", got)
	}
	got := s.withoutTallies("d").list
	if got[0].damageFromInside != 5 || got[1].damageFromInside != 5 || got[2].damageFromInside != 0 {
		t.Errorf("withoutTallies(d) tallies = %d %d %d, want 5 5 0", got[0].damageFromInside, got[1].damageFromInside, got[2].damageFromInside)
	}
	if replaced := s.replacing(rec("c", "b", 2, 9)).list; replaced[0].damageFromInside != 5 || replaced[1].damageFromInside != 9 || replaced[2].damageFromInside != 5 {
		t.Errorf("replacing(b) = %v", replaced)
	}
}
