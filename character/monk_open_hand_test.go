package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

const openHand = character.OpenHandSubclassID

func technique(hit bool, effects ...character.OpenHandEffect) character.OpenHandTechniqueFacts {
	return character.OpenHandTechniqueFacts{HitByFlurryOfBlows: hit, Effects: effects}
}

// Requirement: doc/specs/class-monk.md FR-105 to FR-109, FR-142, FR-148 - Open Hand Technique: Addle stops Opportunity Attacks with no save; Push against the
// Monk's Focus save DC pushes 15 feet on a failure only; Topple against that DC makes the target Prone on a failure only; the DC used is recorded; exactly
// one valid effect is allowed per hit and only on a hit by an attack Flurry of Blows granted, from level 3 and in the Warrior of the Open Hand.
// Path: per case (happy, unhappy: invalid input, rule interaction, boundary).
func TestOpenHandTechnique(t *testing.T) {
	t.Parallel()
	o := character.OpenHandRules{}
	st, ab := mnkStats(t, 3), mnkAbilities(t, 3, 3, 3)
	addle := must(o.ResolveTechnique(st, openHand, ab, technique(true, character.OpenHandAddle), 0))
	if !addle.NoOpportunityAttacks() || addle.DifficultyClass() != 0 || addle.PushedFeet() != 0 || addle.Prone() || addle.Effect() != character.OpenHandAddle {
		t.Errorf("Addle = %+v", addle)
	}
	for _, c := range []struct {
		name   string
		effect character.OpenHandEffect
		o      d20.Outcome
		pushed int
		prone  bool
	}{
		{"push failed", character.OpenHandPush, d20.Failure, 15, false}, {"push resisted", character.OpenHandPush, d20.Success, 0, false},
		{"topple failed", character.OpenHandTopple, d20.Failure, 0, true}, {"topple resisted", character.OpenHandTopple, d20.Success, 0, false},
	} {
		got, err := o.ResolveTechnique(st, openHand, ab, technique(true, c.effect), c.o)
		if err != nil || got.DifficultyClass() != 13 || got.PushedFeet() != c.pushed || got.Prone() != c.prone || got.NoOpportunityAttacks() || got.Effect() != c.effect {
			t.Errorf("%s = %+v, %v", c.name, got, err)
		}
	}
	for _, c := range []struct {
		name     string
		stats    character.MonkLevelStats
		subclass content.SubclassID
		facts    character.OpenHandTechniqueFacts
		want     character.MonkReason
	}{
		{"a valid use", st, openHand, technique(true, character.OpenHandPush), 0},
		{"level 2", mnkStats(t, 2), openHand, technique(true, character.OpenHandPush), character.MonkLevelTooLow},
		{"no subclass", st, "", technique(true, character.OpenHandPush), character.MonkNotEligible},
		{"another subclass", st, "other", technique(true, character.OpenHandPush), character.MonkNotEligible},
		{"not a Flurry of Blows hit", st, openHand, technique(false, character.OpenHandPush), character.MonkNotEligible},
		{"two effects", st, openHand, technique(true, character.OpenHandPush, character.OpenHandTopple), character.MonkNotEligible},
		{"no effect", st, openHand, technique(true), character.MonkNotEligible},
		{"an unknown effect", st, openHand, technique(true, 0), character.MonkNotEligible},
		{"zero stats", character.MonkLevelStats{}, openHand, technique(true, character.OpenHandPush), character.MonkNotAMonk},
		{"level 2 and no subclass", mnkStats(t, 2), "", technique(true, character.OpenHandPush), character.MonkLevelTooLow},
	} {
		wantReason(t, o.TechniqueAvailability(c.stats, c.subclass, c.facts), c.want, c.name)
	}
	_, err := o.ResolveTechnique(st, openHand, ab, technique(true, character.OpenHandPush), 0)
	wantInvalid(t, err, "outcome")
	_, err = o.ResolveTechnique(st, "", ab, technique(true, character.OpenHandPush), d20.Failure)
	wantInvalid(t, err, "no subclass")
}

// Requirement: doc/specs/class-monk.md FR-113, FR-148 - Fleet Step makes Step of the Wind available after a Bonus Action other than Step of the Wind, from level
// 11 in the Warrior of the Open Hand; Step of the Wind itself, a missing or different subclass and an unknown action are notEligible, level 10 is
// levelTooLow, in that order.
// Path: per case (unhappy: boundary, rule interaction).
func TestOpenHandFleetStep(t *testing.T) {
	t.Parallel()
	o := character.OpenHandRules{}
	got, ok := o.FleetStep(mnkStats(t, 11), openHand, character.MonkBonusActionOther).(character.FleetStepAvailable)
	if !ok || got.Granted() != character.MonkStepOfTheWind {
		t.Errorf("FleetStep = %#v, want Step of the Wind available", got)
	}
	for _, c := range []struct {
		name     string
		stats    character.MonkLevelStats
		subclass content.SubclassID
		taken    character.MonkBonusAction
		want     character.MonkReason
	}{
		{"Step of the Wind taken", mnkStats(t, 11), openHand, character.MonkBonusActionStepOfTheWind, character.MonkNotEligible},
		{"unknown action", mnkStats(t, 11), openHand, 0, character.MonkNotEligible},
		{"no subclass", mnkStats(t, 11), "", character.MonkBonusActionOther, character.MonkNotEligible},
		{"level 10", mnkStats(t, 10), openHand, character.MonkBonusActionOther, character.MonkLevelTooLow},
		{"level 10 and no subclass", mnkStats(t, 10), "", character.MonkBonusActionStepOfTheWind, character.MonkLevelTooLow},
		{"zero stats", character.MonkLevelStats{}, openHand, character.MonkBonusActionOther, character.MonkNotAMonk},
	} {
		if r, ok := o.FleetStep(c.stats, c.subclass, c.taken).(character.FleetStepRejected); !ok || r.Reason != c.want {
			t.Errorf("%s: FleetStep = %#v, want rejected %v", c.name, r, c.want)
		}
	}
}

// Requirement: doc/specs/class-monk.md Table 4.10a, FR-110, FR-111, FR-112, FR-134, FR-135, FR-148, Table 4.11a - Wholeness of Body regains the die roll plus the
// Wisdom modifier, at least 1 Hit Point (uncapped), and spends one use; rows: typical, high, zero modifier, sum below and exactly at the minimum; it is
// levelTooLow below 6 (before the subclass), notEligible without the Warrior of the Open Hand (before the uses) and noUsesRemaining with every use spent, also
// after the Wisdom modifier fell below the spent count.
// Path: per case (rows of Table 4.10a, unhappy: state error, boundary).
func TestWholenessOfBody(t *testing.T) {
	t.Parallel()
	w := character.WholenessOfBody{}
	st := mnkStats(t, 6)
	for _, c := range []struct {
		name      string
		roll, wis int
		want      int
	}{{"typical", 3, 3, 6}, {"high", 6, 5, 11}, {"zero modifier", 1, 0, 1}, {"sum below minimum", 1, -1, 1}, {"sum exactly at minimum", 2, -1, 1}} {
		in := mnkState(1, false, 0)
		got, err := w.Resolve(st, in, openHand, modifierOf(t, c.wis), c.roll)
		if err != nil || got.HitPointsToRegain() != c.want || got.Rolled() != c.roll || got.State() != mnkState(1, false, 1) {
			t.Errorf("%s: Resolve = %+v, %v; want %d", c.name, got, err, c.want)
		}
	}
	three := modifierOf(t, 3)
	wantReason(t, w.Availability(mnkStats(t, 5), mnkState(0, false, 3), "", three), character.MonkLevelTooLow, "level 5, no uses and no subclass")
	wantReason(t, w.Availability(st, mnkState(0, false, 3), "", three), character.MonkNotEligible, "no subclass and no uses")
	wantReason(t, w.Availability(st, mnkState(0, false, 3), openHand, three), character.MonkNoUsesRemaining, "all uses spent")
	wantReason(t, w.Availability(st, mnkState(0, false, 2), openHand, three), 0, "one use left")
	wantReason(t, w.Availability(st, mnkState(0, false, 1), openHand, modifierOf(t, -3)), character.MonkNoUsesRemaining, "Wisdom fell, a later rise restores nothing")
	wantReason(t, w.Availability(st, mnkState(0, false, 2), openHand, modifierOf(t, 0)), character.MonkNoUsesRemaining, "Wisdom 0, one use, 2 spent")
	wantReason(t, w.Availability(character.MonkLevelStats{}, mnkState(0, false, 0), openHand, three), character.MonkNotAMonk, "zero stats")
	for _, roll := range []int{0, 9} {
		_, err := w.Resolve(st, mnkState(0, false, 0), openHand, three, roll)
		wantInvalid(t, err, "roll")
	}
	_, err := w.Resolve(st, mnkState(0, false, 3), openHand, three, 3)
	wantInvalid(t, err, "no uses")
	_, err = w.Resolve(character.MonkLevelStats{}, mnkState(0, false, 0), openHand, three, 3)
	wantInvalid(t, err, "zero stats")
}

// Requirement: doc/specs/class-monk.md FR-105 to FR-109 - the Open Hand Technique effect names are snake_case and an undefined value is unknown and not valid.
// Path: per case.
func TestOpenHandEffectNames(t *testing.T) {
	t.Parallel()
	for e, want := range map[character.OpenHandEffect]string{character.OpenHandAddle: "addle", character.OpenHandPush: "push", character.OpenHandTopple: "topple", 0: "unknown", 4: "unknown"} {
		if e.String() != want || e.Valid() != (want != "unknown") {
			t.Errorf("effect %d = %q valid %v, want %q", int(e), e.String(), e.Valid(), want)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-109, FR-113, FR-148, FR-110 - a subclass id that sorts after the Warrior of the Open Hand is as foreign as one that sorts before it, an unknown
// Bonus Action and an unknown outcome are refused, the highest Martial Arts roll (8 on the d8 of level 6) is valid, and the declared Bonus Action values start at 1.
// Path: per case (unhappy: boundary, invalid input).
func TestOpenHandBoundaries(t *testing.T) {
	t.Parallel()
	o := character.OpenHandRules{}
	st := mnkStats(t, 11)
	const after = content.SubclassID("zzz_after")
	wantReason(t, o.TechniqueAvailability(st, after, technique(true, character.OpenHandPush)), character.MonkNotEligible, "a later subclass")
	if r, ok := o.FleetStep(st, after, character.MonkBonusActionOther).(character.FleetStepRejected); !ok || r.Reason != character.MonkNotEligible {
		t.Errorf("FleetStep for a later subclass = %#v", r)
	}
	if r, ok := o.FleetStep(st, openHand, character.MonkBonusAction(3)).(character.FleetStepRejected); !ok || r.Reason != character.MonkNotEligible {
		t.Errorf("FleetStep for an unknown action = %#v", r)
	}
	wantReason(t, character.WholenessOfBody{}.Availability(mnkStats(t, 6), mnkState(0, false, 0), after, modifierOf(t, 3)), character.MonkNotEligible, "a later subclass")
	got, err := character.WholenessOfBody{}.Resolve(mnkStats(t, 6), mnkState(0, false, 0), openHand, modifierOf(t, 3), 8)
	if err != nil || got.HitPointsToRegain() != 11 {
		t.Errorf("a roll of 8 = %+v, %v, want 11 Hit Points", got, err)
	}
	for _, bad := range []d20.Outcome{0, 3} {
		_, err := o.ResolveTechnique(mnkStats(t, 3), openHand, mnkAbilities(t, 3, 3, 3), technique(true, character.OpenHandTopple), bad)
		wantInvalid(t, err, "outcome")
	}
	if character.MonkBonusActionStepOfTheWind != 1 || character.MonkBonusActionOther != 2 {
		t.Error("the declared Bonus Action values must start at 1 so that the zero value is none of them")
	}
}
