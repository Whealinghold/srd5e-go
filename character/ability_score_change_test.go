package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

func changeScore(t *testing.T, c character.Character, a ability.Ability, to, limit int) character.Transition[character.AbilityScoreChangeRejection] {
	t.Helper()
	tr, err := c.ChangeAbilityScore(a, ability.MustScore(to), limit)
	if err != nil {
		t.Fatalf("ChangeAbilityScore: %v", err)
	}
	return tr
}

// Requirement: doc/specs/character-aggregate.md FR-044 to FR-046, FR-049, FR-050 -
// Constitution changes: a modifier rise adds the rise times the total level and
// emits HitPointMaximumChanged after AbilityScoreChanged; an unchanged modifier,
// a lower score of the same modifier and other abilities change nothing else;
// lowering the modifier is rejected; the same score is a no-op.
// Path: per case.
func TestChangeConstitution(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		a        ability.Ability
		from, to int
		level    int
		rise     int
		hpEvent  bool
		rejected bool
		noop     bool
	}{
		{"SRD example", con, 17, 18, 8, 8, true, false, false}, {"level 1", con, 17, 18, 1, 1, true, false, false},
		{"modifier +2 to +3", con, 15, 17, 5, 5, true, false, false}, {"two-step rise", con, 14, 18, 3, 6, true, false, false},
		{"level 20", con, 17, 18, 20, 20, true, false, false}, {"modifier unchanged", con, 16, 17, 8, 0, false, false, false},
		{"lower, same modifier", con, 17, 16, 8, 0, false, false, false}, {"other ability", str, 10, 12, 8, 0, false, false, false},
		{"lowering the modifier", con, 18, 17, 8, 0, false, true, false}, {"same score", con, 17, 17, 8, 0, false, false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := must(flatScores(t, 10).With(c.a, ability.MustScore(c.from)))
			ch := restored(t, 400000, 76, s, cl("fighter", c.level))
			before := ch
			tr := changeScore(t, ch, c.a, c.to, 20)
			if !ch.Equal(before) {
				t.Error("the receiver must not change")
			}
			if c.rejected {
				r, ok := tr.(character.TransitionRejected[character.AbilityScoreChangeRejection])
				if !ok || r.Rejection() != (character.AbilityScoreChangeConstitutionDecrease{}) {
					t.Errorf("transition = %#v", tr)
				}
				return
			}
			ok, isChange := tr.(character.TransitionChanged[character.AbilityScoreChangeRejection])
			if !isChange {
				t.Fatalf("transition = %#v", tr)
			}
			n := ok.Character()
			if n.AbilityScores().Score(c.a).Value() != c.to || n.HitPointMaximum().Value() != 76+c.rise {
				t.Errorf("score %d maximum %d, want %d", n.AbilityScores().Score(c.a).Value(), n.HitPointMaximum().Value(), 76+c.rise)
			}
			evs := ok.Events()
			switch {
			case c.noop:
				if len(evs) != 0 || !n.Equal(ch) {
					t.Errorf("no-op: %v", evs)
				}
			case c.hpEvent:
				changed, a1 := evs[0].(character.AbilityScoreChanged)
				hp, a2 := evs[1].(character.HitPointMaximumChanged)
				if len(evs) != 2 || !a1 || !a2 || changed.Ability() != c.a || changed.Previous().Value() != c.from || changed.Current().Value() != c.to ||
					hp.Previous().Value() != 76 || hp.Current().Value() != 76+c.rise {
					t.Errorf("events = %#v", evs)
				}
			default:
				if _, only := evs[0].(character.AbilityScoreChanged); len(evs) != 1 || !only {
					t.Errorf("events = %#v", evs)
				}
			}
			for _, other := range ability.All() {
				if other != c.a && n.AbilityScores().Score(other) != ch.AbilityScores().Score(other) {
					t.Errorf("%v changed", other)
				}
			}
		})
	}
}

// Requirement: doc/specs/character-aggregate.md FR-047, FR-074, FR-075 - the cap
// defaults to 20 and may be 20 to 30; a score above it is rejected; the order is
// no-op, cap, Constitution decrease; a cap outside 20 to 30 is a validation error.
// Path: per case.
func TestChangeAbilityScoreCap(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		a        ability.Ability
		from, to int
		limit    int
		want     string
	}{
		{"default cap edge", str, 19, 20, 20, "changed"}, {"over default cap", str, 19, 21, 20, "cap"}, {"raised cap", str, 20, 22, 24, "changed"},
		{"at raised cap", str, 20, 24, 24, "changed"}, {"over raised cap", str, 20, 25, 24, "cap"}, {"cap at 30", str, 29, 30, 30, "changed"},
		{"same score above cap", str, 22, 22, 20, "changed"}, {"lowering, still over", str, 24, 22, 20, "cap"},
		{"over cap and a decrease", con, 24, 21, 20, "cap"}, {"Con decrease within cap", con, 18, 17, 20, "con"},
	} {
		s := must(flatScores(t, 10).With(c.a, ability.MustScore(c.from)))
		ch := restored(t, 400000, 60, s, cl("fighter", 5))
		var got string
		switch r := changeScore(t, ch, c.a, c.to, c.limit).(type) {
		case character.TransitionChanged[character.AbilityScoreChangeRejection]:
			got = "changed"
		case character.TransitionRejected[character.AbilityScoreChangeRejection]:
			switch v := r.Rejection().(type) {
			case character.AbilityScoreChangeExceedsMaximum:
				got = "cap"
				if v.Cap != c.limit {
					t.Errorf("%s: cap %d", c.name, v.Cap)
				}
			case character.AbilityScoreChangeConstitutionDecrease:
				got = "con"
			}
		}
		if got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	ch := restored(t, 0, 10, flatScores(t, 10), cl("fighter", 1))
	for _, limit := range []int{19, 31, 0, -1} {
		if _, err := ch.ChangeAbilityScore(str, ability.MustScore(12), limit); !isInvalid(err) {
			t.Errorf("cap %d: %v", limit, err)
		}
	}
	if _, err := ch.ChangeAbilityScore(0, ability.MustScore(12), 20); !isInvalid(err) {
		t.Errorf("invalid ability: %v", err)
	}
	if _, err := ch.ChangeAbilityScore(str, ability.Score{}, 20); !isInvalid(err) {
		t.Errorf("zero score: %v", err)
	}
	if _, err := (character.Character{}).ChangeAbilityScore(str, ability.MustScore(12), 20); !isInvalid(err) {
		t.Errorf("zero character: %v", err)
	}
}
