package scenarios_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpD3AllSkills is the 18 skills of SRD 5.2.1, "Skills", p. 6.
func grpD3AllSkills() []content.Skill {
	var skills []content.Skill
	for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
		skills = append(skills, s)
	}
	return skills
}

// grpD3CircumstanceSkills counts the skills a travel mode puts a circumstance on.
func grpD3CircumstanceSkills(mode character.TravelMode) (count int) {
	for _, s := range grpD3AllSkills() {
		if len(character.TravelRules{}.Circumstances(mode, s)) > 0 {
			count++
		}
	}
	return count
}

// A party picks a travel pace for a journey: the printed figures per minute,
// hour and day, the order the toolbox relies on, and the fact that the figures
// are data, not multiples of each other (SRD 5.2.1, "Travel Pace", p. 12).
// Requirement: doc/specs/exploration.md FR-001 to FR-006, FR-046, FR-047 - three paces slowest first, every printed figure, positive and strictly falling, RaisedOneStep stops at fast, the figures are not derivable from one another.
// Path: happy; unhappy (boundary).
func TestGrpD3TravelPaceTableAndRaisedOneStep(t *testing.T) {
	t.Parallel()
	paces := character.TravelPaces()
	if len(paces) != 3 || paces[0] != character.PaceSlow || paces[1] != character.PaceNormal || paces[2] != character.PaceFast {
		t.Fatalf("TravelPaces = %v, want slow, normal, fast", paces)
	}
	want := map[character.TravelPace][3]int{
		character.PaceFast: {400, 4, 30}, character.PaceNormal: {300, 3, 24}, character.PaceSlow: {200, 2, 18},
	}
	for p, w := range want {
		row := p.Row()
		if row.FeetPerMinute() != w[0] || row.MilesPerHour() != w[1] || row.MilesPerDay() != w[2] {
			t.Errorf("%v = %d ft/min, %d mph, %d mi/day; want %v", p, row.FeetPerMinute(), row.MilesPerHour(), row.MilesPerDay(), w)
		}
	}
	// The printed figures disagree with arithmetic on each other (FR-047), so none may be computed.
	fast := character.PaceFast.Row()
	if fast.FeetPerMinute()*60 == fast.MilesPerHour()*5280 {
		t.Error("400 ft/min is not exactly 4 mph; the figures must be pinned, not derived")
	}
	if fast.MilesPerHour()*8 == fast.MilesPerDay() || character.PaceSlow.Row().MilesPerHour()*8 == character.PaceSlow.Row().MilesPerDay() {
		t.Error("the day figures are not eight hours of the hour figure")
	}
	for i := 1; i < len(paces); i++ {
		a, b := paces[i-1].Row(), paces[i].Row()
		if a.FeetPerMinute() >= b.FeetPerMinute() || a.MilesPerHour() >= b.MilesPerHour() || a.MilesPerDay() >= b.MilesPerDay() {
			t.Errorf("figures do not fall strictly from %v to %v", paces[i], paces[i-1])
		}
	}
	for in, out := range map[character.TravelPace]character.TravelPace{
		character.PaceSlow: character.PaceNormal, character.PaceNormal: character.PaceFast, character.PaceFast: character.PaceFast,
	} {
		if got := in.RaisedOneStep(); got != out {
			t.Errorf("%v raised one step = %v, want %v", in, got, out)
		}
	}
}

// The party's pace changes how well it notices and sneaks: Fast gives
// Disadvantage on Perception, Survival and Stealth, Normal only on Stealth,
// Slow gives Advantage on Perception and Survival, and a vehicle changes
// nothing (SRD 5.2.1, "Travel Pace", p. 12).
// Requirement: doc/specs/exploration.md FR-007 to FR-012, FR-014, FR-015, FR-017, FR-048, FR-049 - one circumstance on the named skills only (3, 1 and 2 of 18), joined by modifier.Resolve, a vehicle gives none, miles per hour are unscaled, slices are copies.
// Path: happy; unhappy (boundary, rule interaction, immutability, determinism).
func TestGrpD3TravelPaceChangesPerceptionSurvivalAndStealth(t *testing.T) {
	t.Parallel()
	rules := character.TravelRules{}
	fast, normal, slow := character.LandTravel{Pace: character.PaceFast}, character.LandTravel{Pace: character.PaceNormal}, character.LandTravel{Pace: character.PaceSlow}

	for _, c := range []struct {
		mode  character.TravelMode
		skill content.Skill
		want  modifier.RollCircumstance
	}{
		{fast, content.SkillPerception, modifier.Disadvantage{}}, {fast, content.SkillSurvival, modifier.Disadvantage{}},
		{fast, content.SkillStealth, modifier.Disadvantage{}}, {normal, content.SkillStealth, modifier.Disadvantage{}},
		{slow, content.SkillPerception, modifier.Advantage{}}, {slow, content.SkillSurvival, modifier.Advantage{}},
	} {
		got := rules.Circumstances(c.mode, c.skill)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%v on skill %v = %v, want [%v]", c.mode, c.skill, got, c.want)
		}
	}
	// Nothing else moves: the unnamed Wisdom and Dexterity skills included.
	for _, c := range []struct {
		mode  character.TravelMode
		skill content.Skill
	}{
		{normal, content.SkillPerception}, {normal, content.SkillSurvival}, {slow, content.SkillStealth},
		{fast, content.SkillInsight}, {fast, content.SkillMedicine}, {fast, content.SkillAnimalHandling},
		{fast, content.SkillAcrobatics}, {fast, content.SkillSleightOfHand}, {slow, content.SkillInsight},
	} {
		if got := rules.Circumstances(c.mode, c.skill); len(got) != 0 {
			t.Errorf("%v on %v = %v, want none", c.mode, c.skill, got)
		}
	}
	if grpD3CircumstanceSkills(fast) != 3 || grpD3CircumstanceSkills(normal) != 1 || grpD3CircumstanceSkills(slow) != 2 {
		t.Errorf("skills touched: fast %d, normal %d, slow %d; want 3, 1, 2", grpD3CircumstanceSkills(fast), grpD3CircumstanceSkills(normal), grpD3CircumstanceSkills(slow))
	}

	// The pace circumstance joins other sources only through the kernel: they cancel and never stack.
	dis := rules.Circumstances(fast, content.SkillPerception)
	if got := modifier.Resolve(append(dis, modifier.Advantage{})...); got != modifier.Normal {
		t.Errorf("fast pace with Advantage from elsewhere = %v, want normal", got)
	}
	adv := rules.Circumstances(slow, content.SkillPerception)
	if got := modifier.Resolve(append(adv, modifier.Disadvantage{}, modifier.Disadvantage{})...); got != modifier.Normal {
		t.Errorf("slow pace with two Disadvantages = %v, want normal", got)
	}
	dis[0] = modifier.Advantage{} // the caller's copy only
	if again := rules.Circumstances(fast, content.SkillPerception); again[0] != (modifier.Disadvantage{}) {
		t.Errorf("a second call = %v after mutating the first", again)
	}

	// A vehicle with a Speed does not "choose a travel pace".
	galley := character.VehicleTravel{Speed: character.MustVehicleSpeed(8)}
	if grpD3CircumstanceSkills(galley) != 0 {
		t.Errorf("a vehicle touched %d skills, want 0", grpD3CircumstanceSkills(galley))
	}

	// Miles per hour are the table's figure for land and the speed's for a vehicle, never scaled.
	for mode, want := range map[character.TravelMode]float64{fast: 4, normal: 3, slow: 2, galley: 4} {
		if got := rules.MilesPerHour(mode); got != want {
			t.Errorf("MilesPerHour(%v) = %v, want %v", mode, got, want)
		}
	}
	if got := rules.MilesPerHour(character.VehicleTravel{Speed: character.MustVehicleSpeed(3)}); got != 1.5 {
		t.Errorf("a 1.5 mph vehicle = %v, want 1.5", got)
	}
	for p, want := range map[character.TravelPace]int{character.PaceFast: 8, character.PaceNormal: 6, character.PaceSlow: 4} {
		if got := rules.MountedHourMiles(p); got != want {
			t.Errorf("mounted hour at %v = %d, want %d", p, got, want)
		}
	}
}

// A vehicle's Speed is a whole number of half-miles per hour, so a decoded
// value that is not a multiple of 0.5 is turned away (SRD 5.2.1, "Airborne and
// Waterborne Vehicles", p. 101).
// Requirement: doc/specs/exploration.md FR-016 - 1 to 2^52 half-miles per hour are accepted; zero, negative, too large or a non-multiple of 0.5 is an ExplorationError matching ErrInvalidInput.
// Path: unhappy (boundary, invalid input).
func TestGrpD3VehicleSpeedBoundsAndConversion(t *testing.T) {
	t.Parallel()
	for _, ok := range []int64{1, 2, 7, 1 << 52} {
		if s, err := character.NewVehicleSpeed(ok); err != nil || s.HalfMilesPerHour() != ok || s.MilesPerHour() != float64(ok)/2 || s.IsZero() {
			t.Errorf("NewVehicleSpeed(%d) = %v, %v", ok, s, err)
		}
	}
	for _, bad := range []int64{0, -1, 1<<52 + 1} {
		_, err := character.NewVehicleSpeed(bad)
		var ee *character.ExplorationError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ee) || ee.Input != character.ExplorationVehicleSpeed {
			t.Errorf("NewVehicleSpeed(%d) error = %v, want an ExplorationVehicleSpeed input error", bad, err)
		}
	}
	if s, err := character.VehicleSpeedFromMilesPerHour(4.5); err != nil || s.HalfMilesPerHour() != 9 {
		t.Errorf("4.5 mph = %v, %v; want 9 half-miles", s, err)
	}
	for _, bad := range []float64{0, -2, 0.3, 4.25} {
		if _, err := character.VehicleSpeedFromMilesPerHour(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%v mph error = %v, want ErrInvalidInput", bad, err)
		}
	}
	if !(character.VehicleSpeed{}).IsZero() {
		t.Error("the zero VehicleSpeed is invalid")
	}
}

// An adventuring Wizard keeps watch while the party travels: the pace changes
// the Perception roll mode and so the Passive Perception from the real
// derived-stats use case (SRD 5.2.1, "Passive Perception", p. 186).
// Requirement: doc/specs/exploration.md FR-012, FR-013 - the pace circumstance resolved by modifier.Resolve and fed to CalculateDerivedStats gives 5 more at slow, none at normal and 5 less at fast.
// Path: unhappy (rule interaction).
func TestGrpD3PaceFeedsPassivePerception(t *testing.T) {
	t.Parallel()
	scores, err := ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: grpD3Score(8), ability.Dexterity: grpD3Score(14), ability.Constitution: grpD3Score(12),
		ability.Intelligence: grpD3Score(16), ability.Wisdom: grpD3Score(15), ability.Charisma: grpD3Score(10),
	})
	if err != nil {
		t.Fatal(err)
	}
	levels := character.MustClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 5}})
	uc, err := character.NewCalculateDerivedStats(character.WithSource(data.MustBundledSource()))
	if err != nil {
		t.Fatal(err)
	}
	for pace, want := range map[character.TravelPace]int{character.PaceSlow: 17, character.PaceNormal: 12, character.PaceFast: 7} {
		mode := modifier.Resolve(character.TravelRules{}.Circumstances(character.LandTravel{Pace: pace}, content.SkillPerception)...)
		res, err := uc.Run(context.Background(), character.CalculateDerivedStatsCommand{
			ClassLevels: levels, AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(27), PerceptionRollMode: mode,
		})
		if err != nil {
			t.Fatal(err)
		}
		ok, isOK := res.(character.CalculateDerivedStatsSucceeded)
		if !isOK {
			t.Fatalf("pace %v: result = %#v", pace, res)
		}
		if got := ok.Stats.PassivePerception().Value(); got != want {
			t.Errorf("Passive Perception at %v = %d, want %d (10 + Wisdom +2, then +5, 0 or -5)", pace, got, want)
		}
	}
}

// A rider presses the horses for a double-distance hour, then cannot do it
// again until the mounts have rested; a rest that did not finish is never
// reported (SRD 5.2.1, "Travel Pace", p. 12).
// Requirement: doc/specs/exploration.md FR-019 to FR-025, FR-055 - a new boost is available; spend is accepted once then refused with mountsNeedRest; any finished rest restores; restoring an available boost changes nothing; events carry the character id.
// Path: happy; unhappy (state error, repeated action, invalid input).
func TestGrpD3MountedPaceBoostCycle(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("rider-1")
	boost, err := character.NewMountedPaceBoost(id)
	if err != nil {
		t.Fatal(err)
	}
	if boost.State() != character.MountedBoostAvailable || boost.CharacterID() != id {
		t.Fatalf("new boost = %v for %v", boost.State(), boost.CharacterID())
	}

	// A rest report on an available boost changes nothing.
	if r, ok := boost.RestFinished().(character.MountedBoostRestUnchanged); !ok || r.Boost().State() != character.MountedBoostAvailable {
		t.Errorf("rest on an available boost = %#v", boost.RestFinished())
	}

	spent, ok := boost.Spend().(character.MountedBoostSpendAccepted)
	if !ok {
		t.Fatalf("first spend = %#v", boost.Spend())
	}
	if spent.Boost().State() != character.MountedBoostSpent {
		t.Errorf("after spending the state is %v", spent.Boost().State())
	}
	var ev character.Event = spent.Event()
	if ev.CharacterID() != id {
		t.Errorf("spend event id = %v, want %v", ev.CharacterID(), id)
	}
	if boost.State() != character.MountedBoostAvailable {
		t.Error("Spend changed the boost it was called on")
	}

	// The second spend in a row is refused: the mounts need rest. No event, state kept.
	refused, ok := spent.Boost().Spend().(character.MountedBoostSpendRefused)
	if !ok || refused.Reason() != character.MountedBoostMountsNeedRest || refused.Boost().State() != character.MountedBoostSpent {
		t.Fatalf("second spend = %#v", spent.Boost().Spend())
	}

	// Short or Long, any finished rest restores, and the cycle repeats.
	restored, ok := spent.Boost().RestFinished().(character.MountedBoostRestRestored)
	if !ok || restored.Boost().State() != character.MountedBoostAvailable {
		t.Fatalf("rest on a spent boost = %#v", spent.Boost().RestFinished())
	}
	var rev character.Event = restored.Event()
	if rev.CharacterID() != id {
		t.Errorf("restore event id = %v", rev.CharacterID())
	}
	if _, again := restored.Boost().Spend().(character.MountedBoostSpendAccepted); !again {
		t.Error("a restored boost could not be spent again")
	}

	_, err = character.NewMountedPaceBoost(character.Identifier{})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero id error = %v, want ErrInvalidInput", err)
	}
}

// A dungeon crawl: a torch, a candle, a hooded lantern and a bullseye lantern
// each light their own area, and a space takes the brightest of them (SRD 5.2.1,
// "Light", p. 11; "Torch", p. 100; "Lantern, Hooded", p. 98).
// Requirement: doc/specs/exploration.md FR-026 to FR-035, FR-050 - bright within the radius (edge inclusive), dim beyond to the dim distance, darkness past it; a cone only inside the cone; a lowered hood is dim only; the six printed emissions; invalid emissions and spaces are errors; Strongest picks the brightest.
// Path: happy; unhappy (boundary, rule interaction, invalid input).
func TestGrpD3LightSourcesLightTheirOwnArea(t *testing.T) {
	t.Parallel()
	rules := character.LightSourceRules{}
	srd := character.SrdLightSources{}
	space := func(feet int, inCone bool) character.LightSpace {
		s, err := character.NewLightSpace(feet, inCone)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	torch := srd.Torch() // 20 ft bright, 20 ft dim
	for _, c := range []struct {
		feet int
		want content.LightLevel
	}{
		{0, content.LightBright}, {20, content.LightBright}, {21, content.LightDim}, {40, content.LightDim}, {41, content.LightDarkness},
	} {
		if got := rules.LevelAt(torch, space(c.feet, false)); got != c.want {
			t.Errorf("Torch at %d ft = %v, want %v", c.feet, got, c.want)
		}
	}
	// A radius ignores the cone report.
	if rules.LevelAt(torch, space(10, true)) != content.LightBright || rules.LevelAt(torch, space(10, false)) != content.LightBright {
		t.Error("a radius emission must ignore withinCone")
	}

	// The bullseye lantern lights a Cone only; outside it is dark at any distance.
	bullseye := srd.BullseyeLantern()
	if bullseye.Shape() != character.LightShapeCone || bullseye.BrightFeet() != 60 || bullseye.DimFeet() != 60 {
		t.Errorf("Bullseye = %v %d/%d, want a 60/60 cone", bullseye.Shape(), bullseye.BrightFeet(), bullseye.DimFeet())
	}
	if rules.LevelAt(bullseye, space(30, true)) != content.LightBright || rules.LevelAt(bullseye, space(90, true)) != content.LightDim ||
		rules.LevelAt(bullseye, space(30, false)) != content.LightDarkness || rules.LevelAt(bullseye, space(0, false)) != content.LightDarkness {
		t.Error("the Cone is lit only inside the Cone")
	}

	// A hooded lantern: raised 30/30, lowered Dim Light in 5 feet and no Bright Light at all.
	raised, lowered := srd.HoodedLantern(character.HoodRaised), srd.HoodedLantern(character.HoodLowered)
	if raised.BrightFeet() != 30 || raised.DimFeet() != 30 {
		t.Errorf("raised = %d/%d, want 30/30", raised.BrightFeet(), raised.DimFeet())
	}
	if lowered.BrightFeet() != 0 || rules.LevelAt(lowered, space(0, false)) != content.LightDim || rules.LevelAt(lowered, space(5, false)) != content.LightDim ||
		rules.LevelAt(lowered, space(6, false)) != content.LightDarkness {
		t.Error("a lowered hood must give Dim Light out to 5 feet and nothing else")
	}
	if !srd.HoodedLantern(0).IsZero() {
		t.Error("an invalid hood must give the zero emission")
	}

	// The six printed emissions and no others.
	all := srd.All()
	if len(all) != 6 || all[0] != srd.Candle() || all[1] != srd.Lamp() || all[5] != srd.Torch() {
		t.Fatalf("SrdLightSources.All = %v, want the six printed emissions in order", all)
	}
	if c, l := srd.Candle(), srd.Lamp(); c.BrightFeet() != 5 || c.DimFeet() != 5 || l.BrightFeet() != 15 || l.DimFeet() != 30 {
		t.Errorf("Candle %d/%d, Lamp %d/%d; want 5/5 and 15/30", c.BrightFeet(), c.DimFeet(), l.BrightFeet(), l.DimFeet())
	}

	// A space takes the brightest light it gets; no light at all is darkness.
	ambient := content.LightDim
	fromTorch := rules.LevelAt(torch, space(30, false)) // dim
	fromCandle := rules.LevelAt(srd.Candle(), space(3, false))
	if got := rules.Strongest([]content.LightLevel{ambient, fromTorch, fromCandle}); got != content.LightBright {
		t.Errorf("strongest of dim, dim and a candle's bright = %v, want bright", got)
	}
	if got := rules.Strongest([]content.LightLevel{content.LightDarkness, content.LightDim}); got != content.LightDim {
		t.Errorf("darkness and dim = %v, want dim", got)
	}
	if got := rules.Strongest(nil); got != content.LightDarkness {
		t.Errorf("no levels = %v, want darkness", got)
	}

	// A source that sheds no light, a negative radius and a negative distance are not accepted.
	for _, bad := range [][3]int{{0, 0, 0}, {1, -1, 5}, {1, 5, -1}} {
		_, err := character.NewLightEmission(character.LightShape(bad[0]), bad[1], bad[2])
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewLightEmission(%v) error = %v, want ErrInvalidInput", bad, err)
		}
	}
	if _, err := character.NewLightEmission(character.LightShapeRadius, 0, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an emission with neither bright nor dim light = %v, want ErrInvalidInput", err)
	}
	var ee *character.ExplorationError
	if _, err := character.NewLightSpace(-1, false); !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ee) || ee.Input != character.ExplorationLightDistance {
		t.Errorf("negative distance error = %v", err)
	}
	if _, err := character.NewLightEmission(character.LightShapeRadius, -5, 5); !errors.As(err, &ee) || ee.Input != character.ExplorationLightRadius {
		t.Errorf("negative radius error = %v", err)
	}
	if got := rules.LevelAt(character.LightEmission{}, space(0, true)); got != content.LightDarkness {
		t.Errorf("the zero emission = %v, want darkness", got)
	}
}

// A torch is lit, burns in pieces over an expedition and goes out: burn time
// adds up across calls, an exhausted supply raises one event, and a reloaded
// supply keeps what is left (SRD 5.2.1, "Torch", p. 100; "Oil", p. 99).
// Requirement: doc/specs/exploration.md FR-036 to FR-042, FR-052, FR-055 - capacities 1h, 1h, 6h; burns add up; equal or over exhausts with one event and never more than was held; zero keeps the supply; negative is invalid and checked first; a spent supply stays spent with no second event; Restore bounds.
// Path: happy; unhappy (boundary, invalid input, repeated action).
func TestGrpD3FuelBurnsInPiecesAndRunsOut(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("scout-1")
	for kind, want := range map[character.FuelKind]time.Duration{character.FuelCandle: time.Hour, character.FuelTorch: time.Hour, character.FuelOilFlask: 6 * time.Hour} {
		s, err := character.NewFreshFuelSupply(id, kind)
		if err != nil || s.Remaining() != want || s.Kind() != kind || s.CharacterID() != id || kind.Capacity() != want {
			t.Errorf("fresh supply of kind %v = %v, %v; want %v", kind, s.Remaining(), err, want)
		}
	}

	// An oil flask burned 2, 3 and 1 hours: gone on the third call, whenever it was lit.
	flask, _ := character.NewFreshFuelSupply(id, character.FuelOilFlask)
	for i, h := range []time.Duration{2, 3} {
		res, err := flask.Burn(h * time.Hour)
		cont, ok := res.(character.FuelBurnContinues)
		if err != nil || !ok {
			t.Fatalf("burn %d = %#v, %v", i+1, res, err)
		}
		flask = cont.Supply()
	}
	if flask.Remaining() != time.Hour {
		t.Fatalf("after 5 hours %v remains, want 1h", flask.Remaining())
	}
	last, err := flask.Burn(time.Hour)
	out, ok := last.(character.FuelBurnExhausted)
	if err != nil || !ok || out.BurnedFor() != time.Hour || out.Supply().Remaining() != 0 {
		t.Fatalf("the third burn = %#v, %v", last, err)
	}
	ev, raised := out.Event()
	var asEvent character.Event = ev
	if !raised || ev.Kind() != character.FuelOilFlask || asEvent.CharacterID() != id {
		t.Errorf("exhaustion event = %v raised %v", ev, raised)
	}

	// A candle burned for more than it holds burns only what it held.
	candle, _ := character.NewFreshFuelSupply(id, character.FuelCandle)
	over, err := candle.Burn(3 * time.Hour)
	if o, ok := over.(character.FuelBurnExhausted); err != nil || !ok || o.BurnedFor() != time.Hour {
		t.Errorf("a candle burned 3h = %#v, %v; want 1h burned", over, err)
	}
	// Zero changes nothing; negative is invalid whatever the supply holds.
	if z, err := candle.Burn(0); err != nil || z.(character.FuelBurnContinues).Supply().Remaining() != time.Hour {
		t.Errorf("zero burn = %#v, %v", z, err)
	}
	_, err = candle.Burn(-time.Second)
	var ee *character.ExplorationError
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ee) || ee.Input != character.ExplorationBurnDuration {
		t.Errorf("negative burn error = %v", err)
	}
	// A spent supply: nothing burns and no second event, even for a zero burn; a negative burn is still an error.
	spent := over.(character.FuelBurnExhausted).Supply()
	again, err := spent.Burn(time.Minute)
	if a, ok := again.(character.FuelBurnExhausted); err != nil || !ok || a.BurnedFor() != 0 {
		t.Errorf("burning a spent supply = %#v, %v", again, err)
	} else if _, raised := a.Event(); raised {
		t.Error("a spent supply raised a second event")
	}
	if zero, _ := spent.Burn(0); zero.(character.FuelBurnExhausted).BurnedFor() != 0 {
		t.Errorf("a zero burn on a spent supply = %#v", zero)
	}
	if _, err = spent.Burn(-time.Second); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative burn on a spent supply = %v", err)
	}

	// Reloading a part-burned supply: any amount up to the capacity, zero giving a spent supply.
	for _, c := range []struct {
		remaining time.Duration
		ok        bool
	}{{0, true}, {30 * time.Minute, true}, {time.Hour, true}, {time.Hour + time.Second, false}, {-time.Second, false}} {
		s, err := character.RestoreFuelSupply(id, character.FuelTorch, c.remaining)
		if c.ok != (err == nil) || (c.ok && s.Remaining() != c.remaining) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("RestoreFuelSupply(torch, %v) = %v, %v", c.remaining, s.Remaining(), err)
		}
	}
	if _, err := character.NewFreshFuelSupply(character.Identifier{}, character.FuelTorch); !errors.As(err, &ee) || ee.Input != character.ExplorationCharacterID {
		t.Errorf("zero id error = %v", err)
	}
	if _, err := character.NewFreshFuelSupply(id, 0); !errors.As(err, &ee) || ee.Input != character.ExplorationFuelKind {
		t.Errorf("invalid kind error = %v", err)
	}
}
