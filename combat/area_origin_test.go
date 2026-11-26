package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func meCone(n int) content.ConeArea { return meMust(content.NewConeArea(n)) }

// meShapes are the six shapes with their default origin inclusion (spell-effect-data.md FR-014).
func meShapes() map[string]struct {
	shape content.AreaShape
	page  string
	incl  bool
} {
	return map[string]struct {
		shape content.AreaShape
		page  string
		incl  bool
	}{
		"cone":      {meCone(60), "p. 179", false},
		"cube":      {meMust(content.NewCubeArea(15)), "p. 179", false},
		"cylinder":  {meMust(content.NewCylinderArea(10, 40)), "p. 180", true},
		"emanation": {meMust(content.NewEmanationArea(10)), "p. 181", false},
		"line":      {meMust(content.NewLineArea(30, 5)), "p. 184", false},
		"sphere":    {meMust(content.NewSphereArea(20)), "p. 188", true},
	}
}

// Requirement: per case
// Path: per case.
func TestConeWidthAt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path     string
		length, dist   int
		want           int
		wantInvalidErr bool
	}{
		{"the SRD's own example", "happy", 60, 15, 15, false},
		{"at the origin", "unhappy (boundary)", 60, 0, 0, false},
		{"at maximum length", "unhappy (boundary)", 60, 60, 60, false},
		{"one beyond maximum length", "unhappy (boundary)", 60, 61, 0, true},
		{"negative distance", "unhappy (invalid input)", 60, -1, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: FR-013, FR-014; table 4.3b; SRD 5.2.1, "Cone", p. 179
			got, err := combat.ConeWidthAt(meCone(c.length), c.dist)
			if got != c.want || c.wantInvalidErr != errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("ConeWidthAt(%d, %d) = %d, %v, want %d (error %v) [%s]", c.length, c.dist, got, err, c.want, c.wantInvalidErr, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-014 - the zero (invalid) cone has no length, so every distance is an ErrInvalidInput error, 0 included.
// Path: unhappy (invalid input).
func TestConeWidthAtZeroCone(t *testing.T) {
	t.Parallel()
	if _, err := combat.ConeWidthAt(content.ConeArea{}, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ConeWidthAt(zero cone, 0) error = %v", err)
	}
}

// Requirement: doc/specs/magical-effects.md FR-016, table 4.3c row "each shape's default" - ShapeDefaultOrigin gives OriginIncluded equal to the shape's IncludesOrigin: false for Cone, Cube, Emanation and Line, true for Cylinder and Sphere.
// Path: happy.
func TestShapeDefaultOriginFollowsTheShape(t *testing.T) {
	t.Parallel()
	for name, s := range meShapes() {
		d, err := combat.ShapeDefaultOrigin(s.shape)
		if err != nil || d.OriginIncluded() != s.incl || d.OriginIncluded() != s.shape.IncludesOrigin() || d.ByCreator() || d.Shape() != s.shape {
			t.Errorf("%s (%s): got %v, %v, want included %v and not by creator", name, s.page, d, err, s.incl)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-016, table 4.3c rows "Cone/Cube/Emanation/Line included by creator" - CreatorIncludesOrigin gives OriginIncluded true for each of the four overridable shapes, by the creator.
// Path: unhappy (rule interaction).
func TestCreatorIncludesOriginForTheFourOverridableShapes(t *testing.T) {
	t.Parallel()
	shapes := meShapes()
	for name, o := range map[string]combat.OverridableAreaShape{
		"cone":      combat.OverridableCone{Area: shapes["cone"].shape.(content.ConeArea)},
		"cube":      combat.OverridableCube{Area: shapes["cube"].shape.(content.CubeArea)},
		"emanation": combat.OverridableEmanation{Area: shapes["emanation"].shape.(content.EmanationArea)},
		"line":      combat.OverridableLine{Area: shapes["line"].shape.(content.LineArea)},
	} {
		d, err := combat.CreatorIncludesOrigin(o)
		if err != nil || !d.OriginIncluded() || !d.ByCreator() || d.Shape() != shapes[name].shape || o.Shape() != shapes[name].shape {
			t.Errorf("%s: got %v, %v", name, d, err)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-016, table 4.3c row "the four overridable shapes" - OverridableAreaShape has exactly four variants (Cone, Cube, Emanation, Line), so a Cylinder or Sphere override cannot be built (SRD 5.2.1, "Cylinder", p. 180; "Sphere", p. 188).
// Path: unhappy (boundary).
func TestOverridableAreaShapeHasExactlyFourVariants(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, o := range []combat.OverridableAreaShape{combat.OverridableCone{}, combat.OverridableCube{}, combat.OverridableEmanation{}, combat.OverridableLine{}} {
		switch o.(type) {
		case combat.OverridableCone:
			seen["cone"] = true
		case combat.OverridableCube:
			seen["cube"] = true
		case combat.OverridableEmanation:
			seen["emanation"] = true
		case combat.OverridableLine:
			seen["line"] = true
		}
	}
	if len(seen) != 4 {
		t.Errorf("saw %v", seen)
	}
}

// Requirement: doc/specs/magical-effects.md FR-016 - an invalid shape (nil, zero or with a zero dimension) is an ErrInvalidInput error for both decisions.
// Path: unhappy (invalid input).
func TestOriginDecisionsRejectInvalidShapes(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]content.AreaShape{
		"nil": nil, "zero cone": content.ConeArea{}, "zero cube": content.CubeArea{}, "zero cylinder": content.CylinderArea{},
		"zero emanation": content.EmanationArea{}, "zero line": content.LineArea{}, "zero sphere": content.SphereArea{},
	} {
		if d, err := combat.ShapeDefaultOrigin(s); !errors.Is(err, srderr.ErrInvalidInput) || !d.IsZero() {
			t.Errorf("ShapeDefaultOrigin(%s) = %v, %v", name, d, err)
		}
	}
	for name, o := range map[string]combat.OverridableAreaShape{
		"nil": nil, "zero cone": combat.OverridableCone{}, "zero cube": combat.OverridableCube{},
		"zero emanation": combat.OverridableEmanation{}, "zero line": combat.OverridableLine{},
	} {
		if d, err := combat.CreatorIncludesOrigin(o); !errors.Is(err, srderr.ErrInvalidInput) || !d.IsZero() {
			t.Errorf("CreatorIncludesOrigin(%s) = %v, %v", name, d, err)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestEmanationEffectTimingMovesWithOrigin(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		timing     combat.EmanationEffectTiming
		want       bool
		str        string
	}{
		{"ongoing moves with its origin", "happy", combat.EmanationOngoing, true, "ongoing"},
		{"instantaneous does not", "unhappy (boundary)", combat.EmanationInstantaneous, false, "instantaneous"},
		{"stationary does not", "unhappy (boundary)", combat.EmanationStationary, false, "stationary"},
		{"an undefined timing does not", "unhappy (invalid input)", 0, false, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: FR-017; SRD 5.2.1, "Emanation", p. 181
			if got := c.timing.MovesWithOrigin(); got != c.want || c.timing.String() != c.str {
				t.Errorf("%s: MovesWithOrigin = %v, String = %q [%s]", c.name, got, c.timing.String(), c.path)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestAreaOriginResolve(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		placement  combat.OriginPlacement
		want       combat.AreaOriginResolution
		str        string
	}{
		{"a seen point is the chosen point", "happy", combat.SeenPoint, combat.ChosenPoint, "chosen_point"},
		{"an unseen point with no obstruction is the chosen point", "unhappy (rule interaction)", combat.UnseenPointNoObstruction, combat.ChosenPoint, "chosen_point"},
		{"an unseen point behind an obstruction is the near side", "unhappy (rule interaction)", combat.UnseenPointBehindObstruction, combat.NearSideOfObstruction, "near_side_of_obstruction"},
		{"an undefined placement is invalid", "unhappy (invalid input)", 0, 0, "unknown"},
		{"a placement past the last is invalid", "unhappy (boundary)", combat.UnseenPointBehindObstruction + 1, 0, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: FR-018; SRD 5.2.1, "Area of Effect", p. 177
			if got := (combat.AreaOrigin{}).Resolve(c.placement); got != c.want || got.String() != c.str {
				t.Errorf("Resolve(%d) = %v, want %v [%s]", c.placement, got, c.want, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-013, FR-014 and spell-effect-data.md FR-013 (a dimension is at least 1 foot) - a cone of 1 foot has width 0 at the origin and 1 at its end, and every shape with all dimensions at 1 foot is a valid shape for both decisions.
// Path: unhappy (boundary).
func TestMinimumShapesAreValid(t *testing.T) {
	t.Parallel()
	one := meCone(1)
	for d, want := range map[int]int{0: 0, 1: 1} {
		if got, err := combat.ConeWidthAt(one, d); err != nil || got != want {
			t.Errorf("ConeWidthAt(1, %d) = %d, %v, want %d", d, got, err, want)
		}
	}
	if _, err := combat.ConeWidthAt(one, 2); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ConeWidthAt(1, 2) error = %v", err)
	}
	for name, s := range map[string]content.AreaShape{
		"cone": one, "cube": meMust(content.NewCubeArea(1)), "cylinder": meMust(content.NewCylinderArea(1, 1)),
		"emanation": meMust(content.NewEmanationArea(1)), "line": meMust(content.NewLineArea(1, 1)), "sphere": meMust(content.NewSphereArea(1)),
	} {
		if d, err := combat.ShapeDefaultOrigin(s); err != nil || d.IsZero() {
			t.Errorf("ShapeDefaultOrigin(1-foot %s) = %v, %v", name, d, err)
		}
	}
	for name, o := range map[string]combat.OverridableAreaShape{
		"cone": combat.OverridableCone{Area: one}, "cube": combat.OverridableCube{Area: meMust(content.NewCubeArea(1))},
		"emanation": combat.OverridableEmanation{Area: meMust(content.NewEmanationArea(1))}, "line": combat.OverridableLine{Area: meMust(content.NewLineArea(1, 1))},
	} {
		if d, err := combat.CreatorIncludesOrigin(o); err != nil || !d.OriginIncluded() {
			t.Errorf("CreatorIncludesOrigin(1-foot %s) = %v, %v", name, d, err)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-017 - only the ongoing timing moves with its origin; a timing past the last is undefined and does not.
// Path: unhappy (boundary).
func TestEmanationTimingPastTheLastDoesNotMove(t *testing.T) {
	t.Parallel()
	if (combat.EmanationOngoing + 1).MovesWithOrigin() {
		t.Error("an undefined timing must not move with its origin")
	}
}
