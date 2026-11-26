package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func meOther(c combat.ShapeContainment, lines ...combat.Cover) combat.AreaLocationFacts {
	return meMust(combat.NewOtherLocationFacts(c, lines))
}

func meOrigin(shapeName string, creator bool) combat.AreaLocationFacts {
	shape := meShapes()[shapeName].shape
	d := meMust(combat.ShapeDefaultOrigin(shape))
	if creator {
		d = meMust(combat.CreatorIncludesOrigin(combat.OverridableCone{Area: meCone(60)}))
	}
	return meMust(combat.NewOriginLocationFacts(d))
}

// Requirement: per case
// Path: per case.
func TestAreaMembershipAssess(t *testing.T) {
	t.Parallel()
	const none = combat.Cover(0)
	cases := []struct {
		name, requirement, path string
		facts                   combat.AreaLocationFacts
		want                    combat.AreaMembershipResult
	}{
		{"inside, open line", meSpec + `table 4.4a; FR-020, FR-022; SRD 5.2.1, "Area of Effect", p. 177`, "happy", meOther(combat.InsideShape, combat.HalfCover), combat.AreaMembershipIncluded{}},
		{"inside, no cover on the line", meSpec + "table 4.4a; FR-020", "happy", meOther(combat.InsideShape, none), combat.AreaMembershipIncluded{}},
		{"inside, three-quarters", meSpec + `table 4.4a; FR-022; SRD 5.2.1, "Cover", p. 15`, "unhappy (boundary)", meOther(combat.InsideShape, combat.ThreeQuartersCover), combat.AreaMembershipIncluded{}},
		{"inside, one open of many", meSpec + "table 4.4a; FR-020", "unhappy (rule interaction)", meOther(combat.InsideShape, combat.TotalCover, combat.TotalCover, combat.HalfCover), combat.AreaMembershipIncluded{}},
		{"inside, all Total Cover", meSpec + "table 4.4a; FR-021", "unhappy (rule interaction)", meOther(combat.InsideShape, combat.TotalCover, combat.TotalCover), combat.AreaMembershipExcluded{Reason: combat.ExcludedAllLinesBlocked}},
		{"inside, a single Total Cover line", meSpec + "table 4.4a; FR-021", "unhappy (boundary)", meOther(combat.InsideShape, combat.TotalCover), combat.AreaMembershipExcluded{Reason: combat.ExcludedAllLinesBlocked}},
		{"outside, open", meSpec + "table 4.4a; FR-019", "happy", meOther(combat.OutsideShape, combat.HalfCover), combat.AreaMembershipExcluded{Reason: combat.ExcludedOutsideShape}},
		{"outside and blocked", meSpec + "table 4.4a; FR-025", "unhappy (rule interaction)", meOther(combat.OutsideShape, combat.TotalCover), combat.AreaMembershipExcluded{Reason: combat.ExcludedOutsideShape}},
		{"inside, no lines", meSpec + "table 4.4a; FR-023", "unhappy (empty and minimal)", meOther(combat.InsideShape), combat.AreaMembershipNoLinesConsidered{}},
		{"outside, no lines", meSpec + "table 4.4a; FR-019, FR-023", "unhappy (rule interaction)", meOther(combat.OutsideShape), combat.AreaMembershipExcluded{Reason: combat.ExcludedOutsideShape}},
		{"origin of a Sphere", meSpec + `table 4.4a; FR-024; SRD 5.2.1, "Sphere", p. 188`, "happy", meOrigin("sphere", false), combat.AreaMembershipIncluded{}},
		{"origin of a Cone", meSpec + `table 4.4a; FR-024; SRD 5.2.1, "Cone", p. 179`, "unhappy (rule interaction)", meOrigin("cone", false), combat.AreaMembershipExcluded{Reason: combat.ExcludedOrigin}},
		{"origin of a Cone its creator includes", meSpec + "FR-024, FR-016", "unhappy (rule interaction)", meOrigin("cone", true), combat.AreaMembershipIncluded{}},
		{"nil facts", meSpec + "FR-144: facts the engine cannot use", "unhappy (invalid input)", nil, combat.AreaMembershipNoLinesConsidered{}},
		{"the zero other-location facts", meSpec + "FR-144", "unhappy (invalid input)", combat.OtherLocationFacts{}, combat.AreaMembershipNoLinesConsidered{}},
		{"the zero origin facts", meSpec + "FR-144", "unhappy (invalid input)", combat.OriginLocationFacts{}, combat.AreaMembershipNoLinesConsidered{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (combat.AreaMembership{}).Assess(c.facts); got != c.want {
				t.Errorf("Assess = %#v, want %#v [%s; %s]", got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-026 - the facts keep their own copy of the lines (a caller's later change does not alter them), the returned view is a copy, and equal inputs give an equal result.
// Path: unhappy (immutability).
func TestOtherLocationFactsHoldTheirOwnCopyOfTheLines(t *testing.T) {
	t.Parallel()
	lines := []combat.Cover{combat.TotalCover, combat.TotalCover}
	f := meMust(combat.NewOtherLocationFacts(combat.InsideShape, lines))
	lines[0] = combat.HalfCover
	view := f.Lines()
	view[1] = combat.HalfCover
	if got := f.Lines(); got[0] != combat.TotalCover || got[1] != combat.TotalCover || f.Containment() != combat.InsideShape {
		t.Errorf("facts changed to %v", got)
	}
	a, b := (combat.AreaMembership{}).Assess(f), (combat.AreaMembership{}).Assess(f)
	if a != b || a != (combat.AreaMembershipExcluded{Reason: combat.ExcludedAllLinesBlocked}) {
		t.Errorf("Assess twice = %v, %v", a, b)
	}
}

// Requirement: doc/specs/magical-effects.md section 5 - the fact constructors reject an undefined containment, an invalid Cover and a zero origin decision with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestLocationFactsConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	for name, err := range map[string]error{
		"containment 0":      second(combat.NewOtherLocationFacts(0, nil)),
		"containment past":   second(combat.NewOtherLocationFacts(combat.OutsideShape+1, nil)),
		"negative cover":     second(combat.NewOtherLocationFacts(combat.InsideShape, []combat.Cover{-1})),
		"cover past total":   second(combat.NewOtherLocationFacts(combat.InsideShape, []combat.Cover{combat.TotalCover + 1})),
		"zero origin choice": second(combat.NewOriginLocationFacts(combat.OriginInclusionDecision{})),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

// Requirement: none - pins the String values of the exclusion reasons.
// Path: happy.
func TestAreaExclusionString(t *testing.T) {
	t.Parallel()
	for r, want := range map[combat.AreaExclusion]string{combat.ExcludedOutsideShape: "outside_shape", combat.ExcludedAllLinesBlocked: "all_lines_blocked",
		combat.ExcludedOrigin: "origin_excluded", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("%d.String() = %q, want %q", int(r), r.String(), want)
		}
	}
}
