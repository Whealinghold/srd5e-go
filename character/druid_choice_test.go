package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func chooseInput(tb testing.TB, level int, circle bool, f character.DruidStateFields, choice character.DruidChoice) character.ApplyDruidChoiceInput {
	tb.Helper()
	return character.ApplyDruidChoiceInput{State: druState(tb, level, circle, f), Tables: druTables(tb), Facts: druFacts(tb, level, circle), Choice: choice}
}

// choose applies the choice and returns the applied result, or the rejection when rejected.
func choose(tb testing.TB, in character.ApplyDruidChoiceInput) (character.ApplyDruidChoiceApplied, character.DruidRejection) {
	tb.Helper()
	r, err := character.ApplyDruidChoice(in)
	if err != nil {
		tb.Fatal(err)
	}
	switch got := r.(type) {
	case character.ApplyDruidChoiceApplied:
		return got, 0
	case character.ApplyDruidChoiceRejected:
		return character.ApplyDruidChoiceApplied{}, got.Reason
	}
	tb.Fatalf("result %#v", r)
	return character.ApplyDruidChoiceApplied{}, 0
}

// Requirement: doc/specs/class-druid.md FR-048, FR-049, FR-056 - the Primal Order is chosen once, as the starting class or a multiclass: the first choice is applied with a
// PrimalOrderChosen event, a second is PrimalOrderAlreadyChosen and the state is kept.
// Path: per case (happy / unhappy: state error).
func TestApplyDruidChoicePrimalOrder(t *testing.T) {
	t.Parallel()
	a, reason := choose(t, chooseInput(t, 1, false, character.DruidStateFields{}, character.DruidChoicePrimalOrder{Order: character.PrimalOrderWarden}))
	ev, ok := a.Events()[0].(character.PrimalOrderChosen)
	if reason != 0 || a.State().PrimalOrder() != character.PrimalOrderWarden || len(a.Events()) != 1 || !ok || ev.Order() != character.PrimalOrderWarden || ev.CharacterID() != druTestID {
		t.Errorf("first choice: %v, %+v, %#v", reason, a.State(), a.Events())
	}
	for _, level := range []int{1, 20} {
		_, reason = choose(t, chooseInput(t, level, false, character.DruidStateFields{PrimalOrder: character.PrimalOrderMagician}, character.DruidChoicePrimalOrder{Order: character.PrimalOrderWarden}))
		if reason != character.DruidPrimalOrderAlreadyChosen {
			t.Errorf("second choice at %d: %v, want PrimalOrderAlreadyChosen", level, reason)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-104 (SRD 5.2.1, "Level 7: Elemental Fury", p. 43) - Elemental Fury is a choice from level 7: below it FeatureNotGranted, a second choice
// ElementalFuryAlreadyChosen, and when both hold the first of the order.
// Path: per case (unhappy: boundary, state error, rule interaction).
func TestApplyDruidChoiceElementalFury(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label string
		level int
		held  character.ElementalFury
		want  character.DruidRejection
	}{
		{"level 6", 6, 0, character.DruidFeatureNotGranted}, {"level 7", 7, 0, 0}, {"level 20", 20, 0, 0},
		{"chosen", 7, character.ElementalFuryPotentSpellcasting, character.DruidElementalFuryAlreadyChosen},
	} {
		a, reason := choose(t, chooseInput(t, c.level, false, character.DruidStateFields{ElementalFury: c.held}, character.DruidChoiceElementalFury{Option: character.ElementalFuryPrimalStrike}))
		if reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
		if reason == 0 {
			if ev, ok := a.Events()[0].(character.ElementalFuryChosen); a.State().ElementalFury() != character.ElementalFuryPrimalStrike || !ok || ev.Option() != character.ElementalFuryPrimalStrike {
				t.Errorf("%s: state %+v events %#v", c.label, a.State(), a.Events())
			}
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-118, FR-121, FR-154, FR-157, FR-138 (SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46) - the land type is a choice of the Circle of the Land
// from level 3: the first choice is always open, a later one needs the window a Long Rest opens (ChangeNotAllowedNow otherwise), a choice closes the window and emits LandTypeChosen, and
// below level 3 or without the subclass it is FeatureNotGranted.
// Path: per case (happy / unhappy: boundary, state error).
func TestApplyDruidChoiceLandType(t *testing.T) {
	t.Parallel()
	open := []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowKnownFormReplacement}
	for _, c := range []struct {
		label  string
		level  int
		circle bool
		f      character.DruidStateFields
		want   character.DruidRejection
	}{
		{"first choice, no window", 3, true, character.DruidStateFields{}, 0},
		{"level 2", 2, true, character.DruidStateFields{}, character.DruidFeatureNotGranted},
		{"no subclass", 3, false, character.DruidStateFields{}, character.DruidFeatureNotGranted},
		{"change after a Long Rest", 5, true, character.DruidStateFields{LandType: character.LandArid, RestWindow: open}, 0},
		{"change with the window closed", 5, true, character.DruidStateFields{LandType: character.LandArid}, character.DruidChangeNotAllowedNow},
		{"change with another window open", 6, true, character.DruidStateFields{LandType: character.LandArid, RestWindow: []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}}, character.DruidChangeNotAllowedNow},
	} {
		a, reason := choose(t, chooseInput(t, c.level, c.circle, c.f, character.DruidChoiceLandType{Land: character.LandPolar}))
		if reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
			continue
		}
		if reason == 0 {
			ev, ok := a.Events()[0].(character.LandTypeChosen)
			wantWindow := slices.DeleteFunc(slices.Clone(c.f.RestWindow), func(p character.RestWindowPermission) bool { return p == character.RestWindowLandTypeChange })
			if a.State().LandType() != character.LandPolar || !ok || ev.Land() != character.LandPolar || !slices.Equal(a.State().RestWindow(), wantWindow) {
				t.Errorf("%s: state %+v events %#v", c.label, a.State(), a.Events())
			}
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-138 - the land type is never cached: land arid chosen after Long Rest A and polar after Long Rest B give the polar Circle Spells and the Cold
// Resistance, with no arid spell or Fire.
// Path: unhappy (rule interaction).
func TestLandTypeIsReadFromTheCurrentState(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	state := druState(t, 9, true, character.DruidStateFields{})
	apply := func(s character.DruidFeatureState, land character.LandType) character.DruidFeatureState {
		rest, err := character.ApplyDruidTrigger(character.ApplyDruidTriggerInput{State: s, Tables: tables, Facts: druFacts(t, 9, true), Trigger: character.DruidTriggerRestFinished{Kind: character.LongRest}})
		if err != nil {
			t.Fatal(err)
		}
		a, reason := choose(t, character.ApplyDruidChoiceInput{State: rest.State(), Tables: tables, Facts: druFacts(t, 9, true), Choice: character.DruidChoiceLandType{Land: land}})
		if reason != 0 {
			t.Fatalf("land %v: %v", land, reason)
		}
		return a.State()
	}
	state = apply(state, character.LandArid)
	state = apply(state, character.LandPolar)
	spells := druRules.CircleSpells(tables, state.LandType(), character.MustDruidLevel(9))
	if state.LandType() != character.LandPolar || slices.Contains(spells, "fireball") || !slices.Contains(spells, "cone_of_cold") {
		t.Errorf("land %v, Circle Spells %v", state.LandType(), spells)
	}
	if r, _ := tables.Resistance(state.LandType()); r != content.DamageCold {
		t.Errorf("Resistance = %v, want Cold", r)
	}
}

// Requirement: doc/specs/class-druid.md FR-072 to FR-075, FR-078, D-5 (SRD 5.2.1, "Level 2: Wild Shape", p. 42) - known forms are chosen from level 2: each must be an eligible Beast
// (NotABeast, ChallengeRatingTooHigh, FlySpeedNotAllowed), distinct from the others and from the known forms (DuplicateForm), and no more than the level allows in all (TooManyKnownForms);
// fewer than allowed is accepted, an empty list changes nothing, and known forms stay valid when the level rises.
// Path: per case (happy / unhappy: boundary, invalid input, rule interaction).
func TestApplyDruidChoiceKnownForms(t *testing.T) {
	t.Parallel()
	wolf, rat, spider, horse := druForm("wolf", "1/4", false), druForm("rat", "0", false), druForm("spider", "1/4", false), druForm("riding_horse", "1/4", false)
	bear := druForm("bear", "1/2", false)
	owl := druForm("owl", "1/4", true)
	var nineForms []character.BeastFormFacts
	for _, id := range []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9"} {
		nineForms = append(nineForms, druForm(id, "0", false))
	}
	dragon := must(character.NewBeastFormFacts("dragon", content.CreatureTypeDragon, content.MustChallengeRating("0"), false))
	for _, c := range []struct {
		label string
		level int
		held  []string
		forms []character.BeastFormFacts
		want  character.DruidRejection
		after int
	}{
		{"four at level 2", 2, nil, []character.BeastFormFacts{wolf, rat, spider, horse}, 0, 4},
		{"two of four, the rest pending", 2, nil, []character.BeastFormFacts{wolf, rat}, 0, 2},
		{"empty list", 2, nil, nil, 0, 0},
		{"a fifth", 2, []string{"a", "b", "c", "d"}, []character.BeastFormFacts{wolf}, character.DruidTooManyKnownForms, 0},
		{"five in one list", 2, nil, []character.BeastFormFacts{wolf, rat, spider, horse, druForm("fox", "0", false)}, character.DruidTooManyKnownForms, 0},
		{"two more at level 4", 4, []string{"a", "b", "c", "d"}, []character.BeastFormFacts{bear, wolf}, 0, 6},
		{"a seventh at level 4", 4, []string{"a", "b", "c", "d", "e", "f"}, []character.BeastFormFacts{wolf}, character.DruidTooManyKnownForms, 0},
		{"eight at level 8", 8, []string{"a", "b", "c", "d", "e", "f"}, []character.BeastFormFacts{wolf, owl}, 0, 8},
		{"nine distinct forms, none stored", 20, nil, nineForms, character.DruidTooManyKnownForms, 0},
		{"not in the list twice", 3, nil, []character.BeastFormFacts{wolf, wolf}, character.DruidDuplicateForm, 0},
		{"already known", 3, []string{"wolf"}, []character.BeastFormFacts{wolf}, character.DruidDuplicateForm, 0},
		{"level 1", 1, nil, []character.BeastFormFacts{wolf}, character.DruidFeatureNotGranted, 0},
		{"not a Beast", 8, nil, []character.BeastFormFacts{dragon}, character.DruidNotABeast, 0},
		{"too high", 3, nil, []character.BeastFormFacts{bear}, character.DruidChallengeRatingTooHigh, 0},
		{"flying at 7", 7, nil, []character.BeastFormFacts{owl}, character.DruidFlySpeedNotAllowed, 0},
		{"too many and ineligible", 2, []string{"a", "b", "c", "d"}, []character.BeastFormFacts{bear}, character.DruidChallengeRatingTooHigh, 0},
		{"too many and duplicate", 2, []string{"wolf", "b", "c", "d"}, []character.BeastFormFacts{wolf}, character.DruidDuplicateForm, 0},
	} {
		a, reason := choose(t, chooseInput(t, c.level, false, character.DruidStateFields{KnownForms: druIDs(c.held...)}, character.DruidChoiceKnownForms{Forms: c.forms}))
		if reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
			continue
		}
		if reason == 0 && (a.State().KnownForms().Len() != len(c.held)+len(c.forms) || len(c.forms) > 0 && len(a.Events()) != 1 || len(c.forms) == 0 && len(a.Events()) != 0) {
			t.Errorf("%s: known forms %v events %#v", c.label, a.State().KnownForms().IDs(), a.Events())
		}
	}
	a, _ := choose(t, chooseInput(t, 3, false, character.DruidStateFields{}, character.DruidChoiceKnownForms{Forms: []character.BeastFormFacts{wolf, rat}}))
	ev, ok := a.Events()[0].(character.KnownFormsChosen)
	if !ok || !slices.Equal(ev.Forms(), druIDs("wolf", "rat")) || !slices.Equal(a.State().KnownForms().IDs(), druIDs("wolf", "rat")) {
		t.Errorf("event = %#v", a.Events())
	}
	ev.Forms()[0] = "changed"
	if ev.Forms()[0] != "wolf" {
		t.Error("KnownFormsChosen shares storage")
	}
	// the forms stay valid when the level rises
	grown := gained(t, gainInput(t, 3, 4, a.State()))
	if grown.State().KnownForms() != a.State().KnownForms() {
		t.Error("a level gain changed the known forms")
	}
}

// Requirement: doc/specs/class-druid.md FR-076, FR-077, D-5 and table B (SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42) - a known form is replaced one at a time while the window a
// Long Rest opens is open, at most once per Long Rest; the rejection is the first of: ChangeNotAllowedNow, FormNotKnown, the eligibility reasons, DuplicateForm, TooManyReplacements; the
// window stays open after a replacement.
// Path: per case (happy / unhappy: state error, rule interaction).
func TestApplyDruidChoiceReplaceForm(t *testing.T) {
	t.Parallel()
	window := []character.RestWindowPermission{character.RestWindowKnownFormReplacement}
	held := []string{"wolf", "rat", "spider", "riding_horse", "a", "b", "c", "d"}
	cr1, cr2, known1 := druForm("badger", "1", false), druForm("bear", "2", false), druForm("rat", "1", false)
	for _, c := range []struct {
		label   string
		level   int
		window  []character.RestWindowPermission
		swaps   int
		replace string
		with    character.BeastFormFacts
		want    character.DruidRejection
	}{
		{"replace Wolf", 8, window, 0, "wolf", cr1, 0},
		{"closed and unknown", 8, nil, 0, "bear", cr1, character.DruidChangeNotAllowedNow},
		{"closed", 8, nil, 0, "wolf", cr1, character.DruidChangeNotAllowedNow},
		{"unknown and ineligible", 8, window, 0, "bear", cr2, character.DruidFormNotKnown},
		{"ineligible and duplicate", 8, window, 0, "wolf", must(character.NewBeastFormFacts("rat", content.CreatureTypeBeast, content.MustChallengeRating("2"), false)), character.DruidChallengeRatingTooHigh},
		{"duplicate", 8, window, 0, "wolf", known1, character.DruidDuplicateForm},
		{"replace by itself", 8, window, 0, "wolf", must(character.NewBeastFormFacts("wolf", content.CreatureTypeBeast, content.MustChallengeRating("1/4"), false)), character.DruidDuplicateForm},
		{"second replacement and ineligible", 8, window, 1, "wolf", cr2, character.DruidChallengeRatingTooHigh},
		{"second replacement", 8, window, 1, "wolf", cr1, character.DruidTooManyReplacements},
		{"not a Beast", 8, window, 0, "wolf", must(character.NewBeastFormFacts("x", content.CreatureTypeDragon, content.MustChallengeRating("0"), false)), character.DruidNotABeast},
		{"flying below 8", 7, window, 0, "wolf", druForm("owl", "1/2", true), character.DruidFlySpeedNotAllowed},
		{"all known at level 7", 7, window, 0, "wolf", druForm("owl", "1/4", true), character.DruidFlySpeedNotAllowed},
	} {
		forms := held
		if c.level < 8 {
			forms = held[:6]
		}
		a, reason := choose(t, chooseInput(t, c.level, false, character.DruidStateFields{KnownForms: druIDs(forms...), RestWindow: c.window, FormReplacements: c.swaps}, character.DruidChoiceReplaceForm{Replace: content.MonsterID(c.replace), With: c.with}))
		if reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
			continue
		}
		if reason == 0 {
			ev, ok := a.Events()[0].(character.KnownFormReplaced)
			if a.State().KnownForms().Contains("wolf") || !a.State().KnownForms().Contains(c.with.ID()) || a.State().KnownForms().Len() != len(forms) || a.State().FormReplacements() != 1 ||
				!slices.Equal(a.State().RestWindow(), window) || !ok || ev.Replaced() != "wolf" || ev.Replacing() != c.with.ID() {
				t.Errorf("%s: state %+v events %#v", c.label, a.State(), a.Events())
			}
			if idx := slices.Index(a.State().KnownForms().IDs(), c.with.ID()); idx != 0 {
				t.Errorf("%s: the new form is at %d, want the place of the old one", c.label, idx)
			}
		}
	}
	_, reason := choose(t, chooseInput(t, 1, false, character.DruidStateFields{}, character.DruidChoiceReplaceForm{Replace: "wolf", With: cr1}))
	if reason != character.DruidFeatureNotGranted {
		t.Errorf("level 1: %v, want FeatureNotGranted", reason)
	}
}

// Requirement: doc/specs/class-druid.md FR-141, FR-164, FR-148 - ApplyDruidChoice returns an error matching ErrInvalidInput for zero facts or tables, a state that breaks an invariant, a nil choice and
// a value outside its set; a rejected choice leaves the state as it was and the input state is never modified.
// Path: per case (unhappy: invalid input, immutability).
func TestApplyDruidChoiceInvalidInput(t *testing.T) {
	t.Parallel()
	ok := chooseInput(t, 7, true, character.DruidStateFields{}, character.DruidChoicePrimalOrder{Order: character.PrimalOrderWarden})
	for _, c := range []struct {
		label  string
		mutate func(*character.ApplyDruidChoiceInput)
	}{
		{"zero facts", func(in *character.ApplyDruidChoiceInput) { in.Facts = character.DruidCharacterFacts{} }},
		{"zero tables", func(in *character.ApplyDruidChoiceInput) { in.Tables = character.DruidFeatureTables{} }},
		{"state breaks an invariant", func(in *character.ApplyDruidChoiceInput) {
			in.Facts = druFacts(t, 1, false)
			in.State = druState(t, 7, true, character.DruidStateFields{WildShapeUsesRemaining: 3})
		}},
		{"nil choice", func(in *character.ApplyDruidChoiceInput) { in.Choice = nil }},
		{"order 0", func(in *character.ApplyDruidChoiceInput) { in.Choice = character.DruidChoicePrimalOrder{} }},
		{"fury 0", func(in *character.ApplyDruidChoiceInput) { in.Choice = character.DruidChoiceElementalFury{} }},
		{"land 0", func(in *character.ApplyDruidChoiceInput) { in.Choice = character.DruidChoiceLandType{} }},
	} {
		in := ok
		c.mutate(&in)
		if _, err := character.ApplyDruidChoice(in); err == nil {
			t.Errorf("%s: no error", c.label)
		} else {
			wantInvalid(t, err, c.label)
		}
	}
	in := chooseInput(t, 7, true, character.DruidStateFields{PrimalOrder: character.PrimalOrderMagician}, character.DruidChoicePrimalOrder{Order: character.PrimalOrderWarden})
	before := in.State
	r, err := character.ApplyDruidChoice(in)
	if _, rejected := r.(character.ApplyDruidChoiceRejected); !rejected || err != nil || in.State != before {
		t.Errorf("rejected choice: %#v, %v", r, err)
	}
	applied, _ := choose(t, ok)
	applied.Events()[0] = nil
	if applied.Events()[0] == nil {
		t.Error("Events shares storage")
	}
}
