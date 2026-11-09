package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ClassLevel is a class and the character's level in it. Construct it with a
// keyed literal.
type ClassLevel struct {
	Class content.ClassID
	Level int
}

// ClassLevels is the ordered sequence of a character's classes and levels
// (SRD 5.2.1, "Multiclassing", p. 24-25). The first entry is the initial class.
// It is immutable and defensively copied; it holds a slice, so compare two with
// Equal, not ==. The zero ClassLevels is empty and invalid; see IsZero. Safe
// for concurrent use.
type ClassLevels struct{ entries []ClassLevel }

// ClassLevelsProblem names an invariant of ClassLevels that an input violates.
// The zero value is not a problem.
type ClassLevelsProblem int

// The class level problems, in the order NewClassLevels reports them.
const (
	// ClassLevelsEmpty: there are no entries.
	ClassLevelsEmpty ClassLevelsProblem = iota + 1
	// ClassLevelsLevelBelowMinimum: an entry has a level below 1.
	ClassLevelsLevelBelowMinimum
	// ClassLevelsDuplicateClass: a class appears more than once.
	ClassLevelsDuplicateClass
	// ClassLevelsTotalAboveMaximum: the levels sum to more than 20.
	ClassLevelsTotalAboveMaximum
)

// String returns "empty", "levelBelowMinimum", "duplicateClass",
// "totalAboveMaximum" or "unknown".
func (p ClassLevelsProblem) String() string {
	switch p {
	case ClassLevelsEmpty:
		return "empty"
	case ClassLevelsLevelBelowMinimum:
		return "levelBelowMinimum"
	case ClassLevelsDuplicateClass:
		return "duplicateClass"
	case ClassLevelsTotalAboveMaximum:
		return "totalAboveMaximum"
	}
	return "unknown"
}

// ClassLevelsError reports one violated invariant of ClassLevels. It matches
// srderr.ErrInvalidInput with errors.Is.
type ClassLevelsError struct {
	Problem ClassLevelsProblem
}

// Error renders the error. The text is not a contract.
func (e *ClassLevelsError) Error() string {
	return "character: invalid class levels: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ClassLevelsError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ClassLevelsError with the same Problem.
func (e *ClassLevelsError) Is(target error) bool {
	t, ok := target.(*ClassLevelsError)
	return ok && t != nil && e != nil && t.Problem == e.Problem
}

// NewClassLevels returns the class levels in entries, copied. The first entry
// is the initial class. Every entry needs a level of at least 1, no class may
// repeat, there must be at least one entry and the total is at most 20.
//
// It returns an errors.Join of one *ClassLevelsError per violated invariant,
// each at most once, in the order of ClassLevelsProblem, each matching
// srderr.ErrInvalidInput. It never panics.
func NewClassLevels(entries []ClassLevel) (ClassLevels, error) {
	var problems []error
	add := func(p ClassLevelsProblem) { problems = append(problems, &ClassLevelsError{Problem: p}) }
	total, below, dup := 0, false, false
	seen := make(map[content.ClassID]bool, len(entries))
	for _, e := range entries {
		total += e.Level
		below = below || e.Level < 1
		dup = dup || seen[e.Class]
		seen[e.Class] = true
	}
	if len(entries) == 0 {
		add(ClassLevelsEmpty)
	}
	if below {
		add(ClassLevelsLevelBelowMinimum)
	}
	if dup {
		add(ClassLevelsDuplicateClass)
	}
	if total > MaxLevel {
		add(ClassLevelsTotalAboveMaximum)
	}
	if len(problems) > 0 {
		return ClassLevels{}, errors.Join(problems...)
	}
	return ClassLevels{entries: slices.Clone(entries)}, nil
}

// MustClassLevels is NewClassLevels for constants known to be valid. It panics
// with the error NewClassLevels returns, and only then.
func MustClassLevels(entries []ClassLevel) ClassLevels {
	l, err := NewClassLevels(entries)
	if err != nil {
		panic(err)
	}
	return l
}

// Entries returns a new slice of the entries in order.
func (l ClassLevels) Entries() []ClassLevel { return slices.Clone(l.entries) }

// IsZero reports whether l has no entries (the invalid zero value).
func (l ClassLevels) IsZero() bool { return len(l.entries) == 0 }

// Len returns the number of classes held.
func (l ClassLevels) Len() int { return len(l.entries) }

// Equal reports whether l and o hold the same entries in the same order.
func (l ClassLevels) Equal(o ClassLevels) bool { return slices.Equal(l.entries, o.entries) }

// Initial returns the initial class, the first entry; the zero ClassID when l
// is empty.
func (l ClassLevels) Initial() content.ClassID {
	if l.IsZero() {
		return ""
	}
	return l.entries[0].Class
}

// LevelIn returns the level in class, or 0 when the class is not held.
func (l ClassLevels) LevelIn(class content.ClassID) int {
	for _, e := range l.entries {
		if e.Class == class {
			return e.Level
		}
	}
	return 0
}

// TotalLevel returns the sum of the levels, a valid Level; the zero
// Level when l is empty.
func (l ClassLevels) TotalLevel() Level {
	n := 0
	for _, e := range l.entries {
		n += e.Level
	}
	return Level{n: n}
}

// WithLevelIn returns a new ClassLevels with one more level in class: a held
// class is raised by 1 and keeps its position, a class not held is appended
// last at level 1. The receiver is unchanged and the first entry stays the
// initial class.
//
// It returns a *ClassLevelsError with ClassLevelsTotalAboveMaximum (matching
// srderr.ErrInvalidInput) when the total would exceed 20. It never panics.
func (l ClassLevels) WithLevelIn(class content.ClassID) (ClassLevels, error) {
	if l.TotalLevel().n+1 > MaxLevel {
		return ClassLevels{}, &ClassLevelsError{Problem: ClassLevelsTotalAboveMaximum}
	}
	next := slices.Clone(l.entries)
	if i := slices.IndexFunc(next, func(e ClassLevel) bool { return e.Class == class }); i >= 0 {
		next[i].Level++
	} else {
		next = append(next, ClassLevel{Class: class, Level: 1})
	}
	return ClassLevels{entries: next}, nil
}
