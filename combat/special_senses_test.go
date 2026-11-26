package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var senseR combat.SpecialSenseRules

func senses(rs ...combat.SenseRange) combat.SenseRanges     { return combat.NewSenseRanges(rs...) }
func sr(s content.SpecialSense, feet int) combat.SenseRange { return combat.MustSenseRange(s, feet) }
func space(l content.LightLevel, k content.DarknessKind, o content.Obscurement) combat.SpaceVisibility {
	return combat.MustSpaceVisibility(l, k, o)
}

var (
	dv  = content.SenseDarkvision
	bs  = content.SenseBlindsight
	ts  = content.SenseTruesight
	trs = content.SenseTremorsense
)

// Requirement: doc/specs/hiding-searching-and-senses.md FR-017, FR-097 - a SenseRange rejects a negative range and an
// undefined sense with a typed error matching ErrInvalidInput, accepts 0, and MustSenseRange panics with that error.
// Path: unhappy (invalid input).
func TestNewSenseRange(t *testing.T) {
	t.Parallel()
	if r, err := combat.NewSenseRange(dv, 0); err != nil || r.Sense() != dv || r.RangeFeet() != 0 {
		t.Errorf("range 0 = %v, %v", r, err)
	}
	for name, c := range map[string]struct {
		sense  content.SpecialSense
		feet   int
		reason combat.SenseRangeReason
	}{
		"negative range": {dv, -1, combat.SenseRangeReasonNegativeRange},
		"unknown sense":  {0, 5, combat.SenseRangeReasonUnknownSense},
		"beyond senses":  {5, 5, combat.SenseRangeReasonUnknownSense},
	} {
		_, err := combat.NewSenseRange(c.sense, c.feet)
		var e *combat.SenseRangeError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &e) || e.Reason != c.reason || !errors.Is(err, &combat.SenseRangeError{Reason: c.reason}) || errors.Is(err, &combat.SenseRangeError{Reason: 99}) {
			t.Errorf("%s: error = %v, want reason %v", name, err, c.reason)
		}
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustSenseRange panic = %v", r)
		}
	}()
	combat.MustSenseRange(dv, -1)
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-018, FR-100, FR-104 (SRD 5.2.1, "Drow", p. 85) - SenseRanges keeps the
// greater of two ranges for a sense in either order, ignores a zero SenseRange, orders by sense, and is a copy.
// Path: unhappy (rule interaction).
func TestSenseRanges(t *testing.T) {
	t.Parallel()
	for _, in := range [][]combat.SenseRange{{sr(dv, 60), sr(dv, 120)}, {sr(dv, 120), sr(dv, 60), {}}} {
		if r, ok := combat.NewSenseRanges(in...).Range(dv); !ok || r != 120 {
			t.Errorf("Range(darkvision) = %d, %v, want 120", r, ok)
		}
	}
	zero := combat.SenseRanges{}
	if _, ok := zero.Range(dv); ok || len(zero.All()) != 0 || zero.All() == nil {
		t.Error("the zero SenseRanges holds a sense or returns nil")
	}
	if _, ok := zero.Range(0); ok {
		t.Error("an undefined sense is held")
	}
	if r, ok := senses(sr(trs, 0)).Range(trs); !ok || r != 0 {
		t.Errorf("a range of 0 = %d, %v, want held", r, ok)
	}
	in := []combat.SenseRange{sr(ts, 30), sr(bs, 10)}
	s := combat.NewSenseRanges(in...)
	in[0] = sr(ts, 999)
	all := s.All()
	if want := []combat.SenseRange{sr(bs, 10), sr(ts, 30)}; !slices.Equal(all, want) {
		t.Errorf("All = %v, want %v", all, want)
	}
	all[0] = sr(bs, 999)
	if r, _ := s.Range(bs); r != 10 {
		t.Error("All is not a copy [FR-100]")
	}
	if s != combat.NewSenseRanges(sr(bs, 10), sr(ts, 30)) {
		t.Error("equal senses are not equal")
	}
}

// Requirement: per case
// Path: per case.
func TestReach(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Darkvision", p. 180`
	w, b := combat.SenseReachWithinRange, combat.SenseReachBeyondRange
	cases := []struct {
		name, requirement, path string
		held                    combat.SenseRanges
		ask                     content.SpecialSense
		distance                int
		want                    combat.SenseReach
	}{
		{"well inside", "FR-019, " + p, "happy", senses(sr(dv, 60)), dv, 30, w},
		{"at the range", "FR-019", "unhappy (boundary)", senses(sr(dv, 60)), dv, 60, w},
		{"one beyond", "FR-019", "unhappy (boundary)", senses(sr(dv, 60)), dv, 61, b},
		{"zero distance", "FR-019", "unhappy (boundary)", senses(sr(dv, 60)), dv, 0, w},
		{"range 0, distance 0", "FR-019, FR-017", "unhappy (boundary)", senses(sr(bs, 0)), bs, 0, w},
		{"range 0, distance 1", "FR-019, FR-017", "unhappy (boundary)", senses(sr(bs, 0)), bs, 1, b},
		{"sense not held", "FR-019", "unhappy (empty and minimal)", senses(sr(dv, 60)), ts, 5, b},
		{"no senses", "FR-019", "unhappy (empty and minimal)", senses(), dv, 0, b},
		{"two Darkvision entries", "FR-018", "unhappy (rule interaction)", senses(sr(dv, 60), sr(dv, 120)), dv, 100, w},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got, err := senseR.Reach(c.ask, c.distance, c.held); err != nil || got != c.want {
				t.Errorf("Reach(%v, %d) = %v, %v, want %v [%s]", c.ask, c.distance, got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-020 - a negative distance is a typed ErrInvalidInput error from Reach,
// PerceivedLight and EffectiveObscurement, as is an invalid space.
// Path: unhappy (invalid input).
func TestSenseQueriesRejectBadDistanceAndSpace(t *testing.T) {
	t.Parallel()
	dark := space(content.LightDarkness, 0, 0)
	_, e1 := senseR.Reach(dv, -1, senses(sr(dv, 60)))
	_, e2 := senseR.PerceivedLight(dark, -1, senses())
	_, e3 := senseR.EffectiveObscurement(dark, -1, senses())
	_, e4 := senseR.PerceivedLight(combat.SpaceVisibility{}, 1, senses())
	_, e5 := senseR.EffectiveObscurement(combat.SpaceVisibility{}, 1, senses())
	for i, err := range []error{e1, e2, e3, e4, e5} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("query %d: error = %v, want ErrInvalidInput", i, err)
		}
	}
	if !errors.Is(e1, &combat.SenseRangeError{Reason: combat.SenseRangeReasonNegativeDistance}) {
		t.Error("Reach does not report the negative distance")
	}
}

// Requirement: per case
// Path: per case.
func TestPerceivedLightAndEffectiveObscurement(t *testing.T) {
	t.Parallel()
	br, di, da := content.LightBright, content.LightDim, content.LightDarkness
	nm, mg := content.DarknessNonmagical, content.DarknessMagical
	none, lightly, heavily := content.ObscurementNone, content.ObscurementLightly, content.ObscurementHeavily
	full, gray := combat.ColorPerceptionFullColor, combat.ColorPerceptionShadesOfGray
	cases := []struct {
		name, requirement, path string
		sp                      combat.SpaceVisibility
		held                    combat.SenseRanges
		distance                int
		level                   content.LightLevel
		color                   combat.ColorPerception
		obscurement             content.Obscurement
	}{
		{"dim, Darkvision", "FR-021", "happy", space(di, 0, 0), senses(sr(dv, 60)), 30, br, full, none},
		{"dark, Darkvision", "FR-022, FR-027", "happy", space(da, nm, 0), senses(sr(dv, 60)), 30, di, gray, lightly},
		{"dark, at range", "FR-022, FR-024", "unhappy (boundary)", space(da, nm, 0), senses(sr(dv, 60)), 60, di, gray, lightly},
		{"dark, one beyond", "FR-024", "unhappy (boundary)", space(da, nm, 0), senses(sr(dv, 60)), 61, da, full, heavily},
		{"dark, no Darkvision", "FR-024", "unhappy (empty and minimal)", space(da, nm, 0), senses(), 5, da, full, heavily},
		{"dim, beyond Darkvision", "FR-024", "unhappy (boundary)", space(di, 0, 0), senses(sr(dv, 60)), 61, di, full, lightly},
		{"bright, Darkvision", "FR-023", "unhappy (boundary)", space(br, 0, 0), senses(sr(dv, 60)), 10, br, full, none},
		{"magical darkness", "FR-025", "unhappy (rule interaction)", space(da, mg, 0), senses(sr(dv, 60)), 10, da, full, heavily},
		{"dark, heavy fog", "FR-026", "unhappy (rule interaction)", space(da, nm, heavily), senses(sr(dv, 60)), 10, di, gray, heavily},
		{"dim, Blindsight only", "FR-024, FR-101", "unhappy (rule interaction)", space(di, 0, 0), senses(sr(bs, 30)), 10, di, full, none},
		{"Darkvision and Truesight, in both", "FR-101", "unhappy (rule interaction)", space(da, nm, 0), senses(sr(dv, 60), sr(ts, 30)), 20, di, gray, none},
		{"only Darkvision reaches", "FR-101, FR-026", "unhappy (rule interaction)", space(da, nm, 0), senses(sr(dv, 60), sr(ts, 30)), 40, di, gray, lightly},
		{"magical darkness, both in range", "FR-101, FR-040", "unhappy (rule interaction)", space(da, mg, 0), senses(sr(dv, 60), sr(ts, 30)), 20, da, full, none},
		{"magical darkness, Truesight out", "FR-101, FR-025", "unhappy (rule interaction)", space(da, mg, 0), senses(sr(dv, 60), sr(ts, 30)), 40, da, full, heavily},
		{"Darkness, Blindsight one beyond", "FR-101, FR-032", "unhappy (boundary)", space(da, nm, 0), senses(sr(bs, 30)), 31, da, full, heavily},
		{"Tremorsense only", "FR-101, FR-038", "unhappy (rule interaction)", space(da, nm, 0), senses(sr(trs, 60)), 10, da, full, heavily},
		{"Truesight, Darkness", "FR-040", "happy", space(da, nm, 0), senses(sr(ts, 30)), 30, da, full, none},
		{"Truesight, Dim Light", "FR-040, D-11", "unhappy (boundary)", space(di, 0, 0), senses(sr(ts, 30)), 30, di, full, lightly},
		{"Truesight, magical Darkness beyond", "FR-040", "unhappy (boundary)", space(da, mg, 0), senses(sr(ts, 30)), 31, da, full, heavily},
		{"Truesight, heavy fog", "FR-041", "unhappy (rule interaction)", space(br, 0, heavily), senses(sr(ts, 60)), 10, br, full, heavily},
		{"Truesight, Darkness with patchy fog", "FR-041, FR-101", "unhappy (rule interaction)", space(da, nm, lightly), senses(sr(ts, 30)), 10, da, full, lightly},
		{"Blindsight, heavy fog", "FR-029, FR-101", "unhappy (rule interaction)", space(br, 0, heavily), senses(sr(bs, 30)), 30, br, full, none},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := senseR.PerceivedLight(c.sp, c.distance, c.held)
			if err != nil || p.Level() != c.level || p.Color() != c.color {
				t.Errorf("PerceivedLight = %v %v, %v, want %v %v [%s]", p.Level(), p.Color(), err, c.level, c.color, c.requirement)
			}
			if o, err := senseR.EffectiveObscurement(c.sp, c.distance, c.held); err != nil || o != c.obscurement {
				t.Errorf("EffectiveObscurement = %v, %v, want %v [%s]", o, err, c.obscurement, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-101 (4.14a) - SightCheckEffect of the effective obscurement is the
// effect on a check made to see for every observer: a Truesight observer in Darkness in range is unaffected; Darkvision
// only is Disadvantage; Tremorsense only is an automatic failure.
// Path: unhappy (rule interaction).
func TestEffectiveObscurementDrivesSightCheckEffect(t *testing.T) {
	t.Parallel()
	dark := space(content.LightDarkness, 0, 0)
	for _, c := range []struct {
		held distanceAndSenses
		want combat.SightCheckEffect
	}{
		{distanceAndSenses{20, senses(sr(dv, 60), sr(ts, 30))}, combat.SightCheckEffectUnaffected},
		{distanceAndSenses{40, senses(sr(dv, 60), sr(ts, 30))}, combat.SightCheckEffectDisadvantage},
		{distanceAndSenses{10, senses(sr(trs, 60))}, combat.SightCheckEffectAutomaticFailure},
	} {
		o, _ := senseR.EffectiveObscurement(dark, c.held.distance, c.held.ranges)
		if got := lightR.SightCheckEffect(o, combat.SightCheckKindPerceptionToSee); got != c.want {
			t.Errorf("effect at %d = %v, want %v", c.held.distance, got, c.want)
		}
	}
}

type distanceAndSenses struct {
	distance int
	ranges   combat.SenseRanges
}

// Requirement: per case
// Path: per case.
func TestTremorsense(t *testing.T) {
	t.Parallel()
	shared, none, air := combat.TremorsenseContactSharedSurfaceOrLiquid, combat.TremorsenseContactNoSharedSurface, combat.TremorsenseContactTargetInTheAir
	cr, mov, sta := combat.DetectedThingCreature, combat.DetectedThingMovingObject, combat.DetectedThingStationaryObject
	held := senses(sr(trs, 60))
	cases := []struct {
		name, requirement, path string
		distance                int
		held                    combat.SenseRanges
		contact                 combat.TremorsenseContact
		thing                   combat.DetectedThing
		want                    combat.TremorsenseResult
	}{
		{"creature", "FR-033", "happy", 30, held, shared, cr, combat.TremorsenseResultPinpointed},
		{"moving object", "FR-033", "happy", 30, held, shared, mov, combat.TremorsenseResultPinpointed},
		{"at the range", "FR-033", "unhappy (boundary)", 60, held, shared, cr, combat.TremorsenseResultPinpointed},
		{"one beyond", "FR-034", "unhappy (boundary)", 61, held, shared, cr, combat.TremorsenseResultNotDetected},
		{"no shared surface", "FR-035", "unhappy (state error)", 30, held, none, cr, combat.TremorsenseResultNotDetected},
		{"flying creature", "FR-036", "unhappy (state error)", 30, held, air, cr, combat.TremorsenseResultNotDetected},
		{"flying moving object", "FR-036", "unhappy (state error)", 30, held, air, mov, combat.TremorsenseResultNotDetected},
		{"stationary object", "FR-037", "unhappy (boundary)", 30, held, shared, sta, combat.TremorsenseResultNotDetected},
		{"no Tremorsense", "FR-034", "unhappy (empty and minimal)", 10, senses(), shared, cr, combat.TremorsenseResultNotDetected},
		{"undefined thing", "FR-033", "unhappy (invalid input)", 10, held, shared, 0, combat.TremorsenseResultNotDetected},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			reach, err := senseR.Reach(trs, c.distance, c.held)
			if err != nil {
				t.Fatal(err)
			}
			if got := senseR.Tremorsense(reach, c.contact, c.thing); got != c.want {
				t.Errorf("Tremorsense = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-042 to FR-046, FR-048, FR-095, FR-096 (SRD 5.2.1, "Truesight", p. 190) - the
// Truesight queries at 30 ft: within range the illusion is Transparent and its save an AutomaticSuccess, a transformed
// creature that is seen is Discerned, the Ethereal Plane and an Invisible thing are seen; beyond range, without Truesight,
// for an illusion that does not deceive sight, or for an unseen or untransformed creature, nothing changes.
// Path: per case.
func TestTruesightQueries(t *testing.T) {
	t.Parallel()
	scene := func(d int, rs ...combat.SenseRange) combat.TruesightScene {
		s, err := combat.NewTruesightScene(d, senses(rs...))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	deceives, plain := combat.IllusionFacts{DeceivesSight: true}, combat.IllusionFacts{}
	in, edge, out, none := scene(10, sr(ts, 30)), scene(30, sr(ts, 30)), scene(31, sr(ts, 30)), scene(10)
	for _, c := range []struct {
		name  string
		s     combat.TruesightScene
		facts combat.IllusionFacts
		want  bool
	}{{"in range", in, deceives, true}, {"at range", edge, deceives, true}, {"beyond", out, deceives, false},
		{"no Truesight", none, deceives, false}, {"does not deceive sight", in, plain, false}} {
		app, save := senseR.IllusionAppearance(c.s, c.facts), senseR.IllusionSave(c.s, c.facts)
		wantApp, wantSave := combat.IllusionAppearanceAsCreated, combat.IllusionSaveRollNormally
		if c.want {
			wantApp, wantSave = combat.IllusionAppearanceTransparent, combat.IllusionSaveAutomaticSuccess
		}
		if app != wantApp || save != wantSave {
			t.Errorf("%s: appearance %v, save %v, want %v, %v [FR-043, FR-044, FR-048, FR-095]", c.name, app, save, wantApp, wantSave)
		}
	}
	seen, unseen := combat.TrueFormFacts{TransformedByMagic: true, ObserverSeesIt: combat.SightRelationCanSee}, combat.TrueFormFacts{TransformedByMagic: true, ObserverSeesIt: combat.SightRelationCannotSee}
	untransformed := combat.TrueFormFacts{ObserverSeesIt: combat.SightRelationCanSee}
	for _, c := range []struct {
		name  string
		s     combat.TruesightScene
		facts combat.TrueFormFacts
		want  combat.TrueFormPerception
	}{{"seen", in, seen, combat.TrueFormPerceptionDiscerned}, {"not seen", in, unseen, combat.TrueFormPerceptionNotDiscerned},
		{"not transformed", in, untransformed, combat.TrueFormPerceptionNotDiscerned}, {"beyond range", out, seen, combat.TrueFormPerceptionNotDiscerned}} {
		if got := senseR.TrueForm(c.s, c.facts); got != c.want {
			t.Errorf("TrueForm %s = %v, want %v [FR-045, FR-095]", c.name, got, c.want)
		}
	}
	if senseR.EtherealView(scene(10, sr(ts, 30))) != combat.EtherealViewSeen || senseR.EtherealView(scene(50, sr(ts, 60))) != combat.EtherealViewSeen ||
		senseR.EtherealView(scene(50, sr(ts, 30))) != combat.EtherealViewNotSeen {
		t.Error("EtherealView does not follow the Truesight range [FR-046]")
	}
	if !senseR.SeesInvisible(edge) || senseR.SeesInvisible(out) || senseR.SeesInvisible(none) {
		t.Error("SeesInvisible does not follow the Truesight range [FR-042]")
	}
	if _, err := combat.NewTruesightScene(-1, senses()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative distance error = %v [FR-096]", err)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md section 5 - the sense enums print their snake_case names.
// Path: unhappy (boundary).
func TestSenseEnumStrings(t *testing.T) {
	t.Parallel()
	got := []string{combat.SenseReachWithinRange.String(), combat.SenseReachBeyondRange.String(), combat.SenseReach(0).String(),
		combat.ColorPerceptionFullColor.String(), combat.ColorPerceptionShadesOfGray.String(), combat.ColorPerception(0).String(),
		combat.TremorsenseResultPinpointed.String(), combat.TremorsenseResultNotDetected.String(), combat.TremorsenseResult(0).String(),
		combat.SenseRangeReasonUnknownSense.String(), combat.SenseRangeReasonNegativeRange.String(), combat.SenseRangeReasonNegativeDistance.String(), combat.SenseRangeReason(0).String()}
	want := []string{"within_range", "beyond_range", "unknown", "full_color", "shades_of_gray", "unknown", "pinpointed", "not_detected", "unknown",
		"unknown_sense", "negative_range", "negative_distance", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("strings = %v, want %v", got, want)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-020, FR-096, FR-019 - distance 0 is valid everywhere a distance is taken; an
// undefined reach, contact or sight relation never counts as within range, shared ground or seen.
// Path: unhappy (boundary).
func TestSenseQueriesAtZeroAndUndefinedFacts(t *testing.T) {
	t.Parallel()
	dark := space(content.LightDarkness, 0, 0)
	if _, err := combat.NewTruesightScene(0, senses()); err != nil {
		t.Errorf("TruesightScene at 0 ft: %v", err)
	}
	if p, err := senseR.PerceivedLight(dark, 0, senses(sr(dv, 0))); err != nil || p.Level() != content.LightDim {
		t.Errorf("PerceivedLight at 0 ft = %v, %v", p.Level(), err)
	}
	if o, err := senseR.EffectiveObscurement(dark, 0, senses()); err != nil || o != content.ObscurementHeavily {
		t.Errorf("EffectiveObscurement at 0 ft = %v, %v", o, err)
	}
	if senseR.Tremorsense(0, combat.TremorsenseContactSharedSurfaceOrLiquid, combat.DetectedThingCreature) != combat.TremorsenseResultNotDetected ||
		senseR.Tremorsense(combat.SenseReachWithinRange, 0, combat.DetectedThingCreature) != combat.TremorsenseResultNotDetected {
		t.Error("an undefined reach or contact pinpoints")
	}
	scene, _ := combat.NewTruesightScene(5, senses(sr(ts, 30)))
	if senseR.TrueForm(scene, combat.TrueFormFacts{TransformedByMagic: true}) != combat.TrueFormPerceptionNotDiscerned {
		t.Error("an undefined sight relation discerns the true form")
	}
}
