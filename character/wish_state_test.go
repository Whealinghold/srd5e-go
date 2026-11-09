package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/wish-spell.md FR-060 - the state is stored under the key "wish_stress"; an absent state reads as the zero
// WishStress (no weakness, no cast damage, can cast); a nil part reads as its first variant; the state and its parts compare by value.
// Path: happy.
func TestWishStressZeroValueKeyAndEquality(t *testing.T) {
	t.Parallel()
	if got := character.WishStressKey().Name(); got != "wish_stress" {
		t.Errorf("key = %q", got)
	}
	c := wishWizard(t)
	got, ok, err := character.ReadFeatureState(c.FeatureStates(), character.WishStressKey())
	if err != nil || ok || got != (character.WishStress{}) {
		t.Errorf("absent state = %v, %v, %v", got, ok, err)
	}
	z := character.WishStress{}
	if z.Weakness() != (character.NoWishWeakness{}) || z.CastDamage() != (character.WishCastDamageInactive{}) || z.CastingAbility() != (character.CanCastWish{}) || !z.CanCastWish() {
		t.Errorf("zero = %v %v %v", z.Weakness(), z.CastDamage(), z.CastingAbility())
	}
	if nilParts := must(character.NewWishStress(nil, nil, nil)); nilParts != z {
		t.Errorf("nil parts = %v, want the zero value", nilParts)
	}
	s := wishStress(t, 4, true, true)
	if s != wishStress(t, 4, true, true) || s == wishStress(t, 4, true, false) || s == wishStress(t, 3, true, true) || s == wishStress(t, 4, false, true) {
		t.Error("stress does not compare by value")
	}
	if s.Weakness() != any(must(character.NewWishWeakened(4))) || s.CastDamage() != (character.WishCastDamageActive{}) || s.CastingAbility() != (character.CannotCastWishAgain{}) || s.CanCastWish() {
		t.Errorf("parts = %v %v %v", s.Weakness(), s.CastDamage(), s.CastingAbility())
	}
	if !s.Equal(wishStress(t, 4, true, true)) || s.Equal(z) || s.Equal(character.SpentHitDice{}) {
		t.Error("Equal does not compare by value and type")
	}
	if w := must(character.NewWishWeakened(4)); w.DaysRemaining() != 4 {
		t.Errorf("DaysRemaining = %d", w.DaysRemaining())
	}
}

// Requirement: doc/specs/wish-spell.md FR-060, FR-068 - a weakness of fewer than 1 day is an *InvalidError, in NewWishWeakened and in
// NewWishStress (the zero WishWeakened).
// Path: unhappy (invalid input).
func TestWishWeakenedMustHaveDays(t *testing.T) {
	t.Parallel()
	for _, days := range []int{0, -1} {
		if _, err := character.NewWishWeakened(days); !isInvalid(err) {
			t.Errorf("NewWishWeakened(%d) = %v", days, err)
		}
	}
	if _, err := character.NewWishWeakened(1); err != nil {
		t.Errorf("NewWishWeakened(1) = %v", err)
	}
	if _, err := character.NewWishStress(character.WishWeakened{}, nil, nil); !isInvalid(err) {
		t.Errorf("zero WishWeakened = %v", err)
	}
}

// Requirement: per case.
// Path: per case.
func TestNewWishStressRoll(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		days, percentile        int
		invalid, locks          bool
	}{
		{"shortest days, 34 is safe", wishSpec + " FR-022, S1", "unhappy (boundary)", 2, 34, false, false},
		{"longest days, 33 locks", wishSpec + " FR-022, S2", "unhappy (boundary)", 8, 33, false, true},
		{"percentile 1 locks", wishSpec + " FR-022, S3", "unhappy (boundary)", 5, 1, false, true},
		{"percentile 100 is safe", wishSpec + " FR-022, S4", "unhappy (boundary)", 5, 100, false, false},
		{"days 1", wishSpec + " FR-022, S8", "unhappy (invalid input)", 1, 50, true, false},
		{"days 9", wishSpec + " FR-022, S8", "unhappy (invalid input)", 9, 50, true, false},
		{"days 0", wishSpec + " FR-022, S8", "unhappy (invalid input)", 0, 50, true, false},
		{"percentile 0", wishSpec + " FR-022, S8", "unhappy (invalid input)", 5, 0, true, false},
		{"percentile 101", wishSpec + " FR-022, S8", "unhappy (invalid input)", 5, 101, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := character.NewWishStressRoll(c.days, c.percentile)
			if c.invalid {
				if !isInvalid(err) || !r.IsZero() {
					t.Errorf("= %v, %v, want invalid input [%s]", r, err, c.requirement)
				}
				return
			}
			if err != nil || r.Days() != c.days || r.Percentile() != c.percentile || r.LocksOut() != c.locks || r.IsZero() {
				t.Errorf("= %+v, %v, locks %v [%s]", r, err, r.LocksOut(), c.requirement)
			}
		})
	}
}

// Requirement: per case.
// Path: per case.
func TestIncurWishStress(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		prior                   character.WishStress
		days, percentile        int
		want                    character.WishStress
	}{
		{"none, 2 days, 34", wishSpec + " FR-022, FR-023, FR-024, S1", "unhappy (boundary)", wishStress(t, 0, false, false), 2, 34, wishStress(t, 2, true, false)},
		{"none, 8 days, 33 locks", wishSpec + " FR-022, FR-025, S2", "unhappy (boundary)", wishStress(t, 0, false, false), 8, 33, wishStress(t, 8, true, true)},
		{"none, percentile 1", wishSpec + " FR-025, S3", "unhappy (boundary)", wishStress(t, 0, false, false), 5, 1, wishStress(t, 5, true, true)},
		{"none, percentile 100", wishSpec + " FR-022, S4", "unhappy (boundary)", wishStress(t, 0, false, false), 5, 100, wishStress(t, 5, true, false)},
		{"weakened 6, newest roll 3 governs", wishSpec + " FR-023, S5", "unhappy (rule interaction)", wishStress(t, 6, true, false), 3, 50, wishStress(t, 3, true, false)},
		{"weakened 2, newest roll 7 governs", wishSpec + " FR-023, S6", "unhappy (rule interaction)", wishStress(t, 2, true, false), 7, 50, wishStress(t, 7, true, false)},
		{"the lockout survives a percentile of 90", wishSpec + " FR-025, S7", "unhappy (state error)", wishStress(t, 0, false, true), 4, 90, wishStress(t, 4, true, true)},
		{"cast damage stays active", wishSpec + " FR-024", "happy", wishStress(t, 0, true, false), 4, 90, wishStress(t, 4, true, false)},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roll := must(character.NewWishStressRoll(c.days, c.percentile))
			if got := character.IncurWishStress(c.prior, roll); got != c.want {
				t.Errorf("= %+v, want %+v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-030 (SRD 5.2.1, "Wish", p. 175) - while weakened Strength is exactly 3 whatever the stored value (1, 2,
// 3, 4 or 14), the other five abilities and the stored scores never change, and without the weakness the scores are returned equal.
// Path: unhappy (boundary).
func TestWishEffectiveScores(t *testing.T) {
	t.Parallel()
	weak, none := wishStress(t, 3, true, false), wishStress(t, 0, true, false)
	for _, str := range []int{1, 2, 3, 4, 14} {
		stored := scoresOf(t, str, 11, 12, 13, 14, 15)
		got := character.WishEffectiveScores(stored, weak)
		if got.Score(ability.Strength) != ability.MustScore(3) {
			t.Errorf("stored %d: Strength = %v, want 3", str, got.Score(ability.Strength))
		}
		for _, a := range ability.All()[1:] {
			if got.Score(a) != stored.Score(a) {
				t.Errorf("stored %d: %v changed to %v", str, a, got.Score(a))
			}
		}
		if stored != scoresOf(t, str, 11, 12, 13, 14, 15) || character.WishEffectiveScores(stored, none) != stored {
			t.Errorf("stored %d: input changed or not equal without weakness", str)
		}
	}
	if z := character.WishEffectiveScores(ability.Scores{}, weak); !z.IsZero() {
		t.Errorf("zero scores = %v, want the zero Scores", z)
	}
}

// Requirement: per case.
// Path: per case.
func TestWishAfterDay(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		prior                   character.WishStress
		day                     character.WishRecoveryDay
		want                    character.WishStress
		ended                   bool
	}{
		{"weakened 5, ordinary day", wishSpec + " FR-050, D10", "happy", wishStress(t, 5, true, true), character.WishOrdinaryDay, wishStress(t, 4, true, true), false},
		{"weakened 5, resting day", wishSpec + " FR-050, D11", "happy", wishStress(t, 5, true, false), character.WishRestingDayWithLightActivity, wishStress(t, 3, true, false), false},
		{"weakened 1, resting day ends it", wishSpec + " FR-050, D12", "unhappy (boundary)", wishStress(t, 1, true, false), character.WishRestingDayWithLightActivity, wishStress(t, 0, true, false), true},
		{"weakened 2, ordinary day leaves 1", wishSpec + " FR-050, D13", "unhappy (boundary)", wishStress(t, 2, false, false), character.WishOrdinaryDay, wishStress(t, 1, false, false), false},
		{"weakened 2, resting day ends it", wishSpec + " FR-050, D14", "unhappy (boundary)", wishStress(t, 2, false, false), character.WishRestingDayWithLightActivity, wishStress(t, 0, false, false), true},
		{"weakened 1, ordinary day ends it", wishSpec + " FR-050", "unhappy (boundary)", wishStress(t, 1, false, false), character.WishOrdinaryDay, wishStress(t, 0, false, false), true},
		{"no weakness, ordinary", wishSpec + " FR-051, D15", "unhappy (empty and minimal)", wishStress(t, 0, true, true), character.WishOrdinaryDay, wishStress(t, 0, true, true), false},
		{"no weakness, resting", wishSpec + " FR-051, D15", "unhappy (empty and minimal)", wishStress(t, 0, false, false), character.WishRestingDayWithLightActivity, wishStress(t, 0, false, false), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			prior := c.prior
			tr := character.WishAfterDay(caster, c.prior, c.day)
			ev := tr.Events()
			if tr.Stress() != c.want || c.prior != prior {
				t.Errorf("= %+v, want %+v [%s]", tr.Stress(), c.want, c.requirement)
			}
			switch {
			case c.ended && (len(ev) != 1 || !isWeaknessEndedFor(ev[0], caster)):
				t.Errorf("events = %#v, want WishWeaknessEnded", ev)
			case !c.ended && len(ev) != 0:
				t.Errorf("events = %#v, want none", ev)
			}
		})
	}
	if character.WishRestingDayReduction != 2 || character.WishOrdinaryDay.String() != "ordinary" || character.WishRestingDayWithLightActivity.String() != "resting_with_light_activity" {
		t.Error("recovery day constants or names changed")
	}
}

// Requirement: doc/specs/wish-spell.md FR-052, FR-053, Table D (D16 to D21) - a finished Long Rest ends the cast damage with WishCastDamageEnded and
// changes neither the weakness nor the lockout; a Short Rest, an interrupted Long Rest and an inactive state are returned equal with no
// events (a Long Rest is not a recovery day); a rest of another character is an *InvalidError.
// Path: per case.
func TestWishAfterRest(t *testing.T) {
	t.Parallel()
	taker := character.NewRestTaker()
	rest := func(kind character.RestKind, end character.RestEnd) character.RestApplied {
		t.Helper()
		switch r := taker.TakeRest(t.Context(), simpleRest(t, kind, end)).(type) {
		case character.TakeRestApplied:
			return r.Applied()
		case character.TakeRestAppliedAfterInterruption:
			return r.Applied()
		default:
			t.Fatalf("rest = %#v", r)
			return character.RestApplied{}
		}
	}
	long, short := rest(character.LongRest, ran(8, 0)), rest(character.ShortRest, ran(1, 0))
	interrupted := rest(character.LongRest, stopped(rDmg, 3, 0))
	if interrupted.BenefitsOf() != character.ShortRest {
		t.Fatalf("fixture: interrupted rest gives %v", interrupted.BenefitsOf())
	}
	for _, c := range []struct {
		name, requirement, path string
		prior                   character.WishStress
		rest                    character.RestApplied
		want                    character.WishStress
		ended                   bool
	}{
		{"Long Rest ends the cast damage", wishSpec + " FR-052, D16", "happy", wishStress(t, 6, true, true), long, wishStress(t, 6, false, true), true},
		{"Short Rest changes nothing", wishSpec + " FR-052, D17", "unhappy (rule interaction)", wishStress(t, 6, true, true), short, wishStress(t, 6, true, true), false},
		{"interrupted Long Rest changes nothing", wishSpec + " FR-052, D18", "unhappy (rule interaction)", wishStress(t, 0, true, false), interrupted, wishStress(t, 0, true, false), false},
		{"inactive state, Long Rest", wishSpec + " FR-052, D19", "unhappy (empty and minimal)", wishStress(t, 0, false, false), long, wishStress(t, 0, false, false), false},
		{"a Long Rest is not a recovery day", wishSpec + " FR-053, D20", "unhappy (rule interaction)", wishStress(t, 6, false, false), long, wishStress(t, 6, false, false), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := character.WishAfterRest(restHero, c.prior, c.rest)
			ev := tr.Events()
			if err != nil || tr.Stress() != c.want {
				t.Fatalf("= %+v, %v, want %+v [%s]", tr.Stress(), err, c.want, c.requirement)
			}
			if c.ended && (len(ev) != 1 || ev[0].(character.WishCastDamageEnded).CharacterID() != restHero) || !c.ended && len(ev) != 0 {
				t.Errorf("events = %#v, ended %v", ev, c.ended)
			}
		})
	}
	if _, err := character.WishAfterRest(caster, wishStress(t, 0, true, false), long); !isInvalid(err) {
		t.Errorf("rest of another character = %v, want invalid input [FR-062, D21]", err)
	}
}

// Requirement: doc/specs/wish-spell.md FR-065 - the events of a transition are a copy.
// Path: unhappy (immutability).
func TestWishTransitionEventsAreACopy(t *testing.T) {
	t.Parallel()
	tr := character.WishAfterDay(caster, wishStress(t, 1, false, false), character.WishOrdinaryDay)
	tr.Events()[0] = nil
	if ev := tr.Events(); len(ev) != 1 || ev[0] == nil {
		t.Errorf("events = %#v after a mutation of the returned slice", ev)
	}
}

func isWeaknessEndedFor(e character.Event, id character.Identifier) bool {
	we, ok := e.(character.WishWeaknessEnded)
	return ok && we.CharacterID() == id
}
