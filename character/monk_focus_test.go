package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/class-monk.md Table 4.4b, FR-018, FR-037 - the Monk's Focus save DC is 8 plus the Wisdom modifier plus the
// Proficiency Bonus of the total level; the multiclass row (Monk 5, Fighter 5, total level 10, Wisdom +3) is 15, not 14.
// Path: per case (rows of Table 4.4b).
func TestMonkSaveDC(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		wis, prof int
		dc        d20.DifficultyClass
	}{
		{"typical", 3, 2, 13}, {"zero modifier", 0, 2, 10}, {"negative modifier", -1, 2, 9}, {"mid level", 5, 4, 17}, {"top level", 5, 6, 19},
	} {
		got := character.MonkSaveDC(modifierOf(t, c.wis), character.ProficiencyBonusFor(character.MustLevel(map[int]int{2: 1, 4: 9, 6: 17}[c.prof])))
		if got != c.dc {
			t.Errorf("%s: DC = %d, want %d", c.name, got, c.dc)
		}
	}
	multi := mnkChar(t, cl("monk", 5), cl("fighter", 5))
	if got := character.MonkAbilitiesOf(multi).SaveDC(); got != 15 {
		t.Errorf("Monk 5 / Fighter 5 with Wisdom +3: DC = %d, want 15", got)
	}
	a := character.MonkAbilitiesOf(multi)
	if a.Dexterity.Value() != 3 || a.Wisdom.Value() != 3 || a.Proficiency.Value() != 4 {
		t.Errorf("MonkAbilitiesOf = %+v, want +3, +3, +4", a)
	}
}

func mnkUse(f character.MonkFocusFeature, spend bool) character.MonkFocusUse {
	return character.MonkFocusUse{Feature: f, SpendPoint: spend}
}

// Requirement: doc/specs/class-monk.md FR-038 to FR-043, FR-135 - Availability of the Focus features: below level 2 levelTooLow whatever the
// request, Flurry of Blows without the point and an unknown feature are notEligible, with no point available notEnoughFocusPoints (only when the
// point is spent), and the free Patient Defense and Step of the Wind need no point.
// Path: per case (unhappy: boundary, state error).
func TestMonkFocusAvailability(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		level int
		state character.MonkState
		use   character.MonkFocusUse
		want  character.MonkReason
	}{
		{"flurry at level 2", 2, mnkState(0, false, 0), mnkUse(character.MonkFlurryOfBlows, true), 0},
		{"flurry at level 1", 1, mnkState(0, false, 0), mnkUse(character.MonkFlurryOfBlows, true), character.MonkLevelTooLow},
		{"free patient defense at level 1", 1, mnkState(0, false, 0), mnkUse(character.MonkPatientDefense, false), character.MonkLevelTooLow},
		{"flurry without the point", 2, mnkState(0, false, 0), mnkUse(character.MonkFlurryOfBlows, false), character.MonkNotEligible},
		{"unknown feature", 2, mnkState(0, false, 0), mnkUse(0, true), character.MonkNotEligible},
		{"flurry with no point available", 2, mnkState(2, false, 0), mnkUse(character.MonkFlurryOfBlows, true), character.MonkNotEnoughFocusPoints},
		{"patient defense free with no point", 2, mnkState(2, false, 0), mnkUse(character.MonkPatientDefense, false), 0},
		{"step of the wind paid with no point", 2, mnkState(2, false, 0), mnkUse(character.MonkStepOfTheWind, true), character.MonkNotEnoughFocusPoints},
		{"step of the wind free with no point", 2, mnkState(2, false, 0), mnkUse(character.MonkStepOfTheWind, false), 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.MonkFocus{}.Availability(mnkStats(t, c.level), c.state, c.use)
			if c.want == 0 {
				if _, ok := got.(character.MonkAvailable); !ok {
					t.Errorf("Availability = %#v, want available", got)
				}
				return
			}
			if un, ok := got.(character.MonkUnavailable); !ok || un.Reason != c.want {
				t.Errorf("Availability = %#v, want unavailable %v", got, c.want)
			}
		})
	}
	if un, ok := (character.MonkFocus{}).Availability(character.MonkLevelStats{}, mnkState(0, false, 0), mnkUse(character.MonkFlurryOfBlows, true)).(character.MonkUnavailable); !ok || un.Reason != character.MonkNotAMonk {
		t.Errorf("zero stats = %#v, want notAMonk", un)
	}
}

// Requirement: doc/specs/class-monk.md FR-038 to FR-042, FR-087 to FR-090, FR-089 - Resolve: Flurry of Blows costs 1 point and gives two Unarmed Strikes
// (three from level 10); Patient Defense gives Disengage free or Disengage and Dodge for the point, with two Martial Arts dice of Temporary Hit Points
// from level 10 only when paid; Step of the Wind gives Dash free or Disengage and Dash and a doubled jump for the point, and from level 10 carries a
// willing creature of Large or smaller within 5 feet; every other carried creature is notEligible.
// Path: per case (happy, unhappy: boundary, rule interaction).
func TestMonkFocusResolve(t *testing.T) {
	t.Parallel()
	large := character.MonkCarriedCreature{Chosen: true, Willing: true, Size: content.SizeLarge, DistanceFeet: 5}
	for _, c := range []struct {
		name       string
		level      int
		use        character.MonkFocusUse
		thp        int
		strikes    int
		spent      int
		actions    []character.MonkAction
		jump, carr bool
	}{
		{"flurry 9", 9, mnkUse(character.MonkFlurryOfBlows, true), 0, 2, 1, nil, false, false},
		{"flurry 10", 10, mnkUse(character.MonkFlurryOfBlows, true), 0, 3, 1, nil, false, false},
		{"free patient defense", 10, mnkUse(character.MonkPatientDefense, false), 0, 0, 0, []character.MonkAction{character.MonkDisengage}, false, false},
		{"paid patient defense 9", 9, mnkUse(character.MonkPatientDefense, true), 0, 0, 1, []character.MonkAction{character.MonkDisengage, character.MonkDodge}, false, false},
		{"paid patient defense 10", 10, mnkUse(character.MonkPatientDefense, true), 9, 0, 1, []character.MonkAction{character.MonkDisengage, character.MonkDodge}, false, false},
		{"free step of the wind", 2, mnkUse(character.MonkStepOfTheWind, false), 0, 0, 0, []character.MonkAction{character.MonkDash}, false, false},
		{"paid step of the wind", 9, mnkUse(character.MonkStepOfTheWind, true), 0, 0, 1, []character.MonkAction{character.MonkDisengage, character.MonkDash}, true, false},
		{"step of the wind carrying", 10, character.MonkFocusUse{Feature: character.MonkStepOfTheWind, SpendPoint: true, Carried: large}, 0, 0, 1, []character.MonkAction{character.MonkDisengage, character.MonkDash}, true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := mnkStats(t, c.level)
			in := mnkState(0, false, 0)
			res, err := character.MonkFocus{}.Resolve(stats, in, c.use, c.thp)
			if err != nil {
				t.Fatal(err)
			}
			if res.Feature() != c.use.Feature || res.UnarmedStrikes() != c.strikes || res.FocusPointsSpent() != c.spent || res.JumpDoubled() != c.jump || res.CarriesCreature() != c.carr ||
				res.TemporaryHitPoints() != c.thp || !slices.Equal(res.Actions(), c.actions) || res.State().FocusExpended() != c.spent {
				t.Errorf("Resolve = %+v", res)
			}
			if in != mnkState(0, false, 0) {
				t.Error("Resolve altered its input")
			}
			if len(res.Actions()) > 0 {
				res.Actions()[0] = 0
				if res.Actions()[0] == 0 {
					t.Error("Actions returned a shared slice")
				}
			}
		})
	}
	// Temporary Hit Points dice and carried creature rules.
	stats10 := mnkStats(t, 10)
	pd := mnkUse(character.MonkPatientDefense, true)
	if n := (character.MonkFocus{}).TemporaryHitPointDice(stats10, pd); n != 2 {
		t.Errorf("dice at level 10 = %d, want 2", n)
	}
	for _, u := range []character.MonkFocusUse{mnkUse(character.MonkPatientDefense, false), mnkUse(character.MonkFlurryOfBlows, true)} {
		if n := (character.MonkFocus{}).TemporaryHitPointDice(stats10, u); n != 0 {
			t.Errorf("dice for %+v = %d, want 0", u, n)
		}
	}
	if n := (character.MonkFocus{}).TemporaryHitPointDice(mnkStats(t, 9), pd); n != 0 {
		t.Errorf("dice at level 9 = %d, want 0", n)
	}
	for _, thp := range []int{0, 1, 17} { // two d8 sum between 2 and 16
		_, err := character.MonkFocus{}.Resolve(stats10, mnkState(0, false, 0), pd, thp)
		wantInvalid(t, err, "Temporary Hit Points roll")
	}
	_, err := character.MonkFocus{}.Resolve(stats10, mnkState(0, false, 0), mnkUse(character.MonkFlurryOfBlows, true), 4)
	wantInvalid(t, err, "a roll when no die is rolled")
	_, err = character.MonkFocus{}.Resolve(stats10, mnkState(10, false, 0), pd, 9)
	wantInvalid(t, err, "unavailable")
	step := func(c character.MonkCarriedCreature, spend bool) character.MonkFocusUse {
		return character.MonkFocusUse{Feature: character.MonkStepOfTheWind, SpendPoint: spend, Carried: c}
	}
	for _, c := range []struct {
		name  string
		level int
		use   character.MonkFocusUse
	}{
		{"unwilling", 10, step(character.MonkCarriedCreature{Chosen: true, Willing: false, Size: content.SizeMedium, DistanceFeet: 5}, true)},
		{"Huge", 10, step(character.MonkCarriedCreature{Chosen: true, Willing: true, Size: content.SizeHuge, DistanceFeet: 5}, true)},
		{"6 feet away", 10, step(character.MonkCarriedCreature{Chosen: true, Willing: true, Size: content.SizeMedium, DistanceFeet: 6}, true)},
		{"an invalid size", 10, step(character.MonkCarriedCreature{Chosen: true, Willing: true, DistanceFeet: 5}, true)},
		{"level 9", 9, step(large, true)},
		{"without the point", 10, step(large, false)},
		{"with Patient Defense", 10, character.MonkFocusUse{Feature: character.MonkPatientDefense, SpendPoint: true, Carried: large}},
	} {
		got := character.MonkFocus{}.Availability(mnkStats(t, c.level), mnkState(0, false, 0), c.use)
		if un, ok := got.(character.MonkUnavailable); !ok || un.Reason != character.MonkNotEligible {
			t.Errorf("carrying: %s = %#v, want notEligible", c.name, got)
		}
	}
	for _, size := range []content.CreatureSize{content.SizeTiny, content.SizeSmall, content.SizeMedium, content.SizeLarge} {
		cc := character.MonkCarriedCreature{Chosen: true, Willing: true, Size: size, DistanceFeet: 0}
		if _, ok := (character.MonkFocus{}).Availability(stats10, mnkState(0, false, 0), step(cc, true)).(character.MonkAvailable); !ok {
			t.Errorf("a willing %v within 5 feet must be carried", size)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-147 - the Focus feature names are the srd5e.feature values, snake_case, and an undefined value is unknown and not valid.
// Path: per case.
func TestMonkFocusFeatureNames(t *testing.T) {
	t.Parallel()
	for f, want := range map[character.MonkFocusFeature]string{
		character.MonkFlurryOfBlows: "flurry_of_blows", character.MonkPatientDefense: "patient_defense", character.MonkStepOfTheWind: "step_of_the_wind", 0: "unknown", 4: "unknown",
	} {
		if f.String() != want || f.Valid() != (want != "unknown") {
			t.Errorf("feature %d = %q valid %v, want %q", int(f), f.String(), f.Valid(), want)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-034, FR-042, FR-088, FR-089 - the last Focus Point can be spent (one available is enough for a paid use), a carried creature and the
// Temporary Hit Points are allowed at every level from 10 (level 11 and 20 as well as 10), and a negative roll total is refused whether or not a die is rolled.
// Path: per case (unhappy: boundary).
func TestMonkFocusBoundaries(t *testing.T) {
	t.Parallel()
	last := mnkState(1, false, 0) // level 2: 2 points, 1 expended
	wantReason(t, character.MonkFocus{}.Availability(mnkStats(t, 2), last, mnkUse(character.MonkFlurryOfBlows, true)), 0, "the last point")
	wantReason(t, character.MonkFocus{}.Availability(mnkStats(t, 2), mnkState(2, false, 0), mnkUse(character.MonkFlurryOfBlows, true)), character.MonkNotEnoughFocusPoints, "no point")
	large := character.MonkCarriedCreature{Chosen: true, Willing: true, Size: content.SizeLarge, DistanceFeet: 5}
	for _, level := range []int{10, 11, 20} {
		stats := mnkStats(t, level)
		wantReason(t, character.MonkFocus{}.Availability(stats, mnkState(0, false, 0), character.MonkFocusUse{Feature: character.MonkStepOfTheWind, SpendPoint: true, Carried: large}), 0, "carrying")
		if n := (character.MonkFocus{}).TemporaryHitPointDice(stats, mnkUse(character.MonkPatientDefense, true)); n != 2 {
			t.Errorf("Temporary Hit Point dice at level %d = %d, want 2", level, n)
		}
	}
	_, err := character.MonkFocus{}.Resolve(mnkStats(t, 5), mnkState(0, false, 0), mnkUse(character.MonkFlurryOfBlows, true), -1)
	wantInvalid(t, err, "a negative roll when no die is rolled")
	_, err = character.MonkFocus{}.Resolve(mnkStats(t, 10), mnkState(0, false, 0), mnkUse(character.MonkPatientDefense, true), -1)
	wantInvalid(t, err, "a negative roll")
}
