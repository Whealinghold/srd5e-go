package content_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: spell-effect-data.md FR-013, FR-014, table A1 to A6 - each of
// the six shapes holds its dimensions and states whether it includes its
// origin.
// Path: per case.
func TestAreaShapesHoldDimensionsAndOrigin(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		shape   content.AreaShape
		origin  bool
		dimsAre func() bool
	}{
		{"A1 cone", must(content.NewConeArea(15)), false, func() bool { return must(content.NewConeArea(15)).LengthFeet() == 15 }},
		{"A2 cube", must(content.NewCubeArea(10)), false, func() bool { return must(content.NewCubeArea(10)).SideFeet() == 10 }},
		{"A3 cylinder", must(content.NewCylinderArea(10, 40)), true, func() bool {
			c := must(content.NewCylinderArea(10, 40))
			return c.RadiusFeet() == 10 && c.HeightFeet() == 40
		}},
		{"A4 emanation", must(content.NewEmanationArea(30)), false, func() bool { return must(content.NewEmanationArea(30)).DistanceFeet() == 30 }},
		{"A5 line", must(content.NewLineArea(100, 5)), false, func() bool {
			l := must(content.NewLineArea(100, 5))
			return l.LengthFeet() == 100 && l.WidthFeet() == 5
		}},
		{"A6 sphere", must(content.NewSphereArea(20)), true, func() bool { return must(content.NewSphereArea(20)).RadiusFeet() == 20 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.shape.IncludesOrigin() != c.origin {
				t.Errorf("IncludesOrigin() = %v, want %v [FR-014]", c.shape.IncludesOrigin(), c.origin)
			}
			if !c.dimsAre() {
				t.Error("the dimensions are not those given [FR-013]")
			}
		})
	}
}

// Requirement: spell-effect-data.md FR-013, table A7 to A10 - every dimension
// is at least 1; 1 is accepted.
// Path: unhappy (boundary).
func TestAreaDimensionBoundaries(t *testing.T) {
	t.Parallel()
	_, e1 := content.NewSphereArea(0)
	wantProblem(t, e1, "SphereArea", content.BlockOutOfRange)
	if s, err := content.NewSphereArea(1); err != nil || !s.IncludesOrigin() { // A8
		t.Errorf("NewSphereArea(1) = %v, %v", s, err)
	}
	_, e2 := content.NewLineArea(100, 0)
	wantProblem(t, e2, "LineArea", content.BlockOutOfRange)
	_, e3 := content.NewLineArea(0, 5)
	wantProblem(t, e3, "LineArea", content.BlockOutOfRange)
	_, e4 := content.NewCylinderArea(10, -40)
	wantProblem(t, e4, "CylinderArea", content.BlockOutOfRange)
	_, e5 := content.NewCylinderArea(0, 40)
	wantProblem(t, e5, "CylinderArea", content.BlockOutOfRange)
	_, e6 := content.NewConeArea(-1)
	wantProblem(t, e6, "ConeArea", content.BlockOutOfRange)
	_, e7 := content.NewCubeArea(0)
	wantProblem(t, e7, "CubeArea", content.BlockOutOfRange)
	_, e8 := content.NewEmanationArea(0)
	wantProblem(t, e8, "EmanationArea", content.BlockOutOfRange)
}

// Requirement: spell-effect-data.md FR-015, table T1 to T3 - a restriction has
// a non-empty set of valid, distinct creature types, in any order.
// Path: per case.
func TestCreatureTypeRestriction(t *testing.T) {
	t.Parallel()
	_, err := content.NewOnlyCreatureTypes(nil) // T1
	wantProblem(t, err, "OnlyCreatureTypes", content.BlockEmpty)
	_, err = content.NewOnlyCreatureTypes([]content.CreatureType{0})
	wantProblem(t, err, "OnlyCreatureTypes", content.BlockInvalidMember)
	_, err = content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeFey, content.CreatureTypeFey})
	wantProblem(t, err, "OnlyCreatureTypes", content.BlockDuplicate)
	h, err := content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}) // T3
	if err != nil || !h.Contains(content.CreatureTypeHumanoid) || h.Contains(content.CreatureTypeBeast) || h.Contains(0) {
		t.Errorf("Humanoid restriction = %v, %v", h, err)
	}
	a := must(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeUndead, content.CreatureTypeBeast}))
	b := must(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeBeast, content.CreatureTypeUndead}))
	if a != b {
		t.Error("the set is order-sensitive [FR-008]")
	}
	if got := a.Types(); !slices.Equal(got, []content.CreatureType{content.CreatureTypeBeast, content.CreatureTypeUndead}) {
		t.Errorf("Types() = %v, want vocabulary order", got)
	}
}

// Requirement: spell-effect-data.md FR-015, table T2 - a restriction other than
// AnyCreatureType is rejected with an object subject (chosen and area).
// Path: unhappy (invalid input).
func TestRestrictionOnObjectSubjectIsRejected(t *testing.T) {
	t.Parallel()
	h := must(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
	_, err := content.NewTargetsChosen(content.SubjectObject, exactly(1, nil), h, content.MayIncludeCaster)
	wantProblem(t, err, "TargetsChosen", content.BlockConstraint)
	_, err = content.NewTargetsInArea(sphere(5), content.SubjectObject, h)
	wantProblem(t, err, "TargetsInArea", content.BlockConstraint)
	for _, s := range []content.TargetSubject{content.SubjectCreature, content.SubjectCreatureOrObject} {
		if _, err := content.NewTargetsChosen(s, exactly(1, nil), h, content.MayIncludeCaster); err != nil {
			t.Errorf("subject %v with a restriction: %v", s, err)
		}
	}
	if got, err := content.NewTargetsChosen(content.SubjectObject, exactly(1, nil), content.AnyCreatureType{}, content.MayIncludeCaster); err != nil || got.Restriction() != content.CreatureTypeRestriction(content.AnyCreatureType{}) {
		t.Errorf("object with AnyCreatureType = %v, %v", got, err)
	}
}

// Requirement: spell-effect-data.md FR-010, FR-011, FR-040 - the three
// targetings, the subjects and the self policy are held and validated; a
// nil restriction is AnyCreatureType.
// Path: happy.
func TestTargetingVariants(t *testing.T) {
	t.Parallel()
	c, err := content.NewTargetsChosen(content.SubjectCreatureOrObject, content.TargetsByInstance{}, nil, content.ExcludesCaster)
	if err != nil || c.Subject() != content.SubjectCreatureOrObject || c.Count() != content.TargetCount(content.TargetsByInstance{}) ||
		c.SelfPolicy() != content.ExcludesCaster || c.Restriction() != content.CreatureTypeRestriction(content.AnyCreatureType{}) {
		t.Errorf("TargetsChosen = %v, %v", c, err)
	}
	a, err := content.NewTargetsInArea(sphere(20), content.SubjectCreature, nil)
	if err != nil || a.Shape() != content.AreaShape(sphere(20)) || a.Subject() != content.SubjectCreature ||
		a.Restriction() != content.CreatureTypeRestriction(content.AnyCreatureType{}) {
		t.Errorf("TargetsInArea = %v, %v", a, err)
	}
	for i, s := range []content.TargetSubject{content.SubjectCreature, content.SubjectObject, content.SubjectCreatureOrObject, 0, 4} {
		if got, want := s.Valid(), i < 3; got != want {
			t.Errorf("TargetSubject(%d).Valid() = %v, want %v", s, got, want)
		}
	}
}

// Requirement: spell-effect-data.md FR-010, FR-040 - invalid parts of a
// targeting are rejected, each with its own reason.
// Path: unhappy (invalid input).
func TestTargetingRejectsInvalidParts(t *testing.T) {
	t.Parallel()
	one := exactly(1, nil)
	_, err := content.NewTargetsChosen(0, one, nil, content.MayIncludeCaster)
	wantProblem(t, err, "TargetsChosen", content.BlockInvalidMember)
	_, err = content.NewTargetsChosen(content.SubjectCreature, one, nil, 0)
	wantProblem(t, err, "TargetsChosen", content.BlockInvalidMember)
	_, err = content.NewTargetsChosen(content.SubjectCreature, nil, nil, content.MayIncludeCaster)
	wantProblem(t, err, "TargetCount", content.BlockMissing)
	_, err = content.NewTargetsChosen(content.SubjectCreature, content.ExactlyTargets{}, nil, content.MayIncludeCaster)
	wantProblem(t, err, "ExactlyTargets", content.BlockOutOfRange)
	_, err = content.NewTargetsChosen(content.SubjectCreature, one, content.OnlyCreatureTypes{}, content.MayIncludeCaster)
	wantProblem(t, err, "OnlyCreatureTypes", content.BlockEmpty)
	_, err = content.NewTargetsInArea(nil, content.SubjectCreature, nil)
	wantProblem(t, err, "AreaShape", content.BlockMissing)
	_, err = content.NewTargetsInArea(content.SphereArea{}, content.SubjectCreature, nil)
	wantProblem(t, err, "SphereArea", content.BlockOutOfRange)
	_, err = content.NewTargetsInArea(sphere(5), 0, nil)
	wantProblem(t, err, "TargetsInArea", content.BlockInvalidMember)
}

// Requirement: spell-effect-data.md FR-012 - a counted target count is at
// least 1 and its scaling is valid; 0 and negative are rejected.
// Path: unhappy (boundary).
func TestTargetCountBoundaries(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		_, err := content.NewExactlyTargets(n, nil)
		wantProblem(t, err, "ExactlyTargets", content.BlockOutOfRange)
		_, err = content.NewUpToTargets(n, nil)
		wantProblem(t, err, "UpToTargets", content.BlockOutOfRange)
		_, err = content.NewPartInstances(n, nil)
		wantProblem(t, err, "PartInstances", content.BlockOutOfRange) // FR-029
	}
	if u, err := content.NewUpToTargets(3, nil); err != nil || u.Base() != 3 || u.Scaling() != content.CountScaling(content.NoCountScaling{}) {
		t.Errorf("NewUpToTargets(3) = %v, %v", u, err)
	}
	_, err := content.NewExactlyTargets(1, content.PerSlotLevelCount{})
	wantProblem(t, err, "PerSlotLevelCount", content.BlockOutOfRange)
	one, err := content.NewPartInstances(1, nil)
	if err != nil || one.Base() != 1 {
		t.Errorf("NewPartInstances(1) = %v, %v", one, err)
	}
	e := exactly(2, nil)
	if e.Base() != 2 {
		t.Errorf("Base() = %d, want 2", e.Base())
	}
}

// Requirement: spell-effect-data.md FR-013 - the zero value of every area
// shape is invalid and is rejected by its own reason when used in a
// targeting.
// Path: unhappy (invalid input).
func TestZeroAreaShapesAreRejected(t *testing.T) {
	t.Parallel()
	shapes := map[string]content.AreaShape{
		"ConeArea":      content.ConeArea{},
		"CubeArea":      content.CubeArea{},
		"CylinderArea":  content.CylinderArea{},
		"EmanationArea": content.EmanationArea{},
		"LineArea":      content.LineArea{},
		"SphereArea":    content.SphereArea{},
	}
	for name, shape := range shapes {
		_, err := content.NewTargetsInArea(shape, content.SubjectCreature, nil)
		wantProblem(t, err, name, content.BlockOutOfRange)
	}
}

// Requirement: spell-effect-data.md FR-012 - a count of exactly 1 with no
// scaling resolves to 1, and a zero UpToTargets is not a valid count.
// Path: boundary / unhappy (invalid input).
func TestUnscaledCountOfOneAndZeroUpToTargets(t *testing.T) {
	t.Parallel()
	if got, err := exactly(1, nil).At(0, 0); err != nil || got != 1 {
		t.Errorf("At() = %d, %v, want 1, nil", got, err)
	}
	_, err := content.NewTargetsChosen(content.SubjectCreature, content.UpToTargets{}, nil, content.MayIncludeCaster)
	wantProblem(t, err, "UpToTargets", content.BlockOutOfRange)
}
