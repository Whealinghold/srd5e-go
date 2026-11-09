package character_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-monk.md FR-094, FR-095, FR-096, FR-141 - Disciplined Survivor only expends 1 Focus Point (the host rerolls); it is
// levelTooLow below 14, notEligible when the saving throw did not fail, notEnoughFocusPoints with none, and Resolve refuses an unavailable use; the
// event names the Character.
// Path: per case (unhappy: boundary, state error).
func TestDisciplinedSurvivor(t *testing.T) {
	t.Parallel()
	d := character.DisciplinedSurvivor{}
	st, none := mnkStats(t, 14), mnkState(0, false, 0)
	res, err := d.Resolve(st, none, true)
	if err != nil || res.FocusPointsSpent() != 1 || res.State() != mnkState(1, false, 0) {
		t.Fatalf("Resolve = %+v, %v", res, err)
	}
	id := character.MustIdentifier("pc-1")
	if ev, ok := d.Event(id).(character.DisciplinedSurvivorUsed); !ok || ev.CharacterID() != id {
		t.Errorf("event = %#v", d.Event(id))
	}
	wantReason(t, d.Availability(mnkStats(t, 13), none, true), character.MonkLevelTooLow, "level 13")
	wantReason(t, d.Availability(st, none, false), character.MonkNotEligible, "the save did not fail")
	wantReason(t, d.Availability(st, mnkState(14, false, 0), true), character.MonkNotEnoughFocusPoints, "no points")
	wantReason(t, d.Availability(st, mnkState(14, false, 0), false), character.MonkNotEligible, "no failure and no points")
	wantReason(t, d.Availability(character.MonkLevelStats{}, none, true), character.MonkNotAMonk, "zero stats")
	_, err = d.Resolve(st, mnkState(14, false, 0), true)
	wantInvalid(t, err, "unavailable")
}

// Requirement: doc/specs/class-monk.md FR-100, FR-101, FR-102 - Superior Defense expends 3 Focus Points at the start of the Monk's turn from level 18 and
// is a SuperiorDefenseEffect lasting 1 minute that ends when Incapacitated and resists every damage type except Force (a test of the resisted set for each
// of the 13 types); it is levelTooLow below 18, notEligible outside the start of the turn, notEnoughFocusPoints with fewer than 3.
// Path: per case (unhappy: boundary, state error).
func TestSuperiorDefense(t *testing.T) {
	t.Parallel()
	s := character.SuperiorDefense{}
	st := mnkStats(t, 18)
	res, err := s.Resolve(st, mnkState(0, false, 0), true)
	if err != nil || res.FocusPointsSpent() != 3 || res.State() != mnkState(3, false, 0) {
		t.Fatalf("Resolve = %+v, %v", res, err)
	}
	eff := s.Effect()
	if eff.Duration() != 60*time.Second || !eff.EndsWhenIncapacitated() || !slices.Equal(eff.ResistsAllExcept(), []content.DamageType{content.DamageForce}) {
		t.Errorf("effect = %v %v %v", eff.Duration(), eff.EndsWhenIncapacitated(), eff.ResistsAllExcept())
	}
	for t2 := content.DamageAcid; t2 <= content.DamageThunder; t2++ {
		if eff.Resists(t2) == (t2 == content.DamageForce) {
			t.Errorf("Resists(%v) = %v", t2, eff.Resists(t2))
		}
	}
	if eff.Resists(0) {
		t.Error("an invalid damage type is not resisted")
	}
	id := character.MustIdentifier("pc-1")
	if ev, ok := s.Event(id).(character.SuperiorDefenseActivated); !ok || ev.CharacterID() != id {
		t.Errorf("event = %#v", s.Event(id))
	}
	wantReason(t, s.Availability(mnkStats(t, 17), mnkState(0, false, 0), true), character.MonkLevelTooLow, "level 17")
	wantReason(t, s.Availability(st, mnkState(0, false, 0), false), character.MonkNotEligible, "outside the start of the turn")
	wantReason(t, s.Availability(st, mnkState(16, false, 0), true), character.MonkNotEnoughFocusPoints, "2 points")
	wantReason(t, s.Availability(st, mnkState(15, false, 0), true), 0, "3 points")
	wantReason(t, s.Availability(character.MonkLevelStats{}, mnkState(0, false, 0), true), character.MonkNotAMonk, "zero stats")
	_, err = s.Resolve(st, mnkState(16, false, 0), true)
	wantInvalid(t, err, "unavailable")
}

// Requirement: doc/specs/class-monk.md FR-095, FR-096 - the last Focus Point is enough to reroll a saving throw.
// Path: unhappy (boundary).
func TestDisciplinedSurvivorLastPoint(t *testing.T) {
	t.Parallel()
	st := mnkStats(t, 14)
	wantReason(t, character.DisciplinedSurvivor{}.Availability(st, mnkState(13, false, 0), true), 0, "1 point left")
	res, err := character.DisciplinedSurvivor{}.Resolve(st, mnkState(13, false, 0), true)
	if err != nil || res.State().FocusPoints(st).Available() != 0 {
		t.Errorf("Resolve = %+v, %v; want the last point spent", res, err)
	}
}
