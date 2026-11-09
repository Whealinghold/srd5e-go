package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

func trigger(tb testing.TB, level int, circle bool, f character.DruidStateFields, tr character.DruidTrigger) character.ApplyDruidTriggerApplied {
	tb.Helper()
	r, err := character.ApplyDruidTrigger(character.ApplyDruidTriggerInput{State: druState(tb, level, circle, f), Tables: druTables(tb), Facts: druFacts(tb, level, circle), Trigger: tr})
	if err != nil {
		tb.Fatal(err)
	}
	return r
}

// Requirement: doc/specs/class-druid.md FR-153, FR-154, FR-156, FR-069, FR-090, FR-139, FR-160, table N (SRD 5.2.1, "Level 2: Wild Shape", p. 42; "Level 5: Wild Resurgence", p. 43; "Level 6: Natural Recovery", p. 46) -
// a finished Long Rest clears the four once-until-Long-Rest flags and the known form replacement count, keeps the once-per-turn flags, regains all Wild Shape uses (the maximum of the Druid level, not the total level),
// replaces the window by known form replacement (from level 2) and land type change (with the subclass from level 3), and dismisses a Wild Companion familiar with an event; a feature not granted opens nothing.
// Path: per case (happy / unhappy: boundary at levels 1, 2 and 3, rule interaction).
func TestApplyDruidTriggerLongRest(t *testing.T) {
	t.Parallel()
	long := character.DruidTriggerRestFinished{Kind: character.LongRest}
	full := character.DruidStateFields{WildShapeUsesRemaining: 0, KnownForms: wolfKnown, FormReplacements: 1, WildResurgenceUseForSlot: true, NaturalRecoveryCast: true, NaturalRecoveryRecovery: true,
		NatureMagician: true, WildResurgenceSlotForUseThisTurn: true, PrimalStrikeThisTurn: true, ElementalFury: character.ElementalFuryPrimalStrike, LandType: character.LandArid, FamiliarPresent: true,
		RestWindow: []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}}
	got := trigger(t, 20, true, full, long)
	s := got.State()
	ev, ok := got.Events()[0].(character.WildCompanionDismissed)
	if s.WildShapeUsesRemaining() != 4 || s.FormReplacements() != 0 || s.WildResurgenceUseForSlot() || s.NaturalRecoveryCast() || s.NaturalRecoveryRecovery() || s.NatureMagician() ||
		!s.WildResurgenceSlotForUseThisTurn() || !s.PrimalStrikeThisTurn() || s.FamiliarPresent() || len(got.Events()) != 1 || !ok || ev.CharacterID() != druTestID ||
		!slices.Equal(s.RestWindow(), []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowKnownFormReplacement}) ||
		s.LandType() != character.LandArid || s.ElementalFury() != character.ElementalFuryPrimalStrike || s.KnownForms() != druState(t, 20, true, full).KnownForms() {
		t.Errorf("Long Rest at 20: %+v events %#v", s, got.Events())
	}
	for _, c := range []struct {
		label  string
		level  int
		circle bool
		uses   int
		window []character.RestWindowPermission
	}{
		{"level 1 opens nothing", 1, true, 0, nil},
		{"level 2 opens known forms", 2, true, 2, []character.RestWindowPermission{character.RestWindowKnownFormReplacement}},
		{"level 3 with the subclass", 3, true, 2, []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowKnownFormReplacement}},
		{"level 3 without the subclass", 3, false, 2, []character.RestWindowPermission{character.RestWindowKnownFormReplacement}},
		{"level 6 uses are 3", 6, false, 3, []character.RestWindowPermission{character.RestWindowKnownFormReplacement}},
		{"Druid 3 / Fighter 5 keeps 2 uses", 3, true, 2, []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowKnownFormReplacement}},
	} {
		g := trigger(t, c.level, c.circle, character.DruidStateFields{}, long)
		if g.State().WildShapeUsesRemaining() != c.uses || !slices.Equal(g.State().RestWindow(), c.window) || len(g.Events()) != 0 {
			t.Errorf("%s: uses %d window %v events %#v", c.label, g.State().WildShapeUsesRemaining(), g.State().RestWindow(), g.Events())
		}
	}
	got.Events()[0] = nil
	if got.Events()[0] == nil {
		t.Error("Events shares storage")
	}
}

// Requirement: doc/specs/class-druid.md FR-068, FR-153, FR-154, FR-158, table N (SRD 5.2.1, "Level 2: Wild Shape", p. 42; "Level 6: Natural Recovery", p. 46) - a finished Short Rest regains exactly one expended Wild Shape use,
// never above the maximum and none when none are expended, clears no flag and no replacement count, replaces the window by natural recovery recovery (with the subclass from level 6), and a recovery used before stays used.
// Path: per case (happy / unhappy: boundary).
func TestApplyDruidTriggerShortRest(t *testing.T) {
	t.Parallel()
	short := character.DruidTriggerRestFinished{Kind: character.ShortRest}
	flags := character.DruidStateFields{KnownForms: wolfKnown, FormReplacements: 1, WildResurgenceUseForSlot: true, NaturalRecoveryCast: true, NaturalRecoveryRecovery: true, NatureMagician: true,
		RestWindow: []character.RestWindowPermission{character.RestWindowKnownFormReplacement}, LandType: character.LandPolar, FamiliarPresent: true}
	for _, c := range []struct {
		label  string
		level  int
		circle bool
		uses   int
		want   int
		window []character.RestWindowPermission
	}{
		{"regains one", 6, true, 0, 1, []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}},
		{"regains one of two", 6, true, 1, 2, []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}},
		{"none expended", 6, true, 3, 3, []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}},
		{"level 10 still opens it", 10, true, 0, 1, []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}},
		{"level 5 opens nothing", 5, true, 0, 1, nil},
		{"level 6 without the subclass", 6, false, 0, 1, nil},
	} {
		f := flags
		f.WildShapeUsesRemaining = c.uses
		f.NatureMagician = false
		f.WildResurgenceUseForSlot = c.level >= 5
		f.NaturalRecoveryCast, f.NaturalRecoveryRecovery = c.circle && c.level >= 6, c.circle && c.level >= 6
		if !c.circle {
			f.LandType = 0
		}
		g := trigger(t, c.level, c.circle, f, short)
		s := g.State()
		if s.WildShapeUsesRemaining() != c.want || !slices.Equal(s.RestWindow(), c.window) || s.FormReplacements() != 1 || s.WildResurgenceUseForSlot() != f.WildResurgenceUseForSlot ||
			s.NaturalRecoveryCast() != f.NaturalRecoveryCast || s.NaturalRecoveryRecovery() != f.NaturalRecoveryRecovery || !s.FamiliarPresent() || len(g.Events()) != 0 {
			t.Errorf("%s: %+v events %#v", c.label, s, g.Events())
		}
	}
	// a second recovery after a later Short Rest is rejected
	used := trigger(t, 6, true, character.DruidStateFields{NaturalRecoveryRecovery: true}, short).State()
	r, err := character.UseDruidFeature(character.UseDruidFeatureInput{State: used, Tables: druTables(t), Facts: druFacts(t, 6, true), Spellcasting: druSlots(t, 6, map[int]int{1: 1}),
		Use: character.DruidUseNaturalRecoveryRecover{Pool: character.PoolSpellcasting, Levels: []int{1}}})
	if rej, ok := r.(character.UseDruidFeatureRejected); err != nil || !ok || rej.Reason != character.DruidAlreadyUsedSinceLongRest {
		t.Errorf("second recovery: %#v, %v", r, err)
	}
	level1 := trigger(t, 1, false, character.DruidStateFields{}, short)
	if level1.State().WildShapeUsesRemaining() != 0 {
		t.Error("a Short Rest at level 1 gave a use")
	}
}

// Requirement: doc/specs/class-druid.md FR-154, FR-159, table N - the start of a turn clears exactly the two once-per-turn flags and closes the window, changing nothing else.
// Path: unhappy (rule interaction).
func TestApplyDruidTriggerTurnStarted(t *testing.T) {
	t.Parallel()
	f := character.DruidStateFields{WildShapeUsesRemaining: 1, KnownForms: wolfKnown, FormReplacements: 1, WildResurgenceSlotForUseThisTurn: true, PrimalStrikeThisTurn: true, ElementalFury: character.ElementalFuryPrimalStrike,
		WildResurgenceUseForSlot: true, NaturalRecoveryCast: true, NaturalRecoveryRecovery: true, LandType: character.LandPolar, FamiliarPresent: true, ActiveForm: "wolf",
		RestWindow: []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowKnownFormReplacement, character.RestWindowNaturalRecoveryRecovery}}
	before := druState(t, 9, true, f)
	g := trigger(t, 9, true, f, character.DruidTriggerTurnStarted{})
	want := f
	want.WildResurgenceSlotForUseThisTurn, want.PrimalStrikeThisTurn, want.RestWindow = false, false, nil
	if g.State() != druState(t, 9, true, want) || len(g.Events()) != 0 {
		t.Errorf("turn started: %+v events %#v", g.State(), g.Events())
	}
	if before != druState(t, 9, true, f) {
		t.Error("the input changed")
	}
}

// Requirement: doc/specs/class-druid.md FR-112, FR-116, FR-154 and table N (SRD 5.2.1, "Level 20: Archdruid" (Evergreen Wild Shape), p. 43) - rolling Initiative closes the window and, at level 20 with no Wild Shape use
// left, regains one use with WildShapeUsesChanged (0 to 1, Evergreen Wild Shape); at level 19, or with a use remaining, only the window changes and there is no event.
// Path: per case (happy / unhappy: boundary at 19 and 20, rule interaction).
func TestApplyDruidTriggerInitiativeRolled(t *testing.T) {
	t.Parallel()
	window := []character.RestWindowPermission{character.RestWindowKnownFormReplacement}
	for _, c := range []struct {
		label string
		level int
		uses  int
		want  int
		event bool
	}{{"level 19, none left", 19, 0, 0, false}, {"level 20, none left", 20, 0, 1, true}, {"level 20, one left", 20, 1, 1, false}, {"level 20, all left", 20, 4, 4, false}} {
		g := trigger(t, c.level, false, character.DruidStateFields{WildShapeUsesRemaining: c.uses, RestWindow: window, KnownForms: wolfKnown}, character.DruidTriggerInitiativeRolled{})
		if g.State().WildShapeUsesRemaining() != c.want || len(g.State().RestWindow()) != 0 || (len(g.Events()) == 1) != c.event {
			t.Errorf("%s: uses %d window %v events %#v", c.label, g.State().WildShapeUsesRemaining(), g.State().RestWindow(), g.Events())
		}
		if c.event {
			ev, ok := g.Events()[0].(character.WildShapeUsesChanged)
			if !ok || ev.From() != 0 || ev.To() != 1 || ev.Cause() != character.WildShapeUsesEvergreenWildShape || ev.CharacterID() != druTestID {
				t.Errorf("%s: event %#v", c.label, g.Events()[0])
			}
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-141, FR-160, FR-164 - ApplyDruidTrigger returns an error matching ErrInvalidInput for zero facts or tables, a state that breaks an invariant, a nil trigger and a rest kind
// that is neither short nor long; it takes and returns no spell slot.
// Path: per case (unhappy: invalid input).
func TestApplyDruidTriggerInvalidInput(t *testing.T) {
	t.Parallel()
	ok := character.ApplyDruidTriggerInput{State: character.DruidFeatureState{}, Tables: druTables(t), Facts: druFacts(t, 5, false), Trigger: character.DruidTriggerTurnStarted{}}
	for _, c := range []struct {
		label  string
		mutate func(*character.ApplyDruidTriggerInput)
	}{
		{"zero facts", func(in *character.ApplyDruidTriggerInput) { in.Facts = character.DruidCharacterFacts{} }},
		{"zero tables", func(in *character.ApplyDruidTriggerInput) { in.Tables = character.DruidFeatureTables{} }},
		{"state breaks an invariant", func(in *character.ApplyDruidTriggerInput) {
			in.State = druState(t, 5, true, character.DruidStateFields{LandType: character.LandArid})
		}},
		{"nil trigger", func(in *character.ApplyDruidTriggerInput) { in.Trigger = nil }},
		{"rest kind 0", func(in *character.ApplyDruidTriggerInput) { in.Trigger = character.DruidTriggerRestFinished{} }},
		{"rest kind 3", func(in *character.ApplyDruidTriggerInput) { in.Trigger = character.DruidTriggerRestFinished{Kind: 3} }},
	} {
		in := ok
		c.mutate(&in)
		if _, err := character.ApplyDruidTrigger(in); err == nil {
			t.Errorf("%s: no error", c.label)
		} else {
			wantInvalid(t, err, c.label)
		}
	}
}
