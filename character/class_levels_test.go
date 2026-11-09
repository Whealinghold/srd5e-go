package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func problemsOf(t *testing.T, err error) []character.ClassLevelsProblem {
	t.Helper()
	var out []character.ClassLevelsProblem
	for _, e := range joined(err) {
		var ce *character.ClassLevelsError
		if !errors.As(e, &ce) || !errors.Is(e, srderr.ErrInvalidInput) {
			t.Fatalf("error %v is not a *ClassLevelsError matching ErrInvalidInput", e)
		}
		out = append(out, ce.Problem)
	}
	return out
}

// Requirement: doc/specs/derived-stats.md FR-058, FR-075 - NewClassLevels joins
// one error per violated invariant, in enum order (SRD 5.2.1, "Multiclassing",
// p. 25).
// Path: per case.
func TestNewClassLevels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		entries    []character.ClassLevel
		want       []character.ClassLevelsProblem
	}{
		{"empty", "unhappy (empty and minimal)", nil, []character.ClassLevelsProblem{character.ClassLevelsEmpty}},
		{"empty, non-nil", "unhappy (empty and minimal)", []character.ClassLevel{}, []character.ClassLevelsProblem{character.ClassLevelsEmpty}},
		{"class repeated", "unhappy (invalid input)", []character.ClassLevel{cl("fighter", 2), cl("rogue", 1), cl("fighter", 1)}, []character.ClassLevelsProblem{character.ClassLevelsDuplicateClass}},
		{"total 20", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 10), cl("rogue", 10)}, nil},
		{"total 21", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 11), cl("rogue", 10)}, []character.ClassLevelsProblem{character.ClassLevelsTotalAboveMaximum}},
		{"one level", "unhappy (empty and minimal)", []character.ClassLevel{cl("fighter", 1)}, nil},
		{"level 0", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 0)}, []character.ClassLevelsProblem{character.ClassLevelsLevelBelowMinimum}},
		{"several problems", "unhappy (invalid input)", []character.ClassLevel{cl("fighter", 0), cl("fighter", 1), cl("rogue", 20)},
			[]character.ClassLevelsProblem{character.ClassLevelsLevelBelowMinimum, character.ClassLevelsDuplicateClass, character.ClassLevelsTotalAboveMaximum}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := character.NewClassLevels(c.entries)
			if c.want == nil {
				if err != nil || l.IsZero() {
					t.Fatalf("NewClassLevels = %v, %v", l, err)
				}
				return
			}
			if got := problemsOf(t, err); !slices.Equal(got, c.want) {
				t.Errorf("problems = %v, want %v", got, c.want)
			}
			if !l.IsZero() {
				t.Error("a value came back with an error")
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-058 - first entry is the initial
// class, TotalLevel is the sum, entries are copied in and out.
// Path: unhappy (immutability).
func TestClassLevelsAccessorsAndCopies(t *testing.T) {
	t.Parallel()
	in := []character.ClassLevel{cl("rogue", 2), cl("fighter", 3)}
	l := character.MustClassLevels(in)
	in[0].Level = 9
	out := l.Entries()
	out[1].Level = 9
	if l.Initial() != "rogue" || l.TotalLevel().Value() != 5 || l.LevelIn("fighter") != 3 || l.LevelIn("wizard") != 0 || l.Len() != 2 {
		t.Errorf("accessors: initial %q total %d fighter %d", l.Initial(), l.TotalLevel().Value(), l.LevelIn("fighter"))
	}
	if !l.Equal(character.MustClassLevels([]character.ClassLevel{cl("rogue", 2), cl("fighter", 3)})) || l.Equal(character.ClassLevels{}) {
		t.Error("Equal wrong")
	}
	var zero character.ClassLevels
	if zero.Initial() != "" || !zero.TotalLevel().IsZero() {
		t.Error("zero ClassLevels accessors")
	}
	if got := recovered(t, func() { character.MustClassLevels(nil) }); !errors.Is(got, srderr.ErrInvalidInput) {
		t.Errorf("MustClassLevels(nil) panic = %v", got)
	}
	if got := recovered(t, func() { character.MustClassLevels(in) }); got != nil {
		t.Errorf("MustClassLevels(valid) panicked: %v", got)
	}
	for p, want := range map[character.ClassLevelsProblem]string{
		character.ClassLevelsEmpty: "empty", character.ClassLevelsLevelBelowMinimum: "levelBelowMinimum",
		character.ClassLevelsDuplicateClass: "duplicateClass", character.ClassLevelsTotalAboveMaximum: "totalAboveMaximum", 0: "unknown",
	} {
		if p.String() != want {
			t.Errorf("%d.String() = %q", p, p.String())
		}
	}
	e := &character.ClassLevelsError{Problem: character.ClassLevelsEmpty}
	if e.Error() == "" || errors.Is(e, &character.ClassLevelsError{Problem: character.ClassLevelsDuplicateClass}) {
		t.Error("error text or Is wrong")
	}
}

// Requirement: doc/specs/derived-stats.md FR-072 - WithLevelIn raises a held
// class in place, appends a new one at level 1, keeps the initial class and the
// receiver, and fails past total 20.
// Path: per case.
func TestWithLevelIn(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		start      []character.ClassLevel
		add        string
		want       []character.ClassLevel
	}{
		{"new class appended last", "happy", []character.ClassLevel{cl("fighter", 3)}, "rogue", []character.ClassLevel{cl("fighter", 3), cl("rogue", 1)}},
		{"held class raised", "happy", []character.ClassLevel{cl("fighter", 3), cl("rogue", 2)}, "rogue", []character.ClassLevel{cl("fighter", 3), cl("rogue", 3)}},
		{"initial class stays first", "unhappy (rule interaction)", []character.ClassLevel{cl("rogue", 2), cl("fighter", 3)}, "fighter", []character.ClassLevel{cl("rogue", 2), cl("fighter", 4)}},
		{"total 19 plus one", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 10), cl("rogue", 9)}, "fighter", []character.ClassLevel{cl("fighter", 11), cl("rogue", 9)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			recv := character.MustClassLevels(c.start)
			got, err := recv.WithLevelIn(content.ClassID(c.add))
			if err != nil || !slices.Equal(got.Entries(), c.want) {
				t.Fatalf("WithLevelIn = %v, %v, want %v", got.Entries(), err, c.want)
			}
			if !slices.Equal(recv.Entries(), c.start) {
				t.Errorf("receiver changed to %v", recv.Entries())
			}
			if _, err := character.NewClassLevels(got.Entries()); err != nil {
				t.Errorf("result violates FR-058: %v", err)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-072 - at total 20 a level cannot
// be added, held class or not.
// Path: unhappy (boundary).
func TestWithLevelInAtTwenty(t *testing.T) {
	t.Parallel()
	l := character.MustClassLevels([]character.ClassLevel{cl("fighter", 10), cl("rogue", 10)})
	for _, id := range []content.ClassID{"fighter", "wizard"} {
		got, err := l.WithLevelIn(id)
		if p := problemsOf(t, err); !slices.Equal(p, []character.ClassLevelsProblem{character.ClassLevelsTotalAboveMaximum}) || !got.IsZero() {
			t.Errorf("WithLevelIn(%s) = %v, problems %v", id, got.Entries(), p)
		}
	}
	// from the empty receiver the class becomes the initial class
	if got, err := (character.ClassLevels{}).WithLevelIn("fighter"); err != nil || got.Initial() != "fighter" {
		t.Errorf("zero.WithLevelIn = %v, %v", got.Entries(), err)
	}
}
