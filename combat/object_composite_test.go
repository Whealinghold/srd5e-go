package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func twoSections(t *testing.T, a, b combat.CollapseRuling) combat.CompositeObject {
	t.Helper()
	live := envObject(t, 10, 0, combat.NoAffinities())
	c, err := combat.NewCompositeObject([]combat.CompositeSection{{Status: live, Ruling: a}, {Status: live, Ruling: b}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-089, FR-090, FR-114 - a composite object has two or more sections with a ruling each, copied in and out; one section, a zero status, an invalid ruling and a bad index are ErrInvalidInput errors.
// Path: unhappy (invalid input).
func TestCompositeObjectConstruction(t *testing.T) {
	t.Parallel()
	live := envObject(t, 10, 0, combat.NoAffinities())
	good := combat.CompositeSection{Status: live, Ruling: combat.CollapsesWhole}
	for name, sections := range map[string][]combat.CompositeSection{"none": nil, "one section": {good},
		"zero status": {good, {Ruling: combat.DoesNotCollapse}}, "invalid ruling": {good, {Status: live}}, "unknown ruling": {good, {Status: live, Ruling: 9}}} {
		if _, err := combat.NewCompositeObject(sections); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	in := []combat.CompositeSection{good, good}
	c, err := combat.NewCompositeObject(in)
	if err != nil {
		t.Fatal(err)
	}
	in[0].Ruling = combat.DoesNotCollapse
	c.Sections()[1].Ruling = combat.DoesNotCollapse
	if c.Sections()[0] != good || c.Sections()[1] != good || len(c.Sections()) != 2 {
		t.Error("the composite aliases a slice")
	}
	if three, err := combat.NewCompositeObject([]combat.CompositeSection{good, good, good}); err != nil || len(three.Sections()) != 3 {
		t.Errorf("three sections = %v, %v, want a composite", three, err)
	}
	for _, i := range []int{-1, 2, 5} {
		if _, ev, err := c.ReplaceSection(i, live); !errors.Is(err, srderr.ErrInvalidInput) || ev != nil {
			t.Errorf("ReplaceSection(%d) = %v, %v, want ErrInvalidInput", i, ev, err)
		}
	}
	if _, _, err := c.ReplaceSection(0, combat.ObjectStatus{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero status = %v", err)
	}
	if (combat.CompositeObject{}).Status() != combat.CompositeStanding {
		t.Error("the zero composite is not standing")
	}
	for r, n := range map[combat.CollapseRuling]string{combat.DoesNotCollapse: "does_not_collapse", combat.CollapsesWhole: "collapses_whole", 0: "unknown"} {
		if r.String() != n || (r != 0) != r.Valid() {
			t.Errorf("ruling %d = %q valid %v", int(r), r, r.Valid())
		}
	}
	for s, n := range map[combat.CompositeStatus]string{combat.CompositeStanding: "standing", combat.CompositeCollapsed: "collapsed", combat.CompositeAllSectionsDestroyed: "all_sections_destroyed", 0: "unknown"} {
		if s.String() != n {
			t.Errorf("status %d = %q", int(s), s)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-091 to FR-093 (table 4.9e) - a destroyed section whose ruling is CollapsesWhole collapses the composite once with one ObjectCollapsed; another section destroyed does not; every section destroyed with no collapse is allSectionsDestroyed; collapse takes precedence; a later change does not repeat the event.
// Path: per case.
func TestCompositeObjectCollapse(t *testing.T) {
	t.Parallel()
	live := envObject(t, 10, 0, combat.NoAffinities())
	hurt := envObject(t, 1, 0, combat.NoAffinities())
	dead := envObject(t, 0, 0, combat.NoAffinities())
	type step struct {
		index  int
		status combat.ObjectStatus
		events int
		after  combat.CompositeStatus
	}
	cases := []struct {
		name, requirement, path string
		a, b                    combat.CollapseRuling
		steps                   []step
	}{
		{"damaged, not destroyed", "FR-091", "happy", combat.CollapsesWhole, combat.DoesNotCollapse, []step{{0, hurt, 0, combat.CompositeStanding}}},
		{"collapsing section destroyed", "FR-091", "happy", combat.CollapsesWhole, combat.DoesNotCollapse, []step{{0, dead, 1, combat.CompositeCollapsed}}},
		{"other section destroyed", "FR-091", "unhappy (rule interaction)", combat.CollapsesWhole, combat.DoesNotCollapse, []step{{1, dead, 0, combat.CompositeStanding}}},
		{"all destroyed, no collapse", "FR-092", "unhappy (rule interaction)", combat.DoesNotCollapse, combat.DoesNotCollapse, []step{{0, dead, 0, combat.CompositeStanding}, {1, dead, 0, combat.CompositeAllSectionsDestroyed}}},
		{"collapse beats all destroyed", "FR-092, FR-093", "unhappy (rule interaction)", combat.CollapsesWhole, combat.DoesNotCollapse, []step{{1, dead, 0, combat.CompositeStanding}, {0, dead, 1, combat.CompositeCollapsed}}},
		{"already collapsed", "FR-093", "unhappy (state error)", combat.CollapsesWhole, combat.DoesNotCollapse, []step{{0, dead, 1, combat.CompositeCollapsed}, {1, dead, 0, combat.CompositeCollapsed}}},
		{"collapsing section replaced again", "FR-093", "unhappy (state error)", combat.CollapsesWhole, combat.CollapsesWhole, []step{{0, dead, 1, combat.CompositeCollapsed}, {1, dead, 0, combat.CompositeCollapsed}, {0, live, 0, combat.CompositeCollapsed}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			comp := twoSections(t, c.a, c.b)
			for i, s := range c.steps {
				before := comp.Sections()
				next, ev, err := comp.ReplaceSection(s.index, s.status)
				if err != nil || len(ev) != s.events || next.Status() != s.after || comp.Sections()[s.index] != before[s.index] || next.Sections()[s.index].Status != s.status {
					t.Fatalf("step %d = %d events, %v, %v, want %d events and %v [%s]", i, len(ev), next.Status(), err, s.events, s.after, c.requirement)
				}
				if s.events == 1 && ev[0] != (combat.ObjectCollapsed{}) {
					t.Errorf("step %d event = %v", i, ev[0])
				}
				comp = next
			}
		})
	}
}
