package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice/dicetest"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-098 and table "Conflict DC" - DC 12 plus the item's Charisma modifier).
// Path: per case.
func TestSentientConflictDC(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Conflict", p. 208`
	cases := []struct {
		name, requirement, path string
		charisma, dc            int
	}{
		{"average", p, "happy", 10, 12},
		{"one below", p, "unhappy (boundary)", 9, 11},
		{"strong", p, "happy", 18, 16},
		{"lowest", p, "unhappy (boundary)", 1, 7},
		{"highest", p, "unhappy (boundary)", 30, 22},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (character.SentientConflict{}).DC(score(t, c.charisma)); got != d20.DifficultyClass(c.dc) {
				t.Errorf("DC = %d, want %d [%s]", got, c.dc, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-099 - demands are one to four distinct SentientDemands.
// Path: per case.
func TestNewSentientDemands(t *testing.T) {
	t.Parallel()
	d := []content.SentientDemand{content.SentientDemandChaseMyDreams, content.SentientDemandGetRidOfIt, content.SentientDemandItsTimeForAChange, content.SentientDemandKeepMeClose}
	for n := 1; n <= 4; n++ {
		got, err := character.NewSentientDemands(d[:n])
		if err != nil || len(got.Demands()) != n || got.IsZero() {
			t.Errorf("%d demands: %+v, %v", n, got, err)
		}
	}
	cases := []struct {
		name string
		in   []content.SentientDemand
		want character.SentientDemandsReason
	}{
		{"empty", nil, character.SentientDemandsEmpty},
		{"duplicate", []content.SentientDemand{d[0], d[1], d[0]}, character.SentientDemandsDuplicate},
		{"too many", append(append([]content.SentientDemand(nil), d...), d[0]), character.SentientDemandsTooMany},
	}
	for _, c := range cases {
		var rej *character.SentientDemandsRejected
		if _, err := character.NewSentientDemands(c.in); !errors.As(err, &rej) || rej.Reason != c.want || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.SentientDemandsRejected{Reason: c.want}) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if _, err := character.NewSentientDemands([]content.SentientDemand{0}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid demand: %v", err)
	}
	got := must(character.NewSentientDemands(d[:2]))
	copyOf := got.Demands()
	copyOf[0] = 0
	if got.Demands()[0] != d[0] || !(character.SentientDemands{}).IsZero() {
		t.Error("Demands is not a copy")
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-103, FR-113 - the control lasts 1 to 12 hours, charms the bearer and obliges it to follow the item's commands; the power returns at dawn.
// Path: unhappy (boundary).
func TestSentientControl(t *testing.T) {
	t.Parallel()
	for _, h := range []int{1, 6, 12} {
		e, err := character.NewSentientControlEffect(h)
		if err != nil || e.Hours() != h || e.Condition() != content.ConditionCharmed || !e.MustTryToFollowItemCommands() {
			t.Errorf("hours %d: %+v, %v", h, e, err)
		}
	}
	for _, h := range []int{-1, 0, 13} {
		if _, err := character.NewSentientControlEffect(h); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("hours %d: %v", h, err)
		}
	}
	if character.ControlPowerSpentUntilNextDawn.AfterDawn() != character.ControlPowerAvailable || character.ControlPowerAvailable != character.SentientControlPower(0) {
		t.Error("AfterDawn")
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-123 - the conflict save is a Charisma saving throw: a rolled save is a saving throw
// against the conflict DC with the bearer's modifiers, and a forced failure carries its own reason (chosen to fail, automatic) and the saving throw kind.
// Path: per case.
func TestSentientSaveKindsAndForcedReasons(t *testing.T) {
	t.Parallel()
	use := func(a d20.SaveAttempt, faces ...int) character.SentientSave {
		res := must(character.NewRepeatSentientControlSave(dicetest.NewFixedRoller(faces...))).Do
		out, err := res(t.Context(), character.RepeatSentientControlSaveCommand{CharacterID: holder, Instance: inst("s"), ItemCharisma: score(t, 10), Attempt: a})
		if err != nil {
			t.Fatal(err)
		}
		switch r := out.(type) {
		case character.RepeatSentientControlSaveEnded:
			return r.Save()
		case character.RepeatSentientControlSaveContinues:
			return r.Save()
		}
		return nil
	}
	rolled := use(rollSave(3), 10).(character.SentientSaveRolled)
	if rolled.Check.Kind() != d20.SavingThrow || rolled.Check.Total() != 13 || rolled.Check.Target() != 12 {
		t.Errorf("rolled = %+v", rolled.Check)
	}
	for _, c := range []struct {
		a      d20.SaveAttempt
		reason d20.ForcedFailureReason
	}{{d20.ChooseToFail{}, d20.ChosenToFail}, {d20.AutomaticFailure{}, d20.Automatic}} {
		f := use(c.a).(character.SentientSaveForced).Failure
		if f.Reason() != c.reason || f.Kind() != d20.SavingThrow {
			t.Errorf("forced = %+v, want %v", f, c.reason)
		}
	}
}
