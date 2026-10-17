package ability_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func TestAbilityStringParseRoundTrip(t *testing.T) {
	t.Parallel()
	all := ability.All()
	if len(all) != 6 {
		t.Fatalf("All() has %d abilities", len(all))
	}
	for _, a := range all {
		got, err := ability.ParseAbility(strings.ToUpper(a.String()))
		if err != nil || got != a {
			t.Errorf("ParseAbility(%q) = %v, %v", a, got, err)
		}
	}
	if all[0] != ability.Strength || all[5] != ability.Charisma {
		t.Error("All() is not in SRD order")
	}
	all[0] = ability.Charisma
	if ability.All()[0] != ability.Strength {
		t.Error("All() shares storage between calls")
	}
}

func TestAbilityInvalid(t *testing.T) {
	t.Parallel()
	for _, a := range []ability.Ability{0, -1, 7} {
		if a.Valid() || a.String() != "unknown" {
			t.Errorf("Ability(%d) valid=%v string=%q", int(a), a.Valid(), a.String())
		}
	}
	_, err := ability.ParseAbility(strings.Repeat("x", 100))
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &ability.InvalidError{Reason: ability.ReasonUnknownAbility}) {
		t.Errorf("err = %v", err)
	}
	var ie *ability.InvalidError
	if !errors.As(err, &ie) || !strings.Contains(ie.Error(), "...") {
		t.Errorf("long input not truncated: %v", err)
	}
}

// Requirement: doc/specs/ability-scores.md FR-012 - every score 1 to 30 maps to
// the modifier in the SRD table, written out as literals (not recomputed).
// Path: happy (bands) and boundary (1, 10, 30).
func TestScoreModifierTable(t *testing.T) {
	t.Parallel()
	want := [...]int{
		1: -5, 2: -4, 3: -4, 4: -3, 5: -3, 6: -2, 7: -2, 8: -1, 9: -1, 10: 0,
		11: 0, 12: 1, 13: 1, 14: 2, 15: 2, 16: 3, 17: 3, 18: 4, 19: 4, 20: 5,
		21: 5, 22: 6, 23: 6, 24: 7, 25: 7, 26: 8, 27: 8, 28: 9, 29: 9, 30: 10,
	}
	for v := ability.MinScore; v <= ability.MaxScore; v++ {
		s, err := ability.NewScore(v)
		if err != nil {
			t.Fatalf("NewScore(%d): %v", v, err)
		}
		if s.Value() != v || s.Modifier().Value() != want[v] {
			t.Errorf("score %d: modifier %d, want %d", v, s.Modifier().Value(), want[v])
		}
	}
}

func TestNewScoreRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		v      int
		reason ability.Reason
	}{{0, ability.ReasonBelowMinimum}, {-7, ability.ReasonBelowMinimum}, {31, ability.ReasonAboveMaximum}, {1 << 40, ability.ReasonAboveMaximum}}
	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.v), func(t *testing.T) {
			t.Parallel()
			s, err := ability.NewScore(tc.v)
			if !s.IsZero() || !errors.Is(err, &ability.InvalidError{Reason: tc.reason}) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("got %v, %v", s, err)
			}
		})
	}
}

func TestMustScorePanicsWithError(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("recovered %v", r)
		}
	}()
	ability.MustScore(0)
}

func TestScoreAndModifierStrings(t *testing.T) {
	t.Parallel()
	var zero ability.Score
	if !zero.IsZero() || zero.String() != "invalid" || zero.Value() != 0 || zero.Modifier().Value() != 0 {
		t.Error("zero Score misbehaves")
	}
	if ability.MustScore(14).String() != "14" {
		t.Error("Score.String")
	}
	for v, w := range map[int]string{8: "-1", 10: "+0", 15: "+2"} {
		if g := ability.MustScore(v).Modifier().String(); g != w {
			t.Errorf("score %d modifier %q, want %q", v, g, w)
		}
	}
}

func full() map[ability.Ability]ability.Score {
	m := map[ability.Ability]ability.Score{}
	for i, a := range ability.All() {
		m[a] = ability.MustScore(8 + i)
	}
	return m
}

func TestNewScores(t *testing.T) {
	t.Parallel()
	s, err := ability.NewScores(full())
	if err != nil || s.IsZero() {
		t.Fatalf("NewScores: %v", err)
	}
	if s.Score(ability.Charisma).Value() != 13 || s.Modifier(ability.Charisma).Value() != 1 {
		t.Error("wrong lookup")
	}
	if !s.Score(0).IsZero() || !s.Score(9).IsZero() {
		t.Error("invalid ability should give zero Score")
	}
	again, _ := ability.NewScores(full())
	if s != again {
		t.Error("equal Scores are not ==")
	}
	if !strings.HasPrefix(s.String(), "Scores(strength: 8,") {
		t.Errorf("String = %s", s)
	}
}

func TestNewScoresReportsEverything(t *testing.T) {
	t.Parallel()
	m := full()
	delete(m, ability.Wisdom)
	m[ability.Charisma] = ability.Score{}
	m[ability.Ability(42)] = ability.MustScore(10)
	s, err := ability.NewScores(m)
	if !s.IsZero() {
		t.Error("expected zero Scores on error")
	}
	if !errors.Is(err, &ability.InvalidError{Reason: ability.ReasonMissingScore}) ||
		!errors.Is(err, &ability.InvalidError{Reason: ability.ReasonUnknownAbility}) {
		t.Fatalf("err = %v", err)
	}
	if n := strings.Count(err.Error(), "\n") + 1; n != 3 {
		t.Errorf("%d problems reported, want 3:\n%v", n, err)
	}
}

func TestScoresWith(t *testing.T) {
	t.Parallel()
	s, _ := ability.NewScores(full())
	n, err := s.With(ability.Strength, ability.MustScore(20))
	if err != nil || n.Score(ability.Strength).Value() != 20 || s.Score(ability.Strength).Value() != 8 {
		t.Fatalf("With: %v (original must not change)", err)
	}
	if _, err := s.With(0, ability.MustScore(10)); !errors.Is(err, &ability.InvalidError{Reason: ability.ReasonUnknownAbility}) {
		t.Errorf("invalid ability: %v", err)
	}
	if _, err := s.With(ability.Wisdom, ability.Score{}); !errors.Is(err, &ability.InvalidError{Reason: ability.ReasonMissingScore}) {
		t.Errorf("zero score: %v", err)
	}
	var zero ability.Scores
	if !zero.IsZero() || zero.Modifier(ability.Strength).Value() != 0 {
		t.Error("zero Scores misbehaves")
	}
}

func TestReasonStringAndInvalidError(t *testing.T) {
	t.Parallel()
	if ability.Reason(0).String() != "unknown" || ability.Reason(99).String() != "unknown" || ability.ReasonBelowMinimum.String() != "below minimum" {
		t.Error("Reason.String")
	}
	e := &ability.InvalidError{Reason: ability.ReasonAboveMaximum}
	if e.Error() != "ability: above maximum" || errors.Is(e, errors.New("x")) {
		t.Error("InvalidError")
	}
	if !errors.Is(e, &ability.InvalidError{Reason: ability.ReasonAboveMaximum, Detail: "ignored"}) {
		t.Error("Is should ignore Detail")
	}
}

func FuzzParseAbility(f *testing.F) {
	for _, s := range []string{"strength", "WISDOM", "", "str", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		a, err := ability.ParseAbility(s)
		if err != nil {
			if a != 0 || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("bad failure: %v %v", a, err)
			}
			return
		}
		if !a.Valid() || !strings.EqualFold(a.String(), s) {
			t.Fatalf("ParseAbility(%q) = %v", s, a)
		}
	})
}

func FuzzNewScore(f *testing.F) {
	for _, v := range []int{-1, 0, 1, 30, 31} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, v int) {
		s, err := ability.NewScore(v)
		if (err == nil) != (v >= 1 && v <= 30) {
			t.Fatalf("NewScore(%d) err = %v", v, err)
		}
		if err == nil && s.Value() != v {
			t.Fatalf("value %d", s.Value())
		}
	})
}

func ExampleNewScore() {
	s, _ := ability.NewScore(15)
	fmt.Println(s, s.Modifier())
	_, err := ability.NewScore(31)
	fmt.Println(errors.Is(err, srderr.ErrInvalidInput))
	// Output:
	// 15 +2
	// true
}

func ExampleScores_With() {
	m := map[ability.Ability]ability.Score{}
	for _, a := range ability.All() {
		m[a] = ability.MustScore(10)
	}
	s, _ := ability.NewScores(m)
	s, _ = s.With(ability.Dexterity, ability.MustScore(16))
	fmt.Println(s.Modifier(ability.Dexterity), s.Modifier(ability.Strength))
	// Output: +3 +0
}

func TestNewScoresNilMap(t *testing.T) {
	t.Parallel()
	_, err := ability.NewScores(nil)
	if n := strings.Count(err.Error(), "\n") + 1; n != 6 || !errors.Is(err, &ability.InvalidError{Reason: ability.ReasonMissingScore}) {
		t.Errorf("want 6 missing-score problems, got %d:\n%v", n, err)
	}
}

func TestNewScoresDeterministicKeys(t *testing.T) {
	t.Parallel()
	m := full()
	m[ability.Ability(42)] = ability.MustScore(10)
	m[ability.Ability(7)] = ability.MustScore(10)
	_, first := ability.NewScores(m)
	if !strings.Contains(first.Error(), "7") || !strings.Contains(first.Error(), "42") {
		t.Fatalf("keys missing from %v", first)
	}
	for range 20 {
		if _, err := ability.NewScores(m); err.Error() != first.Error() {
			t.Fatalf("nondeterministic:\n%v\nvs\n%v", err, first)
		}
	}
	if strings.Index(first.Error(), "7") > strings.Index(first.Error(), "42") {
		t.Errorf("not sorted: %v", first)
	}
}

// Requirement: none - pins current behaviour of NewScore's error text.
// Path: out-of-range scores on both sides of the bounds.
func TestNewScoreErrorMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		v    int
		want string
	}{
		{0, "ability: below minimum: score 0 is below 1"},
		{-12, "ability: below minimum: score -12 is below 1"},
		{31, "ability: above maximum: score 31 is above 30"},
	}
	for _, tc := range tests {
		_, err := ability.NewScore(tc.v)
		if err == nil || err.Error() != tc.want {
			t.Errorf("NewScore(%d) error = %v, want %q", tc.v, err, tc.want)
		}
	}
	for _, v := range []int{ability.MinScore, ability.MaxScore} {
		if s, err := ability.NewScore(v); err != nil || s.Value() != v {
			t.Errorf("NewScore(%d) = %v, %v", v, s, err)
		}
	}
}

// Requirement: SRD 5.2.1, "Ability Scores", p. 5-6 - the modifier table.
// Path: Modifier.String for negative, zero and positive values across the table.
func TestModifierStringAcrossTable(t *testing.T) {
	t.Parallel()
	for v, want := range map[int]string{1: "-5", 8: "-1", 9: "-1", 10: "+0", 11: "+0", 12: "+1", 30: "+10"} {
		if got := ability.MustScore(v).Modifier().String(); got != want {
			t.Errorf("score %d: %q, want %q", v, got, want)
		}
	}
}

// Requirement: none - pins current behaviour of Reason and InvalidError.Is.
// Path: every defined Reason, the first undefined value above, and mismatched
// reasons on both sides of the ordering.
func TestReasonNamesAndMismatch(t *testing.T) {
	t.Parallel()
	names := map[ability.Reason]string{
		ability.ReasonUnknownAbility: "unknown ability",
		ability.ReasonBelowMinimum:   "below minimum",
		ability.ReasonAboveMaximum:   "above maximum",
		ability.ReasonMissingScore:   "missing score",
	}
	for r, want := range names {
		if r.String() != want {
			t.Errorf("Reason(%d) = %q, want %q", int(r), r.String(), want)
		}
	}
	if ability.ReasonUnknownAbility != 1 || ability.Reason(5).String() != "unknown" || ability.Reason(-1).String() != "unknown" {
		t.Error("Reason numbering or bounds")
	}
	mid := &ability.InvalidError{Reason: ability.ReasonBelowMinimum}
	for _, r := range []ability.Reason{ability.ReasonUnknownAbility, ability.ReasonAboveMaximum} {
		if errors.Is(mid, &ability.InvalidError{Reason: r}) {
			t.Errorf("below minimum should not match reason %v", r)
		}
	}
	if (&ability.InvalidError{Reason: ability.ReasonBelowMinimum, Detail: "x"}).Error() != "ability: below minimum: x" {
		t.Error("Error with detail")
	}
}
